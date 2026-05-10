package dto

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
)

type NamespaceCreateReq struct {
	Name string `json:"name" binding:"required" example:"my-namespace"` // 命名空间名称
}
type NamespaceCreateResp struct {
	Success bool   `json:"success" example:"true"`     // 是否成功
	Message string `json:"message" example:"命名空间创建成功"` // 响应消息
}

type CreateAppReq struct {
	User               string `json:"user" swaggerignore:"true"`                     // 租户用户名（应用所有者，决定应用的所有权）- 内部字段，不显示在文档中
	Code               string `json:"code" binding:"required" example:"myapp"`       // 应用名
	Name               string `json:"name" binding:"required" example:"腾讯oa系统"`      // 应用名
	IsPublic           *bool  `json:"is_public,omitempty" example:"true"`            // 是否公开，默认 true（公开）
	Admins             string `json:"admins,omitempty" example:"user1,user2,user3"`  // 管理员列表，逗号分隔的用户名
	ShowOnlyPermitted  *bool  `json:"show_only_permitted,omitempty" example:"false"` // 仅展示有权限的空间：开启后非管理员只看到有权限的目录
	PermissionEnforced *bool  `json:"permission_enforced,omitempty" example:"false"` // 是否启用权限管控：社区版默认 false，企业版权限特性默认 true
}

type CreateAppResp struct {
	User   string `json:"user" example:"beiluo"`                    // 用户名
	App    string `json:"app" example:"myapp"`                      // 应用名
	AppDir string `json:"app_dir" example:"namespace/beiluo/myapp"` // 应用目录
}

// RequestAppReq 请求应用
type RequestAppReq struct {
	TraceId         string `json:"trace_id" example:"req-123456"` // 追踪ID（由中间件自动填充）
	IsCallback      bool   `json:"is_callback" example:"true"`
	RequestUser     string `json:"request_user" swaggerignore:"true"`            // 请求用户（由中间件自动填充）
	RequestUserDept string `json:"request_user_dept" swaggerignore:"true"`       // 请求用户部门（由中间件自动填充）
	Token           string `json:"token" swaggerignore:"true"`                   // 认证 Token（由中间件自动填充，透传到 SDK）
	ClientSource    string `json:"client_source,omitempty" swaggerignore:"true"` // 客户端来源（browser、scheduled_task、agent、api）
	User            string `json:"user" binding:"required" example:"beiluo"`     // 租户用户名（应用所有者）
	App             string `json:"app" binding:"required" example:"myapp"`       // 应用名
	Version         string `json:"version" binding:"required" example:"v1"`      // 版本号
	Router          string `json:"router" binding:"required" example:"/users"`   // 路由路径
	Method          string `json:"method" example:"GET"`                         // 应用内部方法名（可选）
	Body            []byte `json:"body" example:"eyJpZCI6MX0="`                  // 请求体（Base64编码）
	UrlQuery        string `json:"url_query" example:"page=1&size=10"`           // URL 查询参数
}

// CallbackAppReq 回调请求
type CallbackAppReq struct {
	Type   string      `json:"type" binding:"required" example:""`
	Router string      `json:"router" binding:"required" example:"/users/app/xxxx"` // 路由路径
	Body   interface{} `json:"body" example:"eyJpZCI6MX0="`                         // 请求体（Base64编码）
}

// RequestAppResp 应用响应
type RequestAppResp struct {
	TraceId string      `json:"trace_id" example:"req-123456"` // 追踪ID
	Version string      `json:"version" example:"v1"`
	Result  interface{} `json:"result,omitempty"`                 // 结果
	Error   string      `json:"error,omitempty" example:"应用内部错误"` // 错误信息
	ErrCode int         `json:"err_code" example:"0"`             //0 是正常，>0 是系统错误，<0 是业务错误，业务错误用户自己处理，系统错误需要考虑用ai来分析代码是哪里出了问题
}

func (r *RequestAppResp) IsError() bool {
	return r.ErrCode != 0
}

// SourceFileWrite 源码文件写入描述
// runtime 底层只关心“往某个目录写入一个源码文件”，不直接承载“函数”业务语义。
type SourceFileWrite struct {
	DirectoryPath string `json:"directory_path"` // 目标目录路径（相对于 code/api，如 "crm" 或 "plugins/cashier"）
	FileName      string `json:"file_name"`      // 文件名（不含 .go 扩展名）
	SourceCode    string `json:"source_code"`    // 源代码内容
}

