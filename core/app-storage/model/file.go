package model

import (
	"time"
)

// FileMetadata 文件元数据表（用于秒传去重）
type FileMetadata struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Hash        string    `gorm:"type:varchar(64);uniqueIndex;not null;comment:文件SHA256哈希" json:"hash"`
	Size        int64     `gorm:"not null;comment:文件大小（字节）" json:"size"`
	ContentType string    `gorm:"type:varchar(100);comment:MIME类型" json:"content_type"`
	StorageKey  string    `gorm:"type:varchar(500);not null;comment:MinIO中的实际存储位置" json:"storage_key"`
	RefCount    int       `gorm:"default:1;comment:引用计数" json:"ref_count"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (FileMetadata) TableName() string {
	return "file_metadata"
}

// FileReference 文件引用表（记录哪些函数使用了哪些文件）
type FileReference struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FileID     int64     `gorm:"not null;index;comment:关联file_metadata.id" json:"file_id"`
	Router     string    `gorm:"type:varchar(500);not null;index;comment:函数路径" json:"router"`
	LogicalKey string    `gorm:"type:varchar(500);not null;uniqueIndex;comment:逻辑Key（用户看到的）" json:"logical_key"`
	UploadedBy string    `gorm:"type:varchar(100);comment:上传者" json:"uploaded_by"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	
	// 关联
	FileMeta *FileMetadata `gorm:"foreignKey:FileID" json:"file_meta,omitempty"`
}

// TableName 指定表名
func (FileReference) TableName() string {
	return "file_references"
}

// FileUpload 文件上传记录表（审计）
type FileUpload struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FileKey     string    `gorm:"type:varchar(500);not null;uniqueIndex;comment:文件Key" json:"file_key"`
	Router      string    `gorm:"type:varchar(500);not null;index;comment:函数路径" json:"router"`
	FileName    string    `gorm:"type:varchar(255);not null;comment:原始文件名" json:"file_name"`
	FileSize    int64     `gorm:"not null;comment:文件大小（字节）" json:"file_size"`
	ContentType string    `gorm:"type:varchar(100);comment:MIME类型" json:"content_type"`
	Hash        string    `gorm:"type:varchar(64);index;comment:文件hash（用于秒传）" json:"hash"`
	
	// 用户信息（username 不可变，无需记录 user_id）
	UserID   *int64 `gorm:"index;comment:上传用户ID（已废弃，username 不可变）" json:"user_id,omitempty"`
	Username string `gorm:"type:varchar(100);not null;comment:上传用户名" json:"username"`
	Tenant   string `gorm:"type:varchar(100);not null;index;comment:租户" json:"tenant"`
	
	// 状态
	Status string `gorm:"type:varchar(20);default:'pending';comment:状态：pending/completed/failed" json:"status"`
	
	// 时间
	UploadedAt time.Time `gorm:"autoCreateTime;index" json:"uploaded_at"`
}

// TableName 指定表名
func (FileUpload) TableName() string {
	return "file_uploads"
}

// FileDownload 文件下载记录表（可选，审计）
type FileDownload struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	FileKey  string `gorm:"type:varchar(500);not null;index;comment:文件Key" json:"file_key"`
	
	// 下载用户（可能未登录）
	UserID   *int64  `gorm:"index;comment:下载用户ID" json:"user_id"`
	Username *string `gorm:"type:varchar(100);comment:下载用户名" json:"username"`
	
	// 下载信息
	IPAddress string    `gorm:"type:varchar(45);comment:IP地址" json:"ip_address"`
	UserAgent string    `gorm:"type:varchar(500);comment:User Agent" json:"user_agent"`
	
	// 时间
	DownloadedAt time.Time `gorm:"autoCreateTime;index" json:"downloaded_at"`
}

// TableName 指定表名
func (FileDownload) TableName() string {
	return "file_downloads"
}

