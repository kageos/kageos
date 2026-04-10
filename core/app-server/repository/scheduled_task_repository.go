package repository

import (
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// ScheduledTaskRepository 定时任务仓储
type ScheduledTaskRepository struct {
	db *gorm.DB
}

func NewScheduledTaskRepository(db *gorm.DB) *ScheduledTaskRepository {
	return &ScheduledTaskRepository{db: db}
}

func (r *ScheduledTaskRepository) Create(task *model.ScheduledTask) error {
	return r.db.Create(task).Error
}

func (r *ScheduledTaskRepository) GetByID(id int64) (*model.ScheduledTask, error) {
	var t model.ScheduledTask
	if err := r.db.Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// ListPendingDue 查询待执行且已到点、未被有效租约占用的任务。
func (r *ScheduledTaskRepository) ListPendingDue(now time.Time, limit int) ([]*model.ScheduledTask, error) {
	var list []*model.ScheduledTask
	query := r.db.
		Where("status = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (lease_until IS NULL OR lease_until < ?)", "pending", now, now).
		Order("next_run_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&list).Error
	return list, err
}

// ListByUser 分页列表。
// 规则：
// 1. 传 full_code_path 时，按路径维度返回该节点及子路径下的任务，不再额外按 created_by 过滤。
// 2. 未传 full_code_path 时，返回当前创建人的任务。
// 3. 按创建时间倒序，可按 status 筛选。
func (r *ScheduledTaskRepository) ListByUser(createdBy string, status string, fullCodePath string, offset, limit int) ([]*model.ScheduledTask, int64, error) {
	var list []*model.ScheduledTask
	query := r.db.Model(&model.ScheduledTask{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if fullCodePath != "" {
		prefix := strings.TrimSuffix(strings.TrimSpace(fullCodePath), "/")
		// 目录节点下列出子函数上的任务：/a/b 匹配 /a/b 与 /a/b/...；用 "/%" 避免 /a/b 误匹配 /a/b-extra
		query = query.Where("full_code_path = ? OR full_code_path LIKE ?", prefix, prefix+"/%")
	} else if createdBy != "" {
		query = query.Where("created_by = ?", createdBy)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ScheduledTaskRepository) Update(task *model.ScheduledTask) error {
	return r.db.Save(task).Error
}

// TryAcquireLease 为待执行任务抢占执行租约，返回是否抢占成功。
func (r *ScheduledTaskRepository) TryAcquireLease(id int64, owner string, now, leaseUntil time.Time) (bool, error) {
	result := r.db.Model(&model.ScheduledTask{}).
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

// UpdateAfterRun 按租约持有者落执行结果，避免取消/抢占后的旧执行覆盖最新状态。
func (r *ScheduledTaskRepository) UpdateAfterRun(task *model.ScheduledTask, leaseOwner string) (bool, error) {
	result := r.db.Model(&model.ScheduledTask{}).
		Where("id = ? AND status = ? AND lease_owner = ?", task.ID, "pending", leaseOwner).
		Updates(map[string]interface{}{
			"run_count":     task.RunCount,
			"error_message": task.ErrorMessage,
			"status":        task.Status,
			"next_run_at":   task.NextRunAt,
			"lease_owner":   "",
			"lease_until":   nil,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// Cancel 取消任务（status 改为 cancelled，仅创建人可取消）
func (r *ScheduledTaskRepository) Cancel(id int64, createdBy string) error {
	return r.db.Model(&model.ScheduledTask{}).Where("id = ? AND created_by = ?", id, createdBy).Updates(map[string]interface{}{
		"status":      "cancelled",
		"next_run_at": nil,
		"lease_owner": "",
		"lease_until": nil,
	}).Error
}
