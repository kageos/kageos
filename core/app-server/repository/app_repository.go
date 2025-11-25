package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

type AppRepository struct {
	db *gorm.DB
}

func NewAppRepository(db *gorm.DB) *AppRepository {
	return &AppRepository{db: db}
}

// GetAppByUserName 根据用户名和应用代码获取应用信息
func (r *AppRepository) GetAppByUserName(user, app string) (*model.App, error) {
	var appModel model.App
	// 使用code字段查询，因为app参数是应用代码
	err := r.db.Where("user = ? AND code = ?", user, app).First(&appModel).Error
	if err != nil {
		return nil, err
	}
	return &appModel, nil
}

// CreateApp 创建应用
func (r *AppRepository) CreateApp(app *model.App) error {
	return r.db.Create(app).Error
}

// ExistsAppNameForUser 判断指定用户下是否已存在同名应用（按中文名称 Name 判断）
func (r *AppRepository) ExistsAppNameForUser(user, name string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.App{}).Where("user = ? AND name = ?", user, name).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateApp 更新应用
func (r *AppRepository) UpdateApp(app *model.App) error {
	return r.db.Save(app).Error
}

// DeleteAppAndVersions 删除应用及其所有版本
func (r *AppRepository) DeleteAppAndVersions(user, app string) error {
	// 删除应用记录（使用code字段，因为app参数是应用代码）
	err := r.db.Where("user = ? AND code = ?", user, app).Delete(&model.App{}).Error
	if err != nil {
		return err
	}

	// 注意：app-server 中没有 AppVersion 表，所以只删除 App 记录即可

	return nil
}

// GetAppByID 根据ID获取应用信息
func (r *AppRepository) GetAppByID(id int64) (*model.App, error) {
	var app model.App
	err := r.db.Where("id = ?", id).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetAppsByUser 根据用户获取所有应用
func (r *AppRepository) GetAppsByUser(user string) ([]*model.App, error) {
	var apps []*model.App
	err := r.db.Where("user = ?", user).Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}

// GetAppsByUserWithPage 根据用户获取分页应用列表（支持搜索）
func (r *AppRepository) GetAppsByUserWithPage(user string, page, pageSize int, search string) ([]*model.App, int64, error) {
	var apps []*model.App
	var totalCount int64

	// 构建查询条件
	query := r.db.Model(&model.App{}).Where("user = ?", user)

	// 如果有关键词，添加搜索条件（按名称或代码搜索）
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", searchPattern, searchPattern)
	}

	// 获取总数
	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据
	err = query.Offset(offset).Limit(pageSize).Find(&apps).Error
	if err != nil {
		return nil, 0, err
	}

	return apps, totalCount, nil
}
