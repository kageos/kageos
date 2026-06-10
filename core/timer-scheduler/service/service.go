package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/core/timer-scheduler/repository"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
	"gorm.io/gorm"
)

const (
	defaultListPageSize        = 20
	maxListPageSize            = 100
	defaultDispatchLease       = 30 * time.Second
	defaultExecutionLease      = time.Hour
	defaultQueueAckTimeout     = 2 * time.Minute
	defaultMaxDispatchAttempts = 3
	defaultMaxOutboxAttempts   = 8
	defaultPayloadLimitBytes   = 256 * 1024

	outboxStatusPending = "pending"
	eventTypeRequested  = "timer.execution.requested"
	eventTypeFinished   = "timer.execution.finished"
	triggerScheduled    = "scheduled"
	triggerManual       = "manual"
)

var (
	ErrTaskBusy          = errors.New("timer-scheduler: task has inflight execution")
	ErrInvalidTaskStatus = errors.New("timer-scheduler: invalid task status")
)

type Options struct {
	DispatchLeaseDuration  time.Duration
	ExecutionLeaseDuration time.Duration
	QueueAckTimeout        time.Duration
	MaxDispatchAttempts    int
	MaxOutboxAttempts      int
	PayloadLimitBytes      int
	Now                    func() time.Time
}

type Service struct {
	db            *gorm.DB
	taskRepo      *repository.TimerTaskRepository
	executionRepo *repository.TimerExecutionRepository
	outboxRepo    *repository.TimerOutboxRepository
	now           func() time.Time
	opts          Options
}

func NewService(db *gorm.DB, opts Options) *Service {
	if opts.DispatchLeaseDuration <= 0 {
		opts.DispatchLeaseDuration = defaultDispatchLease
	}
	if opts.ExecutionLeaseDuration <= 0 {
		opts.ExecutionLeaseDuration = defaultExecutionLease
	}
	if opts.QueueAckTimeout <= 0 {
		opts.QueueAckTimeout = defaultQueueAckTimeout
	}
	if opts.MaxDispatchAttempts <= 0 {
		opts.MaxDispatchAttempts = defaultMaxDispatchAttempts
	}
	if opts.MaxOutboxAttempts <= 0 {
		opts.MaxOutboxAttempts = defaultMaxOutboxAttempts
	}
	if opts.PayloadLimitBytes <= 0 {
		opts.PayloadLimitBytes = defaultPayloadLimitBytes
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		db:            db,
		taskRepo:      repository.NewTimerTaskRepository(db),
		executionRepo: repository.NewTimerExecutionRepository(db),
		outboxRepo:    repository.NewTimerOutboxRepository(db),
		now:           now,
		opts:          opts,
	}
}

