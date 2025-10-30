package dto

import "strings"

// CreateServiceTreeRuntimeReq 创建服务目录运行时请求
type CreateServiceTreeRuntimeReq struct {
	User        string                  `json:"user" example:"beiluo"` // 用户名
	App         string                  `json:"app" example:"myapp"`   // 应用名
	ServiceTree *ServiceTreeRuntimeData `json:"service_tree"`          // 服务目录数据
}

// ServiceTreeRuntimeData 服务目录运行时数据
type ServiceTreeRuntimeData struct {
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
}

// /user/app/test-> /test
func (s *ServiceTreeRuntimeData) GetSubPath() string {
	if s.FullCodePath == "" {
		return "/"
	}

	// 去掉首尾斜杠并分割路径
	pathParts := strings.Split(strings.Trim(s.FullCodePath, "/"), "/")
	if len(pathParts) <= 2 {
		return ""
	}

	// 返回去掉前两个部分（user/app）后的路径
	subParts := pathParts[2:]
	return "/" + strings.Join(subParts, "/")
}

// CreateServiceTreeRuntimeResp 创建服务目录运行时响应
type CreateServiceTreeRuntimeResp struct {
	User        string `json:"user" example:"beiluo"`       // 用户名
	App         string `json:"app" example:"myapp"`         // 应用名
	ServiceTree string `json:"service_tree" example:"user"` // 服务目录名称
}
