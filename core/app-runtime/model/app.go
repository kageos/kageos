package model

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// 数据库连接
var DB *gorm.DB

// InitDB 初始化数据库
func InitDB() {
	ctx := context.Background()

	// 数据库文件路径 - 使用当前目录的 data/app-runtime 目录
	dbPath := filepath.Join("data", "app-runtime", "app_runtime.db")

	// 获取绝对路径
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		logger.Errorf(ctx, "[Model] Failed to get absolute path: %v", err)
		panic(fmt.Errorf("failed to get absolute path: %w", err))
	}

	// 确保目录存在
	dbDir := filepath.Dir(absPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		logger.Errorf(ctx, "[Model] Failed to create database directory: %v", err)
		panic(fmt.Errorf("failed to create database directory: %w", err))
	}

	logger.Infof(ctx, "[Model] Initializing SQLite database at: %s", absPath)

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(absPath), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		logger.Errorf(ctx, "[Model] Failed to connect to database: %v", err)
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	// 自动迁移表结构
	err = InitTables(db)
	if err != nil {
		logger.Errorf(ctx, "[Model] Failed to migrate database: %v", err)
		panic(fmt.Errorf("failed to migrate database: %w", err))
	}

	DB = db
}

// CloseDB 关闭数据库连接
func CloseDB() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// App 应用信息表
type App struct {
	models.Base
	User        string    `gorm:"size:100;not null;index" json:"user"`              // 用户名
	App         string    `gorm:"size:100;not null;index" json:"app"`               // 应用名
	Version     string    `gorm:"size:50;not null" json:"version"`                  // 当前版本
	Status      string    `gorm:"size:20;not null;default:'stopped'" json:"status"` // 状态：running, stopped, error
	ContainerID string    `gorm:"size:100" json:"container_id"`                     // 容器ID
	StartTime   time.Time `json:"start_time"`                                       // 启动时间
	LastSeen    time.Time `json:"last_seen"`                                        // 最后发现时间
}

// TableName 指定表名
func (App) TableName() string {
	return "apps"
}

// AppVersion 应用版本历史表
type AppVersion struct {
	models.Base
	User        string     `gorm:"size:100;not null;index" json:"user"`               // 用户名
	App         string     `gorm:"size:100;not null;index" json:"app"`                // 应用名
	Version     string     `gorm:"size:50;not null" json:"version"`                   // 版本号
	Status      string     `gorm:"size:20;not null;default:'inactive'" json:"status"` // 状态：active, inactive, running, stopped
	ContainerID string     `gorm:"size:100" json:"container_id"`                      // 容器ID
	ProcessID   int        `json:"process_id"`                                        // 进程ID
	StartTime   time.Time  `json:"start_time"`                                        // 该版本启动时间
	StopTime    *time.Time `json:"stop_time"`                                         // 停止时间
	LastSeen    time.Time  `json:"last_seen"`                                         // 最后发现时间
}

// TableName 指定表名
func (AppVersion) TableName() string {
	return "app_versions"
}

// GetKey 获取应用唯一标识
func (a *App) GetKey() string {
	return a.User + "/" + a.App
}

// IsRunning 检查应用是否正在运行
func (a *App) IsRunning() bool {
	return a.Status == "running"
}

// IsActive 检查版本是否激活
func (v *AppVersion) IsActive() bool {
	return v.Status == "active"
}

// IsRunning 检查版本是否正在运行
func (v *AppVersion) IsRunning() bool {
	return v.Status == "running"
}

// DeleteAppAndVersions 删除应用及其所有版本记录
func DeleteAppAndVersions(db *gorm.DB, user, app string) error {
	// 删除应用记录
	if err := db.Where("user = ? and app = ?", user, app).Delete(&App{}).Error; err != nil {
		return err
	}

	// 删除应用版本记录
	if err := db.Where("user = ? and app = ?", user, app).Delete(&AppVersion{}).Error; err != nil {
		return err
	}

	return nil
}

// CreateApp 创建应用记录
func CreateApp(user, app string) error {
	appRecord := &App{
		User:      user,
		App:       app,
		Version:   "v1", // 初始版本
		Status:    "stopped",
		StartTime: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := DB.Create(appRecord).Error; err != nil {
		return fmt.Errorf("failed to save app to database: %w", err)
	}

	// 创建初始版本记录
	versionRecord := &AppVersion{
		User:      user,
		App:       app,
		Version:   "v1",
		Status:    "inactive", // 初始状态为 inactive，第一次启动时变为 active
		StartTime: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := DB.Create(versionRecord).Error; err != nil {
		return fmt.Errorf("failed to save app version to database: %w", err)
	}

	return nil
}
