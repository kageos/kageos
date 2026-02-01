package dto

// ReadDirectoryFilesRuntimeReq 读取目录文件请求（app-server -> app-runtime）
type ReadDirectoryFilesRuntimeReq struct {
	User          string `json:"user" binding:"required" example:"beiluo"`                     // 租户用户名
	App           string `json:"app" binding:"required" example:"myapp"`                       // 应用名
	DirectoryPath string `json:"directory_path" binding:"required" example:"/beiluo/myapp/hr"` // 目录完整路径（包含应用前缀）
}

// DirectoryFileInfo 目录文件信息
type DirectoryFileInfo struct {
	FileName     string `json:"file_name" example:"attendance"`        // 文件名（不含 .go 后缀）
	RelativePath string `json:"relative_path" example:"attendance.go"` // 相对路径（相对于目录）
	Content      string `json:"content" example:"package hr\n..."`     // 文件内容
	// 向后兼容：保留 group_code（如果存在，优先使用 file_name）
	GroupCode string `json:"group_code,omitempty" example:"attendance"` // 函数组代码（已废弃，使用 file_name）
}

// ReadDirectoryFilesRuntimeResp 读取目录文件响应（app-runtime -> app-server）
type ReadDirectoryFilesRuntimeResp struct {
	Success bool                `json:"success" example:"true"` // 是否成功
	Message string              `json:"message" example:"读取成功"` // 响应消息
	Files   []DirectoryFileInfo `json:"files"`                  // 文件列表
}

// ReplaceInFileRuntimeReq 文件内容 search-replace 请求（app-server -> app-runtime）
type ReplaceInFileRuntimeReq struct {
	User              string `json:"user" binding:"required"`           // 租户用户名
	App               string `json:"app" binding:"required"`            // 应用名
	DirectoryPath     string `json:"directory_path" binding:"required"` // 目录完整路径（如 /user/app/pkg1）
	FileName          string `json:"file_name" binding:"required"`      // 文件名（如 handler 或 handler.go）
	SearchString      string `json:"search_string" binding:"required"`  // 要被替换的原文
	ReplaceString     string `json:"replace_string"`                    // 替换后的内容
	ReplaceAll        bool   `json:"replace_all"`                       // 是否替换全部出现，默认 true
	ReturnFullContent bool   `json:"return_full_content"`               // 是否在响应中返回替换后的完整文件内容
}

// ReplaceInFileRuntimeResp 文件内容 search-replace 响应
type ReplaceInFileRuntimeResp struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ReplaceCount int    `json:"replace_count"` // 替换次数
	FullContent  string `json:"full_content"`  // 替换后的完整文件内容（仅当请求 return_full_content=true 时填充）
}

// DeleteFileRuntimeReq 删除磁盘文件请求（app-server -> app-runtime）
type DeleteFileRuntimeReq struct {
	User          string `json:"user" binding:"required"`           // 租户用户名
	App           string `json:"app" binding:"required"`            // 应用名
	DirectoryPath string `json:"directory_path" binding:"required"` // 目录完整路径（如 /user/app/pkg1）
	FileName      string `json:"file_name" binding:"required"`      // 文件名（如 handler 或 handler.go）
}

// DeleteFileRuntimeResp 删除磁盘文件响应
type DeleteFileRuntimeResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
