package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/kageos/kageos/core/app-runtime/model"
	"github.com/kageos/kageos/dto"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	appDBCapabilityTTL = 15 * time.Minute
	appDBStatusActive  = "active"
	appDBStatusPending = "pending"
	appDBRootPackage   = "_root"

	appDBRuntimePrivileges   = "SELECT, INSERT, UPDATE"
	appDBMigrationPrivileges = "SELECT, CREATE, ALTER, INDEX, REFERENCES"
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

func (s *AppDatabaseService) IssueCapability(user, app, version, router string) (*dto.AppDBCapability, error) {
	if !s.IsEnabled() {
		return nil, nil
	}
	nonce, err := randomToken(18)
	if err != nil {
		return nil, err
	}
	capability := &dto.AppDBCapability{
		User:      user,
		App:       app,
		Version:   version,
		Router:    router,
		ExpiresAt: time.Now().Add(appDBCapabilityTTL).Unix(),
		Nonce:     nonce,
	}
	capability.Signature = s.signCapability(capability)
	return capability, nil
}

func (s *AppDatabaseService) Resolve(ctx context.Context, req *dto.AppDBResolveReq) (*dto.AppDBResolveResp, error) {
	if !s.IsEnabled() {
		return nil, ErrAppDatabaseDisabled
	}
	if req == nil {
		return nil, fmt.Errorf("resolve request is nil")
	}
	access, err := normalizeAppDBAccess(req.Access)
	if err != nil {
		return nil, err
	}
	if err := s.validateCapability(req); err != nil {
		return nil, err
	}

	packagePath := cleanPackagePath(req.PackagePath)
	if packagePath == "" {
		packagePath = appDBRootPackage
	}

	record, passwords, err := s.ensurePackageDatabase(ctx, req.User, req.App, packagePath)
	if err != nil {
		return nil, err
	}

	databaseUser := record.DatabaseUser
	password := passwords.runtime
	if access == dto.AppDBAccessMigration {
		databaseUser = record.MigrationDatabaseUser
		password = passwords.migration
	}
	if strings.TrimSpace(databaseUser) == "" || strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("app database %s credentials are unavailable", access)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		databaseUser, password, s.cfg.Host, s.cfg.Port, record.DatabaseName)
	return &dto.AppDBResolveResp{
		Dialect:      "mysql",
		Access:       access,
		DatabaseName: record.DatabaseName,
		DSN:          dsn,
		MaxOpenConns: s.cfg.MaxOpenConns,
		MaxIdleConns: s.cfg.MaxIdleConns,
		MaxIdleTime:  s.cfg.MaxIdleTime,
		MaxLifetime:  s.cfg.MaxLifetime,
	}, nil
}

func (s *AppDatabaseService) EnsureDatabaseForPackage(ctx context.Context, user, app, packagePath string) error {
	if !s.IsEnabled() {
		return nil
	}
	_, _, err := s.ensurePackageDatabase(ctx, user, app, packagePath)
	return err
}

