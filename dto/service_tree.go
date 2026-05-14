package dto

import "github.com/ai-agent-os/ai-agent-os/pkg/functionschema"

// 注意：DiffData 定义在 dto/app_runtime_namespace.go 中

// CreateServiceTreeReq 创建服务目录请求
type CreateServiceTreeReq struct {
	User               string `json:"user" binding:"required" example:"beiluo"`      // 用户名
	App                string `json:"app" binding:"required" example:"myapp"`        // 应用名
	Name               string `json:"name" binding:"required" example:"用户管理"`        // 服务目录名称
	Code               string `json:"code" binding:"required" example:"user"`        // 服务目录代码
	ParentFullCodePath string `json:"parent_full_code_path" example:"/beiluo/myapp"` // 父目录完整路径，空字符串表示根目录
	Type               string `json:"type" example:"package"`                        // 节点类型: package(服务目录/包), docs(文档), function(函数/文件)
	Description        string `json:"description" example:"用户相关的API接口"`              // 描述
	Tags               string `json:"tags" example:"user,management"`                // 标签
	Admins             string `json:"admins" example:"user1,user2"`                  // 管理员列表，逗号分隔的用户名
	// ⭐ 文档相关字段（仅当 type=docs 时使用）
	DocContent string `json:"doc_content" example:"# 文档内容\n\n这是文档内容..."` // 文档内容（仅 docs 类型）
	DocFormat  string `json:"doc_format" example:"markdown"`             // 文档格式（仅 docs 类型，默认为 markdown）
	DocSummary string `json:"doc_summary" example:"文档摘要"`                // 文档摘要（仅 docs 类型，可选）
}

// CreateServiceTreeResp 创建服务目录响应
type CreateServiceTreeResp struct {
	ID           int64  `json:"id" example:"1"`                              // 服务目录ID
	Name         string `json:"name" example:"用户管理"`                         // 服务目录名称
	Code         string `json:"code" example:"user"`                         // 服务目录代码
	Type         string `json:"type" example:"package"`                      // 节点类型
	Description  string `json:"description" example:"用户相关的API接口"`            // 描述
	Tags         string `json:"tags" example:"user,management"`              // 标签
	AppID        int64  `json:"app_id" example:"1"`                          // 应用ID
	RefID        int64  `json:"ref_id" example:"0"`                          // 引用ID
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径（父路径可由此推导）
	Version      string `json:"version" example:"v1"`                        // 节点当前版本号
	VersionNum   int    `json:"version_num" example:"1"`                     // 节点当前版本号（数字部分）
	Admins       string `json:"admins" example:"user1,user2"`                // 管理员列表
	Status       string `json:"status" example:"enabled"`                    // 状态
}

