package repository

import (
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
)

type SystemResourceRepository struct {
	db *gorm.DB
}

func NewSystemResourceRepository(db *gorm.DB) *SystemResourceRepository {
	return &SystemResourceRepository{db: db}
}

func (r *SystemResourceRepository) Create(sample *model.SystemResourceSample) error {
	return r.db.Create(sample).Error
}

func (r *SystemResourceRepository) History(since time.Time, limit int) ([]model.SystemResourceSample, error) {
	if limit <= 0 || limit > 10000 {
		limit = 10000
	}
	var samples []model.SystemResourceSample
	err := r.db.Where("collected_at >= ?", since).
		Order("collected_at ASC, id ASC").
		Limit(limit).
		Find(&samples).Error
	return samples, err
}

func (r *SystemResourceRepository) DeleteBefore(cutoff time.Time) error {
	return r.db.Where("collected_at < ?", cutoff).Delete(&model.SystemResourceSample{}).Error
}
