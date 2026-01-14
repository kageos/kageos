package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	permissionrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	permissionpkg "github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// ApprovalService 审批流程服务（企业版，仅使用角色系统）
// ⭐ 实现 enterprise.ApprovalService 接口
type ApprovalService struct {
	permissionRequestRepo *permissionrepo.PermissionRequestRepository
	serviceTreeRepo       *repository.ServiceTreeRepository // ⭐ 使用社区版的 repository，不重复造轮子
	roleService           *RoleService                       // ⭐ 角色服务，用于分配角色
	roleCache             *RoleCache                         // ⭐ 角色缓存，用于从 RoleID 获取 RoleCode
}

// NewApprovalService 创建审批服务（企业版内部使用）
func NewApprovalService(
	permissionRequestRepo *permissionrepo.PermissionRequestRepository,
	serviceTreeRepo *repository.ServiceTreeRepository, // ⭐ 使用社区版的 repository
	roleService *RoleService, // ⭐ 角色服务
	roleCache *RoleCache, // ⭐ 角色缓存
) *ApprovalService {
	return &ApprovalService{
		permissionRequestRepo: permissionRequestRepo,
		serviceTreeRepo:       serviceTreeRepo,
		roleService:           roleService,
		roleCache:             roleCache,
	}
}

// CreateRequest 创建权限申请
// ⭐ 实现 enterprise.ApprovalService 接口
func (s *ApprovalService) CreateRequest(ctx context.Context, req *enterprise.InternalCreatePermissionRequestReq) (*enterprise.PermissionRequest, error) {
	// 1. 创建申请记录
	request := &model.PermissionRequest{
		AppID:             req.AppID, // ⭐ 使用传入的 AppID
		ApplicantUsername: req.ApplicantUsername,
		SubjectType:       req.SubjectType,
		Subject:           req.Subject,
		ResourcePath:      req.ResourcePath,
		RoleID:            req.RoleID, // ⭐ 角色ID（必填）
		StartTime:         req.StartTime, // ⭐ 直接赋值，无需转换
		EndTime:           req.EndTime,   // ⭐ 直接赋值，无需转换
		Reason:            req.Reason,
		Status:            model.PermissionRequestStatusPending,
	}

	if err := s.permissionRequestRepo.SaveRequest(ctx, request); err != nil {
		return nil, fmt.Errorf("保存申请记录失败: %w", err)
	}

	// 3. 确定审批人（节点管理员）
	approvers, err := s.resolveApprovers(ctx, req.ResourcePath)
	if err != nil {
		logger.Warnf(ctx, "[ApprovalService] 确定审批人失败: resource=%s, error=%v", req.ResourcePath, err)
		// 如果确定审批人失败，申请仍然创建，状态为 pending
		// 后续可以手动分配审批人
	}

	logger.Infof(ctx, "[ApprovalService] 创建角色申请: id=%d, applicant=%s, resource=%s, role_id=%d, approvers=%v",
		request.ID, req.ApplicantUsername, req.ResourcePath, req.RoleID, approvers)

	// 4. 转换为 enterprise.PermissionRequest
	return &enterprise.PermissionRequest{
		ID:                request.ID,
		AppID:             request.AppID,
		ApplicantUsername: request.ApplicantUsername,
		SubjectType:       request.SubjectType,
		Subject:           request.Subject,
		ResourcePath:      request.ResourcePath,
		RoleID:            request.RoleID,
		Status:            request.Status,
	}, nil
}

