package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
)

func TestResolveLLMContextWindowPrecedence(t *testing.T) {
	cfg := &model.LLMConfig{ContextWindow: 200000, DetectedContextWindow: 64000}
	if got, source := ResolveLLMContextWindow(cfg); got != 200000 || source != "manual" {
		t.Fatalf("manual resolution = %d/%s", got, source)
	}
	cfg.ContextWindow = 0
	if got, source := ResolveLLMContextWindow(cfg); got != 64000 || source != "detected" {
		t.Fatalf("detected resolution = %d/%s", got, source)
	}
	cfg.DetectedContextWindow = 0
	if got, source := ResolveLLMContextWindow(cfg); got != DefaultLLMContextWindow || source != "default" {
		t.Fatalf("default resolution = %d/%s", got, source)
	}
}

func TestGetLLMContextWindowMetadataSelectsRequestedModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"other","context_length":4096},{"id":"wanted","top_provider":{"context_length":131072}}]}`))
	}))
	defer server.Close()

	got, err := getLLMContextWindowMetadata(context.Background(), server.Client(), server.URL, nil, "wanted")
	if err != nil || got != 131072 {
		t.Fatalf("metadata result = %d, err=%v, want 131072", got, err)
	}
}

func TestLLMMetadataBaseURLHandlesAnthropicVersionedEndpoint(t *testing.T) {
	got := llmMetadataBaseURL("https://api.anthropic.com", "/v1/messages", "anthropic_messages")
	if got != "https://api.anthropic.com/v1" {
		t.Fatalf("base URL = %q", got)
	}
}

func TestLLMMetadataBaseURLDoesNotAppendChatDirectory(t *testing.T) {
	got := llmMetadataBaseURL("https://api.openai.com/v1", "/chat/completions", "openai_chat_completions")
	if got != "https://api.openai.com/v1" {
		t.Fatalf("base URL = %q", got)
	}
}
