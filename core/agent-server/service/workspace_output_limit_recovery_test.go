package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/llms"
)

func TestWorkspaceOutputLimitRecoveryAddsInstructionAndKeepsTokenCeiling(t *testing.T) {
	deps := &workspaceStreamLoopDeps{}
	if !deps.RequestOutputLimitRecovery(context.Background(), "length") {
		t.Fatal("first output limit recovery should be accepted")
	}
	if deps.RequestOutputLimitRecovery(context.Background(), "length") {
		t.Fatal("second output limit recovery should be rejected")
	}
	if deps.contextReductionLevel != workspaceContextReductionNone {
		t.Fatalf("reduction level = %d, want none", deps.contextReductionLevel)
	}

	msgs := applyWorkspaceOutputLimitRecoveryInstruction([]llms.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: "完成任务"},
	})
	if len(msgs) != 2 || !strings.Contains(msgs[0].Content, "优先产出可见且完整的结果") {
		t.Fatalf("recovery messages = %#v, want instruction merged into system message", msgs)
	}
	if msgs[1].Content != "完成任务" {
		t.Fatalf("user message changed: %#v", msgs[1])
	}
	if got := workspaceOutputTokenLimit(8196, workspaceContextReductionStrict, deps.outputLimitRecovery, DefaultLLMContextWindow, 12000); got != 8196 {
		t.Fatalf("max tokens = %d, want configured ceiling 8196 kept during recovery", got)
	}
}

func TestWorkspaceOutputLimitUsesRemainingContextInsteadOfFixedQuarter(t *testing.T) {
	if got := workspaceOutputTokenLimit(64000, workspaceContextReductionNone, false, 200000, 20000); got != 64000 {
		t.Fatalf("max tokens = %d, want provider ceiling 64000", got)
	}
	if got := workspaceOutputTokenLimit(128000, workspaceContextReductionNone, false, 128000, 100000); got != 8800 {
		t.Fatalf("max tokens = %d, want remaining soft context 8800", got)
	}
}

func TestLowerWorkspaceRecoveryReasoningEffortOnlyChangesConfiguredHighEffort(t *testing.T) {
	cases := map[string]struct{ value, model, want string }{
		"xhigh":       {value: "xhigh", model: "gpt-5.6-sol", want: "low"},
		"high":        {value: "high", model: "gpt-5.6-sol", want: "low"},
		"medium":      {value: "medium", model: "gpt-5.6-sol", want: "low"},
		"low":         {value: "low", model: "gpt-5.6-sol", want: "low"},
		"auto-reason": {value: "", model: "gpt-5.6-sol", want: "low"},
		"non-reason":  {value: "", model: "gpt-4.1", want: ""},
	}
	for name, tc := range cases {
		if got := lowerWorkspaceRecoveryReasoningEffort(tc.value, tc.model); got != tc.want {
			t.Fatalf("%s: lowerWorkspaceRecoveryReasoningEffort(%q, %q) = %q, want %q", name, tc.value, tc.model, got, tc.want)
		}
	}
}
