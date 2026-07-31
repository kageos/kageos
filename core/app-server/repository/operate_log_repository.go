package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
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

// GetOperateLogs 查询通用操作日志。
func (r *OperateLogRepository) GetOperateLogs(ctx context.Context, req *dto.GetOperateLogsReq) ([]*model.OperateLog, int64, error) {
	var logs []*model.OperateLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.OperateLog{})
	if req.ID > 0 {
		query = query.Where("id = ?", req.ID)
	}
	if req.TenantUser != "" {
		query = query.Where("tenant_user = ?", req.TenantUser)
	}
	if req.ActorUser != "" {
		query = query.Where("actor_user = ?", req.ActorUser)
	}
	if req.TargetUser != "" {
		query = query.Where("target_user = ?", req.TargetUser)
	}
	if req.App != "" {
		query = query.Where("app = ?", req.App)
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}
	if req.ResourcePath != "" {
		query = query.Where("resource_path = ?", req.ResourcePath)
	}
	if req.ResourcePathPrefix != "" {
		query = query.Where("(resource_path = ? OR resource_path LIKE ?)", req.ResourcePathPrefix, req.ResourcePathPrefix+"/%")
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Source != "" {
		query = query.Where("source = ?", req.Source)
	}
	if req.SourceType != "" {
		query = query.Where("source_type = ?", req.SourceType)
	}
	if req.SourceRef != "" {
		query = query.Where("source_ref = ?", req.SourceRef)
	}
	if req.ExecutorType != "" {
		query = query.Where("executor_type = ?", req.ExecutorType)
	}
	if req.WorkspaceSessionID != "" {
		query = query.Where("workspace_session_id = ?", req.WorkspaceSessionID)
	}
	if req.InitiatorUser != "" {
		query = query.Where("initiator_user = ?", req.InitiatorUser)
	}
	if req.WorkspaceMessageID > 0 {
		query = query.Where("workspace_message_id = ?", req.WorkspaceMessageID)
	}
	if req.ToolCallID != "" {
		query = query.Where("tool_call_id = ?", req.ToolCallID)
	}
	if req.ToolName != "" {
		query = query.Where("tool_name = ?", req.ToolName)
	}
	if req.TraceID != "" {
		query = query.Where("trace_id = ?", req.TraceID)
	}
	if req.RowID > 0 {
		query = query.Where("target_id = ?", fmt.Sprintf("%d", req.RowID))
	}
	if req.Keyword != "" {
		keyword := strings.TrimSpace(req.Keyword)
		likeKeyword := "%" + keyword + "%"
		query = query.Where(
			"actor_user LIKE ? OR target_user LIKE ? OR resource_path LIKE ? OR source_type LIKE ? OR source_ref LIKE ? OR executor_type LIKE ? OR workspace_session_id LIKE ? OR workspace_session_title LIKE ? OR initiator_user LIKE ? OR tool_call_id LIKE ? OR tool_name LIKE ? OR trace_id LIKE ? OR summary LIKE ?",
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通用操作日志总数失败: %w", err)
	}
	if req.Page > 0 && req.PageSize > 0 {
		query = query.Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize)
	}

	orderBy := normalizeOperateLogOrderBy(req.OrderBy)
	if err := query.Order(orderBy).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通用操作日志失败: %w", err)
	}

	return logs, total, nil
}

func normalizeOperateLogOrderBy(orderBy string) string {
	switch strings.ToLower(strings.TrimSpace(orderBy)) {
	case "created_at asc":
		return "created_at ASC, id ASC"
	default:
		return "created_at DESC, id DESC"
	}
}

// CreateOperateLog 创建平台级操作日志。
func (r *OperateLogRepository) CreateOperateLog(ctx context.Context, log *model.OperateLog) error {
	if log == nil {
		return nil
	}
	return r.db.WithContext(ctx).Create(log).Error
}
