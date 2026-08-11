package model

import (
	"time"

	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type AuthOAuthState struct {
	ID            int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt     models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	State         string         `json:"state" gorm:"column:state;type:varchar(128);uniqueIndex;not null"`
	ProviderCode  string         `json:"provider_code" gorm:"column:provider_code;type:varchar(64);index;not null"`
	RedirectAfter string         `json:"redirect_after" gorm:"column:redirect_after;type:varchar(1000)"`
	PKCEVerifier  string         `json:"-" gorm:"column:pkce_verifier;type:varchar(128)"`
	ExpiresAt     time.Time      `json:"expires_at" gorm:"column:expires_at;index;not null"`
	UsedAt        *time.Time     `json:"used_at" gorm:"column:used_at;index"`
}

func (AuthOAuthState) TableName() string {
	return "auth_oauth_states"
}
