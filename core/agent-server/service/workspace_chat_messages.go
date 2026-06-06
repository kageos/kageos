package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
	"github.com/kageos/kageos/pkg/logger"
)

const filesInstruction = "以上 <files> 标签中的 JSON 为本轮用户上传的文件引用。files 字段的新标准是 bucket/object_key 字符串；提交表单或表格时，直接把 refs 字符串填到对应 files 字段；调用 run_python 时把 refs 字符串填到工具顶层 input_files 参数，不要猜 URL。工具结果里的 files/output_files 会由工作台渲染成文件组件，最终回复不要手写下载文件名、Markdown 下载链接或伪 URL。"

// userContentForStorage 入库用：只保留用户文字到 Content，文件引用字符串单独到 Files。
// 返回 (content, filesRefs)；无文件时 filesRefs 为 nil。
func userContentForStorage(files string, userContent string) (content string, filesRefs *string) {
	demand := strings.TrimSpace(userContent)
	files = strings.TrimSpace(files)
	if files == "" {
		return demand, nil
	}
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	return demand, &files
}

// userContentForLLM 从库中取出的 user 消息：若有 Files 则拼出完整内容（<files>+说明+content）供 LLM 使用。
func userContentForLLM(content string, filesRefs *string) string {
	if filesRefs == nil || *filesRefs == "" {
		return content
	}
	demand := strings.TrimSpace(content)
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	return "<files>\n" + filesPayloadForLLM(*filesRefs) + "\n</files>\n\n" + filesInstruction + "\n\n" + demand
}

func filesPayloadForLLM(files string) string {
	files = strings.TrimSpace(files)
	if files == "" {
		return "{}"
	}
	payload := map[string]interface{}{
		"refs": files,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"refs":%q}`, files)
	}
	return string(raw)
}

func normalizeMessageContextUsage(usage string) string {
	switch strings.TrimSpace(usage) {
	case MessageContextDisplayOnly:
		return MessageContextDisplayOnly
	case MessageContextArtifact:
		return MessageContextArtifact
	default:
		return MessageContextInclude
	}
}

func llmUsageInfoFromUsage(usage *llms.Usage) *dto.LLMUsageInfo {
	if usage == nil {
		return nil
	}
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 && usage.CachedTokens == 0 {
		return nil
	}
	return &dto.LLMUsageInfo{
		PromptTokens:         usage.PromptTokens,
		CompletionTokens:     usage.CompletionTokens,
		TotalTokens:          usage.TotalTokens,
		CachedTokens:         usage.CachedTokens,
		CachedTokensReported: usage.CachedTokensReported,
	}
}

func marshalLLMUsageField(ctx context.Context, usage *llms.Usage) *string {
	info := llmUsageInfoFromUsage(usage)
	if info == nil {
		return nil
	}
	b, err := json.Marshal(info)
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 保存 LLM usage 失败: %v", err)
		return nil
	}
	out := string(b)
	return &out
}

// sanitizeContentForMySQLUtf8 去掉 4 字节 UTF-8 字符（BMP 外），避免 MySQL utf8 列报 Error 1366；表为 utf8mb4 时无需此过滤。
func sanitizeContentForMySQLUtf8(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x10000 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func marshalToolResultField(ctx context.Context, toolCallID, fieldName string, value any) *string {
	if value == nil {
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 保存 tool %s 失败 ToolCallID=%s: %v", fieldName, toolCallID, err)
		return nil
	}
	out := string(b)
	return &out
}

// saveToolMessage 保存一条 role=tool 的消息。失败时返回 error，调用方应中止下一轮以免 400 insufficient tool messages。
func (s *WorkspaceChatService) saveToolMessage(ctx context.Context, sessionID string, toolCallID, toolName, status string, result ToolResult, user string) error {
	toolMsg := &model.AgentChatMessage{
		SessionID:      sessionID,
		Role:           RoleTool,
		Content:        sanitizeContentForMySQLUtf8(result.Content),
		ToolCallID:     toolCallID,
		ToolStatus:     status,
		ResultData:     marshalToolResultField(ctx, toolCallID, "result_data", result.Data),
		ResultMetadata: marshalToolResultField(ctx, toolCallID, "result_metadata", result.Metadata),
		User:           user,
	}
	toolMsg.CreatedBy = user
	toolMsg.UpdatedBy = user
	if err := s.messageRepo.Create(toolMsg); err != nil {
		return err
	}
	return nil
}

// saveAssistantMessageWithToolCalls 保存 assistant 消息（包含 tool_calls）
func (s *WorkspaceChatService) saveAssistantMessageWithToolCalls(
	ctx context.Context,
	sessionID string,
	content string,
	allToolCalls []llms.ToolCall,
	user string,
	llmMeta messageLLMMetadata,
	usage *llms.Usage,
) error {
	toolCallsJSON, _ := json.Marshal(allToolCalls)
	toolCallsStr := string(toolCallsJSON)
	asstMsg := &model.AgentChatMessage{
		SessionID:     sessionID,
		Role:          RoleAssistant,
		Content:       content,
		ToolCalls:     &toolCallsStr,
		LLMConfigID:   llmMeta.ConfigID,
		LLMConfigName: llmMeta.ConfigName,
		LLMProvider:   llmMeta.Provider,
		LLMModel:      llmMeta.Model,
		LLMUsage:      marshalLLMUsageField(ctx, usage),
		User:          user,
	}
	asstMsg.CreatedBy = user
	asstMsg.UpdatedBy = user
	if err := s.messageRepo.Create(asstMsg); err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 保存 assistant 消息失败: %v", err)
		return err
	}
	return nil
}

// saveAssistantMessage 保存普通 assistant 消息
func (s *WorkspaceChatService) saveAssistantMessage(
	ctx context.Context,
	sessionID string,
	content string,
	user string,
	llmMeta messageLLMMetadata,
	usage *llms.Usage,
) error {
	asstMsg := &model.AgentChatMessage{
		SessionID:     sessionID,
		Role:          RoleAssistant,
		Content:       content,
		LLMConfigID:   llmMeta.ConfigID,
		LLMConfigName: llmMeta.ConfigName,
		LLMProvider:   llmMeta.Provider,
		LLMModel:      llmMeta.Model,
		LLMUsage:      marshalLLMUsageField(ctx, usage),
		User:          user,
	}
	asstMsg.CreatedBy = user
	asstMsg.UpdatedBy = user
	return s.messageRepo.Create(asstMsg)
}
