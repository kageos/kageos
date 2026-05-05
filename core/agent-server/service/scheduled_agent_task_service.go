package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/auth"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

const (
	ScheduledAgentScheduleAtime = "atime"
	ScheduledAgentScheduleCron  = "cron"
	ScheduledAgentScheduleEvery = "every"

	ScheduledAgentNotifyNone    = "none"
	ScheduledAgentNotifyAll     = "all"
	ScheduledAgentNotifySuccess = "success"
	ScheduledAgentNotifyFailed  = "failed"

	ScheduledAgentSourceType = "scheduled_agent_task"

	defaultScheduledAgentTaskTimeout = 30 * time.Minute
)

var scheduledAgentCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type ScheduledAgentTaskServiceOptions struct {
	MaxConcurrency      int
	DefaultTimeout      time.Duration
	MessagePublisher    scheduledAgentMessagePublisher
	NotificationBaseURL string
	TimerClient         *scheduledsdk.Client
}

type ScheduledAgentTaskService struct {
	workspaceChat *WorkspaceChatService
	chatRunner    scheduledAgentWorkspaceChatRunner
	taskRepo      *repository.ScheduledAgentTaskRepository
	executionRepo *repository.ScheduledAgentExecutionRepository
	tokenIssuer   *auth.JWTService
	options       ScheduledAgentTaskServiceOptions
	workerID      string
	workerSlots   chan struct{}
	runWG         sync.WaitGroup
}

type scheduledAgentBudgetPolicy struct {
	MaxDurationSeconds int `json:"max_duration_seconds"`
	MaxToolRounds      int `json:"max_tool_rounds"`
	MaxTokens          int `json:"max_tokens"`
}

type scheduledAgentWorkspaceChatRunner interface {
	RunWorkspaceChat(ctx context.Context, req *dto.WorkspaceChatReq, sink WorkspaceChatEventSink) error
}

type scheduledAgentEventCollector struct {
	sessionID string
}

func (c *scheduledAgentEventCollector) Send(event string, data interface{}) {
	if event != EventSession {
		return
	}
	if session, ok := data.(StreamEventSession); ok {
		c.sessionID = session.SessionID
	}
}

func normalizeScheduledAgentTaskOptions(opts ScheduledAgentTaskServiceOptions) ScheduledAgentTaskServiceOptions {
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = 3
	}
	if opts.DefaultTimeout <= 0 {
		opts.DefaultTimeout = defaultScheduledAgentTaskTimeout
	}
	return opts
}

func NewScheduledAgentTaskService(
	workspaceChat *WorkspaceChatService,
	taskRepo *repository.ScheduledAgentTaskRepository,
	executionRepo *repository.ScheduledAgentExecutionRepository,
	tokenIssuer *auth.JWTService,
	opts ScheduledAgentTaskServiceOptions,
) *ScheduledAgentTaskService {
	opts = normalizeScheduledAgentTaskOptions(opts)
	return &ScheduledAgentTaskService{
		workspaceChat: workspaceChat,
		chatRunner:    workspaceChat,
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		tokenIssuer:   tokenIssuer,
		options:       opts,
		workerID:      "scheduled-agent-worker-" + uuid.NewString(),
		workerSlots:   make(chan struct{}, opts.MaxConcurrency),
	}
}

func normalizeScheduledAgentStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func joinScheduledAgentStringList(values []string) string {
	return strings.Join(normalizeScheduledAgentStringList(values), ",")
}

func SplitScheduledAgentRecipientsForAPI(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return normalizeScheduledAgentStringList(strings.Split(raw, ","))
}

func normalizeScheduledAgentNotifyOn(raw string, hasRecipients bool) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		if hasRecipients {
			return ScheduledAgentNotifyAll, nil
		}
		return ScheduledAgentNotifyNone, nil
	}
	switch value {
	case ScheduledAgentNotifyNone, ScheduledAgentNotifyAll, ScheduledAgentNotifySuccess, ScheduledAgentNotifyFailed:
		return value, nil
	default:
		return "", fmt.Errorf("notify_on 不支持，必须是 none/all/success/failed")
	}
}

