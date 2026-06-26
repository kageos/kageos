package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
	"gorm.io/gorm"
)

// prepareLLMRequest 工作台只认 LLM：llmConfigID > 0 用该配置，否则用默认
func (s *WorkspaceChatService) prepareLLMRequest(ctx context.Context, llmConfigID int64, msgs []llms.Message, tools []llms.ToolDef) (*model.LLMConfig, llms.LLMClient, *llms.ChatRequest, error) {
	var llmConfig *model.LLMConfig
	var err error

	if llmConfigID > 0 {
		llmConfig, err = s.llmRepo.GetByID(llmConfigID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
		}
	}
	if llmConfig == nil {
		llmConfig, err = s.llmRepo.GetDefault()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil, nil, fmt.Errorf("未设置默认 LLM，请在 LLM 管理中配置")
			}
			return nil, nil, nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
		}
	}

	opts := llms.DefaultClientOptions()
	if llmConfig.Model != "" {
		opts = opts.WithModel(llmConfig.Model)
	}
	if llmConfig.Timeout > 0 {
		opts = opts.WithTimeout(time.Duration(llmConfig.Timeout) * time.Second)
	}
	if llmConfig.APIBase != "" {
		opts = opts.WithBaseURL(llmConfig.APIBase)
	}
	client := llms.NewOpenAIClientWithOptions(llmConfig.APIKey, opts)

	var extraConfig map[string]interface{}
	if llmConfig.ExtraConfig != nil && *llmConfig.ExtraConfig != "" {
		_ = json.Unmarshal([]byte(*llmConfig.ExtraConfig), &extraConfig)
	}
	chatReq := &llms.ChatRequest{
		Messages: msgs,
		Model:    llmConfig.Model,
		Tools:    tools, // 添加工具定义
	}
	if maxTokens, ok := extraConfig["max_tokens"].(float64); ok && maxTokens > 0 {
		chatReq.MaxTokens = int(maxTokens)
	} else if llmConfig.MaxTokens > 0 {
		chatReq.MaxTokens = llmConfig.MaxTokens
	} else {
		chatReq.MaxTokens = 4096
	}
	if temperature, ok := extraConfig["temperature"].(float64); ok {
		chatReq.Temperature = temperature
	}
	return llmConfig, client, chatReq, nil
}

func buildMessageLLMMetadata(llmConfig *model.LLMConfig, client llms.LLMClient) messageLLMMetadata {
	meta := messageLLMMetadata{}
	if llmConfig != nil {
		meta.ConfigID = llmConfig.ID
		meta.ConfigName = llmConfig.Name
		meta.Provider = llmConfig.Provider
		meta.Model = llmConfig.Model
	}
	if client != nil {
		if modelName := strings.TrimSpace(client.GetModelName()); modelName != "" {
			meta.Model = modelName
		}
	}
	return meta
}

// convertToLLMTools 将 dto.ToolDef 转换为 llms.ToolDef（标准格式）
func convertToLLMTools(toolsDesc []dto.ToolDef) []llms.ToolDef {
	llmTools := make([]llms.ToolDef, 0, len(toolsDesc))
	for _, t := range toolsDesc {
		llmTools = append(llmTools, llms.ToolDef{
			Type: "function",
			Function: llms.ToolFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema, // InputSchema 已经是 JSON Schema 格式
			},
		})
	}
	return llmTools
}

// workspaceCtxToEnvInput 将 dto 工作台上下文转为 prompt 包的环境输入，供 BuildWorkspaceEnvData 使用
func workspaceCtxToEnvInput(c *dto.GetWorkspaceContextResp) *prompt.WorkspaceEnvInput {
	if c == nil {
		return nil
	}
	dirDesc := ""
	if c.Directory.Description != "" {
		dirDesc = "\n- 目录介绍：" + c.Directory.Description
	}
	children := make([]prompt.WorkspaceEnvNode, 0, len(c.Children))
	for _, n := range c.Children {
		children = append(children, prompt.WorkspaceEnvNode{
			Name:         n.Name,
			Code:         n.Code,
			Description:  n.Description,
			Type:         n.Type,
			FullCodePath: n.FullCodePath,
			TemplateType: n.TemplateType,
			Callbacks:    n.Callbacks,
			Schema:       n.Schema,
		})
	}
	files := make([]prompt.WorkspaceEnvFile, 0, len(c.Files))
	for _, f := range c.Files {
		files = append(files, prompt.WorkspaceEnvFile{
			RelativePath: f.RelativePath,
			FileType:     f.FileType,
			LineCount:    f.LineCount,
		})
	}
	return &prompt.WorkspaceEnvInput{
		User:                   c.User,
		DepartmentFullPath:     c.DepartmentFullPath,
		DepartmentFullNamePath: c.DepartmentFullNamePath,
		DirName:                c.Directory.Name,
		DirCode:                c.Directory.Code,
		FullCodePath:           c.Directory.FullCodePath,
		DirType:                c.Directory.Type,
		DirDescription:         dirDesc,
		Children:               children,
		Files:                  files,
	}
}

func normalizeWorkspaceModeCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "dev"
	}
	return code
}

func (s *WorkspaceChatService) buildLLMMessages(ctx context.Context, sessionID, fullCodePath, directoryName string, workspaceCtx *dto.GetWorkspaceContextResp, modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string, fallbackSystemPrompt string) ([]llms.Message, []llms.ToolDef, error) {
	msgs, tools, _, err := s.buildLLMMessagesWithPlan(ctx, sessionID, fullCodePath, directoryName, workspaceCtx, modeProvider, fallbackToolNames, fallbackSystemPrompt, 0)
	return msgs, tools, err
}

func (s *WorkspaceChatService) buildLLMMessagesWithPlan(ctx context.Context, sessionID, fullCodePath, directoryName string, workspaceCtx *dto.GetWorkspaceContextResp, modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string, fallbackSystemPrompt string, round int) ([]llms.Message, []llms.ToolDef, *dto.WorkspaceModelContextPlan, error) {
	list, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return nil, nil, nil, err
	}
	allMessages := append([]*model.AgentChatMessage(nil), list...)
	var session *model.AgentChatSession
	contextPolicy := ContextPolicyFull
	parentSessionID := ""
	modelContextAnchorMessageID := int64(0)
	if s.sessionRepo != nil {
		if gotSession, err := s.sessionRepo.GetBySessionID(sessionID); err == nil && gotSession != nil {
			session = gotSession
			contextPolicy = ContextPolicyFull
			parentSessionID = strings.TrimSpace(session.ParentSessionID)
		}
	}
	var toolNames []string
	var systemPromptFragment string
	if modeProvider != nil {
		systemPromptFragment = "" // 在 env 填好后由 provider.SystemPrompt(data) 产出
	} else {
		systemPromptFragment = fallbackSystemPrompt
	}
	toolNames = workspaceToolNamesForMode(modeProvider, fallbackToolNames)
	var toolsDesc []dto.ToolDef
	if s.toolReg != nil {
		toolsDesc, _ = s.toolReg.ListTools(ctx, toolNames)
	}
	llmTools := convertToLLMTools(toolsDesc)
	llmToolNames := toolNamesFromWorkspaceToolDefs(toolsDesc)

	// 环境数据与 env 块统一由 prompt 包构建
	now := time.Now()
	var envInput *prompt.WorkspaceEnvInput
	if workspaceCtx != nil {
		envInput = workspaceCtxToEnvInput(workspaceCtx)
		envInput.ScheduledTasksSection = buildWorkspaceScheduledTasksSection(ctx, envInput.FullCodePath)
	}
	catalog := prompt.LoadPromptDocCatalog(ctx)
	data := prompt.BuildWorkspaceEnvDataWithCatalog(envInput, directoryName, fullCodePath, now, catalog)
	envTemplate := prompt.LoadWorkspaceEnvTemplate(ctx)
	envBlock := prompt.BuildWorkspaceEnvBlockWithTemplate(envTemplate, data, workspaceCtx != nil, directoryName, fullCodePath)

	if modeProvider != nil {
		systemPromptFragment = modeProvider.SystemPrompt(data)
	}

	// 工作台固定系统提示词（不再依赖智能体）
	system := "你是智能工作台的助手。\n\n" + envBlock
	if systemPromptFragment != "" {
		system += "\n\n" + systemPromptFragment
	}
	if hint := workspaceFirstTurnDirectoryRAGHint(list, workspaceCtx); hint != "" {
		system += "\n\n" + hint
	}
	if dynamicTime := workspaceDynamicTimeHint(data); dynamicTime != "" {
		system += "\n\n" + dynamicTime
	}

	msgs := []llms.Message{{Role: "system", Content: system}}
	historyMessages, includedMessages, excludedUnsupported := buildWorkspaceLLMHistory(ctx, list)
	msgs = append(msgs, historyMessages...)
	plan := s.buildWorkspaceModelContextPlan(ctx, workspaceModelContextPlanInput{
		SessionID:                   sessionID,
		Round:                       round,
		FullCodePath:                fullCodePath,
		DirectoryName:               directoryName,
		WorkspaceCtx:                workspaceCtx,
		Session:                     session,
		ModeProvider:                modeProvider,
		ContextPolicy:               contextPolicy,
		ParentSessionID:             parentSessionID,
		ModelContextAnchorMessageID: modelContextAnchorMessageID,
		AllMessages:                 allMessages,
		ScopedMessages:              list,
		IncludedMessages:            includedMessages,
		ExcludedUnsupported:         excludedUnsupported,
		RequestedToolNames:          toolNames,
		LLMToolNames:                llmToolNames,
		LLMMessageCount:             len(msgs),
		LLMToolCount:                len(llmTools),
	})
	return msgs, llmTools, plan, nil
}

