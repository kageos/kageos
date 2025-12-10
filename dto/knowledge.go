package dto

// KnowledgeListReq 获取知识库列表请求
type KnowledgeListReq struct {
	Scope    string `json:"scope" form:"scope"` // mine: 我的, market: 市场
	Page     int    `json:"page" form:"page" binding:"required" example:"1"`
	PageSize int    `json:"page_size" form:"page_size" binding:"required" example:"10"`
}

// KnowledgeInfo 知识库信息
type KnowledgeInfo struct {
	ID            int64  `json:"id" example:"1"`
	Name          string `json:"name" example:"Excel知识库"`
	Description   string `json:"description" example:"Excel相关的知识库"`
	Status        string `json:"status" example:"active"` // active/inactive
	DocumentCount int    `json:"document_count" example:"10"`
	ContentHash   string `json:"content_hash" example:"abc123..."`
	User          string `json:"user" example:"admin"` // 保留用于向后兼容
	Visibility    int    `json:"visibility" example:"0"` // 0: 公开, 1: 私有
	Admin         string `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
	IsAdmin       bool   `json:"is_admin" example:"true"` // 当前用户是否是管理员
	CreatedAt     string `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt     string `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// KnowledgeListResp 获取知识库列表响应
type KnowledgeListResp struct {
	KnowledgeBases []KnowledgeInfo `json:"knowledge_bases"`
	Total          int64           `json:"total" example:"100"`
}

// KnowledgeGetReq 获取知识库详情请求
type KnowledgeGetReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"`
}

// KnowledgeGetResp 获取知识库详情响应
type KnowledgeGetResp struct {
	KnowledgeInfo
}

// KnowledgeCreateReq 创建知识库请求
type KnowledgeCreateReq struct {
	Name        string `json:"name" binding:"required" example:"Excel知识库"`
	Description string `json:"description" example:"Excel相关的知识库"`
	Status      string `json:"status" example:"active"` // active/inactive
	Visibility  int    `json:"visibility" example:"0"` // 0: 公开, 1: 私有（默认0）
	Admin       string `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔，默认创建用户）
}

// KnowledgeCreateResp 创建知识库响应
type KnowledgeCreateResp struct {
	ID int64 `json:"id" example:"1"`
}

// KnowledgeUpdateReq 更新知识库请求
type KnowledgeUpdateReq struct {
	ID          int64  `json:"id" binding:"required" example:"1"`
	Name        string `json:"name" binding:"required" example:"Excel知识库"`
	Description string `json:"description" example:"Excel相关的知识库"`
	Status      string `json:"status" example:"active"` // active/inactive
	Visibility  int    `json:"visibility" example:"0"` // 0: 公开, 1: 私有
	Admin       string `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
}

// KnowledgeUpdateResp 更新知识库响应
type KnowledgeUpdateResp struct {
	ID int64 `json:"id" example:"1"`
}

// KnowledgeDeleteReq 删除知识库请求
type KnowledgeDeleteReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"`
}

// KnowledgeAddDocumentReq 添加文档请求
type KnowledgeAddDocumentReq struct {
	KnowledgeBaseID int64  `json:"knowledge_base_id" binding:"required" example:"1"`
	ParentID        int64  `json:"parent_id" example:"0"` // 父文档ID（0表示根目录）
	Title           string `json:"title" binding:"required" example:"Excel使用指南"`
	Content         string `json:"content" binding:"required" example:"Excel是一个电子表格软件..."`
	FileType        string `json:"file_type" example:"pdf"` // pdf/txt/doc/md
	SortOrder       int    `json:"sort_order" example:"0"`   // 排序字段（可选，默认为0）
}

// KnowledgeAddDocumentResp 添加文档响应
type KnowledgeAddDocumentResp struct {
	DocID string `json:"doc_id" example:"doc_001"`
}

// KnowledgeListDocumentsReq 获取文档列表请求
type KnowledgeListDocumentsReq struct {
	KnowledgeBaseID int64 `json:"knowledge_base_id" form:"knowledge_base_id" binding:"required" example:"1"`
	Page            int   `json:"page" form:"page" binding:"required" example:"1"`
	PageSize        int   `json:"page_size" form:"page_size" binding:"required" example:"10"`
}

