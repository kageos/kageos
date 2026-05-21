package model

import (
	"errors"

	"gorm.io/gorm"
)

// InitModels 初始化所有模型（自动迁移）
func InitModels(db *gorm.DB) error {
	// ⭐ 先创建被引用的表（父表），再创建引用它们的表（子表）
	// 这样可以确保外键约束能够正确创建
	err := db.AutoMigrate(
		// 第一层：基础表（不被其他表引用）
		&Company{},
		&User{}, // 被 UserSession、EmailVerification 引用

		// 第二层：依赖 User 的表
		&UserSession{},       // 引用 User
		&EmailVerification{}, // 引用 User
		&EmailCode{},         // 不引用其他表，但依赖 User 存在

		// 第三层：部门表（自引用）
		&Department{}, // 自引用（ParentID -> ID）
	)
	if err != nil {
		return err
	}

	if err := ensureCompanyLogoColumn(db); err != nil {
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
