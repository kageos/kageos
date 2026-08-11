package repository

import (
	"errors"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrWechatLoginAttemptInvalid = errors.New("wechat login attempt is invalid or expired")
	ErrWechatLoginAttemptPending = errors.New("wechat login attempt is pending")
)

type AuthWechatLoginAttemptRepository struct {
	db *gorm.DB
}

func NewAuthWechatLoginAttemptRepository(db *gorm.DB) *AuthWechatLoginAttemptRepository {
	return &AuthWechatLoginAttemptRepository{db: db}
}

func (r *AuthWechatLoginAttemptRepository) Create(attempt *model.AuthWechatLoginAttempt) error {
	return r.db.Create(attempt).Error
}

func (r *AuthWechatLoginAttemptRepository) MarkScanned(sceneHash, externalID, nickname, avatar string, now time.Time) error {
	result := r.db.Model(&model.AuthWechatLoginAttempt{}).
		Where("scene_hash = ? AND scanned_at IS NULL AND used_at IS NULL AND expires_at > ?", sceneHash, now).
		Updates(map[string]interface{}{
			"external_id": externalID,
			"nickname":    nickname,
			"avatar":      avatar,
			"scanned_at":  now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWechatLoginAttemptInvalid
	}
	return nil
}

func (r *AuthWechatLoginAttemptRepository) Consume(tokenHash string, now time.Time) (*model.AuthWechatLoginAttempt, error) {
	var attempt model.AuthWechatLoginAttempt
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", tokenHash).First(&attempt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWechatLoginAttemptInvalid
			}
			return err
		}
		if attempt.UsedAt != nil || !now.Before(attempt.ExpiresAt) {
			return ErrWechatLoginAttemptInvalid
		}
		if attempt.ScannedAt == nil || attempt.ExternalID == "" {
			return ErrWechatLoginAttemptPending
		}
		result := tx.Model(&model.AuthWechatLoginAttempt{}).
			Where("id = ? AND used_at IS NULL", attempt.ID).
			Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrWechatLoginAttemptInvalid
		}
		attempt.UsedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}
