package service

import (
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
)

func TestResolveScheduledTaskRunAt(t *testing.T) {
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	cronRunAt, err := resolveScheduledTaskRunAt("cron", "", now)
	if err != nil {
		t.Fatalf("cron resolve run_at returned error: %v", err)
	}
	if !cronRunAt.Equal(now) {
		t.Fatalf("cron run_at = %s, want %s", cronRunAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}

	everyRunAt, err := resolveScheduledTaskRunAt("every", "", now)
	if err != nil {
		t.Fatalf("every resolve run_at returned error: %v", err)
	}
	if !everyRunAt.Equal(now) {
		t.Fatalf("every run_at = %s, want %s", everyRunAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}

	if _, err := resolveScheduledTaskRunAt("atime", "", now); err == nil {
		t.Fatalf("atime resolve run_at should reject empty run_at")
	}
}

func TestNextCronRunAfterUsesNextMatch(t *testing.T) {
	base := time.Date(2026, 4, 15, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	want := time.Date(2026, 4, 15, 10, 1, 0, 0, time.FixedZone("CST", 8*60*60))

	next, err := nextCronRunAfter("*/1 * * * *", base)
	if err != nil {
		t.Fatalf("nextCronRunAfter returned error: %v", err)
	}
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestComputeTaskNextStateUsesScheduledTimeForCatchUp(t *testing.T) {
	scheduledAt := time.Date(2026, 4, 15, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	cronTask := &model.ScheduledTask{
		ScheduleType: "cron",
		CronExpr:     "*/1 * * * *",
	}
	cronStatus, cronNext, cronErr := computeTaskNextState(cronTask, scheduledAt, true)
	if cronStatus != "pending" || cronNext == nil || !cronNext.Equal(scheduledAt.Add(time.Minute)) || cronErr != "" {
		t.Fatalf("cron next state = (%s, %v, %q), want pending, %s, empty error", cronStatus, cronNext, cronErr, scheduledAt.Add(time.Minute).Format(time.RFC3339))
	}

	everyTask := &model.ScheduledTask{
		ScheduleType:    "every",
		IntervalSeconds: 60,
		ErrorMessage:    "temporary failure",
	}
	everyStatus, everyNext, everyErr := computeTaskNextState(everyTask, scheduledAt, false)
	if everyStatus != "pending" || everyNext == nil || !everyNext.Equal(scheduledAt.Add(time.Minute)) || everyErr != "temporary failure" {
		t.Fatalf("every next state = (%s, %v, %q), want pending, %s, original error", everyStatus, everyNext, everyErr, scheduledAt.Add(time.Minute).Format(time.RFC3339))
	}
}
