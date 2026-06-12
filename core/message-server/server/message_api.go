package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
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
	if err := s.messageConsumerService.Consume(ctx, envelope); err != nil {
		response.FailWithMessage(c, "消息发送失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.MessageSendResp{
		Message:      "消息已提交发送",
		Meta:         envelope.Meta,
		Payload:      envelope.Message,
		From:         envelope.Meta.From,
		FullCodePath: envelope.Meta.FullCodePath,
		ToUsers:      envelope.Message.ToUsers,
		ContentType:  envelope.Message.ContentType,
	})
}

func resolveRequestMessageMeta(c *gin.Context, incoming *dto.MessageSendMeta) (context.Context, dto.MessageSendMeta, error) {
	ctx := contextx.ToContext(c)
	sender := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if sender == "" {
		return ctx, dto.MessageSendMeta{}, fmt.Errorf("无法解析发送人")
	}
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

func (s *Server) messageAPIHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "message-server",
		"api":     "message",
	})
}
