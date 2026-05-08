package scheduledsdk

import "context"

type Adapter interface {
	CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error)
	UpdateTask(ctx context.Context, taskID int64, req UpdateTaskRequest) (*Task, error)
	PauseTask(ctx context.Context, taskID int64) error
	ResumeTask(ctx context.Context, taskID int64) error
	CancelTask(ctx context.Context, taskID int64) error
	RunNow(ctx context.Context, taskID int64) (*Execution, error)
	GetTask(ctx context.Context, taskID int64) (*Task, error)
	ListTasks(ctx context.Context, req ListTasksRequest) (*ListTasksResponse, error)
	GetExecution(ctx context.Context, taskID, executionID int64) (*Execution, error)
	ListExecutions(ctx context.Context, taskID int64, req ListExecutionsRequest) (*ListExecutionsResponse, error)
}

type ExecutionEventAdapter interface {
	PublishExecutionRequested(ctx context.Context, event ExecutionRequestedEvent) error
	MarkExecutionStarted(ctx context.Context, req MarkExecutionStartedRequest) error
	MarkExecutionFinished(ctx context.Context, req MarkExecutionFinishedRequest) error
}
