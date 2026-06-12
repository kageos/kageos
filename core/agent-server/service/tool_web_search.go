package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"golang.org/x/net/html"
)

type WebSearchTool struct{}

type webSearchArgs struct {
	Query      string `json:"query" schema_desc:"要搜索的公开网页关键词。必须具体，不要只传泛泛的一个词" schema_required:"true"`
	MaxResults int    `json:"max_results" schema_desc:"最多返回结果数，默认 5，最大 10"`
	Site       string `json:"site" schema_desc:"可选，限制搜索站点域名，例如 docs.example.com；不要带 http:// 或 https://"`
}

type webSearchResultData struct {
	Query      string                `json:"query"`
	Source     string                `json:"source"`
	ResultList []webSearchResultItem `json:"results"`
}

type webSearchResultItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

var webSearchToolDef = toolDefinitionWithOutput[webSearchArgs, structuredToolResultSchema[webSearchResultData]](
	"web_search",
	"搜索公开互联网网页，返回标题、链接和摘要。适合查最新公开资料、官方文档、错误信息、行业资料或外部事实；结果来自公开网页摘要，重要事实需要优先采信官方来源，并在最终回答中说明不确定性。只读工具，不会修改工作区或业务数据。",
)

func (t *WebSearchTool) Definition() dto.ToolDef {
	return webSearchToolDef
}

func (t *WebSearchTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[webSearchArgs](call.Args)
	if err != nil {
		return toolResult("web_search 参数解析失败: "+err.Error(), true)
	}
	data, err := runWebSearchTool(ctx, args)
	if err != nil {
		return toolResult("web_search 调用失败: "+err.Error(), true)
	}
	return toolResultWithData(formatWebSearchOutput(data), false, data)
}

func runWebSearchTool(ctx context.Context, args webSearchArgs) (webSearchResultData, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return webSearchResultData{}, fmt.Errorf("query 不能为空")
	}
	if site := normalizeWebSearchSite(args.Site); site != "" {
		query = query + " site:" + site
	}
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	results, source, err := duckDuckGoHTMLSearch(ctx, query, maxResults)
	if err != nil || len(results) == 0 {
		fallback, fallbackErr := duckDuckGoInstantAnswerSearch(ctx, query, maxResults)
		if fallbackErr != nil {
			if err != nil {
				return webSearchResultData{}, err
			}
			return webSearchResultData{}, fallbackErr
		}
		results = fallback
		source = "duckduckgo_instant_answer"
	}

	return webSearchResultData{
		Query:      query,
		Source:     source,
		ResultList: results,
	}, nil
}

func duckDuckGoHTMLSearch(ctx context.Context, query string, maxResults int) ([]webSearchResultItem, string, error) {
	endpoint := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "KageOS-Agent/1.0")
	resp, err := webSearchHTTPClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("duckduckgo html http %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, "", err
	}
	return parseDuckDuckGoHTMLResults(strings.NewReader(string(data)), maxResults), "duckduckgo_html", nil
}

func duckDuckGoInstantAnswerSearch(ctx context.Context, query string, maxResults int) ([]webSearchResultItem, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")
	endpoint := "https://api.duckduckgo.com/?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "KageOS-Agent/1.0")
	resp, err := webSearchHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("duckduckgo instant answer http %d", resp.StatusCode)
	}
	var payload duckDuckGoInstantPayload
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.SearchResults(maxResults), nil
}

type duckDuckGoInstantPayload struct {
	Heading       string                   `json:"Heading"`
	AbstractText  string                   `json:"AbstractText"`
	AbstractURL   string                   `json:"AbstractURL"`
	Results       []duckDuckGoInstantTopic `json:"Results"`
	RelatedTopics []duckDuckGoInstantTopic `json:"RelatedTopics"`
}

type duckDuckGoInstantTopic struct {
	Text     string                   `json:"Text"`
	FirstURL string                   `json:"FirstURL"`
	Topics   []duckDuckGoInstantTopic `json:"Topics"`
}

