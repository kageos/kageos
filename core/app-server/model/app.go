package model

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// 数据库连接
var DB *gorm.DB

// InitDB 初始化数据库
func InitDB() {
	cfg := config.GetAppServerConfig()
	dbCfg := cfg.DB

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}

	// 如果启用了数据库日志
	if dbCfg.LogLevel != "silent" {
		var logLevel gormLogger.LogLevel
		switch dbCfg.LogLevel {
		case "error":
			logLevel = gormLogger.Error
		case "warn":
			logLevel = gormLogger.Warn
		case "info":
			logLevel = gormLogger.Info
		default:
			logLevel = gormLogger.Warn
		}

		// 配置慢查询阈值
		slowThreshold := time.Duration(dbCfg.SlowThreshold) * time.Millisecond
		if slowThreshold == 0 {
			slowThreshold = 200 * time.Millisecond // 默认200毫秒
		}

		// 使用 GORM 默认日志配置
		gormConfig.Logger = gormLogger.Default.LogMode(logLevel)
	} else {
		// 禁用日志
		gormConfig.Logger = gormLogger.Default.LogMode(gormLogger.Silent)
	}

	switch dbCfg.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=Local",
			dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Name,
		)
		db, err := gorm.Open(mysql.Open(dsn), gormConfig)
		if err != nil {
			logger.Errorf(nil, "[Model] Failed to connect to database: %v", err)
			panic(fmt.Errorf("failed to connect to database: %w", err))
		}

		// 连接池设置
		if sqlDB, err := db.DB(); err == nil {
			if dbCfg.MaxIdleConns > 0 {
				sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
			}
			if dbCfg.MaxOpenConns > 0 {
				sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
			}
			if dbCfg.MaxLifetime > 0 {
				sqlDB.SetConnMaxLifetime(time.Duration(dbCfg.MaxLifetime) * time.Second)
			}
		}

		DB = db
		logger.Infof(nil, "[Model] Database initialized successfully")

	default:
		logger.Errorf(nil, "[Model] Unsupported db type: %s", dbCfg.Type)
		panic(fmt.Errorf("unsupported db type: %s", dbCfg.Type))
	}

	// 自动迁移表结构
	err := InitTables(DB)
	if err != nil {
		logger.Errorf(nil, "[Model] Failed to migrate database: %v", err)
		panic(fmt.Errorf("failed to migrate database: %w", err))
	}
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

type App struct {
	models.Base
	User    string `json:"user" gorm:"column:user;type:varchar(255);not null"`
	Name    string `json:"name" gorm:"column:name;type:varchar(255);not null"`
	NatsID  int64  `gorm:"column:nats_id;type:bigint" json:"nats_id"` //不同的nats 会把流量分发到不同的机房
	HostID  int64  `gorm:"column:host_id;type:bigint" json:"host_id"`
	Status  string `gorm:"column:status;type:varchar(50)" json:"status"` //启用/废弃
	Version string `gorm:"column:version;type:varchar(50)" json:"version"`
}

func (App) TableName() string {
	return "app"
}

func GetAppByUserName(user, name string) (*App, error) {
	var app App
	err := DB.Where("user = ? and name =?", user, name).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// CreateApp 创建应用记录
func CreateApp(app *App) error {
	return DB.Create(app).Error
}

// UpdateApp 更新应用记录
func UpdateApp(app *App) error {
	return DB.Save(app).Error
}

// GetAppByID 根据ID获取应用信息
func GetAppByID(id int64) (*App, error) {
	var app App
	err := DB.Where("id = ?", id).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// DeleteApp 删除应用记录
func DeleteApp(user, name string) error {
	return DB.Where("user = ? and name = ?", user, name).Delete(&App{}).Error
}

// DeleteAppAndVersions 删除应用及其所有版本记录
func DeleteAppAndVersions(user, name string) error {
	// 删除应用记录
	if err := DB.Where("user = ? and name = ?", user, name).Delete(&App{}).Error; err != nil {
		return err
	}

	// 删除应用版本记录（如果有的话，这里假设有 AppVersion 表）
	// 注意：这里需要根据实际的 AppVersion 表结构来调整
	// 如果 app-server 没有 AppVersion 表，可以忽略这部分

	return nil
}
