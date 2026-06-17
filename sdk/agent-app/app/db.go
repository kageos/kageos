package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/msgx"
	"github.com/kageos/kageos/pkg/netprobe"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/kageos/kageos/sdk/agent-app/env"
	_ "github.com/ncruces/go-sqlite3/driver" // register database/sql driver "sqlite3" for uploaded SQLite files
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var (
	dbLock             = new(sync.Mutex)
	dbs                = make(map[string]*dbCacheEntry)
	dbCleanupOnce      sync.Once
	mysqlEndpointCache sync.Map
)

const (
	appDBRootPackagePath = "_root"
	appDBResolveTimeout  = 5 * time.Second
	appDBEndpointTimeout = time.Second
	appDBIdleTTL         = 10 * time.Minute
	appDBCleanupInterval = time.Minute
	defaultMySQLMaxOpen  = 2
	defaultMySQLMaxIdle  = 0
	defaultMySQLIdleTime = 30 * time.Second
	defaultMySQLLifeTime = 10 * time.Minute
)

type dbCacheEntry struct {
	db       *gorm.DB
	dialect  string
	lastUsed time.Time
}

func (c *Context) GetGormDB() *gorm.DB {
	packagePath := appDBRootPackagePath
	if c != nil && c.routerInfo != nil && c.routerInfo.Options != nil {
		packagePath = strings.Trim(c.routerInfo.Options.PackagePath, "/")
	}
	if packagePath == "" {
		packagePath = appDBRootPackagePath
	}
	if c == nil || c.dbCapability == nil {
		logger.Errorf(context.Background(), "MySQL 应用数据库能力不可用 package=%s；请在请求或回调 Context 中使用 ctx.GetGormDB()", packagePath)
		return nil
	}
	db, err := getOrInitMySQLDB(packagePath, c.dbCapability)
	if err != nil {
		logger.Errorf(context.Background(), "获取 MySQL 应用数据库失败 package=%s: %v", packagePath, err)
		return nil
	}
	return db
}

// GetDBByPackagePath 已废弃。应用业务数据库是 MySQL-only，并且必须依赖当前请求或回调的
// package-scoped capability；无 Context 的后台入口不能直接获取业务库连接。
func GetDBByPackagePath(packagePath string) (*gorm.DB, error) {
	return nil, errors.New("KageOS app business database is MySQL-only and requires request Context capability; use ctx.GetGormDB() inside handlers/callbacks")
}

func getOrInitMySQLDB(packagePath string, capability *dto.AppDBCapability) (*gorm.DB, error) {
	return getOrInitMySQLDBForAccess(packagePath, capability, dto.AppDBAccessRuntime)
}

func getOrInitMySQLMigrationDB(packagePath string, capability *dto.AppDBCapability) (*gorm.DB, error) {
	return getOrInitMySQLDBForAccess(packagePath, capability, dto.AppDBAccessMigration)
}

func getOrInitMySQLDBForAccess(packagePath string, capability *dto.AppDBCapability, access string) (*gorm.DB, error) {
	packagePath = strings.Trim(packagePath, "/")
	if packagePath == "" {
		packagePath = appDBRootPackagePath
	}
	if strings.TrimSpace(access) == "" {
		access = dto.AppDBAccessRuntime
	}
	if capability == nil {
		return nil, errors.New("runtime MySQL app database capability is unavailable")
	}

	dbLock.Lock()
	defer dbLock.Unlock()
	dbCleanupOnce.Do(startDBCleanupLoop)

	cacheKey := "mysql:" + access + ":" + packagePath
	if entry, ok := dbs[cacheKey]; ok && entry != nil {
		entry.lastUsed = time.Now()
		return entry.db, nil
	}

	resp, err := resolveRuntimeAppDatabase(packagePath, capability, access)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(resp.Dialect, "mysql") {
		return nil, fmt.Errorf("unsupported runtime app database dialect: %s", resp.Dialect)
	}

	dsn, err := resolveMySQLDSNEndpoint(resp.DSN)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Warn),
	})
	if err != nil {
		logger.Errorf(context.Background(), "打开 MySQL 应用数据库失败 package=%s db=%s: %v", packagePath, resp.DatabaseName, err)
		return nil, err
	}
	if err := configureMySQLConnectionPool(db, resp); err != nil {
		return nil, err
	}

	dbs[cacheKey] = &dbCacheEntry{db: db, dialect: "mysql", lastUsed: time.Now()}
	logger.Infof(context.Background(), "MySQL 应用数据库连接已创建: package=%s access=%s db=%s", packagePath, resp.Access, resp.DatabaseName)
	return db, nil
}

