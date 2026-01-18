package dto

// 注意：DiffData 定义在 dto/app_runtime_namespace.go 中

// PublishDirectoryToHubReq 发布目录到 Hub 请求
type PublishDirectoryToHubReq struct {
	SourceUser           string   `json:"source_user" binding:"required"`           // 源用户
	SourceApp            string   `json:"source_app" binding:"required"`            // 源应用
	SourceDirectoryPath  string   `json:"source_directory_path" binding:"required"` // 源目录完整路径
	Name                 string   `json:"name" binding:"required"`                  // 目录名称
	Description          string   `json:"description"`                              // 目录描述
	Category             string   `json:"category"`                                 // 分类
	Tags                 []string `json:"tags"`                                     // 标签
	ServiceFeePersonal   float64  `json:"service_fee_personal"`                     // 个人用户服务费
	ServiceFeeEnterprise float64  `json:"service_fee_enterprise"`                   // 企业用户服务费
}

// PublishDirectoryToHubResp 发布目录到 Hub 响应
type PublishDirectoryToHubResp struct {
	HubDirectoryID  int64  `json:"hub_directory_id"`
	HubDirectoryURL string `json:"hub_directory_url"`
	DirectoryCount  int    `json:"directory_count"` // 包含的子目录数量
	FileCount       int    `json:"file_count"`      // 包含的文件数量
}

// PushDirectoryToHubReq 推送目录到 Hub 请求（用于 push，更新已发布的目录）
type PushDirectoryToHubReq struct {
	SourceUser           string   `json:"source_user"`            // 源用户
	SourceApp            string   `json:"source_app"`             // 源应用
	SourceDirectoryPath  string   `json:"source_directory_path"`  // 源目录完整路径
	Name                 string   `json:"name"`                   // 目录名称（可选，不传则保持原值）
	Description          string   `json:"description"`            // 目录描述（可选，不传则保持原值）
	Category             string   `json:"category"`               // 分类（可选，不传则保持原值）
	Tags                 []string `json:"tags"`                   // 标签（可选，不传则保持原值）
	ServiceFeePersonal   float64  `json:"service_fee_personal"`   // 个人用户服务费（可选）
	ServiceFeeEnterprise float64  `json:"service_fee_enterprise"` // 企业用户服务费（可选）
	Version              string   `json:"version"`                // 新版本号（必需，必须大于当前版本）
	APIKey               string   `json:"api_key"`                // API Key（私有化部署需要）
}

// PushDirectoryToHubResp 推送目录到 Hub 响应
type PushDirectoryToHubResp struct {
	HubDirectoryID  int64  `json:"hub_directory_id"`
	HubDirectoryURL string `json:"hub_directory_url"`
	DirectoryCount  int    `json:"directory_count"` // 包含的子目录数量
	FileCount       int    `json:"file_count"`      // 包含的文件数量
	OldVersion      string `json:"old_version"`     // 旧版本号
	NewVersion      string `json:"new_version"`     // 新版本号
}

// CreateServiceTreeReq 创建服务目录请求
type CreateServiceTreeReq struct {
	User        string `json:"user" binding:"required" example:"beiluo"` // 用户名
	App         string `json:"app" binding:"required" example:"myapp"`   // 应用名
	Name        string `json:"name" binding:"required" example:"用户管理"`   // 服务目录名称
	Code        string `json:"code" binding:"required" example:"user"`   // 服务目录代码
	ParentID    int64  `json:"parent_id" example:"0"`                    // 父目录ID，0表示根目录
	Description string `json:"description" example:"用户相关的API接口"`         // 描述
	Tags        string `json:"tags" example:"user,management"`           // 标签
	Admins      string `json:"admins" example:"user1,user2"`              // 管理员列表，逗号分隔的用户名
}

