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
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/llms"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

const (
	workspaceLLMHistoryUserContentMaxRunes      = 6000
	workspaceLLMHistoryAssistantContentMaxRunes = 6000
	workspaceLLMHistoryToolContentMaxRunes      = 2500
	workspaceLLMHistoryToolArgsMaxRunes         = 1800
)

// prepareLLMRequest 工作台只认 LLM：llmConfigID > 0 用该配置，否则用默认
func (s *WorkspaceChatService) prepareLLMRequest(ctx context.Context, llmConfigID int64, msgs []llms.Message, tools []llms.ToolDef) (*model.LLMConfig, llms.LLMClient, *llms.ChatRequest, error) {
	llmConfig, err := s.resolveWorkspaceLLMConfig(ctx, llmConfigID)
	if err != nil {
		return nil, nil, nil, err
	}
	apiKey, err := openLLMAPIKey(s.apiKeyVault, s.apiKeyVaultErr, llmConfig.APIKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解密 LLM API Key 失败: %w", err)
	}
	if llmConfig.APIKey != "" && !isSealedLLMAPIKey(s.apiKeyVault, llmConfig.APIKey) {
		if sealed, sealErr := sealLLMAPIKey(s.apiKeyVault, s.apiKeyVaultErr, llmConfig.APIKey); sealErr == nil {
			_ = s.llmRepo.UpdateAPIKey(llmConfig.ID, sealed)
		}
	}

	var extraConfig map[string]interface{}
	if llmConfig.ExtraConfig != nil && *llmConfig.ExtraConfig != "" {
		_ = json.Unmarshal([]byte(*llmConfig.ExtraConfig), &extraConfig)
	}
	headers, err := llmHeadersFromJSON(llmConfig.Headers)
	if err != nil {
		return nil, nil, nil, err
	}
	clientConfig := llms.ClientConfig{
		Provider:     llmConfig.Provider,
		Protocol:     llmConfig.Protocol,
		APIKey:       apiKey,
		BaseURL:      llmConfig.APIBase,
		EndpointPath: llmConfig.EndpointPath,
		APIVersion:   llmConfig.APIVersion,
		AuthScheme:   llmConfig.AuthScheme,
		Headers:      headers,
		Model:        llmConfig.Model,
	}
	originalProvider := llmConfig.Provider
	originalProtocol := llmConfig.Protocol
	clientConfig.Provider, clientConfig.Protocol = llms.InferProviderProtocol(clientConfig.Provider, clientConfig.Protocol, clientConfig.BaseURL, clientConfig.EndpointPath)
	llmConfig.Provider = clientConfig.Provider
	llmConfig.Protocol = clientConfig.Protocol
	if llmConfig.ID > 0 && (originalProvider != clientConfig.Provider || originalProtocol != clientConfig.Protocol) {
		if err := s.llmRepo.UpdateProviderProtocol(llmConfig.ID, clientConfig.Provider, clientConfig.Protocol); err != nil {
			logger.Warnf(ctx, "[WorkspaceChatStream] 同步 LLM 有效协议失败 - id=%d provider=%s protocol=%s err=%v",
				llmConfig.ID, clientConfig.Provider, clientConfig.Protocol, err)
		}
	}
	logger.Infof(ctx, "[WorkspaceChatStream] 使用 LLM 配置 - id=%d name=%s provider=%s protocol=%s model=%s api_base=%s endpoint_path=%s",
		llmConfig.ID, llmConfig.Name, clientConfig.Provider, clientConfig.Protocol, llmConfig.Model, llmConfig.APIBase, llmConfig.EndpointPath)
	if llmConfig.Timeout > 0 {
		clientConfig.Timeout = time.Duration(llmConfig.Timeout) * time.Second
	}
	if maxRetries, ok := extraConfig["max_retries"].(float64); ok && maxRetries >= 0 {
		clientConfig.MaxRetries = int(maxRetries)
	}
	if userAgent, ok := extraConfig["user_agent"].(string); ok {
		clientConfig.UserAgent = userAgent
	}
	client, err := llms.NewClientFromConfig(clientConfig)
	if err != nil {
		return nil, nil, nil, err
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
	if reasoningEffort, ok := extraConfig["reasoning_effort"].(string); ok {
		chatReq.ReasoningEffort = strings.TrimSpace(reasoningEffort)
	}
	if verbosity, ok := extraConfig["verbosity"].(string); ok {
		chatReq.Verbosity = strings.TrimSpace(verbosity)
	}
	if promptCacheKey, ok := extraConfig["prompt_cache_key"].(string); ok {
		chatReq.PromptCacheKey = strings.TrimSpace(promptCacheKey)
	}
	if promptCacheRetention, ok := extraConfig["prompt_cache_retention"].(string); ok {
		chatReq.PromptCacheRetention = strings.TrimSpace(promptCacheRetention)
	}
	return llmConfig, client, chatReq, nil
}

func workspaceConfiguredMaxTokens(cfg *model.LLMConfig) int {
	if cfg == nil {
		return workspaceContextDefaultOutputReserve
	}
	if cfg.ExtraConfig != nil && strings.TrimSpace(*cfg.ExtraConfig) != "" {
		var extra map[string]interface{}
		if json.Unmarshal([]byte(*cfg.ExtraConfig), &extra) == nil {
			if value, ok := extra["max_tokens"].(float64); ok && value > 0 {
				return int(value)
			}
		}
	}
	if cfg.MaxTokens > 0 {
		return cfg.MaxTokens
	}
	return workspaceContextDefaultOutputReserve
}

func (s *WorkspaceChatService) resolveWorkspaceLLMConfig(ctx context.Context, llmConfigID int64) (*model.LLMConfig, error) {
	var llmConfig *model.LLMConfig
	var err error
	if llmConfigID > 0 {
		llmConfig, err = s.llmRepo.GetByID(llmConfigID)
		if err != nil {
			return nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
		}
	}
	if llmConfig == nil {
		llmConfig, err = s.llmRepo.GetDefault()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("未设置默认 LLM，请在 LLM 管理中配置")
			}
			return nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
		}
	}
	if !canViewLLMConfig(llmConfig, contextx.GetRequestUser(ctx)) {
		return nil, fmt.Errorf("无权限使用该 LLM 配置")
	}
	return llmConfig, nil
}