func (s *Service) CreateTask(ctx context.Context, req scheduledsdk.CreateTaskRequest) (*scheduledsdk.Task, error) {
	if err := validateCreateTaskRequest(req, s.opts.PayloadLimitBytes); err != nil {
		return nil, err
	}
	if existing, err := s.taskRepo.GetByIdempotencyKey(req.IdempotencyKey); err == nil {
		return taskToSDK(existing), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	var idempotencyKeyPtr *string
	if idempotencyKey != "" {
		idempotencyKeyPtr = &idempotencyKey
	}
	now := s.now()
	nextRunAt, err := nextRunForSchedule(req.Schedule, now)
	if err != nil {
		return nil, err
	}
	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = strings.TrimSpace(req.RequestUser)
	}
	task := &model.TimerTask{
		Title:           strings.TrimSpace(req.Title),
		Description:     strings.TrimSpace(req.Description),
		Category:        strings.TrimSpace(req.Category),
		TagsJSON:        mustJSON(req.Tags),
		IdempotencyKey:  idempotencyKeyPtr,
		ExecutorKey:     strings.TrimSpace(req.ExecutorKey),
		ExecutorPayload: cloneRaw(req.ExecutorPayload),
		MetadataJSON:    mustJSON(req.Metadata),
		ScheduleType:    string(req.Schedule.Type),
		CronExpr:        strings.TrimSpace(req.Schedule.CronExpr),
		IntervalSeconds: req.Schedule.IntervalSeconds,
		Timezone:        strings.TrimSpace(req.Schedule.Timezone),
		MaxRuns:         req.Schedule.MaxRuns,
		NextRunAt:       nextRunAt,
		Status:          string(scheduledsdk.TaskStatusPending),
		SourceType:      strings.TrimSpace(req.SourceType),
		SourceRef:       strings.TrimSpace(req.SourceRef),
		ResourceScope:   strings.TrimSpace(req.ResourceScope),
		ResourceKey:     strings.TrimSpace(req.ResourceKey),
		RequestUser:     strings.TrimSpace(req.RequestUser),
		RequestUserDept: strings.TrimSpace(req.RequestUserDept),
		CreatedBy:       createdBy,
	}
	if !req.Schedule.RunAt.IsZero() {
		runAt := req.Schedule.RunAt
		task.RunAt = &runAt
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	return taskToSDK(task), nil
}

func (s *Service) UpdateTask(ctx context.Context, taskID int64, req scheduledsdk.UpdateTaskRequest) (*scheduledsdk.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}
	if task.Status == string(scheduledsdk.TaskStatusCancelled) || task.Status == string(scheduledsdk.TaskStatusDone) {
		return nil, ErrInvalidTaskStatus
	}
	if req.Title != nil {
		task.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		task.Description = strings.TrimSpace(*req.Description)
	}
	if req.Category != nil {
		task.Category = strings.TrimSpace(*req.Category)
	}
	if req.Tags != nil {
		task.TagsJSON = mustJSON(*req.Tags)
	}
	if req.ExecutorPayload != nil {
		if err := validatePayload(req.ExecutorPayload, s.opts.PayloadLimitBytes); err != nil {
			return nil, err
		}
		task.ExecutorPayload = cloneRaw(req.ExecutorPayload)
	}
	if req.Metadata != nil {
		task.MetadataJSON = mustJSON(*req.Metadata)
	}
	if req.SourceType != nil {
		task.SourceType = strings.TrimSpace(*req.SourceType)
	}
	if req.SourceRef != nil {
		task.SourceRef = strings.TrimSpace(*req.SourceRef)
	}
	if req.ResourceScope != nil {
		task.ResourceScope = strings.TrimSpace(*req.ResourceScope)
	}
	if req.ResourceKey != nil {
		task.ResourceKey = strings.TrimSpace(*req.ResourceKey)
	}
	if req.RequestUser != nil {
		task.RequestUser = strings.TrimSpace(*req.RequestUser)
	}
	if req.RequestUserDept != nil {
		task.RequestUserDept = strings.TrimSpace(*req.RequestUserDept)
	}
	if req.Schedule != nil {
		if err := req.Schedule.Validate(); err != nil {
			return nil, err
		}
		nextRunAt, err := nextRunForSchedule(*req.Schedule, s.now())
		if err != nil {
			return nil, err
		}
		task.ScheduleType = string(req.Schedule.Type)
		task.CronExpr = strings.TrimSpace(req.Schedule.CronExpr)
		task.IntervalSeconds = req.Schedule.IntervalSeconds
		task.Timezone = strings.TrimSpace(req.Schedule.Timezone)
		task.MaxRuns = req.Schedule.MaxRuns
		task.NextRunAt = nextRunAt
		task.RunAt = nil
		if !req.Schedule.RunAt.IsZero() {
			runAt := req.Schedule.RunAt
			task.RunAt = &runAt
		}
	}
	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}
	return taskToSDK(task), nil
}

func (s *Service) PauseTask(ctx context.Context, taskID int64) error {
	return s.taskRepo.Pause(taskID)
}

func (s *Service) ResumeTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}
	nextRunAt, err := nextRunForTask(task, s.now())
	if err != nil {
		return err
	}
	return s.taskRepo.Resume(taskID, nextRunAt)
}

func (s *Service) CancelTask(ctx context.Context, taskID int64) error {
	return s.taskRepo.Cancel(taskID)
}

