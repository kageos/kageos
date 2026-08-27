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
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
	"gorm.io/gorm"
)

const (
	defaultListPageSize        = 20
	maxListPageSize            = 100
	defaultDispatchLease       = 30 * time.Second
	defaultExecutionLease      = 3 * time.Minute
	defaultQueueAckTimeout     = 2 * time.Minute
	defaultMaxDispatchAttempts = 30 // Two-minute retries for about one hour.
	defaultMaxHeartbeatMisses  = 3
	defaultMaxOutboxAttempts   = 8
	defaultPayloadLimitBytes   = 256 * 1024

	outboxStatusPending = "pending"
	eventTypeRequested  = "timer.execution.requested"
	eventTypeFinished   = "timer.execution.finished"
	triggerScheduled    = "scheduled"
	triggerManual       = "manual"
)

var (
	ErrTaskBusy              = errors.New("timer-scheduler: task dispatch was not acquired")
	ErrInvalidTaskStatus     = errors.New("timer-scheduler: invalid task status")
	ErrBuiltinTaskDefinition = errors.New("timer-scheduler: built-in task definition is read-only")
)

type Options struct {
	DispatchLeaseDuration  time.Duration
	ExecutionLeaseDuration time.Duration
	QueueAckTimeout        time.Duration
	MaxDispatchAttempts    int
	MaxHeartbeatMisses     int
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

func (s *Service) PlatformStats(ctx context.Context) (dto.SystemPlatformServiceStats, error) {
	stats := dto.SystemPlatformServiceStats{}
	if err := s.db.WithContext(ctx).Model(&model.TimerTask{}).Count(&stats.ScheduledTasksTotal).Error; err != nil {
		return stats, err
	}
	if err := s.db.WithContext(ctx).Model(&model.TimerTask{}).Where("status = ?", scheduledsdk.TaskStatusPending).Count(&stats.ScheduledTasksActive).Error; err != nil {
		return stats, err
	}
	return stats, nil
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
	if opts.MaxHeartbeatMisses <= 0 {
		opts.MaxHeartbeatMisses = defaultMaxHeartbeatMisses
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

func scheduledTaskMetadataWithContext(ctx context.Context, metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata)+2)
	for key, value := range metadata {
		out[key] = value
	}
	if token := strings.TrimSpace(contextx.GetToken(ctx)); token != "" {
		if claims, err := auth.NewJWTService().ValidateAccessToken(token); err == nil && claims != nil {
			if claims.Email != "" && strings.TrimSpace(out[scheduledsdk.MetadataRequestEmail]) == "" {
				out[scheduledsdk.MetadataRequestEmail] = claims.Email
			}
			if claims.LeaderUsername != nil && strings.TrimSpace(*claims.LeaderUsername) != "" && strings.TrimSpace(out[scheduledsdk.MetadataLeaderUsername]) == "" {
				out[scheduledsdk.MetadataLeaderUsername] = strings.TrimSpace(*claims.LeaderUsername)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Service) CreateTask(ctx context.Context, req scheduledsdk.CreateTaskRequest) (*scheduledsdk.Task, error) {
	if err := validateCreateTaskRequest(req, s.opts.PayloadLimitBytes); err != nil {
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
	initialStatus := createTaskInitialStatus(req.Status)
	requestUser := strings.TrimSpace(req.RequestUser)
	if requestUser == "" {
		requestUser = strings.TrimSpace(contextx.GetRequestUser(ctx))
	}
	requestUserDept := strings.TrimSpace(req.RequestUserDept)
	if requestUserDept == "" {
		requestUserDept = strings.TrimSpace(contextx.GetRequestDepartmentFullPath(ctx))
	}
	if requestUserDept == "" {
		if claims, err := auth.NewJWTService().ValidateAccessToken(strings.TrimSpace(contextx.GetToken(ctx))); err == nil && claims != nil && claims.DepartmentFullPath != nil {
			requestUserDept = strings.TrimSpace(*claims.DepartmentFullPath)
		}
	}
	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = requestUser
	}
	metadata := scheduledTaskMetadataWithContext(ctx, req.Metadata)
	overlapPolicy := normalizedOverlapPolicy(string(req.OverlapPolicy))
	maxParallelism := normalizedMaxParallelism(string(overlapPolicy), req.MaxParallelism)
	var created *model.TimerTask
	err = s.db.Transaction(func(tx *gorm.DB) error {
		taskRepo := s.taskRepo.WithDB(tx)
		if existing, err := taskRepo.GetByIdempotencyKey(req.IdempotencyKey); err == nil {
			if !isTerminalTaskStatus(existing.Status) {
				created = existing
				return nil
			}
			if err := taskRepo.ReleaseIdempotencyKey(existing.ID, idempotencyKey); err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		task := &model.TimerTask{
			Title:           strings.TrimSpace(req.Title),
			Description:     strings.TrimSpace(req.Description),
			Category:        strings.TrimSpace(req.Category),
			TagsJSON:        mustJSON(req.Tags),
			IdempotencyKey:  idempotencyKeyPtr,
			ExecutorKey:     strings.TrimSpace(req.ExecutorKey),
			ExecutorPayload: cloneRaw(req.ExecutorPayload),
			MetadataJSON:    mustJSON(metadata),
			ScheduleType:    string(req.Schedule.Type),
			CronExpr:        strings.TrimSpace(req.Schedule.CronExpr),
			IntervalSeconds: req.Schedule.IntervalSeconds,
			Timezone:        strings.TrimSpace(req.Schedule.Timezone),
			MaxRuns:         req.Schedule.MaxRuns,
			NextRunAt:       nextRunAt,
			OverlapPolicy:   string(overlapPolicy),
			MaxParallelism:  maxParallelism,
			Status:          string(initialStatus),
			SourceType:      strings.TrimSpace(req.SourceType),
			SourceRef:       strings.TrimSpace(req.SourceRef),
			ResourceScope:   strings.TrimSpace(req.ResourceScope),
			ResourceKey:     strings.TrimSpace(req.ResourceKey),
			RequestUser:     requestUser,
			RequestUserDept: requestUserDept,
			CreatedBy:       createdBy,
		}
		if !req.Schedule.RunAt.IsZero() {
			runAt := req.Schedule.RunAt
			task.RunAt = &runAt
		}
		if err := taskRepo.Create(task); err != nil {
			return err
		}
		created = task
		return nil
	})
	if err != nil {
		return nil, err
	}
	return taskToSDK(created), nil
}

func (s *Service) UpdateTask(ctx context.Context, taskID int64, req scheduledsdk.UpdateTaskRequest) (*scheduledsdk.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}
	if isBuiltinTaskDefinition(task) && !canManageBuiltinTaskDefinition(ctx) {
		return nil, ErrBuiltinTaskDefinition
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
		task.MetadataJSON = mustJSON(scheduledTaskMetadataWithContext(ctx, *req.Metadata))
	}
	if req.OverlapPolicy != nil || req.MaxParallelism != nil {
		previousPolicy := normalizedOverlapPolicy(task.OverlapPolicy)
		policy := previousPolicy
		if req.OverlapPolicy != nil {
			policy = *req.OverlapPolicy
		}
		maxParallelism := task.MaxParallelism
		if req.OverlapPolicy != nil && policy == scheduledsdk.OverlapPolicyAllow && previousPolicy != scheduledsdk.OverlapPolicyAllow && req.MaxParallelism == nil {
			maxParallelism = 0
		}
		if req.MaxParallelism != nil {
			maxParallelism = *req.MaxParallelism
		}
		if err := validateOverlapConfig(policy, maxParallelism); err != nil {
			return nil, err
		}
		task.OverlapPolicy = string(normalizedOverlapPolicy(string(policy)))
		task.MaxParallelism = normalizedMaxParallelism(task.OverlapPolicy, maxParallelism)
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
	if normalizedOverlapPolicy(task.OverlapPolicy) == scheduledsdk.OverlapPolicyForbid {
		if err := s.executionRepo.CancelWaitingByTask(task.ID, s.now(), "overlap policy changed to forbid"); err != nil {
			return nil, err
		}
	}
	return taskToSDK(task), nil
}

func (s *Service) PauseTask(ctx context.Context, taskID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.taskRepo.WithDB(tx).Pause(taskID); err != nil {
			return err
		}
		return s.executionRepo.WithDB(tx).CancelWaitingByTask(taskID, s.now(), "task paused before queued overlap could run")
	})
}

func (s *Service) ResumeTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}
	if scheduledsdk.ScheduleType(task.ScheduleType) == scheduledsdk.ScheduleManual {
		return fmt.Errorf("%w: configure a schedule before enabling a manual task", scheduledsdk.ErrInvalidRequest)
	}
	nextRunAt, err := nextRunForTask(task, s.now())
	if err != nil {
		return err
	}
	return s.taskRepo.Resume(taskID, nextRunAt)
}

func (s *Service) CancelTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}
	if isBuiltinTaskDefinition(task) && !canManageBuiltinTaskDefinition(ctx) {
		return ErrBuiltinTaskDefinition
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.taskRepo.WithDB(tx).Cancel(taskID); err != nil {
			return err
		}
		return s.executionRepo.WithDB(tx).CancelWaitingByTask(taskID, s.now(), "task cancelled before queued overlap could run")
	})
}