// ApproveRequest 审批通过
func (s *ApprovalService) ApproveRequest(ctx context.Context, requestID int64, approverUsername string) error {
	// 1. 查询申请记录
	request, err := s.permissionRequestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("查询申请记录失败: %w", err)
	}

	// 2. 检查状态
	if request.Status != model.PermissionRequestStatusPending {
		return fmt.Errorf("申请状态不是待审批，无法审批: status=%s", request.Status)
	}

	// 3. 检查审批人权限（是否是节点管理员）
	admins, err := s.serviceTreeRepo.GetNodeAdmins(ctx, request.ResourcePath)
	if err != nil {
		return fmt.Errorf("查询节点管理员失败: %w", err)
	}

	isAdmin := false
	for _, admin := range admins {
		if admin == approverUsername {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		return fmt.Errorf("用户 %s 不是节点 %s 的管理员，无权审批", approverUsername, request.ResourcePath)
	}

	// 4. 更新申请状态
	if err := s.permissionRequestRepo.UpdateRequestStatus(ctx, requestID, model.PermissionRequestStatusApproved, approverUsername, "", ""); err != nil {
		return fmt.Errorf("更新申请状态失败: %w", err)
	}

	// ⭐ 从 resourcePath 解析 user 和 app
	_, user, app := permissionpkg.ParseFullCodePath(request.ResourcePath)
	if user == "" || app == "" {
		logger.Errorf(ctx, "[ApprovalService] 无法从资源路径解析 user 和 app: resourcePath=%s", request.ResourcePath)
		return fmt.Errorf("无法从资源路径解析 user 和 app: %s", request.ResourcePath)
	}

	logger.Infof(ctx, "[ApprovalService] 解析资源路径: resourcePath=%s, user=%s, app=%s", request.ResourcePath, user, app)

	// 5. ⭐ 分配角色（角色申请）
	// 从 RoleID 获取 RoleCode
	role, exists := s.roleCache.GetRole(request.RoleID)
	if !exists {
		return fmt.Errorf("角色不存在: role_id=%d", request.RoleID)
	}
	
	startTime := models.Time(time.Time(request.StartTime))
	var endTime *models.Time
	if request.EndTime != nil {
		t := models.Time(time.Time(*request.EndTime))
		endTime = &t
	}
	
	var roleAssignmentID *int64
	
	// 分配角色给用户或组织架构
	// ⭐ 从 role 中获取 resourceType（角色已绑定到特定资源类型）
	if request.SubjectType == "user" {
		// 分配角色给用户
		assignReq := &dto.AssignRoleToUserReq{
			User:         user,
			App:          app,
			Username:     request.Subject,
			RoleCode:     role.Code,
			ResourceType: role.ResourceType, // ⭐ 从角色中获取资源类型
			ResourcePath: request.ResourcePath,
			StartTime:    &startTime,
			EndTime:      endTime,
		}
		resp, err := s.roleService.AssignRoleToUser(ctx, assignReq)
		if err != nil {
			logger.Errorf(ctx, "[ApprovalService] 分配角色给用户失败: user=%s, app=%s, username=%s, role_id=%d, error=%v",
				user, app, request.Subject, request.RoleID, err)
			return fmt.Errorf("分配角色给用户失败: %w", err)
		}
		if resp != nil && resp.Assignment != nil {
			roleAssignmentID = &resp.Assignment.ID
		}
		logger.Infof(ctx, "[ApprovalService] 分配角色给用户成功: user=%s, app=%s, username=%s, role_id=%d, role_code=%s, resource_type=%s",
			user, app, request.Subject, request.RoleID, role.Code, role.ResourceType)
	} else if request.SubjectType == "department" {
		// 分配角色给组织架构
		assignReq := &dto.AssignRoleToDepartmentReq{
			User:           user,
			App:            app,
			DepartmentPath: request.Subject,
			RoleCode:       role.Code,
			ResourceType:   role.ResourceType, // ⭐ 从角色中获取资源类型
			ResourcePath:   request.ResourcePath,
			StartTime:      &startTime,
			EndTime:        endTime,
		}
		resp, err := s.roleService.AssignRoleToDepartment(ctx, assignReq)
		if err != nil {
			logger.Errorf(ctx, "[ApprovalService] 分配角色给组织架构失败: user=%s, app=%s, dept=%s, role_id=%d, error=%v",
				user, app, request.Subject, request.RoleID, err)
			return fmt.Errorf("分配角色给组织架构失败: %w", err)
		}
		if resp != nil && resp.Assignment != nil {
			roleAssignmentID = &resp.Assignment.ID
		}
		logger.Infof(ctx, "[ApprovalService] 分配角色给组织架构成功: user=%s, app=%s, dept=%s, role_id=%d, role_code=%s",
			user, app, request.Subject, request.RoleID, role.Code)
	}
	
	// 6. 更新申请记录的 role_assignment_id
	if roleAssignmentID != nil {
		if err := s.permissionRequestRepo.UpdateRoleAssignmentID(ctx, requestID, *roleAssignmentID); err != nil {
			logger.Warnf(ctx, "[ApprovalService] 更新申请记录的 role_assignment_id 失败: request_id=%d, role_assignment_id=%d, error=%v",
				requestID, *roleAssignmentID, err)
			// 不影响主流程
		}
	}
	
	logger.Infof(ctx, "[ApprovalService] 审批通过（角色申请）: request_id=%d, approver=%s, role_id=%d",
		requestID, approverUsername, request.RoleID)

	return nil
}

