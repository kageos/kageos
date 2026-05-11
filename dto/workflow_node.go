package dto

// CreateWorkflowNodeReq 创建 workflow 类型服务树节点请求。
type CreateWorkflowNodeReq struct {
	User               string `json:"user" binding:"required" example:"beiluo"`
	App                string `json:"app" binding:"required" example:"myapp"`
	Name               string `json:"name" binding:"required" example:"客户入库工作流"`
	Code               string `json:"code" binding:"required" example:"customer_onboarding"`
	ParentFullCodePath string `json:"parent_full_code_path" example:"/beiluo/myapp"`
	Description        string `json:"description"`
	Tags               string `json:"tags"`
	Admins             string `json:"admins"`
}

// CreateWorkflowNodeResp 创建 workflow 类型服务树节点响应。
type CreateWorkflowNodeResp struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Type         string `json:"type"` // "workflow"
	Description  string `json:"description"`
	Tags         string `json:"tags"`
	AppID        int64  `json:"app_id"`
	FullCodePath string `json:"full_code_path"`
	Admins       string `json:"admins"`
}

// UpdateWorkflowNodeReq 更新 workflow 类型服务树节点元数据。
type UpdateWorkflowNodeReq struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	Admins      string `json:"admins"`
}
