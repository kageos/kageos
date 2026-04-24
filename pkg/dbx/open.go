package dbx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type OpenOptions struct {
	LogMode                                  gormlogger.LogLevel
	DisableForeignKeyConstraintWhenMigrating bool
	DefaultMaxOpenConns                      int
	DefaultMaxIdleConns                      int
	DefaultMaxLifetime                       time.Duration
}

func OpenMySQL(cfg appconfig.DBConfig, opts OpenOptions) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := gorm.Open(mysql.Open(dsn), buildGORMConfig(opts))
	if err != nil {
		return nil, err
	}
	if err := applyPoolConfig(db, cfg, opts); err != nil {
		return nil, err
	}
	return db, nil
}

func EnsureMySQLDatabase(cfg appconfig.DBConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("database name is required")
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	return db.Exec("CREATE DATABASE IF NOT EXISTS " + quoteMySQLIdentifier(cfg.Name) + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
}

func OpenSQLite(path string, opts OpenOptions) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return gorm.Open(sqlite.Open(path), buildGORMConfig(opts))
}

func buildGORMConfig(opts OpenOptions) *gorm.Config {
	logMode := opts.LogMode
	if logMode == 0 {
		logMode = gormlogger.Info
	}
	return &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: opts.DisableForeignKeyConstraintWhenMigrating,
		Logger:                                   gormlogger.Default.LogMode(logMode),
	}
}

func applyPoolConfig(db *gorm.DB, cfg appconfig.DBConfig, opts OpenOptions) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = opts.DefaultMaxOpenConns
	}
	if maxOpenConns <= 0 {
		maxOpenConns = 100
	}

	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = opts.DefaultMaxIdleConns
	}
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}

	maxLifetime := time.Duration(cfg.MaxLifetime) * time.Second
	if maxLifetime <= 0 {
		maxLifetime = opts.DefaultMaxLifetime
	}
	if maxLifetime <= 0 {
		maxLifetime = 5 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(maxLifetime)
	return nil
}

func quoteMySQLIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
