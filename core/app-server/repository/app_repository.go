package repository

import (
	"strconv"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// AppRepository 应用仓储
type AppRepository struct {
	db *gorm.DB

	// ✅ 缓存相关（使用读写锁 + map，与 roleCache 保持一致）
	appCache     map[string]*cachedApp // key: "user:app", value: *cachedApp
	appIDCache   map[int64]*cachedApp  // key: appID (int64), value: *cachedApp
	appCacheMu   sync.RWMutex          // 读写锁（保护 appCache）
	appIDCacheMu sync.RWMutex          // 读写锁（保护 appIDCache）
	cacheGroup   singleflight.Group    // 防止缓存击穿（用于 user:app）
	cacheIDGroup singleflight.Group    // 防止缓存击穿（用于 appID）
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
		db:           db,
		appCache:     make(map[string]*cachedApp), // ✅ 显式初始化缓存（user:app）
		appIDCache:   make(map[int64]*cachedApp),  // ✅ 显式初始化缓存（appID）
		cacheGroup:   singleflight.Group{},        // ✅ 显式初始化 singleflight（user:app）
		cacheIDGroup: singleflight.Group{},        // ✅ 显式初始化 singleflight（appID）
	}
}

// GetDB 获取数据库连接（用于复杂查询或创建其他仓库）
func (r *AppRepository) GetDB() *gorm.DB {
	return r.db
}

