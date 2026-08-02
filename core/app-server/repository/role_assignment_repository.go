package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/access"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleAssignmentRepository struct {
	db *gorm.DB
}

func NewRoleAssignmentRepository(db *gorm.DB) *RoleAssignmentRepository {
	return &RoleAssignmentRepository{db: db}
}

func (r *RoleAssignmentRepository) Transaction(
	ctx context.Context,
	fn func(tx *gorm.DB) error,
) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *RoleAssignmentRepository) UpsertAssignment(ctx context.Context, assignment *model.WorkspaceRoleAssignment) error {
	if assignment == nil {
		return fmt.Errorf("assignment 不能为空")
	}

	key := access.PermissionAssignmentKey(
		assignment.TenantUser,
		assignment.App,
		access.Principal{
			Type: access.PrincipalType(assignment.PrincipalType),
			Key:  assignment.PrincipalKey,
		},
		assignment.ResourcePath,
		access.RoleCode(assignment.RoleCode),
	)
	assignment.AssignmentKey = &key
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "assignment_key"}},
			DoUpdates: clause.Assignments(map[string]any{
				"expires_at": assignment.ExpiresAt,
				"updated_by": assignment.UpdatedBy,
				"updated_at": time.Now(),
				"deleted_at": nil,
				"deleted_by": "",
			}),
		}).
		Create(assignment).Error
}

func (r *RoleAssignmentRepository) RemoveAssignment(
	ctx context.Context,
	tenantUser, app string,
	principal access.Principal,
	resourcePath string,
	roleCode access.RoleCode,
) (int64, error) {
	principal = access.NormalizePrincipal(principal)
	query := r.db.WithContext(ctx).Where(
		"tenant_user = ? AND app = ? AND principal_type = ? AND principal_key = ? AND resource_path = ?",
		tenantUser, app, principal.Type, principal.Key, resourcePath,
	)
	if roleCode != "" {
		query = query.Where("role_code = ?", string(roleCode))
	}
	res := query.Delete(&model.WorkspaceRoleAssignment{})
	return res.RowsAffected, res.Error
}

func (r *RoleAssignmentRepository) ListAssignmentsForPrincipals(
	ctx context.Context,
	tenantUser, app string,
	principals []access.Principal,
) ([]*model.WorkspaceRoleAssignment, error) {
	var assignments []*model.WorkspaceRoleAssignment
	query := r.db.WithContext(ctx).Where("tenant_user = ? AND app = ?", tenantUser, app)
	query = filterAssignmentPrincipals(query, principals)
	err := query.
		Order("resource_path ASC, role_code ASC").
		Find(&assignments).Error
	return assignments, err
}

func (r *RoleAssignmentRepository) ListAssignmentsForWorkspace(ctx context.Context, tenantUser, app string) ([]*model.WorkspaceRoleAssignment, error) {
	var assignments []*model.WorkspaceRoleAssignment
	err := r.db.WithContext(ctx).
		Where("tenant_user = ? AND app = ?", tenantUser, app).
		Order("resource_path ASC, principal_type ASC, principal_key ASC, role_code ASC").
		Find(&assignments).Error
	return assignments, err
}

func (r *RoleAssignmentRepository) ListActiveAssignmentsForPrincipals(
	ctx context.Context,
	principals []access.Principal,
	now time.Time,
) ([]*model.WorkspaceRoleAssignment, error) {
	var assignments []*model.WorkspaceRoleAssignment
	query := filterAssignmentPrincipals(r.db.WithContext(ctx), principals)
	err := query.
		Where("expires_at IS NULL OR expires_at > ?", now).
		Order("tenant_user ASC, app ASC, resource_path ASC").
		Find(&assignments).Error
	return assignments, err
}

func filterAssignmentPrincipals(query *gorm.DB, principals []access.Principal) *gorm.DB {
	userKeys := make([]string, 0, 1)
	departmentKeys := make([]string, 0, len(principals))
	seen := make(map[string]struct{}, len(principals))
	for _, principal := range principals {
		principal = access.NormalizePrincipal(principal)
		if !access.IsValidPrincipal(principal) {
			continue
		}
		identity := string(principal.Type) + "\x00" + principal.Key
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		switch principal.Type {
		case access.PrincipalUser:
			userKeys = append(userKeys, principal.Key)
		case access.PrincipalDepartment:
			departmentKeys = append(departmentKeys, principal.Key)
		}
	}
	switch {
	case len(userKeys) > 0 && len(departmentKeys) > 0:
		return query.Where(
			"(principal_type = ? AND principal_key IN ?) OR (principal_type = ? AND principal_key IN ?)",
			access.PrincipalUser, userKeys, access.PrincipalDepartment, departmentKeys,
		)
	case len(userKeys) > 0:
		return query.Where("principal_type = ? AND principal_key IN ?", access.PrincipalUser, userKeys)
	case len(departmentKeys) > 0:
		return query.Where("principal_type = ? AND principal_key IN ?", access.PrincipalDepartment, departmentKeys)
	default:
		return query.Where("1 = 0")
	}
}
