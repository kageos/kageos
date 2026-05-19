package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
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
	list, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return nil, nil, err
	}
	var toolNames []string
	var systemPromptFragment string
	if modeProvider != nil {
		systemPromptFragment = "" // 在 env 填好后由 provider.SystemPrompt(data) 产出
	} else {
		systemPromptFragment = fallbackSystemPrompt
	}
	toolNames = workspaceToolNamesForMode(modeProvider, fallbackToolNames)
	toolsDesc, _ := s.toolReg.ListTools(ctx, toolNames)
	llmTools := convertToLLMTools(toolsDesc)

	// 环境数据与 env 块统一由 prompt 包构建
	now := time.Now()
	var envInput *prompt.WorkspaceEnvInput
	if workspaceCtx != nil {
		envInput = workspaceCtxToEnvInput(workspaceCtx)
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

	msgs := []llms.Message{{Role: "system", Content: system}}
	for _, m := range list {
		if normalizeMessageContextUsage(m.ContextUsage) == MessageContextDisplayOnly {
			continue
		}
		switch m.Role {
		case RoleUser:
			userContent := userContentForLLM(m.Content, m.Files)
			msgs = append(msgs, llms.Message{Role: RoleUser, Content: userContent})
		case RoleAssistant:
			// 检查是否有 tool_calls（从 ToolCalls JSON 字段解析）
			msg := llms.Message{Role: RoleAssistant, Content: m.Content}
			if m.ToolCalls != nil && *m.ToolCalls != "" {
				// 解析 tool_calls JSON（如果存在）
				var toolCalls []llms.ToolCall
				if err := json.Unmarshal([]byte(*m.ToolCalls), &toolCalls); err == nil {
					msg.ToolCalls = toolCalls
				}
			}
			msgs = append(msgs, msg)
		case RoleTool:
			// 使用标准的 tool 角色消息
			msgs = append(msgs, llms.Message{
				Role:       RoleTool,
				ToolCallID: m.ToolCallID,
				Content:    m.Content,
			})
		}
	}
	return msgs, llmTools, nil
}