func (s *Service) RunNow(ctx context.Context, taskID int64) (*scheduledsdk.Execution, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}
	if task.InflightExecutionID != 0 {
		return nil, ErrTaskBusy
	}
	exec, err := s.dispatchTask(ctx, task, "", s.now(), triggerManual)
	if err != nil {
		return nil, err
	}
	return executionToSDK(exec), nil
}

func (s *Service) GetTask(ctx context.Context, taskID int64) (*scheduledsdk.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}
	return taskToSDK(task), nil
}

func (s *Service) ListTasks(ctx context.Context, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	list, total, err := s.taskRepo.List(repository.ListTasksFilter{
		ExecutorKey:   req.ExecutorKey,
		Status:        req.Status,
		Category:      req.Category,
		SourceType:    req.SourceType,
		SourceRef:     req.SourceRef,
		ResourceScope: req.ResourceScope,
		ResourceKey:   req.ResourceKey,
		CreatedBy:     req.CreatedBy,
		Offset:        (page - 1) * pageSize,
		Limit:         pageSize,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*scheduledsdk.Task, 0, len(list))
	for _, item := range list {
		out = append(out, taskToSDK(item))
	}
	return &scheduledsdk.ListTasksResponse{List: out, Total: total}, nil
}

func (s *Service) GetExecution(ctx context.Context, taskID, executionID int64) (*scheduledsdk.Execution, error) {
	exec, err := s.executionRepo.GetByID(taskID, executionID)
	if err != nil {
		return nil, err
	}
	return executionToSDK(exec), nil
}

func (s *Service) ListExecutions(ctx context.Context, taskID int64, req scheduledsdk.ListExecutionsRequest) (*scheduledsdk.ListExecutionsResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	list, total, err := s.executionRepo.ListByTaskID(taskID, req.Status, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]*scheduledsdk.Execution, 0, len(list))
	for _, item := range list {
		out = append(out, executionToSDK(item))
	}
	return &scheduledsdk.ListExecutionsResponse{List: out, Total: total}, nil
}

func (s *Service) DispatchDue(ctx context.Context, owner string, limit int) ([]*scheduledsdk.Execution, error) {
	now := s.now()
	tasks, err := s.taskRepo.ListDue(now, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*scheduledsdk.Execution, 0, len(tasks))
	for _, task := range tasks {
		ok, err := s.taskRepo.TryAcquireDispatch(task.ID, owner, now, now.Add(s.opts.DispatchLeaseDuration))
		if err != nil {
			return out, err
		}
		if !ok {
			continue
		}
		exec, err := s.dispatchTask(ctx, task, owner, scheduledAtForTask(task, now), triggerScheduled)
		if err != nil {
			return out, err
		}
		out = append(out, executionToSDK(exec))
	}
	return out, nil
}

func (s *Service) RecoverStaleExecutions(ctx context.Context, limit int) (int, error) {
	now := s.now()
	execs, err := s.executionRepo.ListStale(now, limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	for _, exec := range execs {
		switch exec.Status {
		case string(scheduledsdk.ExecutionStatusQueued):
			if exec.Attempt < s.opts.MaxDispatchAttempts {
				if err := s.requeueExecution(ctx, exec, now); err != nil {
					return recovered, err
				}
				recovered++
				continue
			}
			if err := s.timeoutExecution(ctx, exec, now, "timer-scheduler execution was not picked up before timeout"); err != nil {
				return recovered, err
			}
			recovered++
		case string(scheduledsdk.ExecutionStatusRunning):
			if err := s.timeoutExecution(ctx, exec, now, "timer-scheduler execution heartbeat expired"); err != nil {
				return recovered, err
			}
			recovered++
		}
	}
	return recovered, nil
}

func (s *Service) MarkExecutionStarted(ctx context.Context, req scheduledsdk.MarkExecutionStartedRequest) error {
	startedAt := req.StartedAt
	if startedAt.IsZero() {
		startedAt = s.now()
	}
	ok, err := s.executionRepo.TryMarkRunning(req.TaskID, req.ExecutionID, strings.TrimSpace(req.WorkerID), strings.TrimSpace(req.ExecutorRunID), startedAt, startedAt.Add(s.opts.ExecutionLeaseDuration))
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTaskStatus
	}
	return nil
}