func (s *Service) DeleteTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}
	if isBuiltinTaskDefinition(task) && !canManageBuiltinTaskDefinition(ctx) {
		return ErrBuiltinTaskDefinition
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		taskRepo := s.taskRepo.WithDB(tx)
		execRepo := s.executionRepo.WithDB(tx)
		task, err := taskRepo.GetByID(taskID)
		if err != nil {
			return err
		}
		if task.IdempotencyKey != nil {
			if err := taskRepo.ReleaseIdempotencyKey(task.ID, *task.IdempotencyKey); err != nil {
				return err
			}
			task.IdempotencyKey = nil
		}
		if err := execRepo.CancelWaitingByTask(taskID, s.now(), "task deleted before queued overlap could run"); err != nil {
			return err
		}
		return taskRepo.Delete(task)
	})
}

func isBuiltinTaskDefinition(task *model.TimerTask) bool {
	if task == nil {
		return false
	}
	metadata := decodeStringMap(task.MetadataJSON)
	if strings.TrimSpace(metadata["origin"]) == "manifest" {
		return true
	}
	switch strings.TrimSpace(metadata["managed_by"]) {
	case "app_manifest", "capability_bundle":
		return true
	default:
		return false
	}
}

func canManageBuiltinTaskDefinition(ctx context.Context) bool {
	switch strings.TrimSpace(contextx.ResolveClientSource(ctx)) {
	case "app_manifest", "capability_bundle":
		return true
	default:
		return false
	}
}

