package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

type PermissionService struct {
	roleAssignmentRepo *repository.RoleAssignmentRepository
	operateLogRepo     *repository.OperateLogRepository
	appRepo            *repository.AppRepository
	userLookup         func(ctx context.Context, username string) (*dto.UserInfo, error)
	departmentLookup   func(ctx context.Context, departmentPath string) (bool, error)
}

func NewPermissionService(
	roleAssignmentRepo *repository.RoleAssignmentRepository,
	operateLogRepo *repository.OperateLogRepository,
	appRepo *repository.AppRepository,
) *PermissionService {
	return &PermissionService{
		roleAssignmentRepo: roleAssignmentRepo,
		operateLogRepo:     operateLogRepo,
		appRepo:            appRepo,
		userLookup:         lookupUserForPermission,
		departmentLookup:   lookupDepartmentForPermission,
	}
}

func (s *PermissionService) RequirePermission(ctx context.Context, tenantUser, app, username, resourcePath string, action access.Action) error {
	ok, err := s.HasPermission(ctx, tenantUser, app, username, resourcePath, action)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("无权限执行 %s: %s", action, access.NormalizeResourcePath(resourcePath))
	}
	return nil
}

func (s *PermissionService) HasPermission(ctx context.Context, tenantUser, app, username, resourcePath string, action access.Action) (bool, error) {
	result, err := s.ResolvePermissions(ctx, tenantUser, app, username, resourcePath)
	if err != nil {
		return false, err
	}
	return access.HasPermission(result.Permissions, action), nil
}

func (s *PermissionService) ResolvePermissions(ctx context.Context, tenantUser, app, username, resourcePath string) (*access.Result, error) {
	tenantUser = strings.TrimSpace(tenantUser)
	app = strings.TrimSpace(app)
	username = strings.TrimSpace(username)
	resourcePath = access.NormalizeResourcePath(resourcePath)
	if isSystemRootUser(username) {
		return ownerAccessResult(resourcePath), nil
	}
	if tenantUser == "" || app == "" || username == "" || resourcePath == "" {
		return &access.Result{
			ResourcePath: resourcePath,
			Permissions:  access.EmptyPermissionSet(),
		}, nil
	}

	if s.isWorkspaceOwnerOrLegacyAdmin(ctx, tenantUser, app, username) {
		return &access.Result{
			ResourcePath: resourcePath,
			RoleCodes:    []access.RoleCode{access.RoleOwner},
			Permissions:  access.RolePermissions(access.RoleOwner),
		}, nil
	}

	principals := s.principalsForUser(ctx, username)
	assignments, err := s.roleAssignmentRepo.ListAssignmentsForPrincipals(ctx, tenantUser, app, principals)
	if err != nil {
		return nil, err
	}
	return access.Resolve(toAccessAssignments(assignments), resourcePath, time.Now()), nil
}

func (s *PermissionService) GrantRole(ctx context.Context, req access.GrantRoleRequest) error {
	req.ResourcePath = access.NormalizeResourcePath(req.ResourcePath)
	req.RoleCode = access.NormalizeRoleCode(req.RoleCode)
	req.Principal = access.NormalizePrincipal(req.Principal)
	req.TenantUser = strings.TrimSpace(req.TenantUser)
	req.App = strings.TrimSpace(req.App)
	if err := validateGrantRoleRequest(req); err != nil {
		return err
	}
	if req.CreatedBy == "" {
		req.CreatedBy = contextx.GetRequestUser(ctx)
	}
	if err := s.requireAdminForGrant(ctx, req.TenantUser, req.App, req.CreatedBy, req.ResourcePath, req.RoleCode); err != nil {
		return err
	}
	if err := s.ensureAssignablePrincipal(ctx, req.Principal); err != nil {
		return err
	}

	assignment := &model.WorkspaceRoleAssignment{
		TenantUser:    req.TenantUser,
		App:           req.App,
		PrincipalType: string(req.Principal.Type),
		PrincipalKey:  req.Principal.Key,
		ResourcePath:  req.ResourcePath,
		RoleCode:      string(req.RoleCode),
		ExpiresAt:     req.ExpiresAt,
	}
	assignment.CreatedBy = req.CreatedBy
	if err := s.roleAssignmentRepo.UpsertAssignment(ctx, assignment); err != nil {
		return err
	}

	s.writeOperateLog(ctx, operateLogInput{
		TenantUser:   req.TenantUser,
		App:          req.App,
		ActorUser:    req.CreatedBy,
		Action:       "permission.role.granted",
		ResourceType: "permission",
		ResourcePath: req.ResourcePath,
		TargetUser:   principalTargetUser(req.Principal),
		TargetID:     principalAuditID(req.Principal),
		Summary:      fmt.Sprintf("%s granted %s to %s on %s", req.CreatedBy, req.RoleCode, principalAuditID(req.Principal), req.ResourcePath),
		Status:       "success",
		NewValues: dto.PermissionRoleGrantedValues{
			PrincipalType: string(req.Principal.Type),
			PrincipalKey:  req.Principal.Key,
			RoleCode:      string(req.RoleCode),
			ExpiresAt:     req.ExpiresAt,
		},
	})
	return nil
}

