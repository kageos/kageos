package llms

import "testing"

func TestApplyDeepSeekThinkingDefaultsDisabled(t *testing.T) {
	t.Parallel()

	apiReq := map[string]interface{}{}
	applyDeepSeekThinking(apiReq, &ChatRequest{})

	thinking, ok := apiReq["thinking"].(map[string]string)
	if !ok {
		t.Fatalf("thinking = %#v, want map[string]string", apiReq["thinking"])
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type = %q, want disabled", thinking["type"])
	}
	if _, ok := apiReq["reasoning_effort"]; ok {
		t.Fatal("reasoning_effort should not be set when thinking is disabled")
	}
}

func TestApplyDeepSeekThinkingEnabled(t *testing.T) {
	t.Parallel()

	useThinking := true
	apiReq := map[string]interface{}{}
	applyDeepSeekThinking(apiReq, &ChatRequest{UseThinking: &useThinking})

	thinking, ok := apiReq["thinking"].(map[string]string)
	if !ok {
		t.Fatalf("thinking = %#v, want map[string]string", apiReq["thinking"])
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("thinking.type = %q, want enabled", thinking["type"])
	}
	if got := apiReq["reasoning_effort"]; got != "high" {
		t.Fatalf("reasoning_effort = %#v, want high", got)
	}
}
