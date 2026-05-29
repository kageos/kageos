package model

import (
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type SystemSetting struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Key       string         `json:"key" gorm:"column:key;type:varchar(128);uniqueIndex;not null"`
	Value     string         `json:"value" gorm:"column:value;type:text"`
	UpdatedBy string         `json:"updated_by" gorm:"column:updated_by;type:varchar(255)"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}
