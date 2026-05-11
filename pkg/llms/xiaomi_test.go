package llms

import "testing"

func TestNormalizeXiaomiBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty uses default chat completions endpoint",
			in:   "",
			want: xiaomiDefaultBaseURL,
		},
		{
			name: "official v1 base appends chat completions",
			in:   "https://api.xiaomimimo.com/v1",
			want: "https://api.xiaomimimo.com/v1/chat/completions",
		},
		{
			name: "token plan v1 base appends chat completions",
			in:   "https://token-plan-cn.xiaomimimo.com/v1",
			want: "https://token-plan-cn.xiaomimimo.com/v1/chat/completions",
		},
		{
			name: "token plan v1 base with trailing slash appends chat completions",
			in:   "https://token-plan-cn.xiaomimimo.com/v1/",
			want: "https://token-plan-cn.xiaomimimo.com/v1/chat/completions",
		},
		{
			name: "full chat completions endpoint is unchanged",
			in:   "https://token-plan-cn.xiaomimimo.com/v1/chat/completions",
			want: "https://token-plan-cn.xiaomimimo.com/v1/chat/completions",
		},
		{
			name: "host only gets versioned chat completions endpoint",
			in:   "https://api.xiaomimimo.com",
			want: "https://api.xiaomimimo.com/v1/chat/completions",
		},
		{
			name: "custom non-v1 endpoint is preserved",
			in:   "https://proxy.example.com/mimo/chat",
			want: "https://proxy.example.com/mimo/chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeXiaomiBaseURL(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeXiaomiBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewXiaomiClientWithOptionsNormalizesBaseURL(t *testing.T) {
	client := NewXiaomiClientWithOptions("test-key", DefaultClientOptions().WithBaseURL("https://token-plan-cn.xiaomimimo.com/v1"))
	if client.BaseURL != "https://token-plan-cn.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("client.BaseURL = %q", client.BaseURL)
	}
}

func TestNormalizeXiaomiModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		fallback string
		want     string
	}{
		{
			name:     "empty uses fallback",
			model:    "",
			fallback: "mimo-v2-flash",
			want:     "mimo-v2-flash",
		},
		{
			name:  "pro is lowercased",
			model: "MiMo-V2.5-Pro",
			want:  "mimo-v2.5-pro",
		},
		{
			name:  "openrouter provider prefix is stripped",
			model: "xiaomi/mimo-v2.5-pro",
			want:  "mimo-v2.5-pro",
		},
		{
			name:  "hyphenated version alias",
			model: "mimo-v2-5-pro",
			want:  "mimo-v2.5-pro",
		},
		{
			name:  "flash is lowercased",
			model: "MiMo-V2-Flash",
			want:  "mimo-v2-flash",
		},
		{
			name:  "non-pro v2.5 is lowercased without changing model",
			model: "MiMo-V2.5",
			want:  "mimo-v2.5",
		},
		{
			name:  "hyphenated non-pro version alias",
			model: "mimo-v2-5",
			want:  "mimo-v2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeXiaomiModel(tt.model, tt.fallback)
			if got != tt.want {
				t.Fatalf("normalizeXiaomiModel(%q, %q) = %q, want %q", tt.model, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestNewXiaomiClientWithOptionsNormalizesModel(t *testing.T) {
	client := NewXiaomiClientWithOptions("test-key", DefaultClientOptions().WithModel("xiaomi/mimo-v2.5-pro"))
	if client.Model != "mimo-v2.5-pro" {
		t.Fatalf("client.Model = %q", client.Model)
	}
}

func TestXiaomiSanitizeMessagesDropsEmptyAssistant(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "system"},
		{Role: "assistant"},
		{Role: "assistant", Content: "   "},
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "ok"},
	}

	got := xiaomiSanitizeMessages(msgs)
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3: %#v", len(got), got)
	}
	if got[0].Role != "system" || got[1].Role != "user" || got[2].Content != "ok" {
		t.Fatalf("unexpected sanitized messages: %#v", got)
	}
}

func TestXiaomiSanitizeMessagesKeepsAssistantToolCalls(t *testing.T) {
	msgs := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "read_file",
						Arguments: "",
					},
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "write_file",
						Arguments: `{"path":"main.go"}`,
					},
				},
			},
		},
	}

	got := xiaomiSanitizeMessages(msgs)
	if len(got) != 1 || len(got[0].ToolCalls) != 2 {
		t.Fatalf("unexpected sanitized messages: %#v", got)
	}
	if got[0].ToolCalls[0].Function.Arguments != "{}" {
		t.Fatalf("empty arguments should be normalized to {}, got %q", got[0].ToolCalls[0].Function.Arguments)
	}
	if got[0].ToolCalls[1].Function.Arguments != `{"path":"main.go"}` {
		t.Fatalf("valid arguments were changed: %q", got[0].ToolCalls[1].Function.Arguments)
	}
}
