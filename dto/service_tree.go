package dto

// CreateServiceTreeReq 创建服务目录请求
type CreateServiceTreeReq struct {
	User        string `json:"user" binding:"required"`
	App         string `json:"app" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Name        string `json:"name" binding:"required"`
	ParentID    int64  `json:"parent_id"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
}

// CreateServiceTreeResp 创建服务目录响应
type CreateServiceTreeResp struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Name         string `json:"name"`
	ParentID     int64  `json:"parent_id"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Tags         string `json:"tags"`
	AppID        int64  `json:"app_id"`
	FullIDPath   string `json:"full_id_path"`
	FullNamePath string `json:"full_name_path"`
	Status       string `json:"status"`
}

// GetServiceTreeResp 获取服务目录响应
type GetServiceTreeResp struct {
	ID           int64                 `json:"id"`
	Title        string                `json:"title"`
	Name         string                `json:"name"`
	ParentID     int64                 `json:"parent_id"`
	Type         string                `json:"type"`
	Description  string                `json:"description"`
	Tags         string                `json:"tags"`
	AppID        int64                 `json:"app_id"`
	FullIDPath   string                `json:"full_id_path"`
	FullNamePath string                `json:"full_name_path"`
	Children     []*GetServiceTreeResp `json:"children,omitempty"`
}

// UpdateServiceTreeReq 更新服务目录请求
type UpdateServiceTreeReq struct {
	ID          int64  `json:"id" binding:"required"`
	Title       string `json:"title"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
}

// DeleteServiceTreeReq 删除服务目录请求
type DeleteServiceTreeReq struct {
	ID int64 `json:"id" binding:"required"`
}
