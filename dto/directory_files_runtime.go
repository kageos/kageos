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

// ReplaceItemRuntime 单次替换项（app-server -> app-runtime）
type ReplaceItemRuntime struct {
	SearchString  string `json:"search_string" binding:"required"`
	ReplaceString string `json:"replace_string"`
	ExpectedCount int    `json:"expected_count"` // 0 表示默认 1
}

// ReplaceInFileBatchReq 文件内容批量 search-replace 请求（app-server -> app-runtime）；内存中按顺序执行，全部校验通过才落盘
type ReplaceInFileBatchReq struct {
	User              string               `json:"user" binding:"required"`
	App               string               `json:"app" binding:"required"`
	DirectoryPath     string               `json:"directory_path" binding:"required"`
	FileName          string               `json:"file_name" binding:"required"`
	Replacements      []ReplaceItemRuntime `json:"replacements" binding:"required"`
	AllOrNothing      bool                 `json:"all_or_nothing"` // 默认 true，仅当所有项 actual==expected 才写盘
	ReturnFullContent bool                 `json:"return_full_content"`
}

// ReplaceItemResultRuntime 单次替换结果（用于未落盘时返回）
type ReplaceItemResultRuntime struct {
	Index         int `json:"index"`
	ExpectedCount int `json:"expected_count"`
	ActualCount   int `json:"actual_count"`
}

// ReplaceInFileBatchResp 批量 search-replace 响应
type ReplaceInFileBatchResp struct {
	Success      bool                       `json:"success"`
	Message      string                     `json:"message"`
	ReplaceCount int                        `json:"replace_count"`
	FullContent  string                     `json:"full_content,omitempty"`
	Details      []ReplaceItemResultRuntime `json:"details,omitempty"` // 未落盘时哪几项不符
}

// ReplaceInFileRuntimeReq 已废弃，请使用 ReplaceInFileBatchReq
type ReplaceInFileRuntimeReq struct {
	User              string `json:"user" binding:"required"`
	App               string `json:"app" binding:"required"`
	DirectoryPath     string `json:"directory_path" binding:"required"`
	FileName          string `json:"file_name" binding:"required"`
	SearchString      string `json:"search_string" binding:"required"`
	ReplaceString     string `json:"replace_string"`
	ReplaceAll        bool   `json:"replace_all"`
	ReturnFullContent bool   `json:"return_full_content"`
}

// ReplaceInFileRuntimeResp 已废弃，请使用 ReplaceInFileBatchResp
type ReplaceInFileRuntimeResp struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ReplaceCount int    `json:"replace_count"`
	FullContent  string `json:"full_content"`
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