// RejectRequest 审批拒绝
func (s *ApprovalService) RejectRequest(ctx context.Context, requestID int64, approverUsername string, reason string) error {
	// 1. 查询申请记录
	request, err := s.permissionRequestRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("查询申请记录失败: %w", err)
	}

	// 2. 检查状态
	if request.Status != model.PermissionRequestStatusPending {
		return fmt.Errorf("申请状态不是待审批，无法拒绝: status=%s", request.Status)
	}

	// 3. 检查审批人权限（是否是节点管理员）
	admins, err := s.serviceTreeRepo.GetNodeAdmins(ctx, request.ResourcePath)
	if err != nil {
		return fmt.Errorf("查询节点管理员失败: %w", err)
	}

	isAdmin := false
	for _, admin := range admins {
		if admin == approverUsername {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		return fmt.Errorf("用户 %s 不是节点 %s 的管理员，无权审批", approverUsername, request.ResourcePath)
	}

	// 4. 更新申请状态
	if err := s.permissionRequestRepo.UpdateRequestStatus(ctx, requestID, model.PermissionRequestStatusRejected, "", approverUsername, reason); err != nil {
		return fmt.Errorf("更新申请状态失败: %w", err)
	}

	logger.Infof(ctx, "[ApprovalService] 审批拒绝: request_id=%d, approver=%s, reason=%s",
		requestID, approverUsername, reason)

	return nil
}

// resolveApprovers 解析审批人（目前只支持节点管理员）
func (s *ApprovalService) resolveApprovers(ctx context.Context, resourcePath string) ([]string, error) {
	// 获取节点管理员
	admins, err := s.serviceTreeRepo.GetNodeAdmins(ctx, resourcePath)
	if err != nil {
		return nil, fmt.Errorf("查询节点管理员失败: %w", err)
	}

	return admins, nil
}

// getResourceTypeFromServiceTree 从 ServiceTree 查询资源类型
// ⭐ 不能从路径推断，因为层级可能很深，无法判断是目录还是函数
func (s *ApprovalService) getResourceTypeFromServiceTree(ctx context.Context, resourcePath string) string {
	// 尝试从 ServiceTree 查询
	node, err := s.serviceTreeRepo.GetNodeByPath(ctx, resourcePath)
	if err == nil && node != nil {
		// 根据 ServiceTree.Type 转换为资源类型
		if node.Type == model.ServiceTreeTypeFunction {
			return "function"
		} else if node.Type == model.ServiceTreeTypePackage {
			return "directory"
		}
	}

	// 如果查询不到，检查是否是应用级别路径（/user/app）
	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) == 2 {
		return "app"
	}

	// 默认返回 directory（因为大部分是目录）
	return "directory"
}
