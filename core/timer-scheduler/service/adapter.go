package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
)

var _ scheduledsdk.Adapter = (*SDKAdapter)(nil)
var _ scheduledsdk.ExecutionEventAdapter = (*SDKAdapter)(nil)

type SDKAdapter struct {
	service *Service
}

func NewSDKAdapter(service *Service) *SDKAdapter {
	return &SDKAdapter{service: service}
}

func (a *SDKAdapter) CreateTask(ctx context.Context, req scheduledsdk.CreateTaskRequest) (*scheduledsdk.Task, error) {
	return a.service.CreateTask(ctx, req)
}

func (a *SDKAdapter) UpdateTask(ctx context.Context, taskID int64, req scheduledsdk.UpdateTaskRequest) (*scheduledsdk.Task, error) {
	return a.service.UpdateTask(ctx, taskID, req)
}

func (a *SDKAdapter) PauseTask(ctx context.Context, taskID int64) error {
	return a.service.PauseTask(ctx, taskID)
}

func (a *SDKAdapter) ResumeTask(ctx context.Context, taskID int64) error {
	return a.service.ResumeTask(ctx, taskID)
}

func (a *SDKAdapter) CancelTask(ctx context.Context, taskID int64) error {
	return a.service.CancelTask(ctx, taskID)
}

func (a *SDKAdapter) RunNow(ctx context.Context, taskID int64) (*scheduledsdk.Execution, error) {
	return a.service.RunNow(ctx, taskID)
}

func (a *SDKAdapter) GetTask(ctx context.Context, taskID int64) (*scheduledsdk.Task, error) {
	return a.service.GetTask(ctx, taskID)
}

func (a *SDKAdapter) ListTasks(ctx context.Context, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	return a.service.ListTasks(ctx, req)
}

func (a *SDKAdapter) GetExecution(ctx context.Context, taskID, executionID int64) (*scheduledsdk.Execution, error) {
	return a.service.GetExecution(ctx, taskID, executionID)
}

func (a *SDKAdapter) ListExecutions(ctx context.Context, taskID int64, req scheduledsdk.ListExecutionsRequest) (*scheduledsdk.ListExecutionsResponse, error) {
	return a.service.ListExecutions(ctx, taskID, req)
}

func (a *SDKAdapter) PublishExecutionRequested(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) error {
	return a.service.PublishExecutionRequested(ctx, event)
}

func (a *SDKAdapter) MarkExecutionStarted(ctx context.Context, req scheduledsdk.MarkExecutionStartedRequest) error {
	return a.service.MarkExecutionStarted(ctx, req)
}

func (a *SDKAdapter) MarkExecutionFinished(ctx context.Context, req scheduledsdk.MarkExecutionFinishedRequest) error {
	return a.service.MarkExecutionFinished(ctx, req)
}