type workspaceLLMHistoryEntry struct {
	msg    llms.Message
	source *model.AgentChatMessage
}

func buildWorkspaceLLMHistory(ctx context.Context, messages []*model.AgentChatMessage) ([]llms.Message, []*model.AgentChatMessage, []*model.AgentChatMessage) {
	entries := make([]workspaceLLMHistoryEntry, 0, len(messages))
	includedMessages := make([]*model.AgentChatMessage, 0, len(messages))
	excludedUnsupported := make([]*model.AgentChatMessage, 0)

	for _, m := range messages {
		if m == nil {
			continue
		}
		switch m.Role {
		case RoleUser:
			userContent := userContentForLLMWithFileProfile(ctx, m.Content, m.Files)
			if strings.TrimSpace(userContent) == "" {
				excludedUnsupported = append(excludedUnsupported, m)
				continue
			}
			entries = append(entries, workspaceLLMHistoryEntry{
				msg:    llms.Message{Role: RoleUser, Content: userContent},
				source: m,
			})
		case RoleAssistant:
			msg := llms.Message{Role: RoleAssistant, Content: m.Content}
			if toolCalls, ok := storedToolCallsForLLM(m.ToolCalls); ok {
				msg.ToolCalls = toolCalls
			} else if strings.TrimSpace(msg.Content) == "" {
				excludedUnsupported = append(excludedUnsupported, m)
				continue
			}
			if strings.TrimSpace(msg.Content) == "" && len(msg.ToolCalls) == 0 {
				excludedUnsupported = append(excludedUnsupported, m)
				continue
			}
			entries = append(entries, workspaceLLMHistoryEntry{msg: msg, source: m})
		case RoleTool:
			if strings.TrimSpace(m.ToolCallID) == "" {
				excludedUnsupported = append(excludedUnsupported, m)
				continue
			}
			content := m.Content
			if strings.TrimSpace(content) == "" {
				content = "工具调用没有返回内容。"
			}
			entries = append(entries, workspaceLLMHistoryEntry{
				msg: llms.Message{
					Role:       RoleTool,
					ToolCallID: strings.TrimSpace(m.ToolCallID),
					Content:    content,
				},
				source: m,
			})
		default:
			excludedUnsupported = append(excludedUnsupported, m)
		}
	}

	historyMessages, sanitizedIncluded, sanitizedExcluded := sanitizeWorkspaceLLMToolSequence(entries)
	includedMessages = append(includedMessages, sanitizedIncluded...)
	excludedUnsupported = append(excludedUnsupported, sanitizedExcluded...)
	return historyMessages, includedMessages, excludedUnsupported
}

func storedToolCallsForLLM(raw *string) ([]llms.ToolCall, bool) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, true
	}
	var toolCalls []llms.ToolCall
	if err := json.Unmarshal([]byte(*raw), &toolCalls); err != nil {
		return nil, false
	}
	out := make([]llms.ToolCall, 0, len(toolCalls))
	seen := make(map[string]struct{}, len(toolCalls))
	for _, tc := range toolCalls {
		id := strings.TrimSpace(tc.ID)
		name := strings.TrimSpace(tc.Function.Name)
		if id == "" || name == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		tc.ID = id
		tc.Function.Name = name
		if strings.TrimSpace(tc.Type) == "" {
			tc.Type = "function"
		}
		out = append(out, tc)
	}
	return out, true
}

