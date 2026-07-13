package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
	"gorm.io/gorm"
)

const (
	appDBCapabilityTTL = 15 * time.Minute
	appDBStatusActive  = "active"
	appDBStatusPending = "pending"
	appDBRootPackage   = "_root"

	appDBRuntimePrivileges   = "SELECT, INSERT, UPDATE"
	appDBMigrationPrivileges = "SELECT, CREATE, ALTER, INDEX"
)

var ErrAppDatabaseDisabled = errors.New("app database is disabled")

// AppDatabaseService owns runtime-managed per-package application databases.
// Admin credentials never leave app-runtime; SDK apps receive only one
// package-scoped low-privilege DSN through a short-lived capability flow.
type AppDatabaseService struct {
	db     *gorm.DB
	cfg    appconfig.AppDatabaseConfig
	secret []byte

	mu       sync.Mutex
	keyLocks map[string]*sync.Mutex
}

type appDatabasePasswords struct {
	runtime   string
	migration string
}

func NewAppDatabaseService(db *gorm.DB, cfg appconfig.AppDatabaseConfig) (*AppDatabaseService, error) {
	cfg = cfg.WithDefaults()
	s := &AppDatabaseService{
		db:       db,
		cfg:      cfg,
		keyLocks: make(map[string]*sync.Mutex),
	}
	if !cfg.Enabled {
		return s, nil
	}
	if strings.ToLower(cfg.Dialect) != "mysql" {
		return nil, fmt.Errorf("unsupported app database dialect: %s", cfg.Dialect)
	}
	if strings.TrimSpace(cfg.AdminUser) == "" || strings.TrimSpace(cfg.AdminPassword) == "" {
		return nil, fmt.Errorf("app database admin credentials are required when app_database.enabled=true")
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("app_database.secret_key or KAGEOS_APP_DB_SECRET_KEY is required when app_database.enabled=true")
	}
	sum := sha256.Sum256([]byte(cfg.SecretKey))
	s.secret = sum[:]
	return s, nil
}

func (s *AppDatabaseService) IsEnabled() bool {
	return s != nil && s.cfg.Enabled
}
