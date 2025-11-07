package storage

import "time"

// 上传来源常量（使用字符串值，与 dto.UploadSource 保持一致）
const (
	// UploadSourceBrowser 浏览器上传
	UploadSourceBrowser = "browser"
	// UploadSourceServer 服务端上传（容器内SDK）
	UploadSourceServer = "server"
)

// 时间相关常量
const (
	// DefaultDownloadURLExpiry 默认下载URL过期时间（7天）
	DefaultDownloadURLExpiry = 7 * 24 * time.Hour
	// DefaultUploadTokenExpiry 默认上传token过期时间（1小时）
	DefaultUploadTokenExpiry = 1 * time.Hour
)

// 时间格式常量
const (
	// TimeFormat 标准时间格式
	TimeFormat = "2006-01-02 15:04:05"
)

// 大小单位常量
const (
	// BytesPerKB 每KB的字节数
	BytesPerKB = 1024
	// BytesPerMB 每MB的字节数
	BytesPerMB = 1024 * 1024
	// BytesPerGB 每GB的字节数
	BytesPerGB = 1024 * 1024 * 1024
)

// HTTP 相关常量
const (
	// ContentTypeHeader HTTP Content-Type 头
	ContentTypeHeader = "Content-Type"
	// DefaultContentType 默认内容类型
	DefaultContentType = "application/octet-stream"
)

// 批量操作常量
const (
	// MaxBatchFiles 批量操作最大文件数
	MaxBatchFiles = 100
	// MinBatchFiles 批量操作最小文件数
	MinBatchFiles = 1
)

// IP 地址相关常量
const (
	// IPv6Loopback IPv6 回环地址
	IPv6Loopback = "::1"
	// IPv4Loopback IPv4 回环地址
	IPv4Loopback = "127.0.0.1"
)

