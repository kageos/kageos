// Package maintenance registers platform jobs with the existing durable scheduler.
package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledauth"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/nats-io/nats.go"
	"strings"
)

type Job struct {
	RunOnCreate             bool
	Key, Title, Description string
	Schedule                scheduledsdk.Schedule
	Handler                 func(context.Context) (any, error)
}

// Start preserves runtime pause state by using the scheduler's idempotent create.
// Definition changes are shipped under the same key; schedule defaults are set once.
func Start(ctx context.Context, nc *nats.Conn, jobs ...Job) error {
	workerClient := scheduledsdk.NewClient(scheduledsdk.Options{Adapter: scheduledsdk.NewNATSAdapter(nc, scheduledsdk.NATSAdapterOptions{})})
	client, err := NewManagementClient()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		job := job
		worker, err := scheduledsdk.NewWorker(scheduledsdk.WorkerOptions{Client: workerClient, NATSConn: nc, ExecutorKey: job.Key, Concurrency: 1, Handler: func(ctx context.Context, e scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
			ctx, err := scheduledauth.WithExecutionToken(ctx, e, 2*time.Hour)
			if err != nil {
				return nil, err
			}
			report, err := job.Handler(ctx)
			payload, marshalErr := json.Marshal(report)
			if marshalErr != nil {
				return nil, marshalErr
			}
			summary := string(payload)
			if len(summary) > 1000 {
				summary = job.Title + "：详情见执行结果"
			}
			return &scheduledsdk.ExecutionResult{OutputSummary: summary, ResultPayload: payload}, err
		}, OnError: func(ctx context.Context, err error) { logger.Warnf(ctx, "[Maintenance] %s: %v", job.Key, err) }})
		if err != nil {
			return err
		}
		if err = worker.Start(ctx); err != nil {
			return err
		}
		go func() { <-ctx.Done(); _ = worker.Stop() }()
		go func() {
			managed := contextx.WithRequestInfo(ctx, contextx.RequestInfo{RequestUser: "system", ClientSource: "app_manifest"})
			for {
				task, err := client.CreateTask(managed, scheduledsdk.CreateTaskRequest{Title: job.Title, Description: job.Description, Category: "platform_maintenance", ExecutorKey: job.Key, IdempotencyKey: job.Key + "-v1", Schedule: job.Schedule, Status: scheduledsdk.TaskStatusPending, OverlapPolicy: scheduledsdk.OverlapPolicyForbid, MaxParallelism: 1, SourceType: "platform", SourceRef: job.Key, ResourceScope: "system", ResourceKey: job.Key, RequestUser: "system", CreatedBy: "system", Metadata: map[string]string{"kind": "platform_maintenance", "managed_by": "app_manifest", "origin": "app_manifest"}})
				if err == nil {
					if job.RunOnCreate && task.RunCount == 0 && task.Status == scheduledsdk.TaskStatusPending && task.InflightExecutionID == 0 {
						_, err = client.RunNow(managed, task.ID)
						if err != nil {
							logger.Warnf(ctx, "[Maintenance] initial run %s: %v", job.Key, err)
						}
					}
					return
				}
				logger.Warnf(ctx, "[Maintenance] register %s: %v", job.Key, err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(15 * time.Second):
				}
			}
		}()
	}
	return nil
}

func Cron(expr string) scheduledsdk.Schedule {
	return scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: expr, Timezone: "Asia/Shanghai"}
}
func Every(seconds int64) scheduledsdk.Schedule {
	return scheduledsdk.Schedule{Type: scheduledsdk.ScheduleEvery, IntervalSeconds: seconds}
}
func FailedItems(count int) error {
	if count > 0 {
		return fmt.Errorf("%d items failed; inspect execution result", count)
	}
	return nil
}

// Management operations use HTTP; the NATS adapter only supports worker lifecycle operations.
func newManagementClient(baseURL string) *scheduledsdk.Client {
	return scheduledsdk.NewClient(scheduledsdk.Options{BaseURL: baseURL})
}

// NewManagementClient connects to the trusted internal scheduler endpoint.
// Public gateway requests strip unsigned identity headers; core-to-core calls
// must use the configured backend address, never impersonate a browser session.
func NewManagementClient() (*scheduledsdk.Client, error) {
	baseURL, err := managementURL(config.GetAPIGatewayConfig().Routes)
	if err != nil {
		return nil, err
	}
	return newManagementClient(baseURL), nil
}
func managementURL(routes []config.RouteConfig) (string, error) {
	for _, route := range routes {
		if (route.ServiceName == "timer" || route.ServiceName == "timer-scheduler") && len(route.Targets) > 0 {
			target := strings.TrimRight(strings.TrimSpace(route.Targets[0].URL), "/")
			if target != "" {
				return target + "/timer/api/v1", nil
			}
		}
	}
	return "", fmt.Errorf("internal timer-scheduler backend is not configured")
}
