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
		From:                  sender,
		RequestUser:           sender,
		DepartmentFullPath:    strings.TrimSpace(contextx.GetRequestDepartmentFullPath(ctx)),
		TraceID:               strings.TrimSpace(contextx.GetTraceId(ctx)),
		ClientSource:          strings.TrimSpace(contextx.GetClientSource(ctx)),
		SourceType:            strings.TrimSpace(contextx.GetSourceType(ctx)),
		SourceRef:             strings.TrimSpace(contextx.GetSourceRef(ctx)),
		SourcePath:            strings.TrimSpace(contextx.GetSourcePath(ctx)),
		SourceTitle:           strings.TrimSpace(contextx.GetSourceTitle(ctx)),
		SourceParentPath:      strings.TrimSpace(contextx.GetSourceParentPath(ctx)),
		SourceParentTitle:     strings.TrimSpace(contextx.GetSourceParentTitle(ctx)),
		SourceTemplateType:    strings.TrimSpace(contextx.GetSourceTemplateType(ctx)),
		SourceIcon:            strings.TrimSpace(contextx.GetSourceIcon(ctx)),
		SourceColor:           strings.TrimSpace(contextx.GetSourceColor(ctx)),
		SourceParentIcon:      strings.TrimSpace(contextx.GetSourceParentIcon(ctx)),
		SourceParentColor:     strings.TrimSpace(contextx.GetSourceParentColor(ctx)),
		WorkspaceSessionID:    strings.TrimSpace(contextx.GetWorkspaceSessionID(ctx)),
		WorkspaceSessionTitle: strings.TrimSpace(contextx.GetWorkspaceSessionTitle(ctx)),
		WorkspaceRole:         strings.TrimSpace(contextx.GetWorkspaceRole(ctx)),
	}
	if strings.HasPrefix(meta.SourceRef, "/") {
		meta.FullCodePath = meta.SourceRef
	}
	if incoming != nil && meta.FullCodePath == "" {
		meta.FullCodePath = strings.TrimSpace(incoming.FullCodePath)
	}
	if incoming != nil {
		if meta.SourcePath == "" {
			meta.SourcePath = strings.TrimSpace(incoming.SourcePath)
		}
		if meta.SourceTitle == "" {
			meta.SourceTitle = strings.TrimSpace(incoming.SourceTitle)
		}
		if meta.SourceParentPath == "" {
			meta.SourceParentPath = strings.TrimSpace(incoming.SourceParentPath)
		}
		if meta.SourceParentTitle == "" {
			meta.SourceParentTitle = strings.TrimSpace(incoming.SourceParentTitle)
		}
		if meta.SourceTemplateType == "" {
			meta.SourceTemplateType = strings.TrimSpace(incoming.SourceTemplateType)
		}
		if meta.SourceIcon == "" {
			meta.SourceIcon = strings.TrimSpace(incoming.SourceIcon)
		}
		if meta.SourceColor == "" {
			meta.SourceColor = strings.TrimSpace(incoming.SourceColor)
		}
		if meta.SourceParentIcon == "" {
			meta.SourceParentIcon = strings.TrimSpace(incoming.SourceParentIcon)
		}
		if meta.SourceParentColor == "" {
			meta.SourceParentColor = strings.TrimSpace(incoming.SourceParentColor)
		}
		if meta.WorkspaceSessionID == "" {
			meta.WorkspaceSessionID = strings.TrimSpace(incoming.WorkspaceSessionID)
		}
		if meta.WorkspaceSessionTitle == "" {
			meta.WorkspaceSessionTitle = strings.TrimSpace(incoming.WorkspaceSessionTitle)
		}
		if meta.WorkspaceRole == "" {
			meta.WorkspaceRole = strings.TrimSpace(incoming.WorkspaceRole)
		}
		if meta.ThreadKey == "" {
			meta.ThreadKey = strings.TrimSpace(incoming.ThreadKey)
		}
	}
	if meta.SourcePath == "" {
		meta.SourcePath = meta.FullCodePath
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
