package dto

// 注意：DiffData 定义在 dto/app_runtime_namespace.go 中

// PublishDirectoryToHubReq 发布目录到 Hub 请求
type PublishDirectoryToHubReq struct {
	SourceUser           string   `json:"source_user" binding:"required"`            // 源用户
	SourceApp            string   `json:"source_app" binding:"required"`             // 源应用
	SourceDirectoryPath  string   `json:"source_directory_path" binding:"required"`  // 源目录完整路径
	Name                 string   `json:"name" binding:"required"`                   // 目录名称
	Description          string   `json:"description"`                                // 目录描述
	Category             string   `json:"category"`                                   // 分类
	Tags                 []string `json:"tags"`                                      // 标签
	ServiceFeePersonal   float64  `json:"service_fee_personal"`                      // 个人用户服务费
	ServiceFeeEnterprise float64  `json:"service_fee_enterprise"`                   // 企业用户服务费
}

// PublishDirectoryToHubResp 发布目录到 Hub 响应
type PublishDirectoryToHubResp struct {
	HubDirectoryID  int64  `json:"hub_directory_id"`
	HubDirectoryURL string `json:"hub_directory_url"`
	DirectoryCount  int    `json:"directory_count"` // 包含的子目录数量
	FileCount       int    `json:"file_count"`      // 包含的文件数量
}

// CreateServiceTreeReq 创建服务目录请求
type CreateServiceTreeReq struct {
	User        string `json:"user" binding:"required" example:"beiluo"`   // 用户名
	App         string `json:"app" binding:"required" example:"myapp"`     // 应用名
	Name        string `json:"name" binding:"required" example:"用户管理"` // 服务目录名称
	Code        string `json:"code" binding:"required" example:"user"`     // 服务目录代码
	ParentID    int64  `json:"parent_id" example:"0"`                      // 父目录ID，0表示根目录
	Description string `json:"description" example:"用户相关的API接口"`    // 描述
	Tags        string `json:"tags" example:"user,management"`             // 标签
}

// CreateServiceTreeResp 创建服务目录响应
type CreateServiceTreeResp struct {
	ID           int64  `json:"id" example:"1"`                              // 服务目录ID
	Name         string `json:"name" example:"用户管理"`                     // 服务目录名称
	Code         string `json:"code" example:"user"`                         // 服务目录代码
	ParentID     int64  `json:"parent_id" example:"0"`                       // 父目录ID
	Type         string `json:"type" example:"package"`                      // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description  string `json:"description" example:"用户相关的API接口"`     // 描述
	Tags         string `json:"tags" example:"user,management"`              // 标签
	AppID        int64  `json:"app_id" example:"1"`                          // 应用ID
	RefID        int64  `json:"ref_id" example:"0"`                          // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	FullCodePath string `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	Version      string `json:"version" example:"v1"`                         // 节点当前版本号（如 v1, v2），package类型表示目录版本，function类型表示函数版本等
	VersionNum   int    `json:"version_num" example:"1"`                      // 节点当前版本号（数字部分）
	Status       string `json:"status" example:"enabled"`                    // 状态: enabled(启用), disabled(禁用)
}

// GetServiceTreeResp 获取服务目录响应
type GetServiceTreeResp struct {
	ID           int64                 `json:"id" example:"1"`                              // 服务目录ID
	Name         string                `json:"name" example:"用户管理"`                     // 服务目录名称
	Code         string                `json:"code" example:"user"`                         // 服务目录代码
	ParentID     int64                 `json:"parent_id" example:"0"`                       // 父目录ID
	Type         string                `json:"type" example:"package"`                      // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description  string                `json:"description" example:"用户相关的API接口"`     // 描述
	FullGroupCode string                `json:"full_group_code"`                             //完整函数组代码：{full_path}/{group_code}，与 source_code.full_group_code 对齐
	GroupName     string                `json:"group_name"`                                  //组名称
	Tags         string                `json:"tags" example:"user,management"`              // 标签
	AppID        int64                 `json:"app_id" example:"1"`                          // 应用ID
	RefID        int64                 `json:"ref_id" example:"0"`                          // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	FullCodePath string                `json:"full_code_path" example:"/beiluo/myapp/user"` // 完整代码路径
	TemplateType string                `json:"template_type" example:"form"`                 // 模板类型（函数的类型，如 form、table）
	Version      string                `json:"version" example:"v1"`                         // 节点当前版本号（如 v1, v2），package类型表示目录版本，function类型表示函数版本等
	VersionNum   int                   `json:"version_num" example:"1"`                      // 节点当前版本号（数字部分）
	Children     []*GetServiceTreeResp `json:"children,omitempty"`                          // 子目录列表
}

