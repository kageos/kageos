package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apperror"
	"github.com/kageos/kageos/pkg/contextx"
)

func (s *TeamAccessService) Assign(ctx context.Context, req access.AssignRoleRequest) error {
	req.ResourcePath = access.NormalizeResourcePath(req.ResourcePath)
	req.RoleCode = access.NormalizeRoleCode(req.RoleCode)
	req.Username = strings.TrimSpace(req.Username)
	req.TenantUser = strings.TrimSpace(req.TenantUser)
	req.App = strings.TrimSpace(req.App)
	if err := validateAssignRoleRequest(req); err != nil {
		return err
	}
	if req.CreatedBy == "" {
		req.CreatedBy = contextx.GetRequestUser(ctx)
	}
	if err := s.requireAdminForGrant(ctx, req.TenantUser, req.App, req.CreatedBy, req.ResourcePath, req.RoleCode); err != nil {
		return err
	}
	if err := s.ensureAssignableUserInRequesterCompany(ctx, req.Username); err != nil {
		return err
	}

	assignment := &model.WorkspaceRoleAssignment{
		TenantUser:   req.TenantUser,
		App:          req.App,
		Username:     req.Username,
		ResourcePath: req.ResourcePath,
		RoleCode:     string(req.RoleCode),
		ExpiresAt:    req.ExpiresAt,
	}
	assignment.CreatedBy = req.CreatedBy
	if err := s.teamAccessRepo.UpsertAssignment(ctx, assignment); err != nil {
		return err
	}

	s.writeOperateLog(ctx, operateLogInput{
		TenantUser:   req.TenantUser,
		App:          req.App,
		ActorUser:    req.CreatedBy,
		Action:       "team.role.assigned",
		ResourceType: "team_access",
		ResourcePath: req.ResourcePath,
		TargetUser:   req.Username,
		Summary:      fmt.Sprintf("%s assigned %s to %s on %s", req.CreatedBy, req.RoleCode, req.Username, req.ResourcePath),
		Status:       "success",
		NewValues: dto.TeamRoleAssignedValues{
			RoleCode:  string(req.RoleCode),
			ExpiresAt: req.ExpiresAt,
		},
	})
	return nil
}

func (s *TeamAccessService) BatchAssign(ctx context.Context, req access.BatchAssignRoleRequest) error {
	if len(req.ResourcePaths) == 0 || len(req.Usernames) == 0 || len(req.RoleCodes) == 0 {
		return apperror.InvalidArgument("resource_paths、usernames、role_codes 不能为空", nil)
	}
	for _, resourcePath := range req.ResourcePaths {
		for _, username := range req.Usernames {
			for _, roleCode := range req.RoleCodes {
				if err := s.Assign(ctx, access.AssignRoleRequest{
					TenantUser:   req.TenantUser,
					App:          req.App,
					Username:     username,
					ResourcePath: resourcePath,
					RoleCode:     roleCode,
					ExpiresAt:    req.ExpiresAt,
					CreatedBy:    req.CreatedBy,
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *TeamAccessService) Remove(ctx context.Context, req access.RemoveRoleRequest) error {
	req.ResourcePath = access.NormalizeResourcePath(req.ResourcePath)
	req.RoleCode = access.NormalizeRoleCode(req.RoleCode)
	req.Username = strings.TrimSpace(req.Username)
	req.TenantUser = strings.TrimSpace(req.TenantUser)
	req.App = strings.TrimSpace(req.App)
	if req.TenantUser == "" || req.App == "" || req.Username == "" || req.ResourcePath == "" {
		return apperror.InvalidArgument("tenant_user、app、username、resource_path 不能为空", nil)
	}
	if req.Actor == "" {
		req.Actor = contextx.GetRequestUser(ctx)
	}
	if err := s.requireAdminForGrant(ctx, req.TenantUser, req.App, req.Actor, req.ResourcePath, req.RoleCode); err != nil {
		return err
	}
	if req.RoleCode == access.RoleOwner && !s.isWorkspaceOwner(ctx, req.TenantUser, req.App, req.Actor) {
		return apperror.PermissionDenied("只有 Owner 可以移除 Owner 角色", nil)
	}

	rows, err := s.teamAccessRepo.RemoveAssignment(ctx, req.TenantUser, req.App, req.Username, req.ResourcePath, req.RoleCode)
	if err != nil {
		return err
	}
	s.writeOperateLog(ctx, operateLogInput{
		TenantUser:   req.TenantUser,
		App:          req.App,
		ActorUser:    req.Actor,
		Action:       "team.role.removed",
		ResourceType: "team_access",
		ResourcePath: req.ResourcePath,
		TargetUser:   req.Username,
		Summary:      fmt.Sprintf("%s removed role %s from %s on %s", req.Actor, req.RoleCode, req.Username, req.ResourcePath),
		Status:       "success",
		Details: dto.TeamRoleRemovedDetails{
			RoleCode:     string(req.RoleCode),
			RowsAffected: rows,
		},
	})
	return nil
}
