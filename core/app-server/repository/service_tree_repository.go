package repository

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

type ServiceTreeRepository struct {
	db *gorm.DB
}

func NewServiceTreeRepository(db *gorm.DB) *ServiceTreeRepository {
	return &ServiceTreeRepository{db: db}
}

// CreateServiceTree 创建服务目录
func (r *ServiceTreeRepository) CreateServiceTree(serviceTree *model.ServiceTree) error {
	// 先保存到数据库获取ID
	if err := r.db.Create(serviceTree).Error; err != nil {
		return err
	}

	// 然后计算路径信息
	if err := r.calculatePaths(serviceTree); err != nil {
		return fmt.Errorf("failed to calculate paths: %w", err)
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

// GetServiceTreeByAppAndName 根据应用ID和名称获取服务目录
func (r *ServiceTreeRepository) GetServiceTreeByAppAndName(appID int64, name string) (*model.ServiceTree, error) {
	var serviceTree model.ServiceTree
	err := r.db.Where("app_id = ? AND name = ?", name).First(&serviceTree).Error
	if err != nil {
		return nil, err
	}
	return &serviceTree, nil
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

	return rootNodes, nil
}

// UpdateServiceTree 更新服务目录
func (r *ServiceTreeRepository) UpdateServiceTree(serviceTree *model.ServiceTree) error {
	// 重新计算路径
	if err := r.calculatePaths(serviceTree); err != nil {
		return fmt.Errorf("failed to calculate paths: %w", err)
	}

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

// calculatePaths 计算路径信息
func (r *ServiceTreeRepository) calculatePaths(serviceTree *model.ServiceTree) error {
	// 计算 FullIDPath
	if serviceTree.ParentID == 0 {
		serviceTree.FullIDPath = fmt.Sprintf("/%d", serviceTree.ID)
		serviceTree.FullNamePath = fmt.Sprintf("/%s", serviceTree.Name)
	} else {
		// 获取父节点信息
		parent, err := r.GetServiceTreeByID(serviceTree.ParentID)
		if err != nil {
			return fmt.Errorf("failed to get parent service tree: %w", err)
		}

		serviceTree.FullIDPath = fmt.Sprintf("%s/%d", parent.FullIDPath, serviceTree.ID)
		serviceTree.FullNamePath = fmt.Sprintf("%s/%s", parent.FullNamePath, serviceTree.Name)
	}

	return nil
}

// GetServiceTreeByFullPath 根据完整路径获取服务目录
func (r *ServiceTreeRepository) GetServiceTreeByFullPath(appID int64, fullPath string) (*model.ServiceTree, error) {
	var serviceTree model.ServiceTree
	err := r.db.Where("app_id = ? AND full_name_path = ?", appID, fullPath).First(&serviceTree).Error
	if err != nil {
		return nil, err
	}
	return &serviceTree, nil
}

// CheckNameExists 检查名称是否已存在（在同一父目录下）
func (r *ServiceTreeRepository) CheckNameExists(appID int64, parentID int64, name string, excludeID int64) (bool, error) {
	var count int64
	query := r.db.Model(&model.ServiceTree{}).Where("app_id = ? AND parent_id = ? AND name = ?", appID, parentID, name)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
