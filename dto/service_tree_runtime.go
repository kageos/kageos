package dto

// DeleteServiceTreeRuntimeReq 删除服务目录运行时请求（app-server -> app-runtime：删磁盘目录并从 main.go 移除 import）
type DeleteServiceTreeRuntimeReq struct {
	User        string `json:"user"`         // 用户名
	App         string `json:"app"`          // 应用名
	PackagePath string `json:"package_path"` // 相对 api 的包路径，如 crm 或 crm/ticket
}

// DeleteServiceTreeRuntimeResp 删除服务目录运行时响应
type DeleteServiceTreeRuntimeResp struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// DirectoryScaffoldItem 目录脚手架项
type DirectoryScaffoldItem struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 完整代码路径，如 /user/app/plugins/cashier
	Name         string `json:"name,omitempty"`                    // 目录名称
	Description  string `json:"description,omitempty"`             // 目录描述
	Tags         string `json:"tags,omitempty"`                    // 目录标签
}

// FileWriteItem 文件写入项
type FileWriteItem struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 目标目录完整路径，如 /user/app/plugins/cashier
	FileName     string `json:"file_name,omitempty"`               // 文件名（不含扩展名）
	FileType     string `json:"file_type,omitempty"`               // 文件类型（go, json, yaml 等）
	Content      string `json:"content,omitempty"`                 // 文件内容
	RelativePath string `json:"relative_path,omitempty"`           // 文件相对路径（如 user.go 或 subdir/user.go）
}

// BatchCreateDirectoryTreeRuntimeReq 批量创建目录树运行时请求
type BatchCreateDirectoryTreeRuntimeReq struct {
	User  string                   `json:"user"`  // 用户名
	App   string                   `json:"app"`   // 应用名
	Items []*DirectoryScaffoldItem `json:"items"` // 目录脚手架项列表
}

// BatchCreateDirectoryTreeRuntimeResp 批量创建目录树运行时响应
type BatchCreateDirectoryTreeRuntimeResp struct {
	DirectoryCount int      `json:"directory_count"` // 创建的目录数量
	FileCount      int      `json:"file_count"`      // 创建的文件数量
	CreatedPaths   []string `json:"created_paths"`   // 创建的路径列表
}

// BatchWriteFilesRuntimeReq 批量写文件运行时请求
type BatchWriteFilesRuntimeReq struct {
	User           string           `json:"user"`                      // 用户名
	App            string           `json:"app"`                       // 应用名
	Files          []*FileWriteItem `json:"files"`                     // 文件写入项列表
	ForceDiff      bool             `json:"force_diff,omitempty"`      // 是否清理 api-logs，让本次更新重新产生 add diff
	OperationName  string           `json:"operation_name,omitempty"`  // 内部操作名，用于日志和发布元数据
	OperationLabel string           `json:"operation_label,omitempty"` // 操作中文名，用于错误信息
}

// BatchWriteFilesRuntimeResp 批量写文件运行时响应
type BatchWriteFilesRuntimeResp struct {
	FileCount     int       `json:"file_count"`                // 写入的文件数量
	WrittenPaths  []string  `json:"written_paths"`             // 写入的文件路径列表
	Diff          *DiffData `json:"diff,omitempty"`            // API diff 信息（编译后）
	OldVersion    string    `json:"old_version"`               // 旧版本号
	NewVersion    string    `json:"new_version"`               // 新版本号
	GitCommitHash string    `json:"git_commit_hash,omitempty"` // Git 提交哈希
}

// ReplaceDirectoryTreeRuntimeReq 完全替换同名目录运行时请求。
type ReplaceDirectoryTreeRuntimeReq struct {
	User                   string                   `json:"user"`                       // 用户名
	App                    string                   `json:"app"`                        // 应用名
	TargetRootFullCodePath string                   `json:"target_root_full_code_path"` // 最终被替换目录完整路径
	Items                  []*DirectoryScaffoldItem `json:"items"`                      // 替换后的目录脚手架项列表
	Files                  []*FileWriteItem         `json:"files"`                      // 替换后的文件写入项列表
	ForceDiff              bool                     `json:"force_diff,omitempty"`       // 是否清理 api-logs，让本次更新重新产生 add diff
	OperationName          string                   `json:"operation_name,omitempty"`   // 内部操作名，用于日志和发布元数据
	OperationLabel         string                   `json:"operation_label,omitempty"`  // 操作中文名，用于错误信息
}

// ReplaceDirectoryTreeRuntimeResp 完全替换同名目录运行时响应。
type ReplaceDirectoryTreeRuntimeResp struct {
	DirectoryCount      int       `json:"directory_count"`                 // 创建/恢复的目录数量
	FileCount           int       `json:"file_count"`                      // 写入的文件数量
	TargetDirectoryPath string    `json:"target_directory_path,omitempty"` // 最终目标目录完整路径
	WrittenPaths        []string  `json:"written_paths"`                   // 写入的文件路径列表
	Diff                *DiffData `json:"diff,omitempty"`                  // API diff 信息（编译后）
	OldVersion          string    `json:"old_version"`                     // 旧版本号
	NewVersion          string    `json:"new_version"`                     // 新版本号
	GitCommitHash       string    `json:"git_commit_hash,omitempty"`       // Git 提交哈希
}
