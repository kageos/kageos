package service

import (
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestWorkspacePromptCacheKeyIsStableAndScoped(t *testing.T) {
	plan := &dto.WorkspaceModelContextPlan{
		ModeCode: "dev",
		Role: dto.WorkspaceModelContextRole{
			ID: WorkspaceRoleAppOperator,
		},
		Execution: dto.WorkspaceModelContextExecution{
			FullCodePath: "/system/demo/app",
		},
		Tools: dto.WorkspaceModelContextTools{
			LLMTools: []string{"change_role", "read_doc", "run_form_submit"},
		},
		CachePlan: dto.WorkspaceModelContextCachePlan{
			StablePrefixItems: []string{"workspace_system_prompt", "workspace_env:/system/demo/app"},
		},
	}

	first := workspacePromptCacheKey(plan)
	second := workspacePromptCacheKey(plan)
	if first == "" || first != second {
		t.Fatalf("cache key should be non-empty and stable, got %q then %q", first, second)
	}
	if len(first) > 64 {
		t.Fatalf("cache key length = %d, want <= 64", len(first))
	}

	plan.Role.ID = WorkspaceRoleQAEngineer
	if got := workspacePromptCacheKey(plan); got != first {
		t.Fatalf("cache key should not change when role changes but mode tools and directory stay stable, got %q want %q", got, first)
	}
	plan.Tools.LLMTools = append(plan.Tools.LLMTools, "write_file")
	if got := workspacePromptCacheKey(plan); got == first {
		t.Fatalf("cache key should change when exposed LLM tool prefix changes")
	}
}

func TestWorkspaceLLMConfigSupportsPromptCacheOnlyForOpenAIHostedChat(t *testing.T) {
	if !workspaceLLMConfigSupportsPromptCache(&model.LLMConfig{Provider: model.LLMProviderOpenAI, Protocol: model.LLMProtocolOpenAIChatCompletions}) {
		t.Fatal("OpenAI hosted chat completions should support prompt cache key")
	}
	if workspaceLLMConfigSupportsPromptCache(&model.LLMConfig{Provider: model.LLMProviderOpenAI, Protocol: model.LLMProtocolOpenAIResponses}) {
		t.Fatal("non-chat protocol should not receive chat completions prompt cache params")
	}
	if workspaceLLMConfigSupportsPromptCache(&model.LLMConfig{Provider: model.LLMProviderOpenAI, Protocol: model.LLMProtocolOpenAIChatCompletions, APIBase: "https://proxy.example.com/v1"}) {
		t.Fatal("custom OpenAI-compatible bases should opt in through extra_config instead of receiving cache params by default")
	}
}

func TestWorkspaceDefaultPromptCacheRetentionForGPT55(t *testing.T) {
	cfg := &model.LLMConfig{Provider: model.LLMProviderOpenAI, Protocol: model.LLMProtocolOpenAIChatCompletions, Model: "gpt-5.5"}
	if got := workspaceDefaultPromptCacheRetention(cfg, ""); got != "24h" {
		t.Fatalf("gpt-5.5 retention = %q, want 24h", got)
	}
	if got := workspaceDefaultPromptCacheRetention(cfg, "gpt-5.5-pro"); got != "24h" {
		t.Fatalf("request model gpt-5.5-pro retention = %q, want 24h", got)
	}
	if got := workspaceDefaultPromptCacheRetention(&model.LLMConfig{Provider: model.LLMProviderOpenAI, Protocol: model.LLMProtocolOpenAIChatCompletions, Model: "gpt-4.1"}, ""); got != "" {
		t.Fatalf("gpt-4.1 retention = %q, want empty default", got)
	}
	if got := workspaceDefaultPromptCacheRetention(&model.LLMConfig{Provider: model.LLMProviderOpenAI, Protocol: model.LLMProtocolOpenAIChatCompletions, APIBase: "https://proxy.example.com/v1", Model: "gpt-5.5"}, ""); got != "" {
		t.Fatalf("custom OpenAI-compatible retention = %q, want empty default", got)
	}
}

func TestScheduledTaskSummaryExcludesVolatileRunState(t *testing.T) {
	nextRunAt := time.Date(2026, 7, 5, 10, 30, 0, 0, time.UTC)
	task := &scheduledsdk.Task{
		ID:              42,
		Title:           "日报巡检",
		Status:          scheduledsdk.TaskStatusPending,
		ResourceKey:     "/system/demo/report",
		Schedule:        scheduledsdk.Every(3600),
		NextRunAt:       &nextRunAt,
		RunCount:        99,
		LastExecutionID: 12345,
		CreatedBy:       "alice",
		Metadata:        map[string]string{"kind": "scheduled_agent_session"},
	}

	got := formatWorkspaceScheduledTaskSummary(task)
	for _, volatile := range []string{"下次执行", "已执行", "最近执行ID", "12345", nextRunAt.Format(time.RFC3339)} {
		if strings.Contains(got, volatile) {
			t.Fatalf("scheduled summary should not include volatile %q: %s", volatile, got)
		}
	}
	for _, stable := range []string{"id=42", "类型=数字员工", "标题=日报巡检", "资源=/system/demo/report", "计划=every 3600s", "创建人=alice"} {
		if !strings.Contains(got, stable) {
			t.Fatalf("scheduled summary missing stable %q: %s", stable, got)
		}
	}
}