func llmHeadersFromJSON(raw *string) (map[string]string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(*raw), &obj); err != nil {
		return nil, fmt.Errorf("headers 不是有效 JSON: %w", err)
	}
	headers := make(map[string]string, len(obj))
	for key, value := range obj {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch v := value.(type) {
		case string:
			headers[key] = v
		default:
			b, _ := json.Marshal(v)
			headers[key] = string(b)
		}
	}
	return headers, nil
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

func (s *WorkspaceChatService) buildLLMMessagesWithPlan(ctx context.Context, sessionID, fullCodePath, directoryName string, workspaceCtx *dto.GetWorkspaceContextResp, modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string, fallbackSystemPrompt string, round int, currentMessageID ...int64) ([]llms.Message, []llms.ToolDef, *dto.WorkspaceModelContextPlan, error) {
	return s.buildLLMMessagesWithPlanAndOptions(ctx, sessionID, fullCodePath, directoryName, workspaceCtx, modeProvider, fallbackToolNames, fallbackSystemPrompt, round, workspaceLLMContextBuildOptions{}, currentMessageID...)
}

func (s *WorkspaceChatService) buildLLMMessagesWithPlanAndOptions(ctx context.Context, sessionID, fullCodePath, directoryName string, workspaceCtx *dto.GetWorkspaceContextResp, modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string, fallbackSystemPrompt string, round int, options workspaceLLMContextBuildOptions, currentMessageID ...int64) ([]llms.Message, []llms.ToolDef, *dto.WorkspaceModelContextPlan, error) {
	list, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return nil, nil, nil, err
	}
	currentTurnMessageID := int64(0)
	if len(currentMessageID) > 0 {
		currentTurnMessageID = currentMessageID[0]
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
	roleIDForToolFilter, _ := workspaceModelContextRole(session, allMessages)
	modeToolNames := workspaceToolNamesForMode(modeProvider, fallbackToolNames)
	roleAllowedToolNames := workspaceToolNamesForRole(modeToolNames, roleIDForToolFilter)
	toolNames = workspaceToolNamesForLLM(roleAllowedToolNames)
	if len(toolNames) == 0 {
		toolNames = workspaceToolNamesForLLM(modeToolNames)
	}
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
		envInput.DirectoryRunbookSection = buildWorkspaceRunbookSection(ctx, envInput.FullCodePath)
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
	systemIdentity := "你是 Kageos 智能工作台的执行助手。身份、公司、协议、Hub 和企业版口径按 /system/prompt/platform-introduction 读取；使用方式和理念按 /system/prompt/platform-usage-and-philosophy 读取；能力边界按 /system/prompt/platform-capability-boundaries 读取。不要自称普通聊天机器人，也不要把当前核心说成 OSI 开源项目。"
	systemParts := []string{systemIdentity}
	if systemPromptFragment != "" {
		systemParts = append(systemParts, systemPromptFragment)
	}
	systemParts = append(systemParts, envBlock)
	if hint := workspaceFirstTurnDirectoryRAGHint(list, workspaceCtx); hint != "" {
		systemParts = append(systemParts, hint)
	}
	if dynamicTime := workspaceDynamicTimeHint(data); dynamicTime != "" {
		systemParts = append(systemParts, dynamicTime)
	}
	if workingState := buildWorkspaceWorkingStateBlock(list, session, currentTurnMessageID); workingState != "" {
		systemParts = append(systemParts, workingState)
	}
	system := strings.Join(systemParts, "\n\n")

	reductionLevel := normalizeWorkspaceContextReductionLevel(options.ReductionLevel)
	reductionReason := strings.TrimSpace(options.ReductionReason)
	checkpoint := s.latestWorkspaceContextCheckpoint(sessionID)
	list, checkpointCoveredMessages := workspaceMessagesAfterCheckpoint(allMessages, checkpoint)
	checkpointAttempted := false
	// A provider-reported overflow is authoritative even when our tokenizer
	// estimate is still below the soft limit. Advance the reversible checkpoint
	// before applying any content-level emergency reduction.
	if reductionLevel > workspaceContextReductionNone {
		checkpointAttempted = true
		if next, changed := s.ensureWorkspaceContextCheckpoint(ctx, sessionID, allMessages, currentTurnMessageID, options, checkpoint); changed {
			checkpoint = next
			list, checkpointCoveredMessages = workspaceMessagesAfterCheckpoint(allMessages, checkpoint)
		}
	}
	for {
		msgs := []llms.Message{{Role: "system", Content: system}}
		if checkpointMessage, ok := workspaceContextCheckpointMessage(checkpoint); ok {
			msgs = append(msgs, checkpointMessage)
		}
		historyMessages, includedMessages, excludedUnsupported, excludedDisplayOnly, excludedByReduction := buildWorkspaceLLMHistoryWithOptions(ctx, list, currentTurnMessageID, workspaceLLMContextBuildOptions{
			ReductionLevel:  reductionLevel,
			ReductionReason: reductionReason,
		})
		msgs = append(msgs, historyMessages...)
		outputReserve := options.OutputReserveTokens
		if outputReserve <= 0 {
			outputReserve = workspaceContextDefaultOutputReserve
		}
		budget := buildWorkspaceModelContextBudget(msgs, llmTools, outputReserve, options.ContextWindowTokens, reductionLevel, reductionReason)
		if budget.Status == "over_soft_limit" && !checkpointAttempted {
			checkpointAttempted = true
			if next, changed := s.ensureWorkspaceContextCheckpoint(ctx, sessionID, allMessages, currentTurnMessageID, options, checkpoint); changed {
				checkpoint = next
				list, checkpointCoveredMessages = workspaceMessagesAfterCheckpoint(allMessages, checkpoint)
				continue
			}
		}
		if budget.Status == "over_soft_limit" && reductionLevel < workspaceContextReductionCritical {
			reductionLevel++
			if reductionReason == "" {
				reductionReason = workspaceContextPreflightReductionReason
			}
			continue
		}
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
			ScopedMessages:              allMessages,
			IncludedMessages:            includedMessages,
			ExcludedUnsupported:         excludedUnsupported,
			ExcludedDisplayOnly:         excludedDisplayOnly,
			ExcludedByReduction:         excludedByReduction,
			ExcludedByCheckpoint:        checkpointCoveredMessages,
			Checkpoint:                  checkpoint,
			RequestedToolNames:          toolNames,
			LLMToolNames:                llmToolNames,
			RoleAllowedToolNames:        roleAllowedToolNames,
			LLMMessageCount:             len(msgs),
			LLMToolCount:                len(llmTools),
			Budget:                      budget,
		})
		return msgs, llmTools, plan, nil
	}
}

