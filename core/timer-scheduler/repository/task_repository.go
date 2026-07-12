package repository

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TimerTaskRepository struct {
	db *gorm.DB
}

type ListTasksFilter struct {
	ExecutorKey       string
	Status            string
	Category          string
	SourceType        string
	SourceRef         string
	ResourceScope     string
	ResourceKey       string
	ResourceKeyPrefix string
	CreatedBy         string
	Offset            int
	Limit             int
}

func NewTimerTaskRepository(db *gorm.DB) *TimerTaskRepository {
	return &TimerTaskRepository{db: db}
}

func (r *TimerTaskRepository) WithDB(db *gorm.DB) *TimerTaskRepository {
	return &TimerTaskRepository{db: db}
}

func (r *TimerTaskRepository) Create(task *model.TimerTask) error {
	return r.db.Create(task).Error
}

func (r *TimerTaskRepository) Update(task *model.TimerTask) error {
	return r.db.Save(task).Error
}

func (r *TimerTaskRepository) GetByID(id int64) (*model.TimerTask, error) {
	var task model.TimerTask
	if err := r.db.Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TimerTaskRepository) GetByIDForUpdate(id int64) (*model.TimerTask, error) {
	var task model.TimerTask
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TimerTaskRepository) GetByIdempotencyKey(key string) (*model.TimerTask, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var task model.TimerTask
	if err := r.db.Where("idempotency_key = ?", key).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TimerTaskRepository) ReleaseIdempotencyKey(id int64, key string) error {
	key = strings.TrimSpace(key)
	if id <= 0 || key == "" {
		return nil
	}
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND idempotency_key = ?", id, key).
		Update("idempotency_key", releasedIdempotencyKey(id, key))
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func releasedIdempotencyKey(id int64, key string) string {
	sum := sha1.Sum([]byte(key))
	return fmt.Sprintf("terminal:%d:%s", id, hex.EncodeToString(sum[:])[:24])
}

func (r *TimerTaskRepository) List(req ListTasksFilter) ([]*model.TimerTask, int64, error) {
	query := r.db.Model(&model.TimerTask{})
	if req.ExecutorKey != "" {
		query = query.Where("executor_key = ?", req.ExecutorKey)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.SourceType != "" {
		query = query.Where("source_type = ?", req.SourceType)
	}
	if req.SourceRef != "" {
		query = query.Where("source_ref = ?", req.SourceRef)
	}
	if req.ResourceScope != "" {
		query = query.Where("resource_scope = ?", req.ResourceScope)
	}
	if req.ResourceKey != "" {
		query = query.Where("resource_key = ?", req.ResourceKey)
	}
	if req.ResourceKeyPrefix != "" {
		prefix := strings.TrimRight(req.ResourceKeyPrefix, "/")
		query = query.Where("resource_key = ? OR resource_key LIKE ?", prefix, prefix+"/%")
	}
	if req.CreatedBy != "" {
		query = query.Where("created_by = ?", req.CreatedBy)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Offset > 0 {
		query = query.Offset(req.Offset)
	}

	var list []*model.TimerTask
	if err := query.Order("created_at DESC, id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *TimerTaskRepository) ListDue(now time.Time, limit int) ([]*model.TimerTask, error) {
	query := r.db.
		Where("status = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (lease_until IS NULL OR lease_until < ?)", "pending", now, now).
		Order("next_run_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var list []*model.TimerTask
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *TimerTaskRepository) ListBrokenInflightReferences(limit int) ([]*model.TimerTask, error) {
	query := r.db.Model(&model.TimerTask{}).
		Joins("LEFT JOIN timer_execution AS inflight_exec ON inflight_exec.id = timer_task.inflight_execution_id AND inflight_exec.task_id = timer_task.id").
		Where("timer_task.inflight_execution_id <> 0").
		Where("(inflight_exec.id IS NULL OR inflight_exec.status NOT IN ? OR inflight_exec.lease_until IS NULL)", []string{"queued", "running"}).
		Order("timer_task.updated_at ASC, timer_task.id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var list []*model.TimerTask
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *TimerTaskRepository) GetBrokenInflightReferenceByID(id int64) (*model.TimerTask, error) {
	var task model.TimerTask
	err := r.db.Model(&model.TimerTask{}).
		Joins("LEFT JOIN timer_execution AS inflight_exec ON inflight_exec.id = timer_task.inflight_execution_id AND inflight_exec.task_id = timer_task.id").
		Where("timer_task.id = ? AND timer_task.inflight_execution_id <> 0", id).
		Where("(inflight_exec.id IS NULL OR inflight_exec.status NOT IN ? OR inflight_exec.lease_until IS NULL)", []string{"queued", "running"}).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TimerTaskRepository) TryAcquireDispatch(id int64, owner string, now, leaseUntil time.Time) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (lease_until IS NULL OR lease_until < ?)", id, "pending", now, now).
		Updates(map[string]interface{}{
			"lease_owner": owner,
			"lease_until": leaseUntil,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) TryClearInflight(id, executionID int64, lastError string) (bool, error) {
	updates := map[string]interface{}{
		"inflight_execution_id": 0,
		"lease_owner":           "",
		"lease_until":           nil,
	}
	if strings.TrimSpace(lastError) != "" {
		updates["last_error_message"] = strings.TrimSpace(lastError)
	}
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND inflight_execution_id = ?", id, executionID).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) TrySetInflight(taskID, executionID int64, leaseOwner string, nextRunAt *time.Time) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ? AND lease_owner = ?", taskID, "pending", leaseOwner).
		Updates(map[string]interface{}{
			"inflight_execution_id": executionID,
			"last_execution_id":     executionID,
			"next_run_at":           nextRunAt,
			"lease_owner":           "",
			"lease_until":           nil,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) TrySetManualInflight(taskID, executionID int64) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status IN ?", taskID, []string{"pending", "paused"}).
		Updates(map[string]interface{}{
			"inflight_execution_id": executionID,
			"last_execution_id":     executionID,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) RecordManualExecutionSubmitted(taskID, executionID int64) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status IN ?", taskID, []string{"pending", "paused"}).
		Update("last_execution_id", executionID)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) TrySettleExecution(task *model.TimerTask, executionID int64, expectedStatus string, clearInflight bool) (bool, error) {
	updates := map[string]interface{}{
		"status":             task.Status,
		"next_run_at":        task.NextRunAt,
		"run_count":          task.RunCount,
		"last_execution_id":  task.LastExecutionID,
		"last_error_message": task.LastErrorMessage,
	}
	if clearInflight {
		updates["inflight_execution_id"] = 0
		updates["lease_owner"] = ""
		updates["lease_until"] = nil
	}
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ?", task.ID, expectedStatus).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) Pause(id int64) error {
	return r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ?", id, "pending").
		Update("status", "paused").Error
}

func (r *TimerTaskRepository) Resume(id int64, nextRunAt *time.Time) error {
	return r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ?", id, "paused").
		Updates(map[string]interface{}{
			"status":      "pending",
			"next_run_at": nextRunAt,
		}).Error
}

func (r *TimerTaskRepository) Cancel(id int64) error {
	return r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status IN ?", id, []string{"pending", "paused"}).
		Updates(map[string]interface{}{
			"status":                "cancelled",
			"next_run_at":           nil,
			"inflight_execution_id": 0,
			"lease_owner":           "",
			"lease_until":           nil,
		}).Error
}

func (r *TimerTaskRepository) Delete(task *model.TimerTask) error {
	return r.db.Delete(task).Error
}