func (s *Service) RunNow(ctx context.Context, taskID int64) (*scheduledsdk.Execution, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
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
		ExecutorKey:       req.ExecutorKey,
		Status:            req.Status,
		Category:          req.Category,
		SourceType:        req.SourceType,
		SourceRef:         req.SourceRef,
		ResourceScope:     req.ResourceScope,
		ResourceKey:       req.ResourceKey,
		ResourceKeyPrefix: req.ResourceKeyPrefix,
		CreatedBy:         req.CreatedBy,
		Offset:            (page - 1) * pageSize,
		Limit:             pageSize,
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
	var dispatchErr error
	for _, task := range tasks {
		ok, err := s.taskRepo.TryAcquireDispatch(task.ID, owner, now, now.Add(s.opts.DispatchLeaseDuration))
		if err != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("timer-scheduler dispatch acquire task %d: %w", task.ID, err))
			continue
		}
		if !ok {
			continue
		}
		exec, err := s.dispatchTask(ctx, task, owner, scheduledAtForTask(task, now), triggerScheduled)
		if err != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("timer-scheduler dispatch task %d: %w", task.ID, err))
			continue
		}
		if exec != nil {
			out = append(out, executionToSDK(exec))
		}
	}
	return out, dispatchErr
}

func (s *Service) RecoverStaleExecutions(ctx context.Context, limit int) (int, error) {
	now := s.now()
	execs, err := s.executionRepo.ListStale(now, limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	var recoverErr error
	for _, exec := range execs {
		switch exec.Status {
		case string(scheduledsdk.ExecutionStatusQueued):
			if exec.Attempt < s.opts.MaxDispatchAttempts {
				if err := s.requeueExecution(ctx, exec, now); err != nil {
					if !errors.Is(err, ErrInvalidTaskStatus) {
						recoverErr = errors.Join(recoverErr, err)
					}
					continue
				}
				recovered++
				continue
			}
			if err := s.timeoutExecution(ctx, exec, now, "timer-scheduler execution was not picked up before timeout"); err != nil {
				if !errors.Is(err, ErrInvalidTaskStatus) {
					recoverErr = errors.Join(recoverErr, err)
				}
				continue
			}
			recovered++
		case string(scheduledsdk.ExecutionStatusRunning):
			handled, err := s.handleExpiredRunningExecution(ctx, exec, now)
			if err != nil {
				if !errors.Is(err, ErrInvalidTaskStatus) {
					recoverErr = errors.Join(recoverErr, err)
				}
				continue
			}
			if handled {
				recovered++
			}
		}
	}
	cleared, err := s.recoverBrokenInflightReferences(limit)
	if err != nil {
		recoverErr = errors.Join(recoverErr, err)
	}
	recovered += cleared
	return recovered, recoverErr
}

func (s *Service) handleExpiredRunningExecution(ctx context.Context, exec *model.TimerExecution, now time.Time) (bool, error) {
	misses := exec.HeartbeatMisses + 1
	if misses < s.opts.MaxHeartbeatMisses {
		ok, err := s.executionRepo.TryRecordHeartbeatMiss(exec, now, now.Add(s.opts.ExecutionLeaseDuration), misses)
		if err != nil || !ok {
			return ok, err
		}
		return true, nil
	}
	return true, s.timeoutExecution(ctx, exec, now, "timer-scheduler execution heartbeat expired")
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

		task, err := taskRepo.GetByIDForUpdate(req.TaskID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return s.createExecutionFinishedOutbox(outboxRepo, req, exec, finishedAt)
			}
			return err
		}
		task.LastErrorMessage = strings.TrimSpace(req.ErrorMessage)
		if exec.TriggerType != triggerManual {
			task.RunCount++
			task.Status, task.NextRunAt, task.LastErrorMessage = computeTaskNextState(task, req.Status == scheduledsdk.ExecutionStatusSuccess, finishedAt)
		}
		if _, err = taskRepo.TryCompleteExecution(task, req.ExecutionID); err != nil {
			return err
		}
		if isTerminalTaskStatus(task.Status) {
			if err := execRepo.CancelWaitingByTask(task.ID, finishedAt, "task became terminal before queued overlap could run"); err != nil {
				return err
			}
		} else if err := s.promoteWaitingAfterCompletion(execRepo, taskRepo, outboxRepo, task, finishedAt); err != nil {
			return err
		}
		return s.createExecutionFinishedOutbox(outboxRepo, req, exec, finishedAt)
	})
}