// GetServiceTreeResp 获取服务目录响应
type GetServiceTreeResp struct {
	ID           int64                 `json:"id,omitempty" example:"1"`                              // 服务目录ID
	Name         string                `json:"name,omitempty" example:"用户管理"`                         // 服务目录名称
	Code         string                `json:"code,omitempty" example:"user"`                         // 服务目录代码
	Type         string                `json:"type,omitempty" example:"package"`                      // 节点类型
	Description  string                `json:"description,omitempty" example:"用户相关的API接口"`            // 描述
	Tags         string                `json:"tags,omitempty" example:"user,management"`              // 标签
	Admins       string                `json:"admins,omitempty" example:"user1,user2"`                // 节点管理员列表，逗号分隔的用户名
	PendingCount int                   `json:"pending_count,omitempty" example:"5"`                   // ⭐ 待审批的权限申请数量
	Owner        string                `json:"owner,omitempty" example:"user1"`                       // 节点创建者（owner）
	AppID        int64                 `json:"app_id,omitempty" example:"1"`                          // 应用ID
	RefID        int64                 `json:"ref_id,omitempty" example:"0"`                          // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	FullCodePath string                `json:"full_code_path,omitempty" example:"/beiluo/myapp/user"` // 完整代码路径
	TemplateType string                `json:"template_type,omitempty" example:"form"`                // 模板类型（函数的类型，如 form、table）
	Version      string                `json:"version,omitempty" example:"v1"`                        // 节点当前版本号（如 v1, v2），package类型表示目录版本，function类型表示函数版本等
	VersionNum   int                   `json:"version_num,omitempty" example:"1"`                     // 节点当前版本号（数字部分）
	HasFunction  bool                  `json:"has_function,omitempty" example:"true"`                 // ⭐ 是否有函数（仅对package类型有效）：如果该package下直接或间接包含function类型的子节点，则为true
	RunCount     int                   `json:"run_count,omitempty"`                                   // ⭐ 运行次数（仅 function 类型有意义），用于排序与展示「已使用 N 次」
	IsAdmin      bool                  `json:"is_admin,omitempty" example:"true"`                     // ⭐ 是否是管理员（企业版功能）：如果用户是工作空间管理员，则为 true，前端优先判断此字段，无需构造每个节点的权限
	Permissions  map[string]bool       `json:"permissions"`                                           // ⭐ 权限信息（企业版功能）：权限点 -> 是否有权限（即使为空也返回 {}，避免前端 undefined）
	Children     []*GetServiceTreeResp `json:"children,omitempty"`                                    // 子目录列表
}

// GetServiceTreeDetailReq 获取服务目录详情请求
type GetServiceTreeDetailReq struct {
	ID           int64  `json:"id" example:"1"`                              // 服务目录ID（优先使用）
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径（如果未提供ID则使用）
}

