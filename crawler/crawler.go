package crawler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sahitya-chandra/web-crawler/queue"
	"golang.org/x/net/html"
)

type PageResult struct {
	URL  string
	HTML string
	Err  error
}

type ParsePage struct {
	Url   string
	Title string
	Body  string
	Err   error
}

func FetchHTML(u string, ch chan<- PageResult) {
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

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		ch <- PageResult{URL: u, Err: fmt.Errorf("skipped non-HTML content: %s", ct)}
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ch <- PageResult{URL: u, Err: fmt.Errorf("read error: %w", err)}
		return
	}

	ch <- PageResult{URL: u, HTML: string(body)}
}

func ParseHTML(
	in <-chan PageResult,
	out chan<- ParsePage,
	crawledContains func(string) bool,
	q *queue.Queue,
) {
	for page := range in {
		if page.Err != nil {
			out <- ParsePage{Url: page.URL, Err: page.Err}
			continue
		}

		doc, err := html.Parse(strings.NewReader(page.HTML))
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
								if err == nil && !crawledContains(norm) {
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

		out <- ParsePage{Url: page.URL, Title: title, Body: bodyText}
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

		for c := n.FirstChild; c != nil; c = c.NextSibling {
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
