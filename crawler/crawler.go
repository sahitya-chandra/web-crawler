package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const userAgent = "GoCrawler/1.0 (learning project; +https://github.com/sahitya-chandra/web-crawler)"

type PageResult struct {
	URL  string
	HTML string
	Err  error
}

type ParsedPage struct {
	URL   string
	Title string
	Body  string
	Links []string
	Err   error
}

func FetchHTML(ctx context.Context, rawURL string) PageResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return PageResult{URL: rawURL, Err: fmt.Errorf("build request: %w", err)}
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return PageResult{URL: rawURL, Err: fmt.Errorf("fetch: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PageResult{URL: rawURL, Err: fmt.Errorf("status %d", resp.StatusCode)}
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		return PageResult{URL: rawURL, Err: fmt.Errorf("non-HTML content type: %s", ct)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PageResult{URL: rawURL, Err: fmt.Errorf("read body: %w", err)}
	}

	return PageResult{URL: rawURL, HTML: string(body)}
}

func ParseHTML(page PageResult) ParsedPage {
	if page.Err != nil {
		return ParsedPage{URL: page.URL, Err: page.Err}
	}

	doc, err := html.Parse(strings.NewReader(page.HTML))
	if err != nil {
		return ParsedPage{URL: page.URL, Err: fmt.Errorf("parse HTML: %w", err)}
	}

	var title, bodyText string
	var bodyNode *html.Node
	var links []string

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				title = extractText(n)
			case "body":
				bodyNode = n
			case "a":
				if href := getAttr(n, "href"); href != "" {
					if norm, err := NormalizeLink(page.URL, href); err == nil {
						links = append(links, norm)
					}
				}
			case "script", "style", "noscript":
				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	if bodyNode != nil {
		bodyText = extractFirstNWords(bodyNode, 500)
	}

	return ParsedPage{
		URL:   page.URL,
		Title: title,
		Body:  bodyText,
		Links: links,
	}
}

func extractText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func extractFirstNWords(n *html.Node, maxWords int) string {
	var buf strings.Builder
	wordCount := 0

	var traverse func(*html.Node) bool
	traverse = func(n *html.Node) bool {
		if wordCount >= maxWords {
			return false
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript":
				return true
			}
		}

		if n.Type == html.TextNode {
			words := strings.Fields(n.Data)
			for _, word := range words {
				if wordCount >= maxWords {
					break
				}
				if buf.Len() > 0 {
					buf.WriteString(" ")
				}
				buf.WriteString(word)
				wordCount++
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !traverse(c) {
				return false
			}
		}
		return true
	}

	traverse(n)
	return buf.String()
}

func NormalizeLink(base, href string) (string, error) {
	href = strings.TrimSpace(href)

	lower := strings.ToLower(href)
	if strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "javascript:") ||
		strings.HasPrefix(lower, "tel:") ||
		strings.HasPrefix(lower, "data:") ||
		href == "#" || href == "" {
		return "", fmt.Errorf("skipped link: %s", href)
	}

	parsedBase, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsedHref, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	resolved := parsedBase.ResolveReference(parsedHref)

	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", fmt.Errorf("non-HTTP scheme: %s", resolved.Scheme)
	}

	resolved.Fragment = ""

	return resolved.String(), nil
}
