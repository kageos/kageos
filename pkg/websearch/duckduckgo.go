package websearch

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	duckDuckGoURL = "https://html.duckduckgo.com/html/"
	userAgent     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// searchDuckDuckGo 请求 DuckDuckGo HTML 搜索页并解析结果，无 API key，自建爬虫
func searchDuckDuckGo(ctx context.Context, keyword string, limit int) ([]Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, duckDuckGoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	q := req.URL.Query()
	q.Set("q", keyword)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // 静默返回空，不打断聚合
	}

	return parseDuckDuckGoHTML(resp.Body, limit)
}

// parseDuckDuckGoHTML 解析 DuckDuckGo 结果页：result__url、result__title、result__snippet
func parseDuckDuckGoHTML(body io.Reader, limit int) ([]Result, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	var results []Result

	var f func(*html.Node)
	f = func(n *html.Node) {
		if len(results) >= limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "result ") {
					var cur struct{ title, url, snippet string }
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						collectResultNode(c, &cur)
					}
					if cur.url != "" && (cur.title != "" || cur.snippet != "") {
						results = append(results, Result{
							Title:   cur.title,
							URL:     cur.url,
							Snippet: cur.snippet,
							Source:  "duckduckgo",
						})
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return results, nil
}

func collectResultNode(n *html.Node, cur *struct{ title, url, snippet string }) {
	if n.Type == html.ElementNode {
		var class string
		for _, a := range n.Attr {
			if a.Key == "class" {
				class = a.Val
				break
			}
		}
		switch {
		case strings.Contains(class, "result__title"):
			cur.title = strings.TrimSpace(textContent(n))
		case strings.Contains(class, "result__snippet"):
			cur.snippet = strings.TrimSpace(textContent(n))
		case n.Data == "a" && strings.Contains(class, "result__url"):
			for _, a := range n.Attr {
				if a.Key == "href" {
					cur.url = resolveDuckDuckGoURL(a.Val)
					break
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectResultNode(c, cur)
	}
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return strings.TrimSpace(b.String())
}

// resolveDuckDuckGoURL 若为 duckduckgo.com/l/?uddg= 则解析出真实 URL，否则原样返回
func resolveDuckDuckGoURL(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if (strings.Contains(u.Host, "duckduckgo.com") && u.Path == "/l/") || u.RawQuery != "" {
		q := u.Query()
		if dec := q.Get("uddg"); dec != "" {
			if d, err := url.QueryUnescape(dec); err == nil {
				return d
			}
		}
	}
	return href
}
