package dto

const (
	HubDirectoryBundleSchemaVersion = 1
	HubDirectoryBundleType          = "hub_directory_bundle"
)

// ExportHubDirectoryBundleRequest 导出目录 JSON 安装包请求
type ExportHubDirectoryBundleRequest struct {
	HubDirectoryID int64  `json:"hub_directory_id" form:"hub_directory_id"` // Hub 目录ID（可选，与 full_code_path 二选一）
	FullCodePath   string `json:"full_code_path" form:"full_code_path"`     // 目录完整路径（可选，与 hub_directory_id 二选一）
	Version        string `json:"version" form:"version"`                   // 版本号（可选，不传则导出当前版本）
}

// HubDirectoryInstallBundle Hub 目录 JSON 安装包
type HubDirectoryInstallBundle struct {
	SchemaVersion    int                `json:"schema_version"`
	BundleType       string             `json:"bundle_type"`
	ExportedAt       string             `json:"exported_at,omitempty"`
	HubDirectoryName string             `json:"hub_directory_name,omitempty"`
	HubFullCodePath  string             `json:"hub_full_code_path,omitempty"`
	HubVersionNum    int                `json:"hub_version_num,omitempty"`
	DirectoryTree    *DirectoryTreeNode `json:"directory_tree"`
}
