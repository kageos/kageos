package dto

import (
	"strings"
)

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

// DirectoryTreeItem 目录树项（支持目录和文件）
type DirectoryTreeItem struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 完整代码路径，如 /user/app/plugins/cashier
	Type         string `json:"type" binding:"required"`            // 类型：directory（目录）或 file（文件）

	// 目录相关字段（Type = directory 时使用）
	Name        string `json:"name,omitempty"`        // 目录名称
	Description string `json:"description,omitempty"`  // 目录描述
	Tags        string `json:"tags,omitempty"`        // 目录标签

	// 文件相关字段（Type = file 时使用）
	FileName     string `json:"file_name,omitempty"`    // 文件名（不含 .go 后缀）
	FileType     string `json:"file_type,omitempty"`    // 文件类型（go, json, yaml 等）
	Content      string `json:"content,omitempty"`      // 文件内容
	RelativePath string `json:"relative_path,omitempty"` // 文件相对路径（如 user.go 或 subdir/user.go）
}

// BatchCreateDirectoryTreeRuntimeReq 批量创建目录树运行时请求
type BatchCreateDirectoryTreeRuntimeReq struct {
	User  string                `json:"user"`  // 用户名
	App   string                `json:"app"`   // 应用名
	Items []*DirectoryTreeItem  `json:"items"` // 目录树项列表
}

// BatchCreateDirectoryTreeRuntimeResp 批量创建目录树运行时响应
type BatchCreateDirectoryTreeRuntimeResp struct {
	DirectoryCount int      `json:"directory_count"` // 创建的目录数量
	FileCount      int      `json:"file_count"`      // 创建的文件数量
	CreatedPaths   []string `json:"created_paths"`   // 创建的路径列表
}

// ServiceTreeNodeOperation 服务树节点操作类型
const (
	ServiceTreeNodeOpAdd    = "add"    // 新增节点
	ServiceTreeNodeOpUpdate = "update" // 更新节点
	ServiceTreeNodeOpDelete = "delete" // 删除节点
)

// ServiceTreeNode 服务树节点（用于 UpdateServiceTree）
type ServiceTreeNode struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 完整代码路径
	Type         string `json:"type" binding:"required"`            // 类型：directory（目录）或 file（文件）
	Operation    string `json:"operation" binding:"required"`       // 操作类型：add（新增）、update（更新）、delete（删除）

	// 目录相关字段（Type = directory 时使用）
	Name        string `json:"name,omitempty"`        // 目录名称
	Description string `json:"description,omitempty"`  // 目录描述
	Tags        string `json:"tags,omitempty"`        // 目录标签

	// 文件相关字段（Type = file 时使用）
	FileName     string `json:"file_name,omitempty"`    // 文件名（不含扩展名）
	FileType     string `json:"file_type,omitempty"`    // 文件类型（go, json, yaml 等）
	Content      string `json:"content,omitempty"`       // 文件内容（新增或更新时需要）
	RelativePath string `json:"relative_path,omitempty"` // 文件相对路径
}

// UpdateServiceTreeRuntimeReq 更新服务树运行时请求
type UpdateServiceTreeRuntimeReq struct {
	User  string             `json:"user"`  // 用户名
	App   string             `json:"app"`   // 应用名
	Nodes []*ServiceTreeNode `json:"nodes"` // 服务树节点列表
}

// UpdateServiceTreeRuntimeResp 更新服务树运行时响应
type UpdateServiceTreeRuntimeResp struct {
	DirectoryCount int      `json:"directory_count"` // 处理的目录数量
	FileCount      int      `json:"file_count"`      // 处理的文件数量
	Diff           *DiffData `json:"diff,omitempty"`  // API diff 信息（如果有文件变更）
	OldVersion     string   `json:"old_version"`     // 旧版本号
	NewVersion     string   `json:"new_version"`     // 新版本号
	GitCommitHash  string   `json:"git_commit_hash,omitempty"` // Git 提交哈希
}
