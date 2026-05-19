package dto

// UploadCompleteReq 上传完成通知请求
type UploadCompleteReq struct {
	Key          string `json:"key" binding:"required"`  // 文件 Key
	Bucket       string `json:"bucket,omitempty"`        // 存储桶；为空时使用默认桶
	Success      bool   `json:"success"`                 // 是否成功
	Error        string `json:"error,omitempty"`         // 错误信息（如果失败）
	Router       string `json:"router,omitempty"`        // ✨ 函数路径（上传成功后需要，用于记录）
	FileName     string `json:"file_name,omitempty"`     // ✨ 文件名（上传成功后需要，用于记录）
	Description  string `json:"description,omitempty"`   // 文件描述
	FileSize     int64  `json:"file_size,omitempty"`     // ✨ 文件大小（上传成功后需要，用于记录）
	ContentType  string `json:"content_type,omitempty"`  // ✨ 文件类型（上传成功后需要，用于记录）
	Hash         string `json:"hash,omitempty"`          // ✨ 文件hash（可选，用于文件标识和 SDK 下载缓存）
	ThumbnailRef string `json:"thumbnail_ref,omitempty"` // 前端生成的缩略图或视频封面文件引用
	PreviewKind  string `json:"preview_kind,omitempty"`  // 预览类型：image/video
}

// UploadCompleteResp 上传完成响应
type UploadCompleteResp struct {
	Message           string `json:"message"`                       // 消息
	Key               string `json:"key,omitempty"`                 // 文件 Key
	Bucket            string `json:"bucket,omitempty"`              // 存储桶
	Ref               string `json:"ref,omitempty"`                 // 稳定文件引用：bucket/object_key
	FileName          string `json:"file_name,omitempty"`           // 文件名
	Description       string `json:"description,omitempty"`         // 文件描述
	FileSize          int64  `json:"file_size,omitempty"`           // 文件大小
	ContentType       string `json:"content_type,omitempty"`        // 文件类型
	Hash              string `json:"hash,omitempty"`                // 文件hash
	ThumbnailRef      string `json:"thumbnail_ref,omitempty"`       // 前端生成的缩略图或视频封面文件引用
	ThumbnailURL      string `json:"thumbnail_url,omitempty"`       // 缩略图或视频封面浏览器访问地址
	PreviewKind       string `json:"preview_kind,omitempty"`        // 预览类型：image/video
	Storage           string `json:"storage,omitempty"`             // 存储引擎
	DownloadURL       string `json:"download_url,omitempty"`        // ✨ 外部访问的下载地址（前端使用）
	ServerDownloadURL string `json:"server_download_url,omitempty"` // ✨ 内部访问的下载地址（服务端使用）
	Expire            string `json:"expire,omitempty"`              // 过期时间
}
