package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
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

// UpdateServiceTree 更新服务目录
// @Summary 更新服务目录
// @Description 更新指定服务目录的信息
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.UpdateServiceTreeReq true "更新服务目录请求"
// @Success 200 {string} string "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree [put]
func (s *ServiceTree) UpdateServiceTree(c *gin.Context) {
	var req dto.UpdateServiceTreeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 更新服务目录
	ctx := contextx.ToContext(c)
	if err := s.serviceTreeService.UpdateServiceTree(ctx, &req); err != nil {
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
