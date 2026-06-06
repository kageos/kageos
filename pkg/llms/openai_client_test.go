package llms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIClientChatUsesSDKAndCustomBaseURL(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-test",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
				"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5,"prompt_tokens_details":{"cached_tokens":2}}
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClientWithOptions("test-key", DefaultClientOptions().WithBaseURL(server.URL+"/v1").WithModel("gpt-test"))
	resp, err := client.Chat(context.Background(), &ChatRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 7,
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("content = %q, want ok", resp.Content)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 5 || resp.Usage.CachedTokens != 2 || !resp.Usage.CachedTokensReported {
		t.Fatalf("usage = %#v, want total 5 and reported cached 2", resp.Usage)
	}
	if payload["model"] != "gpt-test" || payload["max_tokens"].(float64) != 7 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestValidateRequestAllowsAssistantToolCallsWithoutContent(t *testing.T) {
	tc := ToolCall{ID: "call_1", Type: "function"}
	tc.Function.Name = "lookup"
	tc.Function.Arguments = "{}"

	err := validateRequest(context.Background(), "test-key", &ChatRequest{
		Messages: []Message{
			{Role: "assistant", ToolCalls: []ToolCall{tc}},
			{Role: "tool", ToolCallID: "call_1", Content: "{}"},
		},
	})
	if err != nil {
		t.Fatalf("validateRequest returned error: %v", err)
	}
}

func TestSanitizeToolCallArgumentsFallsBackForInvalidJSON(t *testing.T) {
	got := sanitizeToolCallArguments(`{"content":"unterminated`)
	if got != "{}" {
		t.Fatalf("sanitizeToolCallArguments() = %q, want {}", got)
	}
}

func TestOpenAIClientChatStreamToleratesEmptyDataAndBadTrailingJSON(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data:\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5,\"prompt_tokens_details\":{\"cached_tokens\":1}}}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":["))
	}))
	defer server.Close()

	client := NewOpenAIClientWithOptions("test-key", DefaultClientOptions().WithBaseURL(server.URL+"/v1").WithModel("gpt-test"))
	stream, err := client.ChatStream(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	var got strings.Builder
	var usage *Usage
	for chunk := range stream {
		if chunk.Error != "" {
			t.Fatalf("unexpected stream error: %s", chunk.Error)
		}
		got.WriteString(chunk.Content)
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}
	if got.String() != "你好" {
		t.Fatalf("content = %q, want 你好", got.String())
	}
	if usage == nil || usage.TotalTokens != 5 || usage.CachedTokens != 1 || !usage.CachedTokensReported {
		t.Fatalf("usage = %#v, want total 5 and reported cached 1", usage)
	}
	if payload["stream"] != true {
		t.Fatalf("stream = %#v, want true", payload["stream"])
	}
}

func TestOpenAIClientChatStreamParsesToolCallDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{\\\"q\\\"\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\":\\\"x\\\"}\"}}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAIClientWithOptions("test-key", DefaultClientOptions().WithBaseURL(server.URL+"/v1").WithModel("gpt-test"))
	stream, err := client.ChatStream(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	var calls []ToolCall
	for chunk := range stream {
		if chunk.Error != "" {
			t.Fatalf("unexpected stream error: %s", chunk.Error)
		}
		calls = append(calls, chunk.ToolCalls...)
	}
	if len(calls) != 2 {
		t.Fatalf("tool call chunks = %#v, want 2 chunks", calls)
	}
	if calls[0].ID != "call_1" || calls[0].Function.Name != "lookup" || calls[0].Function.Arguments != `{"q"` {
		t.Fatalf("first tool call chunk = %#v", calls[0])
	}
	if calls[1].Function.Arguments != `:"x"}` {
		t.Fatalf("second tool call arguments = %q", calls[1].Function.Arguments)
	}
}
