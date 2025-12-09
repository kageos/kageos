package v1

import (
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
