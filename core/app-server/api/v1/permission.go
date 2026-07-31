package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type Permission struct {
	permissionService *service.PermissionService
}

func NewPermission(permissionService *service.PermissionService) *Permission {
	return &Permission{permissionService: permissionService}
}

func (a *Permission) ListAssignments(c *gin.Context) {
	resourcePath := access.NormalizeResourcePath(c.Query("resource_path"))
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := a.permissionService.RequirePermission(ctx, tenantUser, app, contextx.GetRequestUser(ctx), resourcePath, access.ActionAdmin); err != nil {
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