func sanitizeWorkspaceLLMToolSequence(entries []workspaceLLMHistoryEntry) ([]llms.Message, []*model.AgentChatMessage, []*model.AgentChatMessage) {
	out := make([]llms.Message, 0, len(entries))
	included := make([]*model.AgentChatMessage, 0, len(entries))
	excluded := make([]*model.AgentChatMessage, 0)
	pending := make(map[string]llms.ToolCall)
	pendingOrder := make([]string, 0)

	flushMissingToolResults := func() {
		if len(pendingOrder) == 0 {
			return
		}
		for _, id := range pendingOrder {
			tc, ok := pending[id]
			if !ok {
				continue
			}
			out = append(out, llms.Message{
				Role:       RoleTool,
				ToolCallID: id,
				Content:    missingWorkspaceToolResultContent(tc),
			})
			delete(pending, id)
		}
		pendingOrder = pendingOrder[:0]
	}

	for _, entry := range entries {
		role := strings.ToLower(strings.TrimSpace(entry.msg.Role))
		if role != RoleTool {
			flushMissingToolResults()
		}

		switch role {
		case RoleTool:
			id := strings.TrimSpace(entry.msg.ToolCallID)
			if _, ok := pending[id]; !ok {
				excluded = append(excluded, entry.source)
				continue
			}
			msg := entry.msg
			msg.ToolCallID = id
			if strings.TrimSpace(msg.Content) == "" {
				msg.Content = "工具调用没有返回内容。"
			}
			out = append(out, msg)
			included = append(included, entry.source)
			delete(pending, id)
			if len(pending) == 0 {
				pendingOrder = pendingOrder[:0]
			}
		case RoleAssistant:
			out = append(out, entry.msg)
			included = append(included, entry.source)
			if len(entry.msg.ToolCalls) == 0 {
				continue
			}
			for _, tc := range entry.msg.ToolCalls {
				id := strings.TrimSpace(tc.ID)
				if id == "" {
					continue
				}
				if _, ok := pending[id]; ok {
					continue
				}
				pending[id] = tc
				pendingOrder = append(pendingOrder, id)
			}
		default:
			out = append(out, entry.msg)
			included = append(included, entry.source)
		}
	}
	flushMissingToolResults()
	return out, included, excluded
}

func missingWorkspaceToolResultContent(tc llms.ToolCall) string {
	name := strings.TrimSpace(tc.Function.Name)
	if name == "" {
		name = "工具"
	}
	return fmt.Sprintf("%s 未返回工具结果：上一轮可能被用户中断或工具回传缺失。不要假设该工具已经成功执行；请结合后续用户消息继续。", name)
}

func workspaceDynamicTimeHint(data *prompt.WorkspaceEnvData) string {
	if data == nil || (data.CurrentDate == "" && data.CurrentTime == "") {
		return ""
	}
	var b strings.Builder
	b.WriteString("# 本轮动态信息\n")
	if data.CurrentDate != "" {
		b.WriteString("- 当前日期：")
		b.WriteString(data.CurrentDate)
		b.WriteByte('\n')
	}
	if data.CurrentTime != "" {
		b.WriteString("- 当前时间：")
		b.WriteString(data.CurrentTime)
	}
	return strings.TrimSpace(b.String())
}

func workspaceFirstTurnDirectoryRAGHint(messages []*model.AgentChatMessage, workspaceCtx *dto.GetWorkspaceContextResp) string {
	if workspaceCtx == nil {
		return ""
	}
	userMessages := 0
	for _, msg := range messages {
		if msg != nil && msg.Role == RoleUser {
			userMessages++
		}
	}
	if userMessages != 1 {
		return ""
	}
	return strings.Join([]string{
		"## 首轮目录理解要求",
		"",
		"- 这是当前会话或阶段在本目录的首轮判断；先使用上方目录名称、目录说明、子节点、函数描述和 Schema 摘要理解这个软件能提供什么服务。",
		"- 如果现有 Table/Form/Chart 能完成用户目标，优先按“使用当前软件”处理并进入 `app_operator`；不要先写 PRD 或进入开发。",
		"- 如果用户要把已有函数、已有业务操作或已有工作台目录定时执行，进入 `automation_operator`；如果要定时的能力还不存在，先走产品、开发或维护。",
		"- 只有用户明确要新增或改变软件能力，或现有运行函数无法满足目标时，才考虑产品、开发或维护角色。",
	}, "\n")
}
