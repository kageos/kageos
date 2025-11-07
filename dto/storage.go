package dto

// UploadSource 上传来源
type UploadSource string

const (
	UploadSourceBrowser UploadSource = "browser" // 浏览器上传
	UploadSourceServer  UploadSource = "server" // 服务端上传（容器内SDK）
)

// GetUploadTokenReq 获取上传凭证请求
type GetUploadTokenReq struct {
	FileName     string       `json:"file_name" binding:"required"`
	ContentType  string       `json:"content_type"`
	FileSize     int64        `json:"file_size"`
	Router       string       `json:"router" binding:"required"` // 函数路径，例如：luobei/test88888/tools/cashier_desk
	Hash         string       `json:"hash,omitempty"`            // 文件 hash（预留，用于秒传）
	UploadSource UploadSource `json:"upload_source,omitempty"`   // ✨ 上传来源：browser（浏览器）或 server（服务端），默认为 browser
}

// UploadMethod 上传方式
type UploadMethod string

const (
	UploadMethodPresignedURL UploadMethod = "presigned_url" // 预签名 URL（标准 S3 协议）
	UploadMethodFormUpload   UploadMethod = "form_upload"   // 表单上传（七牛云等）
	UploadMethodSDKUpload    UploadMethod = "sdk_upload"    // SDK 上传（特殊云存储）
)

// GetUploadTokenResp 获取上传凭证响应
type GetUploadTokenResp struct {
	// 通用字段
	Key     string       `json:"key"`               // 文件 Key
	Bucket  string       `json:"bucket"`            // 存储桶
	Expire  string       `json:"expire"`            // 过期时间
	Method  UploadMethod `json:"method"`            // 上传方式 ✨ 新增
	Storage string       `json:"storage,omitempty"` // ✨ 存储引擎（minio/qiniu/tencentcos/aliyunoss/awss3）

	// 预签名 URL 上传（MinIO、COS、OSS、S3）
	URL      string            `json:"url,omitempty"`       // ✨ 外部访问的预签名 URL（前端使用）
	ServerURL string            `json:"server_url,omitempty"` // ✨ 内部访问的预签名 URL（服务端/SDK使用）
	Headers  map[string]string `json:"headers,omitempty"`   // 请求头

	// 上传域名信息 ✨ 新增
	UploadHost   string `json:"upload_host,omitempty"`   // 上传目标 host（例如：localhost:9000，用于 CORS、进度监听）
	UploadDomain string `json:"upload_domain,omitempty"` // 上传完整域名（例如：http://localhost:9000，用于日志、调试）

	// 表单上传（七牛云、又拍云等）
	FormData map[string]string `json:"form_data,omitempty"` // 表单字段
	PostURL  string            `json:"post_url,omitempty"`  // POST 地址

	// SDK 上传（特殊云存储）
	SDKConfig map[string]interface{} `json:"sdk_config,omitempty"` // SDK 配置

	// CDN 域名（可选，用于下载访问）
	CDNDomain string `json:"cdn_domain,omitempty"`
	
	// ✨ 预期的下载地址（在获取token时预先构建，上传成功后可直接使用）
	DownloadURL    string `json:"download_url,omitempty"`     // ✨ 外部访问的下载地址（前端使用）
	ServerDownloadURL string `json:"server_download_url,omitempty"` // ✨ 内部访问的下载地址（服务端/SDK使用）
}

// GetFileURLResp 获取文件 URL 响应
type GetFileURLResp struct {
	URL       string `json:"url"`                  // 预签名下载 URL
	Key       string `json:"key"`                  // 文件 Key
	Expire    string `json:"expire"`               // 过期时间
	CDNDomain string `json:"cdn_domain,omitempty"` // ✨ CDN 域名（可选，用于前端构建 CDN URL）
}

// GetFileInfoResp 获取文件信息响应
type GetFileInfoResp struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ContentType  string `json:"content_type"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
}

// BatchGetUploadTokenReq 批量获取上传凭证请求
type BatchGetUploadTokenReq struct {
	Files        []GetUploadTokenReq `json:"files" binding:"required,min=1,max=100"` // 文件列表（最多100个）
	UploadSource UploadSource         `json:"upload_source,omitempty"`                 // ✨ 上传来源：browser（浏览器）或 server（服务端），默认为 browser
}

// BatchGetUploadTokenResp 批量获取上传凭证响应
type BatchGetUploadTokenResp struct {
	Tokens []GetUploadTokenResp `json:"tokens"` // 上传凭证列表
}

// BatchUploadCompleteReq 批量上传完成通知请求
type BatchUploadCompleteReq struct {
	Items []BatchUploadCompleteItem `json:"items" binding:"required,min=1,max=100"` // 上传结果列表（最多100个）
}

// BatchUploadCompleteItem 批量上传完成项
type BatchUploadCompleteItem struct {
	Key         string `json:"key" binding:"required"`         // 文件 Key
	Success     bool   `json:"success"`                         // 是否成功
	Error       string `json:"error,omitempty"`                // 错误信息（如果失败）
	Router      string `json:"router,omitempty"`               // ✨ 函数路径（上传成功后需要，用于记录）
	FileName    string `json:"file_name,omitempty"`            // ✨ 文件名（上传成功后需要，用于记录）
	FileSize    int64  `json:"file_size,omitempty"`            // ✨ 文件大小（上传成功后需要，用于记录）
	ContentType string `json:"content_type,omitempty"`         // ✨ 文件类型（上传成功后需要，用于记录）
	Hash        string `json:"hash,omitempty"`                 // ✨ 文件hash（可选，用于秒传）
}

// BatchUploadCompleteResp 批量上传完成响应
type BatchUploadCompleteResp struct {
	Results []BatchUploadCompleteResult `json:"results"` // 处理结果列表
}

// BatchUploadCompleteResult 批量上传完成结果
type BatchUploadCompleteResult struct {
	Key              string `json:"key"`                        // 文件 Key
	Status           string `json:"status"`                     // 状态：completed/failed
	DownloadURL      string `json:"download_url,omitempty"`     // ✨ 外部访问的下载地址（前端使用）
	ServerDownloadURL string `json:"server_download_url,omitempty"` // ✨ 内部访问的下载地址（服务端使用）
	Hash             string `json:"hash,omitempty"`              // ✨ 文件hash（用于文件缓存去重）
	Error            string `json:"error,omitempty"`            // 错误信息（如果失败）
}
