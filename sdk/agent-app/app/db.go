package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
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
	appDBDialectEnv       = "KAGEOS_APP_DB_DIALECT"
	appDBRootPackagePath  = "_root"
	appDBResolveTimeout   = 5 * time.Second
	appDBEndpointTimeout  = time.Second
	appDBIdleTTL          = 10 * time.Minute
	appDBCleanupInterval  = time.Minute
	defaultMySQLMaxOpen   = 2
	defaultMySQLMaxIdle   = 0
	defaultMySQLIdleTime  = 30 * time.Second
	defaultMySQLLifeTime  = 10 * time.Minute
	defaultSQLiteLifeTime = time.Hour
)

type dbCacheEntry struct {
	db       *gorm.DB
	dialect  string
	lastUsed time.Time
}

func getDBName() string {
	return fmt.Sprintf("%s_%s.db", env.User, env.App)
}

func (c *Context) GetGormDB() *gorm.DB {
	// 如果 Context 中有 routerInfo 且 PackagePath 不为空，使用 PackagePath 构建数据库名称
	// 否则使用默认的数据库名称（兼容旧代码）
	var dbName string
	var packagePath string

	if c != nil && c.routerInfo != nil && c.routerInfo.Options != nil {
		// 根据 PackagePath 构建数据库名称
		// 例如：/plugins -> plugins.db, /crm/ticket -> crm_ticket.db
		dbName = c.routerInfo.Options.GetDBName(c.msg.User, c.msg.App)
		packagePath = strings.Trim(c.routerInfo.Options.PackagePath, "/")
	} else {
		// 兼容旧代码，使用默认数据库名称
		dbName = getDBName()
		logger.Infof(context.Background(), "使用了旧的db 逻辑 get db name: %s", dbName)
	}

	capability := (*dto.AppDBCapability)(nil)
	if c != nil {
		capability = c.dbCapability
	}
	if isRuntimeMySQLAppDBEnabled() || capability != nil {
		if packagePath == "" {
			packagePath = appDBRootPackagePath
		}
		db, err := getOrInitMySQLDB(packagePath, capability)
		if err != nil {
			logger.Errorf(context.Background(), "获取 MySQL 应用数据库失败 package=%s: %v", packagePath, err)
			return nil
		}
		return db
	}

	db, err := getOrInitDB(dbName)
	if err != nil {
		return nil
	}
	return db
}

// GetDBByPackagePath 根据包路径获取 DB。
// SQLite 兼容模式下可用于无请求上下文的后台任务；runtime MySQL 模式下必须使用 ctx.GetGormDB，
// 避免生成代码绕过当前 Context 的私有能力凭证访问其它 package DB。
func GetDBByPackagePath(packagePath string) (*gorm.DB, error) {
	packagePath = strings.Trim(packagePath, "/")
	if isRuntimeMySQLAppDBEnabled() {
		return nil, errors.New("runtime MySQL app database requires SDK Context capability; use ctx.GetGormDB()")
	}
	opts := &RegisterOptions{PackagePath: packagePath}
	dbName := opts.GetDBName(env.User, env.App)
	return getOrInitDB(dbName)
}

// sanitizeDBName 安全处理数据库名称，防止目录穿越
func sanitizeDBName(dbName string) string {
	// 移除路径前缀
	dbName = strings.TrimPrefix(dbName, "../")
	dbName = strings.TrimPrefix(dbName, "./")

	// 确保只取基本文件名，防止目录穿越
	dbName = filepath.Base(dbName)

	// 确保有.db后缀
	if !strings.HasSuffix(dbName, ".db") {
		dbName = dbName + ".db"
	}

	// 计算数据目录（可配置，默认到 $HOME/.kageos/data）
	base := getDataDir()
	return filepath.Join(base, dbName)
}

