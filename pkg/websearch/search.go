package websearch

import (
	"context"
	"strings"
)

// Search 使用必应（cn.bing.com）搜索，国内可直接访问，无需翻墙
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

	results, err := searchBing(ctx, keyword, limit)
	if err != nil || len(results) == 0 {
		return results, err
	}

	// 对前几条抓取正文
	for i := range results {
		if i >= maxFetchContentResults {
			break
		}
		if results[i].URL != "" {
			results[i].Body = fetchPageContent(ctx, results[i].URL, maxFetchContentChars)
		}
	}
	return results, nil
}
