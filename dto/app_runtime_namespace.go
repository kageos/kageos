package dto

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

type NamespaceCreateReq struct {
	Name string `json:"name" binding:"required" example:"my-namespace"` // 命名空间名称
}
type NamespaceCreateResp struct {
	Success bool   `json:"success" example:"true"`     // 是否成功
	Message string `json:"message" example:"命名空间创建成功"` // 响应消息
}

type CreateAppReq struct {
	User     string `json:"user" swaggerignore:"true"`                    // 租户用户名（应用所有者，决定应用的所有权）- 内部字段，不显示在文档中
	Code     string `json:"code" binding:"required" example:"myapp"`      // 应用名
	Name     string `json:"name" binding:"required" example:"腾讯oa系统"`     // 应用名
	IsPublic *bool  `json:"is_public,omitempty" example:"true"`           // 是否公开，默认 true（公开）
	Admins   string `json:"admins,omitempty" example:"user1,user2,user3"` // 管理员列表，逗号分隔的用户名
}

type CreateAppResp struct {
	User   string `json:"user" example:"beiluo"`                    // 用户名
	App    string `json:"app" example:"myapp"`                      // 应用名
	AppDir string `json:"app_dir" example:"namespace/beiluo/myapp"` // 应用目录
}

// RequestAppReq 请求应用
type RequestAppReq struct {
	TraceId     string `json:"trace_id" example:"req-123456"` // 追踪ID（由中间件自动填充）
	IsCallback  bool   `json:"is_callback" example:"true"`
	RequestUser string `json:"request_user" swaggerignore:"true"`          // 请求用户（由中间件自动填充）
	Token       string `json:"token" swaggerignore:"true"`                 // 认证 Token（由中间件自动填充，透传到 SDK）
	User        string `json:"user" binding:"required" example:"beiluo"`   // 租户用户名（应用所有者）
	App         string `json:"app" binding:"required" example:"myapp"`     // 应用名
	Version     string `json:"version" binding:"required" example:"v1"`    // 版本号
	Router      string `json:"router" binding:"required" example:"/users"` // 路由路径
	Method      string `json:"method" example:"GET"`                       // 应用内部方法名（可选）
	Body        []byte `json:"body" example:"eyJpZCI6MX0="`                // 请求体（Base64编码）
	UrlQuery    string `json:"url_query" example:"page=1&size=10"`         // URL 查询参数
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

// CreateFunctionInfo 创建函数信息
type CreateFunctionInfo struct {
	Package    string `json:"package"`     // 目标 package 路径（如 "crm" 或 "plugins/cashier"）
	GroupCode  string `json:"group_code"`  // 函数组代码（文件名，不含 .go）
	SourceCode string `json:"source_code"` // 源代码内容
}

// CreateFunctionsResp 创建函数响应
type CreateFunctionsResp struct {
	Success      bool     `json:"success" example:"true"`   // 是否成功
	Message      string   `json:"message" example:"文件创建成功"` // 响应消息
	WrittenFiles []string `json:"written_files"`            // 已写入的文件路径列表（用于失败时回滚）
}

// UpdateAppReq 更新应用请求
type UpdateAppReq struct {
	User              string                `json:"user" swaggerignore:"true"`              // 用户名
	App               string                `json:"app" binding:"required" example:"myapp"` // 应用名
	ForkPackages      []*ForkPackageInfo    `json:"fork_packages,omitempty"`                // 可选的 Fork 包列表（如果有，先执行 fork 再更新）
	CreateFunctions   []*CreateFunctionInfo `json:"create_functions,omitempty"`             // 可选的新建函数列表（如果有，先执行创建函数再更新）
	Requirement       string                `json:"requirement,omitempty"`                  // 变更需求（用户在前端输入的）
	ChangeDescription string                `json:"change_description,omitempty"`           // 变更描述（大模型输出的）
	Summary           string                `json:"summary,omitempty"`                      // 变更摘要（详情），兼容旧字段，如果未提供则使用 Requirement + ChangeDescription 组合
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
}

type DiffData struct {
	Add    []*ApiInfo `json:"add"`    // 新增的API
	Update []*ApiInfo `json:"update"` // 修改的API
	Delete []*ApiInfo `json:"delete"` // 删除的API
}

// GetAddFullGroupCodes 获取此次变更新增的group code，一个group code 表示新增了一个文件，新增了一个业务系统
func (d *DiffData) GetAddFullGroupCodes() []string {
	codes := make([]string, 0)
	for _, info := range d.Add {
		codes = append(codes, info.BuildFullGroupCode())
	}
	return codes
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
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	Desc              string   `json:"desc"`
	Tags              []string `json:"tags"`
	Router            string   `json:"router"`
	Method            string   `json:"method"`
	CreateTables      []string `json:"create_tables"`
	Callback          []string `json:"callback"`
	FunctionGroupCode string   `json:"function_group_code"`
	FunctionGroupName string   `json:"function_group_name"`

	Request        []*widget.Field `json:"request"`
	Response       []*widget.Field `json:"response"`
	AddedVersion   string          `json:"added_version"`   // API首次添加的版本
	UpdateVersions []string        `json:"update_versions"` // API更新过的版本列表
	TemplateType   string          `json:"template_type"`
	User           string          `json:"user"`
	App            string          `json:"app"`
	FullCodePath   string          `json:"full_code_path"`
	TreeID         int64           `json:"tree_id"` // ServiceTree节点ID，创建tree后赋值，方便后续写快照时入库

	SourceCodeFilePath string        `json:"source_code_file_path"`
	SourceCode         string        `json:"source_code"`
	CreateTableModels  []interface{} `json:"-"`
}

// BuildFullGroupCode 完整函数组代码：{full_path}/{group_code}，与 source_code.full_group_code 对齐
func (a *ApiInfo) BuildFullGroupCode() string {
	return fmt.Sprintf("%s/%s", a.GetParentFullCodePath(), a.FunctionGroupCode)
}

// DeleteAppReq 删除应用请求
type DeleteAppReq struct {
	User string `json:"user" binding:"required" example:"beiluo"` // 租户名
	App  string `json:"app" binding:"required" example:"myapp"`   // 应用名
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
}

// GetAppsResp 获取应用列表响应
type GetAppsResp struct {
	PageInfoResp
}

// AppInfo 应用信息
type AppInfo struct {
	ID        int64  `json:"id" example:"1"`                           // 应用ID
	User      string `json:"user" example:"beiluo"`                    // 租户名
	Code      string `json:"code" example:"myapp"`                     // 应用代码
	Name      string `json:"name" example:"我的应用"`                      // 应用名称
	Status    string `json:"status" example:"enabled"`                 // 状态: enabled(启用), disabled(禁用)
	Version   string `json:"version" example:"v1"`                     // 版本
	NatsID    int64  `json:"nats_id" example:"1"`                      // NATS ID
	HostID    int64  `json:"host_id" example:"1"`                      // 主机ID
	IsPublic  bool   `json:"is_public" example:"true"`                 // 是否公开
	Admins    string `json:"admins,omitempty" example:"user1,user2"`   // 管理员列表，逗号分隔的用户名
	CreatedAt string `json:"created_at" example:"2006-01-02 15:04:05"` // 创建时间
	UpdatedAt string `json:"updated_at" example:"2006-01-02 15:04:05"` // 更新时间
}

// GetAppDetailReq 获取应用详情请求
type GetAppDetailReq struct {
	User string `json:"user" swaggerignore:"true"`                         // 租户名（从JWT Token获取）
	App  string `json:"app" form:"app" binding:"required" example:"myapp"` // 应用代码
}

// GetAppDetailResp 获取应用详情响应
type GetAppDetailResp struct {
	AppInfo
}

// GetAppWithServiceTreeReq 获取应用详情和服务目录树请求
type GetAppWithServiceTreeReq struct {
	User string `json:"user" swaggerignore:"true"`                         // 租户名（从JWT Token获取）
	App  string `json:"app" form:"app" binding:"required" example:"myapp"` // 应用代码
	Type string `json:"type" form:"type" example:"package"`                // 节点类型过滤（可选），如：package（只显示服务目录/包）、function（只显示函数/文件）
}

// GetAppWithServiceTreeResp 获取应用详情和服务目录树响应
type GetAppWithServiceTreeResp struct {
	App         AppInfo               `json:"app"`          // 应用详情
	ServiceTree []*GetServiceTreeResp `json:"service_tree"` // 服务目录树
}
