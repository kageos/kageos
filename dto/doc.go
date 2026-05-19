package dto

// CreateDocReq 创建文档请求
type CreateDocReq struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 完整路径（如：/user/app/docs/guide）
	Content      string `json:"content" binding:"required"`        // 文档内容
	Format       string `json:"format"`                            // 文档格式（默认 markdown）
	Summary      string `json:"summary"`                           // 文档摘要（可选）
}

// UpdateDocReq 更新文档请求
type UpdateDocReq struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 完整路径（如：/user/app/docs/guide）
	Content      string `json:"content"`                           // 文档内容（可选）
	Format       string `json:"format"`                            // 文档格式（可选）
	Summary      string `json:"summary"`                           // 文档摘要（可选）
}

// DocItem 文档项
type DocItem struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Content      string `json:"content"`
	Format       string `json:"format"`
	FullCodePath string `json:"full_code_path"`
	Summary      string `json:"summary"`
	Category     string `json:"category"`
}

// SearchDocsReq 搜索文档请求（模糊搜索）
type SearchDocsReq struct {
	Keyword        string `form:"keyword" json:"keyword"`                 // 搜索关键词（可选，为空时返回最近创建的文档）
	Page           int    `form:"page" json:"page"`                       // 页码（默认 1）
	PageSize       int    `form:"page_size" json:"page_size"`             // 每页数量（默认 10，最大 100）
	IncludeContent bool   `form:"include_content" json:"include_content"` // 是否包含文档内容（默认 true，设为 false 时只返回元数据，适合列表展示）
}

// SearchDocsResp 搜索文档响应
type SearchDocsResp struct {
	Docs     []*DocItem `json:"docs"`      // 文档列表
	Total    int64      `json:"total"`     // 总数
	Page     int        `json:"page"`      // 当前页码
	PageSize int        `json:"page_size"` // 每页数量
}

// BatchGetDocsReq 批量获取文档请求（精确查询）
type BatchGetDocsReq struct {
	Paths          []string `form:"paths" json:"paths" binding:"required"`  // 文档路径列表（必填，支持 paths[]=value1&paths[]=value2 或 paths=value1&paths=value2）
	IncludeContent bool     `form:"include_content" json:"include_content"` // 是否包含文档内容（默认 true）
}

// BatchGetDocsResp 批量获取文档响应
type BatchGetDocsResp struct {
	Docs []*DocItem `json:"docs"` // 文档列表
}
