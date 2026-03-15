package websearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Wikipedia 使用站点自带的 Opensearch API（免费、无 key），仅做知识补充
const (
	wikipediaAPIZH     = "https://zh.wikipedia.org/w/api.php"
	wikipediaAPIEN     = "https://en.wikipedia.org/w/api.php"
	wikipediaExtractChars = 2500 // 每条条目最多取正文字数
)

// opensearch 返回格式: [ keyword, [title1, title2...], [desc1, desc2...], [url1, url2...] ]
func searchWikipedia(ctx context.Context, keyword string, limit int, lang string) ([]Result, error) {
	api := wikipediaAPIEN
	if lang == "zh" || strings.ContainsAny(keyword, "的一是在不了有和人这中大为上个国我以要他时来地们生到作地于出就分对成会可主发年动同工也能下过子说自产") {
		api = wikipediaAPIZH
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	q := req.URL.Query()
	q.Set("action", "opensearch")
	q.Set("search", keyword)
	q.Set("limit", "5")
	q.Set("format", "json")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw) < 4 {
		return nil, nil
	}

	var titles, descs, urls []string
	_ = json.Unmarshal(raw[1], &titles)
	_ = json.Unmarshal(raw[2], &descs)
	_ = json.Unmarshal(raw[3], &urls)

	out := make([]Result, 0, limit)
	for i := 0; i < len(titles) && i < limit; i++ {
		title := ""
		if i < len(titles) {
			title = titles[i]
		}
		snippet := ""
		if i < len(descs) {
			snippet = descs[i]
		}
		link := ""
		if i < len(urls) {
			link = urls[i]
		}
		if title == "" && link == "" {
			continue
		}
		if link != "" {
			link = resolveWikipediaURL(link)
		}
		out = append(out, Result{
			Title:   title,
			URL:     link,
			Snippet: snippet,
			Source:  "wikipedia",
		})
	}
	// 拉取正文：MediaWiki API prop=extracts
	if len(out) > 0 {
		titlesToFetch := make([]string, 0, len(out))
		for _, r := range out {
			if r.Title != "" {
				titlesToFetch = append(titlesToFetch, r.Title)
			}
		}
		bodyMap := fetchWikipediaExtracts(ctx, api, titlesToFetch)
		for i := range out {
			if b := bodyMap[out[i].Title]; b != "" {
				out[i].Body = b
			}
		}
	}
	return out, nil
}

// fetchWikipediaExtracts 用 query prop=extracts 拉取条目正文，返回 title -> 正文
func fetchWikipediaExtracts(ctx context.Context, apiBase string, titles []string) map[string]string {
	if len(titles) == 0 {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", userAgent)
	q := req.URL.Query()
	q.Set("action", "query")
	q.Set("prop", "extracts")
	q.Set("titles", strings.Join(titles, "|"))
	q.Set("explaintext", "1")
	q.Set("exintro", "0")
	q.Set("exchars", "2500")
	q.Set("format", "json")
	q.Set("redirects", "1")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var data struct {
		Query struct {
			Pages map[string]struct {
				Title   string `json:"title"`
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}
	out := make(map[string]string, len(data.Query.Pages))
	for _, p := range data.Query.Pages {
		extract := strings.TrimSpace(p.Extract)
		if len(extract) > wikipediaExtractChars {
			extract = extract[:wikipediaExtractChars] + "..."
		}
		if extract != "" {
			out[p.Title] = extract
		}
	}
	return out
}

// resolveWikipediaURL 确保是绝对 URL
func resolveWikipediaURL(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "http") {
		return path
	}
	u, _ := url.Parse(path)
	if u != nil && u.Host != "" {
		return path
	}
	return "https://zh.wikipedia.org" + path
}
