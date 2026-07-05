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
	FileType     string `json:"file_type" example:"go"`                // 文件类型（go, json, yaml 等）
	Content      string `json:"content" example:"package hr\n..."`     // 文件内容
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

// WriteFileRuntimeReq 写入单个文本文件请求（app-server -> app-runtime），不触发编译。
type WriteFileRuntimeReq struct {
	User          string `json:"user" binding:"required"`           // 租户用户名
	App           string `json:"app" binding:"required"`            // 应用名
	DirectoryPath string `json:"directory_path" binding:"required"` // 目录完整路径（如 /user/app/pkg1）
	FileName      string `json:"file_name" binding:"required"`      // 文件名（如 config.json 或 handler.go）
	FileType      string `json:"file_type,omitempty"`               // 可选文件类型；不传则从 file_name 推断，无扩展名默认 go
	Content       string `json:"content"`                           // 文本内容
}

// WriteFileRuntimeResp 写入单个文本文件响应。
type WriteFileRuntimeResp struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	RelativePath string `json:"relative_path,omitempty"`
	FileType     string `json:"file_type,omitempty"`
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

// ReadAppLogRuntimeReq 读取应用日志请求（app-server -> app-runtime）
type ReadAppLogRuntimeReq struct {
	User         string `json:"user" binding:"required"` // 租户用户名
	App          string `json:"app" binding:"required"`  // 应用名
	Version      string `json:"version"`                 // 版本号（如 v48），为空时由 app-server 先解析当前版本
	Lines        int    `json:"lines"`                   // 返回行数（无 keyword 时用于 tail；有 keyword 时用于输出上限）
	Keyword      string `json:"keyword"`                 // 关键词（可选）
	ContextLines int    `json:"context_lines"`           // 命中上下文行数（可选）
	MaxMatches   int    `json:"max_matches"`             // 最大命中数（可选）
	IgnoreCase   bool   `json:"ignore_case"`             // 关键词是否忽略大小写（可选）
}

// ReadAppLogRuntimeResp 读取应用日志响应（app-runtime -> app-server）
type ReadAppLogRuntimeResp struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	ResolvedVersion string `json:"resolved_version"` // 实际读取的版本
	LogFile         string `json:"log_file"`         // 日志文件名
	TotalLines      int    `json:"total_lines"`      // 日志总行数
	ReturnedLines   int    `json:"returned_lines"`   // 返回行数
	MatchCount      int    `json:"match_count"`      // 命中数（keyword 模式）
	Truncated       bool   `json:"truncated"`        // 是否因限制被截断
	Content         string `json:"content"`          // 日志内容
}
