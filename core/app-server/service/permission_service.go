package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// PermissionService 权限管理服务
// ⭐ 完全移除 Casbin，使用新的权限系统
type PermissionService struct {
	permissionService enterprise.PermissionService
}

// NewPermissionService 创建权限管理服务
func NewPermissionService(permissionService enterprise.PermissionService) *PermissionService {
	return &PermissionService{
		permissionService: permissionService,
	}
}

// AddPermission 添加权限
// ⭐ 使用新的权限系统，完全移除 Casbin
// 支持两种类型的权限：
//  1. 用户权限：subject = username
//  2. 组织架构权限：subject = department_path（以 /org 开头）
func (s *PermissionService) AddPermission(ctx context.Context, req *dto.AddPermissionReq) error {
	// 确定权限主体类型
	subjectType := "user"
	if strings.HasPrefix(req.Subject, "/org") {
		subjectType = "department"
	}

	// 构建授权请求（AppID 已废弃，从 resourcePath 解析 user 和 app）
	grantReq := &enterprise.GrantPermissionReq{
		AppID:           0,                            // ⭐ 已废弃，不再使用（企业版实现会从 resourcePath 解析 user 和 app）
		GrantorUsername: contextx.GetRequestUser(ctx), // 授权人（当前用户）
		GranteeType:     subjectType,
		Grantee:         req.Subject,
		ResourcePath:    req.ResourcePath,
		Action:          req.Action,
		StartTime:       time.Now().Format(time.RFC3339),
		EndTime:         "", // 永久权限
	}

	// 调用企业版接口授权权限
	if err := s.permissionService.GrantPermission(ctx, grantReq); err != nil {
		logger.Errorf(ctx, "[PermissionService] 授权权限失败: subject=%s, resource=%s, action=%s, error=%v",
			req.Subject, req.ResourcePath, req.Action, err)
		return fmt.Errorf("授权权限失败: %w", err)
	}

	logger.Infof(ctx, "[PermissionService] 添加权限成功: subject=%s, resource=%s, action=%s",
		req.Subject, req.ResourcePath, req.Action)
	return nil
}

// GetWorkspacePermissions 获取工作空间的所有权限
// ⭐ 优化：支持查询用户权限和组织架构权限（v0 可以是用户名或组织架构路径）
// ⭐ 一次性查询用户及其组织架构的所有权限，性能更好
// ⭐ 支持传递用户和组织架构参数，使方法可复用（既可以获取当前用户权限，也可以获取其他用户权限）
func (s *PermissionService) GetWorkspacePermissions(ctx context.Context, req *dto.GetWorkspacePermissionsReq) (*dto.GetWorkspacePermissionsResp, error) {
	// ⭐ 参数验证：必须提供 user 和 app
	if req.User == "" || req.App == "" {
		return nil, fmt.Errorf("必须提供 user 和 app 参数")
	}

	// ⭐ 获取用户名：优先使用请求参数，否则从 context 获取（向后兼容）
	username := req.Username
	if username == "" {
		username = contextx.GetRequestUser(ctx)
		if username == "" {
			return nil, fmt.Errorf("无法获取用户信息（请提供 username 参数或确保 context 中包含用户信息）")
		}
	}

	// ⭐ 构建 v0 列表：用户名 + 组织架构路径 + 父级组织架构路径
	v0List := []string{username}

	// ⭐ 获取组织架构路径：优先使用请求参数，否则从 context 获取（向后兼容）
	deptPath := req.DepartmentFullPath
	if deptPath == "" {
		deptPath = contextx.GetRequestDepartmentFullPath(ctx)
	}

	if deptPath != "" {
		// 添加用户所属组织架构路径及其所有父级路径
		deptPaths := s.getAllParentDeptPaths(deptPath)
		v0List = append(v0List, deptPaths...)

		logger.Debugf(ctx, "[PermissionService] 查询权限: user=%s, deptPath=%s, parentPaths=%v, v0List=%v",
			username, deptPath, deptPaths, v0List)
	} else {
		logger.Debugf(ctx, "[PermissionService] 用户无组织架构信息: user=%s，仅查询用户直接权限", username)
	}

	// ⭐ 直接使用 user 和 app，无需查询 app 表（性能优化）
	enterpriseReq := &enterprise.GetUserWorkspacePermissionsReq{
		User:           req.User,
		App:            req.App,
		Username:       username,
		DepartmentPath: deptPath,
	}

	enterpriseResp, err := s.permissionService.GetUserWorkspacePermissions(ctx, enterpriseReq)
	if err != nil {
		logger.Errorf(ctx, "[PermissionService] 查询权限记录失败: user=%s, app=%s, username=%s, error=%v", req.User, req.App, username, err)
		return nil, fmt.Errorf("查询权限记录失败: %w", err)
	}

	// ⭐ 转换为 DTO 格式
	records := make([]dto.PermissionRecord, 0, len(enterpriseResp.Records))
	for _, record := range enterpriseResp.Records {
		records = append(records, dto.PermissionRecord{
			ID:       0,  // 新系统不需要 ID
			User:     "", // 从 record.Resource 和 record.Action 中提取
			Resource: record.Resource,
			Action:   record.Action,
			AppID:    0, // 不再使用 AppID
		})
	}

	logger.Debugf(ctx, "[PermissionService] 查询权限成功: user=%s, app=%s, username=%s, total_records=%d", req.User, req.App, username, len(records))

	// ⭐ 返回所有权限记录（包括用户权限和组织架构权限）
	return &dto.GetWorkspacePermissionsResp{
		Records: records,
	}, nil
}

