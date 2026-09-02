package v1

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/llms"
	"github.com/kageos/kageos/pkg/logger"
)

func llmAPIKeyForResponse(cfg *model.LLMConfig) (string, bool) {
	if cfg.APIKey == "" {
		return "", false
	}
	return "", true
}

func llmInfoCount(resp *dto.LLMListResp) int {
	if resp == nil {
		return 0
	}
	return len(resp.Configs)
}

func llmStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func llmStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func llmProviderProtocolForResponse(cfg *model.LLMConfig) (string, string) {
	provider := model.LLMProviderOpenAI
	protocol := model.LLMProtocolOpenAIChatCompletions
	apiBase := ""
	endpointPath := ""
	if cfg != nil {
		if cfg.Provider != "" {
			provider = cfg.Provider
		}
		if cfg.Protocol != "" {
			protocol = cfg.Protocol
		}
		apiBase = cfg.APIBase
		endpointPath = cfg.EndpointPath
	}
	return llms.InferProviderProtocol(provider, protocol, apiBase, endpointPath)
}

func llmInfoFromConfig(cfg *model.LLMConfig, currentUser string) dto.LLMInfo {
	apiKey, hasAPIKey := llmAPIKeyForResponse(cfg)
	provider, protocol := llmProviderProtocolForResponse(cfg)
	effectiveContextWindow, contextWindowSource := service.ResolveLLMContextWindow(cfg)
	effectiveMaxOutputTokens, maxOutputTokenSource := service.ResolveLLMMaxOutputTokens(cfg)
	isAdmin := cfg.IsAdminUser(currentUser)
	headers := ""
	extraConfig := ""
	admin := ""
	if isAdmin {
		headers = llmStringValue(cfg.Headers)
		extraConfig = llmStringValue(cfg.ExtraConfig)
		admin = cfg.Admin
	}
	return dto.LLMInfo{
		ID:                           cfg.ID,
		Code:                         cfg.Code,
		Name:                         cfg.Name,
		Provider:                     provider,
		Protocol:                     protocol,
		Model:                        cfg.Model,
		APIKey:                       apiKey,
		HasAPIKey:                    hasAPIKey,
		APIBase:                      cfg.APIBase,
		EndpointPath:                 cfg.EndpointPath,
		APIVersion:                   cfg.APIVersion,
		AuthScheme:                   cfg.AuthScheme,
		Headers:                      headers,
		Timeout:                      cfg.Timeout,
		MaxTokens:                    cfg.MaxTokens,
		DetectedMaxOutputTokens:      cfg.DetectedMaxOutputTokens,
		DetectedMaxOutputTokenSource: cfg.DetectedMaxOutputTokenSource,
		EffectiveMaxOutputTokens:     effectiveMaxOutputTokens,
		MaxOutputTokenSource:         maxOutputTokenSource,
		ContextWindow:                cfg.ContextWindow,
		DetectedContextWindow:        cfg.DetectedContextWindow,
		DetectedContextWindowSource:  cfg.DetectedContextWindowSource,
		EffectiveContextWindow:       effectiveContextWindow,
		ContextWindowSource:          contextWindowSource,
		ExtraConfig:                  extraConfig,
		Capabilities:                 llmStringValue(cfg.Capabilities),
		IsDefault:                    cfg.IsDefault,
		Visibility:                   cfg.Visibility,
		Admin:                        admin,
		IsAdmin:                      isAdmin,
		CreatedAt:                    time.Time(cfg.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:                    time.Time(cfg.UpdatedAt).Format("2006-01-02T15:04:05Z"),
	}
}

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
// @Router /agent/api/v1/llm/list [get]
func (h *LLM) List(c *gin.Context) {
	var req dto.LLMListReq
	var resp *dto.LLMListResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		total := int64(0)
		if resp != nil {
			total = resp.Total
		}
		logger.Debugf(c, "LLM.List scope=%s page=%d page_size=%d count=%d total=%d err:%v", req.Scope, req.Page, req.PageSize, llmInfoCount(resp), total, err)
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
		llmInfos = append(llmInfos, llmInfoFromConfig(cfg, currentUser))
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
// @Router /agent/api/v1/llm/get [get]
func (h *LLM) Get(c *gin.Context) {
	var req dto.LLMGetReq
	var resp *dto.LLMGetResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		hasAPIKey := false
		code := ""
		if resp != nil {
			hasAPIKey = resp.LLMInfo.HasAPIKey || resp.LLMInfo.APIKey != ""
			code = resp.LLMInfo.Code
		}
		logger.Debugf(c, "LLM.Get id=%d found=%v code=%s has_api_key=%v err:%v", req.ID, resp != nil, code, hasAPIKey, err)
	}()

	ctx := contextx.ToContext(c)
	cfg, err := h.service.GetViewableLLMConfig(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.LLMGetResp{
		LLMInfo: llmInfoFromConfig(cfg, contextx.GetRequestUser(ctx)),
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
// @Router /agent/api/v1/llm/get_default [get]
func (h *LLM) GetDefault(c *gin.Context) {
	var resp *dto.LLMGetDefaultResp
	var err error

	defer func() {
		hasAPIKey := false
		code := ""
		if resp != nil {
			hasAPIKey = resp.LLMInfo.HasAPIKey || resp.LLMInfo.APIKey != ""
			code = resp.LLMInfo.Code
		}
		logger.Debugf(c, "LLM.GetDefault found=%v code=%s has_api_key=%v err:%v", resp != nil, code, hasAPIKey, err)
	}()

	ctx := contextx.ToContext(c)
	cfg, err := h.service.GetViewableDefaultLLMConfig(ctx)
	if err != nil {
		if errors.Is(err, service.ErrDefaultLLMNotConfigured) {
			response.OkWithData(c, nil)
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.LLMGetDefaultResp{
		LLMInfo: llmInfoFromConfig(cfg, contextx.GetRequestUser(ctx)),
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
// @Router /agent/api/v1/llm/create [post]
func (h *LLM) Create(c *gin.Context) {
	var req dto.LLMCreateReq
	var resp *dto.LLMCreateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		respID := int64(0)
		if resp != nil {
			respID = resp.ID
		}
		logger.Debugf(c, "LLM.Create name=%s model=%s visibility=%d admin_set=%v is_default=%v api_key_provided=%v extra_config_provided=%v id=%d err:%v", req.Name, req.Model, req.Visibility, req.Admin != "", req.IsDefault, req.APIKey != "", req.ExtraConfig != nil && *req.ExtraConfig != "", respID, err)
	}()

	ctx := contextx.ToContext(c)
	cfg := &model.LLMConfig{
		Name:                         req.Name,
		Provider:                     req.Provider,
		Protocol:                     req.Protocol,
		Model:                        req.Model,
		APIKey:                       req.APIKey,
		APIBase:                      req.APIBase,
		EndpointPath:                 req.EndpointPath,
		APIVersion:                   req.APIVersion,
		AuthScheme:                   req.AuthScheme,
		Headers:                      req.Headers,
		Timeout:                      req.Timeout,
		MaxTokens:                    req.MaxTokens,
		DetectedMaxOutputTokens:      req.DetectedMaxOutputTokens,
		DetectedMaxOutputTokenSource: req.DetectedMaxOutputTokenSource,
		ContextWindow:                req.ContextWindow,
		DetectedContextWindow:        req.DetectedContextWindow,
		DetectedContextWindowSource:  req.DetectedContextWindowSource,
		ExtraConfig:                  req.ExtraConfig,
		Capabilities:                 req.Capabilities,
		IsDefault:                    req.IsDefault,
		Visibility:                   req.Visibility,
		Admin:                        req.Admin,
	}

	if err := h.service.CreateLLMConfig(ctx, cfg); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.LLMCreateResp{ID: cfg.ID}
	response.OkWithData(c, resp)
}

// Probe 检测 LLM 协议与密钥可用性
// @Summary 检测 LLM 协议与密钥可用性
// @Description 根据当前表单内容发起最小请求，自动识别 OpenAI Chat、OpenAI Responses 或 Anthropic Messages
// @Tags LLM管理
// @Accept json
// @Produce json
// @Param request body dto.LLMProbeReq true "LLM 检测请求"
// @Success 200 {object} dto.LLMProbeResp "检测结果"
// @Failure 400 {string} string "请求参数错误"
// @Router /agent/api/v1/llm/probe [post]
func (h *LLM) Probe(c *gin.Context) {
	var req dto.LLMProbeReq
	var resp *dto.LLMProbeResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		ok := false
		protocol := ""
		attemptCount := 0
		if resp != nil {
			ok = resp.OK
			protocol = resp.Protocol
			attemptCount = len(resp.Attempts)
		}
		logger.Debugf(c, "LLM.Probe id=%d provider=%s protocol=%s model=%s ok=%v attempts=%d err:%v", req.ID, req.Provider, protocol, req.Model, ok, attemptCount, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = h.service.ProbeLLMConfig(ctx, req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

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
// @Router /agent/api/v1/llm/update [post]
func (h *LLM) Update(c *gin.Context) {
	var req dto.LLMUpdateReq
	var resp *dto.LLMUpdateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		respID := int64(0)
		if resp != nil {
			respID = resp.ID
		}
		logger.Debugf(c, "LLM.Update id=%d name=%s model=%s visibility=%d admin_set=%v is_default=%v api_key_provided=%v extra_config_provided=%v resp_id=%d err:%v", req.ID, req.Name, req.Model, req.Visibility, req.Admin != "", req.IsDefault, req.APIKey != "", req.ExtraConfig != "", respID, err)
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
	cfg.Protocol = req.Protocol
	cfg.Model = req.Model
	cfg.APIKey = req.APIKey
	cfg.APIBase = req.APIBase
	cfg.EndpointPath = req.EndpointPath
	cfg.APIVersion = req.APIVersion
	cfg.AuthScheme = req.AuthScheme
	cfg.Headers = llmStringPtr(req.Headers)
	cfg.Timeout = req.Timeout
	cfg.MaxTokens = req.MaxTokens
	cfg.DetectedMaxOutputTokens = req.DetectedMaxOutputTokens
	cfg.DetectedMaxOutputTokenSource = req.DetectedMaxOutputTokenSource
	cfg.ContextWindow = req.ContextWindow
	cfg.DetectedContextWindow = req.DetectedContextWindow
	cfg.DetectedContextWindowSource = req.DetectedContextWindowSource
	if req.ExtraConfig != "" {
		extraConfig := req.ExtraConfig
		cfg.ExtraConfig = &extraConfig
	} else {
		cfg.ExtraConfig = nil
	}
	cfg.Capabilities = llmStringPtr(req.Capabilities)
	cfg.IsDefault = req.IsDefault
	cfg.Visibility = req.Visibility
	cfg.Admin = req.Admin

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
// @Router /agent/api/v1/llm/delete [post]
func (h *LLM) Delete(c *gin.Context) {
	var req dto.LLMDeleteReq
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Debugf(c, "LLM.Delete id=%d err:%v", req.ID, err)
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
// @Router /agent/api/v1/llm/set_default [post]
func (h *LLM) SetDefault(c *gin.Context) {
	var req dto.LLMSetDefaultReq
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Debugf(c, "LLM.SetDefault id=%d err:%v", req.ID, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.SetDefaultLLMConfig(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "设置成功")
}
