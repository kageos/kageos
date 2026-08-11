package model

import (
	"time"

	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type AuthWechatLoginAttempt struct {
	ID            int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt     models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	TokenHash     string         `json:"-" gorm:"column:token_hash;type:varchar(64);uniqueIndex;not null"`
	SceneHash     string         `json:"-" gorm:"column:scene_hash;type:varchar(64);uniqueIndex;not null"`
	ProviderCode  string         `json:"provider_code" gorm:"column:provider_code;type:varchar(64);index;not null"`
	RedirectAfter string         `json:"redirect_after" gorm:"column:redirect_after;type:varchar(1000)"`
	ExternalID    string         `json:"external_id" gorm:"column:external_id;type:varchar(255)"`
	Nickname      string         `json:"nickname" gorm:"column:nickname;type:varchar(255)"`
	Avatar        string         `json:"avatar" gorm:"column:avatar;type:varchar(500)"`
	ExpiresAt     time.Time      `json:"expires_at" gorm:"column:expires_at;index;not null"`
	ScannedAt     *time.Time     `json:"scanned_at" gorm:"column:scanned_at;index"`
	UsedAt        *time.Time     `json:"used_at" gorm:"column:used_at;index"`
}

func (AuthWechatLoginAttempt) TableName() string {
	return "auth_wechat_login_attempts"
}