func (s *AppDatabaseService) ensurePackageDatabase(ctx context.Context, user, app, packagePath string) (*model.AppDatabase, appDatabasePasswords, error) {
	packagePath = cleanPackagePath(packagePath)
	lock := s.lockFor(user + "/" + app + "/" + packagePath)
	lock.Lock()
	defer lock.Unlock()

	var record model.AppDatabase
	err := s.db.Where("user = ? AND app = ? AND package_path = ?", user, app, packagePath).First(&record).Error
	if err == nil {
		if record.ClusterKey == "" {
			record.ClusterKey = s.cfg.ClusterKey
			if saveErr := s.db.Save(&record).Error; saveErr != nil {
				return nil, appDatabasePasswords{}, saveErr
			}
		}
		if record.ClusterKey != s.cfg.ClusterKey {
			return nil, appDatabasePasswords{}, fmt.Errorf("app database package belongs to cluster %s, current runtime cluster is %s", record.ClusterKey, s.cfg.ClusterKey)
		}
		passwords, prepareErr := s.prepareDatabaseCredentials(&record)
		if prepareErr != nil {
			return nil, appDatabasePasswords{}, prepareErr
		}
		if record.Status == appDBStatusActive {
			if err := s.provisionMySQL(ctx, &record, passwords); err != nil {
				return nil, appDatabasePasswords{}, err
			}
			return &record, passwords, nil
		}
		return s.activatePendingPackageDatabase(ctx, &record, passwords)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appDatabasePasswords{}, err
	}

	password, err := randomToken(32)
	if err != nil {
		return nil, appDatabasePasswords{}, err
	}
	migrationPassword, err := randomToken(32)
	if err != nil {
		return nil, appDatabasePasswords{}, err
	}
	ciphertext, nonce, err := s.encryptPassword(password)
	if err != nil {
		return nil, appDatabasePasswords{}, err
	}
	migrationCiphertext, migrationNonce, err := s.encryptPassword(migrationPassword)
	if err != nil {
		return nil, appDatabasePasswords{}, err
	}
	tmpSuffix, err := randomHex(6)
	if err != nil {
		return nil, appDatabasePasswords{}, err
	}

	record = model.AppDatabase{
		User:                        user,
		App:                         app,
		PackagePath:                 packagePath,
		FullCodePath:                "/" + path.Join(user, app, packagePath),
		ClusterKey:                  s.cfg.ClusterKey,
		DatabaseName:                "pending_" + tmpSuffix,
		DatabaseUser:                "pending_" + tmpSuffix,
		PasswordCiphertext:          ciphertext,
		PasswordNonce:               nonce,
		MigrationDatabaseUser:       "pending_m_" + tmpSuffix,
		MigrationPasswordCiphertext: migrationCiphertext,
		MigrationPasswordNonce:      migrationNonce,
		Dialect:                     "mysql",
		Status:                      appDBStatusPending,
	}
	if err := s.db.Create(&record).Error; err != nil {
		return nil, appDatabasePasswords{}, err
	}

	return s.activatePendingPackageDatabase(ctx, &record, appDatabasePasswords{
		runtime:   password,
		migration: migrationPassword,
	})
}

func (s *AppDatabaseService) activatePendingPackageDatabase(ctx context.Context, record *model.AppDatabase, passwords appDatabasePasswords) (*model.AppDatabase, appDatabasePasswords, error) {
	suffix := base62Encode(uint64(record.ID))
	record.DatabaseName = s.cfg.DatabasePrefix + suffix
	record.DatabaseUser = runtimeDatabaseUserName(s.cfg.UserPrefix, suffix)
	record.MigrationDatabaseUser = migrationDatabaseUserName(s.cfg.UserPrefix, suffix)
	if err := s.provisionMySQL(ctx, record, passwords); err != nil {
		return nil, appDatabasePasswords{}, err
	}
	record.Status = appDBStatusActive
	if err := s.db.Save(&record).Error; err != nil {
		return nil, appDatabasePasswords{}, err
	}
	logger.Infof(ctx, "[AppDatabase] provisioned package DB: %s/%s/%s -> %s", record.User, record.App, record.PackagePath, record.DatabaseName)
	return record, passwords, nil
}

func (s *AppDatabaseService) provisionMySQL(ctx context.Context, record *model.AppDatabase, passwords appDatabasePasswords) error {
	db, err := s.openAdminDB()
	if err != nil {
		return err
	}
	defer closeGORM(db)

	if record == nil {
		return fmt.Errorf("app database record is nil")
	}
	databaseName := strings.TrimSpace(record.DatabaseName)
	if databaseName == "" {
		return fmt.Errorf("app database name is empty")
	}
	if err := db.Exec("CREATE DATABASE IF NOT EXISTS " + quoteMySQLIdentifier(databaseName) + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		return fmt.Errorf("create app database: %w", err)
	}

	if err := s.provisionMySQLUser(db, databaseName, record.DatabaseUser, passwords.runtime, appDBRuntimePrivileges); err != nil {
		return fmt.Errorf("provision runtime app database user: %w", err)
	}
	if err := s.provisionMySQLUser(db, databaseName, record.MigrationDatabaseUser, passwords.migration, appDBMigrationPrivileges); err != nil {
		return fmt.Errorf("provision migration app database user: %w", err)
	}
	return nil
}

