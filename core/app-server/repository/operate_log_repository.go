package repository

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"gorm.io/gorm"
)

// OperateLogRepository 操作日志仓库
type OperateLogRepository struct {
	db *gorm.DB
}

// GetDB 获取数据库连接（用于复杂查询）
func (r *OperateLogRepository) GetDB() *gorm.DB {
	return r.db
}

// NewOperateLogRepository 创建操作日志仓库
func NewOperateLogRepository(db *gorm.DB) *OperateLogRepository {
	return &OperateLogRepository{db: db}
}

// GetTableOperateLogs 查询 Table 操作日志。
func (r *OperateLogRepository) GetTableOperateLogs(ctx context.Context, req *dto.GetTableOperateLogsReq) ([]*model.TableOperateLog, int64, error) {
	var logs []*model.TableOperateLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.TableOperateLog{})
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

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	if req.Page > 0 && req.PageSize > 0 {
		query = query.Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize)
	}

	orderBy := "created_at DESC"
	if req.OrderBy != "" {
		orderBy = req.OrderBy
	}
	if err := query.Order(orderBy).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询操作日志失败: %w", err)
	}

	return logs, total, nil
}

// CreateTableOperateLog 创建 Table 操作日志
func (r *OperateLogRepository) CreateTableOperateLog(log *model.TableOperateLog) error {
	return r.db.Create(log).Error
}