// DocumentInfo 文档信息
type DocumentInfo struct {
	ID              int64  `json:"id" example:"1"`
	KnowledgeBaseID int64  `json:"knowledge_base_id" example:"1"`
	ParentID        int64  `json:"parent_id" example:"0"` // 父文档ID（0表示根目录）
	DocID           string `json:"doc_id" example:"doc_001"`
	Title           string `json:"title" example:"Excel使用指南"`
	Content         string `json:"content" example:"Excel是一个电子表格软件..."`
	FileType        string `json:"file_type" example:"pdf"`
	FileSize        int64  `json:"file_size" example:"1024"`
	Status          string `json:"status" example:"completed"` // completed/failed
	SortOrder       int    `json:"sort_order" example:"0"`      // 排序字段
	Path            string `json:"path" example:"/目录1/文档"`   // 路径
	User            string `json:"user" example:"admin"`
	CreatedAt       string `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       string `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	Children        []DocumentInfo `json:"children,omitempty"` // 子文档列表（用于树形结构）
}

// KnowledgeListDocumentsResp 获取文档列表响应
type KnowledgeListDocumentsResp struct {
	Documents []DocumentInfo `json:"documents"`
	Total     int64          `json:"total" example:"100"`
}

// KnowledgeGetDocumentReq 获取文档详情请求
type KnowledgeGetDocumentReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"`
}

// KnowledgeGetDocumentResp 获取文档详情响应
type KnowledgeGetDocumentResp struct {
	DocumentInfo
}

// KnowledgeUpdateDocumentReq 更新文档请求
type KnowledgeUpdateDocumentReq struct {
	ID        int64  `json:"id" binding:"required" example:"1"`
	ParentID  int64  `json:"parent_id" example:"0"` // 父文档ID（0表示根目录）
	Title     string `json:"title" binding:"required" example:"Excel使用指南"`
	Content   string `json:"content" binding:"required" example:"Excel是一个电子表格软件..."`
	FileType  string `json:"file_type" example:"pdf"` // pdf/txt/doc/md
	Status    string `json:"status" example:"completed"` // completed/failed
	SortOrder int    `json:"sort_order" example:"0"`    // 排序字段
}

// KnowledgeUpdateDocumentResp 更新文档响应
type KnowledgeUpdateDocumentResp struct {
	ID int64 `json:"id" example:"1"`
}

// KnowledgeDeleteDocumentReq 删除文档请求
type KnowledgeDeleteDocumentReq struct {
	ID int64 `json:"id" binding:"required" example:"1"`
}

// KnowledgeGetDocumentsTreeReq 获取文档树请求
type KnowledgeGetDocumentsTreeReq struct {
	KnowledgeBaseID int64 `json:"knowledge_base_id" form:"knowledge_base_id" binding:"required" example:"1"`
}

// KnowledgeGetDocumentsTreeResp 获取文档树响应
type KnowledgeGetDocumentsTreeResp struct {
	Documents []DocumentInfo `json:"documents"` // 树形结构的文档列表
}

// KnowledgeUpdateDocumentsSortReq 批量更新文档排序请求
type KnowledgeUpdateDocumentsSortReq struct {
	KnowledgeBaseID int64                        `json:"knowledge_base_id" binding:"required" example:"1"`
	Updates         []KnowledgeDocumentSortUpdate `json:"updates" binding:"required"` // 排序更新列表
}

// KnowledgeDocumentSortUpdate 文档排序更新项
type KnowledgeDocumentSortUpdate struct {
	ID        int64  `json:"id" binding:"required" example:"1"`        // 文档ID
	ParentID  int64  `json:"parent_id" example:"0"`                    // 父文档ID（0表示根目录）
	SortOrder int    `json:"sort_order" example:"0"`                   // 排序字段
	Path      string `json:"path" example:"/目录1/文档"`              // 路径
}

