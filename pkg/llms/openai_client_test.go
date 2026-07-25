package llms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openai "github.com/openai/openai-go/v3"
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

func TestOpenAIClientUsesReasoningControlsAndCompletionTokenLimit(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-5-test",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClientWithOptions("test-key", DefaultClientOptions().WithBaseURL(server.URL+"/v1").WithModel("gpt-5-test"))
	_, err := client.Chat(context.Background(), &ChatRequest{
		Messages:        []Message{{Role: "user", Content: "hi"}},
		MaxTokens:       16384,
		ReasoningEffort: "medium",
		Verbosity:       "low",
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if _, exists := payload["max_tokens"]; exists {
		t.Fatalf("reasoning request should not send deprecated max_tokens: %#v", payload)
	}
	if payload["max_completion_tokens"].(float64) != 16384 || payload["reasoning_effort"] != "medium" || payload["verbosity"] != "low" {
		t.Fatalf("unexpected reasoning payload: %#v", payload)
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

func TestValidateRequestRejectsOrphanToolMessage(t *testing.T) {
	err := validateRequest(context.Background(), "test-key", &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "hi"},
			{Role: "tool", ToolCallID: "call_1", Content: "{}"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "tool 结果没有紧邻") {
		t.Fatalf("validateRequest error = %v, want orphan tool error", err)
	}
}

func TestValidateRequestRejectsAssistantToolCallsWithoutResults(t *testing.T) {
	tc := ToolCall{ID: "call_1", Type: "function"}
	tc.Function.Name = "lookup"
	tc.Function.Arguments = "{}"

	err := validateRequest(context.Background(), "test-key", &ChatRequest{
		Messages: []Message{
			{Role: "assistant", ToolCalls: []ToolCall{tc}},
			{Role: "user", Content: "continue"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "缺少 tool 结果") {
		t.Fatalf("validateRequest error = %v, want missing tool result error", err)
	}
}

func TestSanitizeToolCallArgumentsFallsBackForInvalidJSON(t *testing.T) {
	got := sanitizeToolCallArguments(`{"content":"unterminated`)
	if got != "{}" {
		t.Fatalf("sanitizeToolCallArguments() = %q, want {}", got)
	}
}

func TestOpenAIClientChatStreamUsesSDKAndIncludesUsage(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"你\"},\"finish_reason\":null}],\"usage\":null}\n\n"))
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"好\"},\"finish_reason\":null}],\"usage\":null}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5,\"prompt_tokens_details\":{\"cached_tokens\":1}}}\n\n"))
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
	streamOptions, _ := payload["stream_options"].(map[string]interface{})
	if streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %#v, want include_usage true", payload["stream_options"])
	}
}

func TestOpenAIUsageCachedTokensFallsBackToRawJSON(t *testing.T) {
	var usage openai.CompletionUsage
	if err := json.Unmarshal([]byte(`{
		"prompt_tokens": 1200,
		"completion_tokens": 300,
		"total_tokens": 1500,
		"cache_read_input_tokens": 800
	}`), &usage); err != nil {
		t.Fatalf("unmarshal usage: %v", err)
	}

	cached, reported := openAIUsageCachedTokens(usage)
	if !reported || cached != 800 {
		t.Fatalf("cached=%d reported=%v, want 800 true", cached, reported)
	}
}

func TestOpenAIClientChatStreamEmitsReasoningContentAsInternalChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"内部思考\"},\"finish_reason\":null}],\"usage\":null}\n\n"))
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"最终答案\"},\"finish_reason\":\"stop\"}],\"usage\":null}\n\n"))
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

	var gotContent strings.Builder
	var gotReasoning strings.Builder
	for chunk := range stream {
		if chunk.Error != "" {
			t.Fatalf("unexpected stream error: %s", chunk.Error)
		}
		gotContent.WriteString(chunk.Content)
		gotReasoning.WriteString(chunk.ReasoningContent)
	}
	if gotContent.String() != "最终答案" {
		t.Fatalf("content = %q, want final answer only", gotContent.String())
	}
	if gotReasoning.String() != "内部思考" {
		t.Fatalf("reasoning content = %q, want internal reasoning", gotReasoning.String())
	}
}

func TestOpenAIClientChatStreamEmitsToolCallDeltasAndFinalAccumulation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{\\\"q\\\"\"}}]},\"finish_reason\":null}],\"usage\":null}\n\n"))
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\":\\\"x\\\"}\"}}]},\"finish_reason\":null}],\"usage\":null}\n\n"))
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}],\"usage\":null}\n\n"))
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

	var deltas []ToolCallDelta
	var final []ToolCall
	done := false
	for chunk := range stream {
		if chunk.Error != "" {
			t.Fatalf("unexpected stream error: %s", chunk.Error)
		}
		deltas = append(deltas, chunk.ToolCallDeltas...)
		if chunk.Done {
			done = true
			final = chunk.FinalToolCalls
		}
	}
	if len(deltas) != 2 {
		t.Fatalf("tool call chunks = %#v, want 2 chunks", deltas)
	}
	if deltas[0].ID != "call_1" || deltas[0].Function.Name != "lookup" || deltas[0].Function.Arguments != `{"q"` {
		t.Fatalf("first tool call chunk = %#v", deltas[0])
	}
	if deltas[1].Function.Arguments != `:"x"}` {
		t.Fatalf("second tool call arguments = %q", deltas[1].Function.Arguments)
	}
	if !done {
		t.Fatalf("stream did not emit done chunk")
	}
	if len(final) != 1 || final[0].ID != "call_1" || final[0].Function.Name != "lookup" || final[0].Function.Arguments != `{"q":"x"}` {
		t.Fatalf("final tool calls = %#v, want accumulated lookup call", final)
	}
}
