package service

import (
	"encoding/json"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
)

const (
	workspaceContextReductionNone = iota
	workspaceContextReductionLight
	workspaceContextReductionStrict
	workspaceContextReductionEmergency
)

const (
	workspaceContextSoftTotalTokenLimit      = 32000
	workspaceContextDefaultOutputReserve     = 4096
	workspaceContextEstimateCharsPerToken    = 3
	workspaceContextPreflightReductionReason = "preflight_budget"
)

type workspaceLLMContextBuildOptions struct {
	ReductionLevel  int
	ReductionReason string
}

type workspaceLLMHistoryLimits struct {
	UserContentMaxRunes      int
	AssistantContentMaxRunes int
	ToolContentMaxRunes      int
	ArtifactReadMaxRunes     int
	ToolArgsMaxRunes         int
	MaxHistoryEntries        int
}

func normalizeWorkspaceContextReductionLevel(level int) int {
	if level < workspaceContextReductionNone {
		return workspaceContextReductionNone
	}
	if level > workspaceContextReductionEmergency {
		return workspaceContextReductionEmergency
	}
	return level
}

func workspaceLLMHistoryLimitsForLevel(level int) workspaceLLMHistoryLimits {
	switch normalizeWorkspaceContextReductionLevel(level) {
	case workspaceContextReductionLight:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      4000,
			AssistantContentMaxRunes: 3000,
			ToolContentMaxRunes:      1200,
			ArtifactReadMaxRunes:     6000,
			ToolArgsMaxRunes:         1000,
			MaxHistoryEntries:        48,
		}
	case workspaceContextReductionStrict:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      2500,
			AssistantContentMaxRunes: 1500,
			ToolContentMaxRunes:      700,
			ArtifactReadMaxRunes:     3500,
			ToolArgsMaxRunes:         600,
			MaxHistoryEntries:        18,
		}
	case workspaceContextReductionEmergency:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      1600,
			AssistantContentMaxRunes: 800,
			ToolContentMaxRunes:      400,
			ArtifactReadMaxRunes:     1600,
			ToolArgsMaxRunes:         300,
			MaxHistoryEntries:        8,
		}
	default:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      workspaceLLMHistoryUserContentMaxRunes,
			AssistantContentMaxRunes: workspaceLLMHistoryAssistantContentMaxRunes,
			ToolContentMaxRunes:      workspaceLLMHistoryToolContentMaxRunes,
			ArtifactReadMaxRunes:     10000,
			ToolArgsMaxRunes:         workspaceLLMHistoryToolArgsMaxRunes,
			MaxHistoryEntries:        0,
		}
	}
}

func workspaceEstimatedTokenCount(text string) int {
	runes := len([]rune(text))
	if runes <= 0 {
		return 0
	}
	tokens := runes / workspaceContextEstimateCharsPerToken
	if runes%workspaceContextEstimateCharsPerToken != 0 {
		tokens++
	}
	if tokens <= 0 {
		return 1
	}
	return tokens
}

func buildWorkspaceModelContextBudget(msgs []llms.Message, tools []llms.ToolDef, outputReserve int, reductionLevel int, reductionReason string) *dto.WorkspaceModelContextBudget {
	if outputReserve <= 0 {
		outputReserve = workspaceContextDefaultOutputReserve
	}
	inputTokens := estimateWorkspaceLLMMessageTokens(msgs)
	toolTokens := estimateWorkspaceLLMToolTokens(tools)
	total := inputTokens + toolTokens + outputReserve
	remaining := workspaceContextSoftTotalTokenLimit - total
	status := "ok"
	if remaining < 0 {
		remaining = 0
		status = "over_soft_limit"
	} else if remaining < workspaceContextDefaultOutputReserve {
		status = "near_soft_limit"
	}
	return &dto.WorkspaceModelContextBudget{
		ReducerLevel:         normalizeWorkspaceContextReductionLevel(reductionLevel),
		ReducerReason:        strings.TrimSpace(reductionReason),
		EstimatedInputTokens: inputTokens,
		EstimatedToolTokens:  toolTokens,
		OutputReserveTokens:  outputReserve,
		EstimatedTotalTokens: total,
		SoftLimitTokens:      workspaceContextSoftTotalTokenLimit,
		TokensUntilSoftLimit: remaining,
		Status:               status,
	}
}

func estimateWorkspaceLLMMessageTokens(msgs []llms.Message) int {
	total := 0
	for _, msg := range msgs {
		total += workspaceEstimatedTokenCount(msg.Role)
		total += workspaceEstimatedTokenCount(msg.Content)
		total += workspaceEstimatedTokenCount(msg.ToolCallID)
		if len(msg.ToolCalls) > 0 {
			if raw, err := json.Marshal(msg.ToolCalls); err == nil {
				total += workspaceEstimatedTokenCount(string(raw))
			}
		}
	}
	return total
}

func estimateWorkspaceLLMToolTokens(tools []llms.ToolDef) int {
	total := 0
	for _, tool := range tools {
		total += workspaceEstimatedTokenCount(tool.Type)
		total += workspaceEstimatedTokenCount(tool.Function.Name)
		total += workspaceEstimatedTokenCount(tool.Function.Description)
		if tool.Function.Parameters != nil {
			if raw, err := json.Marshal(tool.Function.Parameters); err == nil {
				total += workspaceEstimatedTokenCount(string(raw))
			}
		}
	}
	return total
}

func reduceWorkspaceOutputReserve(maxTokens int, reductionLevel int) int {
	if maxTokens <= 0 {
		return maxTokens
	}
	switch normalizeWorkspaceContextReductionLevel(reductionLevel) {
	case workspaceContextReductionStrict:
		if maxTokens > 2048 {
			return 2048
		}
	case workspaceContextReductionEmergency:
		if maxTokens > 1024 {
			return 1024
		}
	}
	return maxTokens
}