// GetAppByUserName 根据用户名和应用代码获取应用信息（带缓存）
func (r *AppRepository) GetAppByUserName(user, app string) (*model.App, error) {
	cacheKey := user + ":" + app

	// 1. 快速路径：尝试从缓存获取（大部分请求走这里）
	r.appCacheMu.RLock()
	cached, ok := r.appCache[cacheKey]
	r.appCacheMu.RUnlock()

	if ok {
		// 检查是否过期
		if time.Since(cached.cacheTime) < appCacheTTL {
			return cached.app, nil
		}
		// 过期，删除缓存
		r.appCacheMu.Lock()
		delete(r.appCache, cacheKey)
		r.appCacheMu.Unlock()
	}

	// 2. 慢速路径：缓存未命中，使用 singleflight 防止并发查询
	// 多个并发请求只会有一个真正查询数据库
	value, err, _ := r.cacheGroup.Do(cacheKey, func() (interface{}, error) {
		// 双重检查：可能其他协程已经设置了缓存
		r.appCacheMu.RLock()
		cached, ok := r.appCache[cacheKey]
		r.appCacheMu.RUnlock()

		if ok {
			if time.Since(cached.cacheTime) < appCacheTTL {
				return cached.app, nil
			}
		}

		// 从数据库查询
		var appModel model.App
		// 使用code字段查询，因为app参数是应用代码
		err := r.db.Where("user = ? AND code = ?", user, app).First(&appModel).Error
		if err != nil {
			return nil, err
		}

		// 存入缓存（同时更新两个缓存：user:app 和 appID，保持一致性）
		cached = &cachedApp{
			app:       &appModel,
			cacheTime: time.Now(),
		}
		r.appCacheMu.Lock()
		r.appCache[cacheKey] = cached
		r.appCacheMu.Unlock()

		r.appIDCacheMu.Lock()
		r.appIDCache[appModel.ID] = cached
		r.appIDCacheMu.Unlock()

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
	r.appCacheMu.Lock()
	delete(r.appCache, cacheKey)
	r.appCacheMu.Unlock()
}

// InvalidateAppCacheByID 使应用缓存失效（通过 appID）
func (r *AppRepository) InvalidateAppCacheByID(appID int64) {
	r.appIDCacheMu.Lock()
	delete(r.appIDCache, appID)
	r.appIDCacheMu.Unlock()
}

// InvalidateAppCacheBoth 使应用缓存失效（同时清除 user:app 和 appID 缓存）
func (r *AppRepository) InvalidateAppCacheBoth(user, app string, appID int64) {
	cacheKey := user + ":" + app
	r.appCacheMu.Lock()
	delete(r.appCache, cacheKey)
	r.appCacheMu.Unlock()

	r.appIDCacheMu.Lock()
	delete(r.appIDCache, appID)
	r.appIDCacheMu.Unlock()
}

// CreateApp 创建应用
func (r *AppRepository) CreateApp(app *model.App) error {
	err := r.db.Create(app).Error
	if err != nil {
		return err
	}

	// ✅ 创建后可能会立即查询，预热缓存（同时预热 user:app 和 appID 缓存）
	cached := &cachedApp{
		app:       app,
		cacheTime: time.Now(),
	}
	cacheKey := app.User + ":" + app.Code
	r.appCacheMu.Lock()
	r.appCache[cacheKey] = cached
	r.appCacheMu.Unlock()

	r.appIDCacheMu.Lock()
	r.appIDCache[app.ID] = cached
	r.appIDCacheMu.Unlock()

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

	// ✅ 使缓存失效（同时清除 user:app 和 appID 缓存）
	r.InvalidateAppCacheBoth(app.User, app.Code, app.ID)
	return nil
}

// UpdateAppVersion 更新应用版本（仅更新版本字段，更高效）
func (r *AppRepository) UpdateAppVersion(user, app, newVersion string) error {
	// ⚠️ 需要先查询 appID，以便清除 appID 缓存
	appModel, err := r.GetAppByUserName(user, app)
	if err != nil {
		return err
	}

	err = r.db.Model(&model.App{}).
		Where("user = ? AND code = ?", user, app).
		Update("version", newVersion).Error
	if err != nil {
		return err
	}

	// ✅ 使缓存失效（同时清除 user:app 和 appID 缓存）
	r.InvalidateAppCacheBoth(user, app, appModel.ID)
	return nil
}

// DeleteAppAndVersions 删除应用及其所有版本
func (r *AppRepository) DeleteAppAndVersions(user, app string) error {
	// ⚠️ 需要先查询 appID，以便清除 appID 缓存
	appModel, err := r.GetAppByUserName(user, app)
	if err != nil {
		return err
	}

	// 删除应用记录（使用code字段，因为app参数是应用代码）
	err = r.db.Where("user = ? AND code = ?", user, app).Delete(&model.App{}).Error
	if err != nil {
		return err
	}

	// ✅ 使缓存失效（同时清除 user:app 和 appID 缓存）
	r.InvalidateAppCacheBoth(user, app, appModel.ID)

	// 注意：app-server 中没有 AppVersion 表，所以只删除 App 记录即可

	return nil
}

// GetAppByID 根据ID获取应用信息（带缓存）
func (r *AppRepository) GetAppByID(id int64) (*model.App, error) {
	// 1. 快速路径：尝试从缓存获取（大部分请求走这里）
	r.appIDCacheMu.RLock()
	cached, ok := r.appIDCache[id]
	r.appIDCacheMu.RUnlock()

	if ok {
		// 检查是否过期
		if time.Since(cached.cacheTime) < appCacheTTL {
			return cached.app, nil
		}
		// 过期，删除缓存
		r.appIDCacheMu.Lock()
		delete(r.appIDCache, id)
		r.appIDCacheMu.Unlock()
	}

	// 2. 慢速路径：缓存未命中，使用 singleflight 防止并发查询
	// 多个并发请求只会有一个真正查询数据库
	cacheKey := strconv.FormatInt(id, 10) // 将 int64 转换为 string 作为 singleflight 的 key
	value, err, _ := r.cacheIDGroup.Do(cacheKey, func() (interface{}, error) {
		// 双重检查：可能其他协程已经设置了缓存
		r.appIDCacheMu.RLock()
		cached, ok := r.appIDCache[id]
		r.appIDCacheMu.RUnlock()

		if ok {
			if time.Since(cached.cacheTime) < appCacheTTL {
				return cached.app, nil
			}
		}

		// 从数据库查询
		var appModel model.App
		err := r.db.Where("id = ?", id).First(&appModel).Error
		if err != nil {
			return nil, err
		}

		// 存入缓存（同时更新两个缓存：appID 和 user:app）
		cached = &cachedApp{
			app:       &appModel,
			cacheTime: time.Now(),
		}
		r.appIDCacheMu.Lock()
		r.appIDCache[id] = cached
		r.appIDCacheMu.Unlock()

		// 同时更新 user:app 缓存，保持一致性
		cacheKeyStr := appModel.User + ":" + appModel.Code
		r.appCacheMu.Lock()
		r.appCache[cacheKeyStr] = cached
		r.appCacheMu.Unlock()

		return &appModel, nil
	})

	if err != nil {
		return nil, err
	}

	return value.(*model.App), nil
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
// 保留此方法以保持向后兼容
func (r *AppRepository) GetAppsByUserWithPage(user string, page, pageSize int, search string) ([]*model.App, int64, error) {
	return r.GetAppsWithPage(user, page, pageSize, search, false, nil)
}

// GetAppsWithPage 获取分页应用列表（支持搜索和过滤）
// user: 当前用户（用于过滤自己的应用）
// includeAll: 如果为 true，返回自己的应用 + 所有公开的应用；如果为 false，只返回自己的应用
// appType: 应用类型筛选（可选）：nil=不筛选，0=用户空间，1=系统空间
func (r *AppRepository) GetAppsWithPage(user string, page, pageSize int, search string, includeAll bool, appType *int) ([]*model.App, int64, error) {
	var apps []*model.App
	var totalCount int64

	// 构建查询条件
	var query *gorm.DB

	// 如果指定了系统工作空间类型（type=1），返回所有系统工作空间，不受用户限制
	if appType != nil && *appType == 1 {
		// 系统工作空间：返回所有type=1的应用
		query = r.db.Model(&model.App{}).Where("type = ?", 1)
	} else if includeAll {
		// 包含自己的应用 + 所有公开的应用
		query = r.db.Model(&model.App{}).Where("user = ? OR is_public = ?", user, true)
		// 如果指定了应用类型，添加类型筛选
		if appType != nil {
			query = query.Where("type = ?", *appType)
		}
	} else {
		// 只包含自己的应用
		query = r.db.Model(&model.App{}).Where("user = ?", user)
		// 如果指定了应用类型，添加类型筛选
		if appType != nil {
			query = query.Where("type = ?", *appType)
		}
	}

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

	// 获取分页数据，按创建时间倒序
	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error
	if err != nil {
		return nil, 0, err
	}

	return apps, totalCount, nil
}