func normalizeScheduledAgentModeCode(modeCode string) string {
	modeCode = strings.TrimSpace(modeCode)
	if modeCode == "" {
		return "dev"
	}
	return modeCode
}

func resolveScheduledAgentSource(ctx context.Context) (string, string) {
	sourceType := strings.TrimSpace(contextx.GetSourceType(ctx))
	if sourceType == "" {
		sourceType = ScheduledAgentSourceType
	}
	return sourceType, strings.TrimSpace(contextx.GetSourceRef(ctx))
}

func normalizeScheduledAgentRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	if !json.Valid(raw) {
		return nil
	}
	return raw
}

func resolveScheduledAgentLocation(timezone string) (*time.Location, error) {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("timezone 无效: %w", err)
	}
	return loc, nil
}

func parseScheduledAgentRunAt(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("run_at 不能为空")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("支持 RFC3339 或 yyyy-MM-dd HH:mm:ss")
}

func computeScheduledAgentNextRun(task *model.ScheduledAgentTask, scheduledAt time.Time, success bool) (string, *time.Time, string) {
	return computeScheduledAgentNextRunAt(task, scheduledAt, time.Now(), success)
}

func computeScheduledAgentNextRunAt(task *model.ScheduledAgentTask, scheduledAt, completedAt time.Time, success bool) (string, *time.Time, string) {
	if scheduledAt.IsZero() {
		scheduledAt = completedAt
	}
	if completedAt.IsZero() {
		completedAt = time.Now()
	}
	if task.MaxRuns > 0 && task.RunCount >= task.MaxRuns {
		if success {
			return model.ScheduledAgentTaskStatusDone, nil, task.LastErrorMessage
		}
		return model.ScheduledAgentTaskStatusFailed, nil, task.LastErrorMessage
	}

	switch task.ScheduleType {
	case ScheduledAgentScheduleAtime:
		if success {
			return model.ScheduledAgentTaskStatusDone, nil, task.LastErrorMessage
		}
		return model.ScheduledAgentTaskStatusFailed, nil, task.LastErrorMessage
	case ScheduledAgentScheduleCron:
		if strings.TrimSpace(task.CronExpr) == "" {
			if success {
				return model.ScheduledAgentTaskStatusDone, nil, task.LastErrorMessage
			}
			return model.ScheduledAgentTaskStatusFailed, nil, task.LastErrorMessage
		}
		schedule, err := scheduledAgentCronParser.Parse(task.CronExpr)
		if err != nil {
			return model.ScheduledAgentTaskStatusFailed, nil, err.Error()
		}
		nextBase := scheduledAt
		if completedAt.After(nextBase) {
			nextBase = completedAt
		}
		next := schedule.Next(nextBase)
		return model.ScheduledAgentTaskStatusPending, &next, task.LastErrorMessage
	case ScheduledAgentScheduleEvery:
		if task.IntervalSeconds <= 0 {
			if success {
				return model.ScheduledAgentTaskStatusDone, nil, task.LastErrorMessage
			}
			return model.ScheduledAgentTaskStatusFailed, nil, task.LastErrorMessage
		}
		nextBase := scheduledAt
		if completedAt.After(nextBase) {
			nextBase = completedAt
		}
		next := nextBase.Add(time.Duration(task.IntervalSeconds) * time.Second)
		return model.ScheduledAgentTaskStatusPending, &next, task.LastErrorMessage
	default:
		if success {
			return model.ScheduledAgentTaskStatusDone, nil, task.LastErrorMessage
		}
		return model.ScheduledAgentTaskStatusFailed, nil, task.LastErrorMessage
	}
}

