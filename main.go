package main

import (
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/sahitya-chandra/web-crawler/queue"
)

type CrawledSet struct {
	set map[uint64]bool
	length int
	mu sync.RWMutex
}

func (cs *CrawledSet) add(url string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.set[hashUrl(url)] = true
	cs.length++
}

func (cs *CrawledSet) contains(url string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.set[hashUrl(url)]
}

func (cs *CrawledSet) size() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

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

func fetchHTML(u string, ch chan<- PageResult) {
	resp, err := http.Get(u)
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

type ParsePage struct {
	Url string
	Title string
	Body string
	Err error
}

func parseHTML(in <-chan PageResult, out chan<- ParsePage, crawled *CrawledSet, q *Queue) {
	titleRegex := regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	bodyRegex := regexp.MustCompile(`(?is)<body.*?>(.*?)</body>`)
	linkRegex := regexp.MustCompile(`(?i)<a\s+(?:[^>]*?\s+)?href="([^"]*)"`)

	for page := range in {
		if page.Err != nil {
			out <- ParsePage{Url: page.URL, Err: page.Err}
			return
		}

		html := page.HTML

		title := ""
		if match := titleRegex.FindStringSubmatch(html); len(match) > 1 {
			title = strings.TrimSpace(match[1])
		}

		bodyText := ""
		if match := bodyRegex.FindAllSubmatch(html); len(match) > 1 {
			bodyText = stripTags(match[1])
		}

		words := strings.Fields(bodyText)
		if len(words) > 500 {
			words = words[:500]
		}
		bodyText = strings.Join(words, " ")

		links := []string{}
		for _, match := range linkRegex.FindAllSubmatch(html, -1) {
			link := strings.TrimSpace(match[1])
			if link != "" {
				norm, err := normalizeLink(page.URL, link)
				if err != nil {
					continue
				}

				if crawled.contains(norm) {
					continue
				} else {
					q.Enqueue(norm)
				}
			}
		}

		out <- ParsePage{
			Url: page.URL,
			Title: title,
			Body: bodyText,
		}
	}
}

func stripTags(input string) string {
	re := regexp.MustCompile(`<.*?>`)
	return re.ReplaceAllString(input, "")
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
	fmt.Println("hd")
}