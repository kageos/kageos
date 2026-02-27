package repository

import (
	"time"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

type PubKeyRepository struct {
	db *gorm.DB
}

func NewPubKeyRepository(db *gorm.DB) *PubKeyRepository {
	return &PubKeyRepository{db: db}
}

func (r *PubKeyRepository) Create(key *model.PubKey) error {
	return r.db.Create(key).Error
}

func (r *PubKeyRepository) GetByKey(key string) (*model.PubKey, error) {
	var pubKey model.PubKey
	err := r.db.Where("`key` = ?", key).First(&pubKey).Error
	if err != nil {
		return nil, err
	}
	return &pubKey, nil
}

func (r *PubKeyRepository) ListByUsername(username string) ([]model.PubKey, error) {
	var keys []model.PubKey
	err := r.db.Where("username = ?", username).Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (r *PubKeyRepository) DeleteByIDAndUsername(id int64, username string) error {
	result := r.db.Where("id = ? AND username = ?", id, username).Delete(&model.PubKey{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *PubKeyRepository) UpdateLastUsedAt(id int64) error {
	now := time.Now()
	return r.db.Model(&model.PubKey{}).Where("id = ?", id).Update("last_used_at", &now).Error
}
