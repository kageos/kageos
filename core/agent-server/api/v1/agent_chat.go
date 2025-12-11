package v1

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// AgentChat 智能体聊天 API 处理器
type AgentChat struct {
	service *service.AgentChatService
}

// NewAgentChat 创建智能体聊天 API 处理器
func NewAgentChat(service *service.AgentChatService) *AgentChat {
	return &AgentChat{service: service}
}

// FunctionGenChat 智能体聊天 - 函数生成类型
// @Summary 智能体聊天 - 函数生成类型
// @Description 与智能体进行对话（function_gen 类型）
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param request body dto.FunctionGenAgentChatReq true "聊天请求"
// @Success 200 {object} dto.FunctionGenAgentChatResp "聊天成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/chat/function_gen [post]
func (h *AgentChat) FunctionGenChat(c *gin.Context) {
	var req dto.FunctionGenAgentChatReq
	var resp *dto.FunctionGenAgentChatResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(c, "[AgentChat] 参数绑定失败: %v", err)
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	user := contextx.GetRequestUser(ctx)
	traceId := contextx.GetTraceId(ctx)

	// 记录请求日志
	logger.Infof(ctx, "[AgentChat] 收到聊天请求 - AgentID: %d, TreeID: %d, SessionID: %s, User: %s, TraceID: %s, MessageLength: %d, FilesCount: %d",
		req.AgentID, req.TreeID, req.SessionID, user, traceId, len(req.Message.Content), len(req.Message.Files))

	defer func() {
		if err != nil {
			logger.Errorf(ctx, "[AgentChat] 处理失败 - AgentID: %d, TreeID: %d, SessionID: %s, User: %s, TraceID: %s, Error: %v",
				req.AgentID, req.TreeID, req.SessionID, user, traceId, err)
		} else {
			logger.Infof(ctx, "[AgentChat] 处理成功 - AgentID: %d, TreeID: %d, SessionID: %s, User: %s, TraceID: %s, ResponseSessionID: %s, RecordID: %d, Status: %s",
				req.AgentID, req.TreeID, req.SessionID, user, traceId, resp.SessionID, resp.RecordID, resp.Status)
		}
	}()

	// 调用聊天服务
	chatResp, err := h.service.FunctionGenChat(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = chatResp
	response.OkWithData(c, resp)
}

// ListSessions 获取会话列表
// @Summary 获取会话列表
// @Description 根据TreeID获取会话列表（会话不绑定到特定智能体，一个会话可以使用多个智能体）
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param tree_id query int true "服务目录ID"
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Success 200 {object} dto.ChatSessionListResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/chat/sessions [get]
func (h *AgentChat) ListSessions(c *gin.Context) {
	var req dto.ChatSessionListReq
	var resp *dto.ChatSessionListResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Errorf(c, "[AgentChat] 参数绑定失败: %v", err)
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	sessions, total, err := h.service.ListSessions(ctx, req.TreeID, req.Page, req.PageSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 转换为响应格式
	sessionInfos := make([]dto.ChatSessionInfo, 0, len(sessions))
	for _, session := range sessions {
		sessionInfo := dto.ChatSessionInfo{
			ID:        session.ID,
			TreeID:    session.TreeID,
			SessionID: session.SessionID,
			AgentID:   session.AgentID,
			Title:     session.Title,
			User:      session.User,
			CreatedAt: time.Time(session.CreatedAt).Format(time.DateTime),
			UpdatedAt: time.Time(session.UpdatedAt).Format(time.DateTime),
		}
		
		// 如果预加载了智能体信息，转换为 DTO
		if session.Agent != nil {
			agentInfo := &dto.AgentInfo{
				ID:              session.Agent.ID,
				Name:            session.Agent.Name,
				AgentType:       session.Agent.AgentType,
				ChatType:        session.Agent.ChatType,
				Enabled:         session.Agent.Enabled,
				Description:     session.Agent.Description,
				Timeout:         session.Agent.Timeout,
				KnowledgeBaseID: session.Agent.KnowledgeBaseID,
				LLMConfigID:     session.Agent.LLMConfigID,
				Logo:            session.Agent.Logo,
			}
			sessionInfo.Agent = agentInfo
		}
		
		sessionInfos = append(sessionInfos, sessionInfo)
	}

	resp = &dto.ChatSessionListResp{
		Sessions: sessionInfos,
		Total:    total,
	}
	response.OkWithData(c, resp)
}

// ListMessages 获取消息列表
// @Summary 获取消息列表
// @Description 根据SessionID获取消息列表
// @Tags 智能体管理
// @Accept json
// @Produce json
// @Param session_id query string true "会话ID"
// @Success 200 {object} dto.ChatMessageListResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/chat/messages [get]
func (h *AgentChat) ListMessages(c *gin.Context) {
	var req dto.ChatMessageListReq
	var resp *dto.ChatMessageListResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Errorf(c, "[AgentChat] 参数绑定失败: %v", err)
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	messages, err := h.service.ListMessages(ctx, req.SessionID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 转换为响应格式
	messageInfos := make([]dto.ChatMessageInfo, 0, len(messages))
	for _, msg := range messages {
		filesStr := ""
		if msg.Files != nil {
			filesStr = *msg.Files
		}
		messageInfos = append(messageInfos, dto.ChatMessageInfo{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			AgentID:   msg.AgentID, // 处理该消息的智能体ID
			Role:      msg.Role,
			Content:   msg.Content,
			Files:     filesStr,
			User:      msg.User,
			CreatedAt: time.Time(msg.CreatedAt).Format(time.DateTime),
		})
	}

	resp = &dto.ChatMessageListResp{
		Messages: messageInfos,
	}
	response.OkWithData(c, resp)
}
