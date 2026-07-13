package service

import (
	"errors"
	"strings"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/core/timer-scheduler/repository"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/gorm"
)

const (
	defaultListPageSize        = 20
	maxListPageSize            = 100
	defaultDispatchLease       = 30 * time.Second
	defaultExecutionLease      = 3 * time.Minute
	defaultQueueAckTimeout     = 2 * time.Minute
	defaultMaxDispatchAttempts = 3
	defaultMaxHeartbeatMisses  = 3
	defaultMaxOutboxAttempts   = 8
	defaultPayloadLimitBytes   = 256 * 1024

	outboxStatusPending           = "pending"
	eventTypeRequested            = "timer.execution.requested"
	eventTypeFinished             = "timer.execution.finished"
	triggerScheduled              = "scheduled"
	triggerManual                 = "manual"
	taskCancelledExecutionMessage = "timer-scheduler: task was cancelled before execution"
	taskDeletedExecutionMessage   = "timer-scheduler: task was deleted before execution"
)

var (
	ErrTaskBusy          = errors.New("timer-scheduler: task dispatch was not acquired")
	ErrInvalidTaskStatus = errors.New("timer-scheduler: invalid task status")
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
