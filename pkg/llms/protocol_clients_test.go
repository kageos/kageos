package llms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIResponsesClientChatStreamsTextAndUsage(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %q, want /v1/responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"KAGEOS_\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":3,\"output_tokens\":2,\"total_tokens\":5}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAIResponsesClientWithOptions("test-key", DefaultClientOptions().WithBaseURL(server.URL+"/v1").WithModel("gpt-test"))
	resp, err := client.Chat(context.Background(), &ChatRequest{
		Messages:        []Message{{Role: "user", Content: "hi"}},
		MaxTokens:       16,
		ReasoningEffort: "medium",
		Verbosity:       "low",
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.Content != "KAGEOS_OK" {
		t.Fatalf("content = %q, want KAGEOS_OK", resp.Content)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 5 {
		t.Fatalf("usage = %#v, want total 5", resp.Usage)
	}
	if payload["model"] != "gpt-test" || payload["stream"] != true || payload["store"] != false {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload["max_output_tokens"].(float64) != 16 {
		t.Fatalf("max_output_tokens = %#v, want 16", payload["max_output_tokens"])
	}
	reasoning, _ := payload["reasoning"].(map[string]interface{})
	textConfig, _ := payload["text"].(map[string]interface{})
	if reasoning["effort"] != "medium" || textConfig["verbosity"] != "low" {
		t.Fatalf("reasoning/text config = %#v/%#v, want medium/low", reasoning, textConfig)
	}
}

func TestOpenAIResponsesClientMapsIncompleteMaxOutputTokensToLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.reasoning_summary_text.delta\",\"delta\":\"内部思考\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.incomplete\",\"response\":{\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"},\"usage\":{\"input_tokens\":3,\"output_tokens\":16,\"total_tokens\":19}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAIResponsesClientWithOptions("test-key", DefaultClientOptions().WithBaseURL(server.URL+"/v1").WithModel("gpt-test"))
	stream, err := client.ChatStream(context.Background(), &ChatRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 16,
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	finishReason := ""
	for chunk := range stream {
		if chunk.FinishReason != "" {
			finishReason = chunk.FinishReason
		}
	}
	if finishReason != "max_output_tokens" {
		t.Fatalf("finish reason = %q, want max_output_tokens", finishReason)
	}
}

func TestOpenAIResponsesClientRetriesRootResponsesWhenAPIV1Returns404(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/responses":
			http.NotFound(w, r)
		case "/responses":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"KAGEOS_OK\"}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewOpenAIResponsesClientWithOptions("test-key", DefaultClientOptions().
		WithBaseURL(server.URL+"/api/v1").
		WithEndpointPath("/responses").
		WithModel("gpt-test"))
	resp, err := client.Chat(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.Content != "KAGEOS_OK" {
		t.Fatalf("content = %q, want KAGEOS_OK", resp.Content)
	}
	if len(paths) != 2 || paths[0] != "/api/v1/responses" || paths[1] != "/responses" {
		t.Fatalf("paths = %#v, want /api/v1/responses then /responses", paths)
	}
}

func TestInferProviderProtocolUsesResponsesEndpoint(t *testing.T) {
	provider, protocol := InferProviderProtocol(ProviderOpenAI, ProtocolOpenAIChatCompletions, "https://example.com/v1", "/responses")
	if provider != ProviderOpenAI || protocol != ProtocolOpenAIResponses {
		t.Fatalf("provider/protocol = %s/%s, want openai/openai_responses", provider, protocol)
	}
}

func TestBuildEndpointDoesNotDuplicateFullResponsesEndpoint(t *testing.T) {
	got := buildEndpoint("https://example.com/v1/responses", "", ProtocolOpenAIResponses)
	if got != "https://example.com/v1/responses" {
		t.Fatalf("endpoint = %q, want full responses endpoint unchanged", got)
	}
}

func TestAnthropicMessagesClientChatStreamsTextAndUsage(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %q, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("anthropic-version = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":3}}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"KAGEOS_OK\"}}\n\n"))
		_, _ = w.Write([]byte("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n"))
		_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	client := NewAnthropicMessagesClientWithOptions("test-key", DefaultClientOptions().WithBaseURL(server.URL).WithModel("claude-test"))
	resp, err := client.Chat(context.Background(), &ChatRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 16,
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.Content != "KAGEOS_OK" {
		t.Fatalf("content = %q, want KAGEOS_OK", resp.Content)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 5 {
		t.Fatalf("usage = %#v, want total 5", resp.Usage)
	}
	if payload["model"] != "claude-test" || payload["stream"] != true {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload["max_tokens"].(float64) != 16 {
		t.Fatalf("max_tokens = %#v, want 16", payload["max_tokens"])
	}
}
