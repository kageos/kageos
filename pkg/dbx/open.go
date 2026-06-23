package dbx

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	appconfig "github.com/kageos/kageos/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type OpenOptions struct {
	LogMode                                  gormlogger.LogLevel
	SlowThreshold                           time.Duration
	DisableForeignKeyConstraintWhenMigrating bool
	DefaultMaxOpenConns                      int
	DefaultMaxIdleConns                      int
	DefaultMaxLifetime                       time.Duration
	ConnectRetryTimeout                      time.Duration
	ConnectRetryInterval                     time.Duration
}

func OpenMySQL(cfg appconfig.DBConfig, opts OpenOptions) (*gorm.DB, error) {
	opts = applyDBLogOptions(cfg, opts)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	var db *gorm.DB
	err := withMySQLConnectRetry(opts, func() error {
		candidate, err := gorm.Open(mysql.Open(dsn), buildGORMConfig(opts))
		if err != nil {
			return err
		}
		if err := applyPoolConfig(candidate, cfg, opts); err != nil {
			closeGORMDB(candidate)
			return err
		}
		db = candidate
		return nil
	})
	return db, err
}

func EnsureMySQLDatabase(cfg appconfig.DBConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("database name is required")
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)
	return withMySQLConnectRetry(OpenOptions{}, func() error {
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		if err != nil {
			return err
		}
		defer closeGORMDB(db)
		return db.Exec("CREATE DATABASE IF NOT EXISTS " + quoteMySQLIdentifier(cfg.Name) + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
	})
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
		logMode = gormlogger.Warn
	}
	slowThreshold := opts.SlowThreshold
	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond
	}
	return &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: opts.DisableForeignKeyConstraintWhenMigrating,
		Logger: gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold: slowThreshold,
				LogLevel:      logMode,
				Colorful:      false,
			},
		),
	}
}

func applyDBLogOptions(cfg appconfig.DBConfig, opts OpenOptions) OpenOptions {
	if opts.LogMode == 0 {
		opts.LogMode = parseGORMLogLevel(cfg.LogLevel)
	}
	if opts.SlowThreshold <= 0 && cfg.SlowThreshold > 0 {
		opts.SlowThreshold = time.Duration(cfg.SlowThreshold) * time.Millisecond
	}
	return opts
}

func parseGORMLogLevel(level string) gormlogger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "silent", "off", "none", "disabled":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "warn", "warning", "":
		return gormlogger.Warn
	case "info", "debug":
		return gormlogger.Info
	default:
		return gormlogger.Warn
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

func withMySQLConnectRetry(opts OpenOptions, fn func() error) error {
	timeout := opts.ConnectRetryTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	interval := opts.ConnectRetryInterval
	if interval <= 0 {
		interval = time.Second
	}
	if timeout < 0 {
		return fn()
	}

	deadline := time.Now().Add(timeout)
	for {
		err := fn()
		if err == nil {
			return nil
		}
		if !isRetryableMySQLStartupError(err) || time.Now().After(deadline) {
			return err
		}
		sleep := interval
		if remaining := time.Until(deadline); remaining < sleep {
			sleep = remaining
		}
		if sleep <= 0 {
			return err
		}
		time.Sleep(sleep)
	}
}

func isRetryableMySQLStartupError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"bad connection",
		"connection refused",
		"connection reset",
		"connection timed out",
		"i/o timeout",
		"deadline exceeded",
		"no such host",
		"eof",
		"server has gone away",
		"unknown database",
	} {
		if strings.Contains(msg, fragment) {
			return true
		}
	}
	return false
}

func closeGORMDB(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func quoteMySQLIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
