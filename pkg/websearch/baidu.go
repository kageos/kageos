package websearch

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const baiduSearchURL = "https://www.baidu.com/s"

// searchBaidu 请求百度 HTML 搜索页并解析结果（国内常用；可能遇验证码或空结果，可与必应互为兜底）
func searchBaidu(ctx context.Context, keyword string, limit int) ([]Result, error) {
	resp, err := doGET(ctx, searchHTTPClient, baiduSearchURL, func(req *http.Request) {
		applyBaiduSearchHeaders(req)
		q := req.URL.Query()
		q.Set("wd", keyword)
		rn := limit
		if rn > 50 {
			rn = 50
		}
		q.Set("rn", strconv.Itoa(rn))
		req.URL.RawQuery = q.Encode()
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return parseBaiduHTML(resp.Body, limit)
}

// parseBaiduHTML 解析百度结果页：自然结果多在 div.c-container，标题在 h3.t > a，摘要常见 c-abstract / content-right
func parseBaiduHTML(body io.Reader, limit int) ([]Result, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	var results []Result
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= limit {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			cls := bingAttr(n, "class")
			if strings.Contains(cls, "c-container") {
				title, link, snippet := extractBaiduResultBlock(n)
				if link != "" && (title != "" || snippet != "") {
					results = append(results, Result{
						Title:   title,
						URL:     link,
						Snippet: snippet,
						Source:  "baidu",
					})
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return results, nil
}

func extractBaiduResultBlock(root *html.Node) (title, link, snippet string) {
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if title != "" && link != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "h3" {
			if bingHasClassToken(bingAttr(n, "class"), "t") {
				// 常见：h3.t > a，或 h3.t > span > a
				var pickA func(*html.Node) bool
				pickA = func(x *html.Node) bool {
					if x.Type == html.ElementNode && x.Data == "a" {
						href := bingAttr(x, "href")
						resolved := resolveBaiduURL(href)
						if resolved != "" {
							link = resolved
							title = strings.TrimSpace(textContent(x))
							return true
						}
					}
					for ch := x.FirstChild; ch != nil; ch = ch.NextSibling {
						if pickA(ch) {
							return true
						}
					}
					return false
				}
				pickA(n)
			}
		}
		if title != "" && link != "" {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
		}
	}
	findTitle(root)

	// 部分条目在容器 div 的 mu 上带真实落地 URL
	if link == "" {
		for _, a := range root.Attr {
			if a.Key == "mu" {
				u := strings.TrimSpace(a.Val)
				if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
					link = u
				}
				break
			}
		}
	}

	snippet = bestBaiduSnippet(root)
	return title, link, snippet
}

func bestBaiduSnippet(root *html.Node) string {
	var best string
	var bestScore int
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return
		}
		cls := bingAttr(n, "class")
		switch n.Data {
		case "div", "span", "p":
			t := strings.TrimSpace(textContent(n))
			if len([]rune(t)) <= 8 {
				break
			}
			score := len([]rune(t))
			if strings.Contains(cls, "c-abstract") {
				score += 300
			}
			if strings.Contains(cls, "content-right") || strings.Contains(cls, "content_right") {
				score += 200
			}
			if strings.Contains(cls, "abstract") {
				score += 120
			}
			// 压低站点名、标签行
			if strings.Contains(cls, "c-showurl") || strings.Contains(cls, "c-color-text") {
				score -= 100
			}
			if score > bestScore {
				bestScore = score
				best = t
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return best
}

func resolveBaiduURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	low := strings.ToLower(href)
	if strings.HasPrefix(low, "javascript:") {
		return ""
	}
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	if !strings.HasPrefix(href, "http") {
		u, err := url.Parse(href)
		if err != nil {
			return href
		}
		if u.Host == "" {
			return "https://www.baidu.com" + href
		}
	}
	return href
}
