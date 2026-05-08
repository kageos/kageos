package v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
)

func TestBuildScheduledTaskItem(t *testing.T) {
	runAt := time.Date(2026, 4, 15, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	nextRunAt := runAt.Add(time.Minute)
	createdAt := runAt.Add(-time.Hour)
	task := &model.ScheduledTask{
		ID:                3,
		TimerTaskID:       99,
		Name:              "提醒系统-检查并发送提醒",
		User:              "liubeiluo",
		App:               "work",
		FullCodePath:      "/liubeiluo/work/reminder/reminder_check.form",
		Action:            "execute",
		Method:            "POST",
		Payload:           json.RawMessage(`{"window_seconds":60}`),
		RequestUser:       "liubeiluo",
		RequestUserDept:   "/org/unassigned",
		CreatedBy:         "liubeiluo",
		ScheduleType:      "cron",
		RunAt:             runAt,
		NextRunAt:         &nextRunAt,
		CronExpr:          "*/1 * * * *",
		Timezone:          "Asia/Shanghai",
		Status:            "pending",
		NotifyUsers:       "alice,bob",
		NotifyDepartments: "/org/engineering",
		NotifyOn:          "failed",
		CreatedAt:         createdAt,
	}

	item := buildScheduledTaskItem(task)

	if item.Payload != `{"window_seconds":60}` {
		t.Fatalf("payload = %q, want JSON string", item.Payload)
	}
	if item.TimerTaskID != 99 {
		t.Fatalf("timer_task_id = %d, want 99", item.TimerTaskID)
	}
	if item.Action != "execute" {
		t.Fatalf("action = %q, want execute", item.Action)
	}
	if item.RunAt != runAt.Format(time.RFC3339) {
		t.Fatalf("run_at = %q, want %q", item.RunAt, runAt.Format(time.RFC3339))
	}
	if item.NextRunAt == nil || *item.NextRunAt != nextRunAt.Format(time.RFC3339) {
		t.Fatalf("next_run_at = %v, want %q", item.NextRunAt, nextRunAt.Format(time.RFC3339))
	}
	if len(item.NotifyUsers) != 2 || item.NotifyUsers[0] != "alice" || item.NotifyUsers[1] != "bob" {
		t.Fatalf("notify_users = %#v, want alice,bob", item.NotifyUsers)
	}
	if len(item.NotifyDepartments) != 1 || item.NotifyDepartments[0] != "/org/engineering" {
		t.Fatalf("notify_departments = %#v, want /org/engineering", item.NotifyDepartments)
	}
	if item.NotifyOn != "failed" {
		t.Fatalf("notify_on = %q, want failed", item.NotifyOn)
	}
}