func (p duckDuckGoInstantPayload) SearchResults(maxResults int) []webSearchResultItem {
	out := make([]webSearchResultItem, 0, maxResults)
	add := func(title, rawURL, snippet string) {
		if len(out) >= maxResults {
			return
		}
		title = strings.TrimSpace(title)
		rawURL = strings.TrimSpace(rawURL)
		snippet = strings.TrimSpace(snippet)
		if title == "" && snippet != "" {
			title = firstSentence(snippet)
		}
		if title == "" || rawURL == "" {
			return
		}
		out = append(out, webSearchResultItem{Title: title, URL: rawURL, Snippet: snippet})
	}
	add(p.Heading, p.AbstractURL, p.AbstractText)
	var walk func([]duckDuckGoInstantTopic)
	walk = func(topics []duckDuckGoInstantTopic) {
		for _, topic := range topics {
			if len(out) >= maxResults {
				return
			}
			add("", topic.FirstURL, topic.Text)
			if len(topic.Topics) > 0 {
				walk(topic.Topics)
			}
		}
	}
	walk(p.Results)
	walk(p.RelatedTopics)
	return out
}

func parseDuckDuckGoHTMLResults(r io.Reader, maxResults int) []webSearchResultItem {
	doc, err := html.Parse(r)
	if err != nil {
		return nil
	}
	out := make([]webSearchResultItem, 0, maxResults)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil || len(out) >= maxResults {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" && htmlClassContains(n, "result__a") {
			title := normalizeWebSearchText(htmlNodeText(n))
			rawURL := normalizeDuckDuckGoResultURL(htmlAttr(n, "href"))
			if title != "" && rawURL != "" {
				result := webSearchResultItem{Title: title, URL: rawURL}
				if block := nearestDuckDuckGoResultBlock(n); block != nil {
					result.Snippet = normalizeWebSearchText(firstNodeTextByClass(block, "result__snippet"))
				}
				out = append(out, result)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
			if len(out) >= maxResults {
				return
			}
		}
	}
	walk(doc)
	return out
}

func nearestDuckDuckGoResultBlock(n *html.Node) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && htmlClassContains(p, "result") {
			return p
		}
	}
	return nil
}

func firstNodeTextByClass(n *html.Node, class string) string {
	if n == nil {
		return ""
	}
	if n.Type == html.ElementNode && htmlClassContains(n, class) {
		return htmlNodeText(n)
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if text := firstNodeTextByClass(child, class); strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func htmlNodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if text := htmlNodeText(child); text != "" {
			if b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString(text)
		}
	}
	return b.String()
}

func htmlClassContains(n *html.Node, target string) bool {
	classes := strings.Fields(htmlAttr(n, "class"))
	for _, class := range classes {
		if class == target {
			return true
		}
	}
	return false
}

func htmlAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return html.UnescapeString(attr.Val)
		}
	}
	return ""
}

func normalizeDuckDuckGoResultURL(raw string) string {
	raw = strings.TrimSpace(html.UnescapeString(raw))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if strings.Contains(parsed.Host, "duckduckgo.com") && strings.HasPrefix(parsed.Path, "/l/") {
		if target := parsed.Query().Get("uddg"); target != "" {
			return target
		}
	}
	return parsed.String()
}

func normalizeWebSearchText(text string) string {
	return strings.Join(strings.Fields(html.UnescapeString(strings.TrimSpace(text))), " ")
}

func normalizeWebSearchSite(site string) string {
	site = strings.TrimSpace(site)
	site = strings.TrimPrefix(site, "https://")
	site = strings.TrimPrefix(site, "http://")
	site = strings.Trim(site, "/")
	return site
}

func formatWebSearchOutput(data webSearchResultData) string {
	var b strings.Builder
	b.WriteString("网页搜索结果")
	if data.Query != "" {
		b.WriteString("：")
		b.WriteString(data.Query)
	}
	if data.Source != "" {
		b.WriteString("（source=")
		b.WriteString(data.Source)
		b.WriteString("）")
	}
	if len(data.ResultList) == 0 {
		b.WriteString("\n未找到结果。")
		return b.String()
	}
	for i, item := range data.ResultList {
		b.WriteString("\n")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". ")
		b.WriteString(item.Title)
		if item.URL != "" {
			b.WriteString("\n   ")
			b.WriteString(item.URL)
		}
		if item.Snippet != "" {
			b.WriteString("\n   ")
			b.WriteString(item.Snippet)
		}
	}
	return b.String()
}

func firstSentence(text string) string {
	text = normalizeWebSearchText(text)
	for _, sep := range []string{"。", ".", "！", "!", "？", "?"} {
		if idx := strings.Index(text, sep); idx > 0 {
			return strings.TrimSpace(text[:idx])
		}
	}
	runes := []rune(text)
	if len(runes) > 80 {
		return strings.TrimSpace(string(runes[:80]))
	}
	return text
}

func webSearchHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}
