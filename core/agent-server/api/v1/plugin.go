package v1

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/utils"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Plugin 插件 API 处理器
type Plugin struct {
	service *service.PluginService
	cfg     *config.AgentServerConfig
}


// NewPlugin 创建插件 API 处理器
func NewPlugin(service *service.PluginService, cfg *config.AgentServerConfig) *Plugin {
	return &Plugin{service: service, cfg: cfg}
}

// List 获取插件列表
// @Summary 获取插件列表
// @Description 获取所有可用插件列表
// @Tags 插件管理
// @Accept json
// @Produce json
// @Param enabled query bool false "是否启用"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.PluginListResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/plugins/list [get]
func (h *Plugin) List(c *gin.Context) {
	var req dto.PluginListReq
	var resp *dto.PluginListResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		if err != nil {
			logger.Errorf(c, "[Plugin.List] 获取插件列表失败: %v", err)
			response.FailWithMessage(c, err.Error())
		}
	}()

	ctx := contextx.ToContext(c)
	currentUser := contextx.GetRequestUser(ctx)
	enabled := req.Enabled

	plugins, total, err := h.service.ListPlugins(ctx, req.Scope, enabled, req.Page, req.PageSize)
	if err != nil {
		return
	}

	// 转换为 DTO
	pluginInfos := make([]dto.PluginInfo, 0, len(plugins))
	for _, plugin := range plugins {
		pluginInfos = append(pluginInfos, dto.PluginInfo{
			ID:          plugin.ID,
			Name:        plugin.Name,
			Code:        plugin.Code,
			Description: plugin.Description,
			Enabled:     plugin.Enabled,
			FormPath:    plugin.FormPath,
			Config:      plugin.Config,
			User:        plugin.User,
			Visibility:  plugin.Visibility,
			Admin:       plugin.Admin,
			IsAdmin:     utils.IsAdmin(plugin.Admin, currentUser),
			CreatedAt:   time.Time(plugin.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   time.Time(plugin.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		})
	}

	resp = &dto.PluginListResp{
		Plugins: pluginInfos,
		Total:   total,
		Page:    req.Page,
		PageSize: req.PageSize,
	}

	response.OkWithData(c, resp)
}

// Create 创建插件
// @Summary 创建插件
// @Description 创建新的插件
// @Tags 插件管理
// @Accept json
// @Produce json
// @Param plugin body dto.CreatePluginReq true "插件信息"
// @Success 200 {object} dto.PluginInfo "创建成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/plugins [post]
func (h *Plugin) Create(c *gin.Context) {
	var req dto.CreatePluginReq
	var resp *dto.PluginInfo
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		if err != nil {
			logger.Errorf(c, "[Plugin.Create] 创建插件失败: %v", err)
			response.FailWithMessage(c, err.Error())
		}
	}()

	ctx := contextx.ToContext(c)

	plugin := &model.Plugin{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Enabled:     req.Enabled,
		Config:      req.Config,
	}

	if err = h.service.CreatePlugin(ctx, plugin); err != nil {
		return
	}

	resp = &dto.PluginInfo{
		ID:          plugin.ID,
		Name:        plugin.Name,
		Code:        plugin.Code,
		Description: plugin.Description,
		Enabled:     plugin.Enabled,
		FormPath:    plugin.FormPath,
		Config:      plugin.Config,
		User:        plugin.User,
		CreatedAt:   time.Time(plugin.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   time.Time(plugin.UpdatedAt).Format("2006-01-02T15:04:05Z"),
	}

	response.OkWithData(c, resp)
}

// Update 更新插件
// @Summary 更新插件
// @Description 更新插件信息
// @Tags 插件管理
// @Accept json
// @Produce json
// @Param id path int true "插件ID"
// @Param plugin body dto.UpdatePluginReq true "插件信息"
// @Success 200 {object} dto.PluginInfo "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/plugins/{id} [put]
func (h *Plugin) Update(c *gin.Context) {
	var req dto.UpdatePluginReq
	var resp *dto.PluginInfo
	var err error

	var uriParams struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err = c.ShouldBindUri(&uriParams); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	id := uriParams.ID

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		if err != nil {
			logger.Errorf(c, "[Plugin.Update] 更新插件失败: %v", err)
			response.FailWithMessage(c, err.Error())
		}
	}()

	ctx := contextx.ToContext(c)

	// 获取现有插件
	plugin, err := h.service.GetPlugin(ctx, id)
	if err != nil {
		return
	}

	// 更新字段
	plugin.Name = req.Name
	plugin.Code = req.Code
	plugin.Description = req.Description
	plugin.Enabled = req.Enabled
	plugin.FormPath = req.FormPath
	plugin.Config = req.Config

	if err = h.service.UpdatePlugin(ctx, plugin); err != nil {
		return
	}

	resp = &dto.PluginInfo{
		ID:          plugin.ID,
		Name:        plugin.Name,
		Code:        plugin.Code,
		Description: plugin.Description,
		Enabled:     plugin.Enabled,
		FormPath:    plugin.FormPath,
		Config:      plugin.Config,
		User:        plugin.User,
		CreatedAt:   time.Time(plugin.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   time.Time(plugin.UpdatedAt).Format("2006-01-02T15:04:05Z"),
	}

	response.OkWithData(c, resp)
}

// Get 获取插件详情
// @Summary 获取插件详情
// @Description 根据ID获取插件详情
// @Tags 插件管理
// @Accept json
// @Produce json
// @Param id path int true "插件ID"
// @Success 200 {object} dto.PluginInfo "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/plugins/{id} [get]
func (h *Plugin) Get(c *gin.Context) {
	var resp *dto.PluginInfo
	var err error

	var uriParams struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err = c.ShouldBindUri(&uriParams); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	id := uriParams.ID

	defer func() {
		if err != nil {
			logger.Errorf(c, "[Plugin.Get] 获取插件详情失败: %v", err)
			response.FailWithMessage(c, err.Error())
		}
	}()

	ctx := contextx.ToContext(c)

	plugin, err := h.service.GetPlugin(ctx, id)
	if err != nil {
		return
	}

	resp = &dto.PluginInfo{
		ID:          plugin.ID,
		Name:        plugin.Name,
		Code:        plugin.Code,
		Description: plugin.Description,
		Enabled:     plugin.Enabled,
		FormPath:    plugin.FormPath,
		Config:      plugin.Config,
		User:        plugin.User,
		CreatedAt:   time.Time(plugin.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   time.Time(plugin.UpdatedAt).Format("2006-01-02T15:04:05Z"),
	}

	response.OkWithData(c, resp)
}

// Delete 删除插件
// @Summary 删除插件
// @Description 删除插件（软删除）
// @Tags 插件管理
// @Accept json
// @Produce json
// @Param id path int true "插件ID"
// @Success 200 {string} string "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/plugins/{id} [delete]
func (h *Plugin) Delete(c *gin.Context) {
	var err error

	var uriParams struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err = c.ShouldBindUri(&uriParams); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	id := uriParams.ID

	defer func() {
		if err != nil {
			logger.Errorf(c, "[Plugin.Delete] 删除插件失败: %v", err)
			response.FailWithMessage(c, err.Error())
		}
	}()

	ctx := contextx.ToContext(c)

	if err = h.service.DeletePlugin(ctx, id); err != nil {
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// Enable 启用插件
// @Summary 启用插件
// @Description 启用插件
// @Tags 插件管理
// @Accept json
// @Produce json
// @Param id path int true "插件ID"
// @Success 200 {string} string "启用成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/plugins/{id}/enable [post]
func (h *Plugin) Enable(c *gin.Context) {
	var err error

	var uriParams struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err = c.ShouldBindUri(&uriParams); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	id := uriParams.ID

	defer func() {
		if err != nil {
			logger.Errorf(c, "[Plugin.Enable] 启用插件失败: %v", err)
			response.FailWithMessage(c, err.Error())
		}
	}()

	ctx := contextx.ToContext(c)

	if err = h.service.EnablePlugin(ctx, id); err != nil {
		return
	}

	response.OkWithMessage(c, "启用成功")
}

// Disable 禁用插件
// @Summary 禁用插件
// @Description 禁用插件
// @Tags 插件管理
// @Accept json
// @Produce json
// @Param id path int true "插件ID"
// @Success 200 {string} string "禁用成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/plugins/{id}/disable [post]
func (h *Plugin) Disable(c *gin.Context) {
	var err error

	var uriParams struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err = c.ShouldBindUri(&uriParams); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	id := uriParams.ID

	defer func() {
		if err != nil {
			logger.Errorf(c, "[Plugin.Disable] 禁用插件失败: %v", err)
			response.FailWithMessage(c, err.Error())
		}
	}()

	ctx := contextx.ToContext(c)

	if err = h.service.DisablePlugin(ctx, id); err != nil {
		return
	}

	response.OkWithMessage(c, "禁用成功")
}

