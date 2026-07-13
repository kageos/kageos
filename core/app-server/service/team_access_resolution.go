package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apperror"
)

func (s *TeamAccessService) Check(ctx context.Context, tenantUser, app, username, resourcePath string, action access.Action) error {
	ok, err := s.Can(ctx, tenantUser, app, username, resourcePath, action)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.PermissionDenied(fmt.Sprintf("无权限执行 %s: %s", action, access.NormalizeResourcePath(resourcePath)), nil)
	}
	return nil
}

func (s *TeamAccessService) Can(ctx context.Context, tenantUser, app, username, resourcePath string, action access.Action) (bool, error) {
	result, err := s.Resolve(ctx, tenantUser, app, username, resourcePath)
	if err != nil {
		return false, err
	}
	return access.HasPermission(result.Permissions, action), nil
}

// CanWorkspaceData allows the existing explicit authorization first, then
// applies the open-collaboration fallback for authenticated business-data
// operations. Control-plane APIs must continue to call Can/Check directly.
func (s *TeamAccessService) CanWorkspaceData(ctx context.Context, tenantUser, app, username, resourcePath string, action access.Action) (bool, error) {
	ok, err := s.Can(ctx, tenantUser, app, username, resourcePath, action)
	if err != nil || ok {
		return ok, err
	}
	if !isOpenCollaborationDataAction(action) {
		return false, nil
	}
	return s.IsOpenCollaborationWorkspace(ctx, tenantUser, app, username)
}

func (s *TeamAccessService) CheckWorkspaceData(ctx context.Context, tenantUser, app, username, resourcePath string, action access.Action) error {
	ok, err := s.CanWorkspaceData(ctx, tenantUser, app, username, resourcePath, action)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.PermissionDenied(fmt.Sprintf("无权限执行 %s: %s", action, access.NormalizeResourcePath(resourcePath)), nil)
	}
	return nil
}

// IsOpenCollaborationWorkspace intentionally requires an authenticated user.
// Anonymous public sharing remains a separate capability-link flow.
func (s *TeamAccessService) IsOpenCollaborationWorkspace(ctx context.Context, tenantUser, app, username string) (bool, error) {
	if strings.TrimSpace(username) == "" || s.appRepo == nil {
		return false, nil
	}
	appModel, err := s.appRepo.GetAppByUserNameContext(ctx, strings.TrimSpace(tenantUser), strings.TrimSpace(app))
	if err != nil {
		return false, err
	}
	return !appModel.IsDisabled() && appModel.IsOpenCollaboration(), nil
}

func (s *TeamAccessService) Resolve(ctx context.Context, tenantUser, app, username, resourcePath string) (*access.Result, error) {
	tenantUser = strings.TrimSpace(tenantUser)
	app = strings.TrimSpace(app)
	username = strings.TrimSpace(username)
	resourcePath = access.NormalizeResourcePath(resourcePath)
	if isSystemRootUser(username) {
		return ownerAccessResult(resourcePath), nil
	}
	if access.IsSystemBuiltinPath(resourcePath) {
		return &access.Result{
			ResourcePath: resourcePath,
			RoleCodes:    []access.RoleCode{access.RoleViewer},
			Permissions:  access.RolePermissions(access.RoleViewer),
		}, nil
	}
	if tenantUser == "" || app == "" || username == "" || resourcePath == "" {
		return &access.Result{ResourcePath: resourcePath, Permissions: access.EmptyPermissionSet()}, nil
	}

	if s.isWorkspaceOwnerOrLegacyAdmin(ctx, tenantUser, app, username) {
		return &access.Result{
			ResourcePath: resourcePath,
			RoleCodes:    []access.RoleCode{access.RoleOwner},
			Permissions:  access.RolePermissions(access.RoleOwner),
		}, nil
	}

	assignments, err := s.teamAccessRepo.ListAssignmentsForUser(ctx, tenantUser, app, username)
	if err != nil {
		return nil, err
	}
	return access.Resolve(toAccessAssignments(assignments), resourcePath, time.Now()), nil
}
