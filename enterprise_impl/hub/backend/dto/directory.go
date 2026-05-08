package dto

import (
	"encoding/json"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// DirectoryFileSnapshot 目录文件快照（用于上传）
type DirectoryFileSnapshot struct {
	FullCodePath string              `json:"full_code_path"` // 目录完整路径
	ParentPath   string              `json:"parent_path"`    // 父目录完整路径（空字符串表示根目录），用于快速构建目录树
	Files        []*FileSnapshotInfo `json:"files"`          // 该目录下的文件列表
}

// FileSnapshotInfo 文件快照信息
type FileSnapshotInfo struct {
	FileName     string `json:"file_name"`     // 文件名（不含 .go 后缀）
	RelativePath string `json:"relative_path"` // 文件相对路径
	Content      string `json:"content"`       // 文件代码内容
	FileType     string `json:"file_type"`     // 文件类型
	FileVersion  string `json:"file_version"`  // 文件版本号
}

// DirectoryTreeNode 目录树节点（用于发布目录，包含文件内容和函数）
type DirectoryTreeNode struct {
	Type           string               `json:"type"`            // 节点类型：package（目录）或 function（函数）
	Name           string               `json:"name"`            // 目录名称（中文显示名称）
	Code           string               `json:"code"`            // 目录代码（英文标识）
	Path           string               `json:"path"`            // 目录完整路径
	Files          []*FileSnapshotInfo  `json:"files,omitempty"` // 该目录下的文件列表（包含内容，发布时需要，获取详情时不返回）
	Functions      []*HubFunctionInfo   `json:"functions"`       // 该目录下的函数列表（新增）
	Subdirectories []*DirectoryTreeNode `json:"subdirectories"`  // 子目录列表（递归）
}

// HubFunctionInfo 函数信息（用于 Hub 目录树与快照函数定义）
// Schema 为统一扩展字段：内含 request/response，后续可按 template_type 放不同结构（如 form/table/chart 各自 schema）
type HubFunctionInfo struct {
	ID           int64    `json:"id"`             // ServiceTree 节点ID
	Name         string   `json:"name"`           // 函数名称
	Code         string   `json:"code"`           // 函数代码
	FullCodePath string   `json:"full_code_path"` // 完整代码路径
	Description  string   `json:"description"`    // 函数描述
	TemplateType string   `json:"template_type"`  // 函数类型（如 form, table, chart 等）
	Tags         []string `json:"tags"`           // 标签
	RefID        int64    `json:"ref_id"`         // 指向真实的 function ID
	Version      string   `json:"version"`        // 函数版本号
	VersionNum   int      `json:"version_num"`    // 版本号数字部分
	// 函数完整定义，推送到 Hub 时存入 SnapshotFunctionDefs JSON
	Method       string   `json:"method,omitempty"`        // HTTP 方法
	Router       string   `json:"router,omitempty"`        // 路由（full-code-path）
	CreateTables string   `json:"create_tables,omitempty"` // 创建表配置
	Callbacks    []string `json:"callbacks,omitempty"`     // 回调能力摘要
	// Schema：统一函数配置，按 template_type 放 form/table/chart 结构（JSON 对象）
	Schema json.RawMessage `json:"schema,omitempty"` // 如 {"version":1,"type":"form","form":...}
}

// FileNode 文件节点（用于展示，不包含内容）
type FileNode struct {
	Name         string `json:"name"`          // 文件名
	RelativePath string `json:"relative_path"` // 文件相对路径
	FileType     string `json:"file_type"`     // 文件类型
}

// --- 快照三字段：结构(展示) / 文件(复制) / 函数定义(预览) ---

// SnapshotFileEntry 快照文件项（用于「复制」：按相对路径写文件）
type SnapshotFileEntry struct {
	RelativePath string `json:"relative_path"` // 相对路径，复制时写文件用
	Content      string `json:"content"`       // 文件内容
	FileType     string `json:"file_type"`     // 文件类型
}

// SnapshotTree 快照目录结构（用于「展示」：树/列表/面包屑，不含文件内容和函数详情）
// 与 DirectoryTreeNode 同构，但每个节点的 Files 仅含 name/relative_path/file_type，不含 content
type SnapshotTree = DirectoryTreeNode

// SnapshotFunctionDefs 快照函数定义列表（用于「预览」：查看函数入参、描述等）
// 平铺列表，按 full_code_path 可定位到具体函数
type SnapshotFunctionDefs = []*HubFunctionInfo

// PublishHubDirectoryRequest 发布目录到 Hub 请求
type PublishHubDirectoryRequest struct {
	APIKey               string             `json:"api_key"`                // API Key（私有化部署需要）
	SourceUser           string             `json:"source_user"`            // 源用户
	SourceApp            string             `json:"source_app"`             // 源应用
	SourceDirectoryPath  string             `json:"source_directory_path"`  // 源目录完整路径
	Name                 string             `json:"name"`                   // 目录名称
	Description          string             `json:"description"`            // 目录描述
	Category             string             `json:"category"`               // 分类
	Tags                 []string           `json:"tags"`                   // 标签
	ServiceFeePersonal   float64            `json:"service_fee_personal"`   // 个人用户服务费
	ServiceFeeEnterprise float64            `json:"service_fee_enterprise"` // 企业用户服务费
	Version              string             `json:"version"`                // 版本号（默认 v1）
	DirectoryTree        *DirectoryTreeNode `json:"directory_tree"`         // 目录树结构（递归，支持嵌套）
}

// PublishHubDirectoryResponse 发布目录到 Hub 响应
type PublishHubDirectoryResponse struct {
	HubFullCodePath string `json:"hub_full_code_path"` // Hub 目录完整路径，前端用此拼详情 URL
	DirectoryCount  int    `json:"directory_count"`    // 包含的子目录数量
	FileCount       int    `json:"file_count"`         // 包含的文件数量
}

// UpdateHubDirectoryRequest 更新目录到 Hub 请求（用于 push）
type UpdateHubDirectoryRequest struct {
	APIKey               string             `json:"api_key"`                // API Key（私有化部署需要）
	HubDirectoryID       int64              `json:"hub_directory_id"`       // Hub 目录 ID（必需）
	SourceDirectoryPath  string             `json:"source_directory_path"`  // 源目录完整路径
	Name                 string             `json:"name"`                   // 目录名称（可选，不传则保持原值）
	Description          string             `json:"description"`            // 目录描述（可选，不传则保持原值）
	Category             string             `json:"category"`               // 分类（可选，不传则保持原值）
	Tags                 []string           `json:"tags"`                   // 标签（可选，不传则保持原值）
	ServiceFeePersonal   float64            `json:"service_fee_personal"`   // 个人用户服务费（可选）
	ServiceFeeEnterprise float64            `json:"service_fee_enterprise"` // 企业用户服务费（可选）
	Version              string             `json:"version"`                // 新版本号（必需，必须大于当前版本）
	UpdateDescription    string             `json:"update_description"`     // 本版本更新说明（可选，存到快照 Description）
	DirectoryTree        *DirectoryTreeNode `json:"directory_tree"`         // 目录树结构（递归，支持嵌套）
}

// UpdateHubDirectoryResponse 更新目录到 Hub 响应
type UpdateHubDirectoryResponse struct {
	HubFullCodePath string `json:"hub_full_code_path"` // Hub 目录完整路径，前端用此拼详情 URL
	DirectoryCount  int    `json:"directory_count"`    // 包含的子目录数量
	FileCount       int    `json:"file_count"`         // 包含的文件数量
	OldVersion      string `json:"old_version"`        // 旧版本号
	NewVersion      string `json:"new_version"`        // 新版本号
}

// HubDirectoryDTO Hub 目录 DTO（用于 API 返回）
type HubDirectoryDTO struct {
	ID        int64       `json:"id"`
	CreatedAt models.Time `json:"created_at"`
	UpdatedAt models.Time `json:"updated_at"`

	// 状态：active=在架；deleted=已下架（列表不展示，通过链接仍可访问）
	Status string `json:"status"`

	// 基本信息
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"` // 标签数组

	// 目录路径信息
	PackagePath  string `json:"package_path"`   // 目录路径
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

	// 复制链接（用于 copy_directory 工具或前端「复制链接」）
	// 格式：hub://{host}/{full_code_path}@{version}，可直接用于从 Hub 复制到本地
	CopyURL string `json:"copy_url"`

	// 星星数（类似 GitHub star，便于排序与推荐）
	StarCount int `json:"star_count"`

	// 当前用户是否已加星（仅详情返回；未登录或未加星为 false）
	HasStarred bool `json:"has_starred"`
}

// GetHubDirectoryDetailRequest 获取目录详情请求（通用 DTO）
type GetHubDirectoryDetailRequest struct {
	HubDirectoryID int64  `json:"hub_directory_id" form:"hub_directory_id"` // Hub 目录ID（可选，与 full_code_path 二选一）
	FullCodePath   string `json:"full_code_path" form:"full_code_path"`     // 目录完整路径（可选，与 hub_directory_id 二选一）
	Version        string `json:"version" form:"version"`                   // 版本号（可选，不传则使用当前版本）
	IncludeTree    bool   `json:"include_tree" form:"include_tree"`         // 是否包含目录树结构
}

// GetHubDirectoryVersionsRequest 获取目录版本列表请求
type GetHubDirectoryVersionsRequest struct {
	HubDirectoryID int64  `json:"hub_directory_id" form:"hub_directory_id"` // Hub 目录ID（与 full_code_path 二选一）
	FullCodePath   string `json:"full_code_path" form:"full_code_path"`     // 目录完整路径（与 hub_directory_id 二选一）
}

// HubDirectoryVersionItem 目录版本项（用于版本列表）
type HubDirectoryVersionItem struct {
	Version           string `json:"version"`            // 版本号（如 v1, v2）
	VersionNum        int    `json:"version_num"`        // 版本号数字
	SnapshotAt        string `json:"snapshot_at"`        // 快照时间（RFC3339）
	IsCurrent         bool   `json:"is_current"`         // 是否为当前版本
	Description       string `json:"description"`        // 本版本更新说明（可选）
	PublisherUsername string `json:"publisher_username"` // 该版本的上传人
}

// GetHubDirectoryVersionsResponse 获取目录版本列表响应
type GetHubDirectoryVersionsResponse struct {
	Items []*HubDirectoryVersionItem `json:"items"` // 版本列表（按版本号倒序，最新在前）
}

// HubDirectoryDetailDTO Hub 目录详情 DTO
type HubDirectoryDetailDTO struct {
	HubDirectoryDTO
	DirectoryTree      *DirectoryTreeNode `json:"directory_tree,omitempty"`      // 目录树结构（可选）
	VersionDescription string             `json:"version_description,omitempty"` // 当前查看版本的更新说明（可选，推送时填的「本版本更新说明」）
}

// DirectoryFileDTO 目录文件 DTO
type DirectoryFileDTO struct {
	FileName     string `json:"file_name"`         // 文件名
	RelativePath string `json:"relative_path"`     // 文件相对路径
	FileType     string `json:"file_type"`         // 文件类型
	FileSize     int    `json:"file_size"`         // 文件大小
	Content      string `json:"content,omitempty"` // 文件内容（可选，当 include_files=true 且需要内容时返回）
}

// GetHubDirectoryListRequest 获取目录列表请求（通用 DTO）
type GetHubDirectoryListRequest struct {
	Page              int    `json:"page" form:"page"`                             // 页码
	PageSize          int    `json:"page_size" form:"page_size"`                   // 每页数量
	Search            string `json:"search" form:"search"`                         // 搜索关键词
	Category          string `json:"category" form:"category"`                     // 分类
	PublisherUsername string `json:"publisher_username" form:"publisher_username"` // 发布者用户名
	MineOnly          bool   `json:"mine_only" form:"mine_only"`                   // 只看自己：true 时按当前用户过滤（需带 token/网关带 X-Request-User）
	FeeType           string `json:"fee_type" form:"fee_type"`                     // 费用筛选：空=全部，free=免费，paid=收费
	OrderBy           string `json:"order_by" form:"order_by"`                     // 排序：空或 latest=最新(created_at DESC)，hot=热门(star+download 优先)
}

// HubDirectoryListResponse Hub 目录列表响应
type HubDirectoryListResponse struct {
	Items    []*HubDirectoryDTO `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}
