package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/model"
	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const (
	defaultListPageSize       = 20
	defaultDispatchLease      = 30 * time.Second
	defaultExecutionLease     = time.Hour
	eventTypeExecutionRequest = scheduledsdk.SubjectExecutionRequested
	eventTypeExecutionFinish  = scheduledsdk.SubjectExecutionFinished
)

var (
	ErrTaskBusy          = errors.New("timer-scheduler: task has inflight execution")
	ErrInvalidTaskStatus = errors.New("timer-scheduler: invalid task status")
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type Service struct {
	db            *gorm.DB
	taskRepo      *repository.TimerTaskRepository
	executionRepo *repository.TimerExecutionRepository
	outboxRepo    *repository.TimerOutboxRepository
	now           func() time.Time
	opts          Options
}

type Options struct {
	DispatchLeaseDuration  time.Duration
	ExecutionLeaseDuration time.Duration
	Now                    func() time.Time
}

func NewService(db *gorm.DB, opts Options) *Service {
	if opts.DispatchLeaseDuration <= 0 {
		opts.DispatchLeaseDuration = defaultDispatchLease
	}
	if opts.ExecutionLeaseDuration <= 0 {
		opts.ExecutionLeaseDuration = defaultExecutionLease
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
	if err := validateCreateTaskRequest(req); err != nil {
		return nil, err
	}
	now := s.now()
	nextRunAt, err := nextRunForSchedule(req.Schedule, now)
	if err != nil {
		return nil, err
	}
	task := &model.TimerTask{
		Title:               strings.TrimSpace(req.Title),
		Description:         strings.TrimSpace(req.Description),
		Category:            strings.TrimSpace(req.Category),
		TagsJSON:            mustJSON(req.Tags),
		ExecutorKey:         strings.TrimSpace(req.ExecutorKey),
		ExecutorPayload:     cloneRaw(req.ExecutorPayload),
		MetadataJSON:        mustJSON(req.Metadata),
		ScheduleType:        string(req.Schedule.Type),
		CronExpr:            strings.TrimSpace(req.Schedule.CronExpr),
		IntervalSeconds:     req.Schedule.IntervalSeconds,
		MaxRuns:             req.Schedule.MaxRuns,
		Timezone:            strings.TrimSpace(req.Schedule.Timezone),
		NextRunAt:           nextRunAt,
		Status:              string(scheduledsdk.TaskStatusPending),
		SourceType:          strings.TrimSpace(req.SourceType),
		SourceRef:           strings.TrimSpace(req.SourceRef),
		RequestUser:         strings.TrimSpace(req.RequestUser),
		RequestUserDept:     strings.TrimSpace(req.RequestUserDept),
		NotifyUsers:         joinStringList(req.NotifyUsers),
		NotifyDepartments:   joinStringList(req.NotifyDepartments),
		NotifyOn:            normalizeNotifyOn(req.NotifyOn),
		CreatedBy:           strings.TrimSpace(req.RequestUser),
		LastErrorMessage:    "",
		LastExecutionID:     0,
		InflightExecutionID: 0,
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
	if req.ExecutorKey != nil {
		task.ExecutorKey = strings.TrimSpace(*req.ExecutorKey)
	}
	if req.ExecutorPayload != nil {
		task.ExecutorPayload = cloneRaw(req.ExecutorPayload)
	}
	if req.Metadata != nil {
		task.MetadataJSON = mustJSON(*req.Metadata)
	}
	if req.RequestUser != nil {
		task.RequestUser = strings.TrimSpace(*req.RequestUser)
	}
	if req.RequestUserDept != nil {
		task.RequestUserDept = strings.TrimSpace(*req.RequestUserDept)
	}
	if req.NotifyUsers != nil {
		task.NotifyUsers = joinStringList(*req.NotifyUsers)
	}
	if req.NotifyDepartments != nil {
		task.NotifyDepartments = joinStringList(*req.NotifyDepartments)
	}
	if req.NotifyOn != nil {
		task.NotifyOn = normalizeNotifyOn(*req.NotifyOn)
	}
	if req.SourceType != nil {
		task.SourceType = strings.TrimSpace(*req.SourceType)
	}
	if req.SourceRef != nil {
		task.SourceRef = strings.TrimSpace(*req.SourceRef)
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
		task.MaxRuns = req.Schedule.MaxRuns
		task.Timezone = strings.TrimSpace(req.Schedule.Timezone)
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
	exec, err := s.dispatchTask(ctx, task, "", s.now())
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
		ExecutorKey: req.ExecutorKey,
		Status:      req.Status,
		Category:    req.Category,
		SourceType:  req.SourceType,
		SourceRef:   req.SourceRef,
		CreatedBy:   req.CreatedBy,
		Offset:      (page - 1) * pageSize,
		Limit:       pageSize,
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
		exec, err := s.dispatchTask(ctx, task, owner, scheduledAtForTask(task, now))
		if err != nil {
			return out, err
		}
		out = append(out, executionToSDK(exec))
	}
	return out, nil
}

func (s *Service) RecoverStaleExecutions(ctx context.Context, limit int) (int, error) {
	now := s.now()
	queuedBefore := now.Add(-s.opts.ExecutionLeaseDuration)
	execs, err := s.executionRepo.ListStale(now, queuedBefore, limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	for _, exec := range execs {
		errMsg := "timer-scheduler execution timed out"
		if exec.Status == string(scheduledsdk.ExecutionStatusQueued) {
			errMsg = "timer-scheduler execution was not picked up before timeout"
		}
		durationMillis := int64(0)
		if exec.StartedAt != nil && now.After(*exec.StartedAt) {
			durationMillis = now.Sub(*exec.StartedAt).Milliseconds()
		}
		if err := s.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
			TaskID:         exec.TaskID,
			ExecutionID:    exec.ID,
			Status:         scheduledsdk.ExecutionStatusTimeout,
			FinishedAt:     now,
			DurationMillis: durationMillis,
			ErrorMessage:   errMsg,
		}); err != nil {
			if errors.Is(err, ErrInvalidTaskStatus) || errors.Is(err, ErrTaskBusy) {
				continue
			}
			return recovered, err
		}
		recovered++
	}
	return recovered, nil
}

func (s *Service) MarkExecutionStarted(ctx context.Context, req scheduledsdk.MarkExecutionStartedRequest) error {
	startedAt := req.StartedAt
	if startedAt.IsZero() {
		startedAt = s.now()
	}
	ok, err := s.executionRepo.TryMarkRunning(req.TaskID, req.ExecutionID, req.WorkerID, req.ExecutorRunID, startedAt, startedAt.Add(s.opts.ExecutionLeaseDuration))
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
		return fmt.Errorf("timer-scheduler: execution status is required")
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
			"output_summary":  req.OutputSummary,
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
		task.RunCount++
		task.LastErrorMessage = strings.TrimSpace(req.ErrorMessage)
		task.Status, task.NextRunAt, task.LastErrorMessage = computeTaskNextState(task, exec.ScheduledAt, req.Status == scheduledsdk.ExecutionStatusSuccess)
		ok, err = taskRepo.TryCompleteExecution(task, req.ExecutionID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrTaskBusy
		}

		payload, err := json.Marshal(map[string]interface{}{
			"task_id":      req.TaskID,
			"execution_id": req.ExecutionID,
			"executor_key": exec.ExecutorKey,
			"status":       req.Status,
			"finished_at":  finishedAt,
		})
		if err != nil {
			return err
		}
		return outboxRepo.Create(&model.TimerOutboxEvent{
			EventID:     fmt.Sprintf("timer-execution-finished-%d", req.ExecutionID),
			EventType:   eventTypeExecutionFinish,
			AggregateID: req.ExecutionID,
			Payload:     payload,
			Status:      "pending",
		})
	})
}

