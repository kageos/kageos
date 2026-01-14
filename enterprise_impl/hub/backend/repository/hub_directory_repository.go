package repository

import (
	"context"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

type HubDirectoryRepository struct {
	db *gorm.DB
}

func NewHubDirectoryRepository(db *gorm.DB) *HubDirectoryRepository {
	return &HubDirectoryRepository{db: db}
}

// Create 创建目录
func (r *HubDirectoryRepository) Create(ctx context.Context, directory *model.HubDirectory) error {
	return r.db.Create(directory).Error
}

// GetByID 根据ID获取目录
func (r *HubDirectoryRepository) GetByID(ctx context.Context, id int64) (*model.HubDirectory, error) {
	var directory model.HubDirectory
	err := r.db.Where("id = ?", id).First(&directory).Error
	if err != nil {
		return nil, err
	}
	return &directory, nil
}

// GetList 获取目录列表（分页）
func (r *HubDirectoryRepository) GetList(ctx context.Context, page, pageSize int, search, category, publisherUsername string) ([]*model.HubDirectory, int64, error) {
	var directories []*model.HubDirectory
	var total int64

	query := r.db.Model(&model.HubDirectory{})

	// 搜索条件
	if search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 分类筛选
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// 发布者筛选
	if publisherUsername != "" {
		query = query.Where("publisher_username = ?", publisherUsername)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&directories).Error
	if err != nil {
		return nil, 0, err
	}

	return directories, total, nil
}

// GetByPackagePath 根据 package_path 获取目录（用于检查是否已发布）
func (r *HubDirectoryRepository) GetByPackagePath(ctx context.Context, packagePath string) (*model.HubDirectory, error) {
	var directory model.HubDirectory
	err := r.db.Where("package_path = ?", packagePath).Order("created_at DESC").First(&directory).Error
	if err != nil {
		return nil, err
	}
	return &directory, nil
}

// GetByFullCodePath 根据 full_code_path 获取目录（用于通过 Hub 链接查询）
func (r *HubDirectoryRepository) GetByFullCodePath(ctx context.Context, fullCodePath string) (*model.HubDirectory, error) {
	var directory model.HubDirectory
	err := r.db.Where("full_code_path = ?", fullCodePath).Order("created_at DESC").First(&directory).Error
	if err != nil {
		return nil, err
	}
	return &directory, nil
}

// Update 更新目录
func (r *HubDirectoryRepository) Update(ctx context.Context, directory *model.HubDirectory) error {
	return r.db.Save(directory).Error
}