// getAllParentDeptPaths 获取组织架构路径及其所有父级路径
// 例如：/org/master/bizit → [/org/master/bizit, /org/master, /org]
func (s *PermissionService) getAllParentDeptPaths(deptPath string) []string {
	if deptPath == "" {
		return []string{}
	}

	// 移除开头的斜杠
	path := strings.TrimPrefix(deptPath, "/")
	if path == "" {
		return []string{}
	}

	// 分割路径
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return []string{}
	}

	// 构建所有父级路径（包括自身）
	parentPaths := make([]string, 0, len(parts))
	for i := 1; i <= len(parts); i++ {
		parentPath := "/" + strings.Join(parts[:i], "/")
		parentPaths = append(parentPaths, parentPath)
	}

	return parentPaths
}

// CreatePermissionRequest 创建权限申请
func (s *PermissionService) CreatePermissionRequest(ctx context.Context, req *dto.CreatePermissionRequestReq) (*dto.CreatePermissionRequestResp, error) {
	// 获取当前用户名
	username := contextx.GetRequestUser(ctx)
	if username == "" {
		return nil, fmt.Errorf("无法获取当前用户信息")
	}

	// 构建企业版请求（AppID 已废弃，企业版实现会从 resourcePath 解析 user 和 app）
	enterpriseReq := &enterprise.PermissionRequestReq{
		AppID:             0, // ⭐ 已废弃，不再使用（企业版实现会从 resourcePath 解析 user 和 app）
		ApplicantUsername: username,
		SubjectType:       req.SubjectType,
		Subject:           req.Subject,
		ResourcePath:      req.ResourcePath,
		Action:            req.Action,
		StartTime:         req.StartTime,
		EndTime:           req.EndTime,
		Reason:            req.Reason,
	}

	// 调用企业版接口
	requestID, err := s.permissionService.CreatePermissionRequest(ctx, enterpriseReq)
	if err != nil {
		return nil, fmt.Errorf("创建权限申请失败: %w", err)
	}

	return &dto.CreatePermissionRequestResp{
		RequestID: requestID,
		Status:    "pending",
		Message:   "权限申请已提交，等待审批",
	}, nil
}

// ApprovePermissionRequest 审批通过权限申请
func (s *PermissionService) ApprovePermissionRequest(ctx context.Context, req *dto.ApprovePermissionRequestReq) error {
	// 获取当前用户名（审批人）
	approverUsername := contextx.GetRequestUser(ctx)
	if approverUsername == "" {
		return fmt.Errorf("无法获取当前用户信息")
	}

	// 调用企业版接口
	return s.permissionService.ApprovePermissionRequest(ctx, req.RequestID, approverUsername)
}

// RejectPermissionRequest 审批拒绝权限申请
func (s *PermissionService) RejectPermissionRequest(ctx context.Context, req *dto.RejectPermissionRequestReq) error {
	// 获取当前用户名（审批人）
	approverUsername := contextx.GetRequestUser(ctx)
	if approverUsername == "" {
		return fmt.Errorf("无法获取当前用户信息")
	}

	// 调用企业版接口
	return s.permissionService.RejectPermissionRequest(ctx, req.RequestID, approverUsername, req.Reason)
}

// GrantPermission 授权权限（管理员主动授权）
func (s *PermissionService) GrantPermission(ctx context.Context, req *dto.GrantPermissionReq) error {
	// 获取当前用户名（授权人）
	grantorUsername := contextx.GetRequestUser(ctx)
	if grantorUsername == "" {
		return fmt.Errorf("无法获取当前用户信息")
	}

	// 构建企业版请求（AppID 已废弃，企业版实现会从 resourcePath 解析 user 和 app）
	enterpriseReq := &enterprise.GrantPermissionReq{
		AppID:           0, // ⭐ 已废弃，不再使用（企业版实现会从 resourcePath 解析 user 和 app）
		GrantorUsername: grantorUsername,
		GranteeType:     req.GranteeType,
		Grantee:         req.Grantee,
		ResourcePath:    req.ResourcePath,
		Action:          req.Action,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
	}

	// 调用企业版接口
	return s.permissionService.GrantPermission(ctx, enterpriseReq)
}

// GetPermissionRequests 获取权限申请列表
// TODO: 实现此方法，需要查询 permission_request 表
func (s *PermissionService) GetPermissionRequests(ctx context.Context, req *dto.GetPermissionRequestsReq) (*dto.GetPermissionRequestsResp, error) {
	// 暂时返回空列表，后续实现
	return &dto.GetPermissionRequestsResp{
		Total:    0,
		Page:     req.Page,
		PageSize: req.PageSize,
		Records:  []dto.PermissionRequestInfo{},
	}, nil
}
