package dto

// CreateServiceTreeReq 创建服务目录请求
type CreateServiceTreeReq struct {
	User        string `json:"user" binding:"required" example:"beiluo"` // 用户名
	App         string `json:"app" binding:"required" example:"myapp"`   // 应用名
	Name        string `json:"name" binding:"required" example:"用户管理"`   // 服务目录名称
	Code        string `json:"code" binding:"required" example:"user"`   // 服务目录代码
	ParentID    int64  `json:"parent_id" example:"0"`                    // 父目录ID，0表示根目录
	Description string `json:"description" example:"用户相关的API接口"`         // 描述
	Tags        string `json:"tags" example:"user,management"`           // 标签
}

// CreateServiceTreeResp 创建服务目录响应
type CreateServiceTreeResp struct {
	ID           int64  `json:"id" example:"1"`                              // 服务目录ID
	Name         string `json:"name" example:"用户管理"`                         // 服务目录名称
	Code         string `json:"code" example:"user"`                         // 服务目录代码
	ParentID     int64  `json:"parent_id" example:"0"`                       // 父目录ID
	Type         string `json:"type" example:"package"`                      // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description  string `json:"description" example:"用户相关的API接口"`            // 描述
	Tags         string `json:"tags" example:"user,management"`              // 标签
	AppID        int64  `json:"app_id" example:"1"`                          // 应用ID
	RefID        int64  `json:"ref_id" example:"0"`                          // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	Status       string `json:"status" example:"enabled"`                    // 状态: enabled(启用), disabled(禁用)
}

// GetServiceTreeResp 获取服务目录响应
type GetServiceTreeResp struct {
	ID           int64                 `json:"id" example:"1"`                              // 服务目录ID
	Name         string                `json:"name" example:"用户管理"`                         // 服务目录名称
	Code         string                `json:"code" example:"user"`                         // 服务目录代码
	ParentID     int64                 `json:"parent_id" example:"0"`                       // 父目录ID
	Type         string                `json:"type" example:"package"`                      // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description  string                `json:"description" example:"用户相关的API接口"`            // 描述
	Tags         string                `json:"tags" example:"user,management"`              // 标签
	AppID        int64                 `json:"app_id" example:"1"`                          // 应用ID
	RefID        int64                 `json:"ref_id" example:"0"`                          // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	FullCodePath string                `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	Children     []*GetServiceTreeResp `json:"children,omitempty"`                          // 子目录列表
}

// UpdateServiceTreeReq 更新服务目录请求
type UpdateServiceTreeReq struct {
	ID          int64  `json:"id" binding:"required" example:"1"` // 服务目录ID
	Name        string `json:"name" example:"用户管理"`               // 服务目录名称
	Code        string `json:"code" example:"user"`               // 服务目录代码
	Description string `json:"description" example:"用户相关的API接口"`  // 描述
	Tags        string `json:"tags" example:"user,management"`    // 标签
}

// DeleteServiceTreeReq 删除服务目录请求
type DeleteServiceTreeReq struct {
	ID int64 `json:"id" binding:"required" example:"1"` // 服务目录ID
}