type workspaceLLMHistoryEntry struct {
	msg    llms.Message
	source *model.AgentChatMessage
}

func buildWorkspaceLLMHistory(ctx context.Context, messages []*model.AgentChatMessage, currentTurnMessageID int64) ([]llms.Message, []*model.AgentChatMessage, []*model.AgentChatMessage, []*model.AgentChatMessage) {
	historyMessages, includedMessages, excludedUnsupported, excludedDisplayOnly, _ := buildWorkspaceLLMHistoryWithOptions(ctx, messages, currentTurnMessageID, workspaceLLMContextBuildOptions{})
	return historyMessages, includedMessages, excludedUnsupported, excludedDisplayOnly
}

func buildWorkspaceLLMHistoryWithOptions(ctx context.Context, messages []*model.AgentChatMessage, currentTurnMessageID int64, options workspaceLLMContextBuildOptions) ([]llms.Message, []*model.AgentChatMessage, []*model.AgentChatMessage, []*model.AgentChatMessage, []*model.AgentChatMessage) {
	limits := workspaceLLMHistoryLimitsForLevel(options.ReductionLevel)
	entries := make([]workspaceLLMHistoryEntry, 0, len(messages))
	includedMessages := make([]*model.AgentChatMessage, 0, len(messages))
	excludedUnsupported := make([]*model.AgentChatMessage, 0)
	excludedDisplayOnly := make([]*model.AgentChatMessage, 0)
	excludedByReduction := make([]*model.AgentChatMessage, 0)
	skipExpiredCurrentTurnRun := false

	for _, m := range messages {
		if m == nil {
			continue
		}
		usage := normalizeMessageContextUsage(m.ContextUsage)
		if usage == MessageContextDisplayOnly {
			excludedDisplayOnly = append(excludedDisplayOnly, m)
			continue
		}
		if m.Role == RoleUser {
			skipExpiredCurrentTurnRun = false
			if usage == MessageContextCurrentTurn && (currentTurnMessageID == 0 || m.ID != currentTurnMessageID) {
				skipExpiredCurrentTurnRun = true
				excludedUnsupported = append(excludedUnsupported, m)
				continue
			}
		} else if skipExpiredCurrentTurnRun {
			excludedUnsupported = append(excludedUnsupported, m)
			continue
		}
		switch m.Role {
		case RoleUser:
			userContent := ""
			if refContent, ok := workspaceMessageArtifactReferenceContent(m); ok {
				userContent = refContent
			} else {
				userContent = userContentForLLMWithFileProfile(ctx, m.Content, m.Files)
			}
			if strings.TrimSpace(userContent) == "" {
				excludedUnsupported = append(excludedUnsupported, m)
				continue
			}
			entries = append(entries, workspaceLLMHistoryEntry{
				msg:    llms.Message{Role: RoleUser, Content: compactWorkspaceLLMHistoryMessageContent(userContent, limits.UserContentMaxRunes, m.ID)},
				source: m,
			})
		case RoleAssistant:
			content := m.Content
			if refContent, ok := workspaceMessageArtifactReferenceContent(m); ok {
				content = refContent
			}
			msg := llms.Message{Role: RoleAssistant, Content: compactWorkspaceLLMHistoryMessageContent(content, limits.AssistantContentMaxRunes, m.ID)}
			if toolCalls, ok := storedToolCallsForLLMWithSource(m.ToolCalls, limits.ToolArgsMaxRunes, m.ID); ok {
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
			maxContentRunes := limits.ToolContentMaxRunes
			if refContent, ok := workspaceMessageArtifactReferenceContent(m); ok {
				content = refContent
			} else if workspaceMessageIsArtifactReadResult(m) && limits.ArtifactReadMaxRunes > maxContentRunes {
				maxContentRunes = limits.ArtifactReadMaxRunes
			}
			if strings.TrimSpace(content) == "" {
				content = "工具调用没有返回内容。"
			}
			entries = append(entries, workspaceLLMHistoryEntry{
				msg: llms.Message{
					Role:       RoleTool,
					ToolCallID: strings.TrimSpace(m.ToolCallID),
					Content:    compactWorkspaceLLMHistoryMessageContent(content, maxContentRunes, m.ID),
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
	return historyMessages, includedMessages, excludedUnsupported, excludedDisplayOnly, excludedByReduction
}

func storedToolCallsForLLM(raw *string) ([]llms.ToolCall, bool) {
	return storedToolCallsForLLMWithLimit(raw, workspaceLLMHistoryToolArgsMaxRunes)
}

func storedToolCallsForLLMWithLimit(raw *string, maxRunes int) ([]llms.ToolCall, bool) {
	return storedToolCallsForLLMWithSource(raw, maxRunes, 0)
}

func storedToolCallsForLLMWithSource(raw *string, maxRunes int, sourceMessageID int64) ([]llms.ToolCall, bool) {
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
		tc.Function.Arguments = compactWorkspaceToolCallArguments(tc.Function.Arguments, maxRunes, sourceMessageID)
		if strings.TrimSpace(tc.Type) == "" {
			tc.Type = "function"
		}
		out = append(out, tc)
	}
	return out, true
}

func compactWorkspaceToolCallArguments(raw string, maxRunes int, sourceMessageID int64) string {
	if workspaceRuneLen(raw) <= maxRunes {
		return raw
	}
	payload := map[string]interface{}{
		"_kageos_arguments_truncated": true,
		"original_chars":              workspaceRuneLen(raw),
		"preview":                     compactWorkspaceLLMHistoryContent(raw, maxRunes),
	}
	if sourceMessageID > 0 {
		payload["source_message_id"] = sourceMessageID
		payload["recovery"] = "Call read_session_messages with this message_id to read the exact stored tool_calls."
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"_kageos_arguments_truncated":true,"original_chars":%d}`, workspaceRuneLen(raw))
	}
	return string(b)
}

func compactWorkspaceLLMHistoryMessageContent(content string, maxRunes int, sourceMessageID int64) string {
	if workspaceRuneLen(content) <= maxRunes {
		return content
	}
	compacted := compactWorkspaceLLMHistoryContent(content, maxRunes)
	if sourceMessageID <= 0 {
		return compacted
	}
	return fmt.Sprintf("<truncated_message_ref message_id=\"%d\">\n%s\n精确原文仍保存在当前会话；需要完整细节时调用 read_session_messages(message_ids=[%d])，并按 next_offset_chars 分页。\n</truncated_message_ref>", sourceMessageID, compacted, sourceMessageID)
}

func compactWorkspaceLLMHistoryContent(content string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	head := maxRunes * 2 / 3
	tail := maxRunes - head
	if tail < 0 {
		tail = 0
	}
	return string(runes[:head]) +
		fmt.Sprintf("\n\n[历史内容已截断：原始 %d 字符，仅保留前 %d 字符和后 %d 字符；如需完整内容请重新读取对应资源。]\n\n", len(runes), head, tail) +
		string(runes[len(runes)-tail:])
}

func workspaceRuneLen(value string) int {
	return len([]rune(value))
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
		"- 如果用户只是要简单处理一个文件、附件或临时数据，例如转换、压缩、清洗、加水印、解析附件或整理临时结果，也优先进入 `app_operator` 用 `run_python` 完成；复杂、专项或多步骤文件处理才进入 `data_operator`。",
		"- 如果用户要把已有函数、已有业务操作或已有工作台目录定时执行，进入 `automation_operator`；如果要定时的能力还不存在，先走产品、开发或维护。",
		"- 只有用户明确要新增或改变软件能力，或现有运行函数无法满足目标时，才考虑产品、开发或维护角色。",
	}, "\n")
}
