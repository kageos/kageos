package scheduledsdk

import (
	"context"
	"strings"
)

type Client struct {
	adapter Adapter
}

func NewClient(opts Options) *Client {
	if opts.Adapter == nil && strings.TrimSpace(opts.BaseURL) != "" {
		opts.Adapter = NewHTTPAdapter(opts.BaseURL, opts.HTTPClient)
	}
	return &Client{adapter: opts.Adapter}
}

func (c *Client) adapterOrErr() (Adapter, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	if c.adapter == nil {
		return nil, ErrNilAdapter
	}
	return c.adapter, nil
}

func (c *Client) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return nil, err
	}
	return adapter.CreateTask(ctx, req)
}

func (c *Client) UpdateTask(ctx context.Context, taskID int64, req UpdateTaskRequest) (*Task, error) {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return nil, err
	}
	return adapter.UpdateTask(ctx, taskID, req)
}

func (c *Client) PauseTask(ctx context.Context, taskID int64) error {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return err
	}
	return adapter.PauseTask(ctx, taskID)
}

func (c *Client) ResumeTask(ctx context.Context, taskID int64) error {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return err
	}
	return adapter.ResumeTask(ctx, taskID)
}

func (c *Client) CancelTask(ctx context.Context, taskID int64) error {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return err
	}
	return adapter.CancelTask(ctx, taskID)
}

func (c *Client) DeleteTask(ctx context.Context, taskID int64) error {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return err
	}
	return adapter.DeleteTask(ctx, taskID)
}

func (c *Client) RunNow(ctx context.Context, taskID int64) (*Execution, error) {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return nil, err
	}
	return adapter.RunNow(ctx, taskID)
}

func (c *Client) GetTask(ctx context.Context, taskID int64) (*Task, error) {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return nil, err
	}
	return adapter.GetTask(ctx, taskID)
}

func (c *Client) ListTasks(ctx context.Context, req ListTasksRequest) (*ListTasksResponse, error) {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return nil, err
	}
	return adapter.ListTasks(ctx, req)
}

func (c *Client) GetExecution(ctx context.Context, taskID, executionID int64) (*Execution, error) {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return nil, err
	}
	return adapter.GetExecution(ctx, taskID, executionID)
}

func (c *Client) ListExecutions(ctx context.Context, taskID int64, req ListExecutionsRequest) (*ListExecutionsResponse, error) {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return nil, err
	}
	return adapter.ListExecutions(ctx, taskID, req)
}

func (c *Client) MarkExecutionStarted(ctx context.Context, req MarkExecutionStartedRequest) error {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return err
	}
	return adapter.MarkExecutionStarted(ctx, req)
}

func (c *Client) MarkExecutionHeartbeat(ctx context.Context, req MarkExecutionHeartbeatRequest) error {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return err
	}
	return adapter.MarkExecutionHeartbeat(ctx, req)
}

func (c *Client) MarkExecutionFinished(ctx context.Context, req MarkExecutionFinishedRequest) error {
	adapter, err := c.adapterOrErr()
	if err != nil {
		return err
	}
	return adapter.MarkExecutionFinished(ctx, req)
}
