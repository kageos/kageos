package model

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

const (
	ServiceTreeTypePackage  = "package"
	ServiceTreeTypeFunction = "function"
)

// ServiceTree 表示服务树模型，一个app下可以有无数个package，一个package下面有无数个function，ServiceTree是一个抽象的树干，这个树干上可以挂载各种实体
// 例如我有个tools的app，然后，我有个excel的package（目录对应go的package），然后下面有多个function（go文件）
type ServiceTree struct {
	models.Base
	Name          string `json:"name"`
	Code          string `json:"code"`
	ParentID      int64  `json:"parent_id" gorm:"default:0"`
	Type          string `json:"type"` // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description   string `json:"description,omitempty"`
	Tags          string `json:"tags"`
	Admins        string `json:"admins" gorm:"type:varchar(150);comment:节点管理员列表，逗号分隔的用户名（如 user1,user2,user3）"` // 节点管理员列表
	PendingCount  int    `json:"pending_count" gorm:"default:0;comment:待审批的权限申请数量"` // ⭐ 待审批的权限申请数量
	AppID         int64  `json:"app_id"`
	// FullGroupCode 和 GroupName 已移除，不再需要
	RefID         int64  `json:"ref_id" gorm:"default:0"`                   // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	App           *App   `json:"app" gorm:"foreignKey:AppID;references:ID"` // 预加载的完整应用对象
	TemplateType  string `json:"template_type"`                             //函数的类型
	//下面字段是数据库
	FullCodePath     string         `json:"full_code_path"`                                                                              // /$user/$app/plugins/pdf 这种
	AddVersionNum    int            `json:"add_version_num"`                                                                             // 添加版本号（数字部分，如 v1 -> 1），用于版本回滚时过滤
	UpdateVersionNum int            `json:"update_version_num"`                                                                          // 更新版本号（数字部分，如 v2 -> 2），用于版本回滚时过滤
	Version          string         `json:"version" gorm:"type:varchar(50);comment:节点当前版本号（如 v1, v2），package类型表示目录版本，function类型表示函数版本等"` // 节点当前版本号
	VersionNum       int            `json:"version_num" gorm:"comment:节点当前版本号（数字部分）"`                                                    // 节点当前版本号（数字部分）
	HubDirectoryID   int64          `json:"hub_directory_id" gorm:"index;default:0;comment:关联的Hub目录ID（如果已发布到Hub）"` // 关联的Hub目录ID
	HubVersion       string         `json:"hub_version" gorm:"type:varchar(50);default:'';comment:Hub目录版本（如 v1.0.0），用于版本检测和升级"` // Hub目录版本
	HubVersionNum    int            `json:"hub_version_num" gorm:"default:0;comment:Hub目录版本号（数字部分），用于版本比较"` // Hub目录版本号（数字部分）
	Children         []*ServiceTree `json:"children" gorm:"-"`
}

// TableName 指定表名
func (*ServiceTree) TableName() string {
	return "service_tree"
}

// IsPackage 判断是否为package类型
func (st *ServiceTree) IsPackage() bool {
	return st.Type == ServiceTreeTypePackage
}

// IsFunction 判断是否为function类型
func (st *ServiceTree) IsFunction() bool {
	return st.Type == ServiceTreeTypeFunction
}

// HasRefID 判断是否有引用ID
func (st *ServiceTree) HasRefID() bool {
	return st.RefID > 0
}

// GetRefID 获取引用ID
func (st *ServiceTree) GetRefID() int64 {
	return st.RefID
}

// SetRefID 设置引用ID
func (st *ServiceTree) SetRefID(refID int64) {
	st.RefID = refID
}

// IsRoot 判断是否为根节点
func (st *ServiceTree) IsRoot() bool {
	return st.ParentID == 0
}

// GetDepth 获取节点深度（基于ParentID，根节点深度为0）
func (st *ServiceTree) GetDepth() int {
	if st.IsRoot() {
		return 0
	}
	// 深度基于ParentID，如果需要精确深度，需要通过递归查询父节点
	// 这里简化处理，根节点为0，非根节点为1（可根据实际需求调整）
	return 1
}

// IsLeaf 判断是否为叶子节点（没有子节点）
func (st *ServiceTree) IsLeaf() bool {
	return len(st.Children) == 0
}

// GetParentPath 获取父节点路径
func (st *ServiceTree) GetParentPath() string {
	if st.IsRoot() {
		return ""
	}

	pathParts := strings.Split(st.FullCodePath, "/")
	if len(pathParts) > 1 {
		return strings.Join(pathParts[:len(pathParts)-1], "/")
	}
	return ""
}

