package dto

// ForkFunctionGroupReq Fork 函数组请求（API 层，支持批量，使用 map 形式）
// key: 函数组的 full_group_code（源函数组的完整代码路径，格式：{full_group_code}）
// value: 服务目录的 full_code_path（目标服务目录的完整代码路径，格式：/{user}/{app}/{package_path}）
// 例如：a fork到a1目录，b fork到b1目录
type ForkFunctionGroupReq struct {
	SourceToTargetMap map[string]string `json:"source_to_target_map" binding:"required" example:"/luobei/app_a/plugins/tools_cashier:/luobei/app_b/a1,/luobei/app_a/plugins/tools_excel:/luobei/app_b/b1"` // 源到目标的映射：key=函数组的full_group_code，value=服务目录的full_code_path
	TargetAppID       int64             `json:"target_app_id" binding:"required" example:"123"`                                                                                                            // 目标应用 ID
}

// ForkFunctionGroupResp Fork 函数组响应（API 层，简化版）
type ForkFunctionGroupResp struct {
	Message string `json:"message" example:"函数组 Fork 成功"` // 响应消息
}

// ForkFunctionGroupRuntimeReq Fork 函数组运行时请求（app-runtime，支持批量）
// 一次调用可以处理多个 package，每个 package 有自己的文件列表
type ForkFunctionGroupRuntimeReq struct {
	TargetUser string             `json:"target_user"` // 目标应用的用户名
	TargetApp  string             `json:"target_app"`  // 目标应用的代码
	Packages   []*ForkPackageInfo `json:"packages"`    // 多个 package，每个包含自己的文件列表
}

// ForkPackageInfo Fork 的 package 信息
type ForkPackageInfo struct {
	Package string                   `json:"package"` // 目标 package 代码（支持多级，如 plugins/cashier）
	Files   []*ForkFunctionGroupFile `json:"files"`   // 该 package 下的文件列表
}

// ForkFunctionGroupFile Fork 的文件信息
type ForkFunctionGroupFile struct {
	GroupCode     string `json:"group_code"`     // 函数组代码（文件名，不含 .go）
	SourceCode    string `json:"source_code"`    // 源代码内容
	SourcePackage string `json:"source_package"` // 源 package 名称（用于替换）
}

// ForkFunctionGroupRuntimeResp Fork 函数组运行时响应（app-runtime，简化版）
type ForkFunctionGroupRuntimeResp struct {
	Success      bool     `json:"success" example:"true"`   // 是否成功
	Message      string   `json:"message" example:"文件写入成功"` // 响应消息
	WrittenFiles []string `json:"written_files"`            // 已写入的文件路径列表（用于失败时回滚）
}