func (s *Service) promoteWaitingAfterCompletion(execRepo *repository.TimerExecutionRepository, taskRepo *repository.TimerTaskRepository, outboxRepo *repository.TimerOutboxRepository, task *model.TimerTask, now time.Time) error {
	state, err := execRepo.GetActiveState(task.ID)
	if err != nil {
		return err
	}
	if state.ActiveCount() > 0 {
		latestID, latestErr := execRepo.LatestActiveExecutionID(task.ID)
		if latestErr != nil && !errors.Is(latestErr, gorm.ErrRecordNotFound) {
			return latestErr
		}
		if latestID > 0 {
			if err := taskRepo.SetInflightReference(task.ID, latestID); err != nil {
				return err
			}
		}
	}
	if state.Waiting == nil || task.Status != string(scheduledsdk.TaskStatusPending) {
		return nil
	}
	policy := normalizedOverlapPolicy(task.OverlapPolicy)
	capacityAvailable := false
	switch policy {
	case scheduledsdk.OverlapPolicyQueueLatest:
		capacityAvailable = state.ActiveCount() == 0
	case scheduledsdk.OverlapPolicyAllow:
		capacityAvailable = state.ActiveCount() < int64(normalizedMaxParallelism(task.OverlapPolicy, task.MaxParallelism))
	}
	if !capacityAvailable {
		return nil
	}
	return s.promoteWaitingExecution(execRepo, taskRepo, outboxRepo, task, state.Waiting, now)
}

