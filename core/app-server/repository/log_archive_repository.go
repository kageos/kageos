package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

type LogArchiveRepository struct {
	db      *gorm.DB
	logType string
}

func NewLogArchiveRepository(db *gorm.DB) *LogArchiveRepository { return &LogArchiveRepository{db: db} }

func (r *LogArchiveRepository) List(ctx context.Context, page, pageSize int, paths ...string) ([]*model.LogArchiveBatch, int64, error) {
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
	if len(paths) > 0 && paths[0] != "" {
		path := paths[0]
		escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(path)
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 2 {
			q = q.Where("(resource_path = ? OR resource_path LIKE ? ESCAPE '!' OR (COALESCE(resource_path,'') = '' AND tenant_user = ? AND app = ?))", path, escaped+"/%", parts[0], parts[1])
		} else {
			q = q.Where("(resource_path = ? OR resource_path LIKE ? ESCAPE '!')", path, escaped+"/%")
		}
	}

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
		Where("next_retry_at IS NULL OR next_retry_at <= ?", time.Now()).
		Order("id ASC").First(&batch).Error
	return &batch, err
}

func (r *LogArchiveRepository) Create(ctx context.Context, batch *model.LogArchiveBatch) error {
	return r.db.WithContext(ctx).Create(batch).Error
}

func (r *LogArchiveRepository) Save(ctx context.Context, batch *model.LogArchiveBatch) error {
	return r.db.WithContext(ctx).Save(batch).Error
}

func (r *LogArchiveRepository) LoadIDs(ctx context.Context, ids []int64) ([]*model.OperateLog, error) {
	var rows []*model.OperateLog
	err := r.db.WithContext(ctx).Table(r.sourceTable()).Where("id IN ?", ids).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *LogArchiveRepository) SelectedStats(ctx context.Context, ids []int64) (time.Time, time.Time, error) {
	var start, end time.Time
	for offset := 0; offset < len(ids); offset += 500 {
		to := min(offset+500, len(ids))
		var rows []model.OperateLog
		if err := r.db.WithContext(ctx).Table(r.sourceTable()).Select("id, created_at").Where("id IN ?", ids[offset:to]).Find(&rows).Error; err != nil {
			return start, end, err
		}
		if len(rows) != to-offset {
			return start, end, fmt.Errorf("selected archive logs changed")
		}
		for _, row := range rows {
			at := time.Time(row.CreatedAt)
			if start.IsZero() || at.Before(start) {
				start = at
			}
			if end.IsZero() || at.After(end) {
				end = at
			}
		}
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
		result := r.db.WithContext(ctx).Table(archiveSourceTable(batch.ArchiveType)).Unscoped().Where("id IN ?", selectedIDs[offset:to]).Delete(&model.OperateLog{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
	}
	return total, nil
}

func IsArchiveNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// Serialize scheduled and manual runs across processes on a dedicated connection.
// A connection lock is released by MySQL if a worker crashes; no long SQL transaction
// is held while uploading. SQLite is only used by local tests.
var archiveTestMutex sync.Mutex

func (r *LogArchiveRepository) Exclusive(ctx context.Context, run func(*LogArchiveRepository) error) error {
	if r.db.Dialector.Name() != "mysql" {
		if !archiveTestMutex.TryLock() {
			return fmt.Errorf("archive task is already running")
		}
		defer archiveTestMutex.Unlock()
		return run(r)
	}
	return r.db.WithContext(ctx).Connection(func(db *gorm.DB) error {
		var acquired int
		if err := db.Raw("SELECT GET_LOCK('kageos.log_archive', 0)").Scan(&acquired).Error; err != nil {
			return err
		}
		if acquired != 1 {
			return fmt.Errorf("archive task is already running")
		}
		defer db.WithContext(context.WithoutCancel(ctx)).Exec("SELECT RELEASE_LOCK('kageos.log_archive')")
		return run(NewLogArchiveRepository(db))
	})
}

func (r *LogArchiveRepository) Get(ctx context.Context, id int64) (*model.LogArchiveBatch, error) {
	var batch model.LogArchiveBatch
	err := r.db.WithContext(ctx).First(&batch, id).Error
	return &batch, err
}

const unclaimedArchiveLogs = `NOT EXISTS (
 SELECT 1 FROM log_archive_batches b WHERE b.deleted_at IS NULL
 AND b.status <> 'completed' AND COALESCE(NULLIF(b.archive_type, ''), 'operate_log') = 'operate_log' AND b.tenant_user = operate_logs.tenant_user
 AND b.app = operate_logs.app AND operate_logs.id BETWEEN b.min_log_id AND b.max_log_id
)`

func (r *LogArchiveRepository) SaveProgress(ctx context.Context, progress *model.LogArchiveProgress) error {
	return r.db.WithContext(ctx).Save(progress).Error
}
func (r *LogArchiveRepository) Progress(ctx context.Context, executionID int64) (*model.LogArchiveProgress, error) {
	var row model.LogArchiveProgress
	err := r.db.WithContext(ctx).First(&row, "execution_id = ?", executionID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *LogArchiveRepository) HasPending(ctx context.Context) (bool, error) {
	var id int64
	err := r.db.WithContext(ctx).Model(&model.LogArchiveBatch{}).Where("status <> ?", model.LogArchiveStatusCompleted).Select("id").Limit(1).Scan(&id).Error
	return id > 0, err
}

func archiveSourceTable(kind string) string {
	if kind == "scheduled_execution" {
		return "scheduled_execution_logs"
	}
	return "operate_logs"
}
func (r *LogArchiveRepository) sourceTable() string { return archiveSourceTable(r.logType) }
func (r *LogArchiveRepository) ForType(kind string) *LogArchiveRepository {
	return &LogArchiveRepository{db: r.db, logType: kind}
}

// Select one exact resource path; parent directories never absorb low-volume children.
func (r *LogArchiveRepository) SelectArchiveIDs(ctx context.Context, cutoff time.Time, threshold, limit int) (string, string, []int64, error) {
	table := r.sourceTable()
	claimed := strings.ReplaceAll(unclaimedArchiveLogs, "operate_logs.", table+".")
	claimed = strings.Replace(claimed, "= 'operate_log'", "= '"+r.logType+"'", 1)
	q := r.db.WithContext(ctx).Model(&model.OperateLog{}).Table(table).Where("created_at < ? AND COALESCE(status,'') <> 'pending' AND COALESCE(resource_type,'') <> 'log_archive'", cutoff).Where(claimed)
	var scope struct {
		TenantUser, App, ResourcePath string
		Records                       int64
	}
	if err := q.Session(&gorm.Session{}).Select("tenant_user, app, resource_path, COUNT(*) AS records").Group("tenant_user, app, resource_path").Having("COUNT(*) >= ?", max(threshold, 1)).Order("MIN(created_at)").Limit(1).Scan(&scope).Error; err != nil {
		return "", "", nil, err
	}
	if scope.Records == 0 {
		return "", "", nil, gorm.ErrRecordNotFound
	}
	var ids []int64
	err := q.Where("tenant_user = ? AND app = ? AND resource_path = ?", scope.TenantUser, scope.App, scope.ResourcePath).Order("id").Limit(limit).Pluck("id", &ids).Error
	return scope.TenantUser, scope.App, ids, err
}
