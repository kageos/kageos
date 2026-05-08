package dto

// CreateBoardReq 创建版块（board）节点请求
type CreateBoardReq struct {
	User               string `json:"user" binding:"required" example:"beiluo"`
	App                string `json:"app" binding:"required" example:"myapp"`
	Name               string `json:"name" binding:"required" example:"讨论区"`
	Code               string `json:"code" binding:"required" example:"board1"`
	ParentFullCodePath string `json:"parent_full_code_path" example:"/beiluo/myapp"` // 父目录完整路径
	Description        string `json:"description"`
	Tags               string `json:"tags"`
	Admins             string `json:"admins"`
}

// CreateBoardResp 创建版块响应
type CreateBoardResp struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Type         string `json:"type"` // "board"
	Description  string `json:"description"`
	Tags         string `json:"tags"`
	AppID        int64  `json:"app_id"`
	FullCodePath string `json:"full_code_path"` // 完整代码路径（父路径可由此推导）
	Admins       string `json:"admins"`
}

// UpdateBoardReq 更新版块请求（名称、描述、标签等，与目录通用）
type UpdateBoardReq struct {
	ID          int64  `json:"id,omitempty"` // 由 path 提供，body 可不传
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	Admins      string `json:"admins"`
}

// CreatePostReq 发帖请求
type CreatePostReq struct {
	FullCodePath  string   `json:"full_code_path" binding:"required"` // 版块完整路径
	Title         string   `json:"title" binding:"required"`
	Summary       string   `json:"summary"`        // 摘要，列表展示；可选，为空时从正文截取
	Cover         []string `json:"cover"`          // 封面图 URL 数组（可多图）
	Content       string   `json:"content"`        // 正文（富文本）
	ContentFormat string   `json:"content_format"` // markdown / html，默认 markdown
	Status        string   `json:"status"`         // draft / published，默认 published
}

// UpdatePostReq 更新帖子请求
type UpdatePostReq struct {
	ID            int64    `json:"id,omitempty"` // 由 path 提供，body 可不传
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Cover         []string `json:"cover"`
	Content       string   `json:"content"`
	ContentFormat string   `json:"content_format"`
	Status        string   `json:"status"`
}

// ListPostsReq 帖子列表请求（query）
type ListPostsReq struct {
	FullCodePath string `form:"full_code_path" binding:"required"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// PostItem 帖子列表项
type PostItem struct {
	ID        int64    `json:"id"`
	TreeID    int64    `json:"tree_id"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"` // 摘要，列表展示
	Cover     []string `json:"cover"`   // 封面图 URL 数组，列表展示
	Author    string   `json:"author"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// ListPostsResp 帖子列表响应
type ListPostsResp struct {
	List  []PostItem `json:"list"`
	Total int64      `json:"total"`
}

// GetPostResp 帖子详情响应
type GetPostResp struct {
	ID            int64    `json:"id"`
	TreeID        int64    `json:"tree_id"`
	FullCodePath  string   `json:"full_code_path"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Cover         []string `json:"cover"`
	Content       string   `json:"content"`
	ContentFormat string   `json:"content_format"`
	Author        string   `json:"author"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}
