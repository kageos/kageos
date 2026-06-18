package dto

// ForkFunctionGroupReq Fork 函数组请求（API 层，支持批量，使用 map 形式）
// key: 函数组的 full_group_code（源函数组的完整代码路径，格式：{full_group_code}）
// value: 服务目录的 full_code_path（目标服务目录的完整代码路径，格式：/{user}/{app}/{package_path}）
// 例如：a fork到a1目录，b fork到b1目录
type ForkFunctionGroupReq struct {
	SourceToTargetMap map[string]string `json:"source_to_target_map" binding:"required" example:"/luobei/app_a/tools/cashier:/luobei/app_b/a1,/luobei/app_a/tools/excel:/luobei/app_b/b1"` // 源到目标的映射：key=函数组的full_group_code，value=服务目录的full_code_path
	TargetAppID       int64             `json:"target_app_id" binding:"required" example:"123"`                                                                                            // 目标应用 ID
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
	Package string                   `json:"package"` // 目标 package 代码（支持多级，如 tools/cashier）
	Files   []*ForkFunctionGroupFile `json:"files"`   // 该 package 下的文件列表
}

// ForkFunctionGroupFile Fork 的文件信息
type ForkFunctionGroupFile struct {
	FileName      string `json:"file_name"`      // 文件名（不含 .go 后缀）
	SourceCode    string `json:"source_code"`    // 源代码内容
	SourcePackage string `json:"source_package"` // 源 package 名称（用于替换）
}

// ForkFunctionGroupRuntimeResp Fork 函数组运行时响应（app-runtime，简化版）
type ForkFunctionGroupRuntimeResp struct {
	Success      bool     `json:"success" example:"true"`   // 是否成功
	Message      string   `json:"message" example:"文件写入成功"` // 响应消息
	WrittenFiles []string `json:"written_files"`            // 已写入的文件路径列表（用于失败时回滚）
}

// CopyDirectoryReq 复制目录请求（支持递归复制本地目录及其所有子目录）
type CopyDirectoryReq struct {
	SourceDirectoryPath string `json:"source_directory_path" binding:"required" example:"/luobei/app_a/hr"` // 源目录完整路径
	TargetDirectoryPath string `json:"target_directory_path" binding:"required" example:"/luobei/app_b"`    // 目标父目录完整路径
	TargetAppID         int64  `json:"target_app_id" binding:"required" example:"123"`                      // 目标应用ID
	TargetDirectoryName string `json:"target_directory_name,omitempty" example:"人事管理"`                      // 复制后根目录中文展示名
	ReplaceExisting     bool   `json:"replace_existing,omitempty"`                                          // 同名目录已存在时是否完全替换
}

// CopyDirectoryResp 复制目录响应
type CopyDirectoryResp struct {
	Message             string `json:"message" example:"复制目录成功，共复制 3 个目录，15 个文件"`  // 响应消息
	DirectoryCount      int    `json:"directory_count" example:"3"`                // 复制的目录数
	FileCount           int    `json:"file_count" example:"15"`                    // 复制的文件数
	Replaced            bool   `json:"replaced,omitempty"`                         // 是否执行了同名目录完全替换
	TargetDirectoryPath string `json:"target_directory_path,omitempty"`            // 最终目标目录完整路径
	OldVersion          string `json:"old_version,omitempty" example:"v3"`         // 旧版本号
	NewVersion          string `json:"new_version,omitempty" example:"v4"`         // 新版本号
	GitCommitHash       string `json:"git_commit_hash,omitempty" example:"abc123"` // Git 提交哈希
}

// CreateDirectoryReq 创建目录请求
type CreateDirectoryReq struct {
	DirectoryPath string `json:"directory_path" binding:"required" example:"/luobei/app_a/hr/new_dir"` // 目录完整路径
	AppID         int64  `json:"app_id" binding:"required" example:"123"`                              // 应用ID
}

// CreateDirectoryResp 创建目录响应
type CreateDirectoryResp struct {
	Message string `json:"message" example:"创建目录成功"` // 响应消息
}

// RemoveDirectoryReq 删除目录请求
type RemoveDirectoryReq struct {
	DirectoryPath string `json:"directory_path" binding:"required" example:"/luobei/app_a/hr/old_dir"` // 目录完整路径
	AppID         int64  `json:"app_id" binding:"required" example:"123"`                              // 应用ID
}

// RemoveDirectoryResp 删除目录响应
type RemoveDirectoryResp struct {
	Message string `json:"message" example:"删除目录成功"` // 响应消息
}
