package llms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplyKimiTemperatureOmitsFixedSamplingModels(t *testing.T) {
	t.Parallel()

	models := []string{
		"kimi-k2.6",
		"kimi-k2.5",
		"kimi-k2-0711-preview",
		"kimi-latest",
		"kimi-thinking-preview",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			t.Parallel()

			apiReq := map[string]interface{}{}
			applyKimiTemperature(apiReq, model, 0.2)

			if _, ok := apiReq["temperature"]; ok {
				t.Fatalf("temperature should be omitted for fixed-sampling model %q", model)
			}
		})
	}
}

func TestApplyKimiTemperatureKeepsCustomSamplingModels(t *testing.T) {
	t.Parallel()

	apiReq := map[string]interface{}{}
	applyKimiTemperature(apiReq, "moonshot-v1-8k", 0.2)

	if got := apiReq["temperature"]; got != 0.2 {
		t.Fatalf("temperature = %#v, want 0.2", got)
	}
}

func TestKimiChatStreamOmitsTemperatureForK26(t *testing.T) {
	payloadCh := make(chan map[string]interface{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		payloadCh <- payload

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewKimiClientWithOptions(
		"test-key",
		DefaultClientOptions().WithBaseURL(server.URL).WithModel("kimi-k2.6"),
	)
	stream, err := client.ChatStream(context.Background(), &ChatRequest{
		Messages:    []Message{{Role: "user", Content: "hi"}},
		Temperature: 0.2,
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	for chunk := range stream {
		if chunk.Error != "" {
			t.Fatalf("stream returned error: %s", chunk.Error)
		}
		if chunk.Done {
			break
		}
	}

	payload := <-payloadCh
	if _, ok := payload["temperature"]; ok {
		t.Fatalf("temperature should be omitted for kimi-k2.6 request, payload=%#v", payload)
	}
}

func TestApplyKimiThinkingDisablesToolRequests(t *testing.T) {
	t.Parallel()

	useThinking := true
	apiReq := map[string]interface{}{}
	applyKimiThinking(apiReq, "kimi-k2.6", &ChatRequest{
		UseThinking: &useThinking,
		Tools: []ToolDef{
			{Type: "function"},
		},
	})

	thinking, ok := apiReq["thinking"].(map[string]string)
	if !ok {
		t.Fatalf("thinking = %#v, want map[string]string", apiReq["thinking"])
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type = %q, want disabled", thinking["type"])
	}
}

func TestApplyKimiThinkingRespectsNonToolRequests(t *testing.T) {
	t.Parallel()

	useThinking := true
	apiReq := map[string]interface{}{}
	applyKimiThinking(apiReq, "kimi-k2.6", &ChatRequest{UseThinking: &useThinking})

	thinking, ok := apiReq["thinking"].(map[string]string)
	if !ok {
		t.Fatalf("thinking = %#v, want map[string]string", apiReq["thinking"])
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("thinking.type = %q, want enabled", thinking["type"])
	}
}

func TestKimiChatStreamDisablesThinkingForToolRequest(t *testing.T) {
	payloadCh := make(chan map[string]interface{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		payloadCh <- payload

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewKimiClientWithOptions(
		"test-key",
		DefaultClientOptions().WithBaseURL(server.URL).WithModel("kimi-k2.6"),
	)
	stream, err := client.ChatStream(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		Tools: []ToolDef{
			{Type: "function"},
		},
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	for chunk := range stream {
		if chunk.Error != "" {
			t.Fatalf("stream returned error: %s", chunk.Error)
		}
		if chunk.Done {
			break
		}
	}

	payload := <-payloadCh
	thinking, ok := payload["thinking"].(map[string]interface{})
	if !ok {
		t.Fatalf("thinking = %#v, want object in payload %#v", payload["thinking"], payload)
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type = %#v, want disabled", thinking["type"])
	}
}

func TestKimiChatStreamUsageFromChoiceFinishChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\",\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewKimiClientWithOptions(
		"test-key",
		DefaultClientOptions().WithBaseURL(server.URL).WithModel("kimi-k2.6"),
	)
	stream, err := client.ChatStream(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	var final *StreamChunk
	for chunk := range stream {
		if chunk.Error != "" {
			t.Fatalf("stream returned error: %s", chunk.Error)
		}
		if chunk.Done {
			final = chunk
			break
		}
	}

	if final == nil || final.Usage == nil {
		t.Fatalf("final usage = %#v, want usage from finish chunk", final)
	}
	if final.Usage.TotalTokens != 3 {
		t.Fatalf("total_tokens = %d, want 3", final.Usage.TotalTokens)
	}
}

func TestKimiChatStreamRequiresDoneMarker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	}))
	defer server.Close()

	client := NewKimiClientWithOptions(
		"test-key",
		DefaultClientOptions().WithBaseURL(server.URL).WithModel("kimi-k2.6"),
	)
	stream, err := client.ChatStream(context.Background(), &ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	var gotErr string
	for chunk := range stream {
		if chunk.Error != "" {
			gotErr = chunk.Error
			break
		}
	}

	if !strings.Contains(gotErr, "[DONE]") {
		t.Fatalf("stream error = %q, want missing [DONE] error", gotErr)
	}
}
