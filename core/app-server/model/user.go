package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

type User struct {
	ID            int64          `json:"id" gorm:"primary_key"`
	CreatedAt     models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy     string         `json:"created_by" gorm:"column:created_by;type:varchar(255)"`
	Username      string         `json:"username" gorm:"column:username;type:varchar(255);uniqueIndex;not null"`     // 登录用户名，唯一
	Email         string         `json:"email" gorm:"column:email;type:varchar(255);uniqueIndex"`                    // 邮箱，用于注册验证，可为空（第三方登录用户）
	PasswordHash  string         `json:"-" gorm:"column:password_hash;type:varchar(255)"`                            // 密码哈希，不返回给前端
	Status        string         `json:"status" gorm:"column:status;type:varchar(50);default:'pending'"`             // active, inactive, pending
	EmailVerified bool           `json:"email_verified" gorm:"column:email_verified;type:boolean;default:false"`     // 邮箱是否已验证
	RegisterType  string         `json:"register_type" gorm:"column:register_type;type:varchar(50);default:'email'"` // 注册方式: email, wechat, github, google等
	ThirdPartyID  string         `json:"third_party_id" gorm:"column:third_party_id;type:varchar(255)"`              // 第三方平台用户ID
	Avatar        string         `json:"avatar" gorm:"column:avatar;type:varchar(500)"`                              // 头像URL
	HostID        int64          `json:"host_id" gorm:"column:host_id"`                                              //每个用户分配一个host，相当于把每个用户都分配一个主机

	// 关联字段
	Host *Host `json:"host" gorm:"foreignKey:HostID;references:ID"`
}

func (User) TableName() string {
	return "user"
}

// CheckEmailVerificationRequired 检查用户是否需要邮箱验证（仅邮箱注册用户需要）
func (u *User) CheckEmailVerificationRequired() bool {
	return u.RegisterType == "email" && !u.EmailVerified
}

// IsPasswordLoginSupported 检查用户是否支持密码登录
func (u *User) IsPasswordLoginSupported() bool {
	return u.RegisterType == "email" && u.PasswordHash != ""
}
