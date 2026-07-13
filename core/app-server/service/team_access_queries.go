package service

import (
	"context"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/access"
)

func (s *TeamAccessService) ListMembers(ctx context.Context, tenantUser, app, resourcePath string) ([]access.MemberAccess, error) {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	assignments, err := s.teamAccessRepo.ListAssignmentsForWorkspace(ctx, tenantUser, app)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	members := make([]access.MemberAccess, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentPath := access.NormalizeResourcePath(assignment.ResourcePath)
		if !access.PathApplies(assignmentPath, resourcePath) {
			continue
		}
		if assignment.ExpiresAt != nil && !assignment.ExpiresAt.After(now) {
			continue
		}
		createdAt := time.Time(assignment.CreatedAt)
		updatedAt := time.Time(assignment.UpdatedAt)
		roleCode := access.NormalizeRoleCode(access.RoleCode(assignment.RoleCode))
		direct := assignmentPath == resourcePath
		source := "current"
		inheritedFrom := ""
		if !direct {
			source = "inherited"
			inheritedFrom = assignmentPath
		}
		members = append(members, access.MemberAccess{
			TenantUser:     assignment.TenantUser,
			App:            assignment.App,
			Username:       assignment.Username,
			ResourcePath:   assignmentPath,
			RoleCode:       roleCode,
			Permissions:    access.RolePermissions(roleCode),
			Source:         source,
			Direct:         direct,
			InheritedFrom:  inheritedFrom,
			TargetResource: resourcePath,
			ExpiresAt:      assignment.ExpiresAt,
			CreatedBy:      assignment.CreatedBy,
			CreatedAt:      &createdAt,
			UpdatedAt:      &updatedAt,
		})
	}
	return members, nil
}

func (s *TeamAccessService) HasAnyWorkspaceAccess(ctx context.Context, tenantUser, app, username string) (bool, error) {
	if username == "" {
		return false, nil
	}
	if s.isWorkspaceOwnerOrLegacyAdmin(ctx, tenantUser, app, username) {
		return true, nil
	}
	open, err := s.IsOpenCollaborationWorkspace(ctx, tenantUser, app, username)
	if err != nil {
		return false, err
	}
	if open {
		return true, nil
	}
	assignments, err := s.teamAccessRepo.ListAssignmentsForUser(ctx, tenantUser, app, username)
	if err != nil {
		return false, err
	}
	now := time.Now()
	for _, assignment := range assignments {
		if assignment.ExpiresAt == nil || assignment.ExpiresAt.After(now) {
			return true, nil
		}
	}
	return false, nil
}

func (s *TeamAccessService) ListAccessibleApps(ctx context.Context, username string) ([]*model.App, error) {
	if strings.TrimSpace(username) == "" || s.appRepo == nil {
		return []*model.App{}, nil
	}
	assignments, err := s.teamAccessRepo.ListActiveAssignmentsForUsername(ctx, username, time.Now())
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	pairs := make([][2]string, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil || assignment.TenantUser == "" || assignment.App == "" {
			continue
		}
		key := assignment.TenantUser + "/" + assignment.App
		if seen[key] {
			continue
		}
		seen[key] = true
		pairs = append(pairs, [2]string{assignment.TenantUser, assignment.App})
	}
	grantedApps, err := s.appRepo.GetAppsByUserAppPairs(ctx, pairs)
	if err != nil {
		return nil, err
	}
	openApps, err := s.appRepo.GetAppsByAccessMode(ctx, model.AppAccessModeOpenCollaboration)
	if err != nil {
		return nil, err
	}
	apps := make([]*model.App, 0, len(grantedApps)+len(openApps))
	seen = map[string]bool{}
	for _, appModel := range append(grantedApps, openApps...) {
		if appModel == nil || appModel.IsDisabled() {
			continue
		}
		key := appModel.User + "/" + appModel.Code
		if seen[key] {
			continue
		}
		seen[key] = true
		apps = append(apps, appModel)
	}
	return apps, nil
}

func (s *TeamAccessService) PermissionsForTree(ctx context.Context, tenantUser, app, username string, resourcePaths []string) (map[string]*access.Result, error) {
	results := make(map[string]*access.Result, len(resourcePaths))
	if username == "" {
		for _, path := range resourcePaths {
			normalized := access.NormalizeResourcePath(path)
			results[normalized] = &access.Result{ResourcePath: normalized, Permissions: access.EmptyPermissionSet()}
		}
		return results, nil
	}
	if isSystemRootUser(username) {
		for _, path := range resourcePaths {
			normalized := access.NormalizeResourcePath(path)
			results[normalized] = ownerAccessResult(normalized)
		}
		return results, nil
	}
	if s.isWorkspaceOwnerOrLegacyAdmin(ctx, tenantUser, app, username) {
		for _, path := range resourcePaths {
			normalized := access.NormalizeResourcePath(path)
			results[normalized] = ownerAccessResult(normalized)
		}
		return results, nil
	}
	assignments, err := s.teamAccessRepo.ListAssignmentsForUser(ctx, tenantUser, app, username)
	if err != nil {
		return nil, err
	}
	openCollaboration, err := s.IsOpenCollaborationWorkspace(ctx, tenantUser, app, username)
	if err != nil {
		return nil, err
	}
	accessAssignments := toAccessAssignments(assignments)
	now := time.Now()
	for _, path := range resourcePaths {
		normalized := access.NormalizeResourcePath(path)
		result := access.Resolve(accessAssignments, normalized, now)
		if openCollaboration {
			result.Permissions = access.MergePermissionSets(result.Permissions, openCollaborationDataPermissions())
			if result.InheritedFrom == "" {
				result.InheritedFrom = access.AppRootPath(tenantUser, app)
			}
		}
		results[normalized] = result
	}
	return results, nil
}
