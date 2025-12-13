package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getLicenseStatus 获取 License 状态
// GET /api/v1/license/status
func (s *Server) getLicenseStatus(c *gin.Context) {
	status := s.licenseService.GetStatus()
	c.JSON(http.StatusOK, status)
}