// getDataDir 获取数据目录（优先环境变量 KAGEOS_DATA_DIR，其次 $HOME/.kageos/data）
func getDataDir() string {
	// 固定为容器内的绝对路径
	return "/app/workplace/data"
}

// getOrInitDB 获取或初始化数据库连接
// 如果数据库不存在，会自动创建
func getOrInitDB(dbName string) (*gorm.DB, error) {
	dbLock.Lock()
	defer dbLock.Unlock()
	dbCleanupOnce.Do(startDBCleanupLoop)

	// 安全处理数据库名称，防止目录穿越攻击
	dbName = sanitizeDBName(dbName)

	// 检查缓存是否已存在连接
	if entry, ok := dbs[dbName]; ok && entry != nil {
		entry.lastUsed = time.Now()
		return entry.db, nil
	}

	if err := ensureDBDataDir(dbName); err != nil {
		return nil, err
	}

	logFile := openDBLogFile(dbName)
	dbLogger := newGormDBLogger(logFile)

	db, err := openSQLiteDB(dbName, dbLogger)
	if err != nil {
		return nil, err
	}
	logger.Infof(context.Background(), "打开数据库成功 %s", dbName)

	configureSQLitePragmas(db)
	if err := configureSQLiteConnectionPool(db); err != nil {
		return nil, err
	}

	// 🔥 注意：SQLite 不支持 FIND_IN_SET 函数
	// 我们已经在 query1.go 中使用 SQLite 兼容的方式（instr 函数）来实现相同功能
	// 所以不需要在这里注册自定义函数

	// 缓存连接
	dbs[dbName] = &dbCacheEntry{db: db, dialect: "sqlite", lastUsed: time.Now()}
	logger.Infof(context.Background(), "数据库连接已创建: %s", dbName)

	return db, nil
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

func isRuntimeMySQLAppDBEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(appDBDialectEnv)), "mysql")
}

func cloneDBCapability(capability *dto.AppDBCapability) *dto.AppDBCapability {
	if capability == nil {
		return nil
	}
	copied := *capability
	return &copied
}

func ensureDBDataDir(dbName string) error {
	dataDir := filepath.Dir(dbName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Errorf(context.Background(), "创建数据目录失败: %v", err)
		return err
	}
	return nil
}

func dbLogFilePath(dbName string) string {
	logFileName := strings.TrimSuffix(filepath.Base(dbName), ".db") + ".log"
	return filepath.Join(filepath.Dir(dbName), logFileName)
}

func openDBLogFile(dbName string) *os.File {
	logFilePath := dbLogFilePath(dbName)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Errorf(context.Background(), "打开日志文件失败 %s: %v", logFilePath, err)
		return os.Stdout
	}
	return logFile
}

func newGormDBLogger(logFile *os.File) gormLogger.Interface {
	return gormLogger.New(
		log.New(logFile, "\r\n", log.LstdFlags),
		gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogger.Info, // 记录所有 SQL 语句到文件
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

func openSQLiteDB(dbName string, dbLogger gormLogger.Interface) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		logger.Errorf(context.Background(), "打开数据库失败 %s: %v", dbName, err)
		return nil, err
	}
	return db, nil
}

func configureSQLitePragmas(db *gorm.DB) {
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA temp_store=MEMORY;")
	db.Exec("PRAGMA synchronous=NORMAL;")
	db.Exec("PRAGMA busy_timeout=5000;")
	db.Exec("PRAGMA cache_size=-64000;")
	db.Exec("PRAGMA journal_size_limit=67108864;")
}

func configureSQLiteConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf(context.Background(), "获取原生数据库连接失败: %v", err)
		return err
	}

	maxOpenConns := 10
	maxIdleConns := 5
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(defaultSQLiteLifeTime)

	logger.Infof(context.Background(), "数据库连接池已配置: MaxOpenConns=%d, MaxIdleConns=%d, MaxLifetime=%v",
		maxOpenConns, maxIdleConns, defaultSQLiteLifeTime)
	return nil
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
