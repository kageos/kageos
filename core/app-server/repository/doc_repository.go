package repository

import (
	"context"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

// DocRepository 文档数据访问层
type DocRepository struct {
	db *gorm.DB
}

// NewDocRepository 创建文档 Repository
func NewDocRepository(db *gorm.DB) *DocRepository {
	return &DocRepository{db: db}
}

// Create 创建文档
func (r *DocRepository) Create(ctx context.Context, doc *model.Docs) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

// GetByID 根据 ID 获取文档
func (r *DocRepository) GetByID(ctx context.Context, id int64) (*model.Docs, error) {
	var doc model.Docs
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetByTreeID 根据 TreeID 获取文档
func (r *DocRepository) GetByTreeID(ctx context.Context, treeID int64) (*model.Docs, error) {
	var doc model.Docs
	if err := r.db.WithContext(ctx).Where("tree_id = ?", treeID).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// Update 更新文档
func (r *DocRepository) Update(ctx context.Context, doc *model.Docs) error {
	return r.db.WithContext(ctx).Save(doc).Error
}

// Delete 删除文档
func (r *DocRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Docs{}).Error
}

// DeleteByTreeID 根据 TreeID 删除文档
func (r *DocRepository) DeleteByTreeID(ctx context.Context, treeID int64) error {
	return r.db.WithContext(ctx).Where("tree_id = ?", treeID).Delete(&model.Docs{}).Error
}

// ListByAppID 根据 AppID 获取文档列表
func (r *DocRepository) ListByAppID(ctx context.Context, appID int64, offset, limit int) ([]*model.Docs, int64, error) {
	var docs []*model.Docs
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Docs{}).Where("app_id = ?", appID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.Offset(offset).Limit(limit).Order("id DESC").Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}

// ListByTreeIDs 根据 TreeID 列表批量获取文档
func (r *DocRepository) ListByTreeIDs(ctx context.Context, treeIDs []int64) ([]*model.Docs, error) {
	var docs []*model.Docs
	if err := r.db.WithContext(ctx).Where("tree_id IN ?", treeIDs).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// GetByIDs 根据 ID 列表批量获取文档
func (r *DocRepository) GetByIDs(ctx context.Context, ids []int64) ([]*model.Docs, error) {
	if len(ids) == 0 {
		return []*model.Docs{}, nil
	}

	var docs []*model.Docs
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// GetByFullCodePaths 根据路径列表批量获取文档
// paths: 路径列表，如 ["/user/myapp/docs"]
// 直接根据 full_code_path 使用 IN 查询
func (r *DocRepository) GetByFullCodePaths(ctx context.Context, paths []string) ([]*model.Docs, error) {
	if len(paths) == 0 {
		return []*model.Docs{}, nil
	}

	var docs []*model.Docs
	if err := r.db.WithContext(ctx).Where("full_code_path IN ?", paths).Find(&docs).Error; err != nil {
		return nil, err
	}

	return docs, nil
}

// SearchDocs 搜索文档（支持按名称、路径搜索，支持跨应用搜索）
func (r *DocRepository) SearchDocs(ctx context.Context, keyword string, page, pageSize int) ([]*model.Docs, int64, error) {
	var docs []*model.Docs
	var total int64

	// 构建查询
	query := r.db.WithContext(ctx).Model(&model.Docs{})

	// 关键词搜索（名称或路径）
	if keyword != "" {
		keywordPattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR full_code_path LIKE ?", keywordPattern, keywordPattern)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}
