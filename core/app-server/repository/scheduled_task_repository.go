package repository

import (
	"fmt"
	"strings"

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

// UpdateAfterRun 落业务执行结果；中心 timer-scheduler 已负责调度 claim，这里只避免覆盖已取消任务。
func (r *ScheduledTaskRepository) UpdateAfterRun(task *model.ScheduledTask) (bool, error) {
	result := r.db.Model(&model.ScheduledTask{}).
		Where("id = ? AND status = ?", task.ID, "pending").
		Updates(map[string]interface{}{
			"run_count":     task.RunCount,
			"error_message": task.ErrorMessage,
			"status":        task.Status,
			"next_run_at":   task.NextRunAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// Cancel 取消任务（status 改为 cancelled，仅创建人可取消）
func (r *ScheduledTaskRepository) Cancel(id int64, createdBy string) error {
	result := r.db.Model(&model.ScheduledTask{}).Where("id = ? AND created_by = ?", id, createdBy).Updates(map[string]interface{}{
		"status":      "cancelled",
		"next_run_at": nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("任务不存在或无权限操作")
	}
	return nil
}

// Delete 删除任务（软删除，仅创建人可删除）
func (r *ScheduledTaskRepository) Delete(id int64, createdBy string) error {
	result := r.db.Where("id = ? AND created_by = ?", id, createdBy).Delete(&model.ScheduledTask{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("任务不存在或无权限操作")
	}
	return nil
}