func (s *Service) MarkExecutionHeartbeat(ctx context.Context, req scheduledsdk.MarkExecutionHeartbeatRequest) error {
	heartbeatAt := req.HeartbeatAt
	if heartbeatAt.IsZero() {
		heartbeatAt = s.now()
	}
	ok, err := s.executionRepo.TryHeartbeat(req.TaskID, req.ExecutionID, strings.TrimSpace(req.WorkerID), heartbeatAt, heartbeatAt.Add(s.opts.ExecutionLeaseDuration))
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTaskStatus
	}
	return nil
}

func (s *Service) MarkExecutionFinished(ctx context.Context, req scheduledsdk.MarkExecutionFinishedRequest) error {
	if req.Status == "" {
		return fmt.Errorf("%w: execution status is required", scheduledsdk.ErrInvalidRequest)
	}
	finishedAt := req.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = s.now()
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)

		exec, err := execRepo.GetByID(req.TaskID, req.ExecutionID)
		if err != nil {
			return err
		}
		updates := map[string]interface{}{
			"status":          string(req.Status),
			"finished_at":     finishedAt,
			"duration_millis": req.DurationMillis,
			"output_summary":  strings.TrimSpace(req.OutputSummary),
			"result_payload":  cloneRaw(req.ResultPayload),
			"error_message":   strings.TrimSpace(req.ErrorMessage),
			"lease_until":     nil,
		}
		if strings.TrimSpace(req.ExecutorRunID) != "" {
			updates["executor_run_id"] = strings.TrimSpace(req.ExecutorRunID)
		}
		ok, err := execRepo.TryFinish(req.TaskID, req.ExecutionID, updates)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidTaskStatus
		}

		task, err := taskRepo.GetByID(req.TaskID)
		if err != nil {
			return err
		}
		task.LastErrorMessage = strings.TrimSpace(req.ErrorMessage)
		if exec.TriggerType != triggerManual {
			task.RunCount++
			task.Status, task.NextRunAt, task.LastErrorMessage = computeTaskNextState(task, req.Status == scheduledsdk.ExecutionStatusSuccess, finishedAt)
		}
		ok, err = taskRepo.TryCompleteExecution(task, req.ExecutionID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrTaskBusy
		}

		payload, err := json.Marshal(map[string]interface{}{
			"task_id":        req.TaskID,
			"execution_id":   req.ExecutionID,
			"executor_key":   exec.ExecutorKey,
			"status":         req.Status,
			"finished_at":    finishedAt,
			"trace_id":       exec.TraceID,
			"trigger_type":   exec.TriggerType,
			"source_type":    exec.SourceType,
			"source_ref":     exec.SourceRef,
			"resource_scope": exec.ResourceScope,
			"resource_key":   exec.ResourceKey,
		})
		if err != nil {
			return err
		}
		return outboxRepo.Create(&model.TimerOutboxEvent{
			EventID:     fmt.Sprintf("timer-execution-finished-%d", req.ExecutionID),
			EventType:   eventTypeFinished,
			Subject:     subjects.TimerExecutionFinishedSubject,
			AggregateID: req.ExecutionID,
			Payload:     payload,
			Status:      outboxStatusPending,
		})
	})
}

