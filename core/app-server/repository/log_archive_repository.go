package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

type LogArchiveRepository struct{ db *gorm.DB }

func NewLogArchiveRepository(db *gorm.DB) *LogArchiveRepository { return &LogArchiveRepository{db: db} }

func (r *LogArchiveRepository) List(ctx context.Context, page, pageSize int) ([]*model.LogArchiveBatch, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var total int64
	q := r.db.WithContext(ctx).Model(&model.LogArchiveBatch{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []*model.LogArchiveBatch
	if err := q.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *LogArchiveRepository) GetResumable(ctx context.Context) (*model.LogArchiveBatch, error) {
	var batch model.LogArchiveBatch
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{model.LogArchiveStatusExporting, model.LogArchiveStatusUploaded, model.LogArchiveStatusFailed}).
		Order("id ASC").First(&batch).Error
	return &batch, err
}

func (r *LogArchiveRepository) Create(ctx context.Context, batch *model.LogArchiveBatch) error {
	return r.db.WithContext(ctx).Create(batch).Error
}

func (r *LogArchiveRepository) Save(ctx context.Context, batch *model.LogArchiveBatch) error {
	return r.db.WithContext(ctx).Save(batch).Error
}

// NextScope returns the oldest workspace scope with logs older than cutoff.
func (r *LogArchiveRepository) NextScope(ctx context.Context, cutoff time.Time) (string, string, error) {
	var row struct{ TenantUser, App string }
	err := r.db.WithContext(ctx).Model(&model.OperateLog{}).
		Select("tenant_user, app").Where("created_at < ?", cutoff).
		Order("created_at ASC, id ASC").Limit(1).Scan(&row).Error
	if err != nil {
		return "", "", err
	}
	if row.TenantUser == "" && row.App == "" {
		return "", "", gorm.ErrRecordNotFound
	}
	return row.TenantUser, row.App, nil
}

func (r *LogArchiveRepository) SelectIDs(ctx context.Context, tenantUser, app string, cutoff time.Time, limit int) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&model.OperateLog{}).Select("id").
		Where("tenant_user = ? AND app = ? AND created_at < ?", tenantUser, app, cutoff).
		Order("id ASC").Limit(limit).Scan(&ids).Error
	return ids, err
}

func (r *LogArchiveRepository) LoadIDs(ctx context.Context, ids []int64) ([]*model.OperateLog, error) {
	var rows []*model.OperateLog
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *LogArchiveRepository) SelectedStats(ctx context.Context, ids []int64) (time.Time, time.Time, error) {
	var start, end time.Time
	var total int64
	for offset := 0; offset < len(ids); offset += 1000 {
		to := offset + 1000
		if to > len(ids) {
			to = len(ids)
		}
		var row struct {
			Count                      int64
			MinCreatedAt, MaxCreatedAt *time.Time
		}
		err := r.db.WithContext(ctx).Model(&model.OperateLog{}).
			Select("COUNT(*) AS count, MIN(created_at) AS min_created_at, MAX(created_at) AS max_created_at").
			Where("id IN ?", ids[offset:to]).Scan(&row).Error
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		if row.Count != int64(to-offset) || row.MinCreatedAt == nil || row.MaxCreatedAt == nil {
			return time.Time{}, time.Time{}, fmt.Errorf("selected archive logs changed")
		}
		if start.IsZero() || row.MinCreatedAt.Before(start) {
			start = *row.MinCreatedAt
		}
		if end.IsZero() || row.MaxCreatedAt.After(end) {
			end = *row.MaxCreatedAt
		}
		total += row.Count
	}
	if total != int64(len(ids)) {
		return time.Time{}, time.Time{}, fmt.Errorf("selected archive log count changed")
	}
	return start, end, nil
}

func (r *LogArchiveRepository) DeleteRange(ctx context.Context, batch *model.LogArchiveBatch, chunkSize int) (int64, error) {
	if chunkSize < 1 {
		chunkSize = 1000
	}
	var selectedIDs []int64
	if err := json.Unmarshal(batch.SelectedIDsJSON, &selectedIDs); err != nil {
		return 0, fmt.Errorf("decode selected log ids: %w", err)
	}
	var total int64
	for offset := 0; offset < len(selectedIDs); offset += chunkSize {
		to := offset + chunkSize
		if to > len(selectedIDs) {
			to = len(selectedIDs)
		}
		result := r.db.WithContext(ctx).Unscoped().Where("id IN ?", selectedIDs[offset:to]).Delete(&model.OperateLog{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
	}
	return total, nil
}

func IsArchiveNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
