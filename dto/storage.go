package dto

// GetUploadTokenReq 获取上传凭证请求
type GetUploadTokenReq struct {
	FileName      string `json:"file_name" binding:"required"`
	ContentType   string `json:"content_type"`
	FileSize      int64  `json:"file_size"`
	Router        string `json:"router,omitempty"`          // 函数路径，例如：luobei/test88888/cashier/cashier_desk.form（可选，未提供时使用默认路由：/{username}/default）
	Bucket        string `json:"bucket,omitempty"`          // 存储桶；为空时使用 storage 默认桶
	Hash          string `json:"hash,omitempty"`            // 文件 hash（可选，用于文件标识和 SDK 下载缓存）
	PreviewForKey string `json:"preview_for_key,omitempty"` // 原文件 object key；存在时生成与原文件同路径的缩略图/封面 key
}

// UploadMethod 上传方式
type UploadMethod string

const (
	UploadMethodPresignedURL UploadMethod = "presigned_url" // 预签名 URL（当前官方实际使用）
)

// GetUploadTokenResp 获取上传凭证响应
type GetUploadTokenResp struct {
	// 通用字段
	Key      string       `json:"key"`                // 文件 Key
	Bucket   string       `json:"bucket"`             // 存储桶
	Ref      string       `json:"ref"`                // 稳定文件引用：bucket/object_key
	Expire   string       `json:"expire"`             // 过期时间
	Method   UploadMethod `json:"method"`             // 上传方式 ✨ 新增
	Storage  string       `json:"storage,omitempty"`  // ✨ 存储引擎（当前固定为 minio）
	Username string       `json:"username,omitempty"` // ✨ 当前登录用户的用户名

	// 预签名 URL 上传（当前官方仅 MinIO）
	UploadURL       string            `json:"upload_url,omitempty"`        // 外部访问的预签名上传地址（前端使用）
	ServerUploadURL string            `json:"server_upload_url,omitempty"` // 内部访问的预签名上传地址（服务端/SDK使用）
	Headers         map[string]string `json:"headers,omitempty"`           // 请求头

	// 上传域名信息 ✨ 新增
	UploadHost   string `json:"upload_host,omitempty"`   // 上传目标 host（例如：localhost:9000，用于 CORS、进度监听）
	UploadDomain string `json:"upload_domain,omitempty"` // 上传完整域名（例如：http://localhost:9000，用于日志、调试）

	// CDN 域名（可选，用于下载访问）
	CDNDomain string `json:"cdn_domain,omitempty"`

	// ✨ 预期的下载地址（在获取token时预先构建，上传成功后可直接使用）
	DownloadURL       string `json:"download_url,omitempty"`        // ✨ 外部访问的下载地址（前端使用）
	ServerDownloadURL string `json:"server_download_url,omitempty"` // ✨ 内部访问的下载地址（服务端/SDK使用）
}

// GetFileURLResp 获取文件下载地址响应
type GetFileURLResp struct {
	DownloadURL string `json:"download_url"`         // 预签名下载地址
	Key         string `json:"key"`                  // 文件 Key
	Expire      string `json:"expire"`               // 过期时间
	CDNDomain   string `json:"cdn_domain,omitempty"` // CDN 域名（可选，用于前端构建 CDN URL）
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
	Files []GetUploadTokenReq `json:"files" binding:"required,min=1,max=100"` // 文件列表（最多100个）
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

// BatchUploadCompleteResp 批量上传完成响应
type BatchUploadCompleteResp struct {
	Results []BatchUploadCompleteResult `json:"results"` // 处理结果列表
}

// BatchUploadCompleteResult 批量上传完成结果
type BatchUploadCompleteResult struct {
	Key               string `json:"key"`                           // 文件 Key
	Bucket            string `json:"bucket,omitempty"`              // 存储桶
	Ref               string `json:"ref,omitempty"`                 // 稳定文件引用：bucket/object_key
	Status            string `json:"status"`                        // 状态：completed/failed
	DownloadURL       string `json:"download_url,omitempty"`        // ✨ 外部访问的下载地址（前端使用）
	Description       string `json:"description,omitempty"`         // 文件描述
	ServerDownloadURL string `json:"server_download_url,omitempty"` // ✨ 内部访问的下载地址（服务端使用）
	Hash              string `json:"hash,omitempty"`                // ✨ 文件hash（用于 SDK 下载缓存）
	ThumbnailRef      string `json:"thumbnail_ref,omitempty"`       // 前端生成的缩略图或视频封面文件引用
	ThumbnailURL      string `json:"thumbnail_url,omitempty"`       // 缩略图或视频封面浏览器访问地址
	PreviewKind       string `json:"preview_kind,omitempty"`        // 预览类型：image/video
	Error             string `json:"error,omitempty"`               // 错误信息（如果失败）
}

type ResolveFileRefsReq struct {
	Refs     []string `json:"refs" binding:"required,min=1,max=100"`
	Audience string   `json:"audience,omitempty"` // browser/server/all；为空时返回 browser 和 server URL
}

type ResolveFileRefsResp struct {
	Files []ResolvedFile `json:"files"`
}

type UpdateFileDescriptionReq struct {
	Ref         string `json:"ref,omitempty"`
	Bucket      string `json:"bucket,omitempty"`
	Key         string `json:"key,omitempty"`
	Description string `json:"description"`
}

type UpdateFileDescriptionResp struct {
	Ref         string `json:"ref"`
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

type ResolvedFile struct {
	Ref                string `json:"ref"`
	Bucket             string `json:"bucket"`
	Key                string `json:"key"`
	Name               string `json:"name,omitempty"`
	SourceName         string `json:"source_name,omitempty"`
	Storage            string `json:"storage,omitempty"`
	Description        string `json:"description,omitempty"`
	Size               int64  `json:"size,omitempty"`
	ContentType        string `json:"content_type,omitempty"`
	Hash               string `json:"hash,omitempty"`
	UploadUser         string `json:"upload_user,omitempty"`
	UploadTs           int64  `json:"upload_ts,omitempty"`
	DownloadURL        string `json:"download_url,omitempty"`
	ServerDownloadURL  string `json:"server_download_url,omitempty"`
	ThumbnailRef       string `json:"thumbnail_ref,omitempty"`
	ThumbnailURL       string `json:"thumbnail_url,omitempty"`
	ServerThumbnailURL string `json:"server_thumbnail_url,omitempty"`
	PreviewKind        string `json:"preview_kind,omitempty"`
	Error              string `json:"error,omitempty"`
}
