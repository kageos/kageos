package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
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
		messageActionError(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
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
	if authenticatedUser == "" {
		response.NoAuth(c, "请先登录 kageos 后处理该消息")
		return
	}
	if strings.TrimSpace(view.RecipientUser) != authenticatedUser {
		messageActionError(c, http.StatusForbidden, "消息动作用户与当前登录用户不一致")
		return
	}
	fullCodePath := mobileActionFullCodePath(view.Message)
	if fullCodePath == "" {
		messageActionError(c, http.StatusBadRequest, "消息缺少工作台目录，无法提交给 kageos 工作台")
		return
	}
	resp, err := s.messageRepo.BeginActionReply(c.Request.Context(), token, req.Content, req.Files, req.Action, authenticatedUser)
	if err != nil {
		if isMessageActionLoginRequiredError(err) {
			response.NoAuth(c, err.Error())
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	resp.MobileAskURL = buildMobileAskURL(config.GetPublicSiteBaseURL(), fullCodePath, resp.WorkspaceSessionID)
	resp.FullCodePath = firstNonEmptyActionString(resp.FullCodePath, fullCodePath)
	resp.AgentSubmitted = false
	if s.workspaceActionRunner == nil {
		s.releasePublicMessageActionReply(c.Request.Context(), token)
		messageActionError(c, http.StatusInternalServerError, "工作台提交器未初始化，请稍后重试")
		return
	}
	runResult, runErr := s.workspaceActionRunner.Submit(c.Request.Context(), service.WorkspaceActionRequest{
		RecipientUser:         view.RecipientUser,
		DepartmentFullPath:    c.GetHeader(contextx.DepartmentFullPathHeader),
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
		s.releasePublicMessageActionReply(c.Request.Context(), token)
		logger.Warnf(c.Request.Context(), "[message-action] submit workspace failed token_message_id=%d user=%s err=%v", view.Message.ID, view.RecipientUser, runErr)
		messageActionError(c, http.StatusInternalServerError, "创建工作台会话失败，请重试: "+runErr.Error())
		return
	}
	if runResult == nil || !runResult.Accepted || strings.TrimSpace(runResult.SessionID) == "" {
		s.releasePublicMessageActionReply(c.Request.Context(), token)
		messageActionError(c, http.StatusInternalServerError, "工作台没有返回有效会话，请重试")
		return
	}
	resp.WorkspaceSessionID = strings.TrimSpace(runResult.SessionID)
	persistCtx, cancelPersist := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Second)
	submittedAt, err := s.messageRepo.FinalizeActionReply(persistCtx, token, resp.WorkspaceSessionID)
	cancelPersist()
	if err != nil {
		s.releasePublicMessageActionReply(c.Request.Context(), token)
		logger.Warnf(c.Request.Context(), "[message-action] finalize reply failed token_message_id=%d session_id=%s err=%v", view.Message.ID, resp.WorkspaceSessionID, err)
		messageActionError(c, http.StatusInternalServerError, "工作台会话已创建，但回复状态确认失败，请刷新后重试")
		return
	}
	resp.Status = string(dto.MessageActionTokenStatusSubmitted)
	resp.SubmittedAt = submittedAt
	resp.AgentSubmitted = true
	resp.MobileAskURL = buildMobileAskURL(config.GetPublicSiteBaseURL(), fullCodePath, resp.WorkspaceSessionID)
	response.OkWithData(c, resp)
}

func (s *Server) releasePublicMessageActionReply(parent context.Context, token string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancel()
	if err := s.messageRepo.ReleaseActionReply(ctx, token); err != nil {
		logger.Warnf(ctx, "[message-action] release processing reply failed err=%v", err)
	}
}

func messageActionError(c *gin.Context, status int, message string) {
	c.JSON(status, response.Response{Code: response.ERROR, Data: map[string]interface{}{}, Msg: message})
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
	return msgrepo.ResolveMessageWorkspacePath(message.SourcePath, message.FullCodePath, message.SourceParentPath, message.SourceTemplateType)
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
