package dto

import "time"

// SystemStorageAssetsReq 系统管理员查询平台文件资产。
type SystemStorageAssetsReq struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
	RouterPrefix string `form:"router_prefix"`
	Status       string `form:"status"`
	Keyword      string `form:"keyword"`
}

type SystemStorageAsset struct {
	ID             int64      `json:"id"`
	Bucket         string     `json:"bucket"`
	Ref            string     `json:"ref"`
	FileKey        string     `json:"file_key"`
	Router         string     `json:"router"`
	FileName       string     `json:"file_name"`
	Description    string     `json:"description,omitempty"`
	FileSize       int64      `json:"file_size"`
	ContentType    string     `json:"content_type"`
	ThumbnailRef   string     `json:"thumbnail_ref,omitempty"`
	ThumbnailURL   string     `json:"thumbnail_url,omitempty"`
	PreviewKind    string     `json:"preview_kind,omitempty"`
	Previewable    bool       `json:"previewable"`
	Username       string     `json:"username"`
	Tenant         string     `json:"tenant"`
	Status         string     `json:"status"`
	UploadedAt     time.Time  `json:"uploaded_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	DeletedBy      string     `json:"deleted_by,omitempty"`
	DeleteError    string     `json:"delete_error,omitempty"`
	DownloadCount  int64      `json:"download_count"`
	PreviewCount   int64      `json:"preview_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
}

type SystemStorageAssetDirectory struct {
	Router    string `json:"router"`
	FileCount int64  `json:"file_count"`
	SizeBytes int64  `json:"size_bytes"`
}

type SystemStorageAssetWorkspace struct {
	Path      string `json:"path"`
	FileCount int64  `json:"file_count"`
	SizeBytes int64  `json:"size_bytes"`
}

type SystemStorageAssetSummary struct {
	ActiveFiles  int64 `json:"active_files"`
	ActiveBytes  int64 `json:"active_bytes"`
	DeletedFiles int64 `json:"deleted_files"`
	FailedFiles  int64 `json:"failed_files"`
}

type SystemStorageAssetsResp struct {
	List              []SystemStorageAsset          `json:"list"`
	Total             int64                         `json:"total"`
	Page              int                           `json:"page"`
	PageSize          int                           `json:"page_size"`
	Summary           SystemStorageAssetSummary     `json:"summary"`
	Directories       []SystemStorageAssetDirectory `json:"directories"`
	Workspaces        []SystemStorageAssetWorkspace `json:"workspaces"`
	MetadataAvailable bool                          `json:"metadata_available"`
	ConsoleURL        string                        `json:"console_url,omitempty"`
	Coverage          string                        `json:"coverage"`
}

type SystemStorageAssetDownloadReq struct {
	Ref    string `json:"ref" binding:"required"`
	Action string `json:"action"`
}

type SystemStorageAssetAuditsReq struct {
	Ref      string `form:"ref" binding:"required"`
	PageSize int    `form:"page_size"`
}

type SystemStorageAssetAudit struct {
	ID         int64     `json:"id"`
	Action     string    `json:"action"`
	Username   string    `json:"username,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	AccessedAt time.Time `json:"accessed_at"`
}

type SystemStorageAssetAuditsResp struct {
	List []SystemStorageAssetAudit `json:"list"`
}

type SystemStorageAssetDownloadResp struct {
	URL string `json:"url"`
}
