package websearch

import (
	"context"
	"os"
	"strings"
)

// 环境变量 WEB_SEARCH_ENGINE（可选）：
//   - 空或未设置：先百度再必应（百度失败或 0 条时回退必应）
//   - baidu：仅百度
//   - bing：仅必应
func searchEngineMode() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv("WEB_SEARCH_ENGINE")))
}

// Search 默认优先使用百度搜索，失败或 0 条结果时回退必应（cn.bing.com）；可通过 WEB_SEARCH_ENGINE 调整。
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

	var results []Result
	var err error
	switch searchEngineMode() {
	case "bing":
		results, err = searchBing(ctx, keyword, limit)
	case "baidu":
		results, err = searchBaidu(ctx, keyword, limit)
	default:
		results, err = searchBaidu(ctx, keyword, limit)
		if err != nil || len(results) == 0 {
			results, err = searchBing(ctx, keyword, limit)
		}
	}
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
