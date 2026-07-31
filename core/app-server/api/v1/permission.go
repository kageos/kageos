package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type Permission struct {
	permissionService        *service.PermissionService
	permissionRequestService *service.PermissionRequestService
}

func NewPermission(
	permissionService *service.PermissionService,
	permissionRequestService *service.PermissionRequestService,
) *Permission {
	return &Permission{
		permissionService:        permissionService,
		permissionRequestService: permissionRequestService,
	}
}

func (a *Permission) ListAssignments(c *gin.Context) {
	resourcePath := access.NormalizeResourcePath(c.Query("resource_path"))
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := a.permissionService.RequirePermission(ctx, tenantUser, app, contextx.GetRequestUser(ctx), resourcePath, access.ActionRead); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	assignments, err := a.permissionService.ListAssignments(ctx, tenantUser, app, resourcePath)
	if err != nil {
		response.FailWithMessage(c, "获取权限分配失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.PermissionAssignmentsResp{Assignments: assignments})
}

func (a *Permission) GrantRole(c *gin.Context) {
	var req dto.GrantRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	resourcePath := access.NormalizeResourcePath(req.ResourcePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := a.permissionService.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   tenantUser,
		App:          app,
		Principal:    req.Principal,
		ResourcePath: resourcePath,
		RoleCode:     req.RoleCode,
		ExpiresAt:    req.ExpiresAt,
		CreatedBy:    contextx.GetRequestUser(ctx),
	}); err != nil {
		response.FailWithMessage(c, "授权失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (a *Permission) BatchGrantRoles(c *gin.Context) {
	var req dto.BatchGrantRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	if len(req.ResourcePaths) == 0 {
		response.FailWithMessage(c, "resource_paths 不能为空")
		return
	}
	firstPath := access.NormalizeResourcePath(req.ResourcePaths[0])
	tenantUser, app, err := access.ParseUserApp(firstPath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	for _, resourcePath := range req.ResourcePaths {
		pathTenant, pathApp, err := access.ParseUserApp(resourcePath)
		if err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
		if pathTenant != tenantUser || pathApp != app {
			response.FailWithMessage(c, "批量授权暂不支持跨 workspace")
			return
		}
	}
	ctx := contextx.ToContext(c)
	if err := a.permissionService.BatchGrantRoles(ctx, access.BatchGrantRoleRequest{
		TenantUser:    tenantUser,
		App:           app,
		Principals:    req.Principals,
		ResourcePaths: req.ResourcePaths,
		RoleCodes:     req.RoleCodes,
		ExpiresAt:     req.ExpiresAt,
		CreatedBy:     contextx.GetRequestUser(ctx),
	}); err != nil {
		response.FailWithMessage(c, "批量授权失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (a *Permission) RevokeRole(c *gin.Context) {
	var req dto.RevokeRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	resourcePath := access.NormalizeResourcePath(req.ResourcePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := a.permissionService.RevokeRole(ctx, access.RevokeRoleRequest{
		TenantUser:   tenantUser,
		App:          app,
		Principal:    req.Principal,
		ResourcePath: resourcePath,
		RoleCode:     req.RoleCode,
		Actor:        contextx.GetRequestUser(ctx),
	}); err != nil {
		response.FailWithMessage(c, "移除授权失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (a *Permission) GetMyPermissions(c *gin.Context) {
	resourcePath := access.NormalizeResourcePath(c.Query("resource_path"))
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	result, err := a.permissionService.ResolvePermissions(ctx, tenantUser, app, contextx.GetRequestUser(ctx), resourcePath)
	if err != nil {
		response.FailWithMessage(c, "获取当前用户权限失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.MyPermissionsResp{
		ResourcePath:  result.ResourcePath,
		RoleCodes:     result.RoleCodes,
		Permissions:   result.Permissions,
		InheritedFrom: result.InheritedFrom,
		ExpiresAt:     result.ExpiresAt,
	})
}

func (a *Permission) CreateRequest(c *gin.Context) {
	var req dto.CreatePermissionRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	resourcePath := access.NormalizeResourcePath(req.ResourcePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	request, err := a.permissionRequestService.CreateRequest(
		ctx,
		tenantUser,
		app,
		contextx.GetRequestUser(ctx),
		resourcePath,
		req.RoleCode,
		req.Reason,
		req.ExpiresAt,
	)
	if err != nil {
		response.FailWithMessage(c, "提交权限申请失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"request": request})
}

func (a *Permission) ListMyRequests(c *gin.Context) {
	tenantUser, app, ok := permissionRequestWorkspace(c)
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	requests, err := a.permissionRequestService.ListMine(
		ctx,
		tenantUser,
		app,
		contextx.GetRequestUser(ctx),
		c.Query("status"),
	)
	if err != nil {
		response.FailWithMessage(c, "获取我的权限申请失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"requests": requests})
}

func (a *Permission) ListPendingRequests(c *gin.Context) {
	tenantUser, app, ok := permissionRequestWorkspace(c)
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	requests, err := a.permissionRequestService.ListPendingForReviewer(
		ctx,
		tenantUser,
		app,
		contextx.GetRequestUser(ctx),
	)
	if err != nil {
		response.FailWithMessage(c, "获取待审批权限失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"requests": requests})
}

func (a *Permission) ListRequestHistory(c *gin.Context) {
	tenantUser, app, ok := permissionRequestWorkspace(c)
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	requests, err := a.permissionRequestService.ListHistoryForReviewer(
		ctx,
		tenantUser,
		app,
		contextx.GetRequestUser(ctx),
	)
	if err != nil {
		response.FailWithMessage(c, "获取权限审批记录失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"requests": requests})
}

func (a *Permission) PendingRequestCount(c *gin.Context) {
	tenantUser, app, ok := permissionRequestWorkspace(c)
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	count, err := a.permissionRequestService.CountPendingForReviewer(
		ctx,
		tenantUser,
		app,
		contextx.GetRequestUser(ctx),
	)
	if err != nil {
		response.FailWithMessage(c, "获取待审批权限数量失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"count": count})
}

func (a *Permission) ListApprovers(c *gin.Context) {
	resourcePath := access.NormalizeResourcePath(c.Query("resource_path"))
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	approvers, err := a.permissionRequestService.Approvers(ctx, tenantUser, app, resourcePath)
	if err != nil {
		response.FailWithMessage(c, "获取审批人失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"approvers": approvers})
}

func (a *Permission) ApproveRequest(c *gin.Context) {
	a.reviewRequest(c, true)
}

func (a *Permission) RejectRequest(c *gin.Context) {
	a.reviewRequest(c, false)
}

func (a *Permission) CancelRequest(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "权限申请 ID 无效")
		return
	}
	ctx := contextx.ToContext(c)
	request, err := a.permissionRequestService.Cancel(ctx, id, contextx.GetRequestUser(ctx))
	if err != nil {
		response.FailWithMessage(c, "撤销权限申请失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"request": request})
}

func (a *Permission) reviewRequest(c *gin.Context, approve bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "权限申请 ID 无效")
		return
	}
	var req dto.ReviewPermissionRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	var request *service.PermissionRequestView
	if approve {
		request, err = a.permissionRequestService.Approve(ctx, id, contextx.GetRequestUser(ctx), req.Comment)
	} else {
		request, err = a.permissionRequestService.Reject(ctx, id, contextx.GetRequestUser(ctx), req.Comment)
	}
	if err != nil {
		action := "批准"
		if !approve {
			action = "驳回"
		}
		response.FailWithMessage(c, action+"权限申请失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"request": request})
}

func permissionRequestWorkspace(c *gin.Context) (string, string, bool) {
	resourcePath := access.NormalizeResourcePath(c.Query("resource_path"))
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return "", "", false
	}
	return tenantUser, app, true
}
