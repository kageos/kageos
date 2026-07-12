package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serverx"
)

func NewRouter(service *timerservice.Service, gatewayVerifiers ...*controlauth.Verifier) *gin.Engine {
	router := serverx.NewGin(
		serverx.WithRecovery(),
		serverx.WithRegisteredMiddlewares(serverx.ServiceTimerScheduler),
	)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "timer-scheduler"})
	})

	api := router.Group("/timer/api/v1", requireTimerAPIAuthentication(gatewayVerifiers...))
	api.POST("/tasks", createTask(service))
	api.GET("/tasks", listTasks(service))
	api.GET("/tasks/:id", getTask(service))
	api.PUT("/tasks/:id", updateTask(service))
	api.DELETE("/tasks/:id", deleteTask(service))
	api.POST("/tasks/:id/pause", pauseTask(service))
	api.POST("/tasks/:id/resume", resumeTask(service))
	api.POST("/tasks/:id/cancel", cancelTask(service))
	api.POST("/tasks/:id/run_now", runNow(service))
	api.GET("/tasks/:id/executions", listExecutions(service))
	api.GET("/tasks/:id/executions/:execution_id", getExecution(service))
	// Worker execution state is accepted only on the authenticated NATS
	// control channel. Keeping equivalent HTTP writes would bypass its replay
	// and message-integrity checks, so the legacy endpoints are intentionally
	// not registered.
	serverx.ApplyRouteRegistrars(serverx.ServiceTimerScheduler, router)
	return router
}

func createTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.CreateTaskRequest
		if !bindJSON(c, &req) {
			return
		}
		bindCreateTaskToAuthenticatedUser(c, &req)
		task, err := service.CreateTask(requestContext(c), req)
		writeResult(c, task, err)
	}
}

func listTasks(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := service.ListTasks(requestContext(c), scheduledsdk.ListTasksRequest{
			ExecutorKey:       c.Query("executor_key"),
			Status:            c.Query("status"),
			Category:          c.Query("category"),
			SourceType:        c.Query("source_type"),
			SourceRef:         c.Query("source_ref"),
			ResourceScope:     c.Query("resource_scope"),
			ResourceKey:       c.Query("resource_key"),
			ResourceKeyPrefix: c.Query("resource_key_prefix"),
			CreatedBy:         currentTimerRequestUser(c),
			Page:              queryInt(c, "page"),
			PageSize:          queryInt(c, "page_size"),
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
		task, ok := requireOwnedTimerTask(c, service, id)
		if !ok {
			return
		}
		writeResult(c, task, nil)
	}
}

func updateTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		if _, allowed := requireOwnedTimerTask(c, service, id); !allowed {
			return
		}
		var req scheduledsdk.UpdateTaskRequest
		if !bindJSON(c, &req) {
			return
		}
		bindUpdateTaskToAuthenticatedUser(c, &req)
		task, err := service.UpdateTask(requestContext(c), id, req)
		writeResult(c, task, err)
	}
}

func pauseTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		if _, allowed := requireOwnedTimerTask(c, service, id); !allowed {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.PauseTask(requestContext(c), id))
	}
}

func resumeTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		if _, allowed := requireOwnedTimerTask(c, service, id); !allowed {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.ResumeTask(requestContext(c), id))
	}
}

func cancelTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		if _, allowed := requireOwnedTimerTask(c, service, id); !allowed {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.CancelTask(requestContext(c), id))
	}
}

func deleteTask(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		if _, allowed := requireOwnedTimerTask(c, service, id); !allowed {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.DeleteTask(requestContext(c), id))
	}
}

func runNow(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		if _, allowed := requireOwnedTimerTask(c, service, id); !allowed {
			return
		}
		exec, err := service.RunNow(requestContext(c), id)
		writeResult(c, exec, err)
	}
}

func listExecutions(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID, ok := pathInt64(c, "id")
		if !ok {
			return
		}
		if _, allowed := requireOwnedTimerTask(c, service, taskID); !allowed {
			return
		}
		resp, err := service.ListExecutions(requestContext(c), taskID, scheduledsdk.ListExecutionsRequest{
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
		if _, allowed := requireOwnedTimerTask(c, service, taskID); !allowed {
			return
		}
		executionID, ok := pathInt64(c, "execution_id")
		if !ok {
			return
		}
		exec, err := service.GetExecution(requestContext(c), taskID, executionID)
		writeResult(c, exec, err)
	}
}

func markExecutionStarted(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.MarkExecutionStartedRequest
		if !bindJSON(c, &req) {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.MarkExecutionStarted(requestContext(c), req))
	}
}

func markExecutionHeartbeat(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.MarkExecutionHeartbeatRequest
		if !bindJSON(c, &req) {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.MarkExecutionHeartbeat(requestContext(c), req))
	}
}

func markExecutionFinished(service *timerservice.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduledsdk.MarkExecutionFinishedRequest
		if !bindJSON(c, &req) {
			return
		}
		writeResult(c, gin.H{"ok": true}, service.MarkExecutionFinished(requestContext(c), req))
	}
}

func requestContext(c *gin.Context) context.Context {
	if c == nil {
		return context.Background()
	}
	return contextx.ToContext(c)
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
