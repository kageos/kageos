package model

import (
	"errors"
	"strings"

	"github.com/kageos/kageos/pkg/openapitoken"
	"gorm.io/gorm"
)

// InitModels 初始化所有模型（自动迁移）
func InitModels(db *gorm.DB) error {
	// ⭐ 先创建被引用的表（父表），再创建引用它们的表（子表）
	// 这样可以确保外键约束能够正确创建
	err := db.AutoMigrate(
		// 第一层：基础表（不被其他表引用）
		&Company{},
		&SystemSetting{},
		&AuthLoginProvider{},
		&AuthOAuthState{},
		&AuthOAuthRegistrationIntent{},
		&User{}, // 被 UserSession、EmailVerification 引用

		// 第二层：依赖 User 的表
		&AuthExternalIdentity{},
		&UserSession{},       // 引用 User
		&EmailVerification{}, // 引用 User
		&EmailCode{},         // 不引用其他表，但依赖 User 存在
		&openapitoken.OpenAPIToken{},

		// 第三层：部门表（自引用）
		&Department{}, // 自引用（ParentID -> ID）
	)
	if err != nil {
		return err
	}

	if err := ensureCompanyLogoColumn(db); err != nil {
		return err
	}
	if err := ensureUserEmailContactIndex(db); err != nil {
		return err
	}
	if err := ensureOAuthRegistrationIntentEmailNullable(db); err != nil {
		return err
	}

	return initDefaultCompany(db)
}

func ensureCompanyLogoColumn(db *gorm.DB) error {
	if db.Migrator().HasColumn(&Company{}, "LogoURL") {
		return db.Migrator().AlterColumn(&Company{}, "LogoURL")
	}
	return nil
}

func ensureUserEmailContactIndex(db *gorm.DB) error {
	migrator := db.Migrator()
	indexes, err := migrator.GetIndexes(&User{})
	if err != nil {
		return err
	}
	for _, index := range indexes {
		unique, ok := index.Unique()
		if !ok || !unique {
			continue
		}
		columns := index.Columns()
		if len(columns) == 1 && strings.EqualFold(columns[0], "email") {
			if err := migrator.DropIndex(&User{}, index.Name()); err != nil {
				return err
			}
		}
	}
	if migrator.HasIndex(&User{}, "idx_user_email") {
		return nil
	}
	return migrator.CreateIndex(&User{}, "idx_user_email")
}

func ensureOAuthRegistrationIntentEmailNullable(db *gorm.DB) error {
	if db.Migrator().HasColumn(&AuthOAuthRegistrationIntent{}, "Email") {
		return db.Migrator().AlterColumn(&AuthOAuthRegistrationIntent{}, "Email")
	}
	return nil
}

func initDefaultCompany(db *gorm.DB) error {
	company := &Company{}
	err := db.Where("code = ?", DefaultCompanyCode).First(company).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		company = &Company{
			Code:      DefaultCompanyCode,
			Name:      "Default",
			CreatedBy: "system",
			LogoURL:   "",
		}
		if err := db.Create(company).Error; err != nil {
			return err
		}
	}
	return db.Model(&User{}).Where("company_code = '' OR company_code IS NULL").Update("company_code", DefaultCompanyCode).Error
}
