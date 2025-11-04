package dto

// UploadCompleteReq 上传完成通知请求
type UploadCompleteReq struct {
	Key     string `json:"key" binding:"required"`     // 文件 Key
	Success bool   `json:"success"`                    // 是否成功
	Error   string `json:"error,omitempty"`            // 错误信息（如果失败）
}

// UploadCompleteResp 上传完成响应
type UploadCompleteResp struct {
	Message     string `json:"message"`                  // 消息
	DownloadURL string `json:"download_url,omitempty"`   // ✨ 下载 URL（上传成功后返回，方便直接下载）
	Expire      string `json:"expire,omitempty"`         // 过期时间
}

