package service

import (
	"strings"
	"testing"
)

func TestParseDuckDuckGoHTMLResults(t *testing.T) {
	html := `
		<div class="result results_links">
			<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fdocs">Example Docs</a>
			<a class="result__snippet">Official documentation for examples.</a>
		</div>
		<div class="result results_links">
			<a class="result__a" href="https://example.org/blog">Example Blog</a>
			<div class="result__snippet">A useful public article.</div>
		</div>`

	got := parseDuckDuckGoHTMLResults(strings.NewReader(html), 10)
	if len(got) != 2 {
		t.Fatalf("result count = %d, want 2: %#v", len(got), got)
	}
	if got[0].Title != "Example Docs" || got[0].URL != "https://example.com/docs" {
		t.Fatalf("first result = %#v", got[0])
	}
	if got[0].Snippet != "Official documentation for examples." {
		t.Fatalf("first snippet = %q", got[0].Snippet)
	}
	if got[1].URL != "https://example.org/blog" {
		t.Fatalf("second url = %q", got[1].URL)
	}
}

func TestNormalizeWebSearchSite(t *testing.T) {
	if got := normalizeWebSearchSite("https://docs.example.com/"); got != "docs.example.com" {
		t.Fatalf("normalizeWebSearchSite() = %q", got)
	}
}
