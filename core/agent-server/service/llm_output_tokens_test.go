package service

import (
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
)

func TestResolveLLMMaxOutputTokensPrecedence(t *testing.T) {
	cfg := &model.LLMConfig{
		Model:                   "gpt-5.6-sol",
		MaxTokens:               16000,
		DetectedMaxOutputTokens: 128000,
	}
	if got, source := ResolveLLMMaxOutputTokens(cfg); got != 16000 || source != "manual" {
		t.Fatalf("manual result = %d/%s, want 16000/manual", got, source)
	}
	cfg.MaxTokens = 0
	if got, source := ResolveLLMMaxOutputTokens(cfg); got != 128000 || source != "detected" {
		t.Fatalf("detected result = %d/%s, want 128000/detected", got, source)
	}
	cfg.DetectedMaxOutputTokens = 0
	if got, source := ResolveLLMMaxOutputTokens(cfg); got != 128000 || source != "model_registry" {
		t.Fatalf("registry result = %d/%s, want 128000/model_registry", got, source)
	}
	cfg.Model = "custom-model"
	if got, source := ResolveLLMMaxOutputTokens(cfg); got != DefaultLLMMaxOutputTokens || source != "default" {
		t.Fatalf("default result = %d/%s, want %d/default", got, source, DefaultLLMMaxOutputTokens)
	}
}

func TestFindMaxOutputTokenValue(t *testing.T) {
	payload := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"id": "other", "max_output_tokens": float64(4096)},
			map[string]interface{}{"id": "target", "output_token_limit": float64(65536)},
		},
	}
	matched := findLLMModelMetadata(payload, "target")
	if got := findMaxOutputTokenValue(matched); got != 65536 {
		t.Fatalf("max output tokens = %d, want 65536", got)
	}
}

func TestNormalizeLLMOutputTokensKeepsAutoMode(t *testing.T) {
	cfg := &model.LLMConfig{MaxTokens: 0, DetectedMaxOutputTokens: 64000}
	normalizeLLMOutputTokens(cfg)
	if cfg.MaxTokens != 0 || cfg.DetectedMaxOutputTokenSource != "provider_metadata" {
		t.Fatalf("normalized config = %#v", cfg)
	}
}
