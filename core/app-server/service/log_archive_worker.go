package service

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/nats-io/nats.go"
)

func NewLogArchiveWorker(natsConn *nats.Conn, archive *LogArchiveService) (*scheduledsdk.Worker, error) {
	if natsConn == nil {
		return nil, fmt.Errorf("log archive worker requires nats connection")
	}
	if archive == nil {
		return nil, fmt.Errorf("log archive worker requires archive service")
	}
	client := scheduledsdk.NewClient(scheduledsdk.Options{Adapter: scheduledsdk.NewNATSAdapter(natsConn, scheduledsdk.NATSAdapterOptions{})})
	return scheduledsdk.NewWorker(scheduledsdk.WorkerOptions{
		Client: client, NATSConn: natsConn, ExecutorKey: LogArchiveExecutorKey, Handler: archive.RunScheduled,
		Concurrency: 1,
		OnError:     func(ctx context.Context, err error) { logger.Warnf(ctx, "[LogArchiveWorker] %v", err) },
	})
}

func (s *LogArchiveService) ReconcileSchedule(ctx context.Context) error {
	if !s.config.Enabled {
		return nil
	}
	client := newAppScheduleClient()
	managedCtx := contextx.WithClientSource(ctx, scheduledTaskSourceManifest)
	req := scheduledsdk.CreateTaskRequest{
		Title: "操作日志离线归档", Description: "每日将超过保留期的操作日志压缩归档到对象存储", Category: "platform_maintenance",
		Tags: []string{"platform", "maintenance", "log_archive"}, IdempotencyKey: "platform-operate-log-archive-v1",
		ExecutorKey:   LogArchiveExecutorKey,
		Metadata:      map[string]string{"kind": "log_archive", "managed_by": "app_manifest", "origin": scheduledTaskOriginManifest, "default_enabled": "true"},
		Schedule:      scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: s.config.CronExpr, Timezone: s.config.Timezone},
		OverlapPolicy: scheduledsdk.OverlapPolicyForbid, MaxParallelism: 1,
		SourceType: "platform", SourceRef: "log_archive", ResourceScope: "system", ResourceKey: "log_archive",
		RequestUser: SystemUsername, CreatedBy: SystemUsername, Status: scheduledsdk.TaskStatusPending,
	}
	task, err := client.CreateTask(managedCtx, req)
	if err != nil {
		return err
	}
	if task == nil || task.ID <= 0 {
		return fmt.Errorf("timer scheduler returned invalid log archive task")
	}
	_, err = client.UpdateTask(managedCtx, task.ID, updateTaskRequestFromCreate(req))
	return err
}
