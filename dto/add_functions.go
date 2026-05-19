package dto

// AddFunctionsReq 添加函数请求（agent-server -> workspace）
// 用于向服务目录添加函数，将生成的代码写入到工作空间对应的目录下。
// 租户（user/app）由服务端从 full_code_path 解析出的 ServiceTree.App 确定，不传 User，避免访问他人应用时按当前用户查 app 导致 record not found。
type AddFunctionsReq struct {
	// 目录标识：使用 full_code_path（有语意、像函数名）；服务端据此查 ServiceTree，并从 targetTree.App 取租户。
	FullCodePath string `json:"full_code_path" example:"/luobei/demo/crm" binding:"required"` // 服务目录完整路径（必填）
	// 目标文件名与源码内容。
	FileName   string `json:"file_name" example:"crm_ticket"`   // 从代码中提取的文件名（可带 .go 后缀）
	SourceCode string `json:"source_code" example:"package..."` // 处理后的源代码（从 Markdown 中提取）
	// SkipBuild 为 true 时仅写文件不编译不部署（对应 write_go_file 的 build_workspace=false）。
	SkipBuild bool `json:"skip_build,omitempty"`
}

// AddFunctionsResp 添加函数响应（同步写入返回）
type AddFunctionsResp struct {
	Success bool   `json:"success" example:"true"`     // 是否成功
	AppID   int64  `json:"app_id" example:"1"`         // 应用ID
	AppCode string `json:"app_code" example:"myapp"`   // 应用代码
	Error   string `json:"error,omitempty" example:""` // 错误信息（如果失败）
	// 当 SkipBuild=false 且编译成功时，由 app-server 填充以下编译/变更信息，供 write_go_file 返回友好提示。
	BuildOldVersion string   `json:"build_old_version,omitempty"` // 编译前版本
	BuildNewVersion string   `json:"build_new_version,omitempty"` // 编译后版本
	BuildDiffAdd    []string `json:"build_diff_add,omitempty"`    // 新增的接口/路由（如 task）
	BuildDiffUpdate []string `json:"build_diff_update,omitempty"` // 变更的接口/路由
	BuildDiffDelete []string `json:"build_diff_delete,omitempty"` // 删除的接口/路由
}
