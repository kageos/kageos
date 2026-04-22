package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	permissionconstants "github.com/ai-agent-os/ai-agent-os/pkg/permission"
	"github.com/robfig/cron/v3"
)

const (
	ScheduledTaskActionExecute     = "execute"
	ScheduledTaskActionTableCreate = "table_create"
	ScheduledTaskActionTableUpdate = "table_update"
	ScheduledTaskActionTableDelete = "table_delete"
)

var scheduledTaskCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

const (
	defaultSchedulerHeartbeatFile   = "./logs/app-scheduler.heartbeat"
	defaultSchedulerHeartbeatMaxAge = 30 * time.Second
)

const (
	ScheduledTaskNotifyNone    = "none"
	ScheduledTaskNotifyAll     = "all"
	ScheduledTaskNotifySuccess = "success"
	ScheduledTaskNotifyFailed  = "failed"
)

type scheduledTaskAppClient interface {
	RequestApp(ctx context.Context, req *dto.RequestAppReq) (*dto.RequestAppResp, error)
	GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*model.Function, error)
	IncrementFunctionRunCount(ctx context.Context, fullCodePath string)
	RecordFormOperateLog(ctx context.Context, req *dto.RecordFormOperateLogReq) error
	RecordTableOperateLog(ctx context.Context, req *dto.RecordTableOperateLogReq) error
}

type scheduledTaskTokenIssuer interface {
	GenerateAccessTokenWithHR(userID int64, username, email string, departmentFullPath string, leaderUsername string) (string, error)
}

type scheduledTaskMessagePublisher interface {
	PublishMessage(ctx context.Context, payload *dto.MessageSendPayload) error
}

func normalizeScheduledTaskAction(action string) (string, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return ScheduledTaskActionExecute, nil
	}
	switch action {
	case ScheduledTaskActionExecute, ScheduledTaskActionTableCreate, ScheduledTaskActionTableUpdate, ScheduledTaskActionTableDelete:
		return action, nil
	default:
		return "", fmt.Errorf("action 不支持，必须是 execute/table_create/table_update/table_delete")
	}
}

func normalizeScheduledTaskNotifyOn(raw string, hasRecipients bool) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		if hasRecipients {
			return ScheduledTaskNotifyAll, nil
		}
		return ScheduledTaskNotifyNone, nil
	}
	switch value {
	case ScheduledTaskNotifyNone, ScheduledTaskNotifyAll, ScheduledTaskNotifySuccess, ScheduledTaskNotifyFailed:
		return value, nil
	default:
		return "", fmt.Errorf("notify_on 不支持，必须是 none/all/success/failed")
	}
}

func normalizeScheduledTaskStringList(values []string) []string {
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

func joinScheduledTaskStringList(values []string) string {
	return strings.Join(normalizeScheduledTaskStringList(values), ",")
}

func splitScheduledTaskStringList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return normalizeScheduledTaskStringList(strings.Split(raw, ","))
}

func SplitScheduledTaskRecipientsForAPI(raw string) []string {
	return splitScheduledTaskStringList(raw)
}

func parseScheduledTaskCron(expr string) (cron.Schedule, error) {
	sch, err := scheduledTaskCronParser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("cron_expr 解析失败: %w", err)
	}
	return sch, nil
}

func nextCronRunAfter(expr string, base time.Time) (time.Time, error) {
	sch, err := parseScheduledTaskCron(expr)
	if err != nil {
		return time.Time{}, err
	}
	return sch.Next(base), nil
}

func resolveScheduledTaskRunAt(scheduleType, rawRunAt string, now time.Time) (time.Time, error) {
	switch scheduleType {
	case "atime":
		if strings.TrimSpace(rawRunAt) == "" {
			return time.Time{}, fmt.Errorf("atime 类型需提供 run_at")
		}
		runAt, err := parseRunAt(rawRunAt)
		if err != nil {
			return time.Time{}, fmt.Errorf("run_at 格式错误: %w", err)
		}
		return runAt, nil
	case "cron", "every":
		return now, nil
	default:
		return time.Time{}, fmt.Errorf("schedule_type 必须是 atime/cron/every")
	}
}

func computeTaskNextState(task *model.ScheduledTask, scheduledAt time.Time, success bool) (string, *time.Time, string) {
	if scheduledAt.IsZero() {
		scheduledAt = time.Now()
	}
	switch task.ScheduleType {
	case "atime":
		if success {
			return "done", nil, task.ErrorMessage
		}
		return "failed", nil, task.ErrorMessage
	case "cron":
		if task.CronExpr == "" {
			if success {
				return "done", nil, task.ErrorMessage
			}
			return "failed", nil, task.ErrorMessage
		}
		next, err := nextCronRunAfter(task.CronExpr, scheduledAt)
		if err != nil {
			return "failed", nil, err.Error()
		}
		return "pending", &next, task.ErrorMessage
	case "every":
		if task.IntervalSeconds <= 0 {
			if success {
				return "done", nil, task.ErrorMessage
			}
			return "failed", nil, task.ErrorMessage
		}
		if task.MaxRuns > 0 && task.RunCount >= task.MaxRuns {
			if success {
				return "done", nil, task.ErrorMessage
			}
			return "failed", nil, task.ErrorMessage
		}
		next := scheduledAt.Add(time.Duration(task.IntervalSeconds) * time.Second)
		return "pending", &next, task.ErrorMessage
	default:
		if success {
			return "done", nil, task.ErrorMessage
		}
		return "failed", nil, task.ErrorMessage
	}
}

// ScheduledTaskService 定时任务服务
type ScheduledTaskService struct {
	appClient     scheduledTaskAppClient
	tokenIssuer   scheduledTaskTokenIssuer
	taskRepo      *repository.ScheduledTaskRepository
	executionRepo *repository.ScheduledTaskExecutionRepository
	options       ScheduledTaskServiceOptions
	schedulerID   string
	workerSlots   chan struct{}
	runWG         sync.WaitGroup
	lastHeartbeat atomic.Int64
	lastPollAt    atomic.Int64
}

type ScheduledTaskServiceOptions struct {
	PollInterval        time.Duration
	BatchSize           int
	LeaseDuration       time.Duration
	MaxConcurrency      int
	HeartbeatFile       string
	HeartbeatMaxAge     time.Duration
	MessagePublisher    scheduledTaskMessagePublisher
	NotificationBaseURL string
}

