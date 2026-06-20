package model

import (
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type AuthLoginProvider struct {
	ID               int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt        models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
	Code             string         `json:"code" gorm:"column:code;type:varchar(64);uniqueIndex;not null"`
	Name             string         `json:"name" gorm:"column:name;type:varchar(128);not null"`
	Description      string         `json:"description" gorm:"column:description;type:text"`
	Action           string         `json:"action" gorm:"column:action;type:varchar(32);not null"`
	Enabled          bool           `json:"enabled" gorm:"column:enabled;not null;default:false"`
	Configured       bool           `json:"configured" gorm:"column:configured;not null;default:false"`
	Status           string         `json:"status" gorm:"column:status;type:varchar(32);not null;default:'unconfigured'"`
	ConfigSchemaJSON string         `json:"config_schema_json" gorm:"column:config_schema_json;type:text"`
	ConfigValuesJSON string         `json:"config_values_json" gorm:"column:config_values_json;type:text"`
	CallbackPath     string         `json:"callback_path" gorm:"column:callback_path;type:varchar(255)"`
	DocsURL          string         `json:"docs_url" gorm:"column:docs_url;type:varchar(512)"`
	SortOrder        int            `json:"sort_order" gorm:"column:sort_order;not null;default:0"`
	UpdatedBy        string         `json:"updated_by" gorm:"column:updated_by;type:varchar(255)"`
}

func (AuthLoginProvider) TableName() string {
	return "auth_login_providers"
}
