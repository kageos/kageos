package service

import (
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
)

func TestLLMProbeCandidatesPrefersResponsesForDefaultOpenAI(t *testing.T) {
	candidates, err := llmProbeCandidates(dto.LLMProbeReq{
		Provider: model.LLMProviderOpenAI,
		Protocol: model.LLMProtocolOpenAIChatCompletions,
	})
	if err != nil {
		t.Fatalf("llmProbeCandidates() error = %v", err)
	}
	if len(candidates) < 2 {
		t.Fatalf("candidates = %#v, want at least responses and chat", candidates)
	}
	if candidates[0].Protocol != model.LLMProtocolOpenAIResponses {
		t.Fatalf("first protocol = %q, want openai_responses", candidates[0].Protocol)
	}
	if candidates[1].Protocol != model.LLMProtocolOpenAIChatCompletions {
		t.Fatalf("second protocol = %q, want openai_chat_completions", candidates[1].Protocol)
	}
}

func TestLLMProbeCandidatesInfersResponsesFromEndpoint(t *testing.T) {
	candidates, err := llmProbeCandidates(dto.LLMProbeReq{
		Provider:     model.LLMProviderOpenAI,
		Protocol:     model.LLMProtocolOpenAIChatCompletions,
		APIBase:      "https://devcloud.chat",
		EndpointPath: "/responses",
	})
	if err != nil {
		t.Fatalf("llmProbeCandidates() error = %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("candidates empty, want openai_responses")
	}
	if candidates[0].Provider != model.LLMProviderOpenAI || candidates[0].Protocol != model.LLMProtocolOpenAIResponses {
		t.Fatalf("first candidate = %#v, want openai/openai_responses", candidates[0])
	}
}
