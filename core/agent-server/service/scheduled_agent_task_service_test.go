package service

import (
	"context"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

func TestParseScheduledAgentRunAt(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	got, err := parseScheduledAgentRunAt("2026-04-24 09:30:00", loc)
	if err != nil {
		t.Fatalf("parseScheduledAgentRunAt returned error: %v", err)
	}
	want := time.Date(2026, 4, 24, 9, 30, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("run_at = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	if _, err := parseScheduledAgentRunAt("", loc); err == nil {
		t.Fatal("empty run_at should be rejected")
	}
}

func TestComputeScheduledAgentNextRun(t *testing.T) {
	scheduledAt := time.Date(2026, 4, 24, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	cronTask := &model.ScheduledAgentTask{
		ScheduleType: ScheduledAgentScheduleCron,
		CronExpr:     "*/5 * * * *",
		Status:       model.ScheduledAgentTaskStatusPending,
	}
	status, next, stateErr := computeScheduledAgentNextRun(cronTask, scheduledAt, true)
	if status != model.ScheduledAgentTaskStatusPending || next == nil || !next.Equal(scheduledAt.Add(5*time.Minute)) || stateErr != "" {
		t.Fatalf("cron next state = (%s, %v, %q), want pending, %s, empty error", status, next, stateErr, scheduledAt.Add(5*time.Minute).Format(time.RFC3339))
	}

	oneShot := &model.ScheduledAgentTask{ScheduleType: ScheduledAgentScheduleAtime}
	status, next, _ = computeScheduledAgentNextRun(oneShot, scheduledAt, true)
	if status != model.ScheduledAgentTaskStatusDone || next != nil {
		t.Fatalf("atime success next state = (%s, %v), want done, nil", status, next)
	}
}

func TestScheduledAgentRefs(t *testing.T) {
	if got := scheduledAgentTaskRef(42); got != "scheduled_agent_task:42" {
		t.Fatalf("task ref = %q", got)
	}
	if got := scheduledAgentExecutionRef(7); got != "scheduled_agent_execution:7" {
		t.Fatalf("execution ref = %q", got)
	}
}

func TestResolveScheduledAgentSource(t *testing.T) {
	sourceType, sourceRef := resolveScheduledAgentSource(context.Background())
	if sourceType != ScheduledAgentSourceType || sourceRef != "" {
		t.Fatalf("default source = (%q, %q), want (%q, empty)", sourceType, sourceRef, ScheduledAgentSourceType)
	}

	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		SourceType: "function",
		SourceRef:  "function:/system/reminder/send",
	})
	sourceType, sourceRef = resolveScheduledAgentSource(ctx)
	if sourceType != "function" || sourceRef != "function:/system/reminder/send" {
		t.Fatalf("context source = (%q, %q)", sourceType, sourceRef)
	}
}
