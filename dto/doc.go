package dto

// CreateDocReq 创建文档请求
type CreateDocReq struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 完整路径（如：/user/app/docs/guide）
	Title        string `json:"title" binding:"required"`          // 文档标题
	Content      string `json:"content" binding:"required"`        // 文档内容
	Format       string `json:"format"`                            // 文档格式（默认 markdown）
	Summary      string `json:"summary"`                           // 文档摘要（可选）
}

// UpdateDocReq 更新文档请求
type UpdateDocReq struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 完整路径（如：/user/app/docs/guide）
	Title        string `json:"title"`                             // 文档标题（可选）
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

// GetDocsByPathsReq 根据路径批量获取文档请求
type GetDocsByPathsReq struct {
	Paths []string `json:"paths"` // 文档路径列表
}

// GetDocsByPathsResp 根据路径批量获取文档响应
type GetDocsByPathsResp struct {
	Docs []*DocItem `json:"docs"` // 文档列表
}

// SearchDocsReq 搜索文档请求
type SearchDocsReq struct {
	Keyword  string `json:"keyword" form:"keyword"`                         // 搜索关键词（可选，用于搜索名称和路径）
	Page     int    `json:"page" form:"page" binding:"required" example:"1"`            // 页码
	PageSize int    `json:"page_size" form:"page_size" binding:"required" example:"10"` // 每页数量
}

// SearchDocsResp 搜索文档响应
type SearchDocsResp struct {
	Docs     []*DocItem `json:"docs"`     // 文档列表
	Total    int64      `json:"total"`    // 总数
	Page     int        `json:"page"`    // 当前页码
	PageSize int        `json:"page_size"` // 每页数量
}
