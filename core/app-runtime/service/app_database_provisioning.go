package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/kageos/kageos/core/app-runtime/model"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func (s *AppDatabaseService) EnsureDatabaseForPackage(ctx context.Context, user, app, packagePath string) error {
	if !s.IsEnabled() {
		return nil
	}
	_, _, err := s.ensurePackageDatabase(ctx, user, app, packagePath)
	return err
}

func (s *AppDatabaseService) ensurePackageDatabase(ctx context.Context, user, app, packagePath string) (*model.AppDatabase, appDatabasePasswords, error) {
	packagePath, err := normalizeAppDBPackagePath(packagePath)
	if err != nil {
		return nil, appDatabasePasswords{}, err
	}
	lock := s.lockFor(user + "/" + app + "/" + packagePath)
	lock.Lock()
	defer lock.Unlock()

	var record model.AppDatabase
	err = s.db.Where("user = ? AND app = ? AND package_path = ?", user, app, packagePath).First(&record).Error
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