func (s *Service) createExecutionFinishedOutbox(outboxRepo *repository.TimerOutboxRepository, req scheduledsdk.MarkExecutionFinishedRequest, exec *model.TimerExecution, finishedAt time.Time) error {
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

		if triggerType != triggerManual {
			nextRunAt, nextErr := nextRunAfterScheduledDispatch(task, now)
			if nextErr != nil {
				return nextErr
			}
			state, stateErr := execRepo.GetActiveState(task.ID)
			if stateErr != nil {
				return stateErr
			}
			policy := normalizedOverlapPolicy(task.OverlapPolicy)
			atCapacity := false
			switch policy {
			case scheduledsdk.OverlapPolicyForbid:
				atCapacity = state.ActiveCount() > 0 || state.Waiting != nil
			case scheduledsdk.OverlapPolicyQueueLatest:
				atCapacity = state.ActiveCount() > 0
			case scheduledsdk.OverlapPolicyAllow:
				atCapacity = state.ActiveCount() >= int64(normalizedMaxParallelism(task.OverlapPolicy, task.MaxParallelism))
			}

			if atCapacity && policy == scheduledsdk.OverlapPolicyForbid {
				exec := newTimerExecution(task, scheduledsdk.ExecutionStatusSkipped, triggerType, scheduledAt)
				exec.FinishedAt = ptrTime(now)
				exec.OutputSummary = "Skipped because a previous execution is still active"
				if err := execRepo.Create(exec); err != nil {
					return err
				}
				taskStatus := ""
				if scheduledsdk.ScheduleType(task.ScheduleType) == scheduledsdk.ScheduleAt {
					taskStatus = string(scheduledsdk.TaskStatusDone)
				}
				ok, err := taskRepo.TryAdvanceSchedule(task.ID, owner, taskStatus, nextRunAt, exec.ID)
				if err != nil {
					return err
				}
				if !ok {
					return ErrTaskBusy
				}
				created = exec
				return nil
			}

			if atCapacity {
				if state.Waiting != nil {
					ok, err := execRepo.UpdateWaitingScheduledAt(task.ID, state.Waiting.ID, scheduledAt)
					if err != nil {
						return err
					}
					if !ok {
						return ErrTaskBusy
					}
					ok, err = taskRepo.TryAdvanceSchedule(task.ID, owner, "", nextRunAt, state.Waiting.ID)
					if err != nil {
						return err
					}
					if !ok {
						return ErrTaskBusy
					}
					state.Waiting.ScheduledAt = scheduledAt
					return nil
				}
				exec := newTimerExecution(task, scheduledsdk.ExecutionStatusWaiting, triggerType, scheduledAt)
				if err := execRepo.Create(exec); err != nil {
					return err
				}
				ok, err := taskRepo.TryAdvanceSchedule(task.ID, owner, "", nextRunAt, exec.ID)
				if err != nil {
					return err
				}
				if !ok {
					return ErrTaskBusy
				}
				created = exec
				return nil
			}

			if state.Waiting != nil {
				ok, err := execRepo.UpdateWaitingScheduledAt(task.ID, state.Waiting.ID, scheduledAt)
				if err != nil || !ok {
					if err != nil {
						return err
					}
					return ErrTaskBusy
				}
				state.Waiting.ScheduledAt = scheduledAt
				if err := s.promoteWaitingExecution(execRepo, taskRepo, outboxRepo, task, state.Waiting, now); err != nil {
					return err
				}
				ok, err = taskRepo.TryAdvanceSchedule(task.ID, owner, "", nextRunAt, state.Waiting.ID)
				if err != nil {
					return err
				}
				if !ok {
					return ErrTaskBusy
				}
				created = state.Waiting
				return nil
			}
		}

		exec := newTimerExecution(task, scheduledsdk.ExecutionStatusQueued, triggerType, scheduledAt)
		exec.LeaseUntil = ptrTime(now.Add(s.opts.QueueAckTimeout))
		exec.Attempt = 1
		exec.LastDispatchedAt = ptrTime(now)
		if err := execRepo.Create(exec); err != nil {
			return err
		}
		var (
			ok  bool
			err error
		)
		if triggerType == triggerManual {
			ok, err = taskRepo.TrySetManualInflight(task.ID, exec.ID)
			if err == nil && !ok {
				ok, err = taskRepo.RecordManualExecutionSubmitted(task.ID, exec.ID)
			}
		} else {
			nextRunAt, nextErr := nextRunAfterScheduledDispatch(task, now)
			if nextErr != nil {
				return nextErr
			}
			ok, err = taskRepo.TrySetInflight(task.ID, exec.ID, owner, nextRunAt)
		}
		if err != nil {
			return err
		}
		if !ok {
			if triggerType == triggerManual {
				return ErrInvalidTaskStatus
			}
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

func newTimerExecution(task *model.TimerTask, status scheduledsdk.ExecutionStatus, triggerType string, scheduledAt time.Time) *model.TimerExecution {
	return &model.TimerExecution{
		TaskID:          task.ID,
		ExecutorKey:     task.ExecutorKey,
		Status:          string(status),
		TriggerType:     triggerType,
		ScheduledAt:     scheduledAt,
		TraceID:         uuid.NewString(),
		SourceType:      task.SourceType,
		SourceRef:       task.SourceRef,
		ResourceScope:   task.ResourceScope,
		ResourceKey:     task.ResourceKey,
		RequestUser:     scheduledTaskRequestUser(task),
		RequestUserDept: task.RequestUserDept,
	}
}

func (s *Service) promoteWaitingExecution(execRepo *repository.TimerExecutionRepository, taskRepo *repository.TimerTaskRepository, outboxRepo *repository.TimerOutboxRepository, task *model.TimerTask, exec *model.TimerExecution, now time.Time) error {
	ok, err := execRepo.TryPromoteWaiting(exec, now, now.Add(s.opts.QueueAckTimeout))
	if err != nil {
		return err
	}
	if !ok {
		return ErrTaskBusy
	}
	exec.Status = string(scheduledsdk.ExecutionStatusQueued)
	exec.Attempt = 1
	exec.LeaseUntil = ptrTime(now.Add(s.opts.QueueAckTimeout))
	exec.LastDispatchedAt = ptrTime(now)
	if err := taskRepo.SetInflightReference(task.ID, exec.ID); err != nil {
		return err
	}
	return outboxRepo.Create(s.executionRequestedOutbox(task, exec))
}

func (s *Service) requeueExecution(ctx context.Context, exec *model.TimerExecution, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)
		task, err := taskRepo.GetByID(exec.TaskID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return s.timeoutOrphanExecutionWithRepo(execRepo, exec, now, "timer-scheduler task not found during recovery")
			}
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

func (s *Service) recoverBrokenInflightReferences(limit int) (int, error) {
	tasks, err := s.taskRepo.ListBrokenInflightReferences(limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	var recoverErr error
	for _, task := range tasks {
		executionID := task.InflightExecutionID
		if executionID == 0 {
			continue
		}
		message := fmt.Sprintf("timer-scheduler cleared stale inflight execution reference: execution_id=%d", executionID)
		ok, err := s.taskRepo.TryClearInflight(task.ID, executionID, message)
		if err != nil {
			recoverErr = errors.Join(recoverErr, err)
			continue
		}
		if ok {
			recovered++
		}
	}
	return recovered, recoverErr
}

func (s *Service) recoverBrokenInflightReferenceForTask(taskID int64) (bool, error) {
	task, err := s.taskRepo.GetBrokenInflightReferenceByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	executionID := task.InflightExecutionID
	if executionID == 0 {
		return false, nil
	}
	message := fmt.Sprintf("timer-scheduler cleared stale inflight execution reference: execution_id=%d", executionID)
	return s.taskRepo.TryClearInflight(task.ID, executionID, message)
}

func (s *Service) timeoutExecution(ctx context.Context, exec *model.TimerExecution, now time.Time, message string) error {
	durationMillis := int64(0)
	if exec.StartedAt != nil && now.After(*exec.StartedAt) {
		durationMillis = now.Sub(*exec.StartedAt).Milliseconds()
	}
	err := s.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:         exec.TaskID,
		ExecutionID:    exec.ID,
		Status:         scheduledsdk.ExecutionStatusTimeout,
		FinishedAt:     now,
		DurationMillis: durationMillis,
		ErrorMessage:   message,
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.timeoutOrphanExecution(exec, now, message)
	}
	if errors.Is(err, ErrTaskBusy) {
		return s.timeoutOrphanExecution(exec, now, message)
	}
	return err
}

func (s *Service) timeoutOrphanExecution(exec *model.TimerExecution, now time.Time, message string) error {
	return s.timeoutOrphanExecutionWithRepo(s.executionRepo, exec, now, message)
}

func (s *Service) timeoutOrphanExecutionWithRepo(execRepo *repository.TimerExecutionRepository, exec *model.TimerExecution, now time.Time, message string) error {
	durationMillis := int64(0)
	if exec.StartedAt != nil && now.After(*exec.StartedAt) {
		durationMillis = now.Sub(*exec.StartedAt).Milliseconds()
	}
	ok, err := execRepo.TryFinish(exec.TaskID, exec.ID, map[string]interface{}{
		"status":          string(scheduledsdk.ExecutionStatusTimeout),
		"finished_at":     now,
		"duration_millis": durationMillis,
		"error_message":   strings.TrimSpace(message),
		"lease_until":     nil,
	})
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTaskStatus
	}
	return nil
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
		RequestUser:     scheduledTaskRequestUser(task),
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

func scheduledTaskRequestUser(task *model.TimerTask) string {
	if task == nil {
		return ""
	}
	if user := strings.TrimSpace(task.RequestUser); user != "" {
		return user
	}
	return strings.TrimSpace(task.CreatedBy)
}

func isTerminalTaskStatus(status string) bool {
	switch scheduledsdk.TaskStatus(strings.TrimSpace(status)) {
	case scheduledsdk.TaskStatusCancelled, scheduledsdk.TaskStatusDone, scheduledsdk.TaskStatusFailed:
		return true
	default:
		return false
	}
}
