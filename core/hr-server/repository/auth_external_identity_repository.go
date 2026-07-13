package repository

import (
	"context"
	"errors"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
)

type AuthExternalIdentityRepository struct {
	db *gorm.DB
}

func NewAuthExternalIdentityRepository(db *gorm.DB) *AuthExternalIdentityRepository {
	return &AuthExternalIdentityRepository{db: db}
}

func (r *AuthExternalIdentityRepository) GetByProviderSubject(ctx context.Context, providerCode, externalID string) (*model.AuthExternalIdentity, error) {
	var identity model.AuthExternalIdentity
	err := r.db.WithContext(ctx).Where("provider_code = ? AND external_id = ?", providerCode, externalID).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *AuthExternalIdentityRepository) Create(ctx context.Context, identity *model.AuthExternalIdentity) error {
	return r.db.WithContext(ctx).Create(identity).Error
}

func (r *AuthExternalIdentityRepository) Update(ctx context.Context, identity *model.AuthExternalIdentity) error {
	return r.db.WithContext(ctx).Save(identity).Error
}