func (s *ScheduledAgentTaskService) Create(ctx context.Context, req *dto.CreateScheduledAgentTaskReq, requestUser string) (*model.ScheduledAgentTask, error) {
	requestUser = strings.TrimSpace(requestUser)
	if requestUser == "" {
		return nil, fmt.Errorf("请先登录")
	}
	loc, err := resolveScheduledAgentLocation(req.Timezone)
	if err != nil {
		return nil, err
	}
	scheduleType := strings.ToLower(strings.TrimSpace(req.ScheduleType))
	if scheduleType != ScheduledAgentScheduleAtime && scheduleType != ScheduledAgentScheduleCron && scheduleType != ScheduledAgentScheduleEvery {
		return nil, fmt.Errorf("schedule_type 必须是 atime/cron/every")
	}
	now := time.Now().In(loc)
	runAt := now
	if scheduleType == ScheduledAgentScheduleAtime {
		runAt, err = parseScheduledAgentRunAt(req.RunAt, loc)
		if err != nil {
			return nil, err
		}
	}

	notifyUsers := normalizeScheduledAgentStringList(req.NotifyUsers)
	notifyDepartments := normalizeScheduledAgentStringList(req.NotifyDepartments)
	notifyOn, err := normalizeScheduledAgentNotifyOn(req.NotifyOn, len(notifyUsers) > 0 || len(notifyDepartments) > 0)
	if err != nil {
		return nil, err
	}
	if notifyOn != ScheduledAgentNotifyNone && len(notifyUsers) == 0 && len(notifyDepartments) == 0 {
		return nil, fmt.Errorf("notify_on 不为 none 时必须提供 notify_users 或 notify_departments")
	}
	sourceType, sourceRef := resolveScheduledAgentSource(ctx)

	task := &model.ScheduledAgentTask{
		Name:              strings.TrimSpace(req.Name),
		FullCodePath:      strings.TrimSpace(req.FullCodePath),
		Goal:              strings.TrimSpace(req.Goal),
		ModeCode:          normalizeScheduledAgentModeCode(req.ModeCode),
		Files:             strings.TrimSpace(req.Files),
		LLMConfigID:       req.LLMConfigID,
		ContextPolicy:     normalizeScheduledAgentRawJSON(req.ContextPolicy),
		ToolPolicy:        normalizeScheduledAgentRawJSON(req.ToolPolicy),
		ApprovalPolicy:    normalizeScheduledAgentRawJSON(req.ApprovalPolicy),
		BudgetPolicy:      normalizeScheduledAgentRawJSON(req.BudgetPolicy),
		ScheduleType:      scheduleType,
		RunAt:             runAt,
		CronExpr:          strings.TrimSpace(req.CronExpr),
		IntervalSeconds:   req.IntervalSeconds,
		MaxRuns:           req.MaxRuns,
		Timezone:          strings.TrimSpace(req.Timezone),
		Status:            model.ScheduledAgentTaskStatusPending,
		RequestUser:       strings.TrimSpace(req.RequestUser),
		RequestUserDept:   strings.TrimSpace(req.RequestUserDept),
		NotifyUsers:       strings.Join(notifyUsers, ","),
		NotifyDepartments: strings.Join(notifyDepartments, ","),
		NotifyOn:          notifyOn,
		SourceType:        sourceType,
		SourceRef:         sourceRef,
	}
	if task.Name == "" {
		return nil, fmt.Errorf("name 必填")
	}
	if task.FullCodePath == "" {
		return nil, fmt.Errorf("full_code_path 必填")
	}
	if task.Goal == "" {
		return nil, fmt.Errorf("goal 必填")
	}
	if task.RequestUser == "" {
		task.RequestUser = requestUser
	}
	task.CreatedBy = requestUser
	task.UpdatedBy = requestUser

	next, err := s.initialNextRunAt(task, now)
	if err != nil {
		return nil, err
	}
	task.NextRunAt = next

	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	if task.SourceRef == "" {
		task.SourceRef = scheduledAgentTaskRef(task.ID)
		if err := s.taskRepo.Update(task); err != nil {
			return nil, err
		}
	}
	if err := s.createTimerTask(ctx, task); err != nil {
		_ = s.taskRepo.Delete(task.ID, requestUser)
		return nil, fmt.Errorf("注册 timer-scheduler 任务失败: %w", err)
	}
	return task, nil
}

