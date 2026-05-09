package dto

const DirectoryBundleSchemaVersion = 1

// DirectoryBundle 是最小目录树 JSON，用于导出任意目录并粘贴到任意目标目录下。
type DirectoryBundle struct {
	SchemaVersion int                  `json:"schema_version"`
	Root          *DirectoryBundleNode `json:"root"`
}

type DirectoryBundleNode struct {
	Code        string                 `json:"code"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Files       []*DirectoryBundleFile `json:"files,omitempty"`
	Children    []*DirectoryBundleNode `json:"children,omitempty"`
}

type DirectoryBundleFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ExportDirectoryBundleReq struct {
	SourceDirectoryPath string `json:"source_directory_path" form:"source_directory_path" binding:"required"`
}

type ImportDirectoryBundleReq struct {
	TargetDirectoryPath string           `json:"target_directory_path" binding:"required"`
	ConflictPolicy      string           `json:"conflict_policy,omitempty"`
	Bundle              *DirectoryBundle `json:"bundle" binding:"required"`
}

type ImportDirectoryBundleResp struct {
	Message             string   `json:"message"`
	DirectoryCount      int      `json:"directory_count"`
	FileCount           int      `json:"file_count"`
	TargetDirectoryPath string   `json:"target_directory_path"`
	CreatedPaths        []string `json:"created_paths,omitempty"`
	WrittenPaths        []string `json:"written_paths,omitempty"`
	OldVersion          string   `json:"old_version,omitempty"`
	NewVersion          string   `json:"new_version,omitempty"`
}
