package websearch

// Result 单条搜索结果，供工作台 web_search 工具返回给模型
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Body    string `json:"body"`   // 正文（部分源可拉取，如 Wikipedia API）
	Source  string `json:"source"` // "baidu" | "bing"
}
