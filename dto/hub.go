package dto

// PublishHubDirectoryReq 发布目录到 Hub 请求
type PublishHubDirectoryReq struct {
	APIKey               string            `json:"api_key"`                // API Key（私有化部署需要）
	SourceUser           string            `json:"source_user"`            // 源用户
	SourceApp            string            `json:"source_app"`             // 源应用
	SourceDirectoryPath  string            `json:"source_directory_path"`  // 源目录完整路径
	Name                 string            `json:"name"`                  // 目录名称
	Description          string            `json:"description"`            // 目录描述
	Category             string            `json:"category"`               // 分类
	Tags                 []string          `json:"tags"`                   // 标签
	ServiceFeePersonal   float64           `json:"service_fee_personal"`   // 个人用户服务费
	ServiceFeeEnterprise float64           `json:"service_fee_enterprise"` // 企业用户服务费
	Version              string            `json:"version"`                // 版本号（默认 v1）
	DirectoryTree        *DirectoryTreeNode `json:"directory_tree"`        // 目录树结构（递归，支持嵌套）
}

// DirectoryTreeNode 目录树节点（用于发布目录，包含文件内容）
type DirectoryTreeNode struct {
	Name           string              `json:"name"`            // 目录名称
	Path           string              `json:"path"`            // 目录完整路径
	Files          []*FileSnapshotInfo `json:"files"`           // 该目录下的文件列表（包含内容）
	Subdirectories []*DirectoryTreeNode `json:"subdirectories"` // 子目录列表（递归）
}

// FileSnapshotInfo 文件快照信息
type FileSnapshotInfo struct {
	FileName     string `json:"file_name"`     // 文件名（不含 .go 后缀）
	RelativePath string `json:"relative_path"` // 文件相对路径
	Content      string `json:"content"`       // 文件代码内容
	FileType     string `json:"file_type"`     // 文件类型
	FileVersion  string `json:"file_version"`  // 文件版本号
}

// PublishHubDirectoryResp 发布目录到 Hub 响应
type PublishHubDirectoryResp struct {
	HubDirectoryID  int64  `json:"hub_directory_id"`
	HubDirectoryURL string `json:"hub_directory_url"`
	DirectoryCount  int    `json:"directory_count"` // 包含的子目录数量
	FileCount       int    `json:"file_count"`      // 包含的文件数量
}

// UpdateHubDirectoryReq 更新目录到 Hub 请求（用于 push）
type UpdateHubDirectoryReq struct {
	APIKey               string            `json:"api_key"`                // API Key（私有化部署需要）
	HubDirectoryID       int64             `json:"hub_directory_id"`       // Hub 目录 ID（必需）
	SourceDirectoryPath  string            `json:"source_directory_path"`   // 源目录完整路径
	Name                 string            `json:"name"`                   // 目录名称（可选）
	Description          string            `json:"description"`            // 目录描述（可选）
	Category             string            `json:"category"`               // 分类（可选）
	Tags                 []string          `json:"tags"`                  // 标签（可选）
	ServiceFeePersonal   float64           `json:"service_fee_personal"`   // 个人用户服务费（可选）
	ServiceFeeEnterprise float64           `json:"service_fee_enterprise"` // 企业用户服务费（可选）
	Version              string            `json:"version"`                // 新版本号（必需）
	DirectoryTree        *DirectoryTreeNode `json:"directory_tree"`        // 目录树结构
}

// UpdateHubDirectoryResp 更新目录到 Hub 响应
type UpdateHubDirectoryResp struct {
	HubDirectoryID  int64  `json:"hub_directory_id"`
	HubDirectoryURL string `json:"hub_directory_url"`
	DirectoryCount  int    `json:"directory_count"` // 包含的子目录数量
	FileCount       int    `json:"file_count"`      // 包含的文件数量
	OldVersion      string `json:"old_version"`     // 旧版本号
	NewVersion      string `json:"new_version"`     // 新版本号
}

// HubDirectoryListResp Hub 目录列表响应
type HubDirectoryListResp struct {
	Items    []*HubDirectoryDTO `json:"items"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Total    int64               `json:"total"`
}

// HubDirectoryDTO Hub 目录 DTO（用于 API 返回）
type HubDirectoryDTO struct {
	ID        int64   `json:"id"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`

	// 基本信息
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"` // 标签数组

	// 目录路径信息
	PackagePath  string `json:"package_path"`  // 目录路径
	FullCodePath string `json:"full_code_path"` // 完整代码路径

	// 源信息
	SourceUser          string `json:"source_user"`
	SourceApp           string `json:"source_app"`
	SourceDirectoryPath string `json:"source_directory_path"`

	// 发布信息
	PublisherUsername string `json:"publisher_username"`
	PublishedAt       string `json:"published_at"`

	// 服务费信息
	ServiceFeePersonal   float64 `json:"service_fee_personal"`
	ServiceFeeEnterprise float64 `json:"service_fee_enterprise"`

	// 统计信息
	DownloadCount int     `json:"download_count"`
	TrialCount    int     `json:"trial_count"`
	Rating        float64 `json:"rating"`

	// 版本信息
	Version    string `json:"version"`     // 版本号（如 v1.0.0）
	VersionNum int    `json:"version_num"` // 版本号（数字部分）

	// 统计信息（快照）
	DirectoryCount int `json:"directory_count"` // 子目录数量
	FileCount      int `json:"file_count"`      // 文件数量
	FunctionCount  int `json:"function_count"`  // 函数数量
}

// HubDirectoryDetailDetailResp Hub 目录详情响应
type HubDirectoryDetailDetailResp struct {
	HubDirectoryDTO
	DirectoryTree *DirectoryTreeNode  `json:"directory_tree,omitempty"` // 目录树结构（可选，Files 字段可能为空，仅用于展示）
	Files         []*DirectoryFileDTO `json:"files,omitempty"`         // 文件列表（可选）
}

// DirectoryFileDTO 目录文件 DTO
type DirectoryFileDTO struct {
	FileName     string `json:"file_name"`     // 文件名
	RelativePath string `json:"relative_path"` // 文件相对路径
	FileType     string `json:"file_type"`     // 文件类型
	FileSize     int    `json:"file_size"`     // 文件大小
	// 注意：content 不包含在列表中，需要单独获取
}

