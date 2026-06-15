package model

import "github.com/kageos/kageos/pkg/gormx/models"

// AppDatabase maps one workspace package directory to one low-privilege MySQL
// database. The encrypted password is runtime-owned; SDK receives it only via a
// short-lived resolve flow.
type AppDatabase struct {
	models.Base
	User                        string `gorm:"size:100;not null;index:idx_app_db_scope,unique" json:"user"`
	App                         string `gorm:"size:100;not null;index:idx_app_db_scope,unique" json:"app"`
	PackagePath                 string `gorm:"size:512;not null;index:idx_app_db_scope,unique" json:"package_path"`
	FullCodePath                string `gorm:"size:768;not null;index" json:"full_code_path"`
	ClusterKey                  string `gorm:"size:64;not null;default:'default';index" json:"cluster_key"`
	DatabaseName                string `gorm:"size:64;not null;uniqueIndex" json:"database_name"`
	DatabaseUser                string `gorm:"size:32;not null;uniqueIndex" json:"database_user"`
	PasswordCiphertext          string `gorm:"type:text;not null" json:"-"`
	PasswordNonce               string `gorm:"size:128;not null" json:"-"`
	MigrationDatabaseUser       string `gorm:"size:32;uniqueIndex" json:"migration_database_user"`
	MigrationPasswordCiphertext string `gorm:"type:text" json:"-"`
	MigrationPasswordNonce      string `gorm:"size:128" json:"-"`
	Dialect                     string `gorm:"size:32;not null;default:'mysql'" json:"dialect"`
	Status                      string `gorm:"size:32;not null;default:'active';index" json:"status"`
}

func (AppDatabase) TableName() string {
	return "app_databases"
}
