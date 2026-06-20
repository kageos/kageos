package model

import (
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type AuthExternalIdentity struct {
	ID           int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt    models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	UserID       int64          `json:"user_id" gorm:"column:user_id;index;not null"`
	ProviderCode string         `json:"provider_code" gorm:"column:provider_code;type:varchar(64);uniqueIndex:idx_auth_external_identity_provider_subject;not null"`
	ExternalID   string         `json:"external_id" gorm:"column:external_id;type:varchar(255);uniqueIndex:idx_auth_external_identity_provider_subject;not null"`
	Email        string         `json:"email" gorm:"column:email;type:varchar(255);index"`
	Avatar       string         `json:"avatar" gorm:"column:avatar;type:varchar(500)"`
	Nickname     string         `json:"nickname" gorm:"column:nickname;type:varchar(255)"`
}

func (AuthExternalIdentity) TableName() string {
	return "auth_external_identities"
}
