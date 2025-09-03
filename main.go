package main

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sahitya-chandra/web-crawler/db"
	"github.com/sahitya-chandra/web-crawler/queue"
	"golang.org/x/net/html"
)

type CrawledSet struct {
	set map[uint64]bool
	length int
	mu sync.RWMutex
}

func (cs *CrawledSet) add(url string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.set[hashUrl(url)] {
		cs.set[hashUrl(url)] = true
		cs.length++
	}
}

func (cs *CrawledSet) contains(url string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.set[hashUrl(url)]
}

func (cs *CrawledSet) size() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.length
}

func hashUrl(url string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(url))

	return h.Sum64()
}

type PageResult struct {
	URL string
	HTML string
	Err error
}

type ParsePage struct {
	Url string
	Title string
	Body string
	Err error
}

func fetchHTML(u string, ch chan<- PageResult) {
	client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(u)
	if err != nil {
		ch <- PageResult{URL: u, Err: fmt.Errorf("fetch error: %w", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- PageResult{URL: u, Err: fmt.Errorf("status code %d", resp.StatusCode)}
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ch <- PageResult{URL: u, Err: fmt.Errorf("read error: %w", err)}
		return
	}

	ch <- PageResult{URL: u, HTML: string(body)}
}


func parseHTML(in <-chan PageResult, out chan<- ParsePage, crawled *CrawledSet, q *queue.Queue) {

	for page := range in {
		if page.Err != nil {
			out <- ParsePage{Url: page.URL, Err: page.Err}
			continue
		}

		htmlContent := page.HTML
		doc, err := html.Parse(strings.NewReader(htmlContent))
		if err != nil {
            out <- ParsePage{Url: page.URL, Err: fmt.Errorf("parse error: %w", err)}
            continue
        }

		var title, bodyText string
		var extract func(*html.Node)
		extract = func(n *html.Node) {
			if n.Type == html.ElementNode {
				switch n.Data {
				case "title":
					if n.FirstChild != nil {
						title = strings.TrimSpace(n.FirstChild.Data)
					}
				case "body":
					if n.FirstChild != nil {
						bodyText = getFirst500Words(n.FirstChild)
					}
				case "a":
					for _, attr := range n.Attr {
						if attr.Key == "href" {
							link := strings.TrimSpace(attr.Val)
							if link != "" {
								norm, err := normalizeLink(page.URL, link)
								if err == nil && !crawled.contains(norm) {
									q.Enqueue(norm)
								}
							}
						}
					}
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extract(c)
			}
		}
		extract(doc)
		

		out <- ParsePage{
			Url: page.URL,
			Title: title,
			Body: bodyText,
		}
	}
}

func getFirst500Words(n *html.Node) string {
	var buf strings.Builder
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
			if buf.Len() >= 500 {
				return
			}
		}

		for c := n.FirstChild; c != nil; c= c.NextSibling {
			traverse(c)
			if buf.Len() >= 500 {
				return 
			}
		}
	}

	traverse(n)
	result := strings.TrimSpace(buf.String())
    if len(result) > 500 {
        return result[:500]
    }
    return result
}

func normalizeLink(base, href string) (string, error) {
	parsedBase, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	parsedHref, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	return parsedBase.ResolveReference(parsedHref).String(), nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("MONGODB_URI not set in .env file")
	}

	dbInstance, err := db.Connect(uri)
	if err != nil {
		log.Fatal(err)
	}
	defer dbInstance.Disconnect()

	queue := &queue.Queue{}
	crawled := &CrawledSet{set: make(map[uint64]bool)}
	queue.Enqueue("https://www.mjpru.ac.in/")

	fetchChan := make(chan PageResult, 5)
	parseChan := make(chan ParsePage, 5)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		parseHTML(fetchChan, parseChan, crawled, queue)
	}()

	for queue.Size() > 0 && crawled.size() < 500 {
		url, ok := queue.Dequeue()
		if !ok {
			break
		}

		if crawled.contains(url) {
			continue
		}

		crawled.add(url)

		go fetchHTML(url, fetchChan)

		parsed := <-parseChan
		if parsed.Err != nil {
			log.Printf("Error parsing %s: %v", parsed.Url, parsed.Err)
            continue
		}

		err = dbInstance.InsertWebpage("webpages", map[string]interface{}{
			"url": parsed.Url,
			"title": parsed.Title,
			"content": parsed.Body,
		})

		if err != nil {
			log.Printf("Error inserting %s: %v", parsed.Url, err)
		}

		fmt.Printf("Crawled: %s, Title: %s\n", parsed.Url, parsed.Title)
        time.Sleep(1 * time.Second) // Polite delay

	}

	
    close(fetchChan)
    close(parseChan)
    wg.Wait()
}