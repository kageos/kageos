package v1

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/llms"
	"github.com/kageos/kageos/pkg/logger"
)

// Workspace 工作台 API 处理器（只认 LLM，单模式）
type Workspace struct {
	toolReg   *service.ToolRegistry
	wsChatSvc *service.WorkspaceChatService
}

// NewWorkspace 创建工作台 API 处理器
func NewWorkspace(toolReg *service.ToolRegistry, wsChatSvc *service.WorkspaceChatService) *Workspace {
	return &Workspace{toolReg: toolReg, wsChatSvc: wsChatSvc}
}

// ChatStream
// Chat 工作台对话（SSE 流式；WorkspaceChatService 编排 + Tool 循环）
// POST /agent/api/v1/workspace/chat/stream
func (h *Workspace) ChatStream(c *gin.Context) {
	var req dto.WorkspaceChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	logger.Infof(ctx, "[Workspace] Chat - full_code_path=%s, session_id=%s, len(content)=%d",
		req.FullCodePath, req.SessionID, len(req.Message.Content))

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 创建事件 channel，service 层会发送事件到这里
	eventChan := make(chan service.StreamEvent, 100)

	// 在 goroutine 中执行 WorkspaceChatStream（避免阻塞 handler）
	go func() {
		err := h.wsChatSvc.WorkspaceChatStream(ctx, &req, eventChan)
		if err != nil {
			logger.Errorf(ctx, "[Workspace] WorkspaceChatStream 返回错误: %v", err)
		}
		close(eventChan) // 关闭 channel，让主循环退出
	}()

	// 直接在主 goroutine 中循环读取事件并写入 SSE（参考参考实现的简洁方式）
	// 使用 recover 捕获可能的 panic（例如连接已关闭）
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf(ctx, "[Workspace] SSE 写入时发生 panic（可能是连接已关闭）: %v", r)
		}
	}()

	for ev := range eventChan {
		// 检查上下文是否已取消（客户端断开）
		select {
		case <-ctx.Done():
			logger.Infof(ctx, "[Workspace] 客户端断开连接，停止处理")
			return
		default:
		}

		// 写入 SSE 事件
		c.SSEvent(ev.Event, ev.Data)
		c.Writer.Flush()
	}
}

// BatchToolDetails 批量获取内置工作台工具展示详情。
// POST /agent/api/v1/workspace/tools/batch_detail
func (h *Workspace) BatchToolDetails(c *gin.Context) {
	var req dto.BatchWorkspaceToolDetailsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	if h.toolReg == nil {
		response.FailWithMessage(c, "工具注册表未初始化")
		return
	}

	names := normalizeWorkspaceToolDetailNames(req.Names)
	if len(names) > 100 {
		response.FailWithMessage(c, "一次最多查询 100 个工具")
		return
	}

	ctx := contextx.ToContext(c)
	defs, err := h.toolReg.ListTools(ctx, names)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	tools := make([]dto.WorkspaceToolDetail, 0, len(defs))
	found := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		found[def.Name] = struct{}{}
		tools = append(tools, workspaceToolDetailFromDef(def, req.IncludeSchema))
	}

	missing := make([]string, 0)
	for _, name := range names {
		if _, ok := found[name]; !ok {
			missing = append(missing, name)
		}
	}

	response.OkWithData(c, dto.BatchWorkspaceToolDetailsResp{
		Tools:   tools,
		Missing: missing,
	})
}

// ListSessions 获取工作台会话列表（根据 full_code_path）
// GET /agent/api/v1/workspace/sessions
func (h *Workspace) ListSessions(c *gin.Context) {
	var req dto.ListWorkspaceSessionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	sessions, total, agents, err := h.wsChatSvc.ListSessionsFiltered(
		ctx, req.FullCodePath, req.SessionScope, req.AutomationTaskID, req.Page, req.PageSize,
	)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, &dto.ListWorkspaceSessionsResp{
		Sessions:         sessions,
		AutomationAgents: agents,
		Total:            total,
		Page:             req.Page,
		PageSize:         req.PageSize,
	})
}

