package model

import (
	"time"

	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type AuthOAuthRegistrationIntent struct {
	ID                  int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt           models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
	Ticket              string         `json:"ticket" gorm:"column:ticket;type:varchar(128);uniqueIndex;not null"`
	ProviderCode        string         `json:"provider_code" gorm:"column:provider_code;type:varchar(64);index;not null"`
	ExternalID          string         `json:"external_id" gorm:"column:external_id;type:varchar(255);index;not null"`
	Email               string         `json:"email" gorm:"column:email;type:varchar(255);index"`
	EmailVerified       bool           `json:"email_verified" gorm:"column:email_verified;type:boolean;default:false"`
	Nickname            string         `json:"nickname" gorm:"column:nickname;type:varchar(255)"`
	Avatar              string         `json:"avatar" gorm:"column:avatar;type:varchar(500)"`
	SuggestedCode       string         `json:"suggested_code" gorm:"column:suggested_code;type:varchar(64)"`
	CodeSuggestionsJSON string         `json:"code_suggestions_json" gorm:"column:code_suggestions_json;type:text"`
	RedirectAfter       string         `json:"redirect_after" gorm:"column:redirect_after;type:varchar(1000)"`
	ExpiresAt           time.Time      `json:"expires_at" gorm:"column:expires_at;index;not null"`
	UsedAt              *time.Time     `json:"used_at" gorm:"column:used_at;index"`
}

func (AuthOAuthRegistrationIntent) TableName() string {
	return "auth_oauth_registration_intents"
}
