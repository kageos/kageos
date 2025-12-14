package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

// OperateLogService 操作日志服务
type OperateLogService struct {
	operateLogRepo *repository.OperateLogRepository
}

// NewOperateLogService 创建操作日志服务
func NewOperateLogService(operateLogRepo *repository.OperateLogRepository) *OperateLogService {
	return &OperateLogService{
		operateLogRepo: operateLogRepo,
	}
}

// GetTableOperateLogs 查询 Table 操作日志
func (s *OperateLogService) GetTableOperateLogs(ctx context.Context, req *dto.GetTableOperateLogsReq) (*dto.GetTableOperateLogsResp, error) {
	var logs []*model.TableOperateLog
	var total int64

	// 构建查询
	query := s.operateLogRepo.GetDB().Model(&model.TableOperateLog{})

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
		return nil, fmt.Errorf("查询总数失败: %w", err)
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
		return nil, fmt.Errorf("查询操作日志失败: %w", err)
	}

	return &dto.GetTableOperateLogsResp{
		Logs: logs,
		Total: total,
		Page:  req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetFormOperateLogs 查询 Form 操作日志
func (s *OperateLogService) GetFormOperateLogs(ctx context.Context, req *dto.GetFormOperateLogsReq) (*dto.GetFormOperateLogsResp, error) {
	var logs []*model.FormOperateLog
	var total int64

	// 构建查询
	query := s.operateLogRepo.GetDB().Model(&model.FormOperateLog{})

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
		return nil, fmt.Errorf("查询总数失败: %w", err)
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
		return nil, fmt.Errorf("查询操作日志失败: %w", err)
	}

	return &dto.GetFormOperateLogsResp{
		Logs: logs,
		Total: total,
		Page:  req.Page,
		PageSize: req.PageSize,
	}, nil
}

