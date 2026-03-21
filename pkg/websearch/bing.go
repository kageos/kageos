package websearch

import (
	"context"
	"encoding/base64"
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
	applyBingSearchHeaders(req)
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
			class := bingAttr(n, "class")
			// 必须带独立 class token「b_algo」，避免误匹配其它含子串的 class
			if !bingHasClassToken(class, "b_algo") {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
				return
			}
			title, link, snippet := collectBingFromAlgoLI(n)
			if link != "" && (title != "" || snippet != "") {
				results = append(results, Result{
					Title:   title,
					URL:     link,
					Snippet: snippet,
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

// bingHasClassToken 按空白拆分 class，精确匹配 token（避免误匹配含 b_algo 子串的其它类名）
func bingHasClassToken(class, token string) bool {
	if class == "" || token == "" {
		return false
	}
	for _, t := range strings.Fields(class) {
		if t == token {
			return true
		}
	}
	return false
}

func bingAttr(n *html.Node, key string) string {
	if n == nil || n.Type != html.ElementNode {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// collectBingFromAlgoLI 从一条 li.b_algo 提取标题、链接、摘要（兼容 h2 内嵌套、必应 ck/a 跳转）
func collectBingFromAlgoLI(li *html.Node) (title, link, snippet string) {
	var walkH2 func(*html.Node)
	walkH2 = func(n *html.Node) {
		if title != "" && link != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "h2" {
			t, l := bingTitleLinkFromH2(n)
			if l != "" {
				title, link = t, l
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkH2(c)
		}
	}
	walkH2(li)

	snippet = bingSnippetFromAlgoLI(li)
	return title, link, snippet
}

// bingTitleLinkFromH2 在 h2 子树中取所有带有效 href 的 a，选得分最高的一条（标题更长、https 优先，压低纯 bing 站内非 ck 链接）
func bingTitleLinkFromH2(h2 *html.Node) (bestTitle, bestLink string) {
	type cand struct {
		title, link string
		score       int
	}
	var cands []cand
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return
		}
		if n.Data == "a" {
			href := bingAttr(n, "href")
			resolved := resolveBingURL(href)
			if resolved != "" {
				t := strings.TrimSpace(textContent(n))
				score := len([]rune(t))
				low := strings.ToLower(resolved)
				if strings.HasPrefix(low, "https://") {
					score += 10
				} else if strings.HasPrefix(low, "http://") {
					score += 5
				}
				if u, err := url.Parse(resolved); err == nil {
					h := strings.ToLower(u.Hostname())
					if strings.HasSuffix(h, "bing.com") {
						p := strings.ToLower(u.EscapedPath())
						if !strings.Contains(p, "ck/a") {
							score -= 80
						}
					}
				}
				cands = append(cands, cand{t, resolved, score})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(h2)
	if len(cands) == 0 {
		return "", ""
	}
	best := cands[0]
	for _, c := range cands[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best.title, best.link
}

// bingSnippetFromAlgoLI 优先 div.b_caption 内段落（含 b_lineclamp*），否则在整个 li 内选最佳 p
func bingSnippetFromAlgoLI(li *html.Node) string {
	var caption *html.Node
	var findCaption func(*html.Node)
	findCaption = func(n *html.Node) {
		if caption != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" && bingHasClassToken(bingAttr(n, "class"), "b_caption") {
			caption = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findCaption(c)
		}
	}
	findCaption(li)
	if caption != nil {
		if s := bingBestParagraphSnippet(caption); s != "" {
			return s
		}
	}
	return bingBestParagraphSnippet(li)
}

func bingBestParagraphSnippet(root *html.Node) string {
	var best string
	var bestScore int
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			t := strings.TrimSpace(textContent(n))
			if len([]rune(t)) <= 10 {
				// 太短多为噪声
			} else {
				cls := bingAttr(n, "class")
				score := len([]rune(t))
				if strings.Contains(cls, "lineclamp") {
					score += 200
				}
				if strings.Contains(cls, "snippet") {
					score += 80
				}
				if score > bestScore {
					bestScore = score
					best = t
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return best
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
			href = "https://cn.bing.com" + href
		}
	}
	return unwrapBingTrackingURL(href)
}

// unwrapBingTrackingURL 将 bing.com/ck/a?...&u=... 中的 u（常带 a1 等前缀 + Base64）还原为真实目标 URL
func unwrapBingTrackingURL(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, "bing.com") {
		return href
	}
	path := strings.ToLower(u.EscapedPath())
	if !strings.Contains(path, "ck/a") {
		return href
	}
	q := u.Query()
	raw := q.Get("u")
	if raw == "" {
		return href
	}
	raw, _ = url.QueryUnescape(raw)
	if target := decodeBingUEncodedTarget(raw); target != "" {
		return target
	}
	return href
}

func decodeBingUEncodedTarget(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// 必应常见：u=a1<base64>，需去掉少量前缀再 Base64 解码
	maxTrim := 12
	if len(raw) < maxTrim {
		maxTrim = len(raw)
	}
	for trim := 0; trim < maxTrim; trim++ {
		s := raw[trim:]
		if len(s) < 12 {
			continue
		}
		if out, ok := bingBase64URLDecodeToHTTPURL(s); ok {
			return out
		}
	}
	return ""
}

func bingBase64URLDecodeToHTTPURL(s string) (string, bool) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	for len(s)%4 != 0 {
		s += "="
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", false
	}
	out := strings.TrimSpace(string(b))
	low := strings.ToLower(out)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		return out, true
	}
	return "", false
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
