package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const (
	ScheduledTaskActionExecute     = "execute"
	ScheduledTaskActionTableCreate = "table_create"
	ScheduledTaskActionTableUpdate = "table_update"
	ScheduledTaskActionTableDelete = "table_delete"
)

// ScheduledTaskService 定时任务服务
type ScheduledTaskService struct {
	db            *gorm.DB
	appService    *AppService
	jwtService    *JWTService
	taskRepo      *repository.ScheduledTaskRepository
	executionRepo *repository.ScheduledTaskExecutionRepository
	dueQueue      DueQueue // 按 next_run_at 排序的队列，兜底到点执行（类似 Redis ZSET）
}

func NewScheduledTaskService(
	db *gorm.DB,
	appService *AppService,
	jwtService *JWTService,
	taskRepo *repository.ScheduledTaskRepository,
	executionRepo *repository.ScheduledTaskExecutionRepository,
) *ScheduledTaskService {
	return &ScheduledTaskService{
		db:            db,
		appService:    appService,
		jwtService:    jwtService,
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		dueQueue:      NewMemDueQueue(),
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

// Create 创建定时任务，计算 next_run_at
func (s *ScheduledTaskService) Create(ctx context.Context, req *dto.CreateScheduledTaskReq, requestUser string) (*model.ScheduledTask, error) {
	user, appName, _, err := parseFullCodePath(req.FullCodePath)
	if err != nil {
		return nil, err
	}
	runAt, err := parseRunAt(req.RunAt)
	if err != nil {
		return nil, fmt.Errorf("run_at 格式错误: %w", err)
	}
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "POST"
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = ScheduledTaskActionExecute
	}
	switch action {
	case ScheduledTaskActionExecute, ScheduledTaskActionTableCreate, ScheduledTaskActionTableUpdate, ScheduledTaskActionTableDelete:
	default:
		return nil, fmt.Errorf("action 不支持，必须是 execute/table_create/table_update/table_delete")
	}
	reqUser := strings.TrimSpace(req.RequestUser)
	if reqUser == "" {
		reqUser = requestUser
	}
	reqUserDept := strings.TrimSpace(req.RequestUserDept)
	if reqUserDept == "" {
		reqUserDept = contextx.GetRequestDepartmentFullPath(ctx)
	}

	task := &model.ScheduledTask{
		User:            user,
		App:             appName,
		Name:            strings.TrimSpace(req.Name),
		FullCodePath:    req.FullCodePath,
		Action:          action,
		Method:          method,
		Payload:         req.Payload,
		RequestUser:     reqUser,
		RequestUserDept: reqUserDept,
		CreatedBy:       requestUser,
		ScheduleType:    req.ScheduleType,
		RunAt:           runAt,
		CronExpr:        req.CronExpr,
		IntervalSeconds: req.IntervalSeconds,
		MaxRuns:         req.MaxRuns,
		Status:          "pending",
		Timezone:        req.Timezone,
	}

	switch req.ScheduleType {
	case "atime":
		task.NextRunAt = &runAt
	case "cron":
		if req.CronExpr == "" {
			return nil, fmt.Errorf("cron 类型需提供 cron_expr")
		}
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		sch, err := parser.Parse(req.CronExpr)
		if err != nil {
			return nil, fmt.Errorf("cron_expr 解析失败: %w", err)
		}
		next := sch.Next(runAt)
		task.NextRunAt = &next
	case "every":
		if req.IntervalSeconds <= 0 {
			return nil, fmt.Errorf("every 类型需提供 interval_seconds > 0")
		}
		task.IntervalSeconds = req.IntervalSeconds
		task.MaxRuns = req.MaxRuns
		task.NextRunAt = &runAt
	default:
		return nil, fmt.Errorf("schedule_type 必须是 atime/cron/every")
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	if task.NextRunAt != nil {
		s.dueQueue.Push(task.ID, *task.NextRunAt)
	}
	return task, nil
}

// List 分页列表（当前用户创建的任务；可选按 full_code_path 前缀过滤：含该节点与子节点）
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

// Cancel 取消任务（仅创建人可取消）
func (s *ScheduledTaskService) Cancel(ctx context.Context, id int64, createdBy string) error {
	if err := s.taskRepo.Cancel(id, createdBy); err != nil {
		return err
	}
	s.dueQueue.Remove(id)
	return nil
}

// ListExecutions 某任务的执行记录（仅创建人可查）
func (s *ScheduledTaskService) ListExecutions(ctx context.Context, taskID int64, createdBy string, status string, page, pageSize int) ([]*model.ScheduledTaskExecution, int64, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, 0, err
	}
	if task.CreatedBy != createdBy {
		return nil, 0, fmt.Errorf("无权查看该任务")
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

// StartScheduler 启动调度器，应在 server 启动时调用
// 使用 DueQueue（按 next_run_at 排序）：每次 tick 取 score<=now 的任务执行，避免固定间隔漏执行；
// tick 间隔 1 秒，到点任务最多延迟约 1 秒被执行
func (s *ScheduledTaskService) StartScheduler(ctx context.Context) {
	// 启动时从 DB 同步待执行任务到队列
	pending, err := s.taskRepo.ListAllPending()
	if err != nil {
		logger.Errorf(ctx, "[ScheduledTask] ListAllPending on start err: %v", err)
	} else {
		s.dueQueue.Sync(pending)
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDueTasks(ctx)
		}
	}
}

// runDueTasks 从 DueQueue 取已到点任务执行（next_run_at <= now），执行后若仍 pending 则把新 next_run_at 推回队列
func (s *ScheduledTaskService) runDueTasks(ctx context.Context) {
	now := time.Now()
	ids := s.dueQueue.PopDue(now)
	for _, id := range ids {
		task, err := s.taskRepo.GetByID(id)
		if err != nil || task == nil {
			continue
		}
		if task.Status != "pending" || task.NextRunAt == nil || task.NextRunAt.After(now) {
			// 可能已被取消或已执行，或 DB 与队列不一致，跳过
			continue
		}
		s.executeOne(ctx, task)
		// 执行后 task 已被 updateTaskAfterRun 更新；若仍待执行则推回队列
		if task.Status == "pending" && task.NextRunAt != nil {
			s.dueQueue.Push(task.ID, *task.NextRunAt)
		}
	}
}

// executeOne 执行一条任务：注入“请求用户”context、用 JWT 生成 Token，再调 RequestApp、写执行记录、更新任务
func (s *ScheduledTaskService) executeOne(ctx context.Context, task *model.ScheduledTask) {
	executedAt := time.Now()
	user, appName, routerPath, err := parseFullCodePath(task.FullCodePath)
	if err != nil {
		s.recordExecution(ctx, task, nil, nil, "failed", err.Error(), executedAt)
		s.updateTaskAfterRun(task, executedAt, false, err.Error())
		return
	}
	// 定时任务无 HTTP 请求，用 WithRequestInfo 一次性注入与 ToContext 一致的 context
	traceId := fmt.Sprintf("scheduled-%d-%d", task.ID, executedAt.UnixNano())
	var token string
	if s.jwtService != nil && task.RequestUser != "" {
		if t, err := s.jwtService.GenerateAccessTokenWithHR(0, task.RequestUser, "", task.RequestUserDept, ""); err == nil {
			token = t
		}
	}
	runCtx := contextx.WithRequestInfo(ctx, contextx.RequestInfo{
		TraceId:            traceId,
		RequestUser:        task.RequestUser,
		Token:              token,
		DepartmentFullPath: task.RequestUserDept,
	})
	// 若 Payload 是「JSON 字符串」（前端曾用 JSON.stringify(payload) 导致存成 "{\"a\":1}"），
	// 解一层再作为 body，否则表单侧 json.Unmarshal(body, &struct) 会报 cannot unmarshal string into Go value of type ...
	bodyBytes := task.Payload
	if len(task.Payload) >= 2 && task.Payload[0] == '"' {
		var inner string
		if err := json.Unmarshal(task.Payload, &inner); err == nil {
			bodyBytes = []byte(inner)
		}
	}
	req, err := s.buildTaskRequest(runCtx, task, user, appName, routerPath, traceId, token, bodyBytes)
	if err != nil {
		s.recordExecution(ctx, task, task.Payload, nil, "failed", err.Error(), executedAt)
		s.updateTaskAfterRun(task, executedAt, false, err.Error())
		return
	}

	resp, err := s.appService.RequestApp(runCtx, req)
	var respBody []byte
	if resp != nil {
		// 存完整响应（trace_id、result、error、err_code），便于区分「成功但 result 为空」与真正失败
		respBody, _ = json.Marshal(resp)
	}
	reqBody := req.Body
	if err != nil {
		s.recordExecution(ctx, task, reqBody, respBody, "failed", err.Error(), executedAt)
		s.updateTaskAfterRun(task, executedAt, false, err.Error())
		return
	}
	s.recordExecution(ctx, task, reqBody, respBody, "success", "", executedAt)
	s.updateTaskAfterRun(task, executedAt, true, "")
}

func (s *ScheduledTaskService) buildTaskRequest(ctx context.Context, task *model.ScheduledTask, user, appName, routerPath, traceID, token string, bodyBytes []byte) (*dto.RequestAppReq, error) {
	action := strings.ToLower(strings.TrimSpace(task.Action))
	if action == "" {
		action = ScheduledTaskActionExecute
	}

	switch action {
	case ScheduledTaskActionExecute:
		return &dto.RequestAppReq{
			User:            user,
			App:             appName,
			Router:          routerPath,
			Method:          task.Method,
			Body:            bodyBytes,
			RequestUser:     task.RequestUser,
			RequestUserDept: task.RequestUserDept,
			TraceId:         traceID,
			Token:           token,
		}, nil
	case ScheduledTaskActionTableCreate:
		return s.buildTableCallbackReq(user, appName, routerPath, task.Method, "OnTableAddRow", traceID, token, task.RequestUser, task.RequestUserDept, bodyBytes)
	case ScheduledTaskActionTableUpdate:
		bodyBytes, err := s.fillTableUpdateOldValues(ctx, task, user, appName, routerPath, traceID, token, bodyBytes)
		if err != nil {
			return nil, err
		}
		return s.buildTableCallbackReq(user, appName, routerPath, task.Method, "OnTableUpdateRow", traceID, token, task.RequestUser, task.RequestUserDept, bodyBytes)
	case ScheduledTaskActionTableDelete:
		return s.buildTableCallbackReq(user, appName, routerPath, task.Method, "OnTableDeleteRows", traceID, token, task.RequestUser, task.RequestUserDept, bodyBytes)
	default:
		return nil, fmt.Errorf("未知 action: %s", task.Action)
	}
}

// fillTableUpdateOldValues 在执行前实时补齐 old_values，避免创建任务时缓存的旧值变脏。
func (s *ScheduledTaskService) fillTableUpdateOldValues(ctx context.Context, task *model.ScheduledTask, user, appName, routerPath, traceID, token string, bodyBytes []byte) ([]byte, error) {
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
		RequestUser:     task.RequestUser,
		RequestUserDept: task.RequestUserDept,
		Token:           token,
	}

	searchResp, err := s.appService.RequestApp(ctx, searchReq)
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

func (s *ScheduledTaskService) recordExecution(ctx context.Context, task *model.ScheduledTask, requestPayload, responsePayload []byte, status, errMsg string, executedAt time.Time) {
	exec := &model.ScheduledTaskExecution{
		TaskID:          task.ID,
		ExecutedAt:      executedAt,
		Status:          status,
		RequestPayload:  requestPayload,
		ResponsePayload: responsePayload,
		ErrorMessage:    errMsg,
	}
	if err := s.executionRepo.Create(exec); err != nil {
		logger.Errorf(ctx, "[ScheduledTask] Create execution record err: %v", err)
	}
}

func (s *ScheduledTaskService) updateTaskAfterRun(task *model.ScheduledTask, executedAt time.Time, success bool, errMsg string) {
	task.RunCount++
	task.ErrorMessage = errMsg
	if !success {
		task.Status = "failed"
		task.NextRunAt = nil
		if err := s.taskRepo.Update(task); err != nil {
			logger.Errorf(context.Background(), "[ScheduledTask] Update task err: %v", err)
		}
		return
	}
	switch task.ScheduleType {
	case "atime":
		task.Status = "done"
		task.NextRunAt = nil
	case "cron":
		if task.CronExpr == "" {
			task.Status = "done"
			task.NextRunAt = nil
		} else {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			sch, err := parser.Parse(task.CronExpr)
			if err != nil {
				task.Status = "failed"
				task.ErrorMessage = err.Error()
				task.NextRunAt = nil
			} else {
				next := sch.Next(executedAt)
				if next.IsZero() {
					task.Status = "done"
					task.NextRunAt = nil
				} else {
					task.Status = "pending"
					task.NextRunAt = &next
				}
			}
		}
	case "every":
		if task.MaxRuns > 0 && task.RunCount >= task.MaxRuns {
			task.Status = "done"
			task.NextRunAt = nil
		} else {
			next := executedAt.Add(time.Duration(task.IntervalSeconds) * time.Second)
			task.Status = "pending"
			task.NextRunAt = &next
		}
	default:
		task.Status = "done"
		task.NextRunAt = nil
	}
	if err := s.taskRepo.Update(task); err != nil {
		logger.Errorf(context.Background(), "[ScheduledTask] Update task err: %v", err)
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
