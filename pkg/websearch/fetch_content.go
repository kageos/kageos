package websearch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const (
	userAgent              = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	maxFetchContentChars   = 2000 // 单页正文最多取字数
	maxFetchContentResults = 2    // 只对前 N 条搜索结果抓取正文
	maxFetchBodyBytes      = 2 << 20
)

// FetchURLContent 对给定 URL 发 GET，提取可读正文：
// - HTML / XHTML：解析 DOM 取标题与正文（与历史行为一致）
// - 文本类（text/* 含 markdown、plain、json、xml、svg 等）：按 UTF-8 解码后截断返回
// - 其它类型：若内容主要为可打印文本则同样返回；否则返回简短说明（仍保证有返回信息）
func FetchURLContent(ctx context.Context, pageURL string, maxChars int) (title, body string) {
	if pageURL == "" || maxChars <= 0 {
		return "", ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", ""
	}
	applyFollowURLHeaders(req)
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

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBodyBytes))
	if err != nil {
		return "", ""
	}
	ct := resp.Header.Get("Content-Type")

	if isHTMLContentType(ct) {
		doc, err := html.Parse(bytes.NewReader(raw))
		if err != nil {
			return "", ""
		}
		title = getHTMLTitle(doc)
		text := extractTextFromHTML(doc, maxChars)
		body = normalizeSpaceAndTruncate(text, maxChars)
		return title, body
	}

	if shouldTreatAsPlainText(ct, raw) {
		s := bytesToValidUTF8(raw)
		title = firstLineAsTitle(s)
		body = truncateByRunes(strings.TrimSpace(s), maxChars)
		return title, body
	}

	// 非文本：仍返回可读的元信息，避免工具结果为空
	msg := fmt.Sprintf("（该 URL 返回 Content-Type: %q，约 %d 字节；内容判定为非可读文本，未展开正文。）", strings.TrimSpace(ct), len(raw))
	if strings.TrimSpace(ct) == "" {
		msg = fmt.Sprintf("（无 Content-Type，约 %d 字节；内容判定为非可读文本，未展开正文。）", len(raw))
	}
	return "", msg
}

func contentTypeMain(ct string) string {
	ct = strings.TrimSpace(strings.ToLower(ct))
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = ct[:idx]
	}
	return strings.TrimSpace(ct)
}

func isHTMLContentType(ct string) bool {
	m := contentTypeMain(ct)
	return strings.Contains(m, "text/html") || strings.Contains(m, "application/xhtml")
}

func isDeclaredTextualContentType(ct string) bool {
	m := contentTypeMain(ct)
	if m == "" {
		return false
	}
	if strings.HasPrefix(m, "text/") && !strings.Contains(m, "text/html") {
		return true
	}
	switch m {
	case "application/json", "application/ld+json", "application/javascript",
		"application/x-javascript", "application/xml", "application/rss+xml",
		"application/atom+xml":
		return true
	case "image/svg+xml":
		return true
	default:
		return false
	}
}

func isMostlyPrintableText(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	n := len(b)
	const sample = 64 * 1024
	if n > sample {
		n = sample
	}
	bad := 0
	for i := 0; i < n; i++ {
		c := b[i]
		if c == 0 {
			return false
		}
		if c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		if c < 32 {
			bad++
		}
	}
	return float64(bad)/float64(n) < 0.03
}

func shouldTreatAsPlainText(contentType string, raw []byte) bool {
	if isDeclaredTextualContentType(contentType) {
		return true
	}
	// 未声明或浏览器式类型：嗅探是否为文本
	if strings.TrimSpace(contentType) == "" && isMostlyPrintableText(raw) {
		return true
	}
	m := contentTypeMain(contentType)
	if m != "" && !strings.HasPrefix(m, "text/") && !isHTMLContentType(contentType) {
		// 例如 application/octet-stream 但实际是 UTF-8 文本
		if isMostlyPrintableText(raw) {
			return true
		}
	}
	return false
}

func bytesToValidUTF8(b []byte) string {
	return strings.ToValidUTF8(string(b), "")
}

func firstLineAsTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	line := s
	if idx := strings.IndexAny(s, "\r\n"); idx >= 0 {
		line = strings.TrimSpace(s[:idx])
	}
	line = strings.TrimPrefix(line, "#")
	line = strings.TrimSpace(line)
	return truncateByRunes(line, 120)
}

func truncateByRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	var sb strings.Builder
	count := 0
	for _, r := range s {
		if count >= maxRunes {
			break
		}
		sb.WriteRune(r)
		count++
	}
	return sb.String() + "..."
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
