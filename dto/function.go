package dto

// GetFunctionReq 获取函数详情请求
type GetFunctionReq struct {
	FunctionID int64 `json:"function_id" binding:"required" example:"1"` // 函数ID
}

// GetFunctionResp 获取函数详情响应
type GetFunctionResp struct {
	ID           int64       `json:"id" example:"1"`                                            // 函数ID
	AppID        int64       `json:"app_id" example:"1"`                                        // 应用ID
	TreeID       int64       `json:"tree_id" example:"1"`                                       // 服务目录ID
	Method       string      `json:"method" example:"GET"`                                      // HTTP方法
	Router       string      `json:"router" example:"/crm/crm_ticket"`                          // 路由路径
	HasConfig    bool        `json:"has_config" example:"true"`                                 // 是否有配置
	CreateTables string      `json:"create_tables" example:"users,orders"`                      // 创建的表
	Callbacks    string      `json:"callbacks" example:"onCreate,onUpdate"`                     // 回调函数
	TemplateType string      `json:"template_type" example:"default"`                           // 模板类型
	Request      interface{} `json:"request"`                                                   // 请求配置（JSON对象）
	Response     interface{} `json:"response"`                                                  // 响应配置（JSON对象）
	CreatedAt    string      `json:"created_at" example:"2024-01-01T00:00:00Z"`                 // 创建时间
	UpdatedAt    string      `json:"updated_at" example:"2024-01-01T00:00:00Z"`                 // 更新时间
	FullCodePath string      `json:"full_code_path" example:"/beiluo/testapi18/crm/crm_ticket"` //

}

// GetFunctionsByAppReq 获取应用下所有函数请求
type GetFunctionsByAppReq struct {
	AppID int64 `json:"app_id" binding:"required" example:"1"` // 应用ID
}

// GetFunctionsByAppResp 获取应用下所有函数响应
type GetFunctionsByAppResp struct {
	Functions []FunctionInfo `json:"functions"` // 函数列表
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	ID           int64  `json:"id" example:"1"`                            // 函数ID
	AppID        int64  `json:"app_id" example:"1"`                        // 应用ID
	TreeID       int64  `json:"tree_id" example:"1"`                       // 服务目录ID
	Method       string `json:"method" example:"GET"`                      // HTTP方法
	Router       string `json:"router" example:"/users"`                   // 路由路径
	HasConfig    bool   `json:"has_config" example:"true"`                 // 是否有配置
	CreateTables string `json:"create_tables" example:"users,orders"`      // 创建的表
	Callbacks    string `json:"callbacks" example:"onCreate,onUpdate"`     // 回调函数
	TemplateType string `json:"template_type" example:"default"`           // 模板类型
	CreatedAt    string `json:"created_at" example:"2024-01-01T00:00:00Z"` // 创建时间
	UpdatedAt    string `json:"updated_at" example:"2024-01-01T00:00:00Z"` // 更新时间
}
