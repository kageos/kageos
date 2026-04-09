package server

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	backupservice "github.com/ai-agent-os/ai-agent-os/core/backup-service/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Server) setupRoutes() {
	s.httpServer.GET("/", s.consoleHandler)
	s.httpServer.GET("/backup", s.consoleHandler)
	s.httpServer.GET("/health", s.healthHandler)

	apiV1 := s.httpServer.Group("/backup/api/v1")
	apiV1.GET("/status", s.statusHandler)
	apiV1.GET("/tasks", s.listTasksHandler)
	apiV1.GET("/tasks/:id", s.getTaskHandler)
	apiV1.GET("/snapshots", s.listSnapshotsHandler)
	apiV1.GET("/snapshots/:id", s.getSnapshotHandler)
	apiV1.DELETE("/snapshots/:id", s.deleteSnapshotHandler)
	apiV1.POST("/precheck", s.precheckHandler)
	apiV1.POST("/maintenance", s.maintenanceHandler)
	apiV1.POST("/namespace/snapshots", s.namespaceSnapshotHandler)
	apiV1.POST("/namespace/restore", s.namespaceRestoreHandler)
	apiV1.POST("/mysql/snapshots", s.mysqlSnapshotHandler)
	apiV1.POST("/mysql/restore", s.mysqlRestoreHandler)
	apiV1.POST("/minio/snapshots", s.minioSnapshotHandler)
	apiV1.POST("/minio/restore", s.minioRestoreHandler)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "backup-service",
	})
}

func (s *Server) statusHandler(c *gin.Context) {
	status, err := s.controlPlane.GetStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Server) listTasksHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	tasks, err := s.controlPlane.ListTasks(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (s *Server) getTaskHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	task, err := s.controlPlane.GetTask(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (s *Server) listSnapshotsHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	resourceType := c.Query("resource_type")
	snapshots, err := s.controlPlane.ListSnapshots(c.Request.Context(), resourceType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshots)
}

func (s *Server) getSnapshotHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
		return
	}

	snapshot, err := s.controlPlane.GetSnapshot(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

func (s *Server) deleteSnapshotHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
		return
	}

	var req struct {
		RequestedBy string `json:"requested_by"`
		Note        string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := s.controlPlane.DeleteSnapshot(c.Request.Context(), req.RequestedBy, req.Note, id)
	if err != nil {
		switch {
		case errors.Is(err, backupservice.ErrTaskAlreadyRunning):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, task)
}

func (s *Server) precheckHandler(c *gin.Context) {
	var req struct {
		RequestedBy string `json:"requested_by"`
		Note        string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := s.controlPlane.RunPrecheck(c.Request.Context(), req.RequestedBy, req.Note)
	if err != nil {
		if errors.Is(err, backupservice.ErrTaskAlreadyRunning) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (s *Server) maintenanceHandler(c *gin.Context) {
	var req struct {
		Enabled     bool   `json:"enabled"`
		RequestedBy string `json:"requested_by"`
		Reason      string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, err := s.controlPlane.SetMaintenanceMode(c.Request.Context(), req.Enabled, req.RequestedBy, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, state)
}

func (s *Server) namespaceSnapshotHandler(c *gin.Context) {
	var req struct {
		RequestedBy  string `json:"requested_by"`
		Note         string `json:"note"`
		RelativePath string `json:"relative_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := s.controlPlane.CreateNamespaceSnapshot(c.Request.Context(), req.RequestedBy, req.Note, req.RelativePath)
	if err != nil {
		if errors.Is(err, backupservice.ErrTaskAlreadyRunning) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, backupservice.ErrInvalidRelativePath) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (s *Server) namespaceRestoreHandler(c *gin.Context) {
	var req struct {
		RequestedBy string `json:"requested_by"`
		Note        string `json:"note"`
		SnapshotID  int64  `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SnapshotID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_id is required"})
		return
	}

	task, err := s.controlPlane.RestoreNamespaceSnapshot(c.Request.Context(), req.RequestedBy, req.Note, req.SnapshotID)
	if err != nil {
		switch {
		case errors.Is(err, backupservice.ErrTaskAlreadyRunning):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, backupservice.ErrMaintenanceModeRequired):
			c.JSON(http.StatusPreconditionFailed, gin.H{"error": err.Error()})
		case errors.Is(err, backupservice.ErrInvalidRelativePath):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, task)
}

func (s *Server) mysqlSnapshotHandler(c *gin.Context) {
	var req struct {
		RequestedBy string `json:"requested_by"`
		Note        string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := s.controlPlane.CreateMySQLSnapshot(c.Request.Context(), req.RequestedBy, req.Note)
	if err != nil {
		if errors.Is(err, backupservice.ErrTaskAlreadyRunning) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (s *Server) mysqlRestoreHandler(c *gin.Context) {
	var req struct {
		RequestedBy string `json:"requested_by"`
		Note        string `json:"note"`
		SnapshotID  int64  `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SnapshotID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_id is required"})
		return
	}

	task, err := s.controlPlane.RestoreMySQLSnapshot(c.Request.Context(), req.RequestedBy, req.Note, req.SnapshotID)
	if err != nil {
		switch {
		case errors.Is(err, backupservice.ErrTaskAlreadyRunning):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, backupservice.ErrMaintenanceModeRequired):
			c.JSON(http.StatusPreconditionFailed, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, task)
}

func (s *Server) minioSnapshotHandler(c *gin.Context) {
	var req struct {
		RequestedBy string `json:"requested_by"`
		Note        string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := s.controlPlane.CreateMinIOSnapshot(c.Request.Context(), req.RequestedBy, req.Note)
	if err != nil {
		if errors.Is(err, backupservice.ErrTaskAlreadyRunning) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (s *Server) minioRestoreHandler(c *gin.Context) {
	var req struct {
		RequestedBy string `json:"requested_by"`
		Note        string `json:"note"`
		SnapshotID  int64  `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SnapshotID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_id is required"})
		return
	}

	task, err := s.controlPlane.RestoreMinIOSnapshot(c.Request.Context(), req.RequestedBy, req.Note, req.SnapshotID)
	if err != nil {
		switch {
		case errors.Is(err, backupservice.ErrTaskAlreadyRunning):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, backupservice.ErrMaintenanceModeRequired):
			c.JSON(http.StatusPreconditionFailed, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, task)
}
