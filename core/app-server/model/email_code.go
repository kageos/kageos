package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

// EmailCode 邮箱验证码记录
type EmailCode struct {
	ID        int64          `json:"id" gorm:"primary_key"`
	CreatedAt models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Email     string      `json:"email" gorm:"column:email;type:varchar(255);not null;index"`
	Code      string      `json:"code" gorm:"column:code;type:varchar(10);not null"`
	ExpiresAt models.Time `json:"expires_at" gorm:"column:expires_at;type:datetime;not null"`
	Used      bool        `json:"used" gorm:"column:used;type:boolean;default:false"`
	Type      string      `json:"type" gorm:"column:type;type:varchar(50);default:'register'"` // register, reset_password, change_email
	IPAddress string      `json:"ip_address" gorm:"column:ip_address;type:varchar(45)"`
	UserAgent string      `json:"user_agent" gorm:"column:user_agent;type:varchar(500)"`
}

func (EmailCode) TableName() string {
	return "email_code"
}