type ScheduledTaskHealthSnapshot struct {
	Healthy          bool
	SchedulerID      string
	LastHeartbeatAt  time.Time
	HasHeartbeat     bool
	LastPollAt       time.Time
	HasPoll          bool
	HeartbeatAge     time.Duration
	HeartbeatMaxAge  time.Duration
	PollInterval     time.Duration
	InflightWorkers  int
	AvailableWorkers int
	MaxConcurrency   int
}

func normalizeScheduledTaskServiceOptions(opts ScheduledTaskServiceOptions) ScheduledTaskServiceOptions {
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 50
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = 6 * time.Minute
	}
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = 4
	}
	if strings.TrimSpace(opts.HeartbeatFile) == "" {
		opts.HeartbeatFile = defaultSchedulerHeartbeatFile
	}
	if opts.HeartbeatMaxAge <= 0 {
		opts.HeartbeatMaxAge = defaultSchedulerHeartbeatMaxAge
	}
	minHeartbeatMaxAge := opts.PollInterval * 3
	if minHeartbeatMaxAge <= 0 {
		minHeartbeatMaxAge = 3 * time.Second
	}
	if opts.HeartbeatMaxAge < minHeartbeatMaxAge {
		opts.HeartbeatMaxAge = minHeartbeatMaxAge
	}
	return opts
}

func newSchedulerInstanceID() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%d-%d", host, os.Getpid(), time.Now().UnixNano())
}

func NewScheduledTaskService(
	appClient scheduledTaskAppClient,
	tokenIssuer scheduledTaskTokenIssuer,
	taskRepo *repository.ScheduledTaskRepository,
	executionRepo *repository.ScheduledTaskExecutionRepository,
	opts ScheduledTaskServiceOptions,
) *ScheduledTaskService {
	opts = normalizeScheduledTaskServiceOptions(opts)
	return &ScheduledTaskService{
		appClient:     appClient,
		tokenIssuer:   tokenIssuer,
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		options:       opts,
		schedulerID:   newSchedulerInstanceID(),
		workerSlots:   make(chan struct{}, opts.MaxConcurrency),
	}
}

func (s *ScheduledTaskService) availableWorkerSlots() int {
	if cap(s.workerSlots) == 0 {
		return 0
	}
	return cap(s.workerSlots) - len(s.workerSlots)
}

func (s *ScheduledTaskService) tryAcquireWorkerSlot() bool {
	select {
	case s.workerSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *ScheduledTaskService) releaseWorkerSlot() {
	select {
	case <-s.workerSlots:
	default:
	}
}

// parseFullCodePath 解析 full_code_path 为 user, app, router
func parseFullCodePath(fullCodePath string) (user, app, router string, err error) {
	fullCodePath = strings.TrimPrefix(fullCodePath, "/")
	parts := strings.Split(fullCodePath, "/")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("full_code_path 格式错误，至少需要 user/app/function")
	}
	user = parts[0]
	app = parts[1]
	router = strings.Join(parts[2:], "/")
	return user, app, router, nil
}

func normalizeScheduledTaskFullCodePath(fullCodePath string) string {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return ""
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		return "/" + fullCodePath
	}
	return fullCodePath
}

func scheduledTaskCallbackForAction(action string) (string, bool) {
	switch action {
	case ScheduledTaskActionTableCreate:
		return "OnTableAddRow", true
	case ScheduledTaskActionTableUpdate:
		return "OnTableUpdateRow", true
	case ScheduledTaskActionTableDelete:
		return "OnTableDeleteRows", true
	default:
		return "", false
	}
}

func scheduledTaskMethodForAction(action, requestedMethod string, function *model.Function) string {
	switch action {
	case ScheduledTaskActionTableCreate:
		return "POST"
	case ScheduledTaskActionTableUpdate:
		return "PUT"
	case ScheduledTaskActionTableDelete:
		return "DELETE"
	}
	method := strings.ToUpper(strings.TrimSpace(requestedMethod))
	if method == "" && function != nil {
		method = strings.ToUpper(strings.TrimSpace(function.Method))
	}
	if method == "" {
		method = "POST"
	}
	return method
}

func scheduledTaskTemplateType(function *model.Function) string {
	if function == nil {
		return ""
	}
	return functionschema.Type(function.Schema)
}

func scheduledTaskPermissionAction(templateType, action, method string) (string, error) {
	switch action {
	case ScheduledTaskActionTableCreate:
		return permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionWrite), nil
	case ScheduledTaskActionTableUpdate:
		return permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionUpdate), nil
	case ScheduledTaskActionTableDelete:
		return permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionDelete), nil
	case ScheduledTaskActionExecute:
		switch templateType {
		case functionschema.TypeForm:
			return permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, permissionconstants.ActionWrite), nil
		case functionschema.TypeChart:
			return permissionconstants.BuildActionCode(permissionconstants.ResourceTypeChart, permissionconstants.ActionRead), nil
		case functionschema.TypeTable:
			if strings.EqualFold(method, "GET") {
				return permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionRead), nil
			}
			return "", fmt.Errorf("table 函数写操作必须使用 table_create/table_update/table_delete，不允许用 execute 绕过 schema 回调能力")
		default:
			return "", fmt.Errorf("不支持的函数类型: %s", templateType)
		}
	default:
		return "", fmt.Errorf("未知 action: %s", action)
	}
}

func scheduledTaskTableCallbackError(callbackType string) string {
	switch callbackType {
	case "OnTableAddRow":
		return "该表未开启新增能力，通常是只读查询表，不支持定时新增"
	case "OnTableUpdateRow":
		return "该表未开启编辑能力，通常是只读查询表，不支持定时更新"
	case "OnTableDeleteRows":
		return "该表未开启删除能力，不支持定时删除"
	default:
		return "该表未开启对应回调能力"
	}
}

func (s *ScheduledTaskService) resolveScheduledTaskFunction(ctx context.Context, fullCodePath string) (*model.Function, error) {
	if s.appClient == nil {
		return nil, fmt.Errorf("app service 未初始化，无法校验定时任务目标函数")
	}
	function, err := s.appClient.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		return nil, err
	}
	if function == nil {
		return nil, fmt.Errorf("函数不存在: %s", fullCodePath)
	}
	if _, err := functionschema.Parse(function.Schema); err != nil {
		return nil, fmt.Errorf("函数 schema 非法: %w", err)
	}
	return function, nil
}