func (s *ScheduledAgentTaskService) initialNextRunAt(task *model.ScheduledAgentTask, runAt time.Time) (*time.Time, error) {
	switch task.ScheduleType {
	case ScheduledAgentScheduleAtime:
		next := task.RunAt
		if next.IsZero() {
			next = runAt
		}
		if !next.After(runAt) {
			return nil, fmt.Errorf("run_at 必须晚于当前时间")
		}
		return &next, nil
	case ScheduledAgentScheduleCron:
		if strings.TrimSpace(task.CronExpr) == "" {
			return nil, fmt.Errorf("cron 类型需提供 cron_expr")
		}
		schedule, err := scheduledAgentCronParser.Parse(task.CronExpr)
		if err != nil {
			return nil, fmt.Errorf("cron_expr 解析失败: %w", err)
		}
		next := schedule.Next(runAt)
		return &next, nil
	case ScheduledAgentScheduleEvery:
		if task.IntervalSeconds <= 0 {
			return nil, fmt.Errorf("every 类型 interval_seconds 必须大于 0")
		}
		next := runAt.Add(time.Duration(task.IntervalSeconds) * time.Second)
		return &next, nil
	default:
		return nil, fmt.Errorf("schedule_type 必须是 atime/cron/every")
	}
}

func (s *ScheduledAgentTaskService) Update(ctx context.Context, id int64, req *dto.UpdateScheduledAgentTaskReq, requestUser string) (*model.ScheduledAgentTask, error) {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	scheduleChanged := false
	if v := strings.TrimSpace(req.Name); v != "" {
		task.Name = v
	}
	if v := strings.TrimSpace(req.FullCodePath); v != "" {
		task.FullCodePath = v
	}
	if v := strings.TrimSpace(req.Goal); v != "" {
		task.Goal = v
	}
	if v := strings.TrimSpace(req.ModeCode); v != "" {
		task.ModeCode = normalizeScheduledAgentModeCode(v)
	}
	if req.LLMConfigID != nil {
		task.LLMConfigID = *req.LLMConfigID
	}
	if req.Files != nil {
		task.Files = strings.TrimSpace(*req.Files)
	}
	if raw := normalizeScheduledAgentRawJSON(req.ContextPolicy); raw != nil {
		task.ContextPolicy = raw
	}
	if raw := normalizeScheduledAgentRawJSON(req.ToolPolicy); raw != nil {
		task.ToolPolicy = raw
	}
	if raw := normalizeScheduledAgentRawJSON(req.ApprovalPolicy); raw != nil {
		task.ApprovalPolicy = raw
	}
	if raw := normalizeScheduledAgentRawJSON(req.BudgetPolicy); raw != nil {
		task.BudgetPolicy = raw
	}
	if v := strings.TrimSpace(req.RequestUser); v != "" {
		task.RequestUser = v
	}
	task.RequestUserDept = strings.TrimSpace(req.RequestUserDept)
	if req.NotifyUsers != nil || req.NotifyDepartments != nil || strings.TrimSpace(req.NotifyOn) != "" {
		notifyUsers := normalizeScheduledAgentStringList(req.NotifyUsers)
		notifyDepartments := normalizeScheduledAgentStringList(req.NotifyDepartments)
		notifyOn, err := normalizeScheduledAgentNotifyOn(req.NotifyOn, len(notifyUsers) > 0 || len(notifyDepartments) > 0)
		if err != nil {
			return nil, err
		}
		task.NotifyUsers = strings.Join(notifyUsers, ",")
		task.NotifyDepartments = strings.Join(notifyDepartments, ",")
		task.NotifyOn = notifyOn
	}
	if strings.TrimSpace(req.Timezone) != "" {
		timezone := strings.TrimSpace(req.Timezone)
		if timezone != task.Timezone {
			scheduleChanged = true
		}
		task.Timezone = timezone
	}
	loc, err := resolveScheduledAgentLocation(task.Timezone)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ScheduleType) != "" {
		scheduleType := strings.ToLower(strings.TrimSpace(req.ScheduleType))
		if scheduleType != ScheduledAgentScheduleAtime && scheduleType != ScheduledAgentScheduleCron && scheduleType != ScheduledAgentScheduleEvery {
			return nil, fmt.Errorf("schedule_type 必须是 atime/cron/every")
		}
		if scheduleType != task.ScheduleType {
			scheduleChanged = true
		}
		task.ScheduleType = scheduleType
	}
	if strings.TrimSpace(req.RunAt) != "" {
		runAt, err := parseScheduledAgentRunAt(req.RunAt, loc)
		if err != nil {
			return nil, err
		}
		if !runAt.Equal(task.RunAt) {
			scheduleChanged = true
		}
		task.RunAt = runAt
	}
	if strings.TrimSpace(req.CronExpr) != "" {
		cronExpr := strings.TrimSpace(req.CronExpr)
		if cronExpr != task.CronExpr {
			scheduleChanged = true
		}
		task.CronExpr = cronExpr
	}
	if req.IntervalSeconds != nil {
		if *req.IntervalSeconds != task.IntervalSeconds {
			scheduleChanged = true
		}
		task.IntervalSeconds = *req.IntervalSeconds
	}
	if req.MaxRuns != nil {
		task.MaxRuns = *req.MaxRuns
	}
	if scheduleChanged && (task.Status == model.ScheduledAgentTaskStatusPending || task.Status == model.ScheduledAgentTaskStatusPaused) {
		next, err := s.initialNextRunAt(task, time.Now().In(loc))
		if err != nil {
			return nil, err
		}
		task.NextRunAt = next
	}
	task.UpdatedBy = requestUser
	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}
	if err := s.updateTimerTask(ctx, task); err != nil {
		return nil, fmt.Errorf("同步 timer-scheduler 任务失败: %w", err)
	}
	return task, nil
}

