package v1

import (
	"strconv"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
	"github.com/gin-gonic/gin"
)

type ScheduledAgentTask struct {
	service *service.ScheduledAgentTaskService
}

func NewScheduledAgentTask(service *service.ScheduledAgentTaskService) *ScheduledAgentTask {
	return &ScheduledAgentTask{service: service}
}

func buildScheduledAgentTaskItem(t *model.ScheduledAgentTask) dto.ScheduledAgentTaskItem {
	return dto.ScheduledAgentTaskItem{
		ID:                t.ID,
		Name:              t.Name,
		FullCodePath:      t.FullCodePath,
		Goal:              t.Goal,
		ModeCode:          t.ModeCode,
		Files:             t.Files,
		LLMConfigID:       t.LLMConfigID,
		ContextPolicy:     t.ContextPolicy,
		ToolPolicy:        t.ToolPolicy,
		ApprovalPolicy:    t.ApprovalPolicy,
		BudgetPolicy:      t.BudgetPolicy,
		ScheduleType:      t.ScheduleType,
		RunAt:             t.RunAt,
		NextRunAt:         t.NextRunAt,
		CronExpr:          t.CronExpr,
		IntervalSeconds:   t.IntervalSeconds,
		MaxRuns:           t.MaxRuns,
		Timezone:          t.Timezone,
		Status:            t.Status,
		TimerTaskID:       t.TimerTaskID,
		RunCount:          t.RunCount,
		LastSessionID:     t.LastSessionID,
		LastExecutionID:   t.LastExecutionID,
		LastErrorMessage:  t.LastErrorMessage,
		RequestUser:       t.RequestUser,
		RequestUserDept:   t.RequestUserDept,
		NotifyUsers:       service.SplitScheduledAgentRecipientsForAPI(t.NotifyUsers),
		NotifyDepartments: service.SplitScheduledAgentRecipientsForAPI(t.NotifyDepartments),
		NotifyOn:          t.NotifyOn,
		SourceType:        t.SourceType,
		SourceRef:         t.SourceRef,
		CreatedBy:         t.CreatedBy,
		CreatedAt:         time.Time(t.CreatedAt),
		UpdatedAt:         time.Time(t.UpdatedAt),
	}
}

func buildScheduledAgentExecutionItem(e *model.ScheduledAgentExecution) dto.ScheduledAgentExecutionItem {
	return dto.ScheduledAgentExecutionItem{
		ID:             e.ID,
		TaskID:         e.TaskID,
		SessionID:      e.SessionID,
		ScheduledAt:    e.ScheduledAt,
		StartedAt:      e.StartedAt,
		FinishedAt:     e.FinishedAt,
		Status:         e.Status,
		WorkerID:       e.WorkerID,
		DurationMillis: e.DurationMillis,
		InputGoal:      e.InputGoal,
		OutputSummary:  e.OutputSummary,
		ToolCallCount:  e.ToolCallCount,
		TokenUsage:     e.TokenUsage,
		ErrorMessage:   e.ErrorMessage,
		TraceID:        e.TraceID,
		SourceType:     e.SourceType,
		SourceRef:      e.SourceRef,
		CreatedAt:      time.Time(e.CreatedAt),
		UpdatedAt:      time.Time(e.UpdatedAt),
	}
}

func buildScheduledAgentRunNowResp(taskID int64, e *scheduledsdk.Execution) dto.ScheduledAgentRunNowResp {
	resp := dto.ScheduledAgentRunNowResp{TaskID: taskID}
	if e != nil {
		resp.TimerTaskID = e.TaskID
		resp.TimerExecutionID = e.ID
		resp.Status = string(e.Status)
		resp.ScheduledAt = e.ScheduledAt
	}
	return resp
}

func parseScheduledAgentID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的 ID")
		return 0, false
	}
	return id, true
}

