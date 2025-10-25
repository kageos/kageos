package dto

// CreateServiceTreeRuntimeReq 创建服务目录运行时请求
type CreateServiceTreeRuntimeReq struct {
	User        string                  `json:"user"`
	App         string                  `json:"app"`
	ServiceTree *ServiceTreeRuntimeData `json:"service_tree"`
}

// ServiceTreeRuntimeData 服务目录运行时数据
type ServiceTreeRuntimeData struct {
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
}

// CreateServiceTreeRuntimeResp 创建服务目录运行时响应
type CreateServiceTreeRuntimeResp struct {
	User        string `json:"user"`
	App         string `json:"app"`
	ServiceTree string `json:"service_tree"` // 服务目录名称
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
}