// GetBreadcrumbs 获取面包屑路径
func (st *ServiceTree) GetBreadcrumbs() []string {
	pathParts := strings.Split(st.FullCodePath, "/")
	if len(pathParts) > 0 && pathParts[0] == "" {
		pathParts = pathParts[1:] // 移除开头的空字符串
	}
	return pathParts
}

// GetAppPrefix 获取应用前缀路径
// 注意：使用 App.Code 而不是 App.Name，因为 FullCodePath 是基于 Code 构建的
func (st *ServiceTree) GetAppPrefix() string {
	if st.App == nil {
		return ""
	}
	return fmt.Sprintf("/%s/%s", st.App.User, st.App.Code)
}

// GetBasePath 获取基础路径（不含用户应用前缀的部分）
func (st *ServiceTree) GetBasePath() string {
	fullPath := st.FullCodePath
	prefix := st.GetAppPrefix()

	if prefix != "" && strings.HasPrefix(fullPath, prefix) {
		return strings.TrimPrefix(fullPath, prefix)
	}
	return fullPath
}

// BuildFullPath 构建完整路径（基于预加载的app信息）
func (st *ServiceTree) BuildFullPath(nodeName string) string {
	if st.IsRoot() {
		// 根节点，添加应用前缀
		prefix := st.GetAppPrefix()
		if prefix != "" {
			return fmt.Sprintf("%s/%s", prefix, nodeName)
		}
		return fmt.Sprintf("/%s", nodeName)
	}

	// 子节点，在父节点基础上添加
	parentPath := st.FullCodePath
	return fmt.Sprintf("%s/%s", parentPath, nodeName)
}

// GetParentFullPath 获取父节点的完整路径
func (st *ServiceTree) GetParentFullPath() string {
	if st.IsRoot() {
		return ""
	}

	pathParts := strings.Split(strings.Trim(st.FullCodePath, "/"), "/")
	if len(pathParts) <= 1 {
		return ""
	}

	parentParts := pathParts[:len(pathParts)-1]
	return "/" + strings.Join(parentParts, "/")
}

// GetNodeName 获取当前节点名称（路径的最后一部分）
func (st *ServiceTree) GetNodeName() string {
	pathParts := strings.Split(strings.Trim(st.FullCodePath, "/"), "/")
	if len(pathParts) == 0 {
		return ""
	}
	return pathParts[len(pathParts)-1]
}

// GetLevel 获取节点层级（根节点为0，基于ParentID）
func (st *ServiceTree) GetLevel() int {
	if st.IsRoot() {
		return 0
	}
	// 层级基于ParentID，如果需要精确层级，需要通过递归查询父节点
	// 这里简化处理，根节点为0，非根节点为1（可根据实际需求调整）
	return 1
}

// HasChildren 判断是否有子节点
func (st *ServiceTree) HasChildren() bool {
	return len(st.Children) > 0
}

// GetPackageChain 获取完整的包链（从根到当前节点的所有package名称）
func (st *ServiceTree) GetPackageChain() []string {
	fullPath := st.GetBasePath()
	if fullPath == "" || fullPath == "/" {
		return []string{}
	}

	parts := strings.Split(strings.Trim(fullPath, "/"), "/")
	if len(parts) == 0 {
		return []string{}
	}

	// 如果是function节点，排除最后一个function名称，只返回package链
	if st.IsFunction() && len(parts) > 1 {
		return parts[:len(parts)-1]
	}

	return parts
}

// GetPackagePath 获取package路径（不包含function名称）
func (st *ServiceTree) GetPackagePath() string {
	if st.IsPackage() {
		return st.FullCodePath
	}

	// 对于function节点，返回其所在的package路径
	parentPath := st.GetParentFullPath()
	if parentPath == "" {
		return st.GetAppPrefix()
	}
	return parentPath
}

// GetRelativePath 获取相对于应用根目录的路径
func (st *ServiceTree) GetRelativePath() string {
	appPrefix := st.GetAppPrefix()
	if appPrefix == "" {
		return st.GetBasePath()
	}

	if strings.HasPrefix(st.FullCodePath, appPrefix) {
		return strings.TrimPrefix(st.FullCodePath, appPrefix)
	}

	return st.GetBasePath()
}

// IsInPackage 判断是否在指定的package中
func (st *ServiceTree) IsInPackage(packagePath string) bool {
	if packagePath == "" || packagePath == "/" {
		return st.IsRoot()
	}

	// 标准化路径
	packagePath = strings.TrimSuffix(packagePath, "/")
	currentPath := strings.TrimSuffix(st.FullCodePath, "/")

	if packagePath == currentPath {
		return true
	}

	return strings.HasPrefix(currentPath, packagePath+"/")
}