func (s *ScheduledAgentTaskService) List(ctx context.Context, createdBy, status, fullCodePath string, page, pageSize int) ([]*model.ScheduledAgentTask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return s.taskRepo.List(createdBy, status, fullCodePath, offset, pageSize)
}

func (s *ScheduledAgentTaskService) Get(ctx context.Context, id int64, _ string) (*model.ScheduledAgentTask, error) {
	return s.taskRepo.GetByID(id)
}

func (s *ScheduledAgentTaskService) Pause(ctx context.Context, id int64, requestUser string) error {
	if err := s.pauseTimerTask(ctx, id); err != nil {
		return err
	}
	return s.taskRepo.Pause(id, requestUser)
}

func (s *ScheduledAgentTaskService) Resume(ctx context.Context, id int64, requestUser string) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}
	loc, err := resolveScheduledAgentLocation(task.Timezone)
	if err != nil {
		return err
	}
	next, err := s.initialNextRunAt(task, time.Now().In(loc))
	if err != nil {
		return err
	}
	if err := s.resumeTimerTask(ctx, id); err != nil {
		return err
	}
	return s.taskRepo.Resume(id, next, requestUser)
}

func (s *ScheduledAgentTaskService) Cancel(ctx context.Context, id int64, requestUser string) error {
	if err := s.cancelTimerTask(ctx, id); err != nil {
		return err
	}
	return s.taskRepo.Cancel(id, requestUser)
}

func (s *ScheduledAgentTaskService) Delete(ctx context.Context, id int64, requestUser string) error {
	if err := s.cancelTimerTask(ctx, id); err != nil {
		return err
	}
	return s.taskRepo.Delete(id, requestUser)
}

