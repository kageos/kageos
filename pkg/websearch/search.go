package websearch

import (
	"context"
	"strings"
)

// Search 聚合多源搜索（DuckDuckGo 爬虫 + Wikipedia 免费 API），返回去重后的结果，最多 limit 条
func Search(ctx context.Context, keyword string, limit int) ([]Result, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	seen := make(map[string]bool)
	var out []Result

	// 1) DuckDuckGo 爬虫：通用网页结果
	dd, _ := searchDuckDuckGo(ctx, keyword, limit)
	for i, r := range dd {
		k := r.URL
		if k == "" {
			k = r.Title
		}
		if k != "" && !seen[k] {
			seen[k] = true
			// 对前 maxFetchContentResults 条抓取正文（通用网页）
			if i < maxFetchContentResults && r.URL != "" {
				r.Body = fetchPageContent(ctx, r.URL, maxFetchContentChars)
			}
			out = append(out, r)
			if len(out) >= limit {
				return out, nil
			}
		}
	}

	// 2) Wikipedia：知识/百科补充（最多 3 条）
	wikiLimit := 3
	if limit-len(out) < wikiLimit {
		wikiLimit = limit - len(out)
	}
	if wikiLimit > 0 {
		wiki, _ := searchWikipedia(ctx, keyword, wikiLimit, "zh")
		for _, r := range wiki {
			if r.URL != "" {
				r.URL = resolveWikipediaURL(r.URL)
			}
			k := r.URL
			if k == "" {
				k = r.Title
			}
			if k != "" && !seen[k] {
				seen[k] = true
				out = append(out, r)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}

	return out, nil
}
