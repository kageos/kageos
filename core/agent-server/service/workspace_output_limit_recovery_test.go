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
	if deps.contextReductionLevel != workspaceContextReductionLight {
		t.Fatalf("reduction level = %d, want light", deps.contextReductionLevel)
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
	if got := workspaceOutputTokenLimit(8196, workspaceContextReductionStrict, deps.outputLimitRecovery, DefaultLLMContextWindow); got != 8196 {
		t.Fatalf("max tokens = %d, want configured ceiling 8196 kept during recovery", got)
	}
}

func TestLowerWorkspaceRecoveryReasoningEffortOnlyChangesConfiguredHighEffort(t *testing.T) {
	cases := map[string]string{
		"xhigh":  "low",
		"high":   "low",
		"medium": "low",
		"low":    "low",
		"":       "",
	}
	for input, want := range cases {
		if got := lowerWorkspaceRecoveryReasoningEffort(input); got != want {
			t.Fatalf("lowerWorkspaceRecoveryReasoningEffort(%q) = %q, want %q", input, got, want)
		}
	}
}
