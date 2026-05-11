package server

import (
	"net/http"
	"strconv"

	workflowdto "github.com/ai-agent-os/ai-agent-os/core/workflow-server/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	s.httpServer.GET("/health", s.healthHandler)
	s.httpServer.GET("/workflow/health", s.healthHandler)

	api := s.httpServer.Group("/workflow/api/v1")
	api.Use(middleware2.JWTAuth())
	api.POST("/workflows", s.createWorkflow)
	api.GET("/workflows", s.listWorkflows)
	api.GET("/workflows/by_path", s.getWorkflowByPath)
	api.GET("/workflows/:id", s.getWorkflow)
	api.PUT("/workflows/:id", s.updateWorkflow)
	api.POST("/workflows/:id/publish", s.publishWorkflow)
	api.POST("/workflows/:id/run", s.runWorkflow)
	api.GET("/runs/:run_id", s.getRun)
	api.GET("/runs/:run_id/steps", s.listRunSteps)
	api.POST("/runs/:run_id/cancel", s.cancelRun)
}

func (s *Server) createWorkflow(c *gin.Context) {
	var req workflowdto.CreateWorkflowRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := s.service.CreateWorkflow(contextx.ToContext(c), req)
	writeResult(c, resp, err)
}

func (s *Server) updateWorkflow(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		return
	}
	var req workflowdto.UpdateWorkflowRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := s.service.UpdateWorkflow(contextx.ToContext(c), id, req)
	writeResult(c, resp, err)
}

func (s *Server) listWorkflows(c *gin.Context) {
	appID, _ := strconv.ParseInt(c.Query("app_id"), 10, 64)
	resp, err := s.service.ListWorkflows(
		contextx.ToContext(c),
		c.Query("status"),
		appID,
		queryInt(c, "page"),
		queryInt(c, "page_size"),
	)
	writeResult(c, resp, err)
}

func (s *Server) getWorkflow(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		return
	}
	resp, err := s.service.GetWorkflow(contextx.ToContext(c), id)
	writeResult(c, resp, err)
}

func (s *Server) getWorkflowByPath(c *gin.Context) {
	resp, err := s.service.GetWorkflowByFullCodePath(contextx.ToContext(c), c.Query("full_code_path"))
	writeResult(c, resp, err)
}

func (s *Server) publishWorkflow(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		return
	}
	var req workflowdto.PublishWorkflowRequest
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if !bindJSON(c, &req) {
			return
		}
	}
	resp, err := s.service.PublishWorkflow(contextx.ToContext(c), id, req)
	writeResult(c, resp, err)
}

func (s *Server) runWorkflow(c *gin.Context) {
	id, ok := pathInt64(c, "id")
	if !ok {
		return
	}
	var req workflowdto.RunWorkflowRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := s.service.RunWorkflow(contextx.ToContext(c), id, req)
	writeResult(c, resp, err)
}

func (s *Server) getRun(c *gin.Context) {
	runID, ok := pathInt64(c, "run_id")
	if !ok {
		return
	}
	resp, err := s.service.GetRunDetail(contextx.ToContext(c), runID)
	writeResult(c, resp, err)
}

func (s *Server) listRunSteps(c *gin.Context) {
	runID, ok := pathInt64(c, "run_id")
	if !ok {
		return
	}
	resp, err := s.service.ListRunSteps(contextx.ToContext(c), runID)
	writeResult(c, gin.H{"list": resp}, err)
}

func (s *Server) cancelRun(c *gin.Context) {
	runID, ok := pathInt64(c, "run_id")
	if !ok {
		return
	}
	writeResult(c, gin.H{"ok": true}, s.service.CancelRun(contextx.ToContext(c), runID))
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
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, data)
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
