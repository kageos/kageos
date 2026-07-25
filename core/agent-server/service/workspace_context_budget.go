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
	workspaceContextReductionCritical
)

const (
	workspaceContextDefaultOutputReserve     = 4096
	workspaceContextPreflightReductionReason = "preflight_budget"
	workspaceContextSafetyPercent            = 85
)

type workspaceLLMContextBuildOptions struct {
	ReductionLevel      int
	ReductionReason     string
	ContextWindowTokens int
	OutputReserveTokens int
	LLMConfigID         int64
}

type workspaceLLMHistoryLimits struct {
	UserContentMaxRunes      int
	AssistantContentMaxRunes int
	ToolContentMaxRunes      int
	ArtifactReadMaxRunes     int
	ToolArgsMaxRunes         int
}

func normalizeWorkspaceContextReductionLevel(level int) int {
	if level < workspaceContextReductionNone {
		return workspaceContextReductionNone
	}
	if level > workspaceContextReductionCritical {
		return workspaceContextReductionCritical
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
		}
	case workspaceContextReductionStrict:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      2500,
			AssistantContentMaxRunes: 1500,
			ToolContentMaxRunes:      700,
			ArtifactReadMaxRunes:     3500,
			ToolArgsMaxRunes:         600,
		}
	case workspaceContextReductionEmergency:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      1600,
			AssistantContentMaxRunes: 800,
			ToolContentMaxRunes:      400,
			ArtifactReadMaxRunes:     1600,
			ToolArgsMaxRunes:         300,
		}
	case workspaceContextReductionCritical:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      900,
			AssistantContentMaxRunes: 400,
			ToolContentMaxRunes:      200,
			ArtifactReadMaxRunes:     800,
			ToolArgsMaxRunes:         150,
		}
	default:
		return workspaceLLMHistoryLimits{
			UserContentMaxRunes:      workspaceLLMHistoryUserContentMaxRunes,
			AssistantContentMaxRunes: workspaceLLMHistoryAssistantContentMaxRunes,
			ToolContentMaxRunes:      workspaceLLMHistoryToolContentMaxRunes,
			ArtifactReadMaxRunes:     10000,
			ToolArgsMaxRunes:         workspaceLLMHistoryToolArgsMaxRunes,
		}
	}
}

func workspaceEstimatedTokenCount(text string) int {
	if text == "" {
		return 0
	}
	asciiChars := 0
	nonASCIIChars := 0
	for _, value := range text {
		if value <= 0x7f {
			asciiChars++
		} else {
			nonASCIIChars++
		}
	}
	// English/JSON usually averages about four characters per token, while CJK
	// and emoji are much denser. Counting each non-ASCII rune as one token is a
	// deliberately conservative preflight estimate; runtime overflow recovery
	// remains the final authority for provider-specific tokenizers.
	tokens := (asciiChars + 3) / 4
	tokens += nonASCIIChars
	if tokens <= 0 {
		return 1
	}
	return tokens
}

func workspaceContextSoftLimit(contextWindow int) int {
	if contextWindow <= 0 {
		contextWindow = DefaultLLMContextWindow
	}
	limit := contextWindow * workspaceContextSafetyPercent / 100
	if limit < 1024 {
		return 1024
	}
	return limit
}

func buildWorkspaceModelContextBudget(msgs []llms.Message, tools []llms.ToolDef, outputReserve int, contextWindow int, reductionLevel int, reductionReason string) *dto.WorkspaceModelContextBudget {
	if outputReserve <= 0 {
		outputReserve = workspaceContextDefaultOutputReserve
	}
	inputTokens := estimateWorkspaceLLMMessageTokens(msgs)
	toolTokens := estimateWorkspaceLLMToolTokens(tools)
	total := inputTokens + toolTokens + outputReserve
	softLimit := workspaceContextSoftLimit(contextWindow)
	remaining := softLimit - total
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
		SoftLimitTokens:      softLimit,
		TokensUntilSoftLimit: remaining,
		Status:               status,
	}
}

func estimateWorkspaceLLMMessageTokens(msgs []llms.Message) int {
	total := 0
	for _, msg := range msgs {
		total += 4 // role/message framing overhead
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
		total += 8 // function/tool framing overhead
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
	case workspaceContextReductionCritical:
		if maxTokens > 768 {
			return 768
		}
	}
	return maxTokens
}