// GetServiceTreeDetailResp 获取服务目录详情响应
type GetServiceTreeDetailResp struct {
	ID           int64           `json:"id" example:"1"`                              // 服务目录ID
	Name         string          `json:"name" example:"用户管理"`                         // 服务目录名称
	Code         string          `json:"code" example:"user"`                         // 服务目录代码
	Type         string          `json:"type" example:"package"`                      // 节点类型
	Description  string          `json:"description" example:"用户相关的API接口"`            // 描述
	Tags         string          `json:"tags" example:"user,management"`              // 标签
	AppID        int64           `json:"app_id" example:"1"`                          // 应用ID
	RefID        int64           `json:"ref_id" example:"0"`                          // 引用ID
	FullCodePath string          `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	TemplateType string          `json:"template_type,omitempty" example:"form"`      // 模板类型（函数的类型，如 form、table）
	Version      string          `json:"version" example:"v1"`                        // 节点当前版本号
	VersionNum   int             `json:"version_num" example:"1"`                     // 节点当前版本号（数字部分）
	RunCount     int             `json:"run_count,omitempty"`                         // ⭐ 运行次数（仅 function 类型有意义），用于展示「已使用 N 次」
	Permissions  map[string]bool `json:"permissions"`                                 // ⭐ 权限标识（企业版功能）：权限点 -> 是否有权限（即使为空也返回 {}）
}

// UpdateServiceTreeMetadataReq 更新服务目录元数据请求
// 使用指针类型支持增量更新和字段清空
type UpdateServiceTreeMetadataReq struct {
	ID          int64   `json:"id" binding:"required" example:"1"`          // 服务目录ID
	Name        *string `json:"name,omitempty" example:"用户管理"`              // 服务目录名称（指针类型，nil=不更新，""=清空）
	Code        *string `json:"code,omitempty" example:"user"`              // 服务目录代码（暂不支持修改；nil=不更新）
	Description *string `json:"description,omitempty" example:"用户相关的API接口"` // 描述（指针类型，nil=不更新，""=清空）
	Tags        *string `json:"tags,omitempty" example:"user,management"`   // 标签（指针类型，nil=不更新，""=清空）
	Admins      *string `json:"admins,omitempty" example:"user1,user2"`     // 管理员列表（指针类型，nil=不更新，""=清空）
}

// ==================== 按类型分离的更新和删除接口 DTO ====================

// UpdatePackageReq 更新 package 类型节点请求（ID 由 path /packages/:id 提供，body 可不传）
type UpdatePackageReq struct {
	ID          int64   `json:"id,omitempty" example:"1"`                   // 目录ID（由 path 提供，body 可不传）
	Name        *string `json:"name,omitempty" example:"用户管理"`              // 目录名称（指针类型，nil=不更新，""=清空）
	Code        *string `json:"code,omitempty" example:"user"`              // 目录代码（暂不支持修改；nil=不更新）
	Description *string `json:"description,omitempty" example:"用户相关的API接口"` // 描述（指针类型，nil=不更新，""=清空）
	Tags        *string `json:"tags,omitempty" example:"user,management"`   // 标签（指针类型，nil=不更新，""=清空）
	Admins      *string `json:"admins,omitempty" example:"user1,user2"`     // 管理员列表（指针类型，nil=不更新，""=清空）
}

// UpdateFunctionReq 更新 function 类型节点请求（ID 由 path /functions/:id 提供，body 可不传）
type UpdateFunctionReq struct {
	ID          int64   `json:"id,omitempty" example:"1"`               // 函数ID（由 path 提供，body 可不传）
	Name        *string `json:"name,omitempty" example:"用户列表"`          // 函数名称（指针类型，nil=不更新，""=清空）
	Code        *string `json:"code,omitempty" example:"user_list"`     // 函数代码（暂不支持修改；nil=不更新）
	Description *string `json:"description,omitempty" example:"获取用户列表"` // 描述（指针类型，nil=不更新，""=清空）
	Tags        *string `json:"tags,omitempty" example:"user,list"`     // 标签（指针类型，nil=不更新，""=清空）
}

// UpdateDocsReq 更新 docs 类型节点请求（ID 由 path /docs/crud/:id 提供，body 可不传）
type UpdateDocsReq struct {
	ID          int64   `json:"id,omitempty" example:"1"`                // 文档ID（由 path 提供，body 可不传）
	Name        *string `json:"name,omitempty" example:"API文档"`          // 文档名称（指针类型，nil=不更新，""=清空）
	Code        *string `json:"code,omitempty" example:"api_docs"`       // 文档代码（暂不支持修改；nil=不更新）
	Description *string `json:"description,omitempty" example:"API接口文档"` // 描述（指针类型，nil=不更新，""=清空）
	Tags        *string `json:"tags,omitempty" example:"api,docs"`       // 标签（指针类型，nil=不更新，""=清空）
	Admins      *string `json:"admins,omitempty" example:"user1,user2"`  // 管理员列表（指针类型，nil=不更新，""=清空）
	Content     *string `json:"content,omitempty" example:"# 文档内容..."`   // 文档内容（指针类型，nil=不更新，""=清空）
	Format      *string `json:"format,omitempty" example:"markdown"`     // 文档格式（指针类型，nil=不更新，""=清空）
	Summary     *string `json:"summary,omitempty" example:"文档摘要"`        // 文档摘要（指针类型，nil=不更新，""=清空）
}

// BatchCreateDirectoryTreeReq 批量创建目录树请求
type BatchCreateDirectoryTreeReq struct {
	User  string                   `json:"user" binding:"required"`  // 用户名
	App   string                   `json:"app" binding:"required"`   // 应用名
	Items []*DirectoryScaffoldItem `json:"items" binding:"required"` // 目录脚手架项列表
}

// BatchCreateDirectoryTreeResp 批量创建目录树响应
type BatchCreateDirectoryTreeResp struct {
	DirectoryCount int      `json:"directory_count"` // 创建的目录数量
	FileCount      int      `json:"file_count"`      // 创建的文件数量
	CreatedPaths   []string `json:"created_paths"`   // 创建的路径列表
}

// BatchWriteFilesReq 批量写文件请求
type BatchWriteFilesReq struct {
	User           string           `json:"user" binding:"required"`   // 用户名
	App            string           `json:"app" binding:"required"`    // 应用名
	Files          []*FileWriteItem `json:"files" binding:"required"`  // 文件写入项列表
	ForceDiff      bool             `json:"force_diff,omitempty"`      // 是否清理 api-logs，让本次更新重新产生 add diff
	OperationName  string           `json:"operation_name,omitempty"`  // 内部操作名，用于日志和发布元数据
	OperationLabel string           `json:"operation_label,omitempty"` // 操作中文名，用于错误信息
}

// BatchWriteFilesResp 批量写文件响应
type BatchWriteFilesResp struct {
	FileCount     int       `json:"file_count"`                // 写入的文件数量
	WrittenPaths  []string  `json:"written_paths"`             // 写入的文件路径列表
	Diff          *DiffData `json:"diff,omitempty"`            // API diff 信息（编译后）
	OldVersion    string    `json:"old_version"`               // 旧版本号
	NewVersion    string    `json:"new_version"`               // 新版本号
	GitCommitHash string    `json:"git_commit_hash,omitempty"` // Git 提交哈希
	Warnings      []string  `json:"warnings,omitempty"`        // 非阻断告警（如版本已更新但元数据同步失败）
}

// SearchFunctionsReq 搜索函数请求
type SearchFunctionsReq struct {
	User         string `json:"user" form:"user"`                        // 用户名（可选，用于过滤应用）
	App          string `json:"app" form:"app"`                          // 应用名（可选，用于过滤应用）
	Keyword      string `json:"keyword" form:"keyword"`                  // 搜索关键词（可选，用于搜索名称和路径）
	TemplateType string `json:"template_type" form:"template_type"`      // 模板类型过滤（可选，如：form、table、chart）
	Page         int    `json:"page" form:"page"  example:"1"`           // 页码
	PageSize     int    `json:"page_size" form:"page_size" example:"10"` // 每页数量
	CurrentUser  string `json:"-" form:"-"`                              // 当前登录用户（后端注入，用于默认可见范围）
}

// SearchFunctionsResp 搜索函数响应
type SearchFunctionsResp struct {
	Functions []*FunctionSearchResult `json:"functions"` // 函数列表
	Total     int64                   `json:"total"`     // 总数
	Page      int                     `json:"page"`      // 当前页码
	PageSize  int                     `json:"page_size"` // 每页数量
}

// SearchResourcesReq 全站资源搜索请求
type SearchResourcesReq struct {
	User         string `json:"user" form:"user"`                        // 用户名（可选，用于过滤应用）
	App          string `json:"app" form:"app"`                          // 应用名（可选，用于过滤应用）
	Keyword      string `json:"keyword" form:"keyword"`                  // 搜索关键词
	ResourceType string `json:"resource_type" form:"resource_type"`      // 资源类型：all/package/function/docs/board
	Page         int    `json:"page" form:"page" example:"1"`            // 页码
	PageSize     int    `json:"page_size" form:"page_size" example:"20"` // 每页数量
	CurrentUser  string `json:"-" form:"-"`                              // 当前登录用户（后端注入，用于默认可见范围）
}

// SearchResourcesResp 全站资源搜索响应
type SearchResourcesResp struct {
	Items    []*ResourceSearchResult `json:"items"`     // 搜索结果
	Total    int64                   `json:"total"`     // 总数
	Page     int                     `json:"page"`      // 当前页码
	PageSize int                     `json:"page_size"` // 每页数量
}

// ResourceSearchResult 全站资源搜索结果
type ResourceSearchResult struct {
	ID           int64  `json:"id" example:"1"`                         // ServiceTree 节点 ID
	Name         string `json:"name" example:"表格解析"`                    // 资源名称
	Code         string `json:"code" example:"table_parse"`             // 资源代码
	Type         string `json:"type" example:"function"`                // 资源类型
	FullCodePath string `json:"full_code_path" example:"/system/app/a"` // 完整路径
	Description  string `json:"description,omitempty"`                  // 描述
	Tags         string `json:"tags,omitempty"`                         // 标签
	TemplateType string `json:"template_type,omitempty"`                // 函数模板类型
	AppID        int64  `json:"app_id,omitempty"`                       // 应用 ID
	AppUser      string `json:"app_user,omitempty"`                     // 应用所属用户
	AppCode      string `json:"app_code,omitempty"`                     // 应用代码
	RunCount     int    `json:"run_count,omitempty"`                    // 运行次数
	MatchSource  string `json:"match_source,omitempty"`                 // 命中来源：node/doc
	Snippet      string `json:"snippet,omitempty"`                      // 命中摘要
}

// FunctionSearchResult 函数搜索结果（含 schema 摘要，便于调用方构造 body）
type FunctionSearchResult struct {
	ID           int64                          `json:"id" example:"1"`                                               // 函数ID
	Name         string                         `json:"name" example:"表格解析"`                                          // 函数名称
	Code         string                         `json:"code" example:"table_parse"`                                   // 函数代码
	FullCodePath string                         `json:"full_code_path" example:"/system/tools/table/inspect.form"`    // 完整代码路径
	Description  string                         `json:"description" example:"解析Excel/CSV文件为Markdown表格"`               // 函数描述
	TemplateType string                         `json:"template_type" example:"form"`                                 // 模板类型（form、table、chart）
	Callbacks    []string                       `json:"callbacks,omitempty" example:"OnTableAddRow,OnTableUpdateRow"` // 函数回调能力摘要
	AppID        int64                          `json:"app_id" example:"1"`                                           // 应用ID
	AppUser      string                         `json:"app_user" example:"system"`                                    // 应用所属用户
	AppCode      string                         `json:"app_code" example:"tools"`                                     // 应用代码
	RunCount     int                            `json:"run_count,omitempty"`                                          // 运行次数（用于 search_tools 按热度排序）
	Schema       *functionschema.FunctionSchema `json:"schema,omitempty"`                                             // 函数 schema 摘要
}

// GetServiceTreeByIDReq 根据ID获取服务目录请求
type GetServiceTreeByIDReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"` // 服务目录ID
}