// GetHierarchyType 获取层级类型
func (st *ServiceTree) GetHierarchyType() string {
	switch st.Type {
	case ServiceTreeTypePackage:
		if st.IsRoot() {
			return "root"
		}
		return "package"
	case ServiceTreeTypeFunction:
		return "function"
	default:
		return "unknown"
	}
}

// GetDisplayName 获取显示名称（优先使用Name，回退到Code）
func (st *ServiceTree) GetDisplayName() string {
	if st.Name != "" {
		return st.Name
	}
	return st.Code
}

// GetPackagePathForFileCreation 获取用于文件创建的 package 路径
// 封装路径操作，方便维护
// 对于 package 类型的节点，返回其路径（去掉应用前缀，去掉开头的斜杠）
// 对于 function 类型的节点，返回其所在的 package 路径
// 返回格式：例如 "crm" 或 "plugins/cashier"，不包含 user 和 app 名称
func (st *ServiceTree) GetPackagePathForFileCreation() string {
	// 获取应用前缀，例如 "/luobei/demo"
	appPrefix := st.GetAppPrefix()

	// 从 FullCodePath 中去掉应用前缀
	// FullCodePath 格式："/luobei/demo/crm" 或 "/luobei/demo/plugins/cashier"
	// 去掉前缀后："/crm" 或 "/plugins/cashier"
	fullPath := st.FullCodePath
	if appPrefix != "" && strings.HasPrefix(fullPath, appPrefix) {
		fullPath = strings.TrimPrefix(fullPath, appPrefix)
	}

	// 去掉开头的斜杠，得到 "crm" 或 "plugins/cashier"
	basePath := strings.TrimPrefix(fullPath, "/")

	var packagePath string
	if st.IsFunction() {
		// 对于 function 节点，需要获取其所在的 package 路径
		// 去掉路径的最后一部分（function 名称）
		pathParts := strings.Split(basePath, "/")
		if len(pathParts) > 1 {
			packagePath = strings.Join(pathParts[:len(pathParts)-1], "/")
		} else {
			// 如果只有一层，说明 function 直接在应用根目录下，package 为空
			packagePath = ""
		}
	} else {
		// 对于 package 节点，直接使用 basePath
		packagePath = basePath
	}

	if packagePath == "" {
		// 如果是根节点或 function 直接在应用根目录下，使用 code 作为 package
		packagePath = st.Code
	}

	return packagePath
}

// GetTagsSlice 将Tags字符串转换为切片
func (st *ServiceTree) GetTagsSlice() []string {
	if st.Tags == "" {
		return []string{}
	}

	tags := strings.Split(st.Tags, ",")
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result
}

// SetTagsSlice 将标签切片设置到Tags字段
func (st *ServiceTree) SetTagsSlice(tags []string) {
	if len(tags) == 0 {
		st.Tags = ""
		return
	}

	cleanTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	st.Tags = strings.Join(cleanTags, ",")
}

// GetPathSegments 获取路径段（解析后的路径部分）
func (st *ServiceTree) GetPathSegments() []string {
	path := strings.Trim(st.FullCodePath, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// GetFunctionPath 获取function的完整路径（仅对function节点有效）
func (st *ServiceTree) GetFunctionPath() string {
	if !st.IsFunction() {
		return ""
	}
	return st.FullCodePath
}

// IsDescendantOf 判断是否为指定节点的后代
func (st *ServiceTree) IsDescendantOf(ancestor *ServiceTree) bool {
	if ancestor == nil {
		return false
	}

	// 如果路径完全相同，则是同一个节点
	if st.FullCodePath == ancestor.FullCodePath {
		return false
	}

	return strings.HasPrefix(st.FullCodePath, ancestor.FullCodePath+"/")
}

// GetCommonAncestorPath 获取与另一个节点的共同祖先路径
func (st *ServiceTree) GetCommonAncestorPath(other *ServiceTree) string {
	if other == nil {
		return ""
	}

	myParts := st.GetPathSegments()
	otherParts := other.GetPathSegments()

	maxLen := len(myParts)
	if len(otherParts) < maxLen {
		maxLen = len(otherParts)
	}

	commonLength := 0
	for i := 0; i < maxLen; i++ {
		if myParts[i] == otherParts[i] {
			commonLength++
		} else {
			break
		}
	}

	if commonLength == 0 {
		return ""
	}

	commonParts := myParts[:commonLength]
	return "/" + strings.Join(commonParts, "/")
}
