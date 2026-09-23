package service

import (
	"context"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"testing"
	"time"
)

func TestSystemMaintenanceAccessPauseAndManualRun(t *testing.T) {
	now := time.Date(2026, 9, 10, 1, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	admin := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{RequestUser: "system"})
	user := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{RequestUser: "alice"})
	req := scheduledsdk.CreateTaskRequest{Title: "maintenance", ExecutorKey: "platform.test", ResourceScope: "system", Schedule: scheduledsdk.Every(60), IdempotencyKey: "maintenance-v1", OverlapPolicy: scheduledsdk.OverlapPolicyForbid}
	if _, err := svc.CreateTask(user, req); err == nil {
		t.Fatal("ordinary user created system task")
	}
	task, err := svc.CreateTask(admin, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunNow(user, task.ID); err == nil {
		t.Fatal("ordinary user ran system task")
	}
	if err := svc.PauseTask(user, task.ID); err == nil {
		t.Fatal("ordinary user paused system task")
	}
	if _, err := svc.ListExecutions(user, task.ID, scheduledsdk.ListExecutionsRequest{}); err == nil {
		t.Fatal("ordinary user read system history")
	}
	list, err := svc.ListTasks(user, scheduledsdk.ListTasksRequest{})
	if err != nil || list.Total != 0 {
		t.Fatalf("system task leaked: %+v %v", list, err)
	}
	if err := svc.PauseTask(admin, task.ID); err != nil {
		t.Fatal(err)
	}
	again, err := svc.CreateTask(admin, req)
	if err != nil || again.ID != task.ID || again.Status != scheduledsdk.TaskStatusPaused {
		t.Fatalf("registration reset pause: %+v %v", again, err)
	}
	exec, err := svc.RunNow(admin, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exec.TriggerType != "manual" {
		t.Fatalf("manual trigger not recorded: %+v", exec)
	}
	forged := req
	forged.ExecutorKey = "test.executor"
	forged.ResourceScope = "function"
	if _, err := svc.CreateTask(user, forged); err == nil {
		t.Fatal("idempotency key exposed system task")
	}
}
