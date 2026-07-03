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
		return fmt.Errorf("无权限执行 %s: %s", action, access.NormalizeResourcePath(resourcePath))
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
		return fmt.Errorf("resource_paths、usernames、role_codes 不能为空")
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
		return fmt.Errorf("tenant_user、app、username、resource_path 不能为空")
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
	return s.appRepo.GetAppsByUserAppPairs(pairs)
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
	accessAssignments := toAccessAssignments(assignments)
	now := time.Now()
	for _, path := range resourcePaths {
		normalized := access.NormalizeResourcePath(path)
		results[normalized] = access.Resolve(accessAssignments, normalized, now)
	}
	return results, nil
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
	appModel, err := s.appRepo.GetAppByUserName(tenantUser, app)
	if err != nil {
		logger.Debugf(ctx, "[TeamAccess] load app failed for legacy admin fallback: %s/%s err=%v", tenantUser, app, err)
		return false
	}
	return appModel.IsOwnerOrAdmin(username)
}

func (s *TeamAccessService) requireAdminForGrant(ctx context.Context, tenantUser, app, actor, resourcePath string, roleCode access.RoleCode) error {
	if actor == "" {
		return fmt.Errorf("无法获取操作者")
	}
	if roleCode == access.RoleOwner && !s.isWorkspaceOwner(ctx, tenantUser, app, actor) {
		return fmt.Errorf("只有 Owner 可以授予 Owner 角色")
	}
	return s.Check(ctx, tenantUser, app, actor, resourcePath, access.ActionAdmin)
}

func (s *TeamAccessService) ensureAssignableUserInRequesterCompany(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username 不能为空")
	}
	companyCode := strings.TrimSpace(contextx.GetRequestCompanyCode(ctx))
	if companyCode == "" || s.userLookup == nil {
		return nil
	}
	user, err := s.userLookup(ctx, username)
	if err != nil {
		return fmt.Errorf("被授权用户不存在或不属于当前企业: %w", err)
	}
	if user == nil || strings.TrimSpace(user.Username) == "" {
		return fmt.Errorf("被授权用户不存在或不属于当前企业")
	}
	if user.CompanyCode != "" && user.CompanyCode != companyCode {
		return fmt.Errorf("被授权用户不存在或不属于当前企业")
	}
	return nil
}

func lookupUserForTeamAccess(ctx context.Context, username string) (*dto.UserInfo, error) {
	return apicall.GetUserByUsername(ctx, &dto.QueryUserReq{Username: username})
}

func validateAssignRoleRequest(req access.AssignRoleRequest) error {
	if req.TenantUser == "" || req.App == "" || req.Username == "" || req.ResourcePath == "" {
		return fmt.Errorf("tenant_user、app、username、resource_path 不能为空")
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

	go func() {
		if err := s.operateLogRepo.CreateOperateLog(context.Background(), log); err != nil {
			logger.Warnf(ctx, "[TeamAccess] write operate log failed: action=%s err=%v", input.Action, err)
		}
	}()
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