func (s *ScheduledAgentTaskService) ListExecutions(ctx context.Context, taskID int64, _ string, status string, page, pageSize int) ([]*model.ScheduledAgentExecution, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return s.executionRepo.ListByTaskID(taskID, status, offset, pageSize)
}

func (s *ScheduledAgentTaskService) GetExecution(ctx context.Context, taskID, executionID int64, _ string) (*model.ScheduledAgentExecution, error) {
	return s.executionRepo.GetByID(taskID, executionID)
}

func (s *ScheduledAgentTaskService) RunNow(ctx context.Context, id int64, _ string) (*scheduledsdk.Execution, error) {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if task.TimerTaskID <= 0 {
		return nil, fmt.Errorf("timer_task_id 为空，无法通过 timer-scheduler 立即运行")
	}
	if s == nil || s.options.TimerClient == nil {
		return nil, fmt.Errorf("timer-scheduler client 未配置")
	}
	return s.options.TimerClient.RunNow(ctx, task.TimerTaskID)
}

func (s *ScheduledAgentTaskService) tryAcquireWorkerSlot() bool {
	select {
	case s.workerSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *ScheduledAgentTaskService) releaseWorkerSlot() {
	select {
	case <-s.workerSlots:
	default:
	}
}

func (s *ScheduledAgentTaskService) newExecution(task *model.ScheduledAgentTask, scheduledAt time.Time, requestUser string, status string) *model.ScheduledAgentExecution {
	now := time.Now()
	startedAt := (*time.Time)(nil)
	workerID := ""
	if status == "" {
		status = model.ScheduledAgentExecutionStatusRunning
	}
	if status == model.ScheduledAgentExecutionStatusRunning {
		startedAt = &now
		workerID = s.workerID
	}
	sourceType := strings.TrimSpace(task.SourceType)
	if sourceType == "" {
		sourceType = ScheduledAgentSourceType
	}
	sourceRef := strings.TrimSpace(task.SourceRef)
	if sourceRef == "" {
		sourceRef = scheduledAgentTaskRef(task.ID)
	}
	exec := &model.ScheduledAgentExecution{
		TaskID:      task.ID,
		ScheduledAt: scheduledAt,
		StartedAt:   startedAt,
		Status:      status,
		WorkerID:    workerID,
		InputGoal:   task.Goal,
		SourceType:  sourceType,
		SourceRef:   sourceRef,
	}
	exec.CreatedBy = requestUser
	exec.UpdatedBy = requestUser
	if exec.CreatedBy == "" {
		exec.CreatedBy = task.RequestUser
		exec.UpdatedBy = task.RequestUser
	}
	return exec
}

func (s *ScheduledAgentTaskService) createExecution(ctx context.Context, task *model.ScheduledAgentTask, scheduledAt time.Time, requestUser string, status string) (*model.ScheduledAgentExecution, error) {
	exec := s.newExecution(task, scheduledAt, requestUser, status)
	if err := s.executionRepo.Create(exec); err != nil {
		return nil, err
	}
	exec.TraceID = fmt.Sprintf("scheduled-agent-%d-%d-%d", task.ID, exec.ID, time.Now().UnixNano())
	if err := s.executionRepo.Update(exec); err != nil {
		return nil, err
	}
	return exec, nil
}

func (s *ScheduledAgentTaskService) executeOne(parent context.Context, task *model.ScheduledAgentTask, exec *model.ScheduledAgentExecution, updateSchedule bool) {
	started := time.Now()
	requestUser := strings.TrimSpace(task.RequestUser)
	if requestUser == "" {
		requestUser = strings.TrimSpace(task.CreatedBy)
	}
	if exec.StartedAt == nil {
		exec.StartedAt = &started
	}
	exec.Status = model.ScheduledAgentExecutionStatusRunning
	exec.WorkerID = s.workerID
	timeout := s.executionTimeout(task)
	runCtx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	runCtx = s.withExecutionRequestInfo(runCtx, task, exec, requestUser)

	collector := &scheduledAgentEventCollector{}
	req := &dto.WorkspaceChatReq{
		FullCodePath: task.FullCodePath,
		Message: dto.WorkspaceMsg{
			Content: task.Goal,
			Files:   task.Files,
		},
		ModeCode:    task.ModeCode,
		LLMConfigID: task.LLMConfigID,
	}

	var err error
	if s.chatRunner == nil {
		err = fmt.Errorf("workspace chat runner 未配置")
	} else {
		err = s.chatRunner.RunWorkspaceChat(runCtx, req, collector)
	}
	finished := time.Now()
	status := model.ScheduledAgentExecutionStatusSuccess
	errMsg := ""
	success := true
	if err != nil {
		success = false
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			status = model.ScheduledAgentExecutionStatusTimeout
		} else if errors.Is(err, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
			status = model.ScheduledAgentExecutionStatusCancelled
		} else {
			status = model.ScheduledAgentExecutionStatusFailed
		}
		errMsg = err.Error()
	}

	summary, toolCount := s.executionSummary(collector.sessionID)
	exec.SessionID = collector.sessionID
	exec.Status = status
	exec.FinishedAt = &finished
	exec.DurationMillis = finished.Sub(started).Milliseconds()
	if exec.DurationMillis < 0 {
		exec.DurationMillis = 0
	}
	exec.OutputSummary = summary
	exec.ToolCallCount = toolCount
	exec.ErrorMessage = errMsg
	exec.UpdatedBy = requestUser
	if err := s.executionRepo.Update(exec); err != nil {
		logger.Errorf(parent, "[ScheduledAgentTask] Update execution failed: task=%d execution=%d err=%v", task.ID, exec.ID, err)
		return
	}

	task.LastSessionID = collector.sessionID
	task.LastExecutionID = exec.ID
	task.LastErrorMessage = errMsg
	task.UpdatedBy = requestUser
	if updateSchedule {
		task.RunCount++
		nextStatus, nextRunAt, stateErr := computeScheduledAgentNextRunAt(task, exec.ScheduledAt, finished, success)
		task.Status = nextStatus
		task.NextRunAt = nextRunAt
		if stateErr != "" && errMsg == "" {
			task.LastErrorMessage = stateErr
		}
		if _, err := s.taskRepo.UpdateAfterTimerRun(task); err != nil {
			logger.Errorf(parent, "[ScheduledAgentTask] Update scheduled task failed: task=%d err=%v", task.ID, err)
		}
	} else {
		if err := s.taskRepo.UpdateAfterManualRun(task); err != nil {
			logger.Errorf(parent, "[ScheduledAgentTask] Update manual task failed: task=%d err=%v", task.ID, err)
		}
	}
	s.notifyExecutionFinished(parent, task, exec, success)
}