func (s *ScheduledTaskService) validateScheduledTaskTarget(ctx context.Context, fullCodePath, action, method, requestUser string) (*model.Function, string, error) {
	fullCodePath = normalizeScheduledTaskFullCodePath(fullCodePath)
	function, err := s.resolveScheduledTaskFunction(ctx, fullCodePath)
	if err != nil {
		return nil, "", err
	}
	templateType := scheduledTaskTemplateType(function)
	effectiveMethod := scheduledTaskMethodForAction(action, method, function)

	if callbackType, ok := scheduledTaskCallbackForAction(action); ok {
		if templateType != functionschema.TypeTable {
			return nil, "", fmt.Errorf("%s 只支持 table 函数，当前函数类型为 %s", action, templateType)
		}
		if !function.HasCallback(callbackType) {
			return nil, "", fmt.Errorf("%s: %s", scheduledTaskTableCallbackError(callbackType), callbackType)
		}
	}

	permissionAction, err := scheduledTaskPermissionAction(templateType, action, effectiveMethod)
	if err != nil {
		return nil, "", err
	}
	if err := ensureScheduledTaskPermission(ctx, requestUser, fullCodePath, permissionAction); err != nil {
		return nil, "", err
	}
	return function, effectiveMethod, nil
}

func ensureScheduledTaskPermission(ctx context.Context, requestUser, fullCodePath, action string) error {
	requestUser = strings.TrimSpace(requestUser)
	if requestUser == "" {
		return fmt.Errorf("缺少 request_user，无法校验定时任务权限")
	}
	if !license.GetManager().HasFeature(enterprise.FeaturePermission) {
		return nil
	}
	hasPermission, err := enterprise.GetPermissionService().CheckPermission(ctx, requestUser, fullCodePath, action)
	if err != nil {
		return fmt.Errorf("权限检查失败: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("无权限创建或执行该定时任务: resource=%s action=%s user=%s", fullCodePath, action, requestUser)
	}
	return nil
}

// Create 创建定时任务，计算 next_run_at
func (s *ScheduledTaskService) Create(ctx context.Context, req *dto.CreateScheduledTaskReq, requestUser string) (*model.ScheduledTask, error) {
	fullCodePath := normalizeScheduledTaskFullCodePath(req.FullCodePath)
	user, appName, _, err := parseFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}
	scheduleType := strings.TrimSpace(req.ScheduleType)
	if scheduleType == "" {
		return nil, fmt.Errorf("schedule_type 必须是 atime/cron/every")
	}
	action, err := normalizeScheduledTaskAction(req.Action)
	if err != nil {
		return nil, err
	}
	reqUser := strings.TrimSpace(req.RequestUser)
	if reqUser == "" {
		reqUser = requestUser
	}
	reqUserDept := strings.TrimSpace(req.RequestUserDept)
	if reqUserDept == "" {
		reqUserDept = contextx.GetRequestDepartmentFullPath(ctx)
	}
	_, method, err := s.validateScheduledTaskTarget(ctx, fullCodePath, action, req.Method, reqUser)
	if err != nil {
		return nil, err
	}
	runAt, err := resolveScheduledTaskRunAt(scheduleType, req.RunAt, time.Now())
	if err != nil {
		return nil, err
	}
	notifyUsers := normalizeScheduledTaskStringList(req.NotifyUsers)
	notifyDepartments := normalizeScheduledTaskStringList(req.NotifyDepartments)
	hasNotifyRecipients := len(notifyUsers) > 0 || len(notifyDepartments) > 0
	notifyOn, err := normalizeScheduledTaskNotifyOn(req.NotifyOn, hasNotifyRecipients)
	if err != nil {
		return nil, err
	}
	if notifyOn != ScheduledTaskNotifyNone && !hasNotifyRecipients {
		return nil, fmt.Errorf("notify_on 启用时必须提供 notify_users 或 notify_departments")
	}

	task := &model.ScheduledTask{
		User:              user,
		App:               appName,
		Name:              strings.TrimSpace(req.Name),
		FullCodePath:      fullCodePath,
		Action:            action,
		Method:            method,
		Payload:           req.Payload,
		RequestUser:       reqUser,
		RequestUserDept:   reqUserDept,
		CreatedBy:         requestUser,
		ScheduleType:      scheduleType,
		RunAt:             runAt,
		CronExpr:          req.CronExpr,
		IntervalSeconds:   req.IntervalSeconds,
		MaxRuns:           req.MaxRuns,
		Status:            "pending",
		Timezone:          req.Timezone,
		NotifyUsers:       strings.Join(notifyUsers, ","),
		NotifyDepartments: strings.Join(notifyDepartments, ","),
		NotifyOn:          notifyOn,
	}

	switch scheduleType {
	case "atime":
		task.NextRunAt = &runAt
	case "cron":
		if req.CronExpr == "" {
			return nil, fmt.Errorf("cron 类型需提供 cron_expr")
		}
		next, err := nextCronRunAfter(req.CronExpr, runAt)
		if err != nil {
			return nil, err
		}
		task.NextRunAt = &next
	case "every":
		if req.IntervalSeconds <= 0 {
			return nil, fmt.Errorf("every 类型需提供 interval_seconds > 0")
		}
		task.IntervalSeconds = req.IntervalSeconds
		task.MaxRuns = req.MaxRuns
		task.NextRunAt = &runAt
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	return task, nil
}

// List 分页列表。
// 传 full_code_path 时按资源路径列出任务；未传时返回当前用户创建的任务。
func (s *ScheduledTaskService) List(ctx context.Context, createdBy string, status string, fullCodePath string, page, pageSize int) ([]*model.ScheduledTask, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.taskRepo.ListByUser(createdBy, status, fullCodePath, offset, pageSize)
}

func (s *ScheduledTaskService) Get(ctx context.Context, id int64, _ string) (*model.ScheduledTask, error) {
	return s.taskRepo.GetByID(id)
}

// Cancel 取消任务（仅创建人可取消）
func (s *ScheduledTaskService) Cancel(ctx context.Context, id int64, createdBy string) error {
	return s.taskRepo.Cancel(id, createdBy)
}

// ListExecutions 某任务的执行记录。
// 列表按资源路径展示后，这里也允许查看同一路径上的任务执行结果。
func (s *ScheduledTaskService) ListExecutions(ctx context.Context, taskID int64, createdBy string, status string, page, pageSize int) ([]*model.ScheduledTaskExecution, int64, error) {
	if _, err := s.taskRepo.GetByID(taskID); err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return s.executionRepo.ListByTaskID(taskID, status, offset, pageSize)
}

func (s *ScheduledTaskService) GetExecution(ctx context.Context, taskID int64, executionID int64, _ string) (*model.ScheduledTaskExecution, error) {
	if _, err := s.taskRepo.GetByID(taskID); err != nil {
		return nil, err
	}
	return s.executionRepo.GetByID(taskID, executionID)
}

// StartScheduler 启动调度器，应在 worker 或 app-server 启动时调用。
// 调度源以 DB 为准：每次轮询查询到点且无有效租约的任务，再用原子更新抢占租约，
// 这样即使多实例并发运行，也只有抢到租约的实例会实际执行该任务。
func (s *ScheduledTaskService) StartScheduler(ctx context.Context) {
	logger.Infof(ctx, "[ScheduledTask] Scheduler loop started: worker=%s poll_interval=%s batch_size=%d lease_duration=%s max_concurrency=%d",
		s.schedulerID, s.options.PollInterval, s.options.BatchSize, s.options.LeaseDuration, s.options.MaxConcurrency)
	s.writeHeartbeat(ctx, time.Now())
	s.runDueTasks(ctx)
	pollCompletedAt := time.Now()
	s.markPollCompleted(pollCompletedAt)
	s.writeHeartbeat(ctx, pollCompletedAt)
	ticker := time.NewTicker(s.options.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Infof(ctx, "[ScheduledTask] Scheduler stopping: worker=%s waiting_inflight=%d",
				s.schedulerID, len(s.workerSlots))
			s.runWG.Wait()
			logger.Infof(ctx, "[ScheduledTask] Scheduler stopped: worker=%s", s.schedulerID)
			return
		case <-ticker.C:
			s.runDueTasks(ctx)
			pollCompletedAt := time.Now()
			s.markPollCompleted(pollCompletedAt)
			s.writeHeartbeat(ctx, pollCompletedAt)
		}
	}
}

