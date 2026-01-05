package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

// EmailVerification 邮箱验证记录
type EmailVerification struct {
	ID        int64          `json:"id" gorm:"primary_key"`
	CreatedAt models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UserID    int64       `json:"user_id" gorm:"column:user_id;type:bigint;not null"`
	Email     string      `json:"email" gorm:"column:email;type:varchar(255);not null"`
	Token     string      `json:"token" gorm:"column:token;type:varchar(500);uniqueIndex;not null"`
	ExpiresAt models.Time `json:"expires_at" gorm:"column:expires_at;type:datetime;not null"`
	Used      bool        `json:"used" gorm:"column:used;type:boolean;default:false"`
	Type      string      `json:"type" gorm:"column:type;type:varchar(50);default:'register'"` // 验证类型: register(注册), reset_password(重置密码), change_email(更换邮箱), login(登录), bind_email(绑定邮箱)

	// 关联字段
	User *User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

func (EmailVerification) TableName() string {
	return "email_verification"
}