func (s *Service) PublishExecutionRequested(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	eventID := strings.TrimSpace(event.EventID)
	if eventID == "" {
		eventID = fmt.Sprintf("timer-execution-requested-%d", event.ExecutionID)
	}
	return s.outboxRepo.Create(&model.TimerOutboxEvent{
		EventID:     eventID,
		EventType:   eventTypeExecutionRequest,
		AggregateID: event.ExecutionID,
		Payload:     payload,
		Status:      "pending",
	})
}

func (s *Service) ListPendingOutbox(ctx context.Context, limit int) ([]*model.TimerOutboxEvent, error) {
	return s.outboxRepo.ListPending(limit)
}

func (s *Service) PublishPendingOutbox(ctx context.Context, publisher OutboxPublisher, limit int) (int, error) {
	if publisher == nil {
		return 0, fmt.Errorf("timer-scheduler: outbox publisher is nil")
	}
	events, err := s.outboxRepo.ListPending(limit)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, event := range events {
		if err := publisher.Publish(ctx, event.EventType, event.Payload); err != nil {
			if markErr := s.outboxRepo.MarkFailed(event.ID, err.Error()); markErr != nil {
				return published, markErr
			}
			return published, err
		}
		if err := s.outboxRepo.MarkPublished(event.ID, s.now()); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func (s *Service) dispatchTask(ctx context.Context, task *model.TimerTask, owner string, scheduledAt time.Time) (*model.TimerExecution, error) {
	var created *model.TimerExecution
	err := s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)

		exec := &model.TimerExecution{
			TaskID:      task.ID,
			ExecutorKey: task.ExecutorKey,
			Status:      string(scheduledsdk.ExecutionStatusQueued),
			ScheduledAt: scheduledAt,
			SourceType:  task.SourceType,
			SourceRef:   task.SourceRef,
		}
		if err := execRepo.Create(exec); err != nil {
			return err
		}
		var (
			ok  bool
			err error
		)
		if owner == "" {
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
		event := scheduledsdk.ExecutionRequestedEvent{
			EventID:         fmt.Sprintf("timer-execution-requested-%d", exec.ID),
			TaskID:          task.ID,
			ExecutionID:     exec.ID,
			ExecutorKey:     task.ExecutorKey,
			ScheduledAt:     scheduledAt,
			SourceType:      task.SourceType,
			SourceRef:       task.SourceRef,
			Metadata:        decodeStringMap(task.MetadataJSON),
			ExecutorPayload: cloneRaw(task.ExecutorPayload),
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if err := outboxRepo.Create(&model.TimerOutboxEvent{
			EventID:     event.EventID,
			EventType:   eventTypeExecutionRequest,
			AggregateID: exec.ID,
			Payload:     payload,
			Status:      "pending",
		}); err != nil {
			return err
		}
		created = exec
		return nil
	})
	return created, err
}
