package repository

import (
	"context"
	"errors"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrOAuthRegistrationIntentUsed          = errors.New("oauth registration intent already used")
	ErrOAuthRegistrationIntentExpired       = errors.New("oauth registration intent expired")
	ErrOAuthRegistrationUsernameTaken       = errors.New("oauth registration username taken")
	ErrOAuthRegistrationEmailTaken          = errors.New("oauth registration email taken")
	ErrOAuthRegistrationExternalIDBound     = errors.New("oauth registration external identity already bound")
	ErrOAuthRegistrationIntentUnexpectedRow = errors.New("oauth registration intent update failed")
)

type AuthOAuthRegistrationIntentRepository struct {
	db *gorm.DB
}

func NewAuthOAuthRegistrationIntentRepository(db *gorm.DB) *AuthOAuthRegistrationIntentRepository {
	return &AuthOAuthRegistrationIntentRepository{db: db}
}

func (r *AuthOAuthRegistrationIntentRepository) Create(ctx context.Context, intent *model.AuthOAuthRegistrationIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *AuthOAuthRegistrationIntentRepository) GetByTicket(ctx context.Context, ticket string) (*model.AuthOAuthRegistrationIntent, error) {
	var intent model.AuthOAuthRegistrationIntent
	err := r.db.WithContext(ctx).Where("ticket = ?", ticket).First(&intent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &intent, nil
}

func (r *AuthOAuthRegistrationIntentRepository) Complete(ctx context.Context, ticket string, user *model.User, identity *model.AuthExternalIdentity) (*model.AuthOAuthRegistrationIntent, error) {
	var intent model.AuthOAuthRegistrationIntent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("ticket = ?", ticket).
			First(&intent).Error; err != nil {
			return err
		}
		if intent.UsedAt != nil {
			return ErrOAuthRegistrationIntentUsed
		}
		if time.Now().After(intent.ExpiresAt) {
			return ErrOAuthRegistrationIntentExpired
		}

		var existingUser model.User
		if err := tx.Where("username = ?", user.Username).First(&existingUser).Error; err == nil {
			return ErrOAuthRegistrationUsernameTaken
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var existingIdentity model.AuthExternalIdentity
		if err := tx.Where("provider_code = ? AND external_id = ?", identity.ProviderCode, identity.ExternalID).First(&existingIdentity).Error; err == nil {
			return ErrOAuthRegistrationExternalIDBound
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(user).Error; err != nil {
			return err
		}
		identity.UserID = user.ID
		if err := tx.Create(identity).Error; err != nil {
			return err
		}

		now := time.Now()
		result := tx.Model(&model.AuthOAuthRegistrationIntent{}).
			Where("id = ? AND used_at IS NULL", intent.ID).
			Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrOAuthRegistrationIntentUnexpectedRow
		}
		intent.UsedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &intent, nil
}
