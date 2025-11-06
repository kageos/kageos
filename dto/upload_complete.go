package dto

// UploadCompleteReq 上传完成通知请求
type UploadCompleteReq struct {
	Key         string `json:"key" binding:"required"`         // 文件 Key
	Success     bool   `json:"success"`                        // 是否成功
	Error       string `json:"error,omitempty"`                // 错误信息（如果失败）
	Router      string `json:"router,omitempty"`               // ✨ 函数路径（上传成功后需要，用于记录）
	FileName    string `json:"file_name,omitempty"`            // ✨ 文件名（上传成功后需要，用于记录）
	FileSize    int64  `json:"file_size,omitempty"`            // ✨ 文件大小（上传成功后需要，用于记录）
	ContentType string `json:"content_type,omitempty"`         // ✨ 文件类型（上传成功后需要，用于记录）
	Hash        string `json:"hash,omitempty"`                 // ✨ 文件hash（可选，用于秒传）
}

// UploadCompleteResp 上传完成响应
type UploadCompleteResp struct {
	Message           string `json:"message"`                      // 消息
	DownloadURL       string `json:"download_url,omitempty"`       // ✨ 外部访问的下载地址（前端使用）
	ServerDownloadURL string `json:"server_download_url,omitempty"` // ✨ 内部访问的下载地址（服务端使用）
	Expire            string `json:"expire,omitempty"`             // 过期时间
}

