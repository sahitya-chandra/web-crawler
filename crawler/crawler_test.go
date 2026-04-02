package crawler

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestNormalizeLink(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		href    string
		want    string
		wantErr bool
	}{
		{
			name: "absolute URL",
			base: "https://example.com/page",
			href: "https://other.com/foo",
			want: "https://other.com/foo",
		},
		{
			name: "relative path",
			base: "https://example.com/page/",
			href: "about",
			want: "https://example.com/page/about",
		},
		{
			name: "root-relative path",
			base: "https://example.com/page/deep",
			href: "/top",
			want: "https://example.com/top",
		},
		{
			name: "strips fragment",
			base: "https://example.com/",
			href: "/page#section",
			want: "https://example.com/page",
		},
		{
			name:    "rejects mailto",
			base:    "https://example.com/",
			href:    "mailto:user@example.com",
			wantErr: true,
		},
		{
			name:    "rejects javascript",
			base:    "https://example.com/",
			href:    "javascript:void(0)",
			wantErr: true,
		},
		{
			name:    "rejects tel",
			base:    "https://example.com/",
			href:    "tel:+1234567890",
			wantErr: true,
		},
		{
			name:    "rejects empty href",
			base:    "https://example.com/",
			href:    "",
			wantErr: true,
		},
		{
			name:    "rejects bare fragment",
			base:    "https://example.com/",
			href:    "#",
			wantErr: true,
		},
		{
			name:    "rejects ftp scheme",
			base:    "https://example.com/",
			href:    "ftp://files.example.com/a.zip",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeLink(tt.base, tt.href)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFirstNWords(t *testing.T) {
	htmlStr := `<body><p>The quick brown fox jumps over the lazy dog.</p></body>`
	doc, _ := html.Parse(strings.NewReader(htmlStr))

	var body *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)

	if body == nil {
		t.Fatal("could not find body node")
	}

	got := extractFirstNWords(body, 5)
	want := "The quick brown fox jumps"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got = extractFirstNWords(body, 100)
	want = "The quick brown fox jumps over the lazy dog."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestExtractFirstNWordsSkipsScriptStyle(t *testing.T) {
	htmlStr := `<body><script>var x = 1;</script><style>.a{}</style><p>Hello world</p></body>`
	doc, _ := html.Parse(strings.NewReader(htmlStr))

	var body *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)

	got := extractFirstNWords(body, 500)
	if strings.Contains(got, "var") || strings.Contains(got, ".a") {
		t.Fatalf("body text should not contain script/style content, got %q", got)
	}
	if got != "Hello world" {
		t.Fatalf("got %q, want %q", got, "Hello world")
	}
}

func TestParseHTML(t *testing.T) {
	rawHTML := `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
  <p>Hello from the test page.</p>
  <a href="/about">About</a>
  <a href="mailto:hi@example.com">Email</a>
  <a href="https://other.com/page#top">Other</a>
</body>
</html>`

	result := PageResult{URL: "https://example.com/", HTML: rawHTML}
	parsed := ParseHTML(result)

	if parsed.Err != nil {
		t.Fatalf("unexpected error: %v", parsed.Err)
	}

	if parsed.Title != "Test Page" {
		t.Fatalf("title: got %q, want %q", parsed.Title, "Test Page")
	}

	if !strings.Contains(parsed.Body, "Hello from the test page") {
		t.Fatalf("body missing expected text: %q", parsed.Body)
	}

	expectedLinks := map[string]bool{
		"https://example.com/about": true,
		"https://other.com/page":    true, // fragment stripped
	}
	for _, link := range parsed.Links {
		delete(expectedLinks, link)
	}
	if len(expectedLinks) > 0 {
		t.Fatalf("missing expected links: %v (got %v)", expectedLinks, parsed.Links)
	}

	for _, link := range parsed.Links {
		if strings.HasPrefix(link, "mailto:") {
			t.Fatalf("mailto link should have been filtered: %s", link)
		}
	}
}

func TestParseHTMLWithError(t *testing.T) {
	errResult := PageResult{URL: "https://fail.com", Err: fmt.Errorf("connection refused")}
	parsed := ParseHTML(errResult)
	if parsed.Err == nil {
		t.Fatal("expected error to be propagated")
	}
	if parsed.URL != "https://fail.com" {
		t.Fatalf("URL: got %q, want %q", parsed.URL, "https://fail.com")
	}
}