// ==================== 按类型分离的创建接口 DTO ====================

// CreatePackageReq 创建 package 类型节点请求
type CreatePackageReq struct {
	User               string `json:"user" binding:"required" example:"beiluo"`      // 用户名
	App                string `json:"app" binding:"required" example:"myapp"`        // 应用名
	Name               string `json:"name" binding:"required" example:"用户管理"`        // 目录名称
	Code               string `json:"code" binding:"required" example:"user"`        // 目录代码
	ParentFullCodePath string `json:"parent_full_code_path" example:"/beiluo/myapp"` // 父目录完整路径，空字符串表示根目录
	Description        string `json:"description" example:"用户相关的API接口"`              // 描述
	Tags               string `json:"tags" example:"user,management"`                // 标签
	Admins             string `json:"admins" example:"user1,user2"`                  // 管理员列表，逗号分隔的用户名
}

// CreatePackageResp 创建 package 类型节点响应
type CreatePackageResp struct {
	ID           int64  `json:"id" example:"1"`                              // 目录ID
	Name         string `json:"name" example:"用户管理"`                         // 目录名称
	Code         string `json:"code" example:"user"`                         // 目录代码
	Type         string `json:"type" example:"package"`                      // 节点类型（固定为 package）
	Description  string `json:"description" example:"用户相关的API接口"`            // 描述
	Tags         string `json:"tags" example:"user,management"`              // 标签
	AppID        int64  `json:"app_id" example:"1"`                          // 应用ID
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径（父路径可由此推导）
	Version      string `json:"version" example:"v1"`                        // 目录版本号
	VersionNum   int    `json:"version_num" example:"1"`                     // 目录版本号（数字部分）
	Admins       string `json:"admins" example:"user1,user2"`                // 管理员列表
}

