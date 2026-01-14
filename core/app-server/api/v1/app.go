package v1

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

type App struct {
	appService         *service.AppService
	serviceTreeService *service.ServiceTreeService
}

// NewApp 创建 App 处理器（依赖注入）
func NewApp(appService *service.AppService, serviceTreeService *service.ServiceTreeService) *App {
	return &App{
		appService:         appService,
		serviceTreeService: serviceTreeService,
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
// @Router /api/v1/app/create [post]
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
// @Description 更新应用代码并重新编译部署
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param app path string true "应用名"
// @Success 200 {object} dto.UpdateAppResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/app/update/{app} [post]
func (a *App) UpdateApp(c *gin.Context) {
	var resp *dto.UpdateAppResp
	var err error

	// 从JWT Token获取用户信息
	user := contextx.GetRequestUser(c)
	if user == "" {
		response.FailWithMessage(c, "无法获取用户信息")
		return
	}

	// 从路径参数获取应用信息
	app := c.Param("app")

	if app == "" {
		response.FailWithMessage(c, "app parameter is required")
		return
	}

	// 绑定请求体（包含 Requirement、ChangeDescription 等字段）
	req := &dto.UpdateAppReq{
		User: user,
		App:  app,
	}
	if err := c.ShouldBindJSON(req); err != nil {
		// 如果绑定失败，使用默认值（兼容旧版本，只设置 User 和 App）
		// req 已经初始化了 User 和 App，无需额外处理
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
// @Description 更新工作空间配置（只更新 MySQL 记录，不涉及容器更新）。A 用户可以更新 B 租户的工作空间（如果有 app:admin 权限）
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user path string true "租户用户名"
// @Param app path string true "应用名"
// @Param request body dto.UpdateWorkspaceReq true "更新工作空间请求"
// @Success 200 {object} dto.UpdateWorkspaceResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "无权限"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/app/workspace/:user/:app [put]
func (a *App) UpdateWorkspace(c *gin.Context) {
	// 从路径参数获取租户和应用信息
	user := c.Param("user")
	app := c.Param("app")
	if user == "" {
		response.FailWithMessage(c, "user parameter is required")
		return
	}
	if app == "" {
		response.FailWithMessage(c, "app parameter is required")
		return
	}

	// 绑定请求体
	var req dto.UpdateWorkspaceReq
	// ⭐ user 和 app 从路径参数获取，不在请求体中
	req.User = user
	req.App = app
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	// ⭐ 确保 User 和 App 始终来自路径参数（防止请求体中覆盖，虽然 JSON tag 是 "-" 不会绑定）
	req.User = user
	req.App = app

	ctx := contextx.ToContext(c)
	resp, err := a.appService.UpdateWorkspace(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// RequestApp
// @Tags 应用接口请求(运行函数)
// @Accept json
// @Accept application/x-www-form-urlencoded
// @Accept multipart/form-data
// @Accept text/plain
// @Produce json
// @Param app path string true "应用名"
// @Param router path string true "接口路由路径"
// @Param method query string false "应用内部方法名（可选）"
// @Success 200 {object} dto.RequestAppResp "请求成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/run/{full_code_path} [get]
func (a *App) RequestApp(c *gin.Context) {
	var req dto.RequestAppReq
	var resp *dto.RequestAppResp
	var err error

	now := time.Now()

	// 从路径参数获取 app, router
	router := c.Param("router")
	split := strings.Split(strings.Trim(router, "/"), "/")
	user := split[0]
	app := split[1]

	r := split[2:]

	if app == "" || router == "" {
		response.FailWithMessage(c, "app and router parameters are required")
		return
	}

	// 构建请求对象
	req = dto.RequestAppReq{
		User:        user,
		App:         app,
		Router:      strings.Join(r, "/"), // 路由路径
		Method:      c.Request.Method,     // 应用内部方法名（可选）
		TraceId:     contextx.GetTraceId(c),
		RequestUser: contextx.GetRequestUser(c),
		Token:       contextx.GetToken(c), // ✅ 透传 token 到 SDK
	}

	// 绑定请求体（POST、PUT、PATCH 等方法通常有请求体）
	if c.Request.ContentLength > 0 && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH") {
		all, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
		defer c.Request.Body.Close()
		req.Body = all
	}

	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery

	// 调用服务层
	ctx := contextx.ToContext(c)
	resp, err = a.appService.RequestApp(ctx, &req)
	mill := time.Since(now).Milliseconds()
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill
	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	// 如果应用返回了错误，也通过错误返回
	if resp.Error != "" { //无论业务错误还是系统错误都返回，通过返回的code区分，>0系统错误，<0业务错误
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// CallbackApp 回调应用接口
// @Summary 回调应用
// @Description 接收外部系统的回调请求并转发到应用内部的 callback 路由。路径格式：/api/v1/callback/*router?_type=回调类型
// @Tags 应用接口请求(运行函数)
// @Accept json
// @Accept application/x-www-form-urlencoded
// @Accept multipart/form-data
// @Accept text/plain
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param router path string true "回调路由路径，格式：/{router}，例如：/api/v1/callback/*router?_type=回调类型"
// @Param _type query string false "回调类型（可选），用于标识回调类型"
// @Success 200 {object} dto.RequestAppResp "回调处理成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/callback/{router} [post]
func (a *App) CallbackApp(c *gin.Context) {
	var req dto.RequestAppReq
	var resp *dto.RequestAppResp
	var err error

	now := time.Now()

	// 从路径参数获取 app, router
	router := c.Param("router")
	// 获取原函数的 HTTP 方法（用于标识回调所属的函数）
	// 优先使用 _function_method（更清晰的参数名），兼容旧的 _method 参数
	method := c.Query("_function_method")
	if method == "" {
		method = c.Query("_method") // 向后兼容旧版本
	}
	callbackType := c.Query("_type")
	split := strings.Split(strings.Trim(router, "/"), "/")
	user := split[0]
	app := split[1]

	realRouter := split[2:]

	if app == "" || router == "" {
		response.FailWithMessage(c, "app and router parameters are required")
		return
	}

	// 构建请求对象
	req = dto.RequestAppReq{
		User:        user,
		App:         app,
		Router:      "/_callback", // 路由路径
		Method:      method,       // 应用内部方法名（可选）
		TraceId:     contextx.GetTraceId(c),
		RequestUser: contextx.GetRequestUser(c),
		Token:       contextx.GetToken(c), // ✅ 透传 token 到 SDK
	}
	// 绑定请求体（POST、PUT、PATCH 等方法通常有请求体）

	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	defer c.Request.Body.Close()

	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery

	mp := make(map[string]interface{})
	mp["method"] = method
	mp["router"] = strings.Join(realRouter, "/")
	mp["body"] = all
	mp["type"] = callbackType
	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery
	marshal, err := json.Marshal(mp)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	req.Body = marshal

	// 调用服务层
	ctx := contextx.ToContext(c)

	// 如果是 Table 回调，记录 Table 操作日志
	// ⚠️ 注意：OnTableAddRow 不记录操作日志（主要是新增记录，不需要记录）
	if callbackType == "OnTableUpdateRow" || callbackType == "OnTableDeleteRows" {
		var bodyData map[string]interface{}
		if err := json.Unmarshal(all, &bodyData); err == nil {
			logReq := &dto.RecordTableOperateLogReq{
				TenantUser:  user,            // 租户用户（app 的所有者，从路径解析）
				RequestUser: req.RequestUser, // 请求用户（实际执行操作的用户）
				App:         app,
				Router:      strings.Join(realRouter, "/"),
				Action:      callbackType,
				IPAddress:   c.ClientIP(),
				UserAgent:   c.GetHeader("User-Agent"),
				TraceID:     req.TraceId,
			}

			switch callbackType {
			// case "OnTableAddRow":
			// 	// 新增操作：记录 body
			// 	// ⚠️ 已注释：OnTableAddRow 不记录操作日志（主要是新增记录，不需要记录）
			// 	logReq.Body = all

			case "OnTableUpdateRow":
				// 更新操作：优先从查询参数获取 _row_id，如果没有则从 body 中获取 id
				if rowIDStr := c.Query("_row_id"); rowIDStr != "" {
					if id, err := strconv.ParseInt(rowIDStr, 10, 64); err == nil {
						logReq.RowID = id
					}
				} else if id, ok := bodyData["id"].(float64); ok {
					logReq.RowID = int64(id)
				}

				// 获取 updates 和 old_values
				if updatesData, ok := bodyData["updates"].(map[string]interface{}); ok {
					logReq.Updates, _ = json.Marshal(updatesData)
				}
				if oldValuesData, ok := bodyData["old_values"].(map[string]interface{}); ok {
					logReq.OldValues, _ = json.Marshal(oldValuesData)
				}

			case "OnTableDeleteRows":
				// 删除操作：获取 ids 列表
				if ids, ok := bodyData["ids"].([]interface{}); ok {
					rowIDs := make([]int64, 0, len(ids))
					for _, id := range ids {
						if idFloat, ok := id.(float64); ok {
							rowIDs = append(rowIDs, int64(idFloat))
						}
					}
					logReq.RowIDs = rowIDs
				}
			}

			// 异步记录，不阻塞主流程
			go func() {
				if err := a.appService.RecordTableOperateLog(ctx, logReq); err != nil {
					logger.Warnf(ctx, "[CallbackApp] 记录 Table 操作日志失败: %v", err)
				}
			}()
		}
	}

	resp, err = a.appService.RequestApp(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 如果应用返回了错误，也通过错误返回
	if resp.Error != "" {
		response.FailWithMessage(c, resp.Error)
		return
	}
	mill := time.Since(now).Milliseconds()
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	metadata["version"] = resp.Version
	metadata["total_cost_mill"] = mill

	response.OkWithData(c, resp.Result, metadata)
}

// DeleteApp 删除应用
// @Summary 删除应用
// @Description 删除应用及其所有相关资源
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param app path string true "应用名"
// @Success 200 {object} dto.DeleteAppResp "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/app/delete/{app} [delete]
func (a *App) DeleteApp(c *gin.Context) {
	var resp *dto.DeleteAppResp
	var err error

	// 从JWT Token获取用户信息
	user := contextx.GetRequestUser(c)
	if user == "" {
		response.FailWithMessage(c, "无法获取用户信息")
		return
	}

	// 从路径参数获取应用信息
	app := c.Param("app")

	if app == "" {
		response.FailWithMessage(c, "app parameter is required")
		return
	}

	// 构建请求对象
	req := &dto.DeleteAppReq{
		User: user,
		App:  app,
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
// @Router /api/v1/app/list [get]
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

	// 构建请求对象
	req = dto.GetAppsReq{
		PageInfoReq: dto.PageInfoReq{
			Page:     parseIntWithDefault(page, 1),
			PageSize: parseIntWithDefault(pageSize, 10),
		},
		User:       user,
		Search:     search,
		IncludeAll: includeAll,
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
// @Description 根据应用代码获取应用详情信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param app path string true "应用代码"
// @Success 200 {object} dto.GetAppDetailResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "应用不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/app/detail/{app} [get]
func (a *App) GetAppDetail(c *gin.Context) {
	var req dto.GetAppDetailReq
	var resp *dto.GetAppDetailResp
	var err error

	// 从JWT Token获取用户信息
	user := contextx.GetRequestUser(c)
	if user == "" {
		response.FailWithMessage(c, "无法获取用户信息")
		return
	}

	// 从路径参数获取应用代码
	app := c.Param("app")
	if app == "" {
		response.FailWithMessage(c, "app parameter is required")
		return
	}

	// 构建请求对象
	req = dto.GetAppDetailReq{
		User: user,
		App:  app,
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
// @Description 根据应用代码获取应用详情和服务目录树（合并接口，减少请求次数）
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param app path string true "应用代码"
// @Param type query string false "节点类型过滤（可选），如：package（只显示服务目录/包）、function（只显示函数/文件）"
// @Success 200 {object} dto.GetAppWithServiceTreeResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "应用不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/app/{app}/tree [get]
func (a *App) GetAppWithServiceTree(c *gin.Context) {
	var req dto.GetAppWithServiceTreeReq
	var resp *dto.GetAppWithServiceTreeResp
	var err error

	// ⭐ 从路径参数获取 user 和 app（不再从 JWT Token 获取）
	user := c.Param("user")
	app := c.Param("app")

	if user == "" || app == "" {
		response.FailWithMessage(c, "user 和 app 参数不能为空")
		return
	}

	// 从查询参数获取节点类型过滤
	nodeType := c.Query("type")

	// 构建请求对象
	req = dto.GetAppWithServiceTreeReq{
		User: user,
		App:  app,
		Type: nodeType,
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
