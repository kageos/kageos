package v1

import (
	"strconv"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

// ScheduledTask 定时任务 API
type ScheduledTask struct {
	scheduledTaskService *service.ScheduledTaskService
}

// NewScheduledTask 创建定时任务处理器
func NewScheduledTask(scheduledTaskService *service.ScheduledTaskService) *ScheduledTask {
	return &ScheduledTask{scheduledTaskService: scheduledTaskService}
}

func buildScheduledTaskItem(t *model.ScheduledTask) dto.ScheduledTaskItem {
	item := dto.ScheduledTaskItem{
		ID:                t.ID,
		TimerTaskID:       t.TimerTaskID,
		Name:              t.Name,
		User:              t.User,
		App:               t.App,
		FullCodePath:      t.FullCodePath,
		Action:            t.Action,
		Method:            t.Method,
		Payload:           string(t.Payload),
		RequestUser:       t.RequestUser,
		RequestUserDept:   t.RequestUserDept,
		CreatedBy:         t.CreatedBy,
		ScheduleType:      t.ScheduleType,
		RunAt:             t.RunAt.Format(time.RFC3339),
		CronExpr:          t.CronExpr,
		IntervalSeconds:   t.IntervalSeconds,
		MaxRuns:           t.MaxRuns,
		Timezone:          t.Timezone,
		Status:            t.Status,
		RunCount:          t.RunCount,
		ErrorMessage:      t.ErrorMessage,
		NotifyUsers:       service.SplitScheduledTaskRecipientsForAPI(t.NotifyUsers),
		NotifyDepartments: service.SplitScheduledTaskRecipientsForAPI(t.NotifyDepartments),
		NotifyOn:          t.NotifyOn,
		CreatedAt:         t.CreatedAt.Format(time.RFC3339),
	}
	if t.NextRunAt != nil {
		next := t.NextRunAt.Format(time.RFC3339)
		item.NextRunAt = &next
	}
	return item
}

func buildScheduledTaskExecutionItem(e *model.ScheduledTaskExecution) dto.ScheduledTaskExecutionItem {
	return dto.ScheduledTaskExecutionItem{
		ID:              e.ID,
		TaskID:          e.TaskID,
		ExecutedAt:      e.ExecutedAt.Format(time.RFC3339),
		Status:          e.Status,
		DurationMillis:  e.DurationMillis,
		RequestPayload:  string(e.RequestPayload),
		ResponsePayload: string(e.ResponsePayload),
		ErrorMessage:    e.ErrorMessage,
		TraceID:         e.TraceID,
		CreatedAt:       e.CreatedAt.Format(time.RFC3339),
	}
}

// Create 创建定时任务
func (s *ScheduledTask) Create(c *gin.Context) {
	var req dto.CreateScheduledTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	requestUser := contextx.GetRequestUser(c)
	task, err := s.scheduledTaskService.Create(ctx, &req, requestUser)
	if err != nil {
		response.FailWithMessage(c, "创建失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledTaskItem(task))
}

// List 定时任务列表。
// 传 full_code_path 时返回该路径及子路径下的任务；不传时返回当前用户创建的任务。
func (s *ScheduledTask) List(c *gin.Context) {
	requestUser := contextx.GetRequestUser(c)
	if requestUser == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	status := c.Query("status")
	fullCodePath := c.Query("full_code_path")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ctx := contextx.ToContext(c)
	list, total, err := s.scheduledTaskService.List(ctx, requestUser, status, fullCodePath, page, pageSize)
	if err != nil {
		response.FailWithMessage(c, "获取列表失败: "+err.Error())
		return
	}
	items := make([]dto.ScheduledTaskItem, 0, len(list))
	for _, t := range list {
		items = append(items, buildScheduledTaskItem(t))
	}
	response.OkWithData(c, gin.H{"list": items, "total": total})
}

// Get 获取定时任务详情
func (s *ScheduledTask) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的任务ID")
		return
	}
	requestUser := contextx.GetRequestUser(c)
	if requestUser == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	ctx := contextx.ToContext(c)
	task, err := s.scheduledTaskService.Get(ctx, id, requestUser)
	if err != nil {
		response.FailWithMessage(c, "获取任务失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledTaskItem(task))
}

// Cancel 取消定时任务
func (s *ScheduledTask) Cancel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的任务ID")
		return
	}
	requestUser := contextx.GetRequestUser(c)
	if requestUser == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	ctx := contextx.ToContext(c)
	if err := s.scheduledTaskService.Cancel(ctx, id, requestUser); err != nil {
		response.FailWithMessage(c, "取消失败: "+err.Error())
		return
	}
	response.OkWithMessage(c, "已取消")
}

// Delete 删除定时任务
func (s *ScheduledTask) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的任务ID")
		return
	}
	requestUser := contextx.GetRequestUser(c)
	if requestUser == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	ctx := contextx.ToContext(c)
	if err := s.scheduledTaskService.Delete(ctx, id, requestUser); err != nil {
		response.FailWithMessage(c, "删除失败: "+err.Error())
		return
	}
	response.OkWithMessage(c, "已删除")
}

// ListExecutions 某任务的执行记录
func (s *ScheduledTask) ListExecutions(c *gin.Context) {
	idStr := c.Param("id")
	taskID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的任务ID")
		return
	}
	requestUser := contextx.GetRequestUser(c)
	if requestUser == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ctx := contextx.ToContext(c)
	list, total, err := s.scheduledTaskService.ListExecutions(ctx, taskID, requestUser, status, page, pageSize)
	if err != nil {
		response.FailWithMessage(c, "获取执行记录失败: "+err.Error())
		return
	}
	items := make([]dto.ScheduledTaskExecutionItem, 0, len(list))
	for _, e := range list {
		items = append(items, buildScheduledTaskExecutionItem(e))
	}
	response.OkWithData(c, gin.H{"list": items, "total": total})
}

// GetExecution 获取单条执行记录
func (s *ScheduledTask) GetExecution(c *gin.Context) {
	idStr := c.Param("id")
	taskID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的任务ID")
		return
	}
	executionIDStr := c.Param("execution_id")
	executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的执行记录ID")
		return
	}
	requestUser := contextx.GetRequestUser(c)
	if requestUser == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	ctx := contextx.ToContext(c)
	execution, err := s.scheduledTaskService.GetExecution(ctx, taskID, executionID, requestUser)
	if err != nil {
		response.FailWithMessage(c, "获取执行记录失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledTaskExecutionItem(execution))
}