// runDueTasks 查询到点任务并抢占租约，抢占成功的实例负责执行。
func (s *ScheduledTaskService) runDueTasks(ctx context.Context) {
	availableSlots := s.availableWorkerSlots()
	if availableSlots <= 0 {
		return
	}
	limit := s.options.BatchSize
	if limit <= 0 || limit > availableSlots {
		limit = availableSlots
	}
	now := time.Now()
	tasks, err := s.taskRepo.ListPendingDue(now, limit)
	if err != nil {
		logger.Errorf(ctx, "[ScheduledTask] ListPendingDue err: %v", err)
		return
	}
	for _, task := range tasks {
		if !s.tryAcquireWorkerSlot() {
			return
		}
		leaseUntil := now.Add(s.options.LeaseDuration)
		acquired, err := s.taskRepo.TryAcquireLease(task.ID, s.schedulerID, now, leaseUntil)
		if err != nil {
			logger.Errorf(ctx, "[ScheduledTask] TryAcquireLease err: task=%d err=%v", task.ID, err)
			s.releaseWorkerSlot()
			continue
		}
		if !acquired {
			s.releaseWorkerSlot()
			continue
		}
		task.LeaseOwner = s.schedulerID
		task.LeaseUntil = &leaseUntil
		s.runWG.Add(1)
		go func(task *model.ScheduledTask) {
			defer s.runWG.Done()
			defer s.releaseWorkerSlot()
			s.executeOne(ctx, task)
		}(task)
	}
}

// executeOne 执行一条任务：注入“请求用户”context、用 JWT 生成 Token，再调 RequestApp、写执行记录、更新任务
func (s *ScheduledTaskService) executeOne(ctx context.Context, task *model.ScheduledTask) {
	taskCtx := context.WithoutCancel(ctx)
	executedAt := time.Now()
	scheduledAt := executedAt
	if task.NextRunAt != nil {
		scheduledAt = *task.NextRunAt
	}
	elapsedMillis := func() int64 {
		duration := time.Since(executedAt).Milliseconds()
		if duration < 0 {
			return 0
		}
		return duration
	}
	user, appName, routerPath, err := parseFullCodePath(task.FullCodePath)
	if err != nil {
		s.finishExecution(taskCtx, task, nil, nil, "failed", err.Error(), "", executedAt, elapsedMillis(), scheduledAt, false)
		return
	}
	// 定时任务无 HTTP 请求，用 WithRequestInfo 一次性注入与 ToContext 一致的 context
	traceId := fmt.Sprintf("scheduled-%d-%d", task.ID, executedAt.UnixNano())
	requestUser := strings.TrimSpace(task.RequestUser)
	if requestUser == "" {
		requestUser = strings.TrimSpace(task.CreatedBy)
	}
	var token string
	if s.tokenIssuer != nil && requestUser != "" {
		if t, err := s.tokenIssuer.GenerateAccessTokenWithHR(0, requestUser, "", task.RequestUserDept, ""); err == nil {
			token = t
		}
	}
	runCtx := contextx.WithRequestInfo(taskCtx, contextx.RequestInfo{
		TraceId:            traceId,
		RequestUser:        requestUser,
		Token:              token,
		DepartmentFullPath: task.RequestUserDept,
	})
	action, err := normalizeScheduledTaskAction(task.Action)
	if err != nil {
		s.finishExecution(taskCtx, task, nil, nil, "failed", err.Error(), traceId, executedAt, elapsedMillis(), scheduledAt, false)
		return
	}
	function, effectiveMethod, err := s.validateScheduledTaskTarget(runCtx, task.FullCodePath, action, task.Method, requestUser)
	if err != nil {
		s.finishExecution(taskCtx, task, nil, nil, "failed", err.Error(), traceId, executedAt, elapsedMillis(), scheduledAt, false)
		return
	}
	// 若 Payload 是「JSON 字符串」（前端曾用 JSON.stringify(payload) 导致存成 "{\"a\":1}"），
	// 解一层再作为 body，否则表单侧 json.Unmarshal(body, &struct) 会报 cannot unmarshal string into Go value of type ...
	bodyBytes := task.Payload
	if len(task.Payload) >= 2 && task.Payload[0] == '"' {
		var inner string
		if err := json.Unmarshal(task.Payload, &inner); err == nil {
			bodyBytes = []byte(inner)
		}
	}
	req, callbackType, tableBodyBytes, err := s.buildTaskRequest(runCtx, task, user, appName, routerPath, effectiveMethod, traceId, token, requestUser, bodyBytes)
	if err != nil {
		duration := elapsedMillis()
		s.recordFunctionOperateLog(taskCtx, task, function, user, appName, routerPath, task.Payload, nil, err, traceId, duration)
		s.finishExecution(taskCtx, task, task.Payload, nil, "failed", err.Error(), traceId, executedAt, duration, scheduledAt, false)
		return
	}
	s.recordScheduledTableOperateLog(taskCtx, task, user, appName, routerPath, callbackType, tableBodyBytes, traceId)

	resp, err := s.appClient.RequestApp(runCtx, req)
	var respBody []byte
	if resp != nil {
		// 存完整响应（trace_id、result、error、err_code），便于区分「成功但 result 为空」与真正失败
		respBody, _ = json.Marshal(resp)
	}
	reqBody := req.Body
	if err != nil {
		duration := elapsedMillis()
		s.recordFunctionOperateLog(taskCtx, task, function, user, appName, routerPath, reqBody, resp, err, traceId, duration)
		s.finishExecution(taskCtx, task, reqBody, respBody, "failed", err.Error(), traceId, executedAt, duration, scheduledAt, false)
		return
	}
	if resp == nil {
		errMsg := "应用响应为空"
		duration := elapsedMillis()
		s.recordFunctionOperateLog(taskCtx, task, function, user, appName, routerPath, reqBody, resp, errors.New(errMsg), traceId, duration)
		s.finishExecution(taskCtx, task, reqBody, respBody, "failed", errMsg, traceId, executedAt, duration, scheduledAt, false)
		return
	}
	if resp.Error != "" || resp.ErrCode != 0 {
		errMsg := resp.Error
		if errMsg == "" {
			errMsg = fmt.Sprintf("应用返回错误码 %d", resp.ErrCode)
		} else if resp.ErrCode != 0 {
			errMsg = fmt.Sprintf("%s (err_code=%d)", errMsg, resp.ErrCode)
		}
		duration := elapsedMillis()
		s.recordFunctionOperateLog(taskCtx, task, function, user, appName, routerPath, reqBody, resp, nil, traceId, duration)
		s.finishExecution(taskCtx, task, reqBody, respBody, "failed", errMsg, traceId, executedAt, duration, scheduledAt, false)
		return
	}
	duration := elapsedMillis()
	s.incrementScheduledFunctionRunCount(taskCtx, task.FullCodePath)
	s.recordFunctionOperateLog(taskCtx, task, function, user, appName, routerPath, reqBody, resp, nil, traceId, duration)
	s.finishExecution(taskCtx, task, reqBody, respBody, "success", "", traceId, executedAt, duration, scheduledAt, true)
}