func resolveRuntimeAppDatabase(packagePath string, capability *dto.AppDBCapability, access string) (*dto.AppDBResolveResp, error) {
	if initErr != nil {
		return nil, initErr
	}
	if app == nil || app.conn == nil {
		return nil, errors.New("app NATS connection is unavailable")
	}

	req := dto.AppDBResolveReq{
		Capability:  capability,
		User:        env.User,
		App:         env.App,
		Version:     env.Version,
		PackagePath: packagePath,
		Access:      access,
	}
	var resp dto.AppDBResolveResp
	_, err := msgx.RequestJSON(context.Background(), app.conn, subjects.RuntimeAppDBResolveQuerySubject, &req, &resp, appDBResolveTimeout)
	if err != nil {
		return nil, fmt.Errorf("resolve runtime app database: %w", err)
	}
	return &resp, nil
}

func resolveMySQLDSNEndpoint(dsn string) (string, error) {
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("parse runtime MySQL DSN: %w", err)
	}
	if cfg.Net != "tcp" || strings.TrimSpace(cfg.Addr) == "" {
		return dsn, nil
	}
	if cached, ok := mysqlEndpointCache.Load(cfg.Addr); ok {
		cfg.Addr = cached.(string)
		return cfg.FormatDSN(), nil
	}
	resolved, err := netprobe.ResolveTCPEndpoint(context.Background(), cfg.Addr, appDBEndpointTimeout)
	if err != nil {
		return "", fmt.Errorf("resolve runtime MySQL endpoint %s: %w", cfg.Addr, err)
	}
	if resolved != cfg.Addr {
		logger.Infof(context.Background(), "MySQL endpoint auto-resolved: %s -> %s", cfg.Addr, resolved)
	}
	mysqlEndpointCache.Store(cfg.Addr, resolved)
	cfg.Addr = resolved
	return cfg.FormatDSN(), nil
}

func cloneDBCapability(capability *dto.AppDBCapability) *dto.AppDBCapability {
	if capability == nil {
		return nil
	}
	copied := *capability
	return &copied
}

func configureMySQLConnectionPool(db *gorm.DB, resp *dto.AppDBResolveResp) error {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf(context.Background(), "获取 MySQL 原生数据库连接失败: %v", err)
		return err
	}

	maxOpenConns := resp.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = defaultMySQLMaxOpen
	}
	maxIdleConns := resp.MaxIdleConns
	if maxIdleConns < 0 {
		maxIdleConns = defaultMySQLMaxIdle
	}
	idleTime := time.Duration(resp.MaxIdleTime) * time.Second
	if idleTime <= 0 {
		idleTime = defaultMySQLIdleTime
	}
	lifetime := time.Duration(resp.MaxLifetime) * time.Second
	if lifetime <= 0 {
		lifetime = defaultMySQLLifeTime
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxIdleTime(idleTime)
	sqlDB.SetConnMaxLifetime(lifetime)

	logger.Infof(context.Background(), "MySQL 数据库连接池已配置: MaxOpenConns=%d, MaxIdleConns=%d, MaxIdleTime=%v, MaxLifetime=%v",
		maxOpenConns, maxIdleConns, idleTime, lifetime)
	return nil
}

func startDBCleanupLoop() {
	go func() {
		ticker := time.NewTicker(appDBCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			cleanupIdleDatabases()
		}
	}()
}

func cleanupIdleDatabases() {
	dbLock.Lock()
	defer dbLock.Unlock()

	now := time.Now()
	for key, entry := range dbs {
		if entry == nil || entry.db == nil {
			delete(dbs, key)
			continue
		}
		if now.Sub(entry.lastUsed) < appDBIdleTTL {
			continue
		}
		if err := closeDBConnection(entry.db); err != nil {
			logger.Warnf(context.Background(), "关闭空闲数据库连接失败: %s, error: %v", key, err)
			continue
		}
		delete(dbs, key)
		logger.Infof(context.Background(), "空闲数据库连接已释放: %s", key)
	}
}

func closeDBConnection(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil || sqlDB == nil {
		return err
	}
	return sqlDB.Close()
}

// closeAllDatabases 关闭所有数据库连接
// 在应用退出时调用，释放数据库连接占用的内存
func closeAllDatabases() {
	dbLock.Lock()
	defer dbLock.Unlock()

	closedCount := 0
	for dbName, entry := range dbs {
		if entry != nil && entry.db != nil {
			if err := closeDBConnection(entry.db); err != nil {
				logger.Warnf(context.Background(), "关闭数据库连接失败: %s, error: %v", dbName, err)
			} else {
				closedCount++
				logger.Infof(context.Background(), "数据库连接已关闭: %s", dbName)
			}
		}
	}

	// 清空连接缓存
	dbs = make(map[string]*dbCacheEntry)

	if closedCount > 0 {
		logger.Infof(context.Background(), "已关闭 %d 个数据库连接", closedCount)
	}
}