func (s *ScheduledAgentTaskService) withExecutionRequestInfo(ctx context.Context, task *model.ScheduledAgentTask, exec *model.ScheduledAgentExecution, requestUser string) context.Context {
	var token string
	if s.tokenIssuer != nil && requestUser != "" {
		if t, err := s.tokenIssuer.GenerateAccessTokenWithHR(0, requestUser, "", task.RequestUserDept, ""); err == nil {
			token = t
		}
	}
	sourceRef := exec.SourceRef
	if sourceRef == "" {
		sourceRef = scheduledAgentExecutionRef(exec.ID)
	}
	return contextx.WithRequestInfo(ctx, contextx.RequestInfo{
		TraceId:            exec.TraceID,
		RequestUser:        requestUser,
		Token:              token,
		DepartmentFullPath: task.RequestUserDept,
		ClientSource:       ScheduledAgentSourceType,
		SourceType:         ScheduledAgentSourceType,
		SourceRef:          sourceRef,
	})
}

func (s *ScheduledAgentTaskService) executionTimeout(task *model.ScheduledAgentTask) time.Duration {
	timeout := s.options.DefaultTimeout
	if len(task.BudgetPolicy) == 0 {
		return timeout
	}
	var policy scheduledAgentBudgetPolicy
	if err := json.Unmarshal(task.BudgetPolicy, &policy); err != nil {
		return timeout
	}
	if policy.MaxDurationSeconds > 0 {
		return time.Duration(policy.MaxDurationSeconds) * time.Second
	}
	return timeout
}

