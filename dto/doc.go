package dto

// CreateDocReq 创建文档请求
type CreateDocReq struct {
	TreeID  int64  `json:"tree_id" binding:"required"` // ServiceTree 节点 ID
	Title   string `json:"title" binding:"required"`   // 文档标题
	Content string `json:"content" binding:"required"` // 文档内容
	Format  string `json:"format"`                     // 文档格式（默认 markdown）
	Summary string `json:"summary"`                    // 文档摘要（可选）
}

// UpdateDocReq 更新文档请求
type UpdateDocReq struct {
	TreeID       int64  `json:"tree_id,omitempty"`        // ServiceTree 节点 ID（基于 ID 更新时使用）
	FullCodePath string `json:"full_code_path,omitempty"` // 完整路径（基于路径更新时使用）
	Title        string `json:"title"`                    // 文档标题（可选）
	Content      string `json:"content"`                  // 文档内容（可选）
	Format       string `json:"format"`                   // 文档格式（可选）
	Summary      string `json:"summary"`                  // 文档摘要（可选）
}
