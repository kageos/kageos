package v1

import (
	"strconv"
	"time"

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
	response.OkWithData(c, task)
}

// List 定时任务列表（当前用户；可选 query full_code_path：按路径前缀过滤，返回该路径及子路径下的任务）
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
		item := dto.ScheduledTaskItem{
			ID:           t.ID,
			Name:         t.Name,
			User:         t.User,
			App:          t.App,
			FullCodePath: t.FullCodePath,
			Action:       t.Action,
			CreatedBy:    t.CreatedBy,
			ScheduleType: t.ScheduleType,
			RunAt:        t.RunAt.Format(time.RFC3339),
			Status:       t.Status,
			RunCount:     t.RunCount,
			ErrorMessage: t.ErrorMessage,
			CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		}
		if t.NextRunAt != nil {
			next := t.NextRunAt.Format(time.RFC3339)
			item.NextRunAt = &next
		}
		items = append(items, item)
	}
	response.OkWithData(c, gin.H{"list": items, "total": total})
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
		items = append(items, dto.ScheduledTaskExecutionItem{
			ID:              e.ID,
			TaskID:          e.TaskID,
			ExecutedAt:      e.ExecutedAt.Format(time.RFC3339),
			Status:          e.Status,
			RequestPayload:  string(e.RequestPayload),
			ResponsePayload: string(e.ResponsePayload),
			ErrorMessage:    e.ErrorMessage,
			TraceID:         e.TraceID,
			CreatedAt:       e.CreatedAt.Format(time.RFC3339),
		})
	}
	response.OkWithData(c, gin.H{"list": items, "total": total})
}
