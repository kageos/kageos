package websearch

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	userAgent             = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	maxFetchContentChars   = 2000 // 单页正文最多取字数
	maxFetchContentResults = 2    // 只对前 N 条搜索结果抓取正文
)

// FetchURLContent 对给定 URL 发 GET，解析 HTML 提取标题和正文，供「按 URL 访问页面」工具使用
func FetchURLContent(ctx context.Context, pageURL string, maxChars int) (title, body string) {
	if pageURL == "" || maxChars <= 0 {
		return "", ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", ""
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml") {
		return "", ""
	}
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", ""
	}
	title = getHTMLTitle(doc)
	text := extractTextFromHTML(doc, maxChars)
	body = normalizeSpaceAndTruncate(text, maxChars)
	return title, body
}

func getHTMLTitle(doc *html.Node) string {
	var walk func(*html.Node) string
	walk = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "title" {
			var b strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					b.WriteString(c.Data)
				}
			}
			return strings.TrimSpace(b.String())
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if s := walk(c); s != "" {
				return s
			}
		}
		return ""
	}
	return walk(doc)
}

// fetchPageContent 对给定 URL 发 GET，解析 HTML 提取正文（去 script/style，取 body 内文本），截断 maxChars
func fetchPageContent(ctx context.Context, pageURL string, maxChars int) string {
	_, body := FetchURLContent(ctx, pageURL, maxChars)
	return body
}

// extractTextFromHTML 遍历 DOM，跳过 script/style，优先取 main/article 否则 body 内文本
func extractTextFromHTML(doc *html.Node, maxChars int) string {
	var b strings.Builder
	var target *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "main" || n.Data == "article") {
			target = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
			if target != nil {
				return
			}
		}
	}
	walk(doc)
	if target != nil {
		collectText(target, &b, maxChars)
		if b.Len() > 0 {
			return b.String()
		}
	}
	var body *html.Node
	var findBody func(*html.Node)
	findBody = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findBody(c)
		}
	}
	findBody(doc)
	if body != nil {
		collectText(body, &b, maxChars)
	}
	return b.String()
}

func collectText(n *html.Node, b *strings.Builder, maxChars int) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
		return
	}
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "noscript") {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, b, maxChars)
		if b.Len() >= maxChars {
			return
		}
	}
}

var spaceNorm = regexp.MustCompile(`\s+`)

func normalizeSpaceAndTruncate(s string, maxChars int) string {
	s = spaceNorm.ReplaceAllString(strings.TrimSpace(s), " ")
	if len(s) > maxChars {
		s = s[:maxChars] + "..."
	}
	return s
}