// CreateServiceTreeResp 创建服务目录响应
type CreateServiceTreeResp struct {
	ID           int64  `json:"id" example:"1"`                              // 服务目录ID
	Name         string `json:"name" example:"用户管理"`                         // 服务目录名称
	Code         string `json:"code" example:"user"`                         // 服务目录代码
	ParentID     int64  `json:"parent_id" example:"0"`                       // 父目录ID
	Type         string `json:"type" example:"package"`                      // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description  string `json:"description" example:"用户相关的API接口"`            // 描述
	Tags         string `json:"tags" example:"user,management"`              // 标签
	AppID        int64  `json:"app_id" example:"1"`                          // 应用ID
	RefID        int64  `json:"ref_id" example:"0"`                          // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	Version      string `json:"version" example:"v1"`                        // 节点当前版本号（如 v1, v2），package类型表示目录版本，function类型表示函数版本等
	VersionNum   int    `json:"version_num" example:"1"`                     // 节点当前版本号（数字部分）
	Status       string `json:"status" example:"enabled"`                    // 状态: enabled(启用), disabled(禁用)
}

// GetServiceTreeResp 获取服务目录响应
type GetServiceTreeResp struct {
	ID             int64                 `json:"id,omitempty" example:"1"`                              // 服务目录ID
	Name           string                `json:"name,omitempty" example:"用户管理"`                         // 服务目录名称
	Code           string                `json:"code,omitempty" example:"user"`                         // 服务目录代码
	ParentID       int64                 `json:"parent_id,omitempty" example:"0"`                       // 父目录ID
	Type           string                `json:"type,omitempty" example:"package"`                      // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description    string                `json:"description,omitempty" example:"用户相关的API接口"`            // 描述
	Tags           string                `json:"tags,omitempty" example:"user,management"`              // 标签
	Admins         string                `json:"admins,omitempty" example:"user1,user2"`                // 节点管理员列表，逗号分隔的用户名
	PendingCount   int                   `json:"pending_count,omitempty" example:"5"`                  // ⭐ 待审批的权限申请数量
	Owner          string                `json:"owner,omitempty" example:"user1"`                      // 节点创建者（owner）
	AppID          int64                 `json:"app_id,omitempty" example:"1"`                          // 应用ID
	RefID          int64                 `json:"ref_id,omitempty" example:"0"`                          // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	FullCodePath   string                `json:"full_code_path,omitempty" example:"/beiluo/myapp/user"` // 完整代码路径
	TemplateType   string                `json:"template_type,omitempty" example:"form"`                // 模板类型（函数的类型，如 form、table）
	Version        string                `json:"version,omitempty" example:"v1"`                        // 节点当前版本号（如 v1, v2），package类型表示目录版本，function类型表示函数版本等
	VersionNum     int                   `json:"version_num,omitempty" example:"1"`                     // 节点当前版本号（数字部分）
	HubDirectoryID int64                 `json:"hub_directory_id,omitempty" example:"0"`                // 关联的Hub目录ID（如果已发布到Hub）
	HubVersion     string                `json:"hub_version,omitempty" example:""`                      // Hub目录版本（如 v1.0.0），用于版本检测和升级
	HubVersionNum  int                   `json:"hub_version_num,omitempty" example:"0"`                 // Hub目录版本号（数字部分），用于版本比较
	HasFunction    bool                  `json:"has_function,omitempty" example:"true"`                 // ⭐ 是否有函数（仅对package类型有效）：如果该package下直接或间接包含function类型的子节点，则为true
	IsAdmin        bool                  `json:"is_admin,omitempty" example:"true"`                     // ⭐ 是否是管理员（企业版功能）：如果用户是工作空间管理员，则为 true，前端优先判断此字段，无需构造每个节点的权限
	Permissions    map[string]bool       `json:"permissions,omitempty"`                                 // ⭐ 权限信息（企业版功能）：权限点 -> 是否有权限
	Children       []*GetServiceTreeResp `json:"children,omitempty"`                                    // 子目录列表
}

// GetServiceTreeDetailReq 获取服务目录详情请求
type GetServiceTreeDetailReq struct {
	ID           int64  `json:"id" example:"1"`                                    // 服务目录ID（优先使用）
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user"`      // 完整代码路径（如果未提供ID则使用）
}