// WriteSourceFilesResp 源码文件写入响应
type WriteSourceFilesResp struct {
	Success      bool     `json:"success" example:"true"`   // 是否成功
	Message      string   `json:"message" example:"文件创建成功"` // 响应消息
	WrittenFiles []string `json:"written_files"`            // 已写入的文件路径列表（用于失败时回滚）
}

// UpdateAppReq 更新应用请求（更新应用代码并重新编译部署）
type UpdateAppReq struct {
	User              string             `json:"user,omitempty"`               // 租户用户名（兼容字段，优先从 resource_path 解析）
	App               string             `json:"app,omitempty"`                // 应用名（兼容字段，优先从 resource_path 解析）
	ResourcePath      string             `json:"resource_path,omitempty"`      // 资源路径，规范为 /user/app
	SourceFiles       []*SourceFileWrite `json:"source_files,omitempty"`       // 推荐字段：本次需要写入的源码文件列表
	CreateFunctions   []*SourceFileWrite `json:"create_functions,omitempty"`   // 兼容旧字段：历史命名为创建函数，实际语义为写入源码文件
	Requirement       string             `json:"requirement,omitempty"`        // 变更需求（用户在前端输入的）
	ChangeDescription string             `json:"change_description,omitempty"` // 变更描述（大模型输出的）
	Summary           string             `json:"summary,omitempty"`            // 变更摘要（详情），兼容旧字段，如果未提供则使用 Requirement + ChangeDescription 组合
	WriteOnly         bool               `json:"write_only,omitempty"`         // 为 true 时仅写文件不编译不部署
	SkipBuild         bool               `json:"skip_build,omitempty"`         // 兼容旧字段：等价于 write_only
	ForceDiff         bool               `json:"force_diff,omitempty"`         // 为 true 时清理 api-logs，让本次更新重新产生 add diff
}

// BuildSummary 返回本次更新的摘要信息，优先使用 Summary。
func (r *UpdateAppReq) BuildSummary() string {
	if r == nil {
		return ""
	}

	if r.Summary != "" {
		return r.Summary
	}

	if r.Requirement != "" && r.ChangeDescription != "" {
		return "需求：" + r.Requirement + "\n\n变更描述：" + r.ChangeDescription
	}
	if r.Requirement != "" {
		return r.Requirement
	}
	if r.ChangeDescription != "" {
		return r.ChangeDescription
	}

	return ""
}

// RequestedSourceFiles 返回本次请求携带的源码文件列表，优先使用新字段 source_files。
func (r *UpdateAppReq) RequestedSourceFiles() []*SourceFileWrite {
	if r == nil {
		return nil
	}
	if len(r.SourceFiles) > 0 {
		return r.SourceFiles
	}
	return r.CreateFunctions
}

// ShouldWriteOnly 返回本次更新是否仅写文件不发布，优先使用新字段 write_only。
func (r *UpdateAppReq) ShouldWriteOnly() bool {
	if r == nil {
		return false
	}
	return r.WriteOnly || r.SkipBuild
}

// UpdateAppResp 更新应用响应
type UpdateAppResp struct {
	User          string    `json:"user" example:"beiluo"`     // 用户名
	App           string    `json:"app" example:"myapp"`       // 应用名
	OldVersion    string    `json:"old_version" example:"v1"`  // 旧版本号
	NewVersion    string    `json:"new_version" example:"v2"`  // 新版本号
	GitCommitHash string    `json:"git_commit_hash,omitempty"` // Git 提交哈希（用于回滚）
	Diff          *DiffData `json:"diff,omitempty"`            // API diff 信息
	Error         string    `json:"error,omitempty"`           // 回调过程中的错误信息
	Warnings      []string  `json:"warnings,omitempty"`        // 非阻断告警（如发布成功但元数据同步失败）
}

// PackageInfo SDK 返回的 package 元信息，app-server 用于目录对账
type PackageInfo struct {
	Code        string `json:"code"`         // 目录名（如 "pdf"）
	Name        string `json:"name"`         // 显示名称
	Desc        string `json:"desc"`         // 描述
	RouterGroup string `json:"router_group"` // 路由组路径（如 "/plugins/pdf"）
	FullPath    string `json:"full_path"`    // 完整路径（如 "/user/app/plugins/pdf"）
}

