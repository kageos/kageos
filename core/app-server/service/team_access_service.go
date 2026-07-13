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
	"github.com/kageos/kageos/pkg/apperror"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

type TeamAccessService struct {
	teamAccessRepo *repository.TeamAccessRepository
	operateLogRepo *repository.OperateLogRepository
	appRepo        *repository.AppRepository
	userLookup     func(ctx context.Context, username string) (*dto.UserInfo, error)
}

func NewTeamAccessService(
	teamAccessRepo *repository.TeamAccessRepository,
	operateLogRepo *repository.OperateLogRepository,
	appRepo *repository.AppRepository,
) *TeamAccessService {
	return &TeamAccessService{
		teamAccessRepo: teamAccessRepo,
		operateLogRepo: operateLogRepo,
		appRepo:        appRepo,
		userLookup:     lookupUserForTeamAccess,
	}
}

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

type operateLogInput struct {
	TenantUser   string
	CompanyCode  string
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

func (s *TeamAccessService) writeOperateLog(ctx context.Context, input operateLogInput) {
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
		CompanyCode:  firstNonEmpty(input.CompanyCode, contextx.GetRequestCompanyCode(ctx)),
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
			logger.Warnf(ctx, "[TeamAccess] write operate log failed: action=%s err=%v", input.Action, err)
		}
	}(writeCtx)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
