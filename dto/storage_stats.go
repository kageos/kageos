package dto

// GetStorageStatsReq 获取存储统计请求
type GetStorageStatsReq struct {
	Router string `form:"router" binding:"required"` // 函数路径，例如：luobei/test88888/plugins/cashier_desk
}

// GetStorageStatsResp 获取存储统计响应
type GetStorageStatsResp struct {
	Router    string `json:"router"`
	FileCount int    `json:"file_count"`
	TotalSize int64  `json:"total_size"`
	SizeHuman string `json:"size_human"` // 人类可读的大小，例如：10.5 MB
}

// ListFilesReq 列举文件请求
type ListFilesReq struct {
	Router string `form:"router" binding:"required"` // 函数路径
}

// ListFilesResp 列举文件响应
type ListFilesResp struct {
	Router string   `json:"router"`
	Files  []string `json:"files"`
	Count  int      `json:"count"`
}

// DeleteFilesByRouterReq 删除函数路径下所有文件请求
type DeleteFilesByRouterReq struct {
	Router string `json:"router" binding:"required"` // 函数路径
}

// DeleteFilesByRouterResp 删除文件响应
type DeleteFilesByRouterResp struct {
	Router       string `json:"router"`
	DeletedCount int    `json:"deleted_count"`
}
