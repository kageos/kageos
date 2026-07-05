package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
)

const workspacePromptCacheKeyVersion = "workspace-prompt-cache.v2"

func workspaceLLMConfigSupportsPromptCache(llmConfig *model.LLMConfig) bool {
	if llmConfig == nil {
		return false
	}
	provider := strings.ToLower(strings.TrimSpace(llmConfig.Provider))
	if provider != "" && provider != model.LLMProviderOpenAI {
		return false
	}
	protocol := strings.ToLower(strings.TrimSpace(llmConfig.Protocol))
	if protocol != "" && protocol != model.LLMProtocolOpenAIChatCompletions {
		return false
	}
	apiBase := strings.ToLower(strings.TrimSpace(llmConfig.APIBase))
	return apiBase == "" || strings.Contains(apiBase, "api.openai.com")
}

func workspacePromptCacheKey(plan *dto.WorkspaceModelContextPlan) string {
	if plan == nil {
		return ""
	}
	parts := []string{
		workspacePromptCacheKeyVersion,
		strings.TrimSpace(plan.ModeCode),
		strings.TrimSpace(plan.Execution.FullCodePath),
		strings.Join(plan.Tools.LLMTools, ","),
		strings.Join(plan.CachePlan.StablePrefixItems, ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "kws:" + hex.EncodeToString(sum[:16])
}

func workspaceDefaultPromptCacheRetention(llmConfig *model.LLMConfig, requestModel string) string {
	if !workspaceLLMConfigSupportsPromptCache(llmConfig) {
		return ""
	}
	modelName := strings.ToLower(strings.TrimSpace(requestModel))
	if modelName == "" && llmConfig != nil {
		modelName = strings.ToLower(strings.TrimSpace(llmConfig.Model))
	}
	if modelName == "gpt-5.5" || strings.HasPrefix(modelName, "gpt-5.5-") {
		return "24h"
	}
	return ""
}
