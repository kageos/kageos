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
	Name        string `json:"name"`
	Code        string `json:"code"`
	ParentID    int64  `json:"parent_id" gorm:"default:0"`
	Type        string `json:"type"` // 节点类型: package(服务目录/包), function(函数/文件), api(API接口), service(服务), module(模块)
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags"`
	AppID       int64  `json:"app_id"`
	GroupCode   string `json:"group_code"`
	GroupName   string `json:"group_name"`
	RefID       int64  `json:"ref_id" gorm:"default:0"`                   // 引用ID：指向真实资源的ID，如果是package类型指向package的ID，如果是function类型指向function的ID
	App         *App   `json:"app" gorm:"foreignKey:AppID;references:ID"` // 预加载的完整应用对象
	//下面字段是数据库
	FullCodePath string         `json:"full_code_path"` // /$user/$app/tools/pdf 这种
	Children     []*ServiceTree `json:"children" gorm:"-"`
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
func (st *ServiceTree) GetAppPrefix() string {
	if st.App == nil {
		return ""
	}
	return fmt.Sprintf("/%s/%s", st.App.User, st.App.Name)
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
