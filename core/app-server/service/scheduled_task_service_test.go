package service

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
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

func TestNormalizeScheduledTaskServiceOptionsSetsDefaultHeartbeatFile(t *testing.T) {
	opts := normalizeScheduledTaskServiceOptions(ScheduledTaskServiceOptions{})
	if opts.HeartbeatFile != defaultSchedulerHeartbeatFile {
		t.Fatalf("heartbeat file = %q, want %q", opts.HeartbeatFile, defaultSchedulerHeartbeatFile)
	}
	if opts.HeartbeatMaxAge != defaultSchedulerHeartbeatMaxAge {
		t.Fatalf("heartbeat max age = %s, want %s", opts.HeartbeatMaxAge, defaultSchedulerHeartbeatMaxAge)
	}
}

func TestScheduledTaskServiceWriteHeartbeat(t *testing.T) {
	heartbeatFile := filepath.Join(t.TempDir(), "scheduler.heartbeat")
	svc := &ScheduledTaskService{
		options: ScheduledTaskServiceOptions{
			HeartbeatFile: heartbeatFile,
		},
	}

	ts := time.Date(2026, 4, 19, 13, 0, 0, 0, time.UTC)
	svc.writeHeartbeat(context.Background(), ts)

	data, err := os.ReadFile(heartbeatFile)
	if err != nil {
		t.Fatalf("read heartbeat file: %v", err)
	}
	if got, want := string(data), strconv.FormatInt(ts.Unix(), 10); got != want {
		t.Fatalf("heartbeat content = %q, want %q", got, want)
	}
}

func TestScheduledTaskServiceHealthStatusUsesHeartbeatAge(t *testing.T) {
	heartbeatFile := filepath.Join(t.TempDir(), "scheduler.heartbeat")
	svc := &ScheduledTaskService{
		options: ScheduledTaskServiceOptions{
			HeartbeatFile:   heartbeatFile,
			HeartbeatMaxAge: 30 * time.Second,
		},
	}

	base := time.Date(2026, 4, 19, 13, 0, 0, 0, time.UTC)
	if healthy, age := svc.HealthStatus(base); healthy || age != 0 {
		t.Fatalf("health status without heartbeat = (%t, %s), want false, 0", healthy, age)
	}

	svc.writeHeartbeat(context.Background(), base)

	healthy, age := svc.HealthStatus(base.Add(10 * time.Second))
	if !healthy {
		t.Fatalf("health status after fresh heartbeat = false, want true")
	}
	if age != 10*time.Second {
		t.Fatalf("heartbeat age = %s, want %s", age, 10*time.Second)
	}

	healthy, age = svc.HealthStatus(base.Add(31 * time.Second))
	if healthy {
		t.Fatalf("health status after stale heartbeat = true, want false")
	}
	if age != 31*time.Second {
		t.Fatalf("stale heartbeat age = %s, want %s", age, 31*time.Second)
	}
}
