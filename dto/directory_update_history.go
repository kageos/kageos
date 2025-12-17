package dto

import (
	"encoding/json"
	"time"
)

// GetAppVersionUpdateHistoryResp 获取应用版本更新历史响应（App视角）
// 二维数组结构：一个app有多个版本，每个版本里有数组列举了多个目录的变更
type GetAppVersionUpdateHistoryResp struct {
	AppID      int64                   `json:"app_id"`      // 应用ID
	AppVersion string                  `json:"app_version"` // 应用版本号（如果为空，表示返回所有版本）
	Versions   []*AppVersionUpdateInfo `json:"versions"`   // 版本列表（二维数组：第一层是版本，第二层是目录变更）
}

// AppVersionUpdateInfo 应用版本更新信息
type AppVersionUpdateInfo struct {
	AppVersion      string                 `json:"app_version"`       // 应用版本号
	DirectoryChanges []*DirectoryChangeInfo `json:"directory_changes"` // 该版本下的所有目录变更（数组）
}

// GetDirectoryUpdateHistoryResp 获取目录更新历史响应（目录视角）
type GetDirectoryUpdateHistoryResp struct {
	AppID            int64                  `json:"app_id"`             // 应用ID
	FullCodePath     string                 `json:"full_code_path"`     // 目录完整路径
	DirectoryChanges []*DirectoryChangeInfo  `json:"directory_changes"`  // 目录变更历史列表
	Paginated        *PaginatedInfo         `json:"paginated"`          // 分页信息
}

// DirectoryChangeInfo 目录变更信息
type DirectoryChangeInfo struct {
	FullCodePath  string          `json:"full_code_path"`  // 目录完整路径
	DirVersion    string          `json:"dir_version"`     // 目录版本号
	DirVersionNum int             `json:"dir_version_num"` // 目录版本号（数字部分）
	AppVersion    string          `json:"app_version"`     // 应用版本号（目录视角时使用）
	AppVersionNum int             `json:"app_version_num"` // 应用版本号（数字部分，目录视角时使用）
	AddedAPIs     json.RawMessage `json:"added_apis"`      // 新增的API摘要列表
	UpdatedAPIs   json.RawMessage `json:"updated_apis"`    // 更新的API摘要列表
	DeletedAPIs   json.RawMessage `json:"deleted_apis"`    // 删除的API摘要列表
	AddedCount    int             `json:"added_count"`     // 新增API数量
	UpdatedCount  int             `json:"updated_count"`    // 更新API数量
	DeletedCount  int             `json:"deleted_count"`   // 删除API数量
	Summary       string          `json:"summary"`         // 变更摘要（详情）
	UpdatedBy     string          `json:"updated_by"`      // 更新人
	CreatedAt     time.Time       `json:"created_at"`      // 创建时间
}

// PaginatedInfo 分页信息
type PaginatedInfo struct {
	CurrentPage int `json:"current_page"` // 当前页码
	TotalCount  int `json:"total_count"`   // 总数据量
	TotalPages  int `json:"total_pages"`   // 总页数
	PageSize    int `json:"page_size"`    // 每页数量
}