func (h *ScheduledAgentTask) Create(c *gin.Context) {
	var req dto.CreateScheduledAgentTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	requestUser := contextx.GetRequestUser(ctx)
	task, err := h.service.Create(ctx, &req, requestUser)
	if err != nil {
		response.FailWithMessage(c, "创建失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledAgentTaskItem(task))
}

func (h *ScheduledAgentTask) List(c *gin.Context) {
	ctx := contextx.ToContext(c)
	requestUser := contextx.GetRequestUser(ctx)
	if requestUser == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := h.service.List(ctx, requestUser, c.Query("status"), c.Query("full_code_path"), page, pageSize)
	if err != nil {
		response.FailWithMessage(c, "获取列表失败: "+err.Error())
		return
	}
	items := make([]dto.ScheduledAgentTaskItem, 0, len(list))
	for _, task := range list {
		items = append(items, buildScheduledAgentTaskItem(task))
	}
	response.OkWithData(c, gin.H{"list": items, "total": total})
}

func (h *ScheduledAgentTask) Get(c *gin.Context) {
	id, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	task, err := h.service.Get(ctx, id, contextx.GetRequestUser(ctx))
	if err != nil {
		response.FailWithMessage(c, "获取任务失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledAgentTaskItem(task))
}

func (h *ScheduledAgentTask) Update(c *gin.Context) {
	id, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateScheduledAgentTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	task, err := h.service.Update(ctx, id, &req, contextx.GetRequestUser(ctx))
	if err != nil {
		response.FailWithMessage(c, "更新失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledAgentTaskItem(task))
}

func (h *ScheduledAgentTask) Delete(c *gin.Context) {
	id, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	if err := h.service.Delete(ctx, id, contextx.GetRequestUser(ctx)); err != nil {
		response.FailWithMessage(c, "删除失败: "+err.Error())
		return
	}
	response.OkWithMessage(c, "已删除")
}

func (h *ScheduledAgentTask) RunNow(c *gin.Context) {
	id, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	exec, err := h.service.RunNow(ctx, id, contextx.GetRequestUser(ctx))
	if err != nil {
		response.FailWithMessage(c, "启动失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledAgentRunNowResp(id, exec))
}

func (h *ScheduledAgentTask) Pause(c *gin.Context) {
	id, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	if err := h.service.Pause(ctx, id, contextx.GetRequestUser(ctx)); err != nil {
		response.FailWithMessage(c, "暂停失败: "+err.Error())
		return
	}
	response.OkWithMessage(c, "已暂停")
}

func (h *ScheduledAgentTask) Resume(c *gin.Context) {
	id, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	if err := h.service.Resume(ctx, id, contextx.GetRequestUser(ctx)); err != nil {
		response.FailWithMessage(c, "恢复失败: "+err.Error())
		return
	}
	response.OkWithMessage(c, "已恢复")
}

func (h *ScheduledAgentTask) ListExecutions(c *gin.Context) {
	taskID, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := h.service.ListExecutions(ctx, taskID, contextx.GetRequestUser(ctx), c.Query("status"), page, pageSize)
	if err != nil {
		response.FailWithMessage(c, "获取执行记录失败: "+err.Error())
		return
	}
	items := make([]dto.ScheduledAgentExecutionItem, 0, len(list))
	for _, exec := range list {
		items = append(items, buildScheduledAgentExecutionItem(exec))
	}
	response.OkWithData(c, gin.H{"list": items, "total": total})
}

func (h *ScheduledAgentTask) GetExecution(c *gin.Context) {
	taskID, ok := parseScheduledAgentID(c, "id")
	if !ok {
		return
	}
	executionID, ok := parseScheduledAgentID(c, "execution_id")
	if !ok {
		return
	}
	ctx := contextx.ToContext(c)
	exec, err := h.service.GetExecution(ctx, taskID, executionID, contextx.GetRequestUser(ctx))
	if err != nil {
		response.FailWithMessage(c, "获取执行记录失败: "+err.Error())
		return
	}
	response.OkWithData(c, buildScheduledAgentExecutionItem(exec))
}