func (s *ScheduledTaskService) buildTaskRequest(ctx context.Context, task *model.ScheduledTask, user, appName, routerPath, method, traceID, token, requestUser string, bodyBytes []byte) (*dto.RequestAppReq, string, []byte, error) {
	action, err := normalizeScheduledTaskAction(task.Action)
	if err != nil {
		return nil, "", nil, err
	}

	switch action {
	case ScheduledTaskActionExecute:
		return &dto.RequestAppReq{
			User:            user,
			App:             appName,
			Router:          routerPath,
			Method:          method,
			Body:            bodyBytes,
			RequestUser:     requestUser,
			RequestUserDept: task.RequestUserDept,
			TraceId:         traceID,
			Token:           token,
		}, "", bodyBytes, nil
	case ScheduledTaskActionTableCreate:
		req, err := s.buildTableCallbackReq(user, appName, routerPath, method, "OnTableAddRow", traceID, token, requestUser, task.RequestUserDept, bodyBytes)
		return req, "OnTableAddRow", bodyBytes, err
	case ScheduledTaskActionTableUpdate:
		bodyBytes, err := s.fillTableUpdateOldValues(ctx, task, user, appName, routerPath, traceID, token, requestUser, bodyBytes)
		if err != nil {
			return nil, "", nil, err
		}
		req, err := s.buildTableCallbackReq(user, appName, routerPath, method, "OnTableUpdateRow", traceID, token, requestUser, task.RequestUserDept, bodyBytes)
		return req, "OnTableUpdateRow", bodyBytes, err
	case ScheduledTaskActionTableDelete:
		req, err := s.buildTableCallbackReq(user, appName, routerPath, method, "OnTableDeleteRows", traceID, token, requestUser, task.RequestUserDept, bodyBytes)
		return req, "OnTableDeleteRows", bodyBytes, err
	default:
		return nil, "", nil, fmt.Errorf("未知 action: %s", task.Action)
	}
}

// fillTableUpdateOldValues 在执行前实时补齐 old_values，避免创建任务时缓存的旧值变脏。
func (s *ScheduledTaskService) fillTableUpdateOldValues(ctx context.Context, task *model.ScheduledTask, user, appName, routerPath, traceID, token, requestUser string, bodyBytes []byte) ([]byte, error) {
	if len(bodyBytes) == 0 {
		return bodyBytes, fmt.Errorf("table_update payload 不能为空")
	}

	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err != nil {
		return bodyBytes, fmt.Errorf("解析 table_update payload 失败: %w", err)
	}

	// 已携带 old_values 则不覆盖，保持显式传参优先。
	if oldValues, ok := bodyData["old_values"].(map[string]interface{}); ok && len(oldValues) > 0 {
		return bodyBytes, nil
	}

	idValue, ok := bodyData["id"]
	if !ok {
		return bodyBytes, fmt.Errorf("table_update payload 缺少 id，无法自动补 old_values")
	}

	var rowID int64
	switch v := idValue.(type) {
	case float64:
		rowID = int64(v)
	case int64:
		rowID = v
	case int:
		rowID = int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return bodyBytes, fmt.Errorf("table_update payload 的 id 非法: %v", err)
		}
		rowID = parsed
	default:
		return bodyBytes, fmt.Errorf("table_update payload 的 id 类型不支持: %T", idValue)
	}
	if rowID <= 0 {
		return bodyBytes, fmt.Errorf("table_update payload 的 id 必须大于 0")
	}

	searchReq := &dto.RequestAppReq{
		User:            user,
		App:             appName,
		Router:          routerPath,
		Method:          "GET",
		UrlQuery:        "eq=id:" + url.QueryEscape(strconv.FormatInt(rowID, 10)) + "&page=1&page_size=1",
		TraceId:         traceID,
		RequestUser:     requestUser,
		RequestUserDept: task.RequestUserDept,
		Token:           token,
	}

	searchResp, err := s.appClient.RequestApp(ctx, searchReq)
	if err != nil {
		return bodyBytes, fmt.Errorf("执行前查询当前行失败: %w", err)
	}
	if searchResp == nil {
		return bodyBytes, fmt.Errorf("执行前查询当前行失败: 响应为空")
	}
	if searchResp.Error != "" {
		return bodyBytes, fmt.Errorf("执行前查询当前行失败: %s", searchResp.Error)
	}

	oldRow := extractFirstItemFromSearchResult(searchResp.Result)
	if oldRow == nil {
		return bodyBytes, fmt.Errorf("执行前查询当前行失败: 未找到 id=%d 的记录", rowID)
	}

	bodyData["old_values"] = oldRow
	newBody, err := json.Marshal(bodyData)
	if err != nil {
		return bodyBytes, fmt.Errorf("回填 old_values 后序列化失败: %w", err)
	}
	return newBody, nil
}