func (s *PermissionService) BatchGrantRoles(ctx context.Context, req access.BatchGrantRoleRequest) error {
	if len(req.ResourcePaths) == 0 || len(req.Principals) == 0 || len(req.RoleCodes) == 0 {
		return fmt.Errorf("resource_paths、principals、role_codes 不能为空")
	}
	for _, resourcePath := range req.ResourcePaths {
		for _, principal := range req.Principals {
			for _, roleCode := range req.RoleCodes {
				if err := s.GrantRole(ctx, access.GrantRoleRequest{
					TenantUser:   req.TenantUser,
					App:          req.App,
					Principal:    principal,
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

func (s *PermissionService) RevokeRole(ctx context.Context, req access.RevokeRoleRequest) error {
	req.ResourcePath = access.NormalizeResourcePath(req.ResourcePath)
	req.RoleCode = access.NormalizeRoleCode(req.RoleCode)
	req.Principal = access.NormalizePrincipal(req.Principal)
	req.TenantUser = strings.TrimSpace(req.TenantUser)
	req.App = strings.TrimSpace(req.App)
	if req.TenantUser == "" || req.App == "" || !access.IsValidPrincipal(req.Principal) || req.ResourcePath == "" {
		return fmt.Errorf("tenant_user、app、principal、resource_path 不能为空")
	}
	if req.Actor == "" {
		req.Actor = contextx.GetRequestUser(ctx)
	}
	if err := s.requireAdminForGrant(ctx, req.TenantUser, req.App, req.Actor, req.ResourcePath, req.RoleCode); err != nil {
		return err
	}
	if req.RoleCode == access.RoleOwner && !s.isWorkspaceOwner(ctx, req.TenantUser, req.App, req.Actor) {
		return fmt.Errorf("只有 Owner 可以移除 Owner 角色")
	}

	rows, err := s.roleAssignmentRepo.RemoveAssignment(ctx, req.TenantUser, req.App, req.Principal, req.ResourcePath, req.RoleCode)
	if err != nil {
		return err
	}
	s.writeOperateLog(ctx, operateLogInput{
		TenantUser:   req.TenantUser,
		App:          req.App,
		ActorUser:    req.Actor,
		Action:       "permission.role.revoked",
		ResourceType: "permission",
		ResourcePath: req.ResourcePath,
		TargetUser:   principalTargetUser(req.Principal),
		TargetID:     principalAuditID(req.Principal),
		Summary:      fmt.Sprintf("%s revoked role %s from %s on %s", req.Actor, req.RoleCode, principalAuditID(req.Principal), req.ResourcePath),
		Status:       "success",
		Details: dto.PermissionRoleRevokedDetails{
			PrincipalType: string(req.Principal.Type),
			PrincipalKey:  req.Principal.Key,
			RoleCode:      string(req.RoleCode),
			RowsAffected:  rows,
		},
	})
	return nil
}

func (s *PermissionService) ListAssignments(ctx context.Context, tenantUser, app, resourcePath string) ([]access.RoleAssignmentView, error) {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	assignments, err := s.roleAssignmentRepo.ListAssignmentsForWorkspace(ctx, tenantUser, app)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	views := make([]access.RoleAssignmentView, 0, len(assignments))
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
		views = append(views, access.RoleAssignmentView{
			TenantUser:     assignment.TenantUser,
			App:            assignment.App,
			PrincipalType:  access.NormalizePrincipalType(access.PrincipalType(assignment.PrincipalType)),
			PrincipalKey:   assignment.PrincipalKey,
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
	return views, nil
}

func (s *PermissionService) HasAnyWorkspacePermission(ctx context.Context, tenantUser, app, username string) (bool, error) {
	if username == "" {
		return false, nil
	}
	if s.isWorkspaceOwnerOrLegacyAdmin(ctx, tenantUser, app, username) {
		return true, nil
	}
	assignments, err := s.roleAssignmentRepo.ListAssignmentsForPrincipals(ctx, tenantUser, app, s.principalsForUser(ctx, username))
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

func (s *PermissionService) ListAccessibleApps(ctx context.Context, username string) ([]*model.App, error) {
	if strings.TrimSpace(username) == "" || s.appRepo == nil {
		return []*model.App{}, nil
	}
	assignments, err := s.roleAssignmentRepo.ListActiveAssignmentsForPrincipals(ctx, s.principalsForUser(ctx, username), time.Now())
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
	return s.appRepo.GetAppsByUserAppPairs(pairs)
}

func (s *PermissionService) PermissionsForTree(ctx context.Context, tenantUser, app, username string, resourcePaths []string) (map[string]*access.Result, error) {
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
	assignments, err := s.roleAssignmentRepo.ListAssignmentsForPrincipals(ctx, tenantUser, app, s.principalsForUser(ctx, username))
	if err != nil {
		return nil, err
	}
	accessAssignments := toAccessAssignments(assignments)
	now := time.Now()
	for _, path := range resourcePaths {
		normalized := access.NormalizeResourcePath(path)
		results[normalized] = access.Resolve(accessAssignments, normalized, now)
	}
	return results, nil
}

func (s *PermissionService) isWorkspaceOwner(ctx context.Context, tenantUser, app, username string) bool {
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

func (s *PermissionService) isWorkspaceOwnerOrLegacyAdmin(ctx context.Context, tenantUser, app, username string) bool {
	if s.isWorkspaceOwner(ctx, tenantUser, app, username) {
		return true
	}
	if s.appRepo == nil || tenantUser == "" || app == "" || username == "" {
		return false
	}
	appModel, err := s.appRepo.GetAppByUserName(tenantUser, app)
	if err != nil {
		logger.Debugf(ctx, "[Permission] load app failed for legacy admin fallback: %s/%s err=%v", tenantUser, app, err)
		return false
	}
	return appModel.IsOwnerOrAdmin(username)
}

func (s *PermissionService) requireAdminForGrant(ctx context.Context, tenantUser, app, actor, resourcePath string, roleCode access.RoleCode) error {
	if actor == "" {
		return fmt.Errorf("无法获取操作者")
	}
	if roleCode == access.RoleOwner && !s.isWorkspaceOwner(ctx, tenantUser, app, actor) {
		return fmt.Errorf("只有 Owner 可以授予 Owner 角色")
	}
	return s.RequirePermission(ctx, tenantUser, app, actor, resourcePath, access.ActionAdmin)
}

func (s *PermissionService) ensureAssignablePrincipal(ctx context.Context, principal access.Principal) error {
	principal = access.NormalizePrincipal(principal)
	if !access.IsValidPrincipal(principal) {
		return fmt.Errorf("无效授权主体: %s:%s", principal.Type, principal.Key)
	}
	switch principal.Type {
	case access.PrincipalUser:
		if s.userLookup == nil {
			return nil
		}
		user, err := s.userLookup(ctx, principal.Key)
		if err != nil {
			return fmt.Errorf("被授权用户不存在: %w", err)
		}
		if user == nil || strings.TrimSpace(user.Username) == "" {
			return fmt.Errorf("被授权用户不存在")
		}
	case access.PrincipalDepartment:
		if s.departmentLookup == nil {
			return nil
		}
		exists, err := s.departmentLookup(ctx, principal.Key)
		if err != nil {
			return fmt.Errorf("查询授权组织失败: %w", err)
		}
		if !exists {
			return fmt.Errorf("被授权组织不存在: %s", principal.Key)
		}
	}
	return nil
}

func lookupUserForPermission(ctx context.Context, username string) (*dto.UserInfo, error) {
	return apicall.GetUserByUsername(ctx, &dto.QueryUserReq{Username: username})
}

func lookupDepartmentForPermission(ctx context.Context, departmentPath string) (bool, error) {
	result, err := apicall.GetDepartmentsByPaths(ctx, []string{departmentPath})
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, nil
	}
	for _, department := range result.Departments {
		if department != nil && access.NormalizeResourcePath(department.FullCodePath) == access.NormalizeResourcePath(departmentPath) {
			return true, nil
		}
	}
	return false, nil
}

func validateGrantRoleRequest(req access.GrantRoleRequest) error {
	if req.TenantUser == "" || req.App == "" || !access.IsValidPrincipal(req.Principal) || req.ResourcePath == "" {
		return fmt.Errorf("tenant_user、app、principal、resource_path 不能为空")
	}
	if !access.IsValidRoleCode(req.RoleCode) {
		return fmt.Errorf("无效角色: %s", req.RoleCode)
	}
	resourceTenant, resourceApp, err := access.ParseUserApp(req.ResourcePath)
	if err != nil {
		return err
	}
	if resourceTenant != req.TenantUser || resourceApp != req.App {
		return fmt.Errorf("resource_path 与 workspace 不匹配: %s", req.ResourcePath)
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
			TenantUser: assignment.TenantUser,
			App:        assignment.App,
			Principal: access.Principal{
				Type: access.PrincipalType(assignment.PrincipalType),
				Key:  assignment.PrincipalKey,
			},
			ResourcePath: assignment.ResourcePath,
			RoleCode:     access.RoleCode(assignment.RoleCode),
			ExpiresAt:    assignment.ExpiresAt,
			CreatedBy:    assignment.CreatedBy,
		})
	}
	return result
}

func (s *PermissionService) principalsForUser(ctx context.Context, username string) []access.Principal {
	username = strings.ToLower(strings.TrimSpace(username))
	departmentPath := ""
	if username != "" && username == strings.ToLower(strings.TrimSpace(contextx.GetRequestUser(ctx))) {
		departmentPath = contextx.GetRequestDepartmentFullPath(ctx)
	}
	if departmentPath == "" && username != "" && username != "system" && s.userLookup != nil {
		user, err := s.userLookup(ctx, username)
		if err == nil && user != nil {
			departmentPath = user.DepartmentFullPath
		} else if err != nil {
			logger.Debugf(ctx, "[Permission] load organization for %s failed: %v", username, err)
		}
	}
	if departmentPath == "" && username != "" && username != "system" {
		departmentPath = "/org/unassigned"
	}
	return access.PrincipalsForUser(username, departmentPath)
}

func principalTargetUser(principal access.Principal) string {
	principal = access.NormalizePrincipal(principal)
	if principal.Type == access.PrincipalUser {
		return principal.Key
	}
	return ""
}

func principalAuditID(principal access.Principal) string {
	principal = access.NormalizePrincipal(principal)
	return string(principal.Type) + ":" + principal.Key
}

type operateLogInput struct {
	TenantUser   string
	App          string
	ActorUser    string
	Action       string
	ResourceType string
	ResourcePath string
	ResourceName string
	TargetUser   string
	TargetID     string
	Summary      string
	Details      interface{}
	OldValues    interface{}
	NewValues    interface{}
	Status       string
}

func (s *PermissionService) writeOperateLog(ctx context.Context, input operateLogInput) {
	if s.operateLogRepo == nil {
		return
	}
	if input.ActorUser == "" {
		input.ActorUser = contextx.GetRequestUser(ctx)
	}
	if input.Status == "" {
		input.Status = "success"
	}
	auditMeta := buildOperateLogAuditMetadata(ctx, "")
	log := &model.OperateLog{
		TenantUser:   input.TenantUser,
		App:          input.App,
		ActorUser:    input.ActorUser,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourcePath: access.NormalizeResourcePath(input.ResourcePath),
		ResourceName: input.ResourceName,
		TargetUser:   input.TargetUser,
		TargetID:     input.TargetID,
		Summary:      input.Summary,
		Status:       input.Status,
		TraceID:      contextx.GetTraceId(ctx),
	}
	applyOperateLogAuditMetadata(log, auditMeta)
	log.DetailsJSON = mustMarshalRaw(input.Details)
	log.OldValuesJSON = mustMarshalRaw(input.OldValues)
	log.NewValuesJSON = mustMarshalRaw(input.NewValues)

	writeCtx := context.WithoutCancel(ctx)
	go func(ctx context.Context) {
		if err := s.operateLogRepo.CreateOperateLog(ctx, log); err != nil {
			logger.Warnf(ctx, "[Permission] write operate log failed: action=%s err=%v", input.Action, err)
		}
	}(writeCtx)
}

func mustMarshalRaw(v interface{}) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(redactOperateLogValue(v))
	if err != nil {
		return nil
	}
	return data
}
