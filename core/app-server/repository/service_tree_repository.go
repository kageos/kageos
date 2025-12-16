package repository

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

type ServiceTreeRepository struct {
	db *gorm.DB
}

func NewServiceTreeRepository(db *gorm.DB) *ServiceTreeRepository {
	return &ServiceTreeRepository{db: db}
}

// CreateServiceTreeWithParentPath 创建服务目录
func (r *ServiceTreeRepository) CreateServiceTreeWithParentPath(serviceTree *model.ServiceTree, parentFullIDPath string) error {
	// 直接创建，不再计算FullIDPath
	return r.db.Create(serviceTree).Error
}

// CreateServiceTreeWithAppPrefix 创建带有用户应用前缀的服务目录
func (r *ServiceTreeRepository) CreateServiceTreeWithAppPrefix(serviceTree *model.ServiceTree, userAppPrefix string) error {
	// 先保存到数据库获取ID
	if err := r.db.Create(serviceTree).Error; err != nil {
		return err
	}

	// 然后计算包含用户应用前缀的路径信息
	if err := r.calculatePathsWithAppPrefix(serviceTree, userAppPrefix); err != nil {
		return fmt.Errorf("failed to calculate paths with app prefix: %w", err)
	}

	// 更新路径信息到数据库
	return r.db.Save(serviceTree).Error
}

// GetServiceTreeByID 根据ID获取服务目录
func (r *ServiceTreeRepository) GetServiceTreeByID(id int64) (*model.ServiceTree, error) {
	var serviceTree model.ServiceTree
	err := r.db.Where("id = ?", id).First(&serviceTree).Error
	if err != nil {
		return nil, err
	}
	return &serviceTree, nil
}

// GetServiceTreesByAppID 根据应用ID获取所有服务目录
func (r *ServiceTreeRepository) GetServiceTreesByAppID(appID int64) ([]*model.ServiceTree, error) {
	var serviceTrees []*model.ServiceTree
	err := r.db.Where("app_id = ?", appID).Order("created_at ASC").Find(&serviceTrees).Error
	if err != nil {
		return nil, err
	}
	return serviceTrees, nil
}

// GetServiceTreesByAppIDAndType 根据应用ID和类型获取服务目录
func (r *ServiceTreeRepository) GetServiceTreesByAppIDAndType(appID int64, nodeType string) ([]*model.ServiceTree, error) {
	var serviceTrees []*model.ServiceTree
	query := r.db.Where("app_id = ?", appID)
	if nodeType != "" {
		query = query.Where("type = ?", nodeType)
	}
	err := query.Order("created_at ASC").Find(&serviceTrees).Error
	if err != nil {
		return nil, err
	}
	return serviceTrees, nil
}

// GetServiceTreeChildren 获取子服务目录
func (r *ServiceTreeRepository) GetServiceTreeChildren(parentID int64) ([]*model.ServiceTree, error) {
	var children []*model.ServiceTree
	err := r.db.Where("parent_id = ?", parentID).Order("created_at ASC").Find(&children).Error
	if err != nil {
		return nil, err
	}
	return children, nil
}

// BuildServiceTree 构建树形结构
func (r *ServiceTreeRepository) BuildServiceTree(appID int64) ([]*model.ServiceTree, error) {
	// 获取所有服务目录
	allTrees, err := r.GetServiceTreesByAppID(appID)
	if err != nil {
		return nil, err
	}
	return r.buildTreeFromNodes(allTrees), nil
}

// BuildServiceTreeByType 根据类型构建树形结构
func (r *ServiceTreeRepository) BuildServiceTreeByType(appID int64, nodeType string) ([]*model.ServiceTree, error) {
	// 获取指定类型的服务目录
	allTrees, err := r.GetServiceTreesByAppIDAndType(appID, nodeType)
	if err != nil {
		return nil, err
	}
	return r.buildTreeFromNodes(allTrees), nil
}

// BuildServiceTreeByVersion 根据版本号构建树形结构（用于版本回滚）
// versionNum: 目标版本号数字（如 19），只返回 add_version_num <= versionNum 且 (update_version_num = 0 或 update_version_num <= versionNum) 的节点
func (r *ServiceTreeRepository) BuildServiceTreeByVersion(appID int64, versionNum int) ([]*model.ServiceTree, error) {
	// 查询符合条件的节点：add_version_num <= versionNum 且 (update_version_num = 0 或 update_version_num <= versionNum)
	var allTrees []*model.ServiceTree
	err := r.db.Where("app_id = ? AND add_version_num <= ? AND (update_version_num = 0 OR update_version_num <= ?)",
		appID, versionNum, versionNum).
		Order("created_at ASC").
		Find(&allTrees).Error
	if err != nil {
		return nil, err
	}
	return r.buildTreeFromNodes(allTrees), nil
}

// buildTreeFromNodes 从节点列表构建树形结构（内部方法）
func (r *ServiceTreeRepository) buildTreeFromNodes(allTrees []*model.ServiceTree) []*model.ServiceTree {

	// 构建树形结构
	treeMap := make(map[int64]*model.ServiceTree)
	var rootNodes []*model.ServiceTree

	// 先创建所有节点的映射
	for _, tree := range allTrees {
		treeMap[tree.ID] = tree
	}

	// 构建父子关系
	for _, tree := range allTrees {
		if tree.ParentID == 0 {
			// 根节点
			rootNodes = append(rootNodes, tree)
		} else {
			// 子节点
			if parent, exists := treeMap[tree.ParentID]; exists {
				parent.Children = append(parent.Children, tree)
			}
		}
	}

	return rootNodes
}