func extractFirstItemFromSearchResult(result interface{}) map[string]interface{} {
	if result == nil {
		return nil
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}
	items, ok := m["items"].([]interface{})
	if !ok || len(items) == 0 {
		return nil
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return nil
	}
	return first
}

func (s *ScheduledTaskService) buildTableCallbackReq(user, appName, routerPath, method, callbackType, traceID, token, requestUser, requestUserDept string, bodyBytes []byte) (*dto.RequestAppReq, error) {
	callbackBody := map[string]interface{}{
		"type":   callbackType,
		"method": method,
		"router": routerPath,
		"body":   bodyBytes,
	}
	marshalBody, err := json.Marshal(callbackBody)
	if err != nil {
		return nil, fmt.Errorf("构建 table 回调请求失败: %w", err)
	}

	return &dto.RequestAppReq{
		User:            user,
		App:             appName,
		Router:          "/_callback",
		Method:          method,
		Body:            marshalBody,
		RequestUser:     requestUser,
		RequestUserDept: requestUserDept,
		TraceId:         traceID,
		Token:           token,
	}, nil
}

func (s *ScheduledTaskService) finishExecution(ctx context.Context, task *model.ScheduledTask, requestPayload, responsePayload []byte, status, errMsg, traceID string, executedAt time.Time, durationMillis int64, scheduledAt time.Time, success bool) {
	exec := s.recordExecution(ctx, task, requestPayload, responsePayload, status, errMsg, traceID, executedAt, durationMillis)
	s.updateTaskAfterRun(task, task.LeaseOwner, scheduledAt, success, errMsg)
	if exec != nil {
		s.notifyExecutionFinished(ctx, task, exec, success)
	}
}

func (s *ScheduledTaskService) recordExecution(ctx context.Context, task *model.ScheduledTask, requestPayload, responsePayload []byte, status, errMsg, traceID string, executedAt time.Time, durationMillis int64) *model.ScheduledTaskExecution {
	exec := &model.ScheduledTaskExecution{
		TaskID:          task.ID,
		ExecutedAt:      executedAt,
		Status:          status,
		DurationMillis:  durationMillis,
		RequestPayload:  requestPayload,
		ResponsePayload: responsePayload,
		ErrorMessage:    errMsg,
		TraceID:         traceID,
	}
	if err := s.executionRepo.Create(exec); err != nil {
		logger.Errorf(ctx, "[ScheduledTask] Create execution record err: %v", err)
		return nil
	}
	return exec
}

func (s *ScheduledTaskService) notifyExecutionFinished(ctx context.Context, task *model.ScheduledTask, exec *model.ScheduledTaskExecution, success bool) {
	if !shouldNotifyScheduledTask(task, success) {
		return
	}
	if s.options.MessagePublisher == nil {
		logger.Warnf(ctx, "[ScheduledTask] Message publisher unavailable, skip notification: task=%d execution=%d", task.ID, exec.ID)
		return
	}

	resultLink := s.buildExecutionResultURL(task, exec)
	replayLink := ""
	if isScheduledTaskFormFunction(task) {
		replayLink = s.buildExecutionReplayURL(task, exec)
	}
	traceLink := s.buildExecutionTraceURL(task, exec)
	statusLabel := "成功"
	if !success {
		statusLabel = "失败"
	}
	title := fmt.Sprintf("定时任务执行%s：%s", statusLabel, scheduledTaskDisplayName(task))
	content := buildScheduledTaskNotificationContent(task, exec, statusLabel, resultLink, replayLink, traceLink)

	from := strings.TrimSpace(task.RequestUser)
	if from == "" {
		from = strings.TrimSpace(task.CreatedBy)
	}
	if from == "" {
		from = "scheduled-task"
	}

	payload := &dto.MessageSendPayload{
		From:          from,
		FullCodePath:  task.FullCodePath,
		ToUsers:       task.NotifyUsers,
		ToDepartments: task.NotifyDepartments,
		Title:         title,
		Content:       content,
		ContentType:   "markdown",
	}
	if err := s.options.MessagePublisher.PublishMessage(ctx, payload); err != nil {
		logger.Errorf(ctx, "[ScheduledTask] Publish notification failed: task=%d execution=%d err=%v", task.ID, exec.ID, err)
	}
}

func shouldNotifyScheduledTask(task *model.ScheduledTask, success bool) bool {
	if task == nil {
		return false
	}
	if strings.TrimSpace(task.NotifyUsers) == "" && strings.TrimSpace(task.NotifyDepartments) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(task.NotifyOn)) {
	case ScheduledTaskNotifyAll, "":
		return true
	case ScheduledTaskNotifySuccess:
		return success
	case ScheduledTaskNotifyFailed:
		return !success
	default:
		return false
	}
}

func scheduledTaskDisplayName(task *model.ScheduledTask) string {
	if task == nil {
		return "未命名任务"
	}
	if name := strings.TrimSpace(task.Name); name != "" {
		return name
	}
	return "未命名任务"
}

func isScheduledTaskFormFunction(task *model.ScheduledTask) bool {
	if task == nil {
		return false
	}
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(task.FullCodePath)), ".form")
}