// GetServiceTreeDetailResp 获取服务目录详情响应
type GetServiceTreeDetailResp struct {
	ID             int64            `json:"id" example:"1"`                              // 服务目录ID
	Name           string            `json:"name" example:"用户管理"`                         // 服务目录名称
	Code           string            `json:"code" example:"user"`                         // 服务目录代码
	ParentID       int64             `json:"parent_id" example:"0"`                       // 父目录ID
	Type           string            `json:"type" example:"package"`                      // 节点类型: package(服务目录/包), function(函数/文件)
	Description    string            `json:"description" example:"用户相关的API接口"`            // 描述
	Tags           string            `json:"tags" example:"user,management"`              // 标签
	AppID          int64             `json:"app_id" example:"1"`                          // 应用ID
	RefID          int64             `json:"ref_id" example:"0"`                          // 引用ID
	FullCodePath   string            `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	TemplateType   string            `json:"template_type,omitempty" example:"form"`       // 模板类型（函数的类型，如 form、table）
	Version        string            `json:"version" example:"v1"`                         // 节点当前版本号
	VersionNum     int               `json:"version_num" example:"1"`                     // 节点当前版本号（数字部分）
	HubDirectoryID int64             `json:"hub_directory_id,omitempty" example:"0"`      // 关联的Hub目录ID
	HubVersion     string            `json:"hub_version,omitempty" example:""`            // Hub目录版本
	HubVersionNum  int               `json:"hub_version_num,omitempty" example:"0"`         // Hub目录版本号（数字部分）
	Permissions    map[string]bool   `json:"permissions,omitempty"`                       // ⭐ 权限标识（企业版功能）：权限点 -> 是否有权限
}

// GetPackageInfoReq 获取目录信息请求（仅用于获取目录权限，不包含函数）
type GetPackageInfoReq struct {
	ID           int64  `json:"id" form:"id" example:"1"`                                    // 目录ID（优先使用）
	FullCodePath string `json:"full_code_path" form:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径（如果未提供ID则使用）
}

