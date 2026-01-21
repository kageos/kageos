package repository

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"gorm.io/gorm"
)

// OperateLogRepository 操作日志仓库（企业版）
type OperateLogRepository struct {
	db *gorm.DB
}

// NewOperateLogRepository 创建操作日志仓库
func NewOperateLogRepository(db *gorm.DB) *OperateLogRepository {
	return &OperateLogRepository{
		db: db,
	}
}

// GetTableOperateLogs 查询 Table 操作日志
func (r *OperateLogRepository) GetTableOperateLogs(ctx context.Context, req *dto.GetTableOperateLogsReq) ([]*model.TableOperateLog, int64, error) {
	var logs []*model.TableOperateLog
	var total int64

	// 构建查询
	query := r.db.Model(&model.TableOperateLog{})

	// 条件过滤
	if req.TenantUser != "" {
		query = query.Where("tenant_user = ?", req.TenantUser)
	}
	if req.RequestUser != "" {
		query = query.Where("request_user = ?", req.RequestUser)
	}
	if req.App != "" {
		query = query.Where("app = ?", req.App)
	}
	if req.FullCodePath != "" {
		query = query.Where("full_code_path = ?", req.FullCodePath)
	}
	if req.RowID > 0 {
		query = query.Where("row_id = ?", req.RowID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 分页查询
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 排序（默认按创建时间倒序）
	orderBy := "created_at DESC"
	if req.OrderBy != "" {
		orderBy = req.OrderBy
	}
	query = query.Order(orderBy)

	// 执行查询
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询操作日志失败: %w", err)
	}

	return logs, total, nil
}

// CreateTableOperateLog 创建 Table 操作日志
func (r *OperateLogRepository) CreateTableOperateLog(log *model.TableOperateLog) error {
	return r.db.Create(log).Error
}

// CreateFormOperateLog 创建 Form 操作日志
func (r *OperateLogRepository) CreateFormOperateLog(log *model.FormOperateLog) error {
	return r.db.Create(log).Error
}

// GetFormOperateLogs 查询 Form 操作日志
func (r *OperateLogRepository) GetFormOperateLogs(ctx context.Context, req *dto.GetFormOperateLogsReq) ([]*model.FormOperateLog, int64, error) {
	var logs []*model.FormOperateLog
	var total int64

	// 构建查询
	query := r.db.Model(&model.FormOperateLog{})

	// 条件过滤
	if req.TenantUser != "" {
		query = query.Where("tenant_user = ?", req.TenantUser)
	}
	if req.RequestUser != "" {
		query = query.Where("request_user = ?", req.RequestUser)
	}
	if req.App != "" {
		query = query.Where("app = ?", req.App)
	}
	if req.FullCodePath != "" {
		query = query.Where("full_code_path = ?", req.FullCodePath)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 分页查询
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 排序（默认按创建时间倒序）
	orderBy := "created_at DESC"
	if req.OrderBy != "" {
		orderBy = req.OrderBy
	}
	query = query.Order(orderBy)

	// 执行查询
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询操作日志失败: %w", err)
	}

	return logs, total, nil
}
