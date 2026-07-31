package repository

import (
	"crypto/subtle"
	"errors"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

var (
	ErrEmailCodeInvalid      = errors.New("email code invalid or expired")
	ErrEmailCodeTooManyTries = errors.New("email code has too many failed attempts")
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

// InvalidateEmailCode marks one generated code as unusable. The row remains in
// the database so failed SMTP attempts still count towards send rate limits.
func (r *EmailCodeRepository) InvalidateEmailCode(email, code, codeType string) error {
	return r.db.Model(&model.EmailCode{}).
		Where("email = ? AND code = ? AND type = ? AND used = false", email, code, codeType).
		Update("used", true).Error
}

// VerifyAndConsumeLatestEmailCode 原子地验证并消费最新验证码。
// 成功时只有一个并发请求能把 used 从 false 更新为 true；失败尝试也会持久化计数。
func (r *EmailCodeRepository) VerifyAndConsumeLatestEmailCode(email, code, codeType string, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	now := models.Time(time.Now())
	var emailCode model.EmailCode
	err := r.db.
		Where("email = ? AND type = ? AND used = false AND expires_at > ?", email, codeType, now).
		Order("created_at DESC, id DESC").
		First(&emailCode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmailCodeInvalid
		}
		return err
	}
	if emailCode.Attempts >= maxAttempts {
		return ErrEmailCodeTooManyTries
	}

	if subtle.ConstantTimeCompare([]byte(emailCode.Code), []byte(code)) != 1 {
		result := r.db.Model(&model.EmailCode{}).
			Where("id = ? AND used = false AND attempts < ?", emailCode.ID, maxAttempts).
			UpdateColumn("attempts", gorm.Expr("attempts + 1"))
		if result.Error != nil {
			return result.Error
		}
		return ErrEmailCodeInvalid
	}

	result := r.db.Model(&model.EmailCode{}).
		Where("id = ? AND used = false AND expires_at > ? AND attempts < ?", emailCode.ID, now, maxAttempts).
		Update("used", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrEmailCodeInvalid
	}
	return nil
}

// DeleteExpiredEmailCodes 删除过期的邮箱验证码
func (r *EmailCodeRepository) DeleteExpiredEmailCodes() error {
	return r.db.Where("expires_at < ?", models.Time(time.Now())).Delete(&model.EmailCode{}).Error
}

// GetEmailCodeCountByIP 获取来源 IP 在指定时间内的验证码请求数量。
func (r *EmailCodeRepository) GetEmailCodeCountByIP(ipAddress string, minutes int) (int64, error) {
	var count int64
	err := r.db.Model(&model.EmailCode{}).Where("ip_address = ? AND created_at > ?",
		ipAddress, models.Time(time.Now().Add(-time.Duration(minutes)*time.Minute))).Count(&count).Error
	return count, err
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