type DiffData struct {
	Add      []*ApiInfo     `json:"add"`                // 新增的API
	Update   []*ApiInfo     `json:"update"`             // 修改的API
	Delete   []*ApiInfo     `json:"delete"`             // 删除的API
	Packages []*PackageInfo `json:"packages,omitempty"` // 全量 package 列表，每次 update 都返回，用于 app-server 目录对账
}

// GetAddFullCodePaths 获取新增 API 的完整代码路径列表
func (d *DiffData) GetAddFullCodePaths() []string {
	if d == nil || len(d.Add) == 0 {
		return []string{}
	}

	fullCodePaths := make([]string, 0, len(d.Add))
	for _, api := range d.Add {
		if api != nil {
			fullCodePath := api.FullCodePath
			if fullCodePath == "" {
				// 如果 FullCodePath 为空，尝试构建
				fullCodePath = api.BuildFullCodePath()
			}
			if fullCodePath != "" {
				fullCodePaths = append(fullCodePaths, fullCodePath)
			}
		}
	}
	return fullCodePaths
}

func (a *ApiInfo) BuildFullCodePath() string {
	router := strings.Trim(a.Router, "/")
	if router == "" {
		return fmt.Sprintf("/%s/%s", a.User, a.App)
	}
	return fmt.Sprintf("/%s/%s/%s", a.User, a.App, router)
}
func (a *ApiInfo) GetParentFullCodePath() string {
	if a.FullCodePath == "" {
		return ""
	}

	// 去掉末尾的斜杠并分割路径
	pathParts := strings.Split(strings.Trim(a.FullCodePath, "/"), "/")
	if len(pathParts) <= 1 {
		return ""
	}

	// 返回父级路径（去掉最后一个部分）
	parentParts := pathParts[:len(pathParts)-1]
	if len(parentParts) == 0 {
		return ""
	}
	return "/" + strings.Join(parentParts, "/")
}

// GetAppPrefix 获取应用前缀
func (a *ApiInfo) GetAppPrefix() string {
	return fmt.Sprintf("/%s/%s", a.User, a.App)
}

// GetRelativePath 获取相对于应用根目录的路径
func (a *ApiInfo) GetRelativePath() string {
	if a.FullCodePath == "" {
		return ""
	}

	appPrefix := a.GetAppPrefix()
	if strings.HasPrefix(a.FullCodePath, appPrefix) {
		return strings.TrimPrefix(a.FullCodePath, appPrefix)
	}
	return a.FullCodePath
}

// GetFunctionName 获取函数名称（从code字段获取，底层从路由最后一个部分赋值）
func (a *ApiInfo) GetFunctionName() string {
	return a.Code
}

// GetPackagePath 获取包路径（不包含函数名）
func (a *ApiInfo) GetPackagePath() string {
	parentPath := a.GetParentFullCodePath()
	appPrefix := a.GetAppPrefix()

	// 如果父级路径就是应用根目录，返回应用前缀
	if parentPath == "" || parentPath == appPrefix {
		return appPrefix
	}

	return parentPath
}

// GetPackageChain 获取包链（从应用到函数的各级包名）
func (a *ApiInfo) GetPackageChain() []string {
	relativePath := a.GetRelativePath()
	if relativePath == "" || relativePath == "/" {
		return []string{}
	}

	parts := strings.Split(strings.Trim(relativePath, "/"), "/")
	if len(parts) <= 1 {
		return []string{}
	}

	// 排除最后一个函数名，只返回包链
	return parts[:len(parts)-1]
}

type ApiInfo struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Desc         string   `json:"desc"`
	Tags         []string `json:"tags"`
	Router       string   `json:"router"`
	Method       string   `json:"method"`
	CreateTables []string `json:"create_tables"`
	// FunctionGroupCode 和 FunctionGroupName 已移除，不再需要

	Schema         *functionschema.FunctionSchema `json:"schema"`
	AddedVersion   string                         `json:"added_version"`   // API首次添加的版本
	UpdateVersions []string                       `json:"update_versions"` // API更新过的版本列表
	TemplateType   string                         `json:"template_type"`
	User           string                         `json:"user"`
	App            string                         `json:"app"`
	FullCodePath   string                         `json:"full_code_path"`
	TreeID         int64                          `json:"tree_id"` // ServiceTree节点ID，创建tree后赋值，方便后续写快照时入库

}

