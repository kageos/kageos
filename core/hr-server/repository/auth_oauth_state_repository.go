package repository

import (
	"context"
	"errors"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
)

type AuthOAuthStateRepository struct {
	db *gorm.DB
}

func NewAuthOAuthStateRepository(db *gorm.DB) *AuthOAuthStateRepository {
	return &AuthOAuthStateRepository{db: db}
}

func (r *AuthOAuthStateRepository) Create(ctx context.Context, state *model.AuthOAuthState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

func (r *AuthOAuthStateRepository) Consume(ctx context.Context, state, providerCode string) (*model.AuthOAuthState, error) {
	var oauthState model.AuthOAuthState
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("state = ? AND provider_code = ?", state, providerCode).First(&oauthState).Error; err != nil {
			return err
		}
		if oauthState.UsedAt != nil {
			return errors.New("oauth state already used")
		}
		if time.Now().After(oauthState.ExpiresAt) {
			return errors.New("oauth state expired")
		}
		now := time.Now()
		result := tx.Model(&model.AuthOAuthState{}).
			Where("id = ? AND used_at IS NULL", oauthState.ID).
			Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("oauth state already used")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &oauthState, nil
}