// UpdateServiceTree 更新服务目录
func (r *ServiceTreeRepository) UpdateServiceTree(serviceTree *model.ServiceTree) error {
	return r.db.Save(serviceTree).Error
}

// DeleteServiceTree 删除服务目录（级联删除子目录）
func (r *ServiceTreeRepository) DeleteServiceTree(id int64) error {
	// 先删除所有子目录
	children, err := r.GetServiceTreeChildren(id)
	if err != nil {
		return err
	}

	for _, child := range children {
		if err := r.DeleteServiceTree(child.ID); err != nil {
			return err
		}
	}

	// 删除当前目录
	return r.db.Delete(&model.ServiceTree{}, id).Error
}

// calculatePathsWithAppPrefix 计算带有用户应用前缀的路径信息
func (r *ServiceTreeRepository) calculatePathsWithAppPrefix(serviceTree *model.ServiceTree, userAppPrefix string) error {
	// FullCodePath使用预加载的app信息计算
	if serviceTree.App != nil {
		// 使用预加载的App对象构建路径
		appPrefix := fmt.Sprintf("/%s/%s", serviceTree.App.User, serviceTree.App.Code)
		serviceTree.FullCodePath = fmt.Sprintf("%s/%s", appPrefix, serviceTree.Code)
	} else {
		// 回退到传入的用户应用前缀
		serviceTree.FullCodePath = fmt.Sprintf("%s/%s", userAppPrefix, serviceTree.Code)
	}

	return nil
}

// GetServiceTreeByFullPath 根据完整路径获取服务目录（full_code_path全局唯一）
func (r *ServiceTreeRepository) GetServiceTreeByFullPath(fullPath string) (*model.ServiceTree, error) {
	var serviceTree model.ServiceTree
	err := r.db.Where("full_code_path = ?", fullPath).First(&serviceTree).Error
	if err != nil {
		return nil, err
	}
	return &serviceTree, nil
}

// GetServiceTreeByFullPaths 批量根据完整路径获取服务目录
func (r *ServiceTreeRepository) GetServiceTreeByFullPaths(fullPaths []string) (map[string]*model.ServiceTree, error) {
	if len(fullPaths) == 0 {
		return make(map[string]*model.ServiceTree), nil
	}

	var serviceTrees []*model.ServiceTree
	err := r.db.Where("full_code_path IN ?", fullPaths).Find(&serviceTrees).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.ServiceTree)
	for _, tree := range serviceTrees {
		result[tree.FullCodePath] = tree
	}
	return result, nil
}

// CheckNameExists 检查名称是否已存在（在同一父目录下和同一应用内）
func (r *ServiceTreeRepository) CheckNameExists(parentID int64, code string, appID int64) (bool, error) {
	var count int64
	query := r.db.Model(&model.ServiceTree{}).Where("parent_id = ? AND code = ? AND app_id = ?", parentID, code, appID)

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *ServiceTreeRepository) GetByID(parentId int64) (*model.ServiceTree, error) {
	var tree model.ServiceTree
	err := r.db.Model(&model.ServiceTree{}).Where("id = ?", parentId).First(&tree).Error
	if err != nil {
		return nil, err
	}
	return &tree, nil
}

// GetServiceTreesByFullGroupCode 根据完整函数组代码获取服务目录列表（同一个函数组下可能有多个函数，每个函数一条记录）
func (r *ServiceTreeRepository) GetServiceTreesByFullGroupCode(fullGroupCode string) ([]*model.ServiceTree, error) {
	var serviceTrees []*model.ServiceTree
	err := r.db.Where("full_group_code = ? AND type = ?", fullGroupCode, model.ServiceTreeTypeFunction).
		Find(&serviceTrees).Error
	if err != nil {
		return nil, err
	}
	return serviceTrees, nil
}

// GetDescendantDirectories 递归获取所有子目录（包括嵌套）
// 使用路径前缀匹配，一次查询获取所有子目录
func (r *ServiceTreeRepository) GetDescendantDirectories(appID int64, parentFullCodePath string) ([]*model.ServiceTree, error) {
	// 标准化路径（确保以 / 结尾，用于前缀匹配）
	normalizedPath := strings.TrimSuffix(parentFullCodePath, "/") + "/"
	
	var descendants []*model.ServiceTree
	err := r.db.Where("app_id = ? AND full_code_path LIKE ? AND type = ?",
		appID, normalizedPath+"%", model.ServiceTreeTypePackage).
		Order("full_code_path ASC").
		Find(&descendants).Error
	
	if err != nil {
		return nil, err
	}
	
	// 过滤：只返回真正的子目录（路径必须以 parentFullCodePath/ 开头）
	result := make([]*model.ServiceTree, 0)
	for _, dir := range descendants {
		if strings.HasPrefix(dir.FullCodePath, normalizedPath) {
			result = append(result, dir)
		}
	}
	
	return result, nil
}