// DeleteAppReq 删除应用请求
type DeleteAppReq struct {
	User         string `json:"user,omitempty" example:"beiluo"`                 // 租户名（兼容字段，优先从 resource_path 解析）
	App          string `json:"app,omitempty" example:"myapp"`                   // 应用名（兼容字段，优先从 resource_path 解析）
	ResourcePath string `json:"resource_path,omitempty" example:"/beiluo/myapp"` // 工作空间资源路径，规范为 /user/app
}

// DeleteAppResp 删除应用响应
type DeleteAppResp struct {
	User string `json:"user" example:"beiluo"` // 租户名
	App  string `json:"app" example:"myapp"`   // 应用名
}

// GetAppsReq 获取应用列表请求
type GetAppsReq struct {
	PageInfoReq
	User       string `json:"user" swaggerignore:"true"`      // 租户名（从JWT Token获取）
	Search     string `json:"search" form:"search"`           // 搜索关键词（支持按应用名称或代码搜索）
	IncludeAll bool   `json:"include_all" form:"include_all"` // 是否包含所有公开的工作空间（true: 显示自己的+全部公开的，false: 只显示自己的）
	Type       *int   `json:"type,omitempty" form:"type"`     // 应用类型筛选（可选）：0=用户空间，1=系统空间。如果为 nil，不筛选类型
}

// GetAppsResp 获取应用列表响应
type GetAppsResp struct {
	PageInfoResp
}

// AppInfo 应用信息
type AppInfo struct {
	ID                 int64  `json:"id" example:"1"`                           // 应用ID
	User               string `json:"user" example:"beiluo"`                    // 租户名
	Code               string `json:"code" example:"myapp"`                     // 应用代码
	Name               string `json:"name" example:"我的应用"`                      // 应用名称
	Status             string `json:"status" example:"enabled"`                 // 状态: enabled(启用), disabled(禁用)
	Version            string `json:"version" example:"v1"`                     // 版本
	NatsID             int64  `json:"nats_id" example:"1"`                      // NATS ID
	HostID             int64  `json:"host_id" example:"1"`                      // 主机ID
	IsPublic           bool   `json:"is_public" example:"true"`                 // 是否公开
	Admins             string `json:"admins,omitempty" example:"user1,user2"`   // 管理员列表，逗号分隔的用户名
	Type               int    `json:"type" example:"0"`                         // 应用类型：0=用户空间，1=系统空间
	ShowOnlyPermitted  bool   `json:"show_only_permitted" example:"false"`      // 仅展示有权限的空间
	PermissionEnforced bool   `json:"permission_enforced" example:"false"`      // 是否启用权限管控
	CreatedAt          string `json:"created_at" example:"2006-01-02 15:04:05"` // 创建时间
	UpdatedAt          string `json:"updated_at" example:"2006-01-02 15:04:05"` // 更新时间
}

// GetAppDetailReq 获取应用详情请求
type GetAppDetailReq struct {
	User         string `json:"user,omitempty" swaggerignore:"true"`
	App          string `json:"app,omitempty" form:"app,omitempty" example:"myapp"`
	ResourcePath string `json:"resource_path,omitempty" form:"resource_path,omitempty"`
}

// GetAppDetailResp 获取应用详情响应
type GetAppDetailResp struct {
	AppInfo
}

// GetAppWithServiceTreeReq 获取应用详情和服务目录树请求
type GetAppWithServiceTreeReq struct {
	User         string `json:"user,omitempty" swaggerignore:"true"`                    // 租户名（兼容字段，优先从 resource_path 解析）
	App          string `json:"app,omitempty" form:"app,omitempty" example:"myapp"`     // 应用代码（兼容字段，优先从 resource_path 解析）
	ResourcePath string `json:"resource_path,omitempty" form:"resource_path,omitempty"` // 工作空间资源路径，规范为 /user/app
	Type         string `json:"type" form:"type" example:"package"`                     // 节点类型过滤（可选），如：package（只显示服务目录/包）、function（只显示函数/文件）
}

// GetAppWithServiceTreeResp 获取应用详情和服务目录树响应
type GetAppWithServiceTreeResp struct {
	App          AppInfo               `json:"app"`                     // 应用详情
	ServiceTree  []*GetServiceTreeResp `json:"service_tree"`            // 服务目录树
	ExpandedKeys []int64               `json:"expanded_keys,omitempty"` // ⭐ 需要自动展开的节点ID列表（包含所有 pending_count > 0 的节点及其父节点）
}
