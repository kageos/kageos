package dto

// ReadDirectoryFilesRuntimeReq 读取目录文件请求（app-server -> app-runtime）
type ReadDirectoryFilesRuntimeReq struct {
	User          string `json:"user" binding:"required" example:"beiluo"`                    // 租户用户名
	App           string `json:"app" binding:"required" example:"myapp"`                      // 应用名
	DirectoryPath string `json:"directory_path" binding:"required" example:"/beiluo/myapp/hr"` // 目录完整路径（包含应用前缀）
}

// DirectoryFileInfo 目录文件信息
type DirectoryFileInfo struct {
	FileName     string `json:"file_name" example:"attendance"`     // 文件名（不含 .go 后缀）
	RelativePath string `json:"relative_path" example:"attendance.go"` // 相对路径（相对于目录）
	Content      string `json:"content" example:"package hr\n..."`    // 文件内容
	// 向后兼容：保留 group_code（如果存在，优先使用 file_name）
	GroupCode    string `json:"group_code,omitempty" example:"attendance"` // 函数组代码（已废弃，使用 file_name）
}

// ReadDirectoryFilesRuntimeResp 读取目录文件响应（app-runtime -> app-server）
type ReadDirectoryFilesRuntimeResp struct {
	Success bool                `json:"success" example:"true"`   // 是否成功
	Message string              `json:"message" example:"读取成功"` // 响应消息
	Files   []DirectoryFileInfo `json:"files"`                    // 文件列表
}