func buildScheduledTaskNotificationContent(task *model.ScheduledTask, exec *model.ScheduledTaskExecution, statusLabel string, resultLink string, replayLink string, traceLink string) string {
	var b strings.Builder
	b.WriteString("## 定时任务执行")
	b.WriteString(statusLabel)
	b.WriteString("\n\n")
	b.WriteString("- 任务：")
	b.WriteString(scheduledTaskDisplayName(task))
	b.WriteString("\n")
	b.WriteString("- 函数：`")
	b.WriteString(task.FullCodePath)
	b.WriteString("`\n")
	b.WriteString("- 执行时间：")
	b.WriteString(exec.ExecutedAt.Format(time.RFC3339))
	b.WriteString("\n")
	b.WriteString("- 耗时：")
	b.WriteString(strconv.FormatInt(exec.DurationMillis, 10))
	b.WriteString(" ms\n")
	if exec.TraceID != "" {
		b.WriteString("- Trace ID：`")
		b.WriteString(exec.TraceID)
		b.WriteString("`\n")
	}
	if exec.ErrorMessage != "" {
		b.WriteString("- 错误：")
		b.WriteString(exec.ErrorMessage)
		b.WriteString("\n")
	}
	if resultLink != "" {
		b.WriteString("\n[查看执行结果](")
		b.WriteString(resultLink)
		b.WriteString(")\n")
	}
	if replayLink != "" {
		b.WriteString("[回填到表单](")
		b.WriteString(replayLink)
		b.WriteString(")\n")
	}
	if traceLink != "" {
		b.WriteString("[查看函数执行记录](")
		b.WriteString(traceLink)
		b.WriteString(")\n")
	}
	return b.String()
}

func (s *ScheduledTaskService) buildExecutionResultURL(task *model.ScheduledTask, exec *model.ScheduledTaskExecution) string {
	query := url.Values{}
	query.Set("_panel", "scheduledTask")
	query.Set("_scheduled_task_id", strconv.FormatInt(task.ID, 10))
	query.Set("_scheduled_execution_id", strconv.FormatInt(exec.ID, 10))
	return s.buildWorkspaceFunctionURL(task, query)
}

func (s *ScheduledTaskService) buildExecutionReplayURL(task *model.ScheduledTask, exec *model.ScheduledTaskExecution) string {
	query := url.Values{}
	query.Set("_replay", "scheduled_execution")
	query.Set("_scheduled_task_id", strconv.FormatInt(task.ID, 10))
	query.Set("_scheduled_execution_id", strconv.FormatInt(exec.ID, 10))
	return s.buildWorkspaceFunctionURL(task, query)
}

func (s *ScheduledTaskService) buildExecutionTraceURL(task *model.ScheduledTask, exec *model.ScheduledTaskExecution) string {
	if strings.TrimSpace(exec.TraceID) == "" {
		return ""
	}
	query := url.Values{}
	query.Set("_panel", "operateLog")
	query.Set("_trace_id", exec.TraceID)
	return s.buildWorkspaceFunctionURL(task, query)
}

func (s *ScheduledTaskService) buildWorkspaceFunctionURL(task *model.ScheduledTask, query url.Values) string {
	workspacePath := "/workspace" + escapeScheduledTaskFullCodePath(task.FullCodePath)

	resultPath := workspacePath
	if encoded := query.Encode(); encoded != "" {
		resultPath += "?" + encoded
	}

	baseURL := strings.TrimRight(strings.TrimSpace(s.options.NotificationBaseURL), "/")
	if baseURL == "" {
		return resultPath
	}
	return baseURL + resultPath
}

func escapeScheduledTaskFullCodePath(fullCodePath string) string {
	trimmed := strings.Trim(fullCodePath, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return "/" + strings.Join(parts, "/")
}

func (s *ScheduledTaskService) incrementScheduledFunctionRunCount(ctx context.Context, fullCodePath string) {
	if s.appClient == nil {
		return
	}
	s.appClient.IncrementFunctionRunCount(ctx, normalizeScheduledTaskFullCodePath(fullCodePath))
}

func (s *ScheduledTaskService) recordScheduledTableOperateLog(ctx context.Context, task *model.ScheduledTask, user, appName, routerPath, callbackType string, bodyBytes []byte, traceID string) {
	if s.appClient == nil || callbackType == "" {
		return
	}
	logReq := &dto.RecordTableOperateLogReq{
		TenantUser:  user,
		RequestUser: strings.TrimSpace(task.RequestUser),
		App:         appName,
		Router:      routerPath,
		Action:      callbackType,
		Source:      "scheduled_task",
		IPAddress:   "scheduled-task",
		UserAgent:   "scheduled-task",
		TraceID:     traceID,
	}
	if logReq.RequestUser == "" {
		logReq.RequestUser = strings.TrimSpace(task.CreatedBy)
	}

	var bodyData map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &bodyData)
	switch callbackType {
	case "OnTableAddRow":
		logReq.Body = append(json.RawMessage(nil), bodyBytes...)
	case "OnTableUpdateRow":
		if id, ok := scheduledTaskBodyIDInt64(bodyData); ok {
			logReq.RowID = id
		}
		if updatesData, ok := bodyData["updates"].(map[string]interface{}); ok {
			logReq.Updates, _ = json.Marshal(updatesData)
		}
		if oldValuesData, ok := bodyData["old_values"].(map[string]interface{}); ok {
			logReq.OldValues, _ = json.Marshal(oldValuesData)
		}
	case "OnTableDeleteRows":
		logReq.RowIDs = scheduledTaskBodyIDs(bodyData)
	}

	if err := s.appClient.RecordTableOperateLog(ctx, logReq); err != nil {
		logger.Warnf(ctx, "[ScheduledTask] Record table operate log failed: task=%d action=%s err=%v", task.ID, callbackType, err)
	}
}

func scheduledTaskBodyIDInt64(bodyData map[string]interface{}) (int64, bool) {
	if bodyData == nil {
		return 0, false
	}
	idValue, ok := bodyData["id"]
	if !ok {
		return 0, false
	}
	switch v := idValue.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return id, err == nil
	default:
		return 0, false
	}
}

func scheduledTaskBodyIDs(bodyData map[string]interface{}) []int64 {
	if bodyData == nil {
		return nil
	}
	rawIDs, ok := bodyData["ids"].([]interface{})
	if !ok {
		return nil
	}
	rowIDs := make([]int64, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		switch v := rawID.(type) {
		case float64:
			rowIDs = append(rowIDs, int64(v))
		case int64:
			rowIDs = append(rowIDs, v)
		case int:
			rowIDs = append(rowIDs, int64(v))
		case string:
			if id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
				rowIDs = append(rowIDs, id)
			}
		}
	}
	return rowIDs
}

