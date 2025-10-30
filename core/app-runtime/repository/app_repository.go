package repository

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/model"
	"gorm.io/gorm"
)

// AppRepository 应用数据访问层
type AppRepository struct {
	db *gorm.DB
}

// NewAppRepository 创建应用仓库
func NewAppRepository(db *gorm.DB) *AppRepository {
	return &AppRepository{
		db: db,
	}
}

// CreateApp 创建应用记录
func (r *AppRepository) CreateApp(user, app string) error {
	appRecord := &model.App{
		User:      user,
		App:       app,
		Version:   "",         // 初始时无版本，第一次 Update 时才会有版本
		Status:    "inactive", // 创建时默认为未激活状态
		StartTime: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := r.db.Create(appRecord).Error; err != nil {
		return fmt.Errorf("failed to save app to database: %w", err)
	}

	// 不再创建初始版本记录，版本记录应该在第一次 Update 时创建
	return nil
}

// DeleteAppAndVersions 删除应用及其所有版本记录
func (r *AppRepository) DeleteAppAndVersions(user, app string) error {
	// 删除应用记录
	if err := r.db.Where("user = ? and app = ?", user, app).Delete(&model.App{}).Error; err != nil {
		return err
	}

	// 删除应用版本记录
	if err := r.db.Where("user = ? and app = ?", user, app).Delete(&model.AppVersion{}).Error; err != nil {
		return err
	}

	return nil
}

// GetApp 获取应用信息
func (r *AppRepository) GetApp(user, app string) (*model.App, error) {
	var appRecord model.App
	if err := r.db.Where("user = ? and app = ?", user, app).First(&appRecord).Error; err != nil {
		return nil, err
	}
	return &appRecord, nil
}

// UpdateApp 更新应用信息
func (r *AppRepository) UpdateApp(app *model.App) error {
	return r.db.Save(app).Error
}

// GetAppVersions 获取应用的所有版本
func (r *AppRepository) GetAppVersions(user, app string) ([]*model.AppVersion, error) {
	var versions []*model.AppVersion
	if err := r.db.Where("user = ? and app = ?", user, app).Order("created_at DESC").Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// CreateAppVersion 创建应用版本记录
func (r *AppRepository) CreateAppVersion(version *model.AppVersion) error {
	return r.db.Create(version).Error
}

// UpdateAppVersion 更新应用版本信息
func (r *AppRepository) UpdateAppVersion(version *model.AppVersion) error {
	return r.db.Save(version).Error
}

// GetActiveAppVersion 获取活跃的应用版本
func (r *AppRepository) GetActiveAppVersion(user, app string) (*model.AppVersion, error) {
	var version model.AppVersion
	if err := r.db.Where("user = ? and app = ? and status = ?", user, app, "active").First(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

// GetAllApps 获取所有应用
func (r *AppRepository) GetAllApps() ([]*model.App, error) {
	var apps []*model.App
	if err := r.db.Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}
