package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type LoginAnnouncement struct {
	settingsService *service.SystemSettingsService
}

func NewLoginAnnouncement(settingsService *service.SystemSettingsService) *LoginAnnouncement {
	return &LoginAnnouncement{settingsService: settingsService}
}

func (a *LoginAnnouncement) PublicGet(c *gin.Context) {
	announcement, err := a.settingsService.GetLoginAnnouncement()
	if err != nil {
		response.FailWithMessage(c, "获取登录公告失败: "+err.Error())
		return
	}
	if !announcement.Enabled {
		announcement.Markdown = ""
	}
	response.OkWithData(c, announcement)
}

func (a *LoginAnnouncement) Get(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	announcement, err := a.settingsService.GetLoginAnnouncement()
	if err != nil {
		response.FailWithMessage(c, "获取登录公告失败: "+err.Error())
		return
	}
	response.OkWithData(c, announcement)
}

func (a *LoginAnnouncement) Update(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	var req dto.UpdateLoginAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	announcement, err := a.settingsService.UpdateLoginAnnouncement(req, contextx.GetRequestUser(c))
	if err != nil {
		response.FailWithMessage(c, "保存登录公告失败: "+err.Error())
		return
	}
	response.OkWithData(c, announcement)
}