func (s *AppDatabaseService) provisionMySQLUser(db *gorm.DB, databaseName, databaseUser, password, privileges string) error {
	if db == nil {
		return fmt.Errorf("admin db is nil")
	}
	if strings.TrimSpace(databaseUser) == "" || strings.TrimSpace(password) == "" {
		return fmt.Errorf("database user credentials are empty")
	}
	user := quoteMySQLString(databaseUser)
	host := quoteMySQLString(s.cfg.GrantHost)
	pass := quoteMySQLString(password)
	principal := user + "@" + host
	if err := db.Exec("CREATE USER IF NOT EXISTS " + principal + " IDENTIFIED BY " + pass).Error; err != nil {
		return fmt.Errorf("create user %s: %w", databaseUser, err)
	}
	if err := db.Exec("ALTER USER " + principal + " IDENTIFIED BY " + pass).Error; err != nil {
		return fmt.Errorf("set password for user %s: %w", databaseUser, err)
	}
	if err := db.Exec("REVOKE ALL PRIVILEGES, GRANT OPTION FROM " + principal).Error; err != nil && !isNoSuchGrantError(err) {
		return fmt.Errorf("revoke existing privileges for user %s: %w", databaseUser, err)
	}
	if err := db.Exec("GRANT " + privileges + " ON " + quoteMySQLIdentifier(databaseName) + ".* TO " + principal).Error; err != nil {
		return fmt.Errorf("grant %s to user %s: %w", privileges, databaseUser, err)
	}
	return nil
}

func (s *AppDatabaseService) openAdminDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		s.cfg.AdminUser, s.cfg.AdminPassword, s.cfg.Host, s.cfg.Port)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
}

func (s *AppDatabaseService) prepareDatabaseCredentials(record *model.AppDatabase) (appDatabasePasswords, error) {
	if record == nil {
		return appDatabasePasswords{}, fmt.Errorf("app database record is nil")
	}
	changed := false
	if strings.TrimSpace(record.DatabaseUser) == "" && record.ID > 0 {
		record.DatabaseUser = runtimeDatabaseUserName(s.cfg.UserPrefix, base62Encode(uint64(record.ID)))
		changed = true
	}
	runtimePasswordMissing := strings.TrimSpace(record.PasswordCiphertext) == "" || strings.TrimSpace(record.PasswordNonce) == ""
	runtimePassword, err := s.ensureEncryptedPassword(&record.PasswordCiphertext, &record.PasswordNonce)
	if err != nil {
		return appDatabasePasswords{}, err
	}
	if strings.TrimSpace(record.MigrationDatabaseUser) == "" && record.ID > 0 {
		record.MigrationDatabaseUser = migrationDatabaseUserName(s.cfg.UserPrefix, base62Encode(uint64(record.ID)))
		changed = true
	}
	migrationPasswordMissing := strings.TrimSpace(record.MigrationPasswordCiphertext) == "" || strings.TrimSpace(record.MigrationPasswordNonce) == ""
	migrationPassword, err := s.ensureEncryptedPassword(&record.MigrationPasswordCiphertext, &record.MigrationPasswordNonce)
	if err != nil {
		return appDatabasePasswords{}, err
	}
	if changed || runtimePasswordMissing || migrationPasswordMissing {
		if err := s.db.Save(record).Error; err != nil {
			return appDatabasePasswords{}, err
		}
	}
	return appDatabasePasswords{runtime: runtimePassword, migration: migrationPassword}, nil
}

