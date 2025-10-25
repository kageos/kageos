package v1

import (
	"io"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

type App struct {
	appService *service.AppService
}

// NewApp 创建 App 处理器（依赖注入）
func NewApp(appService *service.AppService) *App {
	return &App{
		appService: appService,
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
// @Router /app/create [post]
func (a *App) CreateApp(c *gin.Context) {
	var req dto.CreateAppReq
	var resp dto.CreateAppResp
	var err error
	defer func() {
		logger.Infof(c, "CreateApp req:%+v resp:%+v err:%v", req, resp, err)
	}()
	err = c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	req.User = contextx.GetRequestUser(c)
	app, err := a.appService.CreateApp(c, &req)
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
// @Router /app/update/{app} [post]
func (a *App) UpdateApp(c *gin.Context) {
	var resp *dto.UpdateAppResp
	var err error

	// 从JWT Token获取用户信息
	user := contextx.GetUserInfo(c)
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
	req := &dto.UpdateAppReq{
		User: user,
		App:  app,
	}

	defer func() {
		logger.Infof(c, "UpdateApp req:%+v resp:%+v err:%v", req, resp, err)
	}()

	resp, err = a.appService.UpdateApp(c, req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// RequestApp 应用接口请求
// @Summary 应用接口请求 (支持所有 HTTP 方法)
// @Description
// @Description **支持的 HTTP 方法：**
// @Description - GET: 获取数据
// @Description - POST: 创建数据
// @Description - PUT: 更新数据
// @Description - DELETE: 删除数据
// @Description - PATCH: 部分更新数据
// @Description - HEAD: 获取响应头
// @Description - OPTIONS: 获取支持的方法
// @Description
// @Description **功能说明：**
// @Description 此接口通过 Gin 的 Any 方法注册，支持所有 HTTP 方法。会自动透传查询参数和请求体到目标应用。
// @Description
// @Description **使用示例：**
// @Description - GET /api/v1/app/request/myapp/users?page=1
// @Description - POST /api/v1/app/request/myapp/users (with JSON body)
// @Description - PUT /api/v1/app/request/myapp/users/123 (with JSON body)
// @Description - DELETE /api/v1/app/request/myapp/users/123
// @Tags 应用接口请求
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
// @Router /app/request/{app}/{router} [get]
func (a *App) RequestApp(c *gin.Context) {
	var req dto.RequestAppReq
	var resp *dto.RequestAppResp
	var err error
	defer func() {
		logger.Infof(c, "RequestApp req:%+v resp:%+v err:%v", req, resp, err)
	}()

	now := time.Now()
	// 从JWT Token获取用户信息
	user := contextx.GetUserInfo(c)
	if user == "" {
		response.FailWithMessage(c, "无法获取用户信息")
		return
	}

	// 从路径参数获取 app, router
	app := c.Param("app")
	router := c.Param("router")

	if app == "" || router == "" {
		response.FailWithMessage(c, "app and router parameters are required")
		return
	}

	// 构建请求对象
	req = dto.RequestAppReq{
		User:        user,
		App:         app,
		Router:      router,           // 路由路径
		Method:      c.Request.Method, // 应用内部方法名（可选）
		TraceId:     contextx.GetTraceId(c),
		RequestUser: user,
	}

	// 绑定请求体（POST、PUT、PATCH 等方法通常有请求体）
	if c.Request.ContentLength > 0 && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH") {
		all, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
		req.Body = all
	}

	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery

	// 调用服务层
	resp, err = a.appService.RequestApp(c, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 如果应用返回了错误，也通过错误返回
	if resp.Error != "" {
		response.FailWithMessage(c, resp.Error)
		return
	}
	mill := time.Now().Sub(now).Milliseconds()
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
// @Router /app/delete/{app} [delete]
func (a *App) DeleteApp(c *gin.Context) {
	var resp *dto.DeleteAppResp
	var err error

	// 从JWT Token获取用户信息
	user := contextx.GetUserInfo(c)
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

	defer func() {
		logger.Infof(c, "DeleteApp req:%+v resp:%+v err:%v", req, resp, err)
	}()

	resp, err = a.appService.DeleteApp(c, req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}
