package service

import (
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
)

const DefaultLLMContextWindow = 128000

// ResolveLLMContextWindow applies the configured precedence: manual value,
// provider metadata detected by Probe, then KageOS' conservative large default.
func ResolveLLMContextWindow(cfg *model.LLMConfig) (int, string) {
	if cfg != nil && cfg.ContextWindow > 0 {
		return cfg.ContextWindow, "manual"
	}
	if cfg != nil && cfg.DetectedContextWindow > 0 {
		return cfg.DetectedContextWindow, "detected"
	}
	return DefaultLLMContextWindow, "default"
}

func normalizeLLMContextWindow(cfg *model.LLMConfig) {
	if cfg == nil {
		return
	}
	if cfg.ContextWindow < 0 {
		cfg.ContextWindow = 0
	}
	if cfg.DetectedContextWindow < 0 {
		cfg.DetectedContextWindow = 0
	}
	if cfg.DetectedContextWindow == 0 {
		cfg.DetectedContextWindowSource = ""
	} else {
		cfg.DetectedContextWindowSource = strings.TrimSpace(cfg.DetectedContextWindowSource)
		if cfg.DetectedContextWindowSource == "" {
			cfg.DetectedContextWindowSource = "provider_metadata"
		}
	}
}