func (s *AppDatabaseService) ensureEncryptedPassword(ciphertext, nonce *string) (string, error) {
	if ciphertext == nil || nonce == nil {
		return "", fmt.Errorf("password fields are nil")
	}
	if strings.TrimSpace(*ciphertext) != "" && strings.TrimSpace(*nonce) != "" {
		return s.decryptPassword(*ciphertext, *nonce)
	}
	password, err := randomToken(32)
	if err != nil {
		return "", err
	}
	encrypted, nonceText, err := s.encryptPassword(password)
	if err != nil {
		return "", err
	}
	*ciphertext = encrypted
	*nonce = nonceText
	return password, nil
}

func (s *AppDatabaseService) validateCapability(req *dto.AppDBResolveReq) error {
	capability := req.Capability
	if capability == nil {
		return fmt.Errorf("missing app database capability")
	}
	if capability.User != req.User || capability.App != req.App {
		return fmt.Errorf("app database capability scope mismatch")
	}
	if capability.ExpiresAt <= time.Now().Unix() {
		return fmt.Errorf("app database capability expired")
	}
	if !hmac.Equal([]byte(capability.Signature), []byte(s.signCapability(capability))) {
		return fmt.Errorf("invalid app database capability signature")
	}
	access, err := normalizeAppDBAccess(req.Access)
	if err != nil {
		return err
	}
	if access == dto.AppDBAccessMigration && capability.Router != "" {
		return fmt.Errorf("app database migration access requires lifecycle capability")
	}
	if capability.Router != "" {
		expectedPackagePath := packagePathFromRouter(capability.Router)
		actualPackagePath := cleanPackagePath(req.PackagePath)
		if actualPackagePath == "" {
			actualPackagePath = appDBRootPackage
		}
		if expectedPackagePath != actualPackagePath {
			return fmt.Errorf("app database capability package mismatch")
		}
	}
	return nil
}

func (s *AppDatabaseService) signCapability(capability *dto.AppDBCapability) string {
	copy := *capability
	copy.Signature = ""
	payload, _ := json.Marshal(copy)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *AppDatabaseService) encryptPassword(password string) (string, string, error) {
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(password), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), base64.RawURLEncoding.EncodeToString(nonce), nil
}

func (s *AppDatabaseService) decryptPassword(ciphertextText, nonceText string) (string, error) {
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(ciphertextText)
	if err != nil {
		return "", err
	}
	nonce, err := base64.RawURLEncoding.DecodeString(nonceText)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *AppDatabaseService) lockFor(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := s.keyLocks[key]
	if lock == nil {
		lock = &sync.Mutex{}
		s.keyLocks[key] = lock
	}
	return lock
}

func cleanPackagePath(packagePath string) string {
	packagePath = strings.TrimSpace(packagePath)
	packagePath = strings.Trim(packagePath, "/")
	if packagePath == "" {
		return ""
	}
	return path.Clean(packagePath)
}

func normalizeAppDBAccess(access string) (string, error) {
	access = strings.TrimSpace(strings.ToLower(access))
	if access == "" {
		return dto.AppDBAccessRuntime, nil
	}
	switch access {
	case dto.AppDBAccessRuntime, dto.AppDBAccessMigration:
		return access, nil
	default:
		return "", fmt.Errorf("unsupported app database access: %s", access)
	}
}

func runtimeDatabaseUserName(prefix, suffix string) string {
	return prefix + suffix
}

func migrationDatabaseUserName(prefix, suffix string) string {
	return prefix + "m_" + suffix
}

func packagePathFromRouter(router string) string {
	router = strings.TrimSpace(router)
	router = strings.Trim(router, "/")
	if router == "" {
		return appDBRootPackage
	}
	router = path.Clean(router)
	idx := strings.LastIndex(router, "/")
	if idx <= 0 {
		return appDBRootPackage
	}
	return router[:idx]
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func base62Encode(n uint64) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append([]byte{alphabet[n%62]}, buf...)
		n /= 62
	}
	return string(buf)
}

func quoteMySQLIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quoteMySQLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func isNoSuchGrantError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such grant") || strings.Contains(text, "there is no such grant")
}

func closeGORM(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}
