package repository

import (
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

func (r *AuthExternalIdentityRepository) GetByProviderSubject(providerCode, externalID string) (*model.AuthExternalIdentity, error) {
	var identity model.AuthExternalIdentity
	err := r.db.Where("provider_code = ? AND external_id = ?", providerCode, externalID).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *AuthExternalIdentityRepository) Create(identity *model.AuthExternalIdentity) error {
	return r.db.Create(identity).Error
}

func (r *AuthExternalIdentityRepository) Update(identity *model.AuthExternalIdentity) error {
	return r.db.Save(identity).Error
}
