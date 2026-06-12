package scheduledsdk

import "context"

type Adapter interface {
	CreateTask(context.Context, CreateTaskRequest) (*Task, error)
	UpdateTask(context.Context, int64, UpdateTaskRequest) (*Task, error)
	PauseTask(context.Context, int64) error
	ResumeTask(context.Context, int64) error
	CancelTask(context.Context, int64) error
	DeleteTask(context.Context, int64) error
	RunNow(context.Context, int64) (*Execution, error)
	GetTask(context.Context, int64) (*Task, error)
	ListTasks(context.Context, ListTasksRequest) (*ListTasksResponse, error)
	GetExecution(context.Context, int64, int64) (*Execution, error)
	ListExecutions(context.Context, int64, ListExecutionsRequest) (*ListExecutionsResponse, error)
	MarkExecutionStarted(context.Context, MarkExecutionStartedRequest) error
	MarkExecutionHeartbeat(context.Context, MarkExecutionHeartbeatRequest) error
	MarkExecutionFinished(context.Context, MarkExecutionFinishedRequest) error
}
