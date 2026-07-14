package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/core/timer-scheduler/repository"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/gorm"
)

func scheduledTaskMetadataWithContext(ctx context.Context, metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata)+6)
	for key, value := range metadata {
		out[key] = value
	}
	if token := strings.TrimSpace(contextx.GetToken(ctx)); token != "" {
		if claims, err := auth.NewJWTService().ValidateToken(token); err == nil && claims != nil {
			if claims.UserID > 0 && strings.TrimSpace(out[scheduledsdk.MetadataRequestUserID]) == "" {
				out[scheduledsdk.MetadataRequestUserID] = strconv.FormatInt(claims.UserID, 10)
			}
			if claims.Email != "" && strings.TrimSpace(out[scheduledsdk.MetadataRequestEmail]) == "" {
				out[scheduledsdk.MetadataRequestEmail] = claims.Email
			}
			if claims.LeaderUsername != nil && strings.TrimSpace(*claims.LeaderUsername) != "" && strings.TrimSpace(out[scheduledsdk.MetadataLeaderUsername]) == "" {
				out[scheduledsdk.MetadataLeaderUsername] = strings.TrimSpace(*claims.LeaderUsername)
			}
			if claims.CompanyCode != "" && strings.TrimSpace(out[scheduledsdk.MetadataCompanyCode]) == "" {
				out[scheduledsdk.MetadataCompanyCode] = claims.CompanyCode
			}
			if claims.CompanyName != "" && strings.TrimSpace(out[scheduledsdk.MetadataCompanyName]) == "" {
				out[scheduledsdk.MetadataCompanyName] = claims.CompanyName
			}
			if claims.CompanyLogoURL != "" && strings.TrimSpace(out[scheduledsdk.MetadataCompanyLogoURL]) == "" {
				out[scheduledsdk.MetadataCompanyLogoURL] = claims.CompanyLogoURL
			}
		}
	}
	if userID := strings.TrimSpace(contextx.GetRequestUserID(ctx)); userID != "" && strings.TrimSpace(out[scheduledsdk.MetadataRequestUserID]) == "" {
		out[scheduledsdk.MetadataRequestUserID] = userID
	}
	if email := strings.TrimSpace(contextx.GetRequestUserEmail(ctx)); email != "" && strings.TrimSpace(out[scheduledsdk.MetadataRequestEmail]) == "" {
		out[scheduledsdk.MetadataRequestEmail] = email
	}
	if leader := strings.TrimSpace(contextx.GetRequestLeaderUsername(ctx)); leader != "" && strings.TrimSpace(out[scheduledsdk.MetadataLeaderUsername]) == "" {
		out[scheduledsdk.MetadataLeaderUsername] = leader
	}
	if companyCode := strings.TrimSpace(contextx.GetRequestCompanyCode(ctx)); companyCode != "" && strings.TrimSpace(out[scheduledsdk.MetadataCompanyCode]) == "" {
		out[scheduledsdk.MetadataCompanyCode] = companyCode
	}
	if companyName := strings.TrimSpace(contextx.GetRequestCompanyName(ctx)); companyName != "" && strings.TrimSpace(out[scheduledsdk.MetadataCompanyName]) == "" {
		out[scheduledsdk.MetadataCompanyName] = companyName
	}
	if companyLogoURL := strings.TrimSpace(contextx.GetRequestCompanyLogoURL(ctx)); companyLogoURL != "" && strings.TrimSpace(out[scheduledsdk.MetadataCompanyLogoURL]) == "" {
		out[scheduledsdk.MetadataCompanyLogoURL] = companyLogoURL
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
		if claims, err := auth.NewJWTService().ValidateToken(strings.TrimSpace(contextx.GetToken(ctx))); err == nil && claims != nil && claims.DepartmentFullPath != nil {
			requestUserDept = strings.TrimSpace(*claims.DepartmentFullPath)
		}
	}
	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = requestUser
	}
	metadata := scheduledTaskMetadataWithContext(ctx, req.Metadata)
	if err := scheduledsdk.ValidateExecutionMetadata(metadata); err != nil {
		return nil, err
	}
	var created *model.TimerTask
	err = s.db.Transaction(func(tx *gorm.DB) error {
		taskRepo := s.taskRepo.WithDB(tx)
		if existing, err := taskRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey); err == nil {
			if !isTerminalTaskStatus(existing.Status) {
				created = existing
				return nil
			} else if err := taskRepo.ReleaseIdempotencyKey(ctx, existing.ID, idempotencyKey); err != nil {
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
		if err := taskRepo.Create(ctx, task); err != nil {
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
	task, err := s.taskRepo.GetByID(ctx, taskID)
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
		metadata := scheduledTaskMetadataWithContext(ctx, *req.Metadata)
		if err := scheduledsdk.ValidateExecutionMetadata(metadata); err != nil {
			return nil, err
		}
		task.MetadataJSON = mustJSON(metadata)
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
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}
	return taskToSDK(task), nil
}

func (s *Service) PauseTask(ctx context.Context, taskID int64) error {
	return s.taskRepo.Pause(ctx, taskID)
}

func (s *Service) ResumeTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	nextRunAt, err := nextRunForTask(task, s.now())
	if err != nil {
		return err
	}
	return s.taskRepo.Resume(ctx, taskID, nextRunAt)
}

func (s *Service) CancelTask(ctx context.Context, taskID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		if err := execRepo.CancelActiveByTaskID(ctx, taskID, s.now(), taskCancelledExecutionMessage); err != nil {
			return err
		}
		if err := outboxRepo.DeadLetterExecutionRequestsForTask(ctx, taskID, taskCancelledExecutionMessage); err != nil {
			return err
		}
		return taskRepo.Cancel(ctx, taskID)
	})
}

func (s *Service) DeleteTask(ctx context.Context, taskID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		taskRepo := s.taskRepo.WithDB(tx)
		execRepo := s.executionRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)
		task, err := taskRepo.GetByID(ctx, taskID)
		if err != nil {
			return err
		}
		if err := execRepo.CancelActiveByTaskID(ctx, taskID, s.now(), taskDeletedExecutionMessage); err != nil {
			return err
		}
		if err := outboxRepo.DeadLetterExecutionRequestsForTask(ctx, taskID, taskDeletedExecutionMessage); err != nil {
			return err
		}
		if task.IdempotencyKey != nil {
			if err := taskRepo.ReleaseIdempotencyKey(ctx, task.ID, *task.IdempotencyKey); err != nil {
				return err
			}
			task.IdempotencyKey = nil
		}
		return taskRepo.Delete(ctx, task)
	})
}

func (s *Service) RunNow(ctx context.Context, taskID int64) (*scheduledsdk.Execution, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
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
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return taskToSDK(task), nil
}

func (s *Service) ListTasks(ctx context.Context, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	list, total, err := s.taskRepo.List(ctx, repository.ListTasksFilter{
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
