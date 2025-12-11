package v1

import (
	"fmt"
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

// Agent 智能体 API 处理器
type Agent struct {
	service *service.AgentService
	cfg     *config.AgentServerConfig
}

// NewAgent 创建智能体 API 处理器
func NewAgent(service *service.AgentService, cfg *config.AgentServerConfig) *Agent {
	return &Agent{service: service, cfg: cfg}
}

// List 获取智能体列表
// @Summary 获取智能体列表
// @Description 获取所有可用智能体列表（前端调用）
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param agent_type query string false "智能体类型" Enums(knowledge_only, plugin)
// @Param enabled query bool false "是否启用"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.AgentListResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/agents/list [get]
func (h *Agent) List(c *gin.Context) {
	var req dto.AgentListReq
	var resp *dto.AgentListResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		// 优化日志：不打印 SystemPromptTemplate 等大字段的完整内容
		var safeResp interface{}
		if resp != nil && resp.Agents != nil {
			// 创建一个安全的响应副本，只记录关键信息
			safeAgents := make([]map[string]interface{}, 0, len(resp.Agents))
			for _, agent := range resp.Agents {
				safeAgent := map[string]interface{}{
					"ID":              agent.ID,
					"Name":            agent.Name,
					"AgentType":       agent.AgentType,
					"ChatType":        agent.ChatType,
					"Enabled":         agent.Enabled,
					"Description":     agent.Description,
					"Timeout":         agent.Timeout,
					"KnowledgeBaseID": agent.KnowledgeBaseID,
					"LLMConfigID":      agent.LLMConfigID,
					"PluginID":        agent.PluginID,
					"SystemPromptTemplate": fmt.Sprintf("<len:%d>", len(agent.SystemPromptTemplate)),
				}
				safeAgents = append(safeAgents, safeAgent)
			}
			safeResp = map[string]interface{}{
				"Agents": safeAgents,
				"Total":  resp.Total,
			}
		} else {
			safeResp = resp
		}
		logger.Infof(c, "Agent.List req:%+v resp:%+v err:%v", req, safeResp, err)
	}()

	ctx := contextx.ToContext(c)
	currentUser := contextx.GetRequestUser(ctx)
	agents, total, err := h.service.ListAgents(ctx, req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 转换为响应格式
	agentInfos := make([]dto.AgentInfo, 0, len(agents))
	for _, agent := range agents {
		metadata := ""
		if agent.Metadata != nil {
			metadata = *agent.Metadata
		}

		// 构建知识库信息（如果已预加载）
		var kbInfo *dto.KnowledgeBaseInfo
		if agent.KnowledgeBase.ID > 0 {
			kbInfo = &dto.KnowledgeBaseInfo{
				ID:            agent.KnowledgeBase.ID,
				Name:          agent.KnowledgeBase.Name,
				Description:   agent.KnowledgeBase.Description,
				Status:        agent.KnowledgeBase.Status,
				DocumentCount: agent.KnowledgeBase.DocumentCount,
			}
		}

		// 构建 LLM 配置信息（如果已预加载）
		var llmInfo *dto.LLMConfigInfo
		if agent.LLMConfig.ID > 0 {
			llmInfo = &dto.LLMConfigInfo{
				ID:        agent.LLMConfig.ID,
				Name:      agent.LLMConfig.Name,
				Provider:  agent.LLMConfig.Provider,
				Model:     agent.LLMConfig.Model,
				IsDefault: agent.LLMConfig.IsDefault,
			}
		}

		// 构建插件信息（如果已预加载）
		var pluginInfo *dto.PluginInfo
		if agent.Plugin != nil && agent.Plugin.ID > 0 {
			pluginInfo = &dto.PluginInfo{
				ID:          agent.Plugin.ID,
				Name:        agent.Plugin.Name,
				Code:        agent.Plugin.Code,
				Description: agent.Plugin.Description,
				Enabled:     agent.Plugin.Enabled,
				Subject:     agent.Plugin.Subject,
				NatsHost:    h.cfg.GetNatsHost(),
				Config:      agent.Plugin.Config,
				User:        agent.Plugin.User,
				CreatedAt:   time.Time(agent.Plugin.CreatedAt).Format("2006-01-02T15:04:05Z"),
				UpdatedAt:   time.Time(agent.Plugin.UpdatedAt).Format("2006-01-02T15:04:05Z"),
			}
		}

		agentInfos = append(agentInfos, dto.AgentInfo{
			ID:                   agent.ID,
			Name:                 agent.Name,
			AgentType:            agent.AgentType,
			ChatType:             agent.ChatType,
			Enabled:              agent.Enabled,
			Description:          agent.Description,
			Timeout:              agent.Timeout,
			PluginID:             agent.PluginID,
			Plugin:               pluginInfo,
			KnowledgeBaseID:      agent.KnowledgeBaseID,
			KnowledgeBase:        kbInfo,
			LLMConfigID:          agent.LLMConfigID,
			LLMConfig:            llmInfo,
			SystemPromptTemplate: agent.SystemPromptTemplate,
			Metadata:             metadata,
			Logo:                 agent.Logo,
			Visibility:            agent.Visibility,
			Admin:                agent.Admin,
			IsAdmin:               utils.IsAdmin(agent.Admin, currentUser),
			CreatedAt:            time.Time(agent.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:            time.Time(agent.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		})
	}

	resp = &dto.AgentListResp{
		Agents: agentInfos,
		Total:  total,
	}
	response.OkWithData(c, resp)
}

// Get 获取智能体详情
// @Summary 获取智能体详情
// @Description 根据ID获取智能体详细信息
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param id query int true "智能体ID"
// @Success 200 {object} dto.AgentGetResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/agents/get [get]
func (h *Agent) Get(c *gin.Context) {
	var req dto.AgentGetReq
	var resp *dto.AgentGetResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Agent.Get req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	agent, err := h.service.GetAgent(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	metadata := ""
	if agent.Metadata != nil {
		metadata = *agent.Metadata
	}

	// 构建知识库信息（如果已预加载）
	var kbInfo *dto.KnowledgeBaseInfo
	if agent.KnowledgeBase.ID > 0 {
		kbInfo = &dto.KnowledgeBaseInfo{
			ID:            agent.KnowledgeBase.ID,
			Name:          agent.KnowledgeBase.Name,
			Description:   agent.KnowledgeBase.Description,
			Status:        agent.KnowledgeBase.Status,
			DocumentCount: agent.KnowledgeBase.DocumentCount,
		}
	}

	// 构建 LLM 配置信息（如果已预加载）
	var llmInfo *dto.LLMConfigInfo
	if agent.LLMConfig.ID > 0 {
		llmInfo = &dto.LLMConfigInfo{
			ID:        agent.LLMConfig.ID,
			Name:      agent.LLMConfig.Name,
			Provider:  agent.LLMConfig.Provider,
			Model:     agent.LLMConfig.Model,
			IsDefault: agent.LLMConfig.IsDefault,
		}
	}

	// 构建插件信息（如果已预加载）
	var pluginInfo *dto.PluginInfo
	if agent.Plugin != nil && agent.Plugin.ID > 0 {
		pluginInfo = &dto.PluginInfo{
			ID:          agent.Plugin.ID,
			Name:        agent.Plugin.Name,
			Code:        agent.Plugin.Code,
			Description: agent.Plugin.Description,
			Enabled:     agent.Plugin.Enabled,
			Subject:     agent.Plugin.Subject,
			NatsHost:    h.cfg.GetNatsHost(),
			Config:      agent.Plugin.Config,
			User:        agent.Plugin.User,
			CreatedAt:   time.Time(agent.Plugin.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   time.Time(agent.Plugin.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		}
	}

	resp = &dto.AgentGetResp{
		AgentInfo: dto.AgentInfo{
			ID:                   agent.ID,
			Name:                 agent.Name,
			AgentType:            agent.AgentType,
			ChatType:             agent.ChatType,
			Enabled:              agent.Enabled,
			Description:          agent.Description,
			Timeout:              agent.Timeout,
			PluginID:             agent.PluginID,
			Plugin:               pluginInfo,
			KnowledgeBaseID:      agent.KnowledgeBaseID,
			KnowledgeBase:        kbInfo,
			LLMConfigID:          agent.LLMConfigID,
			LLMConfig:            llmInfo,
			SystemPromptTemplate: agent.SystemPromptTemplate,
			Metadata:             metadata,
			Logo:                 agent.Logo,
			CreatedAt:            time.Time(agent.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:            time.Time(agent.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		},
	}
	response.OkWithData(c, resp)
}

// Create 创建智能体
// @Summary 创建智能体
// @Description 创建新的智能体
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param request body dto.AgentCreateReq true "创建智能体请求"
// @Success 200 {object} dto.AgentCreateResp "创建成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/agents/create [post]
func (h *Agent) Create(c *gin.Context) {
	var req dto.AgentCreateReq
	var resp *dto.AgentCreateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Agent.Create req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	var metadata *string
	if req.Metadata != "" {
		metadata = &req.Metadata
	}
	agent := &model.Agent{
		Name:                 req.Name,
		AgentType:            req.AgentType,
		ChatType:             req.ChatType,
		Description:          req.Description,
		Timeout:              req.Timeout,
		PluginID:             req.PluginID,
		KnowledgeBaseID:      req.KnowledgeBaseID,
		LLMConfigID:          req.LLMConfigID,
		SystemPromptTemplate: req.SystemPromptTemplate,
		Metadata:             metadata,
		Enabled:              true, // 默认启用
	}

	if err := h.service.CreateAgent(ctx, agent); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.AgentCreateResp{ID: agent.ID}
	response.OkWithData(c, resp)
}

// Update 更新智能体
// @Summary 更新智能体
// @Description 更新智能体信息
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param request body dto.AgentUpdateReq true "更新智能体请求"
// @Success 200 {object} dto.AgentUpdateResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/agents/update [post]
func (h *Agent) Update(c *gin.Context) {
	var req dto.AgentUpdateReq
	var resp *dto.AgentUpdateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Agent.Update req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)

	// 先获取现有智能体
	agent, err := h.service.GetAgent(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 更新字段
	agent.Name = req.Name
	agent.AgentType = req.AgentType
	agent.ChatType = req.ChatType
	agent.Description = req.Description
	agent.Timeout = req.Timeout
	agent.PluginID = req.PluginID
	agent.KnowledgeBaseID = req.KnowledgeBaseID
	agent.LLMConfigID = req.LLMConfigID
	agent.SystemPromptTemplate = req.SystemPromptTemplate
	if req.Metadata != "" {
		metadata := req.Metadata
		agent.Metadata = &metadata
	} else {
		agent.Metadata = nil
	}

	if err := h.service.UpdateAgent(ctx, agent); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.AgentUpdateResp{ID: agent.ID}
	response.OkWithData(c, resp)
}

// Delete 删除智能体
// @Summary 删除智能体
// @Description 删除智能体
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param request body dto.AgentDeleteReq true "删除智能体请求"
// @Success 200 "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/agents/delete [post]
func (h *Agent) Delete(c *gin.Context) {
	var req dto.AgentDeleteReq
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Agent.Delete req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.DeleteAgent(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// Enable 启用智能体
// @Summary 启用智能体
// @Description 启用智能体
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param request body dto.AgentEnableReq true "启用智能体请求"
// @Success 200 "启用成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/agents/enable [post]
func (h *Agent) Enable(c *gin.Context) {
	var req dto.AgentEnableReq
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Agent.Enable req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.EnableAgent(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "启用成功")
}

// Disable 禁用智能体
// @Summary 禁用智能体
// @Description 禁用智能体
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param request body dto.AgentDisableReq true "禁用智能体请求"
// @Success 200 "禁用成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/agents/disable [post]
func (h *Agent) Disable(c *gin.Context) {
	var req dto.AgentDisableReq
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Agent.Disable req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.DisableAgent(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "禁用成功")
}
