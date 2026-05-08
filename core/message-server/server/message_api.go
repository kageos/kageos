package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/auth"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

func (s *Server) sendMessage(c *gin.Context) {
	var envelope dto.MessageSendEnvelope
	if err := c.ShouldBindJSON(&envelope); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	s.submitMessage(c, &envelope)
}

func (s *Server) sendMessageToUsers(c *gin.Context) {
	var req dto.MessageSendToUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	envelope := dto.MessageSendEnvelope{
		Message: dto.MessageSendPayload{
			ToUsers:     req.ToUsers,
			Title:       req.Title,
			Content:     req.Content,
			ContentType: req.ContentType,
		},
	}
	s.submitMessage(c, &envelope)
}

func (s *Server) sendMessageToDepartments(c *gin.Context) {
	var req dto.MessageSendToDepartmentsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	envelope := dto.MessageSendEnvelope{
		Message: dto.MessageSendPayload{
			ToDepartments: req.ToDepartments,
			Title:         req.Title,
			Content:       req.Content,
			ContentType:   req.ContentType,
		},
	}
	s.submitMessage(c, &envelope)
}

func (s *Server) submitMessage(c *gin.Context, envelope *dto.MessageSendEnvelope) {
	if s == nil || s.messageConsumerService == nil {
		response.FailWithMessage(c, "message service 未初始化")
		return
	}
	if envelope == nil {
		response.FailWithMessage(c, "消息内容不能为空")
		return
	}

	ctx, meta, err := resolveRequestMessageMeta(c, &envelope.Meta)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	envelope.Meta = meta
	normalizeMessagePayload(&envelope.Message)
	if errMsg := validateMessageEnvelope(envelope); errMsg != "" {
		response.FailWithMessage(c, errMsg)
		return
	}

	if err := s.messageConsumerService.Consume(ctx, envelope); err != nil {
		response.FailWithMessage(c, "消息发送失败: "+err.Error())
		return
	}

	response.OkWithData(c, dto.MessageSendResp{
		Message:       "消息已提交发送",
		Meta:          envelope.Meta,
		Payload:       envelope.Message,
		From:          envelope.Meta.From,
		FullCodePath:  envelope.Meta.FullCodePath,
		ToUsers:       envelope.Message.ToUsers,
		ToDepartments: envelope.Message.ToDepartments,
		ContentType:   envelope.Message.ContentType,
	})
}

func resolveRequestSender(c *gin.Context) (context.Context, string, error) {
	ctx, meta, err := resolveRequestMessageMeta(c, nil)
	return ctx, meta.From, err
}

func resolveRequestMessageMeta(c *gin.Context, incoming *dto.MessageSendMeta) (context.Context, dto.MessageSendMeta, error) {
	ctx := contextx.ToContext(c)
	token := strings.TrimSpace(contextx.GetToken(ctx))
	if token == "" {
		return ctx, dto.MessageSendMeta{}, fmt.Errorf("未提供认证令牌")
	}

	claims, err := auth.NewJWTService().ValidateToken(token)
	if err != nil {
		return ctx, dto.MessageSendMeta{}, fmt.Errorf("认证令牌无效或已过期")
	}
	sender := strings.TrimSpace(claims.Username)
	if sender == "" {
		return ctx, dto.MessageSendMeta{}, fmt.Errorf("认证令牌缺少发送人")
	}
	c.Request.Header.Set(contextx.RequestUserHeader, sender)
	if claims.DepartmentFullPath != nil && strings.TrimSpace(*claims.DepartmentFullPath) != "" {
		c.Request.Header.Set(contextx.DepartmentFullPathHeader, strings.TrimSpace(*claims.DepartmentFullPath))
	}
	ctx = contextx.ToContext(c)

	meta := dto.MessageSendMeta{
		From:               sender,
		RequestUser:        sender,
		DepartmentFullPath: strings.TrimSpace(contextx.GetRequestDepartmentFullPath(ctx)),
		TraceID:            strings.TrimSpace(contextx.GetTraceId(ctx)),
		ClientSource:       strings.TrimSpace(contextx.GetClientSource(ctx)),
		SourceType:         strings.TrimSpace(contextx.GetSourceType(ctx)),
		SourceRef:          strings.TrimSpace(contextx.GetSourceRef(ctx)),
	}
	if strings.HasPrefix(meta.SourceRef, "/") {
		meta.FullCodePath = meta.SourceRef
	}
	if incoming != nil && meta.FullCodePath == "" {
		meta.FullCodePath = strings.TrimSpace(incoming.FullCodePath)
	}
	return ctx, meta, nil
}

func normalizeMessagePayload(payload *dto.MessageSendPayload) {
	payload.ToUsers = strings.TrimSpace(payload.ToUsers)
	payload.ToDepartments = strings.TrimSpace(payload.ToDepartments)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Content = strings.TrimSpace(payload.Content)
	payload.ContentType = strings.ToLower(strings.TrimSpace(payload.ContentType))
	if payload.ContentType == "" {
		payload.ContentType = "markdown"
	}
}

func validateMessagePayload(payload *dto.MessageSendPayload) string {
	if payload == nil {
		return "消息内容不能为空"
	}
	if payload.ToUsers == "" && payload.ToDepartments == "" {
		return "to_users 和 to_departments 至少填写一个"
	}
	if payload.Content == "" {
		return "content 不能为空"
	}
	switch payload.ContentType {
	case "markdown", "html", "text":
		return ""
	default:
		return "content_type 仅支持 markdown、html、text"
	}
}

func validateMessageEnvelope(envelope *dto.MessageSendEnvelope) string {
	if envelope == nil {
		return "消息内容不能为空"
	}
	if strings.TrimSpace(envelope.Meta.From) == "" {
		return "无法解析发送人"
	}
	return validateMessagePayload(&envelope.Message)
}

func (s *Server) messageAPIHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "message-server",
		"api":     "message",
	})
}
