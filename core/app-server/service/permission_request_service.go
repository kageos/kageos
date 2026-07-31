package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

type PermissionApproverView struct {
	PrincipalType access.PrincipalType `json:"principal_type"`
	PrincipalKey  string               `json:"principal_key"`
	RoleCode      access.RoleCode      `json:"role_code"`
	ResourcePath  string               `json:"resource_path"`
	Inherited     bool                 `json:"inherited"`
}

type PermissionRequestView struct {
	ID                 int64                    `json:"id"`
	TenantUser         string                   `json:"tenant_user"`
	App                string                   `json:"app"`
	Requester          string                   `json:"requester"`
	ResourcePath       string                   `json:"resource_path"`
	RequestedRole      access.RoleCode          `json:"requested_role"`
	Reason             string                   `json:"reason"`
	Status             string                   `json:"status"`
	ReviewedBy         string                   `json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time               `json:"reviewed_at,omitempty"`
	ReviewComment      string                   `json:"review_comment,omitempty"`
	RequestedExpiresAt *time.Time               `json:"requested_expires_at,omitempty"`
	CreatedAt          models.Time              `json:"created_at"`
	UpdatedAt          models.Time              `json:"updated_at"`
	Approvers          []PermissionApproverView `json:"approvers,omitempty"`
}

type PermissionRequestService struct {
	requestRepo        *repository.PermissionRequestRepository
	roleAssignmentRepo *repository.RoleAssignmentRepository
	serviceTreeRepo    *repository.ServiceTreeRepository
	appRepo            *repository.AppRepository
	permission         *PermissionService
}

func NewPermissionRequestService(
	requestRepo *repository.PermissionRequestRepository,
	roleAssignmentRepo *repository.RoleAssignmentRepository,
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	permission *PermissionService,
) *PermissionRequestService {
	return &PermissionRequestService{
		requestRepo:        requestRepo,
		roleAssignmentRepo: roleAssignmentRepo,
		serviceTreeRepo:    serviceTreeRepo,
		appRepo:            appRepo,
		permission:         permission,
	}
}

func (s *PermissionRequestService) CreateRequest(
	ctx context.Context,
	tenantUser, app, requester, resourcePath string,
	roleCode access.RoleCode,
	reason string,
	expiresAt *time.Time,
) (*PermissionRequestView, error) {
	tenantUser = strings.TrimSpace(tenantUser)
	app = strings.TrimSpace(app)
	requester = strings.ToLower(strings.TrimSpace(requester))
	resourcePath = access.NormalizeResourcePath(resourcePath)
	roleCode = access.NormalizeRoleCode(roleCode)
	reason = strings.TrimSpace(reason)

	if tenantUser == "" || app == "" || requester == "" || resourcePath == "" {
		return nil, fmt.Errorf("tenant_user、app、requester、resource_path 不能为空")
	}
	if roleCode != access.RoleViewer && roleCode != access.RoleMember && roleCode != access.RoleAdmin {
		return nil, fmt.Errorf("自助申请仅支持 Viewer、Member 或 Admin")
	}
	if reason == "" {
		return nil, fmt.Errorf("请填写申请理由")
	}
	if len([]rune(reason)) > 1000 {
		return nil, fmt.Errorf("申请理由不能超过 1000 个字符")
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return nil, fmt.Errorf("权限到期时间必须晚于当前时间")
	}
	pathTenant, pathApp, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return nil, err
	}
	if pathTenant != tenantUser || pathApp != app {
		return nil, fmt.Errorf("resource_path 与 workspace 不匹配: %s", resourcePath)
	}
	if s.appRepo == nil {
		return nil, fmt.Errorf("权限申请服务未初始化")
	}
	if _, err := s.appRepo.GetAppByUserName(tenantUser, app); err != nil {
		return nil, fmt.Errorf("工作空间不存在: %w", err)
	}
	if s.serviceTreeRepo != nil && resourcePath != access.AppRootPath(tenantUser, app) {
		if _, err := s.serviceTreeRepo.GetServiceTreeByFullPath(resourcePath); err != nil {
			return nil, fmt.Errorf("申请资源不存在: %s", resourcePath)
		}
	}

	resolved, err := s.permission.ResolvePermissions(ctx, tenantUser, app, requester, resourcePath)
	if err != nil {
		return nil, err
	}
	if permissionSetCoversRole(resolved.Permissions, roleCode) {
		return nil, fmt.Errorf("当前账号已具备 %s 权限，无需重复申请", roleCode)
	}
	if existing, err := s.requestRepo.FindPending(ctx, tenantUser, app, requester, resourcePath); err == nil {
		view, viewErr := s.buildRequestView(ctx, existing)
		if viewErr != nil {
			return nil, viewErr
		}
		return view, fmt.Errorf("该资源已有待审批申请")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	pendingKey := permissionRequestPendingKey(tenantUser, app, requester, resourcePath)
	request := &model.WorkspacePermissionRequest{
		TenantUser:       tenantUser,
		App:              app,
		Requester:        requester,
		ResourcePath:     resourcePath,
		RequestedRole:    string(roleCode),
		Reason:           reason,
		Status:           model.PermissionRequestStatusPending,
		PendingKey:       &pendingKey,
		RequestedExpires: expiresAt,
	}
	request.CreatedBy = requester
	request.UpdatedBy = requester
	if err := s.requestRepo.Create(ctx, request); err != nil {
		if existing, findErr := s.requestRepo.FindPending(ctx, tenantUser, app, requester, resourcePath); findErr == nil {
			view, viewErr := s.buildRequestView(ctx, existing)
			if viewErr != nil {
				return nil, viewErr
			}
			return view, fmt.Errorf("该资源已有待审批申请")
		}
		return nil, err
	}

	s.permission.writeOperateLog(ctx, operateLogInput{
		TenantUser:   tenantUser,
		App:          app,
		ActorUser:    requester,
		Action:       "permission.request.created",
		ResourceType: "permission_request",
		ResourcePath: resourcePath,
		TargetUser:   requester,
		TargetID:     strconv.FormatInt(request.ID, 10),
		Summary:      fmt.Sprintf("%s requested %s on %s", requester, roleCode, resourcePath),
		Status:       "success",
		NewValues: map[string]any{
			"requested_role":       roleCode,
			"reason":               reason,
			"requested_expires_at": expiresAt,
		},
	})
	return s.buildRequestView(ctx, request)
}

func (s *PermissionRequestService) ListMine(
	ctx context.Context,
	tenantUser, app, requester, status string,
) ([]PermissionRequestView, error) {
	status = normalizePermissionRequestStatus(status)
	requests, err := s.requestRepo.ListMine(ctx, tenantUser, app, strings.ToLower(strings.TrimSpace(requester)), status)
	if err != nil {
		return nil, err
	}
	return s.buildRequestViews(ctx, requests)
}

func (s *PermissionRequestService) ListPendingForReviewer(
	ctx context.Context,
	tenantUser, app, reviewer string,
) ([]PermissionRequestView, error) {
	requests, err := s.requestRepo.ListWorkspace(ctx, tenantUser, app, []string{model.PermissionRequestStatusPending})
	if err != nil {
		return nil, err
	}
	return s.filterReviewableRequests(ctx, requests, reviewer)
}

func (s *PermissionRequestService) ListHistoryForReviewer(
	ctx context.Context,
	tenantUser, app, reviewer string,
) ([]PermissionRequestView, error) {
	requests, err := s.requestRepo.ListWorkspace(ctx, tenantUser, app, []string{
		model.PermissionRequestStatusApproved,
		model.PermissionRequestStatusRejected,
		model.PermissionRequestStatusCancelled,
	})
	if err != nil {
		return nil, err
	}
	return s.filterReviewableRequests(ctx, requests, reviewer)
}

func (s *PermissionRequestService) CountPendingForReviewer(
	ctx context.Context,
	tenantUser, app, reviewer string,
) (int, error) {
	requests, err := s.ListPendingForReviewer(ctx, tenantUser, app, reviewer)
	if err != nil {
		return 0, err
	}
	return len(requests), nil
}

func (s *PermissionRequestService) Approvers(
	ctx context.Context,
	tenantUser, app, resourcePath string,
) ([]PermissionApproverView, error) {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	appModel, err := s.appRepo.GetAppByUserName(tenantUser, app)
	if err != nil {
		return nil, err
	}

	approvers := make([]PermissionApproverView, 0, 4)
	seen := map[string]struct{}{}
	appendApprover := func(principalType access.PrincipalType, principalKey string, roleCode access.RoleCode, sourcePath string) {
		principal := access.NormalizePrincipal(access.Principal{Type: principalType, Key: principalKey})
		if !access.IsValidPrincipal(principal) {
			return
		}
		identity := string(principal.Type) + "\x00" + principal.Key
		if _, ok := seen[identity]; ok {
			return
		}
		seen[identity] = struct{}{}
		approvers = append(approvers, PermissionApproverView{
			PrincipalType: principal.Type,
			PrincipalKey:  principal.Key,
			RoleCode:      roleCode,
			ResourcePath:  access.NormalizeResourcePath(sourcePath),
			Inherited:     access.NormalizeResourcePath(sourcePath) != resourcePath,
		})
	}

	appendApprover(access.PrincipalUser, tenantUser, access.RoleOwner, access.AppRootPath(tenantUser, app))
	for _, username := range strings.Split(appModel.Admins, ",") {
		appendApprover(access.PrincipalUser, username, access.RoleAdmin, access.AppRootPath(tenantUser, app))
	}

	assignments, err := s.roleAssignmentRepo.ListAssignmentsForWorkspace(ctx, tenantUser, app)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for _, assignment := range assignments {
		if assignment == nil || (assignment.ExpiresAt != nil && !assignment.ExpiresAt.After(now)) {
			continue
		}
		roleCode := access.NormalizeRoleCode(access.RoleCode(assignment.RoleCode))
		if roleCode != access.RoleAdmin && roleCode != access.RoleOwner {
			continue
		}
		sourcePath := access.NormalizeResourcePath(assignment.ResourcePath)
		if !access.PathApplies(sourcePath, resourcePath) {
			continue
		}
		appendApprover(
			access.PrincipalType(assignment.PrincipalType),
			assignment.PrincipalKey,
			roleCode,
			sourcePath,
		)
	}

	sort.SliceStable(approvers, func(i, j int) bool {
		if approvers[i].PrincipalType != approvers[j].PrincipalType {
			return approvers[i].PrincipalType == access.PrincipalUser
		}
		return approvers[i].PrincipalKey < approvers[j].PrincipalKey
	})
	return approvers, nil
}

func (s *PermissionRequestService) Approve(
	ctx context.Context,
	id int64,
	reviewer, comment string,
) (*PermissionRequestView, error) {
	return s.review(ctx, id, reviewer, model.PermissionRequestStatusApproved, comment)
}

func (s *PermissionRequestService) Reject(
	ctx context.Context,
	id int64,
	reviewer, comment string,
) (*PermissionRequestView, error) {
	if strings.TrimSpace(comment) == "" {
		return nil, fmt.Errorf("请填写驳回原因")
	}
	return s.review(ctx, id, reviewer, model.PermissionRequestStatusRejected, comment)
}

func (s *PermissionRequestService) Cancel(
	ctx context.Context,
	id int64,
	requester string,
) (*PermissionRequestView, error) {
	requester = strings.ToLower(strings.TrimSpace(requester))
	request, err := s.requestRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if request.Requester != requester {
		return nil, fmt.Errorf("只能撤销自己的权限申请")
	}
	if request.Status != model.PermissionRequestStatusPending {
		return nil, fmt.Errorf("该权限申请已处理，不能撤销")
	}

	now := time.Now()
	err = s.requestRepo.Transaction(ctx, func(tx *gorm.DB) error {
		return s.requestRepo.UpdateDecisionWithDB(ctx, tx, id, model.PermissionRequestStatusPending, map[string]any{
			"status":         model.PermissionRequestStatusCancelled,
			"pending_key":    nil,
			"reviewed_by":    requester,
			"reviewed_at":    now,
			"review_comment": "申请人撤销",
			"updated_by":     requester,
		})
	})
	if errors.Is(err, repository.ErrPermissionRequestStateChanged) {
		return nil, fmt.Errorf("该权限申请已处理")
	}
	if err != nil {
		return nil, err
	}
	request.Status = model.PermissionRequestStatusCancelled
	request.PendingKey = nil
	request.ReviewedBy = requester
	request.ReviewedAt = &now
	request.ReviewComment = "申请人撤销"
	request.UpdatedBy = requester
	s.writeReviewOperateLog(ctx, request, requester)
	return s.buildRequestView(ctx, request)
}

func (s *PermissionRequestService) review(
	ctx context.Context,
	id int64,
	reviewer, status, comment string,
) (*PermissionRequestView, error) {
	reviewer = strings.ToLower(strings.TrimSpace(reviewer))
	comment = strings.TrimSpace(comment)
	request, err := s.requestRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if request.Status != model.PermissionRequestStatusPending {
		return nil, fmt.Errorf("该权限申请已处理")
	}
	if request.Requester == reviewer {
		return nil, fmt.Errorf("不能审批自己的权限申请")
	}
	if err := s.permission.RequirePermission(
		ctx,
		request.TenantUser,
		request.App,
		reviewer,
		request.ResourcePath,
		access.ActionAdmin,
	); err != nil {
		return nil, fmt.Errorf("当前账号不是该资源的审批人: %w", err)
	}

	now := time.Now()
	err = s.requestRepo.Transaction(ctx, func(tx *gorm.DB) error {
		current, err := s.requestRepo.GetByIDWithDB(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.Status != model.PermissionRequestStatusPending {
			return repository.ErrPermissionRequestStateChanged
		}

		if status == model.PermissionRequestStatusApproved {
			roleCode := access.NormalizeRoleCode(access.RoleCode(current.RequestedRole))
			if roleCode != access.RoleViewer && roleCode != access.RoleMember && roleCode != access.RoleAdmin {
				return fmt.Errorf("权限申请角色无效: %s", current.RequestedRole)
			}
			if current.RequestedExpires != nil && !current.RequestedExpires.After(now) {
				return fmt.Errorf("申请的权限有效期已过，请驳回后由申请人重新提交")
			}
			assignment := &model.WorkspaceRoleAssignment{
				TenantUser:    current.TenantUser,
				App:           current.App,
				PrincipalType: string(access.PrincipalUser),
				PrincipalKey:  current.Requester,
				ResourcePath:  access.NormalizeResourcePath(current.ResourcePath),
				RoleCode:      string(roleCode),
				ExpiresAt:     current.RequestedExpires,
			}
			assignment.CreatedBy = reviewer
			assignment.UpdatedBy = reviewer
			if err := repository.NewRoleAssignmentRepository(tx).UpsertAssignment(ctx, assignment); err != nil {
				return err
			}
		}

		return s.requestRepo.UpdateDecisionWithDB(ctx, tx, id, model.PermissionRequestStatusPending, map[string]any{
			"status":         status,
			"pending_key":    nil,
			"reviewed_by":    reviewer,
			"reviewed_at":    now,
			"review_comment": comment,
			"updated_by":     reviewer,
		})
	})
	if errors.Is(err, repository.ErrPermissionRequestStateChanged) {
		return nil, fmt.Errorf("该权限申请已处理")
	}
	if err != nil {
		return nil, err
	}

	request.Status = status
	request.PendingKey = nil
	request.ReviewedBy = reviewer
	request.ReviewedAt = &now
	request.ReviewComment = comment
	request.UpdatedBy = reviewer
	s.writeReviewOperateLog(ctx, request, reviewer)
	return s.buildRequestView(ctx, request)
}

func (s *PermissionRequestService) filterReviewableRequests(
	ctx context.Context,
	requests []*model.WorkspacePermissionRequest,
	reviewer string,
) ([]PermissionRequestView, error) {
	reviewer = strings.ToLower(strings.TrimSpace(reviewer))
	reviewable := make([]*model.WorkspacePermissionRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		if request.ReviewedBy == reviewer {
			reviewable = append(reviewable, request)
			continue
		}
		canReview, err := s.permission.HasPermission(
			ctx,
			request.TenantUser,
			request.App,
			reviewer,
			request.ResourcePath,
			access.ActionAdmin,
		)
		if err != nil {
			return nil, err
		}
		if canReview {
			reviewable = append(reviewable, request)
		}
	}
	return s.buildRequestViews(ctx, reviewable)
}

func (s *PermissionRequestService) buildRequestViews(
	ctx context.Context,
	requests []*model.WorkspacePermissionRequest,
) ([]PermissionRequestView, error) {
	views := make([]PermissionRequestView, 0, len(requests))
	approversByPath := make(map[string][]PermissionApproverView)
	for _, request := range requests {
		if request == nil {
			continue
		}
		path := access.NormalizeResourcePath(request.ResourcePath)
		approvers, ok := approversByPath[path]
		if !ok {
			var err error
			approvers, err = s.Approvers(ctx, request.TenantUser, request.App, path)
			if err != nil {
				return nil, err
			}
			approversByPath[path] = approvers
		}
		view := permissionRequestViewFromModel(request)
		view.Approvers = approvers
		views = append(views, view)
	}
	return views, nil
}

func (s *PermissionRequestService) buildRequestView(
	ctx context.Context,
	request *model.WorkspacePermissionRequest,
) (*PermissionRequestView, error) {
	if request == nil {
		return nil, nil
	}
	approvers, err := s.Approvers(ctx, request.TenantUser, request.App, request.ResourcePath)
	if err != nil {
		return nil, err
	}
	view := permissionRequestViewFromModel(request)
	view.Approvers = approvers
	return &view, nil
}

func (s *PermissionRequestService) writeReviewOperateLog(
	ctx context.Context,
	request *model.WorkspacePermissionRequest,
	actor string,
) {
	if request == nil {
		return
	}
	s.permission.writeOperateLog(ctx, operateLogInput{
		TenantUser:   request.TenantUser,
		App:          request.App,
		ActorUser:    actor,
		Action:       "permission.request." + request.Status,
		ResourceType: "permission_request",
		ResourcePath: request.ResourcePath,
		TargetUser:   request.Requester,
		TargetID:     strconv.FormatInt(request.ID, 10),
		Summary:      fmt.Sprintf("%s marked permission request %d as %s", actor, request.ID, request.Status),
		Status:       "success",
		NewValues: map[string]any{
			"requester":        request.Requester,
			"requested_role":   request.RequestedRole,
			"request_status":   request.Status,
			"reviewed_by":      request.ReviewedBy,
			"reviewed_at":      request.ReviewedAt,
			"review_comment":   request.ReviewComment,
			"permission_until": request.RequestedExpires,
		},
	})
}

func permissionRequestViewFromModel(request *model.WorkspacePermissionRequest) PermissionRequestView {
	return PermissionRequestView{
		ID:                 request.ID,
		TenantUser:         request.TenantUser,
		App:                request.App,
		Requester:          request.Requester,
		ResourcePath:       access.NormalizeResourcePath(request.ResourcePath),
		RequestedRole:      access.NormalizeRoleCode(access.RoleCode(request.RequestedRole)),
		Reason:             request.Reason,
		Status:             request.Status,
		ReviewedBy:         request.ReviewedBy,
		ReviewedAt:         request.ReviewedAt,
		ReviewComment:      request.ReviewComment,
		RequestedExpiresAt: request.RequestedExpires,
		CreatedAt:          request.CreatedAt,
		UpdatedAt:          request.UpdatedAt,
	}
}

func permissionRequestPendingKey(tenantUser, app, requester, resourcePath string) string {
	payload := strings.Join([]string{
		strings.TrimSpace(tenantUser),
		strings.TrimSpace(app),
		strings.ToLower(strings.TrimSpace(requester)),
		access.NormalizeResourcePath(resourcePath),
	}, "\x00")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
}

func permissionSetCoversRole(current access.PermissionSet, roleCode access.RoleCode) bool {
	for action, required := range access.RolePermissions(roleCode) {
		if required && !access.HasPermission(current, action) {
			return false
		}
	}
	return true
}

func normalizePermissionRequestStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case model.PermissionRequestStatusPending,
		model.PermissionRequestStatusApproved,
		model.PermissionRequestStatusRejected,
		model.PermissionRequestStatusCancelled:
		return status
	default:
		return ""
	}
}
