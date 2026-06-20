package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type SystemSettings struct {
	settingsService *service.SystemSettingsService
}

func NewSystemSettings(settingsService *service.SystemSettingsService) *SystemSettings {
	return &SystemSettings{settingsService: settingsService}
}

func (s *SystemSettings) Get(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	settings, err := s.settingsService.GetSettings()
	if err != nil {
		response.FailWithMessage(c, "获取系统设置失败: "+err.Error())
		return
	}
	response.OkWithData(c, settings)
}

func (s *SystemSettings) Update(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	var req dto.UpdateSystemSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	settings, err := s.settingsService.UpdateSettings(req, contextx.GetRequestUser(c))
	if err != nil {
		response.FailWithMessage(c, "保存系统设置失败: "+err.Error())
		return
	}
	response.OkWithData(c, settings)
}

func (s *SystemSettings) TestEmail(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	var req dto.TestEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	if err := s.settingsService.TestEmail(req.To); err != nil {
		response.FailWithMessage(c, "测试邮件发送失败: "+err.Error())
		return
	}
	response.OkWithMessage(c, "测试邮件已发送")
}

func (s *SystemSettings) GetTLS(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	settings, err := s.settingsService.GetTLSSettings()
	if err != nil {
		response.FailWithMessage(c, "获取 HTTPS 证书状态失败: "+err.Error())
		return
	}
	response.OkWithData(c, settings)
}

func (s *SystemSettings) UpdateTLS(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	var req dto.UpdateTLSCertificateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	settings, err := s.settingsService.UpdateTLSCertificate(req, contextx.GetRequestUser(c))
	if err != nil {
		response.FailWithMessage(c, "保存 HTTPS 证书失败: "+err.Error())
		return
	}
	response.OkWithData(c, settings)
}

func (s *SystemSettings) ReloadTLS(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	if err := s.settingsService.ReloadTLS(); err != nil {
		response.FailWithMessage(c, "热加载 HTTPS 证书失败: "+err.Error())
		return
	}
	settings, err := s.settingsService.GetTLSSettings()
	if err != nil {
		response.FailWithMessage(c, "读取 HTTPS 证书状态失败: "+err.Error())
		return
	}
	response.OkWithData(c, settings)
}

func requireSystemUser(c *gin.Context) bool {
	if contextx.GetRequestUser(c) != "system" {
		response.FailWithMessage(c, "仅 system 超管可操作")
		c.Abort()
		return false
	}
	return true
}
