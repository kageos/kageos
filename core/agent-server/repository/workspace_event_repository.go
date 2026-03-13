package repository

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/gorm"
)

// WorkspaceEventRepository 工作台埋点事件数据访问
type WorkspaceEventRepository struct {
	db *gorm.DB
}

// NewWorkspaceEventRepository 创建
func NewWorkspaceEventRepository(db *gorm.DB) *WorkspaceEventRepository {
	return &WorkspaceEventRepository{db: db}
}

// Create 写入一条埋点事件（使用 ctx 中的用户信息写 CreatedBy/UpdatedBy）
func (r *WorkspaceEventRepository) Create(ctx context.Context, e *model.WorkspaceEvent) error {
	user := contextx.GetRequestUser(ctx)
	if user != "" {
		e.CreatedBy = user
		e.UpdatedBy = user
	}
	return r.db.WithContext(ctx).Create(e).Error
}
