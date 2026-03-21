package websearch

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestDecodeBingUEncodedTarget(t *testing.T) {
	// 与社区常见示例一致：前缀 a1 + Base64(URL)
	raw := "a1aHR0cHM6Ly93d3cuZGFuaWVsc2h2YWMuY29tLw"
	got := decodeBingUEncodedTarget(raw)
	want := "https://www.danielshvac.com"
	if got != want && !strings.HasPrefix(got, want) {
		t.Fatalf("decodeBingUEncodedTarget(%q) = %q, want prefix %q", raw, got, want)
	}
}

func TestUnwrapBingTrackingURL(t *testing.T) {
	href := "https://www.bing.com/ck/a?!&&u=a1aHR0cHM6Ly93d3cuZXhhbXBsZS5jb20v"
	got := unwrapBingTrackingURL(href)
	if !strings.HasPrefix(got, "https://www.example.com") {
		t.Fatalf("unwrapBingTrackingURL = %q", got)
	}
}

func TestParseBingHTMLSample(t *testing.T) {
	const sample = `<!doctype html><html><body><ol id="b_results">
<li class="b_algo"><div class="b_tpcn"><a href="https://ignore.example/">x</a></div>
<h2 class=""><a href="https://ok.example/page">My Title</a></h2>
<div class="b_caption"><p class="b_lineclamp2">This is the real snippet text for the result.</p></div>
</li></ol></body></html>`
	res, err := parseBingHTML(strings.NewReader(sample), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("len = %d", len(res))
	}
	if res[0].Title != "My Title" || res[0].URL != "https://ok.example/page" {
		t.Fatalf("title/url: %+v", res[0])
	}
	if !strings.Contains(res[0].Snippet, "real snippet") {
		t.Fatalf("snippet: %q", res[0].Snippet)
	}
}

func TestBingTitleLinkFromH2NestedAnchor(t *testing.T) {
	const frag = `<h2><span><a href="https://nested.example/z">Nested Title</a></span></h2>`
	doc, err := html.Parse(strings.NewReader(frag))
	if err != nil {
		t.Fatal(err)
	}
	var h2 *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h2" {
			h2 = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	if h2 == nil {
		t.Fatal("no h2")
	}
	title, link := bingTitleLinkFromH2(h2)
	if link != "https://nested.example/z" || title != "Nested Title" {
		t.Fatalf("got %q %q", title, link)
	}
}
