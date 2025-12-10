package v1

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/utils"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// LLM LLM 配置 API 处理器
type LLM struct {
	service *service.LLMService
}

// NewLLM 创建 LLM 配置 API 处理器
func NewLLM(service *service.LLMService) *LLM {
	return &LLM{service: service}
}

// List 获取LLM配置列表
// @Summary 获取LLM配置列表
// @Description 获取所有LLM配置列表
// @Tags LLM管理
// @Accept json
// @Produce json
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.LLMListResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/llm/list [get]
func (h *LLM) List(c *gin.Context) {
	var req dto.LLMListReq
	var resp *dto.LLMListResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "LLM.List req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	currentUser := contextx.GetRequestUser(ctx)
	configs, total, err := h.service.ListLLMConfigs(ctx, req.Scope, req.Page, req.PageSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 转换为响应格式
	llmInfos := make([]dto.LLMInfo, 0, len(configs))
	for _, cfg := range configs {
		extraConfig := ""
		if cfg.ExtraConfig != nil {
			extraConfig = *cfg.ExtraConfig
		}
		llmInfos = append(llmInfos, dto.LLMInfo{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Provider:    cfg.Provider,
			Model:       cfg.Model,
			APIBase:     cfg.APIBase,
			Timeout:     cfg.Timeout,
			MaxTokens:   cfg.MaxTokens,
			ExtraConfig: extraConfig,
			IsDefault:   cfg.IsDefault,
			Visibility:  cfg.Visibility,
			Admin:       cfg.Admin,
			IsAdmin:     utils.IsAdmin(cfg.Admin, currentUser),
			CreatedAt:   time.Time(cfg.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   time.Time(cfg.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		})
	}

	resp = &dto.LLMListResp{
		Configs: llmInfos,
		Total:   total,
	}
	response.OkWithData(c, resp)
}

// Get 获取LLM配置详情
// @Summary 获取LLM配置详情
// @Description 根据ID获取LLM配置详细信息
// @Tags LLM管理
// @Accept json
// @Produce json
// @Param id query int true "LLM配置ID"
// @Success 200 {object} dto.LLMGetResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/llm/get [get]
func (h *LLM) Get(c *gin.Context) {
	var req dto.LLMGetReq
	var resp *dto.LLMGetResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "LLM.Get req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	cfg, err := h.service.GetLLMConfig(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	extraConfig := ""
	if cfg.ExtraConfig != nil {
		extraConfig = *cfg.ExtraConfig
	}
	resp = &dto.LLMGetResp{
		LLMInfo: dto.LLMInfo{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Provider:    cfg.Provider,
			Model:       cfg.Model,
			APIBase:     cfg.APIBase,
			Timeout:     cfg.Timeout,
			MaxTokens:   cfg.MaxTokens,
			ExtraConfig: extraConfig,
			IsDefault:   cfg.IsDefault,
			CreatedAt:   time.Time(cfg.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   time.Time(cfg.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		},
	}
	response.OkWithData(c, resp)
}

// GetDefault 获取默认LLM配置
// @Summary 获取默认LLM配置
// @Description 获取默认的LLM配置
// @Tags LLM管理
// @Accept json
// @Produce json
// @Success 200 {object} dto.LLMGetDefaultResp "获取成功"
// @Failure 400 {string} string "未设置默认配置"
// @Router /api/v1/agent/llm/get_default [get]
func (h *LLM) GetDefault(c *gin.Context) {
	var resp *dto.LLMGetDefaultResp
	var err error

	defer func() {
		logger.Infof(c, "LLM.GetDefault resp:%+v err:%v", resp, err)
	}()

	ctx := contextx.ToContext(c)
	cfg, err := h.service.GetDefaultLLMConfig(ctx)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	extraConfig := ""
	if cfg.ExtraConfig != nil {
		extraConfig = *cfg.ExtraConfig
	}
	resp = &dto.LLMGetDefaultResp{
		LLMInfo: dto.LLMInfo{
			ID:          cfg.ID,
			Name:        cfg.Name,
			Provider:    cfg.Provider,
			Model:       cfg.Model,
			APIBase:     cfg.APIBase,
			Timeout:     cfg.Timeout,
			MaxTokens:   cfg.MaxTokens,
			ExtraConfig: extraConfig,
			IsDefault:   cfg.IsDefault,
			CreatedAt:   time.Time(cfg.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   time.Time(cfg.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		},
	}
	response.OkWithData(c, resp)
}

// Create 创建LLM配置
// @Summary 创建LLM配置
// @Description 创建新的LLM配置
// @Tags LLM管理
// @Accept json
// @Produce json
// @Param request body dto.LLMCreateReq true "创建LLM配置请求"
// @Success 200 {object} dto.LLMCreateResp "创建成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/llm/create [post]
func (h *LLM) Create(c *gin.Context) {
	var req dto.LLMCreateReq
	var resp *dto.LLMCreateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "LLM.Create req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	cfg := &model.LLMConfig{
		Name:        req.Name,
		Provider:    req.Provider,
		Model:       req.Model,
		APIKey:      req.APIKey,
		APIBase:     req.APIBase,
		Timeout:     req.Timeout,
		MaxTokens:   req.MaxTokens,
		ExtraConfig: req.ExtraConfig,
		IsDefault:   req.IsDefault,
	}

	if err := h.service.CreateLLMConfig(ctx, cfg); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.LLMCreateResp{ID: cfg.ID}
	response.OkWithData(c, resp)
}

// Update 更新LLM配置
// @Summary 更新LLM配置
// @Description 更新LLM配置信息
// @Tags LLM管理
// @Accept json
// @Produce json
// @Param request body dto.LLMUpdateReq true "更新LLM配置请求"
// @Success 200 {object} dto.LLMUpdateResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/llm/update [post]
func (h *LLM) Update(c *gin.Context) {
	var req dto.LLMUpdateReq
	var resp *dto.LLMUpdateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "LLM.Update req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	
	// 先获取现有配置
	cfg, err := h.service.GetLLMConfig(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 更新字段
	cfg.Name = req.Name
	cfg.Provider = req.Provider
	cfg.Model = req.Model
	cfg.APIKey = req.APIKey
	cfg.APIBase = req.APIBase
	cfg.Timeout = req.Timeout
	cfg.MaxTokens = req.MaxTokens
	if req.ExtraConfig != "" {
		extraConfig := req.ExtraConfig
		cfg.ExtraConfig = &extraConfig
	} else {
		cfg.ExtraConfig = nil
	}
	cfg.IsDefault = req.IsDefault

	if err := h.service.UpdateLLMConfig(ctx, cfg); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.LLMUpdateResp{ID: cfg.ID}
	response.OkWithData(c, resp)
}

// Delete 删除LLM配置
// @Summary 删除LLM配置
// @Description 删除LLM配置
// @Tags LLM管理
// @Accept json
// @Produce json
// @Param id query int true "LLM配置ID"
// @Success 200 "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/llm/delete [post]
func (h *LLM) Delete(c *gin.Context) {
	var req dto.LLMDeleteReq
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "LLM.Delete req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.DeleteLLMConfig(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// SetDefault 设置默认LLM配置
// @Summary 设置默认LLM配置
// @Description 设置默认的LLM配置
// @Tags LLM管理
// @Accept json
// @Produce json
// @Param id query int true "LLM配置ID"
// @Success 200 "设置成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/llm/set_default [post]
func (h *LLM) SetDefault(c *gin.Context) {
	var req dto.LLMSetDefaultReq
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "LLM.SetDefault req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.SetDefaultLLMConfig(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "设置成功")
}

