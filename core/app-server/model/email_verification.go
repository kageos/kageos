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
	Type      string      `json:"type" gorm:"column:type;type:varchar(50);default:'register'"` // register, reset_password, change_email

	// 关联字段
	User *User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

func (EmailVerification) TableName() string {
	return "email_verification"
}

// CreateEmailVerification 创建邮箱验证记录
func CreateEmailVerification(userID int64, email, token string, expiresAt models.Time, verificationType string) error {
	verification := EmailVerification{
		UserID:    userID,
		Email:     email,
		Token:     token,
		ExpiresAt: expiresAt,
		Type:      verificationType,
	}
	return DB.Create(&verification).Error
}

// GetEmailVerificationByToken 根据token获取邮箱验证记录
func GetEmailVerificationByToken(token string) (*EmailVerification, error) {
	var verification EmailVerification
	err := DB.Where("token = ? AND used = false AND expires_at > ?", token, models.Time{}).First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// MarkEmailVerificationAsUsed 标记邮箱验证记录为已使用
func MarkEmailVerificationAsUsed(token string) error {
	return DB.Model(&EmailVerification{}).Where("token = ?", token).Update("used", true).Error
}

// DeleteExpiredEmailVerifications 删除过期的邮箱验证记录
func DeleteExpiredEmailVerifications() error {
	return DB.Where("expires_at < ?", models.Time{}).Delete(&EmailVerification{}).Error
}
