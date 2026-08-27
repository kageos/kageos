package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/dto"
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, value := range []any{&model.SystemResourceSample{}, &model.SystemCapacitySnapshot{}, &model.SystemPlatformSnapshot{}} {
			if err := tx.Unscoped().Where("collected_at < ?", cutoff).Delete(value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SystemResourceRepository) CreateCapacity(snapshot dto.SystemResourceSnapshot) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return r.db.Create(&model.SystemCapacitySnapshot{CollectedAt: snapshot.CollectedAt, PayloadJSON: string(payload)}).Error
}

func (r *SystemResourceRepository) LatestCapacity() (*dto.SystemResourceSnapshot, error) {
	var row model.SystemCapacitySnapshot
	if err := r.db.Order("collected_at DESC, id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	var snapshot dto.SystemResourceSnapshot
	if err := json.Unmarshal([]byte(row.PayloadJSON), &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (r *SystemResourceRepository) CreatePlatform(metrics dto.SystemPlatformMetrics) error {
	payload, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return r.db.Create(&model.SystemPlatformSnapshot{CollectedAt: metrics.CollectedAt, PayloadJSON: string(payload)}).Error
}

func (r *SystemResourceRepository) LatestPlatform() (*dto.SystemPlatformMetrics, error) {
	var row model.SystemPlatformSnapshot
	if err := r.db.Order("collected_at DESC, id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	var metrics dto.SystemPlatformMetrics
	if err := json.Unmarshal([]byte(row.PayloadJSON), &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

func (r *SystemResourceRepository) CollectPlatformMetrics(now time.Time) (dto.SystemPlatformMetrics, error) {
	metrics := dto.SystemPlatformMetrics{CollectedAt: now.UTC()}
	counts := []struct {
		table, where string
		target       *int64
	}{
		{"user", "deleted_at IS NULL", &metrics.UsersTotal},
		{"user", "deleted_at IS NULL AND status = 'active'", &metrics.UsersActive},
		{"user", "deleted_at IS NULL AND status = 'pending'", &metrics.UsersPending},
	}
	for _, item := range counts {
		if !r.db.Migrator().HasTable(item.table) {
			continue
		}
		if err := r.db.Table(item.table).Where(item.where).Count(item.target).Error; err != nil {
			return dto.SystemPlatformMetrics{}, fmt.Errorf("count %s: %w", item.table, err)
		}
	}

	return metrics, nil
}

func (r *SystemResourceRepository) CollectDatabaseSizes(ctx context.Context) (uint64, []dto.SystemDatabaseSize, bool) {
	databases := []dto.SystemDatabaseSize{}
	query := "SELECT table_schema AS name, COALESCE(SUM(data_length + index_length), 0) AS used_bytes FROM information_schema.tables WHERE table_schema = DATABASE()"
	query += " GROUP BY table_schema ORDER BY used_bytes DESC LIMIT 10"
	if err := r.db.WithContext(ctx).Raw(query).Scan(&databases).Error; err == nil {
		var total uint64
		for _, database := range databases {
			total += database.UsedBytes
		}
		return total, databases, true
	}
	return 0, databases, false
}
