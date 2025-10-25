package repository

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

type EmailCodeRepository struct {
	db *gorm.DB
}

func NewEmailCodeRepository(db *gorm.DB) *EmailCodeRepository {
	return &EmailCodeRepository{db: db}
}

// CreateEmailCode 创建邮箱验证码
func (r *EmailCodeRepository) CreateEmailCode(email, code string, expiresAt models.Time, codeType, ipAddress, userAgent string) error {
	emailCode := model.EmailCode{
		Email:     email,
		Code:      code,
		ExpiresAt: expiresAt,
		Type:      codeType,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	return r.db.Create(&emailCode).Error
}

// GetValidEmailCode 获取有效的邮箱验证码
func (r *EmailCodeRepository) GetValidEmailCode(email, code, codeType string) (*model.EmailCode, error) {
	var emailCode model.EmailCode
	err := r.db.Where("email = ? AND code = ? AND type = ? AND used = false AND expires_at > ?",
		email, code, codeType, models.Time{}).First(&emailCode).Error
	if err != nil {
		return nil, err
	}
	return &emailCode, nil
}

// MarkEmailCodeAsUsed 标记邮箱验证码为已使用
func (r *EmailCodeRepository) MarkEmailCodeAsUsed(email, code, codeType string) error {
	return r.db.Model(&model.EmailCode{}).Where("email = ? AND code = ? AND type = ?",
		email, code, codeType).Update("used", true).Error
}

// DeleteExpiredEmailCodes 删除过期的邮箱验证码
func (r *EmailCodeRepository) DeleteExpiredEmailCodes() error {
	return r.db.Where("expires_at < ?", models.Time{}).Delete(&model.EmailCode{}).Error
}

// GetEmailCodeCount 获取邮箱在指定时间内的验证码数量（防刷）
func (r *EmailCodeRepository) GetEmailCodeCount(email string, minutes int) (int64, error) {
	var count int64
	err := r.db.Model(&model.EmailCode{}).Where("email = ? AND created_at > ?",
		email, models.Time(time.Now().Add(-time.Duration(minutes)*time.Minute))).Count(&count).Error
	return count, err
}

// GetEmailCodeByID 根据ID获取邮箱验证码
func (r *EmailCodeRepository) GetEmailCodeByID(id int64) (*model.EmailCode, error) {
	var emailCode model.EmailCode
	err := r.db.Where("id = ?", id).First(&emailCode).Error
	if err != nil {
		return nil, err
	}
	return &emailCode, nil
}

// GetEmailCodesByEmail 根据邮箱获取所有验证码
func (r *EmailCodeRepository) GetEmailCodesByEmail(email string) ([]*model.EmailCode, error) {
	var emailCodes []*model.EmailCode
	err := r.db.Where("email = ?", email).Find(&emailCodes).Error
	if err != nil {
		return nil, err
	}
	return emailCodes, nil
}
