package model

import "gorm.io/gorm"

func InitModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&MessageEntry{},
		&MessageRecipient{},
	)
}