func (s *ScheduledAgentTaskService) executionSummary(sessionID string) (string, int) {
	if strings.TrimSpace(sessionID) == "" || s.workspaceChat == nil || s.workspaceChat.messageRepo == nil {
		return "", 0
	}
	messages, err := s.workspaceChat.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return "", 0
	}
	var summary string
	toolCount := 0
	for _, msg := range messages {
		switch msg.Role {
		case RoleAssistant:
			if strings.TrimSpace(msg.Content) != "" {
				summary = msg.Content
			}
		case RoleTool:
			toolCount++
		}
	}
	return truncateScheduledAgentSummary(summary), toolCount
}

func truncateScheduledAgentSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	const maxRunes = 2000
	runes := []rune(summary)
	if len(runes) <= maxRunes {
		return summary
	}
	return string(runes[:maxRunes]) + "..."
}

func scheduledAgentTaskRef(taskID int64) string {
	if taskID <= 0 {
		return ""
	}
	return fmt.Sprintf("scheduled_agent_task:%d", taskID)
}

func scheduledAgentExecutionRef(executionID int64) string {
	if executionID <= 0 {
		return ""
	}
	return fmt.Sprintf("scheduled_agent_execution:%d", executionID)
}

func (s *ScheduledAgentTaskService) notifyExecutionFinished(ctx context.Context, task *model.ScheduledAgentTask, exec *model.ScheduledAgentExecution, success bool) {
	if !shouldNotifyScheduledAgentTask(task, success) || s.options.MessagePublisher == nil {
		return
	}
	statusLabel := "成功"
	if !success {
		statusLabel = "失败"
	}
	title := fmt.Sprintf("定时会话执行%s：%s", statusLabel, task.Name)
	content := fmt.Sprintf("任务：%s\n\n状态：%s\n\n会话：%s\n\n摘要：\n%s",
		task.Name, statusLabel, exec.SessionID, strings.TrimSpace(exec.OutputSummary))
	from := strings.TrimSpace(task.RequestUser)
	if from == "" {
		from = task.CreatedBy
	}
	if from == "" {
		from = ScheduledAgentSourceType
	}
	envelope := &dto.MessageSendEnvelope{
		Meta: dto.MessageSendMeta{
			From:               from,
			RequestUser:        from,
			DepartmentFullPath: strings.TrimSpace(task.RequestUserDept),
			FullCodePath:       strings.TrimSpace(task.FullCodePath),
			TraceID:            strings.TrimSpace(exec.TraceID),
			ClientSource:       ScheduledAgentSourceType,
			SourceType:         ScheduledAgentSourceType,
			SourceRef:          scheduledAgentExecutionRef(exec.ID),
		},
		Message: dto.MessageSendPayload{
			ToUsers:       task.NotifyUsers,
			ToDepartments: task.NotifyDepartments,
			Title:         title,
			Content:       content,
			ContentType:   "markdown",
		},
	}
	if err := s.options.MessagePublisher.PublishMessage(ctx, envelope); err != nil {
		logger.Errorf(ctx, "[ScheduledAgentTask] Publish notification failed: task=%d execution=%d err=%v", task.ID, exec.ID, err)
	}
}

func shouldNotifyScheduledAgentTask(task *model.ScheduledAgentTask, success bool) bool {
	if task == nil {
		return false
	}
	if strings.TrimSpace(task.NotifyUsers) == "" && strings.TrimSpace(task.NotifyDepartments) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(task.NotifyOn)) {
	case ScheduledAgentNotifyAll, "":
		return true
	case ScheduledAgentNotifySuccess:
		return success
	case ScheduledAgentNotifyFailed:
		return !success
	default:
		return false
	}
}
