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
	bingSearchURL = "https://cn.bing.com/search"
)

// searchBing 请求必应（cn.bing.com）HTML 搜索页并解析结果，国内可直接访问
func searchBing(ctx context.Context, keyword string, limit int) ([]Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bingSearchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	q := req.URL.Query()
	q.Set("q", keyword)
	q.Set("count", "20")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return parseBingHTML(resp.Body, limit)
}

// parseBingHTML 解析必应结果页：每条结果在 li.b_algo 内，标题与链接在 h2 > a，摘要在 p 或 .b_caption
func parseBingHTML(body io.Reader, limit int) ([]Result, error) {
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
		if n.Type == html.ElementNode && n.Data == "li" {
			var class string
			for _, a := range n.Attr {
				if a.Key == "class" {
					class = a.Val
					break
				}
			}
			if !strings.Contains(class, "b_algo") {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
				return
			}
			var cur struct{ title, link, snippet string }
			collectBingResultNode(n, &cur)
			if cur.link != "" && (cur.title != "" || cur.snippet != "") {
				results = append(results, Result{
					Title:   cur.title,
					URL:     cur.link,
					Snippet: cur.snippet,
					Source:  "bing",
				})
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return results, nil
}

func collectBingResultNode(n *html.Node, cur *struct{ title, link, snippet string }) {
	if n.Type == html.ElementNode {
		var class string
		for _, a := range n.Attr {
			if a.Key == "class" {
				class = a.Val
				break
			}
		}
		// 标题与链接：h2 > a
		if n.Data == "h2" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Data == "a" {
					for _, a := range c.Attr {
						if a.Key == "href" {
							cur.link = resolveBingURL(a.Val)
							break
						}
					}
					cur.title = strings.TrimSpace(textContent(c))
					return
				}
			}
		}
		// 摘要：p 或 .b_caption 内的文本（避免用易变的 class，用 p 或包含 caption 的 div）
		if n.Data == "p" || strings.Contains(class, "caption") || strings.Contains(class, "lineclamp") {
			t := strings.TrimSpace(textContent(n))
			if t != "" && len(t) > 10 {
				cur.snippet = t
				return
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectBingResultNode(c, cur)
	}
}

func resolveBingURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "javascript:") {
		return ""
	}
	if !strings.HasPrefix(href, "http") {
		u, err := url.Parse(href)
		if err != nil {
			return href
		}
		if u.Host == "" {
			return "https://cn.bing.com" + href
		}
	}
	return href
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
