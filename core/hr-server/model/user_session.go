package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

// UserSession 用户会话记录
type UserSession struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UserID       int64       `json:"user_id" gorm:"column:user_id;type:bigint;not null"`
	Token        string      `json:"token" gorm:"column:token;type:varchar(500);uniqueIndex;not null"`
	RefreshToken string      `json:"refresh_token" gorm:"column:refresh_token;type:varchar(500);uniqueIndex;not null"`
	ExpiresAt    models.Time `json:"expires_at" gorm:"column:expires_at;type:datetime;not null"`
	UserAgent    string      `json:"user_agent" gorm:"column:user_agent;type:varchar(500)"`
	IPAddress    string      `json:"ip_address" gorm:"column:ip_address;type:varchar(45)"`
	IsActive     bool        `json:"is_active" gorm:"column:is_active;type:boolean;default:true"`
}

func (UserSession) TableName() string {
	return "user_session"
}

