package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/env"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var (
	dbLock = new(sync.Mutex)
	dbs    = make(map[string]*gorm.DB)
)

func getDBName() string {
	return fmt.Sprintf("%s_%s.db", env.User, env.App)
}

func (c *Context) GetGormDB() *gorm.DB {
	// 如果 Context 中有 routerInfo 且 PackagePath 不为空，使用 PackagePath 构建数据库名称
	// 否则使用默认的数据库名称（兼容旧代码）
	var dbName string

	if c.routerInfo != nil && c.routerInfo.Options != nil {
		// 根据 PackagePath 构建数据库名称
		// 例如：/plugins -> plugins.db, /crm/ticket -> crm_ticket.db
		dbName = c.routerInfo.Options.GetDBName(c.msg.User, c.msg.App)
	} else {
		// 兼容旧代码，使用默认数据库名称
		dbName = getDBName()
		logger.Infof(c, "使用了旧的db 逻辑 get db name: %s", dbName)
	}

	db, err := getOrInitDB(dbName)
	if err != nil {
		return nil
	}
	return db
}

// GetDBByPackagePath 根据包路径获取 DB（与请求里 ctx.GetGormDB() 同源，用于后台任务等无请求上下文的场景）
// packagePath 与 PackageContext.RouterGroup 去掉首尾 "/" 一致，如 "booking"、"luobei/example/code/api/booking"
func GetDBByPackagePath(packagePath string) (*gorm.DB, error) {
	opts := &RegisterOptions{PackagePath: strings.Trim(packagePath, "/")}
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

	// 安全处理数据库名称，防止目录穿越攻击
	dbName = sanitizeDBName(dbName)

	// 检查缓存是否已存在连接
	if db, ok := dbs[dbName]; ok {
		return db, nil
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
	if err := configureDBConnectionPool(db); err != nil {
		return nil, err
	}

	// 🔥 注意：SQLite 不支持 FIND_IN_SET 函数
	// 我们已经在 query1.go 中使用 SQLite 兼容的方式（instr 函数）来实现相同功能
	// 所以不需要在这里注册自定义函数

	// 缓存连接
	dbs[dbName] = db
	logger.Infof(context.Background(), "数据库连接已创建: %s", dbName)

	return db, nil
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

func configureDBConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf(context.Background(), "获取原生数据库连接失败: %v", err)
		return err
	}

	maxOpenConns := 10
	maxIdleConns := 5
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Infof(context.Background(), "数据库连接池已配置: MaxOpenConns=%d, MaxIdleConns=%d, MaxLifetime=%v",
		maxOpenConns, maxIdleConns, time.Hour)
	return nil
}

// closeAllDatabases 关闭所有数据库连接
// 在应用退出时调用，释放数据库连接占用的内存
func closeAllDatabases() {
	dbLock.Lock()
	defer dbLock.Unlock()

	closedCount := 0
	for dbName, db := range dbs {
		if db != nil {
			// 获取原生数据库连接
			sqlDB, err := db.DB()
			if err == nil && sqlDB != nil {
				// 关闭数据库连接
				if err := sqlDB.Close(); err != nil {
					logger.Warnf(context.Background(), "关闭数据库连接失败: %s, error: %v", dbName, err)
				} else {
					closedCount++
					logger.Infof(context.Background(), "数据库连接已关闭: %s", dbName)
				}
			}
		}
	}

	// 清空连接缓存
	dbs = make(map[string]*gorm.DB)

	if closedCount > 0 {
		logger.Infof(context.Background(), "已关闭 %d 个数据库连接", closedCount)
	}
}
