package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/access"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

type TeamAccess struct {
	teamAccessService *service.TeamAccessService
}

func NewTeamAccess(teamAccessService *service.TeamAccessService) *TeamAccess {
	return &TeamAccess{teamAccessService: teamAccessService}
}

func (a *TeamAccess) ListMembers(c *gin.Context) {
	resourcePath := access.NormalizeResourcePath(c.Query("resource_path"))
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := a.teamAccessService.Check(ctx, tenantUser, app, contextx.GetRequestUser(ctx), resourcePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	members, err := a.teamAccessService.ListMembers(ctx, tenantUser, app, resourcePath)
	if err != nil {
		response.FailWithMessage(c, "获取成员授权失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.TeamMemberAccessResp{Members: members})
}

func (a *TeamAccess) Assign(c *gin.Context) {
	var req dto.AssignTeamRoleReq
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
	if err := a.teamAccessService.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   tenantUser,
		App:          app,
		Username:     req.Username,
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

func (a *TeamAccess) BatchAssign(c *gin.Context) {
	var req dto.BatchAssignTeamRoleReq
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
	if err := a.teamAccessService.BatchAssign(ctx, access.BatchAssignRoleRequest{
		TenantUser:    tenantUser,
		App:           app,
		Usernames:     req.Usernames,
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

func (a *TeamAccess) Remove(c *gin.Context) {
	var req dto.RemoveTeamRoleReq
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
	if err := a.teamAccessService.Remove(ctx, access.RemoveRoleRequest{
		TenantUser:   tenantUser,
		App:          app,
		Username:     req.Username,
		ResourcePath: resourcePath,
		RoleCode:     req.RoleCode,
		Actor:        contextx.GetRequestUser(ctx),
	}); err != nil {
		response.FailWithMessage(c, "移除授权失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (a *TeamAccess) MyPermissions(c *gin.Context) {
	resourcePath := access.NormalizeResourcePath(c.Query("resource_path"))
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	result, err := a.teamAccessService.Resolve(ctx, tenantUser, app, contextx.GetRequestUser(ctx), resourcePath)
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
