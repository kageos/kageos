package dto

const (
	HubDirectoryBundleSchemaVersion = 1
	HubDirectoryBundleType          = "hub_directory_bundle"
)

// HubDirectoryInstallBundle 标准 Hub 目录 JSON 安装包
type HubDirectoryInstallBundle struct {
	SchemaVersion    int                `json:"schema_version"`
	BundleType       string             `json:"bundle_type"`
	ExportedAt       string             `json:"exported_at,omitempty"`
	HubDirectoryName string             `json:"hub_directory_name,omitempty"`
	HubFullCodePath  string             `json:"hub_full_code_path,omitempty"`
	HubVersionNum    int                `json:"hub_version_num,omitempty"`
	DirectoryTree    *DirectoryTreeNode `json:"directory_tree"`
}