func (s *ScheduledTaskService) recordFunctionOperateLog(ctx context.Context, task *model.ScheduledTask, function *model.Function, user, appName, routerPath string, requestPayload []byte, resp *dto.RequestAppResp, requestErr error, traceID string, durationMillis int64) {
	if s.appClient == nil {
		return
	}
	action, err := normalizeScheduledTaskAction(task.Action)
	if err != nil || action != ScheduledTaskActionExecute {
		return
	}
	if scheduledTaskTemplateType(function) != functionschema.TypeForm {
		return
	}

	logReq := &dto.RecordFormOperateLogReq{
		TenantUser:     user,
		RequestUser:    strings.TrimSpace(task.RequestUser),
		App:            appName,
		Router:         routerPath,
		Source:         "scheduled_task",
		Action:         "form_submit",
		FunctionMethod: task.Method,
		RequestBody:    requestPayload,
		ResponseBody:   buildScheduledTaskOperateLogResponseBody(resp, requestErr, durationMillis),
		UserAgent:      "scheduled-task",
		TraceID:        traceID,
	}
	if resp != nil {
		logReq.Version = resp.Version
	}
	if logReq.RequestUser == "" {
		logReq.RequestUser = strings.TrimSpace(task.CreatedBy)
	}

	if err := s.appClient.RecordFormOperateLog(ctx, logReq); err != nil {
		logger.Warnf(ctx, "[ScheduledTask] Record form operate log failed: task=%d err=%v", task.ID, err)
	}
}

func (s *ScheduledTaskService) HealthStatus(now time.Time) (bool, time.Duration) {
	lastHeartbeatUnix := s.lastHeartbeat.Load()
	if lastHeartbeatUnix <= 0 {
		return false, 0
	}

	lastHeartbeatAt := time.Unix(lastHeartbeatUnix, 0)
	age := now.Sub(lastHeartbeatAt)
	if age < 0 {
		age = 0
	}
	return age <= s.options.HeartbeatMaxAge, age
}

func (s *ScheduledTaskService) HealthSnapshot(now time.Time) ScheduledTaskHealthSnapshot {
	healthy, age := s.HealthStatus(now)
	maxConcurrency := cap(s.workerSlots)
	if maxConcurrency <= 0 {
		maxConcurrency = s.options.MaxConcurrency
	}
	inflightWorkers := len(s.workerSlots)
	availableWorkers := maxConcurrency - inflightWorkers
	if availableWorkers < 0 {
		availableWorkers = 0
	}

	snapshot := ScheduledTaskHealthSnapshot{
		Healthy:          healthy,
		SchedulerID:      s.schedulerID,
		HeartbeatAge:     age,
		HeartbeatMaxAge:  s.options.HeartbeatMaxAge,
		PollInterval:     s.options.PollInterval,
		InflightWorkers:  inflightWorkers,
		AvailableWorkers: availableWorkers,
		MaxConcurrency:   maxConcurrency,
	}

	if lastHeartbeatUnix := s.lastHeartbeat.Load(); lastHeartbeatUnix > 0 {
		snapshot.HasHeartbeat = true
		snapshot.LastHeartbeatAt = time.Unix(lastHeartbeatUnix, 0)
	}
	if lastPollUnix := s.lastPollAt.Load(); lastPollUnix > 0 {
		snapshot.HasPoll = true
		snapshot.LastPollAt = time.Unix(lastPollUnix, 0)
	}

	return snapshot
}

func (s *ScheduledTaskService) markPollCompleted(now time.Time) {
	s.lastPollAt.Store(now.Unix())
}

func (s *ScheduledTaskService) writeHeartbeat(ctx context.Context, now time.Time) {
	path := strings.TrimSpace(s.options.HeartbeatFile)
	s.lastHeartbeat.Store(now.Unix())
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logger.Warnf(ctx, "[ScheduledTask] Ensure heartbeat dir failed: file=%s err=%v", path, err)
		return
	}
	data := []byte(strconv.FormatInt(now.Unix(), 10))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		logger.Warnf(ctx, "[ScheduledTask] Write heartbeat failed: file=%s err=%v", path, err)
	}
}

func buildScheduledTaskOperateLogResponseBody(resp *dto.RequestAppResp, err error, totalCostMill int64) []byte {
	payload := map[string]interface{}{
		"code": 0,
	}
	if totalCostMill >= 0 {
		payload["total_cost_mill"] = totalCostMill
	}

	switch {
	case resp != nil:
		payload["code"] = resp.ErrCode
		if resp.TraceId != "" {
			payload["trace_id"] = resp.TraceId
		}
		if resp.Version != "" {
			payload["version"] = resp.Version
		}
		if resp.Result != nil {
			payload["result"] = resp.Result
		}
		if resp.Error != "" {
			payload["msg"] = resp.Error
			payload["error"] = resp.Error
		}
	case err != nil:
		payload["code"] = 1
		payload["msg"] = err.Error()
		payload["error"] = err.Error()
	}

	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		fallback := map[string]interface{}{
			"code": 1,
			"msg":  "marshal scheduled task operate log response failed",
		}
		if err != nil {
			fallback["msg"] = err.Error()
			fallback["error"] = err.Error()
		}
		data, _ = json.Marshal(fallback)
	}
	return data
}

func (s *ScheduledTaskService) updateTaskAfterRun(task *model.ScheduledTask, leaseOwner string, scheduledAt time.Time, success bool, errMsg string) {
	task.RunCount++
	task.ErrorMessage = errMsg
	task.LeaseOwner = ""
	task.LeaseUntil = nil
	task.Status, task.NextRunAt, task.ErrorMessage = computeTaskNextState(task, scheduledAt, success)
	updated, err := s.taskRepo.UpdateAfterRun(task, leaseOwner)
	if err != nil {
		logger.Errorf(context.Background(), "[ScheduledTask] Update task err: %v", err)
	} else if !updated {
		logger.Warnf(context.Background(), "[ScheduledTask] Skip stale run result: task=%d worker=%s", task.ID, leaseOwner)
	}
}

// parseRunAt 兼容解析 run_at：
// 1) RFC3339（带时区）
// 2) 本地时间字符串（不带时区，按 time.Local 解析），例如：
//   - 2006-01-02 15:04:05
//   - 2006-01-02 15:04
//   - 2006-01-02T15:04:05
//   - 2006-01-02T15:04
func parseRunAt(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("run_at 不能为空")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	localLayouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}
	for _, layout := range localLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("请使用 RFC3339（如 2026-03-20T23:00:00+08:00）或本地时间（如 2026-03-20 23:00:00）")
}
