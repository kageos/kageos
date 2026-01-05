package dto

// GetFunctionReq 获取函数详情请求
type GetFunctionReq struct {
	FunctionID int64 `json:"function_id" binding:"required" example:"1"` // 函数ID
}

// GetFunctionResp 获取函数详情响应
type GetFunctionResp struct {
	ID           int64                  `json:"id" example:"1"`                                            // 函数ID
	AppID        int64                  `json:"app_id" example:"1"`                                        // 应用ID
	TreeID       int64                  `json:"tree_id" example:"1"`                                       // 服务目录ID
	Method       string                 `json:"method" example:"GET"`                                      // HTTP方法
	Router       string                 `json:"router" example:"/crm/crm_ticket"`                          // 路由路径
	HasConfig    bool                   `json:"has_config" example:"true"`                                 // 是否有配置
	CreateTables string                 `json:"create_tables" example:"users,orders"`                      // 创建的表
	Callbacks    string                 `json:"callbacks" example:"onCreate,onUpdate"`                     // 回调函数
	TemplateType string                 `json:"template_type" example:"default"`                           // 模板类型
	Request      interface{}            `json:"request"`                                                   // 请求配置（JSON对象）
	Response     interface{}            `json:"response"`                                                  // 响应配置（JSON对象）
	CreatedAt    string                 `json:"created_at" example:"2024-01-01T00:00:00Z"`                 // 创建时间
	UpdatedAt    string                 `json:"updated_at" example:"2024-01-01T00:00:00Z"`                 // 更新时间
	CreatedBy    string                 `json:"created_by" example:"beiluo"`                                // 创建者用户名
	FullCodePath string                 `json:"full_code_path" example:"/beiluo/testapi18/crm/crm_ticket"` //
	Permissions  map[string]bool        `json:"permissions,omitempty"`                                     // ⭐ 权限标识（企业版功能）：权限点 -> 是否有权限（按需查询，不在服务树中查询）

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
	Name         string `json:"name" example:"工单管理"`                       // 函数名称（从 ServiceTree 获取）
	Description  string `json:"description" example:"用于管理工单的创建、更新、删除等功能"`  // 函数描述（从 ServiceTree 获取）
}

// GetFunctionGroupInfoResp 获取函数组信息响应（用于 Hub 发布）
type GetFunctionGroupInfoResp struct {
	// 核心数据（用于 clone）
	SourceCode  string `json:"source_code" example:"package main..."` // 源代码（无状态，用于 clone）
	Description string `json:"description" example:"收银相关的工具函数"`       // 描述信息

	// 快照信息（方便排查问题）
	FullGroupCode string         `json:"full_group_code" example:"/luobei/testgroup/plugins/tools_cashier"` // 完整函数组代码
	GroupCode     string         `json:"group_code" example:"tools_cashier"`                                // 函数组代码
	GroupName     string         `json:"group_name" example:"收银工具"`                                         // 函数组名称
	FullPath      string         `json:"full_path" example:"/luobei/testgroup/plugins"`                     // 完整路径
	Version       string         `json:"version" example:"v1"`                                              // 版本号
	AppID         int64          `json:"app_id" example:"123"`                                              // 应用ID
	AppName       string         `json:"app_name" example:"testgroup"`                                      // 应用名称
	FunctionCount int            `json:"function_count" example:"3"`                                        // 函数数量（快照）
	Functions     []FunctionInfo `json:"functions"`                                                         // 函数列表（用于 Hub 展示功能列表）
}