// GetPackageInfoResp 获取目录信息响应（仅包含目录权限）
type GetPackageInfoResp struct {
	ID           int64           `json:"id" example:"1"`                              // 目录ID
	Name         string          `json:"name" example:"用户管理"`                         // 目录名称
	Code         string          `json:"code" example:"user"`                         // 目录代码
	FullCodePath string          `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	Permissions  map[string]bool `json:"permissions,omitempty"`                       // ⭐ 权限信息（企业版功能）：directory:read, directory:create, directory:update, directory:delete, directory:manage
}

// UpdateServiceTreeMetadataReq 更新服务目录元数据请求（旧接口，保留兼容性）
type UpdateServiceTreeMetadataReq struct {
	ID          int64  `json:"id" binding:"required" example:"1"` // 服务目录ID
	Name        string `json:"name" example:"用户管理"`               // 服务目录名称
	Code        string `json:"code" example:"user"`               // 服务目录代码
	Description string `json:"description" example:"用户相关的API接口"`  // 描述
	Tags        string `json:"tags" example:"user,management"`    // 标签
	Admins      string `json:"admins" example:"user1,user2"`      // 管理员列表，逗号分隔的用户名
}

// DeleteServiceTreeReq 删除服务目录请求
type DeleteServiceTreeReq struct {
	ID int64 `json:"id" binding:"required" example:"1"` // 服务目录ID
}

// BatchCreateDirectoryTreeReq 批量创建目录树请求
type BatchCreateDirectoryTreeReq struct {
	User  string               `json:"user" binding:"required"`  // 用户名
	App   string               `json:"app" binding:"required"`   // 应用名
	Items []*DirectoryTreeItem `json:"items" binding:"required"` // 目录树项列表
}

// BatchCreateDirectoryTreeResp 批量创建目录树响应
type BatchCreateDirectoryTreeResp struct {
	DirectoryCount int      `json:"directory_count"` // 创建的目录数量
	FileCount      int      `json:"file_count"`      // 创建的文件数量
	CreatedPaths   []string `json:"created_paths"`   // 创建的路径列表
}

// BatchWriteFilesReq 批量写文件请求
type BatchWriteFilesReq struct {
	User  string               `json:"user" binding:"required"`  // 用户名
	App   string               `json:"app" binding:"required"`   // 应用名
	Files []*DirectoryTreeItem `json:"files" binding:"required"` // 文件列表（只包含文件，不包含目录）
}

// BatchWriteFilesResp 批量写文件响应
type BatchWriteFilesResp struct {
	FileCount     int       `json:"file_count"`                // 写入的文件数量
	WrittenPaths  []string  `json:"written_paths"`             // 写入的文件路径列表
	Diff          *DiffData `json:"diff,omitempty"`            // API diff 信息（编译后）
	OldVersion    string    `json:"old_version"`               // 旧版本号
	NewVersion    string    `json:"new_version"`               // 新版本号
	GitCommitHash string    `json:"git_commit_hash,omitempty"` // Git 提交哈希
}

// PullDirectoryFromHubReq 从 Hub 拉取目录请求
type PullDirectoryFromHubReq struct {
	HubLink             string `json:"hub_link" binding:"required"`    // Hub 链接（如 hub://hub.example.com/123）
	TargetUser          string `json:"target_user" binding:"required"` // 目标用户
	TargetApp           string `json:"target_app" binding:"required"`  // 目标应用
	TargetDirectoryPath string `json:"target_directory_path"`          // 目标目录路径（可选，默认为应用根目录）
}

// PullDirectoryFromHubResp 从 Hub 拉取目录响应
type PullDirectoryFromHubResp struct {
	Message             string `json:"message"`               // 成功消息
	DirectoryCount      int    `json:"directory_count"`       // 安装的目录数量
	FileCount           int    `json:"file_count"`            // 安装的文件数量
	TargetDirectoryPath string `json:"target_directory_path"` // 目标目录路径
	ServiceTreeID       int64  `json:"service_tree_id"`       // 根目录的 ServiceTree ID
	HubDirectoryID      int64  `json:"hub_directory_id"`      // Hub 目录 ID
	HubDirectoryName    string `json:"hub_directory_name"`    // Hub 目录名称
	HubVersion          string `json:"hub_version"`           // Hub 目录版本（如 v1.0.0）
	HubVersionNum       int    `json:"hub_version_num"`       // Hub 目录版本号（数字部分）
}

// GetHubInfoReq 获取目录的 Hub 信息请求
type GetHubInfoReq struct {
	FullCodePath string `json:"full_code_path" form:"full_code_path" binding:"required"` // 目录完整路径
}

// GetHubInfoResp 获取目录的 Hub 信息响应
type GetHubInfoResp struct {
	HubDirectoryID  int64  `json:"hub_directory_id"`  // Hub 目录 ID
	HubDirectoryURL string `json:"hub_directory_url"` // Hub 目录 URL
	PublishedAt     string `json:"published_at"`      // 发布时间
}

// SearchFunctionsReq 搜索函数请求
type SearchFunctionsReq struct {
	User        string `json:"user" form:"user"`                    // 用户名（可选，用于过滤应用）
	App         string `json:"app" form:"app"`                      // 应用名（可选，用于过滤应用）
	Keyword     string `json:"keyword" form:"keyword"`              // 搜索关键词（可选，用于搜索名称和路径）
	TemplateType string `json:"template_type" form:"template_type"` // 模板类型过滤（可选，如：form、table、chart）
	Page        int    `json:"page" form:"page" binding:"required" example:"1"`        // 页码
	PageSize    int    `json:"page_size" form:"page_size" binding:"required" example:"10"` // 每页数量
}

// SearchFunctionsResp 搜索函数响应
type SearchFunctionsResp struct {
	Functions []*FunctionSearchResult `json:"functions"` // 函数列表
	Total     int64                   `json:"total"`     // 总数
	Page      int                     `json:"page"`      // 当前页码
	PageSize  int                     `json:"page_size"` // 每页数量
}

// FunctionSearchResult 函数搜索结果
type FunctionSearchResult struct {
	ID           int64  `json:"id" example:"1"`                              // 函数ID
	Name         string `json:"name" example:"表格解析"`                         // 函数名称
	Code         string `json:"code" example:"table_parse"`                   // 函数代码
	FullCodePath string `json:"full_code_path" example:"/system/official/agent/plugin/excel_or_csv/table_parse"` // 完整代码路径
	Description  string `json:"description" example:"解析Excel/CSV文件为Markdown表格"` // 函数描述
	TemplateType string `json:"template_type" example:"form"`                 // 模板类型（form、table、chart）
	AppID        int64  `json:"app_id" example:"1"`                            // 应用ID
	AppUser      string `json:"app_user" example:"system"`                     // 应用所属用户
	AppCode      string `json:"app_code" example:"official"`                   // 应用代码
}