// CreateFunctionReq 创建 function 类型节点请求
type CreateFunctionReq struct {
	User          string `json:"user" binding:"required" example:"beiluo"`                       // 用户名
	App           string `json:"app" binding:"required" example:"myapp"`                         // 应用名
	Name          string `json:"name" binding:"required" example:"用户列表"`                         // 函数名称
	Code          string `json:"code" binding:"required" example:"user_list"`                    // 函数代码
	DirectoryPath string `json:"directory_path" binding:"required" example:"/beiluo/myapp/user"` // 目录完整路径
	TemplateType  string `json:"template_type" example:"table"`                                  // 模板类型（form、table、chart）
	SourceCode    string `json:"source_code" binding:"required"`                                 // 源代码内容
	Description   string `json:"description" example:"获取用户列表"`                                   // 描述
	Tags          string `json:"tags" example:"user,list"`                                       // 标签
}

// CreateFunctionResp 创建 function 类型节点响应
type CreateFunctionResp struct {
	ID           int64  `json:"id" example:"1"`                                        // 函数ID
	Name         string `json:"name" example:"用户列表"`                                   // 函数名称
	Code         string `json:"code" example:"user_list"`                              // 函数代码
	Type         string `json:"type" example:"function"`                               // 节点类型（固定为 function）
	TemplateType string `json:"template_type" example:"table"`                         // 模板类型
	Description  string `json:"description" example:"获取用户列表"`                          // 描述
	Tags         string `json:"tags" example:"user,list"`                              // 标签
	AppID        int64  `json:"app_id" example:"1"`                                    // 应用ID
	RefID        int64  `json:"ref_id" example:"1"`                                    // 引用ID（指向 Function 表）
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user/user_list"` // 完整代码路径（父路径可由此推导）
	Version      string `json:"version" example:"v1"`                                  // 函数版本号
	VersionNum   int    `json:"version_num" example:"1"`                               // 函数版本号（数字部分）
}

