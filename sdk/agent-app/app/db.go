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

func getGormDB() *gorm.DB {
	db, err := getOrInitDB(getDBName())
	if err != nil {
		return nil
	}
	return db
}
func (c *Context) GetGormDB() *gorm.DB {
	db, err := getOrInitDB(getDBName())
	if err != nil {
		return nil
	}
	return db
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

	// 设置GORM日志配置
	gormLogger := gormLogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogger.Warn,
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

	// 设置SQLite优化参数（恢复 WAL 模式以提升并发读写性能）
	db.Exec("PRAGMA journal_mode=WAL;PRAGMA temp_store=MEMORY;PRAGMA synchronous=NORMAL;")

	// 设置连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf(context.Background(), "获取原生数据库连接失败: %v", err)
		return nil, err
	}

	sqlDB.SetMaxOpenConns(5)            // 增加连接数，支持多协程
	sqlDB.SetMaxIdleConns(2)            // 保持一些空闲连接
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最长生命周期

	// 缓存连接
	dbs[dbName] = db
	logger.Infof(context.Background(), "数据库连接已创建: %s", dbName)

	return db, nil
}
