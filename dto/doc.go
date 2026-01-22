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

// QueryDocsReq 查询文档请求（统一接口，支持路径批量查询和关键词搜索）
type QueryDocsReq struct {
	// 路径批量查询模式：提供 paths
	Paths []string `json:"paths"` // 文档路径列表（与 keyword 二选一）
	
	// 关键词搜索模式：提供 keyword
	Keyword  string `json:"keyword"`  // 搜索关键词（可选，用于搜索名称和路径，与 paths 二选一）
	Page     int    `json:"page"`    // 页码（搜索模式时使用，默认 1）
	PageSize int    `json:"page_size"` // 每页数量（搜索模式时使用，默认 10，最大 100）
	
	// 通用参数
	IncludeContent bool `json:"include_content"` // 是否包含文档内容（默认 true，设为 false 时只返回元数据，适合列表展示）
}

// QueryDocsResp 查询文档响应
type QueryDocsResp struct {
	Docs     []*DocItem `json:"docs"`     // 文档列表
	Total    int64      `json:"total"`    // 总数（搜索模式时返回，路径模式时为实际数量）
	Page     int        `json:"page"`    // 当前页码（搜索模式时返回）
	PageSize int        `json:"page_size"` // 每页数量（搜索模式时返回）
}

// GetDocsByPathsReq 根据路径批量获取文档请求（保留用于向后兼容）
type GetDocsByPathsReq struct {
	Paths []string `json:"paths"` // 文档路径列表
}

// GetDocsByPathsResp 根据路径批量获取文档响应（保留用于向后兼容）
type GetDocsByPathsResp struct {
	Docs []*DocItem `json:"docs"` // 文档列表
}
