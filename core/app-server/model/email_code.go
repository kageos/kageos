package model

import (
	"time"

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

// CreateEmailCode 创建邮箱验证码
func CreateEmailCode(email, code string, expiresAt models.Time, codeType, ipAddress, userAgent string) error {
	emailCode := EmailCode{
		Email:     email,
		Code:      code,
		ExpiresAt: expiresAt,
		Type:      codeType,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	return DB.Create(&emailCode).Error
}

// GetValidEmailCode 获取有效的邮箱验证码
func GetValidEmailCode(email, code, codeType string) (*EmailCode, error) {
	var emailCode EmailCode
	err := DB.Where("email = ? AND code = ? AND type = ? AND used = false AND expires_at > ?",
		email, code, codeType, models.Time{}).First(&emailCode).Error
	if err != nil {
		return nil, err
	}
	return &emailCode, nil
}

// MarkEmailCodeAsUsed 标记邮箱验证码为已使用
func MarkEmailCodeAsUsed(email, code, codeType string) error {
	return DB.Model(&EmailCode{}).Where("email = ? AND code = ? AND type = ?",
		email, code, codeType).Update("used", true).Error
}

// DeleteExpiredEmailCodes 删除过期的邮箱验证码
func DeleteExpiredEmailCodes() error {
	return DB.Where("expires_at < ?", models.Time{}).Delete(&EmailCode{}).Error
}

// GetEmailCodeCount 获取邮箱在指定时间内的验证码数量（防刷）
func GetEmailCodeCount(email string, minutes int) (int64, error) {
	var count int64
	err := DB.Model(&EmailCode{}).Where("email = ? AND created_at > ?",
		email, models.Time(time.Now().Add(-time.Duration(minutes)*time.Minute))).Count(&count).Error
	return count, err
}
