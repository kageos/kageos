package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

type ServiceTree struct {
	serviceTreeService *service.ServiceTreeService
}

// NewServiceTree 创建 ServiceTree 处理器（依赖注入）
func NewServiceTree(serviceTreeService *service.ServiceTreeService) *ServiceTree {
	return &ServiceTree{
		serviceTreeService: serviceTreeService,
	}
}

// CreateServiceTree 创建服务目录
// @Summary 创建服务目录
// @Description 为指定应用创建服务目录
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CreateServiceTreeReq true "创建服务目录请求"
// @Success 200 {object} dto.CreateServiceTreeResp
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree [post]
func (s *ServiceTree) CreateServiceTree(c *gin.Context) {
	var req dto.CreateServiceTreeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 创建服务目录
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.CreateServiceTree(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "创建服务目录失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetServiceTree 获取服务目录树
// @Summary 获取服务目录树
// @Description 获取指定应用的服务目录树形结构，支持按类型过滤（如只显示 package 类型的节点）
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user query string true "用户名"
// @Param app query string true "应用名"
// @Param type query string false "节点类型过滤（可选），如：package（只显示服务目录/包）、function（只显示函数/文件）"
// @Success 200 {object} []dto.GetServiceTreeResp
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree [get]
func (s *ServiceTree) GetServiceTree(c *gin.Context) {
	user := c.Query("user")
	app := c.Query("app")
	nodeType := c.Query("type")

	if user == "" || app == "" {
		response.FailWithMessage(c, "用户和应用名不能为空")
		return
	}

	// 获取服务目录树（支持类型过滤）
	ctx := contextx.ToContext(c)
	trees, err := s.serviceTreeService.GetServiceTree(ctx, user, app, nodeType)
	if err != nil {
		response.FailWithMessage(c, "获取服务目录失败: "+err.Error())
		return
	}

	response.OkWithData(c, trees)
}

// GetServiceTreeDetail 获取服务目录详情（包含权限信息）
// @Summary 获取服务目录详情
// @Description 根据ID或full-code-path获取服务目录详情，包含权限信息
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id query int false "服务目录ID（优先使用）"
// @Param full_code_path query string false "完整代码路径（如果未提供ID则使用）"
// @Success 200 {object} dto.GetServiceTreeDetailResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "服务目录不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/detail [get]
func (s *ServiceTree) GetServiceTreeDetail(c *gin.Context) {
	var req dto.GetServiceTreeDetailReq

	// 从 query 参数获取 ID
	idStr := c.Query("id")
	if idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.FailWithMessage(c, "无效的ID参数")
			return
		}
		req.ID = id
	}

	// 从 query 参数获取 full_code_path
	req.FullCodePath = c.Query("full_code_path")

	if req.ID == 0 && req.FullCodePath == "" {
		response.FailWithMessage(c, "必须提供 ID 或 full_code_path 参数")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "获取服务目录详情失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetPackageInfo 获取目录信息（仅用于获取目录权限）
// @Summary 获取目录信息
// @Description 根据ID或full-code-path获取目录信息，包含权限信息（仅用于目录，函数请使用函数详情接口）
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id query int false "目录ID（优先使用）"
// @Param full_code_path query string false "完整代码路径（如果未提供ID则使用）"
// @Success 200 {object} dto.GetPackageInfoResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "目录不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/package_info [get]
func (s *ServiceTree) GetPackageInfo(c *gin.Context) {
	var req dto.GetPackageInfoReq

	// 从 query 参数获取 ID
	idStr := c.Query("id")
	if idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.FailWithMessage(c, "无效的ID参数")
			return
		}
		req.ID = id
	}

	// 从 query 参数获取 full_code_path
	req.FullCodePath = c.Query("full_code_path")

	if req.ID == 0 && req.FullCodePath == "" {
		response.FailWithMessage(c, "必须提供 ID 或 full_code_path 参数")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.GetPackageInfo(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "获取目录信息失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// UpdateServiceTree 更新服务目录
// @Summary 更新服务目录
// @Description 更新指定服务目录的信息
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.UpdateServiceTreeMetadataReq true "更新服务目录请求"
// @Success 200 {string} string "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree [put]
func (s *ServiceTree) UpdateServiceTree(c *gin.Context) {
	var req dto.UpdateServiceTreeMetadataReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 更新服务目录
	ctx := contextx.ToContext(c)
	if err := s.serviceTreeService.UpdateServiceTreeMetadata(ctx, &req); err != nil {
		response.FailWithMessage(c, "更新服务目录失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "更新成功")
}

// DeleteServiceTree 删除服务目录
// @Summary 删除服务目录
// @Description 删除指定服务目录（级联删除子目录）
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.DeleteServiceTreeReq true "删除服务目录请求"
// @Success 200 {string} string "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree [delete]
func (s *ServiceTree) DeleteServiceTree(c *gin.Context) {
	var req dto.DeleteServiceTreeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 删除服务目录
	ctx := contextx.ToContext(c)
	if err := s.serviceTreeService.DeleteServiceTree(ctx, req.ID); err != nil {
		response.FailWithMessage(c, "删除服务目录失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// CopyServiceTree 复制服务目录（递归复制目录及其所有子目录）
// @Summary 复制服务目录
// @Description 递归复制服务目录及其所有子目录到目标目录，保持目录结构
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CopyDirectoryReq true "复制请求，source_directory_path=源目录完整路径，target_directory_path=目标目录完整路径"
// @Success 200 {object} dto.CopyDirectoryResp "复制成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/copy [post]
func (s *ServiceTree) CopyServiceTree(c *gin.Context) {
	var req dto.CopyDirectoryReq
	var resp *dto.CopyDirectoryResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "CopyServiceTree req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = s.serviceTreeService.CopyServiceTree(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// PublishDirectoryToHub 发布目录到 Hub
// @Summary 发布目录到 Hub
// @Description 将指定目录及其所有子目录发布到 Hub 市场
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.PublishDirectoryToHubReq true "发布目录请求"
// @Success 200 {object} dto.PublishDirectoryToHubResp "发布成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/publish_to_hub [post]
func (s *ServiceTree) PublishDirectoryToHub(c *gin.Context) {
	var req dto.PublishDirectoryToHubReq
	var resp *dto.PublishDirectoryToHubResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "PublishDirectoryToHub req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = s.serviceTreeService.PublishDirectoryToHub(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// PushDirectoryToHub 推送目录到 Hub（更新已发布的目录）
// @Summary 推送目录到 Hub
// @Description 推送目录更新到 Hub（类似 git push），更新已发布的目录
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.PushDirectoryToHubReq true "推送目录请求"
// @Success 200 {object} dto.PushDirectoryToHubResp "推送成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/push_to_hub [post]
func (s *ServiceTree) PushDirectoryToHub(c *gin.Context) {
	var req dto.PushDirectoryToHubReq
	var resp *dto.PushDirectoryToHubResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "PushDirectoryToHub req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = s.serviceTreeService.PushDirectoryToHub(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// AddFunctions 向服务目录添加函数（服务间调用，不需要JWT验证）
// @Summary 向服务目录添加函数
// @Description 接收来自 agent-server 的代码，写入到工作空间对应的目录下，并更新工作空间
// @Description
// @Description **处理模式**：
// @Description - async=true: 异步处理，立即返回 202 Accepted，后台处理完成后通过回调通知 agent-server
// @Description - async=false: 同步处理，等待处理完成后直接返回结果
// @Description
// @Description **回调机制**：
// @Description - 异步模式下，处理完成后会调用 agent-server 的回调接口：POST /agent/api/v1/workspace/update/callback
// @Description - 回调会携带处理结果（成功/失败、生成的函数组代码列表等）
// @Tags 服务目录
// @Accept json
// @Produce json
// @Param X-Trace-Id header string false "追踪ID（用于链路追踪）"
// @Param X-Request-User header string false "请求用户（用于审计）"
// @Param X-Token header string false "Token（服务间调用时透传）"
// @Param request body dto.AddFunctionsReq true "添加函数请求"
// @Success 200 {object} dto.AddFunctionsResp "处理成功（同步模式）"
// @Success 202 {object} map[string]interface{} "已接收，处理中（异步模式）"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/service_tree/add_functions [post]
func (s *ServiceTree) AddFunctions(c *gin.Context) {
	var req dto.AddFunctionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(c, "[ServiceTree API] 解析请求失败: %v", err)
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)

	// 根据 async 参数决定处理方式
	if req.Async {
		// 异步模式：返回已接收，后台处理，通过回调通知
		go func() {
			// 异步处理，不等待结果
			_ = s.serviceTreeService.ProcessFunctionGenResult(ctx, &req)
		}()

		// 立即返回已接收（使用 200 OK 状态码，因为 callAPI 只接受 200）
		response.OkWithData(c, &dto.AddFunctionsAsyncResp{
			RecordID: req.RecordID,
			Message:  "函数添加请求已接收，正在异步处理",
		})
	} else {
		// 同步模式：等待处理完成，直接返回结果
		resp, err := s.serviceTreeService.AddFunctions(ctx, &req)
		if err != nil {
			logger.Errorf(c, "[ServiceTree API] 处理失败: %v", err)
			response.FailWithMessage(c, "处理失败: "+err.Error())
			return
		}

		response.OkWithData(c, resp)
	}
}

// GetHubInfo 获取目录的 Hub 信息
// @Summary 获取目录的 Hub 信息
// @Description 根据目录完整路径获取其关联的 Hub 目录信息
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full_code_path query string true "目录完整路径"
// @Success 200 {object} dto.GetHubInfoResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "目录未发布到 Hub"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/hub_info [get]
func (s *ServiceTree) GetHubInfo(c *gin.Context) {
	var req dto.GetHubInfoReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.GetHubInfo(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// PullDirectoryFromHub 从 Hub 拉取目录到工作空间
// @Summary 从 Hub 拉取目录
// @Description 使用 Hub 链接从 Hub 拉取目录到工作空间（类似 git pull）
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.PullDirectoryFromHubReq true "拉取目录请求"
// @Success 200 {object} dto.PullDirectoryFromHubResp "拉取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/pull_from_hub [post]
func (s *ServiceTree) PullDirectoryFromHub(c *gin.Context) {
	var req dto.PullDirectoryFromHubReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.PullDirectoryFromHub(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// SearchFunctions 搜索函数
// @Summary 搜索函数
// @Description 根据关键词、类型等条件搜索函数，支持分页
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user query string false "用户名（可选，用于过滤应用）"
// @Param app query string false "应用名（可选，用于过滤应用）"
// @Param keyword query string false "搜索关键词（可选，用于搜索名称和路径）"
// @Param template_type query string false "模板类型过滤（可选，如：form、table、chart）"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.SearchFunctionsResp "搜索成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/search_functions [get]
func (s *ServiceTree) SearchFunctions(c *gin.Context) {
	var req dto.SearchFunctionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 验证分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 限制最大每页数量
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.SearchFunctions(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "搜索函数失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}
