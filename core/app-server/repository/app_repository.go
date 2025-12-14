package repository

import (
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// AppRepository 应用仓储
type AppRepository struct {
	db *gorm.DB

	// ✅ 缓存相关
	appCache   sync.Map           // key: "user:app", value: *cachedApp
	cacheGroup singleflight.Group // 防止缓存击穿
}

// cachedApp 缓存的应用信息
type cachedApp struct {
	app       *model.App
	cacheTime time.Time
}

// 缓存过期时间（5 分钟）
const appCacheTTL = 5 * time.Minute

func NewAppRepository(db *gorm.DB) *AppRepository {
	return &AppRepository{
		db:         db,
		appCache:   sync.Map{},           // ✅ 显式初始化缓存
		cacheGroup: singleflight.Group{}, // ✅ 显式初始化 singleflight
	}
}

// GetAppByUserName 根据用户名和应用代码获取应用信息（带缓存）
func (r *AppRepository) GetAppByUserName(user, app string) (*model.App, error) {
	cacheKey := user + ":" + app

	// 1. 快速路径：尝试从缓存获取（大部分请求走这里）
	if cached, ok := r.appCache.Load(cacheKey); ok {
		cachedData := cached.(*cachedApp)
		// 检查是否过期
		if time.Since(cachedData.cacheTime) < appCacheTTL {
			return cachedData.app, nil
		}
		// 过期，删除缓存
		r.appCache.Delete(cacheKey)
	}

	// 2. 慢速路径：缓存未命中，使用 singleflight 防止并发查询
	// 多个并发请求只会有一个真正查询数据库
	value, err, _ := r.cacheGroup.Do(cacheKey, func() (interface{}, error) {
		// 双重检查：可能其他协程已经设置了缓存
		if cached, ok := r.appCache.Load(cacheKey); ok {
			cachedData := cached.(*cachedApp)
			if time.Since(cachedData.cacheTime) < appCacheTTL {
				return cachedData.app, nil
			}
		}

		// 从数据库查询
		var appModel model.App
		// 使用code字段查询，因为app参数是应用代码
		err := r.db.Where("user = ? AND code = ?", user, app).First(&appModel).Error
		if err != nil {
			return nil, err
		}

		// 存入缓存
		r.appCache.Store(cacheKey, &cachedApp{
			app:       &appModel,
			cacheTime: time.Now(),
		})

		return &appModel, nil
	})

	if err != nil {
		return nil, err
	}

	return value.(*model.App), nil
}

// InvalidateAppCache 使应用缓存失效（当应用更新、删除时调用）
func (r *AppRepository) InvalidateAppCache(user, app string) {
	cacheKey := user + ":" + app
	r.appCache.Delete(cacheKey)
}

// CreateApp 创建应用
func (r *AppRepository) CreateApp(app *model.App) error {
	err := r.db.Create(app).Error
	if err != nil {
		return err
	}
	
	// ✅ 创建后可能会立即查询，预热缓存
	r.appCache.Store(app.User+":"+app.Code, &cachedApp{
		app:       app,
		cacheTime: time.Now(),
	})
	
	return nil
}

// ExistsAppNameForUser 判断指定用户下是否已存在同名应用（按中文名称 Name 判断）
func (r *AppRepository) ExistsAppNameForUser(user, name string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.App{}).Where("user = ? AND name = ?", user, name).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountApps 统计应用总数
func (r *AppRepository) CountApps() (int64, error) {
	var count int64
	err := r.db.Model(&model.App{}).Count(&count).Error
	return count, err
}

// CountAppsByUser 统计指定用户的应用数量
func (r *AppRepository) CountAppsByUser(user string) (int64, error) {
	var count int64
	err := r.db.Model(&model.App{}).Where("user = ?", user).Count(&count).Error
	return count, err
}

// UpdateApp 更新应用
func (r *AppRepository) UpdateApp(app *model.App) error {
	err := r.db.Save(app).Error
	if err != nil {
		return err
	}
	
	// ✅ 使缓存失效
	r.InvalidateAppCache(app.User, app.Code)
	return nil
}

// DeleteAppAndVersions 删除应用及其所有版本
func (r *AppRepository) DeleteAppAndVersions(user, app string) error {
	// 删除应用记录（使用code字段，因为app参数是应用代码）
	err := r.db.Where("user = ? AND code = ?", user, app).Delete(&model.App{}).Error
	if err != nil {
		return err
	}

	// ✅ 使缓存失效
	r.InvalidateAppCache(user, app)

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
