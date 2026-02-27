package repository

import (
	"context"
	"strings"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

// listStatusActive 列表只展示未删除的（空或 active 均视为在架）
func listStatusCondition(db *gorm.DB) *gorm.DB {
	return db.Where("(status = ? OR status = '' OR status IS NULL)", model.HubDirectoryStatusActive)
}

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

// splitSearchKeywords 将 search 按 | 拆成多个关键字，去空格、去空串
func splitSearchKeywords(search string) []string {
	var out []string
	for _, part := range strings.Split(search, "|") {
		kw := strings.TrimSpace(part)
		if kw != "" {
			out = append(out, kw)
		}
	}
	return out
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

// GetList 获取目录列表（分页）；仅返回在架应用，已删除的不展示
// feeType: 空=全部，free=免费，paid=收费
// orderBy: 空或 latest=最新(created_at DESC)，hot=热门(star_count DESC, download_count DESC, created_at DESC)
// search: 支持多关键字 OR 搜索，用 | 分隔，例如 "美发|理发|美容|预约"；每个关键字匹配 name、description、tags（任意一个匹配即命中）
func (r *HubDirectoryRepository) GetList(ctx context.Context, page, pageSize int, search, category, publisherUsername, feeType, orderBy string) ([]*model.HubDirectory, int64, error) {
	var directories []*model.HubDirectory
	var total int64

	query := r.db.Model(&model.HubDirectory{})
	query = listStatusCondition(query)

	// 搜索条件：支持多关键字 | 分隔，OR 逻辑；每个关键字在 name、description、tags 中匹配
	if search != "" {
		keywords := splitSearchKeywords(search)
		if len(keywords) > 0 {
			var orParts []string
			var args []interface{}
			for _, kw := range keywords {
				pattern := "%" + kw + "%"
				orParts = append(orParts, "(name LIKE ? OR description LIKE ? OR tags LIKE ?)")
				args = append(args, pattern, pattern, pattern)
			}
			query = query.Where(strings.Join(orParts, " OR "), args...)
		}
	}

	// 分类筛选
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// 发布者筛选
	if publisherUsername != "" {
		query = query.Where("publisher_username = ?", publisherUsername)
	}

	// 费用筛选
	switch feeType {
	case "free":
		query = query.Where("service_fee_personal = 0")
	case "paid":
		query = query.Where("service_fee_personal > 0")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序：latest=最新，hot=热门(星+复制加权)，stars=按星数，downloads=按复制数
	switch orderBy {
	case "hot":
		query = query.Order("(star_count * 2 + download_count * 1) DESC, created_at DESC")
	case "stars":
		query = query.Order("star_count DESC, created_at DESC")
	case "downloads":
		query = query.Order("download_count DESC, created_at DESC")
	default:
		query = query.Order("created_at DESC")
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&directories).Error
	if err != nil {
		return nil, 0, err
	}

	return directories, total, nil
}

// GetByPackagePath 根据 package_path 获取目录（用于检查是否已发布；仅查在架）
func (r *HubDirectoryRepository) GetByPackagePath(ctx context.Context, packagePath string) (*model.HubDirectory, error) {
	var directory model.HubDirectory
	err := r.db.Scopes(listStatusCondition).Where("package_path = ?", packagePath).Order("created_at DESC").First(&directory).Error
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

// IncrementDownloadCount 下载/复制时增加下载次数（按 full_code_path 定位当前版本记录）
func (r *HubDirectoryRepository) IncrementDownloadCount(ctx context.Context, fullCodePath string) error {
	normalized := normalizeFullCodePath(fullCodePath)
	if normalized == "" {
		return nil
	}
	return r.db.Model(&model.HubDirectory{}).Where("full_code_path = ?", normalized).
		UpdateColumn("download_count", gorm.Expr("download_count + 1")).Error
}

// IncrementStarCount 加星时目录表 star_count +1（仅在新插入 hub_directory_stars 时由 service 调用）
// 若表刚加 star_count 列，已有星星可回填：UPDATE hub_directories d SET star_count = (SELECT COUNT(*) FROM hub_directory_stars s WHERE s.hub_directory_id = d.id)
func (r *HubDirectoryRepository) IncrementStarCount(ctx context.Context, hubDirectoryID int64) error {
	return r.db.WithContext(ctx).Model(&model.HubDirectory{}).Where("id = ?", hubDirectoryID).
		UpdateColumn("star_count", gorm.Expr("star_count + 1")).Error
}

// DecrementStarCount 取消星时目录表 star_count -1（不小于 0）
func (r *HubDirectoryRepository) DecrementStarCount(ctx context.Context, hubDirectoryID int64) error {
	return r.db.WithContext(ctx).Model(&model.HubDirectory{}).Where("id = ?", hubDirectoryID).
		UpdateColumn("star_count", gorm.Expr("GREATEST(0, star_count - 1)")).Error
}