// CreateSessionHandoff 创建阶段交接会话。
// POST /agent/api/v1/workspace/sessions/handoff
func (h *Workspace) CreateSessionHandoff(c *gin.Context) {
	var req dto.WorkspaceHandoffReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := h.wsChatSvc.CreateWorkspaceHandoff(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// ResolvePendingInteraction 清除工作台会话的待确认/待测试状态。
// POST /agent/api/v1/workspace/sessions/interaction/resolve
func (h *Workspace) ResolvePendingInteraction(c *gin.Context) {
	var req dto.ResolveWorkspacePendingInteractionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := h.wsChatSvc.ResolveWorkspacePendingInteraction(ctx, req.SessionID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

// RecordInteractionEvent 记录工作台交互卡片事件，仅用于审计展示。
// POST /agent/api/v1/workspace/sessions/interaction/event
func (h *Workspace) RecordInteractionEvent(c *gin.Context) {
	var req dto.RecordWorkspaceInteractionEventReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := h.wsChatSvc.RecordWorkspaceInteractionEvent(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

// ListMessages 获取工作台会话消息列表
// GET /agent/api/v1/workspace/messages
func (h *Workspace) ListMessages(c *gin.Context) {
	var req dto.ListWorkspaceMessagesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	messages, err := h.wsChatSvc.ListMessages(ctx, req.SessionID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 构建 tool_call_id 到 tool 消息的映射（用于查找工具调用结果）
	toolMessageMap := make(map[string]*model.AgentChatMessage)
	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			toolMessageMap[msg.ToolCallID] = msg
		}
	}

	// 转换为响应格式
	messageInfos := make([]dto.WorkspaceMessageInfo, 0, len(messages))
	for _, msg := range messages {
		info := dto.WorkspaceMessageInfo{
			ID:              msg.ID,
			SessionID:       msg.SessionID,
			Role:            msg.Role,
			User:            msg.User,
			Content:         msg.Content,
			DisplayContent:  msg.DisplayContent,
			ThinkingContent: msg.ThinkingContent,
			Files:           msg.Files,
			CreatedAt:       msg.CreatedAt,
			LLMConfigID:     msg.LLMConfigID,
			LLMConfigName:   msg.LLMConfigName,
			LLMProvider:     msg.LLMProvider,
			LLMModel:        msg.LLMModel,
			ContextUsage:    msg.ContextUsage,
			ArtifactKind:    msg.ArtifactKind,
		}
		if msg.LLMUsage != nil && *msg.LLMUsage != "" {
			var usage dto.LLMUsageInfo
			if err := json.Unmarshal([]byte(*msg.LLMUsage), &usage); err == nil {
				info.LLMUsage = &usage
			}
		}
		if msg.ModelContextPlan != nil && *msg.ModelContextPlan != "" {
			var plan dto.WorkspaceModelContextPlan
			if err := json.Unmarshal([]byte(*msg.ModelContextPlan), &plan); err == nil {
				info.ModelContextPlan = &plan
			}
		}
		if msg.ToolCalls != nil && *msg.ToolCalls != "" {
			// 解析 tool_calls JSON
			var toolCalls []llms.ToolCall
			if err := json.Unmarshal([]byte(*msg.ToolCalls), &toolCalls); err == nil {
				// 转换为前端需要的格式，包含完整信息
				for i, tc := range toolCalls {
					toolCallSummary := dto.WorkspaceChatToolCallSummary{
						ID:        tc.ID,
						Index:     i,
						Round:     0,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments, // 包含参数 JSON 字符串
						Status:    "ok",                  // 默认状态
					}
					// 查找对应的 tool 消息，获取结果
					if toolMsg, ok := toolMessageMap[tc.ID]; ok {
						if toolMsg.ToolStatus != "" {
							toolCallSummary.Status = toolMsg.ToolStatus
						}
						if toolCallSummary.Status == "error" {
							toolCallSummary.Error = toolMsg.Content
						} else {
							toolCallSummary.Result = toolMsg.Content
						}
						if toolMsg.ResultData != nil && *toolMsg.ResultData != "" {
							var resultData interface{}
							if err := json.Unmarshal([]byte(*toolMsg.ResultData), &resultData); err == nil {
								toolCallSummary.ResultData = resultData
							}
						}
						if toolMsg.ResultMetadata != nil && *toolMsg.ResultMetadata != "" {
							var metadata dto.ToolResultMetadata
							if err := json.Unmarshal([]byte(*toolMsg.ResultMetadata), &metadata); err == nil {
								toolCallSummary.Metadata = &metadata
							}
						}
						// 如果结果包含错误信息，可以判断状态（这里简化处理，实际可以根据内容判断）
						// 或者从其他地方获取错误状态
					}
					info.ToolCalls = append(info.ToolCalls, toolCallSummary)
				}
			}
		}
		messageInfos = append(messageInfos, info)
	}

	response.OkWithData(c, &dto.ListWorkspaceMessagesResp{
		Messages: messageInfos,
	})
}

// ListRunningSessions 查询当前用户所有正在执行的工作台会话
// GET /agent/api/v1/workspace/sessions/running
func (h *Workspace) ListRunningSessions(c *gin.Context) {
	ctx := contextx.ToContext(c)
	items, err := h.wsChatSvc.ListRunningSessions(ctx)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, gin.H{"sessions": items})
}

// GetSessionSSEStatus 检查 session 的 SSE 连接是否存活（供前端存活检测，SSE 存活则不轮询大消息列表）
// GET /agent/api/v1/workspace/sessions/:session_id/sse-status
func (h *Workspace) GetSessionSSEStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.FailWithMessage(c, "session_id 必填")
		return
	}
	connected := h.wsChatSvc.IsSSEConnected(sessionID)
	response.OkWithData(c, gin.H{"connected": connected})
}

// CancelChat 取消工作台会话执行
// POST /agent/api/v1/workspace/chat/cancel
func (h *Workspace) CancelChat(c *gin.Context) {
	var req dto.CancelWorkspaceChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if err := h.wsChatSvc.CancelSession(ctx, req.SessionID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

// ListFinishedSessions 查询当前用户最近已结束的工作台会话
// GET /agent/api/v1/workspace/sessions/finished
func (h *Workspace) ListFinishedSessions(c *gin.Context) {
	ctx := contextx.ToContext(c)
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if v, err := fmt.Sscanf(limitStr, "%d", &limit); v == 0 || err != nil {
		limit = 20
	}
	items, err := h.wsChatSvc.ListFinishedSessions(ctx, limit)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, gin.H{"sessions": items})
}

func normalizeWorkspaceToolDetailNames(rawNames []string) []string {
	names := make([]string, 0, len(rawNames))
	seen := make(map[string]struct{}, len(rawNames))
	for _, raw := range rawNames {
		name := normalizeWorkspaceToolDetailName(raw)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func normalizeWorkspaceToolDetailName(raw string) string {
	name := strings.TrimSpace(raw)
	if strings.HasPrefix(name, "<") && strings.HasSuffix(name, ">") {
		name = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(name, "<"), ">"))
	}
	name = strings.TrimPrefix(name, "tool:")
	return strings.TrimSpace(name)
}

func workspaceToolDetailFromDef(def dto.ToolDef, includeSchema bool) dto.WorkspaceToolDetail {
	detail := dto.WorkspaceToolDetail{
		Name:        def.Name,
		Description: strings.TrimSpace(def.Description),
		Token:       "<tool:" + def.Name + ">",
		TypeLabel:   "内置工具",
		InputFields: workspaceToolFieldsFromSchema(def.InputSchema),
	}
	if includeSchema {
		detail.InputSchema = def.InputSchema
		detail.OutputSchema = def.OutputSchema
	}
	return detail
}

func workspaceToolFieldsFromSchema(schema map[string]interface{}) []dto.WorkspaceToolField {
	if len(schema) == 0 {
		return nil
	}

	required := workspaceToolRequiredFields(schema["required"])
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok || len(properties) == 0 {
		return nil
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	fields := make([]dto.WorkspaceToolField, 0, len(names))
	for _, name := range names {
		field := dto.WorkspaceToolField{
			Name:     name,
			Required: required[name],
		}
		if prop, ok := properties[name].(map[string]interface{}); ok {
			field.Type = workspaceToolSchemaType(prop["type"])
			if description, ok := prop["description"].(string); ok {
				field.Description = strings.TrimSpace(description)
			}
		}
		fields = append(fields, field)
	}
	return fields
}

func workspaceToolRequiredFields(raw interface{}) map[string]bool {
	required := map[string]bool{}
	switch list := raw.(type) {
	case []interface{}:
		for _, item := range list {
			if name, ok := item.(string); ok && strings.TrimSpace(name) != "" {
				required[strings.TrimSpace(name)] = true
			}
		}
	case []string:
		for _, name := range list {
			if strings.TrimSpace(name) != "" {
				required[strings.TrimSpace(name)] = true
			}
		}
	}
	return required
}

func workspaceToolSchemaType(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case []interface{}:
		types := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				types = append(types, strings.TrimSpace(text))
			}
		}
		return strings.Join(types, " | ")
	case []string:
		return strings.Join(value, " | ")
	default:
		return ""
	}
}
