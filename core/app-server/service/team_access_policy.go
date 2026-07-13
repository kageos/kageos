package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/apperror"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

func isOpenCollaborationDataAction(action access.Action) bool {
	switch access.NormalizeAction(action) {
	case access.ActionRead, access.ActionWrite, access.ActionUpdate:
		return true
	default:
		return false
	}
}

func openCollaborationDataPermissions() access.PermissionSet {
	permissions := access.EmptyPermissionSet()
	permissions[access.ActionRead] = true
	permissions[access.ActionWrite] = true
	permissions[access.ActionUpdate] = true
	return permissions
}

func (s *TeamAccessService) isWorkspaceOwner(ctx context.Context, tenantUser, app, username string) bool {
	tenantUser = strings.TrimSpace(tenantUser)
	username = strings.TrimSpace(username)
	return tenantUser != "" && tenantUser == username
}

func isSystemRootUser(username string) bool {
	return strings.TrimSpace(username) == "system"
}

func ownerAccessResult(resourcePath string) *access.Result {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	return &access.Result{
		ResourcePath: resourcePath,
		RoleCodes:    []access.RoleCode{access.RoleOwner},
		Permissions:  access.RolePermissions(access.RoleOwner),
	}
}

func (s *TeamAccessService) isWorkspaceOwnerOrLegacyAdmin(ctx context.Context, tenantUser, app, username string) bool {
	if s.isWorkspaceOwner(ctx, tenantUser, app, username) {
		return true
	}
	if s.appRepo == nil || tenantUser == "" || app == "" || username == "" {
		return false
	}
	appModel, err := s.appRepo.GetAppByUserNameContext(ctx, tenantUser, app)
	if err != nil {
		logger.Debugf(ctx, "[TeamAccess] load app failed for legacy admin fallback: %s/%s err=%v", tenantUser, app, err)
		return false
	}
	return appModel.IsOwnerOrAdmin(username)
}

func (s *TeamAccessService) requireAdminForGrant(ctx context.Context, tenantUser, app, actor, resourcePath string, roleCode access.RoleCode) error {
	if actor == "" {
		return apperror.Unauthenticated("无法获取操作者", nil)
	}
	if roleCode == access.RoleOwner && !s.isWorkspaceOwner(ctx, tenantUser, app, actor) {
		return apperror.PermissionDenied("只有 Owner 可以授予 Owner 角色", nil)
	}
	return s.Check(ctx, tenantUser, app, actor, resourcePath, access.ActionAdmin)
}

func (s *TeamAccessService) ensureAssignableUserInRequesterCompany(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return apperror.InvalidArgument("username 不能为空", nil)
	}
	companyCode := strings.TrimSpace(contextx.GetRequestCompanyCode(ctx))
	if companyCode == "" || s.userLookup == nil {
		return nil
	}
	user, err := s.userLookup(ctx, username)
	if err != nil {
		return apperror.NotFound("被授权用户不存在或不属于当前企业", err)
	}
	if user == nil || strings.TrimSpace(user.Username) == "" {
		return apperror.NotFound("被授权用户不存在或不属于当前企业", nil)
	}
	if user.CompanyCode != "" && user.CompanyCode != companyCode {
		return apperror.NotFound("被授权用户不存在或不属于当前企业", nil)
	}
	return nil
}

func lookupUserForTeamAccess(ctx context.Context, username string) (*dto.UserInfo, error) {
	return apicall.GetUserByUsername(ctx, &dto.QueryUserReq{Username: username})
}

func validateAssignRoleRequest(req access.AssignRoleRequest) error {
	if req.TenantUser == "" || req.App == "" || req.Username == "" || req.ResourcePath == "" {
		return apperror.InvalidArgument("tenant_user、app、username、resource_path 不能为空", nil)
	}
	if !access.IsValidRoleCode(req.RoleCode) {
		return apperror.InvalidArgument(fmt.Sprintf("无效角色: %s", req.RoleCode), nil)
	}
	resourceTenant, resourceApp, err := access.ParseUserApp(req.ResourcePath)
	if err != nil {
		return err
	}
	if resourceTenant != req.TenantUser || resourceApp != req.App {
		return apperror.InvalidArgument(fmt.Sprintf("resource_path 与 workspace 不匹配: %s", req.ResourcePath), nil)
	}
	return nil
}

func toAccessAssignments(assignments []*model.WorkspaceRoleAssignment) []access.Assignment {
	result := make([]access.Assignment, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}
		result = append(result, access.Assignment{
			TenantUser:   assignment.TenantUser,
			App:          assignment.App,
			Username:     assignment.Username,
			ResourcePath: assignment.ResourcePath,
			RoleCode:     access.RoleCode(assignment.RoleCode),
			ExpiresAt:    assignment.ExpiresAt,
			CreatedBy:    assignment.CreatedBy,
		})
	}
	return result
}
