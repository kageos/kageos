package server

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/message-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

func (s *Server) getPublicMessageAction(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	view, err := s.messageRepo.GetActionView(c.Request.Context(), token, "", contextx.GetRequestUser(c))
	if err != nil {
		if isMessageActionLoginRequiredError(err) {
			response.NoAuth(c, err.Error())
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	authenticatedUser := strings.TrimSpace(contextx.GetRequestUser(c))
	if authenticatedUser == "" || strings.TrimSpace(view.RecipientUser) != authenticatedUser {
		response.NoAuth(c, "消息动作用户与当前登录用户不一致")
		return
	}
	view.MobileAskURL = buildMobileAskURL(config.GetPublicSiteBaseURL(), view.Message.SourcePath, view.WorkspaceSession)
	response.OkWithData(c, view)
}

func (s *Server) submitPublicMessageActionReply(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	var req dto.MessageActionReplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	view, err := s.messageRepo.GetActionView(c.Request.Context(), token, "", contextx.GetRequestUser(c))
	if err != nil {
		if isMessageActionLoginRequiredError(err) {
			response.NoAuth(c, err.Error())
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	authenticatedUser := strings.TrimSpace(contextx.GetRequestUser(c))
	if authenticatedUser == "" || strings.TrimSpace(view.RecipientUser) != authenticatedUser {
		response.NoAuth(c, "消息动作用户与当前登录用户不一致")
		return
	}
	fullCodePath := mobileActionFullCodePath(view.Message)
	if fullCodePath == "" {
		response.FailWithMessage(c, "消息缺少工作台目录，无法提交给 kageos 工作台")
		return
	}
	resp, err := s.messageRepo.SubmitActionReply(c.Request.Context(), token, req.Content, req.Files, req.Action, contextx.GetRequestUser(c))
	if err != nil {
		if isMessageActionLoginRequiredError(err) {
			response.NoAuth(c, err.Error())
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	resp.MobileAskURL = buildMobileAskURL(config.GetPublicSiteBaseURL(), resp.SourcePath, resp.WorkspaceSessionID)
	resp.FullCodePath = firstNonEmptyActionString(resp.FullCodePath, fullCodePath)
	resp.AgentSubmitted = false
	if s.workspaceActionRunner == nil {
		resp.AgentSubmitError = "工作台提交器未初始化"
		response.OkWithData(c, resp)
		return
	}
	runResult, runErr := s.workspaceActionRunner.Submit(c.Request.Context(), service.WorkspaceActionRequest{
		RecipientUser:         view.RecipientUser,
		UserID:                c.GetHeader(contextx.UserIDHeader),
		UserEmail:             c.GetHeader(contextx.UserEmailHeader),
		LeaderUsername:        c.GetHeader(contextx.LeaderUsernameHeader),
		DepartmentFullPath:    c.GetHeader(contextx.DepartmentFullPathHeader),
		CompanyCode:           c.GetHeader(contextx.CompanyCodeHeader),
		CompanyName:           c.GetHeader(contextx.CompanyNameHeader),
		CompanyLogoURL:        c.GetHeader(contextx.CompanyLogoURLHeader),
		Channel:               firstNonEmptyActionString(resp.Channel, view.Channel),
		FullCodePath:          resp.FullCodePath,
		SessionID:             resp.WorkspaceSessionID,
		ThreadKey:             view.Message.ThreadKey,
		Content:               resp.WorkstationDraft,
		DisplayContent:        strings.TrimSpace(req.Content),
		Files:                 strings.TrimSpace(req.Files),
		OriginalTitle:         view.Message.Title,
		TraceID:               view.Message.TraceID,
		SourceRef:             firstNonEmptyActionString(view.Message.SourceRef, fmt.Sprintf("message:%d", view.Message.ID)),
		SourcePath:            view.Message.SourcePath,
		SourceTitle:           view.Message.SourceTitle,
		SourceParentPath:      view.Message.SourceParentPath,
		SourceParentTitle:     view.Message.SourceParentTitle,
		SourceTemplateType:    view.Message.SourceTemplateType,
		WorkspaceSessionTitle: view.Message.WorkspaceSessionTitle,
		WorkspaceRole:         view.Message.WorkspaceRole,
	})
	if runErr != nil {
		resp.AgentSubmitError = runErr.Error()
		logger.Warnf(c.Request.Context(), "[message-action] submit workspace failed token_message_id=%d user=%s err=%v", view.Message.ID, view.RecipientUser, runErr)
		response.OkWithData(c, resp)
		return
	}
	resp.AgentSubmitted = runResult != nil && runResult.Accepted
	if runResult != nil && strings.TrimSpace(runResult.SessionID) != "" {
		resp.WorkspaceSessionID = strings.TrimSpace(runResult.SessionID)
		resp.MobileAskURL = buildMobileAskURL(config.GetPublicSiteBaseURL(), resp.SourcePath, resp.WorkspaceSessionID)
		if err := s.messageRepo.UpdateActionWorkspaceSession(c.Request.Context(), token, resp.WorkspaceSessionID); err != nil {
			logger.Warnf(c.Request.Context(), "[message-action] update workspace session failed token_message_id=%d session_id=%s err=%v", view.Message.ID, resp.WorkspaceSessionID, err)
		}
	}
	response.OkWithData(c, resp)
}

func buildMobileAskURL(baseURL, sourcePath, sessionID string) string {
	route := "/m"
	query := url.Values{}
	if strings.TrimSpace(sourcePath) != "" {
		query.Set("source_path", strings.TrimSpace(sourcePath))
	}
	if strings.TrimSpace(sessionID) != "" {
		query.Set("session_id", strings.TrimSpace(sessionID))
	}
	if len(query) > 0 {
		route += "?" + query.Encode()
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return route
	}
	return fmt.Sprintf("%s%s", baseURL, route)
}

func mobileActionFullCodePath(message dto.MessageInboxItem) string {
	return firstNonEmptyActionString(message.SourcePath, message.FullCodePath, message.SourceParentPath)
}

func firstNonEmptyActionString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func isMessageActionLoginRequiredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "请先登录")
}
