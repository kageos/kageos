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

// GetAppByUserName 根据用户名和应用名获取应用信息
func (r *AppRepository) GetAppByUserName(user, app string) (*model.App, error) {
	var appModel model.App
	err := r.db.Where("user = ? AND name = ?", user, app).First(&appModel).Error
	if err != nil {
		return nil, err
	}
	return &appModel, nil
}

// CreateApp 创建应用
func (r *AppRepository) CreateApp(app *model.App) error {
	return r.db.Create(app).Error
}

// UpdateApp 更新应用
func (r *AppRepository) UpdateApp(app *model.App) error {
	return r.db.Save(app).Error
}

// DeleteAppAndVersions 删除应用及其所有版本
func (r *AppRepository) DeleteAppAndVersions(user, app string) error {
	// 删除应用记录
	err := r.db.Where("user = ? AND name = ?", user, app).Delete(&model.App{}).Error
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
