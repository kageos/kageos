package dto

const CapabilityBundleSchemaVersion = "capability.bundle.v1"

// CapabilityBundle 是跨工作空间复用的能力包，只保存相对 code/api 的包和文件结构。
type CapabilityBundle struct {
	SchemaVersion string                      `json:"schema_version"`
	Name          string                      `json:"name,omitempty"`
	TreeNodes     []*CapabilityBundleTreeNode `json:"tree_nodes,omitempty"`
	Docs          []*CapabilityBundleDoc      `json:"docs,omitempty"`
	Packages      []*CapabilityBundlePackage  `json:"packages"`
	Files         []*CapabilityBundleFile     `json:"files"`
}

type CapabilityBundleFile struct {
	PackagePath string `json:"package_path"`
	Path        string `json:"path"`
	Content     string `json:"content"`
}

type CapabilityBundlePackage struct {
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags,omitempty"`
}

type CapabilityBundleTreeNode struct {
	RelativePath string   `json:"relative_path"`
	ParentPath   string   `json:"parent_path,omitempty"`
	Type         string   `json:"type"`
	Code         string   `json:"code"`
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	TemplateType string   `json:"template_type,omitempty"`
	Method       string   `json:"method,omitempty"`
	Router       string   `json:"router,omitempty"`
	SortOrder    int      `json:"sort_order,omitempty"`
}

type CapabilityBundleDoc struct {
	RelativePath string `json:"relative_path"`
	Name         string `json:"name,omitempty"`
	Content      string `json:"content"`
	Format       string `json:"format,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Category     string `json:"category,omitempty"`
}

type ExportCapabilityBundleReq struct {
	SourceDirectoryPath  string   `json:"source_directory_path" form:"source_directory_path"`
	SourceDirectoryPaths []string `json:"source_directory_paths" form:"source_directory_paths"`
	SourceRootPath       string   `json:"source_root_path" form:"source_root_path"`
	Name                 string   `json:"name" form:"name"`
}

type InstallCapabilityOptions struct {
	TargetDirectoryPath string `json:"target_directory_path" binding:"required"`
	Overwrite           bool   `json:"overwrite,omitempty"`
	ForceDiff           bool   `json:"force_diff,omitempty"`
}

type InstallCapabilityBundleReq struct {
	InstallCapabilityOptions
	Bundle *CapabilityBundle `json:"bundle" binding:"required"`
}

type InstallCapabilityBundleFromURLReq struct {
	InstallCapabilityOptions
	BundleURL  string `json:"bundle_url" binding:"required"`
	InstallKey string `json:"install_key,omitempty"`
}

type InstallCapabilityBundleResp struct {
	Message             string   `json:"message"`
	DirectoryCount      int      `json:"directory_count"`
	FileCount           int      `json:"file_count"`
	DocCount            int      `json:"doc_count,omitempty"`
	TargetDirectoryPath string   `json:"target_directory_path"`
	CreatedPaths        []string `json:"created_paths,omitempty"`
	WrittenPaths        []string `json:"written_paths,omitempty"`
	OldVersion          string   `json:"old_version,omitempty"`
	NewVersion          string   `json:"new_version,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}
