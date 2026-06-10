package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serverx"
)

func NewRouter(service *timerservice.Service) *gin.Engine {
	router := serverx.NewGin(serverx.WithRecovery())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "timer-scheduler"})
	})

	api := router.Group("/timer/api/v1")
	api.POST("/tasks", createTask(service))
	api.GET("/tasks", listTasks(service))
	api.GET("/tasks/:id", getTask(service))
	api.PUT("/tasks/:id", updateTask(service))
	api.POST("/tasks/:id/pause", pauseTask(service))
	api.POST("/tasks/:id/resume", resumeTask(service))
	api.POST("/tasks/:id/cancel", cancelTask(service))
	api.POST("/tasks/:id/run_now", runNow(service))
	api.GET("/tasks/:id/executions", listExecutions(service))
	api.GET("/tasks/:id/executions/:execution_id", getExecution(service))
	api.POST("/executions/started", markExecutionStarted(service))
	api.POST("/executions/heartbeat", markExecutionHeartbeat(service))
	api.POST("/executions/finished", markExecutionFinished(service))
	return router
}

func createTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.CreateTaskRequest
		if !bindJSON(c, &req) {
			return
		}
		task, err := service.CreateTask(c.Request.Context(), req)
		writeResult(c, task, err)
	}
}

func listTasks(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := service.ListTasks(c.Request.Context(), scheduledsdk.ListTasksRequest{
			ExecutorKey:   c.Query("executor_key"),
			Status:        c.Query("status"),
			Category:      c.Query("category"),
			SourceType:    c.Query("source_type"),
			SourceRef:     c.Query("source_ref"),
			ResourceScope: c.Query("resource_scope"),
			ResourceKey:   c.Query("resource_key"),
			CreatedBy:     c.Query("created_by"),
			Page:          queryInt(c, "page"),
			PageSize:      queryInt(c, "page_size"),
		})
		writeResult(c, resp, err)
	}
}

func getTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		task, err := service.GetTask(c.Request.Context(), id)
		writeResult(c, task, err)
	}
}

func updateTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		var req scheduledsdk.UpdateTaskRequest
		if !bindJSON(c, &req) {
			return
		}
		task, err := service.UpdateTask(c.Request.Context(), id, req)
		writeResult(c, task, err)
	}
}

func pauseTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.PauseTask(c.Request.Context(), id))
	}
}

func resumeTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.ResumeTask(c.Request.Context(), id))
	}
}

func cancelTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.CancelTask(c.Request.Context(), id))
	}
}

func runNow(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		exec, err := service.RunNow(c.Request.Context(), id)
		writeResult(c, exec, err)
	}
}

func listExecutions(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		resp, err := service.ListExecutions(c.Request.Context(), taskID, scheduledsdk.ListExecutionsRequest{
			Status:   c.Query("status"),
			Page:     queryInt(c, "page"),
			PageSize: queryInt(c, "page_size"),
		})
		writeResult(c, resp, err)
	}
}

func getExecution(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		executionID, ok := pathInt64(c, "execution_id")
		if !ok {
			return
		}
		exec, err := service.GetExecution(c.Request.Context(), taskID, executionID)
		writeResult(c, exec, err)
	}
}

func markExecutionStarted(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.MarkExecutionStartedRequest
		if !bindJSON(c, &req) {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.MarkExecutionStarted(c.Request.Context(), req))
	}
}

func markExecutionHeartbeat(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.MarkExecutionHeartbeatRequest
		if !bindJSON(c, &req) {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.MarkExecutionHeartbeat(c.Request.Context(), req))
	}
}

func markExecutionFinished(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.MarkExecutionFinishedRequest
		if !bindJSON(c, &req) {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.MarkExecutionFinished(c.Request.Context(), req))
	}
}

func bindJSON(c *gin.Context, out interface{}) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func writeResult(c *gin.Context, data interface{}, err error) {
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func pathInt64(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " is invalid"})
		return 0, false
	}
	return id, true
}

func queryInt(c *gin.Context, name string) int {
	value, _ := strconv.Atoi(c.Query(name))
	return value
}