// UpdateServiceTreeMetadataReq 更新服务目录元数据请求（旧接口，保留兼容性）
type UpdateServiceTreeMetadataReq struct {
	ID          int64  `json:"id" binding:"required" example:"1"`       // 服务目录ID
	Name        string `json:"name" example:"用户管理"`                 // 服务目录名称
	Code        string `json:"code" example:"user"`                     // 服务目录代码
	Description string `json:"description" example:"用户相关的API接口"` // 描述
	Tags        string `json:"tags" example:"user,management"`          // 标签
}

// DeleteServiceTreeReq 删除服务目录请求
type DeleteServiceTreeReq struct {
	ID int64 `json:"id" binding:"required" example:"1"` // 服务目录ID
}

// BatchCreateDirectoryTreeReq 批量创建目录树请求
type BatchCreateDirectoryTreeReq struct {
	User  string                `json:"user" binding:"required"` // 用户名
	App   string                `json:"app" binding:"required"`  // 应用名
	Items []*DirectoryTreeItem  `json:"items" binding:"required"` // 目录树项列表
}

// BatchCreateDirectoryTreeResp 批量创建目录树响应
type BatchCreateDirectoryTreeResp struct {
	DirectoryCount int      `json:"directory_count"` // 创建的目录数量
	FileCount      int      `json:"file_count"`      // 创建的文件数量
	CreatedPaths   []string `json:"created_paths"`   // 创建的路径列表
}

// UpdateServiceTreeReq 更新服务树请求
type UpdateServiceTreeReq struct {
	User  string             `json:"user" binding:"required"` // 用户名
	App   string             `json:"app" binding:"required"`  // 应用名
	Nodes []*ServiceTreeNode `json:"nodes" binding:"required"` // 服务树节点列表
}

// UpdateServiceTreeResp 更新服务树响应
type UpdateServiceTreeResp struct {
	DirectoryCount int      `json:"directory_count"` // 处理的目录数量
	FileCount      int      `json:"file_count"`      // 处理的文件数量
	Diff           *DiffData `json:"diff,omitempty"`  // API diff 信息（如果有文件变更）
	OldVersion     string   `json:"old_version"`     // 旧版本号
	NewVersion     string   `json:"new_version"`     // 新版本号
	GitCommitHash  string   `json:"git_commit_hash,omitempty"` // Git 提交哈希
}

// BatchWriteFilesReq 批量写文件请求
type BatchWriteFilesReq struct {
	User  string                `json:"user" binding:"required"` // 用户名
	App   string                `json:"app" binding:"required"`  // 应用名
	Files []*DirectoryTreeItem  `json:"files" binding:"required"` // 文件列表（只包含文件，不包含目录）
}

// BatchWriteFilesResp 批量写文件响应
type BatchWriteFilesResp struct {
	FileCount     int      `json:"file_count"`      // 写入的文件数量
	WrittenPaths  []string `json:"written_paths"`   // 写入的文件路径列表
	Diff          *DiffData `json:"diff,omitempty"`  // API diff 信息（编译后）
	OldVersion    string   `json:"old_version"`     // 旧版本号
	NewVersion    string   `json:"new_version"`     // 新版本号
	GitCommitHash string   `json:"git_commit_hash,omitempty"` // Git 提交哈希
}
