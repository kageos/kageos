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

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
	"github.com/glebarez/sqlite"
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

// GetDBByPackagePath 根据包路径获取 DB（与请求里 ctx.GetGormDB() 同源，用于定时任务等无请求上下文的场景）
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

	// 计算数据目录（可配置，默认到 $HOME/.ai-agent-os/data）
	base := getDataDir()
	return filepath.Join(base, dbName)
}

// getDataDir 获取数据目录（优先环境变量 AI_AGENT_OS_DATA_DIR，其次 $HOME/.ai-agent-os/data）
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

	// 确保数据目录存在
	dataDir := filepath.Dir(dbName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Errorf(context.Background(), "创建数据目录失败: %v", err)
		return nil, err
	}

	// 🔥 创建日志文件，使用数据库文件名来命名日志文件
	// 例如：luobei_demo_crm_ticket.db -> luobei_demo_crm_ticket.log
	logFileName := strings.TrimSuffix(filepath.Base(dbName), ".db") + ".log"
	logFilePath := filepath.Join(dataDir, logFileName)

	// 打开日志文件（追加模式）
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Errorf(context.Background(), "打开日志文件失败 %s: %v", logFilePath, err)
		// 如果打开日志文件失败，使用标准输出作为降级方案
		logFile = os.Stdout
	}

	// 🔥 只写入文件，不输出到控制台
	// 设置GORM日志配置
	gormLogger := gormLogger.New(
		log.New(logFile, "\r\n", log.LstdFlags),
		gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogger.Info, // 记录所有 SQL 语句到文件
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// 创建数据库连接 - 使用纯 Go SQLite 驱动
	// 使用 github.com/glebarez/sqlite 驱动，无需 CGO
	// 注意：需要在编译时设置 CGO_ENABLED=0 来使用纯 Go 驱动
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		logger.Errorf(context.Background(), "打开数据库失败 %s: %v", dbName, err)
		return nil, err
	}
	logger.Infof(context.Background(), "打开数据库成功 %s", dbName)

	// 设置SQLite优化参数（提升并发读写性能）
	// WAL 模式：提升并发读写性能
	db.Exec("PRAGMA journal_mode=WAL;")
	// 临时存储到内存：提升性能
	db.Exec("PRAGMA temp_store=MEMORY;")
	// 同步模式：NORMAL 平衡性能和安全性
	db.Exec("PRAGMA synchronous=NORMAL;")
	// ✅ 优化：设置忙等待超时 5 秒，减少 "database is locked" 错误
	db.Exec("PRAGMA busy_timeout=5000;")
	// ✅ 优化：设置缓存大小 64MB，提升查询性能（负值表示 KB）
	db.Exec("PRAGMA cache_size=-64000;")
	// ✅ 优化：限制 WAL 日志文件大小 64MB，防止日志文件无限增长
	db.Exec("PRAGMA journal_size_limit=67108864;")

	// 设置连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf(context.Background(), "获取原生数据库连接失败: %v", err)
		return nil, err
	}

	// ✅ 优化：增加连接池大小，支持更高并发
	// SQLite 是文件数据库，并发能力有限，建议不超过 20
	maxOpenConns := 10
	maxIdleConns := 5
	sqlDB.SetMaxOpenConns(maxOpenConns) // ✅ 从 5 增加到 10，支持更高并发
	sqlDB.SetMaxIdleConns(maxIdleConns) // ✅ 从 2 增加到 5，保持更多空闲连接
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最长生命周期 1 小时（合理）

	logger.Infof(context.Background(), "数据库连接池已配置: MaxOpenConns=%d, MaxIdleConns=%d, MaxLifetime=%v",
		maxOpenConns, maxIdleConns, time.Hour)

	// 🔥 注意：SQLite 不支持 FIND_IN_SET 函数
	// 我们已经在 query1.go 中使用 SQLite 兼容的方式（instr 函数）来实现相同功能
	// 所以不需要在这里注册自定义函数

	// 缓存连接
	dbs[dbName] = db
	logger.Infof(context.Background(), "数据库连接已创建: %s", dbName)

	return db, nil
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
