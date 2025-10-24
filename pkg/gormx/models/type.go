package models

import "gorm.io/gorm"

type Base struct {
	ID        int64          `json:"id" gorm:"primary_key"`
	CreatedAt Time           `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt Time           `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy string         `json:"created_by" gorm:"column:created_by"`
	UpdatedBy string         `json:"updated_by" gorm:"column:updated_by"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by"`
}