// CreateDocsReq 创建 docs 类型节点请求
type CreateDocsReq struct {
	User               string `json:"user" binding:"required" example:"beiluo"`      // 用户名
	App                string `json:"app" binding:"required" example:"myapp"`        // 应用名
	Name               string `json:"name" binding:"required" example:"API文档"`       // 文档名称
	Code               string `json:"code" binding:"required" example:"api_docs"`    // 文档代码
	ParentFullCodePath string `json:"parent_full_code_path" example:"/beiluo/myapp"` // 父目录完整路径，空字符串表示根目录
	Description        string `json:"description" example:"API接口文档"`                 // 描述
	Tags               string `json:"tags" example:"api,docs"`                       // 标签
	Admins             string `json:"admins" example:"user1,user2"`                  // 管理员列表，逗号分隔的用户名
	Content            string `json:"content" example:"# 文档内容\n\n这是文档内容..."`         // 文档内容
	Format             string `json:"format" example:"markdown"`                     // 文档格式（默认为 markdown）
	Summary            string `json:"summary" example:"文档摘要"`                        // 文档摘要（可选）
}

// CreateDocsResp 创建 docs 类型节点响应
type CreateDocsResp struct {
	ID           int64  `json:"id" example:"1"`                                  // 文档ID
	Name         string `json:"name" example:"API文档"`                            // 文档名称
	Code         string `json:"code" example:"api_docs"`                         // 文档代码
	Type         string `json:"type" example:"docs"`                             // 节点类型（固定为 docs）
	Description  string `json:"description" example:"API接口文档"`                   // 描述
	Tags         string `json:"tags" example:"api,docs"`                         // 标签
	AppID        int64  `json:"app_id" example:"1"`                              // 应用ID
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/api_docs"` // 完整代码路径（父路径可由此推导）
	Admins       string `json:"admins" example:"user1,user2"`                    // 管理员列表
	DocID        int64  `json:"doc_id" example:"1"`                              // 关联的文档记录ID
}
