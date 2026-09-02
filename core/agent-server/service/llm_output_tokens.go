package service

import (
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
)

const DefaultLLMMaxOutputTokens = 32768

// ResolveLLMMaxOutputTokens applies the configured precedence: a positive
// manual value, provider/model metadata detected by Probe, a built-in model
// specification, then kageos' conservative automatic fallback.
func ResolveLLMMaxOutputTokens(cfg *model.LLMConfig) (int, string) {
	if cfg != nil && cfg.MaxTokens > 0 {
		return cfg.MaxTokens, "manual"
	}
	if cfg != nil && cfg.DetectedMaxOutputTokens > 0 {
		return cfg.DetectedMaxOutputTokens, "detected"
	}
	if cfg != nil {
		if value := knownModelSpecForName(cfg.Model).MaxOutputTokens; value > 0 {
			return value, "model_registry"
		}
	}
	return DefaultLLMMaxOutputTokens, "default"
}

func normalizeLLMOutputTokens(cfg *model.LLMConfig) {
	if cfg == nil {
		return
	}
	if cfg.MaxTokens < 0 {
		cfg.MaxTokens = 0
	}
	if cfg.DetectedMaxOutputTokens < 0 {
		cfg.DetectedMaxOutputTokens = 0
	}
	if cfg.DetectedMaxOutputTokens == 0 {
		cfg.DetectedMaxOutputTokenSource = ""
	} else {
		cfg.DetectedMaxOutputTokenSource = strings.TrimSpace(cfg.DetectedMaxOutputTokenSource)
		if cfg.DetectedMaxOutputTokenSource == "" {
			cfg.DetectedMaxOutputTokenSource = "provider_metadata"
		}
	}
}

type knownModelSpec struct {
	ContextWindow   int
	MaxOutputTokens int
}

func knownModelSpecForName(modelName string) knownModelSpec {
	name := strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case strings.HasPrefix(name, "gpt-5.6"):
		return knownModelSpec{ContextWindow: 1050000, MaxOutputTokens: 128000}
	case strings.HasPrefix(name, "claude-fable-5"),
		strings.HasPrefix(name, "claude-opus-5"),
		strings.HasPrefix(name, "claude-sonnet-5"):
		return knownModelSpec{ContextWindow: 1000000, MaxOutputTokens: 128000}
	case strings.HasPrefix(name, "claude-haiku-4-5"):
		return knownModelSpec{ContextWindow: 200000, MaxOutputTokens: 64000}
	default:
		return knownModelSpec{}
	}
}
