package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/access"
	"gorm.io/gorm"
)

type TeamAccessRepository struct {
	db *gorm.DB
}

func NewTeamAccessRepository(db *gorm.DB) *TeamAccessRepository {
	return &TeamAccessRepository{db: db}
}

func (r *TeamAccessRepository) UpsertAssignment(ctx context.Context, assignment *model.WorkspaceRoleAssignment) error {
	if assignment == nil {
		return fmt.Errorf("assignment 不能为空")
	}

	var existing model.WorkspaceRoleAssignment
	err := r.db.WithContext(ctx).
		Where("tenant_user = ? AND app = ? AND username = ? AND resource_path = ? AND role_code = ?",
			assignment.TenantUser, assignment.App, assignment.Username, assignment.ResourcePath, assignment.RoleCode).
		First(&existing).Error
	if err == nil {
		existing.ExpiresAt = assignment.ExpiresAt
		existing.CreatedBy = assignment.CreatedBy
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *TeamAccessRepository) RemoveAssignment(ctx context.Context, tenantUser, app, username, resourcePath string, roleCode access.RoleCode) (int64, error) {
	query := r.db.WithContext(ctx).Where(
		"tenant_user = ? AND app = ? AND username = ? AND resource_path = ?",
		tenantUser, app, username, resourcePath,
	)
	if roleCode != "" {
		query = query.Where("role_code = ?", string(roleCode))
	}
	res := query.Delete(&model.WorkspaceRoleAssignment{})
	return res.RowsAffected, res.Error
}

func (r *TeamAccessRepository) ListAssignmentsForUser(ctx context.Context, tenantUser, app, username string) ([]*model.WorkspaceRoleAssignment, error) {
	var assignments []*model.WorkspaceRoleAssignment
	err := r.db.WithContext(ctx).
		Where("tenant_user = ? AND app = ? AND username = ?", tenantUser, app, username).
		Order("resource_path ASC, role_code ASC").
		Find(&assignments).Error
	return assignments, err
}

func (r *TeamAccessRepository) ListAssignmentsForResource(ctx context.Context, tenantUser, app, resourcePath string) ([]*model.WorkspaceRoleAssignment, error) {
	var assignments []*model.WorkspaceRoleAssignment
	query := r.db.WithContext(ctx).Where("tenant_user = ? AND app = ?", tenantUser, app)
	if resourcePath != "" {
		query = query.Where("resource_path = ?", resourcePath)
	}
	err := query.Order("resource_path ASC, username ASC, role_code ASC").Find(&assignments).Error
	return assignments, err
}

func (r *TeamAccessRepository) ListAssignmentsForWorkspace(ctx context.Context, tenantUser, app string) ([]*model.WorkspaceRoleAssignment, error) {
	var assignments []*model.WorkspaceRoleAssignment
	err := r.db.WithContext(ctx).
		Where("tenant_user = ? AND app = ?", tenantUser, app).
		Order("resource_path ASC, username ASC, role_code ASC").
		Find(&assignments).Error
	return assignments, err
}

func (r *TeamAccessRepository) ListActiveAssignmentsForUsername(ctx context.Context, username string, now time.Time) ([]*model.WorkspaceRoleAssignment, error) {
	var assignments []*model.WorkspaceRoleAssignment
	err := r.db.WithContext(ctx).
		Where("username = ? AND (expires_at IS NULL OR expires_at > ?)", username, now).
		Order("tenant_user ASC, app ASC, resource_path ASC").
		Find(&assignments).Error
	return assignments, err
}
