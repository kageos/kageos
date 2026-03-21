package repository

import (
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

// ListPendingDue 查询待执行且已到点的任务（next_run_at <= now）
func (r *ScheduledTaskRepository) ListPendingDue(now interface{}) ([]*model.ScheduledTask, error) {
	var list []*model.ScheduledTask
	err := r.db.Where("status = ? AND next_run_at <= ?", "pending", now).Find(&list).Error
	return list, err
}

// ListAllPending 查询所有待执行任务（含 next_run_at），用于调度器启动时同步到 DueQueue
func (r *ScheduledTaskRepository) ListAllPending() ([]*model.ScheduledTask, error) {
	var list []*model.ScheduledTask
	err := r.db.Where("status = ? AND next_run_at IS NOT NULL", "pending").Find(&list).Error
	return list, err
}

// ListByUser 分页列表（按创建人；可选按 full_code_path 前缀过滤：匹配该路径本身及其子路径下的任务；按创建时间倒序），可按 status 筛选
func (r *ScheduledTaskRepository) ListByUser(createdBy string, status string, fullCodePath string, offset, limit int) ([]*model.ScheduledTask, int64, error) {
	var list []*model.ScheduledTask
	query := r.db.Model(&model.ScheduledTask{}).Where("created_by = ?", createdBy)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if fullCodePath != "" {
		prefix := strings.TrimSuffix(strings.TrimSpace(fullCodePath), "/")
		// 目录节点下列出子函数上的任务：/a/b 匹配 /a/b 与 /a/b/...；用 "/%" 避免 /a/b 误匹配 /a/b-extra
		query = query.Where("full_code_path = ? OR full_code_path LIKE ?", prefix, prefix+"/%")
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

// Cancel 取消任务（status 改为 cancelled，仅创建人可取消）
func (r *ScheduledTaskRepository) Cancel(id int64, createdBy string) error {
	return r.db.Model(&model.ScheduledTask{}).Where("id = ? AND created_by = ?", id, createdBy).Updates(map[string]interface{}{
		"status":       "cancelled",
		"next_run_at":  nil,
	}).Error
}