func (s *Service) PublishPendingOutbox(ctx context.Context, publisher OutboxPublisher, limit int) (int, error) {
	if publisher == nil {
		return 0, fmt.Errorf("timer-scheduler: outbox publisher is nil")
	}
	now := s.now()
	events, err := s.outboxRepo.ListReady(now, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, event := range events {
		if err := publisher.Publish(ctx, event.Subject, event.Payload); err != nil {
			attempts := event.Attempts + 1
			if attempts >= s.opts.MaxOutboxAttempts {
				if markErr := s.outboxRepo.MarkDeadLetter(event.ID, attempts, err.Error()); markErr != nil {
					return published, markErr
				}
				continue
			}
			if markErr := s.outboxRepo.MarkRetry(event.ID, attempts, now.Add(outboxBackoff(attempts)), err.Error()); markErr != nil {
				return published, markErr
			}
			continue
		}
		if err := s.outboxRepo.MarkPublished(event.ID, now); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func (s *Service) dispatchTask(ctx context.Context, task *model.TimerTask, owner string, scheduledAt time.Time, triggerType string) (*model.TimerExecution, error) {
	if triggerType == "" {
		triggerType = triggerScheduled
	}
	var created *model.TimerExecution
	now := s.now()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)

		exec := &model.TimerExecution{
			TaskID:           task.ID,
			ExecutorKey:      task.ExecutorKey,
			Status:           string(scheduledsdk.ExecutionStatusQueued),
			TriggerType:      triggerType,
			ScheduledAt:      scheduledAt,
			LeaseUntil:       ptrTime(now.Add(s.opts.QueueAckTimeout)),
			Attempt:          1,
			LastDispatchedAt: &now,
			TraceID:          uuid.NewString(),
			SourceType:       task.SourceType,
			SourceRef:        task.SourceRef,
			ResourceScope:    task.ResourceScope,
			ResourceKey:      task.ResourceKey,
			RequestUser:      task.RequestUser,
			RequestUserDept:  task.RequestUserDept,
		}
		if err := execRepo.Create(exec); err != nil {
			return err
		}
		var (
			ok  bool
			err error
		)
		if triggerType == triggerManual {
			ok, err = taskRepo.TrySetManualInflight(task.ID, exec.ID)
		} else {
			ok, err = taskRepo.TrySetInflight(task.ID, exec.ID, owner)
		}
		if err != nil {
			return err
		}
		if !ok {
			return ErrTaskBusy
		}
		if err := outboxRepo.Create(s.executionRequestedOutbox(task, exec)); err != nil {
			return err
		}
		created = exec
		return nil
	})
	return created, err
}

func (s *Service) requeueExecution(ctx context.Context, exec *model.TimerExecution, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)
		task, err := taskRepo.GetByID(exec.TaskID)
		if err != nil {
			return err
		}
		ok, err := execRepo.TryRequeueQueued(exec, now, now.Add(s.opts.QueueAckTimeout))
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidTaskStatus
		}
		exec.Attempt++
		exec.LeaseUntil = ptrTime(now.Add(s.opts.QueueAckTimeout))
		exec.LastDispatchedAt = &now
		return outboxRepo.Create(s.executionRequestedOutbox(task, exec))
	})
}

func (s *Service) timeoutExecution(ctx context.Context, exec *model.TimerExecution, now time.Time, message string) error {
	durationMillis := int64(0)
	if exec.StartedAt != nil && now.After(*exec.StartedAt) {
		durationMillis = now.Sub(*exec.StartedAt).Milliseconds()
	}
	return s.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:         exec.TaskID,
		ExecutionID:    exec.ID,
		Status:         scheduledsdk.ExecutionStatusTimeout,
		FinishedAt:     now,
		DurationMillis: durationMillis,
		ErrorMessage:   message,
	})
}

func (s *Service) executionRequestedOutbox(task *model.TimerTask, exec *model.TimerExecution) *model.TimerOutboxEvent {
	event := scheduledsdk.ExecutionRequestedEvent{
		EventID:         fmt.Sprintf("timer-execution-requested-%d-attempt-%d", exec.ID, exec.Attempt),
		TaskID:          task.ID,
		ExecutionID:     exec.ID,
		ExecutorKey:     task.ExecutorKey,
		ScheduledAt:     exec.ScheduledAt,
		TraceID:         exec.TraceID,
		Attempt:         exec.Attempt,
		SourceType:      task.SourceType,
		SourceRef:       task.SourceRef,
		ResourceScope:   task.ResourceScope,
		ResourceKey:     task.ResourceKey,
		RequestUser:     task.RequestUser,
		RequestUserDept: task.RequestUserDept,
		Metadata:        decodeStringMap(task.MetadataJSON),
		ExecutorPayload: cloneRaw(task.ExecutorPayload),
	}
	payload, _ := json.Marshal(event)
	return &model.TimerOutboxEvent{
		EventID:     event.EventID,
		EventType:   eventTypeRequested,
		Subject:     subjects.TimerExecutionRequestedSubject(task.ExecutorKey),
		AggregateID: exec.ID,
		Payload:     payload,
		Status:      outboxStatusPending,
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
