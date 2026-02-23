package repository

import (
	"context"
	"strings"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

// normalizeFullCodePath 规范化 full_code_path：与 app-server 一致，便于通过网关查询时能命中
// 去首尾空格、去尾斜杠、保证以单个 / 开头
func normalizeFullCodePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimSuffix(p, "/")
	if p != "" && !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

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
// 会先规范化路径（与 app-server 一致），再查询；若未命中则尝试无前导斜杠形式，兼容历史数据
func (r *HubDirectoryRepository) GetByFullCodePath(ctx context.Context, fullCodePath string) (*model.HubDirectory, error) {
	normalized := normalizeFullCodePath(fullCodePath)
	if normalized == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var directory model.HubDirectory
	err := r.db.Where("full_code_path = ?", normalized).Order("created_at DESC").First(&directory).Error
	if err == nil {
		return &directory, nil
	}
	// 兼容：可能历史数据存的是无前导斜杠（如 beiluo/app/xxx）
	alt := strings.TrimPrefix(normalized, "/")
	if alt != normalized {
		err = r.db.Where("full_code_path = ?", alt).Order("created_at DESC").First(&directory).Error
		if err == nil {
			return &directory, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// Update 更新目录
func (r *HubDirectoryRepository) Update(ctx context.Context, directory *model.HubDirectory) error {
	return r.db.Save(directory).Error
}

