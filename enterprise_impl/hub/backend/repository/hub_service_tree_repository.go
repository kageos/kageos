package repository

import (
	"context"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

// HubServiceTreeRepository Hub 服务树仓库
type HubServiceTreeRepository struct {
	db *gorm.DB
}

// NewHubServiceTreeRepository 创建 Hub 服务树仓库
func NewHubServiceTreeRepository(db *gorm.DB) *HubServiceTreeRepository {
	return &HubServiceTreeRepository{db: db}
}

// Create 创建服务树节点
func (r *HubServiceTreeRepository) Create(ctx context.Context, tree *model.HubServiceTree) error {
	return r.db.WithContext(ctx).Create(tree).Error
}

// CreateBatch 批量创建服务树节点
func (r *HubServiceTreeRepository) CreateBatch(ctx context.Context, trees []*model.HubServiceTree) error {
	if len(trees) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(trees, 100).Error
}

// GetByID 根据ID获取服务树节点
func (r *HubServiceTreeRepository) GetByID(ctx context.Context, id int64) (*model.HubServiceTree, error) {
	var tree model.HubServiceTree
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tree).Error
	if err != nil {
		return nil, err
	}
	return &tree, nil
}

// GetByHubDirectoryID 根据 Hub 目录ID获取所有服务树节点
func (r *HubServiceTreeRepository) GetByHubDirectoryID(ctx context.Context, hubDirectoryID int64) ([]*model.HubServiceTree, error) {
	var trees []*model.HubServiceTree
	err := r.db.WithContext(ctx).
		Where("hub_directory_id = ?", hubDirectoryID).
		Order("parent_id ASC, created_at ASC").
		Find(&trees).Error
	if err != nil {
		return nil, err
	}
	return trees, nil
}

// GetByParentID 根据父节点ID获取子节点
func (r *HubServiceTreeRepository) GetByParentID(ctx context.Context, parentID int64) ([]*model.HubServiceTree, error) {
	var trees []*model.HubServiceTree
	err := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("created_at ASC").
		Find(&trees).Error
	if err != nil {
		return nil, err
	}
	return trees, nil
}

// GetByType 根据类型获取服务树节点
func (r *HubServiceTreeRepository) GetByType(ctx context.Context, hubDirectoryID int64, nodeType string) ([]*model.HubServiceTree, error) {
	var trees []*model.HubServiceTree
	query := r.db.WithContext(ctx).Where("hub_directory_id = ?", hubDirectoryID)
	if nodeType != "" {
		query = query.Where("type = ?", nodeType)
	}
	err := query.Order("created_at ASC").Find(&trees).Error
	if err != nil {
		return nil, err
	}
	return trees, nil
}

// DeleteByHubDirectoryID 根据 Hub 目录ID删除所有服务树节点
func (r *HubServiceTreeRepository) DeleteByHubDirectoryID(ctx context.Context, hubDirectoryID int64) error {
	return r.db.WithContext(ctx).
		Where("hub_directory_id = ?", hubDirectoryID).
		Delete(&model.HubServiceTree{}).Error
}

// Update 更新服务树节点
func (r *HubServiceTreeRepository) Update(ctx context.Context, tree *model.HubServiceTree) error {
	return r.db.WithContext(ctx).Save(tree).Error
}

