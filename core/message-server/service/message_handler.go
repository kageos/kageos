package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/msgx"
	"github.com/nats-io/nats.go"
)

type MessageCommandHandler struct {
	consumer *MessageConsumerService
}

func NewMessageCommandHandler(consumer *MessageConsumerService) *MessageCommandHandler {
	return &MessageCommandHandler{consumer: consumer}
}

func (h *MessageCommandHandler) HandleMessageSend(msg *nats.Msg) {
	ctx := contextx.NatsTraceContext(msg)

	var envelope dto.MessageSendEnvelope
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		logger.Errorf(ctx, "[MessageConsumer] Unmarshal failed: %v", err)
		respondMessageSendFailure(ctx, msg, err)
		return
	}
	if h == nil || h.consumer == nil {
		err := fmt.Errorf("message consumer is not initialized")
		logger.Errorf(ctx, "[MessageConsumer] Consume failed: %v", err)
		respondMessageSendFailure(ctx, msg, err)
		return
	}
	fillMessageMetaFromContext(ctx, &envelope.Meta)
	if err := h.consumer.Consume(ctx, &envelope); err != nil {
		logger.Errorf(ctx, "[MessageConsumer] Consume failed: %v", err)
		respondMessageSendFailure(ctx, msg, err)
		return
	}
	respondMessageSendSuccess(ctx, msg, &envelope)
}

func fillMessageMetaFromContext(ctx context.Context, meta *dto.MessageSendMeta) {
	if meta == nil {
		return
	}
	if strings.TrimSpace(meta.RequestUser) == "" {
		meta.RequestUser = strings.TrimSpace(contextx.GetRequestUser(ctx))
	}
	if strings.TrimSpace(meta.From) == "" {
		meta.From = strings.TrimSpace(meta.RequestUser)
	}
	if strings.TrimSpace(meta.From) == "" {
		meta.From = "system"
	}
	if strings.TrimSpace(meta.DepartmentFullPath) == "" {
		meta.DepartmentFullPath = strings.TrimSpace(contextx.GetRequestDepartmentFullPath(ctx))
	}
	if strings.TrimSpace(meta.TraceID) == "" {
		meta.TraceID = strings.TrimSpace(contextx.GetTraceId(ctx))
	}
	if strings.TrimSpace(meta.ClientSource) == "" {
		meta.ClientSource = strings.TrimSpace(contextx.GetClientSource(ctx))
	}
	if strings.TrimSpace(meta.SourceType) == "" {
		meta.SourceType = strings.TrimSpace(contextx.GetSourceType(ctx))
	}
	if strings.TrimSpace(meta.SourceRef) == "" {
		meta.SourceRef = strings.TrimSpace(contextx.GetSourceRef(ctx))
	}
	if strings.TrimSpace(meta.SourcePath) == "" {
		meta.SourcePath = strings.TrimSpace(contextx.GetSourcePath(ctx))
	}
	if strings.TrimSpace(meta.SourceTitle) == "" {
		meta.SourceTitle = strings.TrimSpace(contextx.GetSourceTitle(ctx))
	}
	if strings.TrimSpace(meta.SourceParentPath) == "" {
		meta.SourceParentPath = strings.TrimSpace(contextx.GetSourceParentPath(ctx))
	}
	if strings.TrimSpace(meta.SourceParentTitle) == "" {
		meta.SourceParentTitle = strings.TrimSpace(contextx.GetSourceParentTitle(ctx))
	}
	if strings.TrimSpace(meta.SourceTemplateType) == "" {
		meta.SourceTemplateType = strings.TrimSpace(contextx.GetSourceTemplateType(ctx))
	}
	if strings.TrimSpace(meta.WorkspaceSessionID) == "" {
		meta.WorkspaceSessionID = strings.TrimSpace(contextx.GetWorkspaceSessionID(ctx))
	}
	if strings.TrimSpace(meta.WorkspaceSessionTitle) == "" {
		meta.WorkspaceSessionTitle = strings.TrimSpace(contextx.GetWorkspaceSessionTitle(ctx))
	}
	if strings.TrimSpace(meta.WorkspaceRole) == "" {
		meta.WorkspaceRole = strings.TrimSpace(contextx.GetWorkspaceRole(ctx))
	}
}

func respondMessageSendSuccess(ctx context.Context, msg *nats.Msg, envelope *dto.MessageSendEnvelope) {
	if msg == nil || msg.Reply == "" || envelope == nil {
		return
	}
	if err := msgx.RespondJSONSuccess(msg, dto.MessageSendResp{
		Message:      "消息已提交发送",
		Meta:         envelope.Meta,
		Payload:      envelope.Message,
		From:         envelope.Meta.From,
		FullCodePath: envelope.Meta.FullCodePath,
		ToUsers:      envelope.Message.ToUsers,
		ContentType:  envelope.Message.ContentType,
	}); err != nil {
		logger.Warnf(ctx, "[MessageConsumer] Respond success failed: %v", err)
	}
}

func respondMessageSendFailure(ctx context.Context, msg *nats.Msg, err error) {
	if msg == nil || msg.Reply == "" {
		return
	}
	if err := msgx.RespondJSONFailure(msg, err); err != nil {
		logger.Warnf(ctx, "[MessageConsumer] Respond failure failed: %v", err)
	}
}
