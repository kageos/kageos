package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Workspace 工作台 API 处理器（智能工作台）
type Workspace struct {
	toolReg   *service.ToolRegistry
	modeRepo  *repository.WorkspaceModeRepository
	wsChatSvc *service.WorkspaceChatService
}

// NewWorkspace 创建工作台 API 处理器
func NewWorkspace(toolReg *service.ToolRegistry, modeRepo *repository.WorkspaceModeRepository, wsChatSvc *service.WorkspaceChatService) *Workspace {
	return &Workspace{toolReg: toolReg, modeRepo: modeRepo, wsChatSvc: wsChatSvc}
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
			// 发送错误事件
			select {
			case eventChan <- service.StreamEvent{Event: "error", Data: service.StreamEventError{Message: err.Error()}}:
			case <-ctx.Done():
			}
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

// ListTools 列出工作台可用工具（临时接口，验证 list_tools）
// GET /agent/api/v1/workspace/tools
func (h *Workspace) ListTools(c *gin.Context) {
	ctx := contextx.ToContext(c)
	list, err := h.toolReg.ListTools(ctx, nil)
	if err != nil {
		response.FailWithMessage(c, "list_tools 失败: "+err.Error())
		return
	}
	response.OkWithData(c, &dto.ListToolsResp{Tools: list})
}

// CallTool 执行工具（临时接口，验证 call_tool；body: full_code_path, tool_name, arguments）
// POST /agent/api/v1/workspace/call_tool
func (h *Workspace) CallTool(c *gin.Context) {
	var req dto.CallToolReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	args := service.ToToolArgs(req.Arguments)
	content, isErr := h.toolReg.CallTool(ctx, req.ToolName, args, req.FullCodePath, nil)
	response.OkWithData(c, &dto.CallToolResp{Content: content, IsError: isErr})
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
	sessions, total, err := h.wsChatSvc.ListSessions(ctx, req.FullCodePath, req.Page, req.PageSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, &dto.ListWorkspaceSessionsResp{
		Sessions: sessions,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
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
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		}
		if msg.AgentID != nil {
			info.AgentID = *msg.AgentID
		}
		if msg.ToolCalls != nil && *msg.ToolCalls != "" {
			// 解析 tool_calls JSON
			var toolCalls []llms.ToolCall
			if err := json.Unmarshal([]byte(*msg.ToolCalls), &toolCalls); err == nil {
				// 转换为前端需要的格式，包含完整信息
				for _, tc := range toolCalls {
					toolCallSummary := dto.WorkspaceChatToolCallSummary{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments, // 包含参数 JSON 字符串
						Status:    "ok",                  // 默认状态
					}
					// 查找对应的 tool 消息，获取结果
					if toolMsg, ok := toolMessageMap[tc.ID]; ok {
						toolCallSummary.Result = toolMsg.Content
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

// modeToItem 将 model.WorkspaceMode 转为 dto.WorkspaceModeItem
func modeToItem(m *model.WorkspaceMode) dto.WorkspaceModeItem {
	item := dto.WorkspaceModeItem{
		ID:                   m.ID,
		Code:                 m.Code,
		Name:                 m.Name,
		Description:          m.Description,
		SystemPromptFragment: m.SystemPromptFragment,
		ToolNames:            m.GetToolNames(),
		AgentID:              m.AgentID,
		SortOrder:            m.SortOrder,
		IsBuiltin:            m.IsBuiltin,
	}
	return item
}

// ListModes 工作台模式列表
// GET /agent/api/v1/workspace/modes
func (h *Workspace) ListModes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	list, total, err := h.modeRepo.List(offset, pageSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	items := make([]dto.WorkspaceModeItem, 0, len(list))
	for _, m := range list {
		items = append(items, modeToItem(m))
	}
	response.OkWithData(c, &dto.ListWorkspaceModesResp{
		List: items, Total: total, Page: page, PageSize: pageSize,
	})
}

// GetMode 工作台模式详情（按 id）
// GET /agent/api/v1/workspace/modes/:id
func (h *Workspace) GetMode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的 id")
		return
	}
	m, err := h.modeRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.FailWithMessage(c, "模式不存在")
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, modeToItem(m))
}

// GetModeByCode 工作台模式详情（按 code，供对话前拉取当前模式）
// GET /agent/api/v1/workspace/modes/by-code?code=dev
func (h *Workspace) GetModeByCode(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.FailWithMessage(c, "缺少 code")
		return
	}
	m, err := h.modeRepo.GetByCode(code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.FailWithMessage(c, "模式不存在")
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, modeToItem(m))
}

// validateToolNames 校验 toolNames 中每个名称均在 ToolRegistry 中存在，返回第一个非法名称；全部合法返回空字符串。调用方传入 ctx := contextx.ToContext(c)。
func (h *Workspace) validateToolNames(ctx context.Context, toolNames []string) string {
	if len(toolNames) == 0 {
		return ""
	}
	list, err := h.toolReg.ListTools(ctx, nil)
	if err != nil {
		return ""
	}
	validSet := make(map[string]struct{}, len(list))
	for _, t := range list {
		validSet[t.Name] = struct{}{}
	}
	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := validSet[name]; !ok {
			return name
		}
	}
	return ""
}

// CreateMode 创建工作台模式
// POST /agent/api/v1/workspace/modes
func (h *Workspace) CreateMode(c *gin.Context) {
	var req dto.CreateWorkspaceModeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	_, err := h.modeRepo.GetByCode(req.Code)
	if err == nil {
		response.FailWithMessage(c, "code 已存在")
		return
	}
	if invalid := h.validateToolNames(ctx, req.ToolNames); invalid != "" {
		response.FailWithMessage(c, fmt.Sprintf("工具名 \"%s\" 不存在，请从可用工具列表中选择", invalid))
		return
	}
	m := &model.WorkspaceMode{
		Code:                 req.Code,
		Name:                 req.Name,
		Description:          req.Description,
		SystemPromptFragment: req.SystemPromptFragment,
		SortOrder:            req.SortOrder,
		IsBuiltin:            false,
		AgentID:              req.AgentID,
	}
	m.SetToolNames(req.ToolNames)
	if err := h.modeRepo.Create(m); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, modeToItem(m))
}

// UpdateMode 更新工作台模式
// PUT /agent/api/v1/workspace/modes/:id
func (h *Workspace) UpdateMode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的 id")
		return
	}
	m, err := h.modeRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.FailWithMessage(c, "模式不存在")
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	var req dto.UpdateWorkspaceModeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	if req.Name != "" {
		m.Name = req.Name
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	m.SystemPromptFragment = req.SystemPromptFragment
	if req.ToolNames != nil {
		if invalid := h.validateToolNames(ctx, req.ToolNames); invalid != "" {
			response.FailWithMessage(c, fmt.Sprintf("工具名 \"%s\" 不存在，请从可用工具列表中选择", invalid))
			return
		}
		m.SetToolNames(req.ToolNames)
	}
	if req.AgentID != nil {
		m.AgentID = req.AgentID
	}
	if req.SortOrder != nil {
		m.SortOrder = *req.SortOrder
	}
	if err := h.modeRepo.Update(m); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, modeToItem(m))
}

// DeleteMode 删除工作台模式（内置不可删）
// DELETE /agent/api/v1/workspace/modes/:id
func (h *Workspace) DeleteMode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的 id")
		return
	}
	m, err := h.modeRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.FailWithMessage(c, "模式不存在")
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	if m.IsBuiltin {
		response.FailWithMessage(c, "内置模式不可删除")
		return
	}
	if err := h.modeRepo.Delete(id); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithMessage(c, "已删除")
}

// ListToolNames 返回所有工具名列表（供模式配置时多选）
// GET /agent/api/v1/workspace/tools/names
func (h *Workspace) ListToolNames(c *gin.Context) {
	ctx := contextx.ToContext(c)
	_ = ctx
	list, err := h.toolReg.ListTools(ctx, nil)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	names := make([]string, 0, len(list))
	for _, t := range list {
		names = append(names, t.Name)
	}
	response.OkWithData(c, gin.H{"names": names})
}
