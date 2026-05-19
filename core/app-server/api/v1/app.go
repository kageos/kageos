package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/pkg/access"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

type App struct {
	appService         *service.AppService
	serviceTreeService *service.ServiceTreeService
	teamAccessService  *service.TeamAccessService
}

// NewApp 创建 App 处理器（依赖注入）
func NewApp(appService *service.AppService, serviceTreeService *service.ServiceTreeService, teamAccessService *service.TeamAccessService) *App {
	return &App{
		appService:         appService,
		serviceTreeService: serviceTreeService,
		teamAccessService:  teamAccessService,
	}
}

// CreateApp 创建应用
// @Summary 创建应用
// @Description 创建一个新的应用实例。租户用户（应用所有者）从请求体获取，请求用户（实际发起请求的用户）从请求头获取。租户用户决定应用的所有权，请求用户用于审计追踪。
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CreateAppReq true "创建应用请求，包含应用名（租户用户通过 header 传递）"
// @Success 200 {object} dto.CreateAppResp "创建成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/app/create [post]
func (a *App) CreateApp(c *gin.Context) {
	var req dto.CreateAppReq
	var err error
	err = c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	req.User = contextx.GetRequestUser(c)
	// 将 gin.Context 转换为标准 context.Context，解析 header 并放入 context.Value
	// 这样即使内部使用 context.WithValue，也能通过 context.Value 获取到值
	ctx := contextx.ToContext(c)
	app, err := a.appService.CreateApp(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, app)
}

// UpdateApp 更新应用
// @Summary 更新应用
// @Description 更新应用代码并重新编译部署，使用 resource_path=/user/app 标识工作空间。
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param body body dto.UpdateAppReq false "SourceFiles、WriteOnly 等"
// @Success 200 {object} dto.UpdateAppResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/app/update [post]
func (a *App) UpdateApp(c *gin.Context) {
	var resp *dto.UpdateAppResp
	var err error

	if contextx.GetRequestUser(c) == "" {
		response.FailWithMessage(c, "无法获取用户信息")
		return
	}

	req := &dto.UpdateAppReq{}
	if err := c.ShouldBindJSON(req); err != nil && err.Error() != "EOF" {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	if err := requireAccess(c, a.teamAccessService, req.ResourcePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err = a.appService.UpdateApp(ctx, req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// UpdateWorkspace 更新工作空间
// @Summary 更新工作空间
// @Description 更新工作空间配置（只更新 MySQL 记录，不涉及容器更新），使用 resource_path=/user/app 标识工作空间。
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.UpdateWorkspaceReq true "更新工作空间请求"
// @Success 200 {object} dto.UpdateWorkspaceResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "无权限"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/app/workspace [put]
func (a *App) UpdateWorkspace(c *gin.Context) {
	var req dto.UpdateWorkspaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	if err := requireAccess(c, a.teamAccessService, req.ResourcePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := a.appService.UpdateWorkspace(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// DeleteApp 删除应用
// @Summary 删除应用
// @Description 删除应用及其所有相关资源，使用 query resource_path=/user/app 标识工作空间。
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param resource_path query string true "工作空间资源路径，格式 /user/app"
// @Success 200 {object} dto.DeleteAppResp "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/app/delete [delete]
func (a *App) DeleteApp(c *gin.Context) {
	var resp *dto.DeleteAppResp
	var err error

	resourcePath := c.Query("resource_path")
	req := &dto.DeleteAppReq{
		ResourcePath: resourcePath,
	}
	if err := requireAccess(c, a.teamAccessService, req.ResourcePath, access.ActionDelete); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err = a.appService.DeleteApp(ctx, req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// GetApps 获取应用列表
// @Summary 获取应用列表
// @Description 获取当前用户的所有应用列表（支持分页和搜索）
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param page query int false "页码，默认为1" default(1)
// @Param page_size query int false "每页数量，默认为10" default(10)
// @Param search query string false "搜索关键词（支持按应用名称或代码搜索）"
// @Success 200 {object} dto.GetAppsResp "获取成功"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/app/list [get]
func (a *App) GetApps(c *gin.Context) {
	var req dto.GetAppsReq
	var resp *dto.GetAppsResp
	var err error

	// 从JWT Token获取用户信息
	user := contextx.GetRequestUser(c)
	if user == "" {
		response.FailWithMessage(c, "无法获取用户信息")
		return
	}

	// 从查询参数获取分页信息和搜索关键词
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "10")
	search := c.Query("search")
	includeAll := c.DefaultQuery("include_all", "false") == "true"
	typeStr := c.Query("type")

	// 解析应用类型（可选）
	var appType *int
	if typeStr != "" {
		if typeVal := parseIntWithDefault(typeStr, -1); typeVal >= 0 {
			appType = &typeVal
		}
	}

	// 构建请求对象
	req = dto.GetAppsReq{
		PageInfoReq: dto.PageInfoReq{
			Page:     parseIntWithDefault(page, 1),
			PageSize: parseIntWithDefault(pageSize, 10),
		},
		User:       user,
		Search:     search,
		IncludeAll: includeAll,
		Type:       appType,
	}

	ctx := contextx.ToContext(c)
	resp, err = a.appService.GetApps(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// GetAppDetail 获取应用详情
// @Summary 获取应用详情
// @Description 根据 resource_path=/user/app 获取应用详情信息。
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param resource_path query string true "工作空间资源路径，格式 /user/app"
// @Success 200 {object} dto.GetAppDetailResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "应用不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/app/detail [get]
func (a *App) GetAppDetail(c *gin.Context) {
	var req dto.GetAppDetailReq
	var resp *dto.GetAppDetailResp
	var err error

	resourcePath := c.Query("resource_path")
	req = dto.GetAppDetailReq{
		ResourcePath: resourcePath,
	}
	if err := requireAccess(c, a.teamAccessService, req.ResourcePath, access.ActionRead); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err = a.appService.GetAppDetail(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// GetAppWithServiceTree 获取应用详情和服务目录树
// @Summary 获取应用详情和服务目录树
// @Description 根据 resource_path=/user/app 获取应用详情和服务目录树（合并接口，减少请求次数）。
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param resource_path query string true "工作空间资源路径，格式 /user/app"
// @Param type query string false "节点类型过滤（可选），如：package（只显示服务目录/包）、function（只显示函数/文件）"
// @Success 200 {object} dto.GetAppWithServiceTreeResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "应用不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/app/tree [get]
func (a *App) GetAppWithServiceTree(c *gin.Context) {
	var req dto.GetAppWithServiceTreeReq
	var resp *dto.GetAppWithServiceTreeResp
	var err error

	req = dto.GetAppWithServiceTreeReq{
		ResourcePath: c.Query("resource_path"),
		Type:         c.Query("type"),
	}
	tenantUser, appCode, err := access.ParseUserApp(req.ResourcePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	hasAccess, err := a.teamAccessService.HasAnyWorkspaceAccess(contextx.ToContext(c), tenantUser, appCode, contextx.GetRequestUser(c))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if !hasAccess {
		response.FailWithMessage(c, "无权限查看该 workspace")
		return
	}

	// 调用 ServiceTreeService 的方法（避免循环依赖）
	ctx := contextx.ToContext(c)
	resp, err = a.serviceTreeService.GetAppWithServiceTree(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// parseIntWithDefault 解析字符串为整数，如果解析失败则返回默认值
func parseIntWithDefault(s string, defaultValue int) int {
	result, err := strconv.Atoi(s)
	if err != nil || result <= 0 {
		return defaultValue
	}
	return result
}
