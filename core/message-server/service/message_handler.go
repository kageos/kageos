package service

import (
	"encoding/json"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// MessageCommandHandler 负责把 NATS 消息解码后转交给 MessageConsumerService。
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
		return
	}
	if strings.TrimSpace(envelope.Meta.RequestUser) == "" {
		envelope.Meta.RequestUser = strings.TrimSpace(contextx.GetRequestUser(ctx))
	}
	if strings.TrimSpace(envelope.Meta.From) == "" {
		envelope.Meta.From = strings.TrimSpace(envelope.Meta.RequestUser)
	}
	if strings.TrimSpace(envelope.Meta.From) == "" {
		envelope.Meta.From = "system"
	}
	if strings.TrimSpace(envelope.Meta.DepartmentFullPath) == "" {
		envelope.Meta.DepartmentFullPath = strings.TrimSpace(contextx.GetRequestDepartmentFullPath(ctx))
	}
	if strings.TrimSpace(envelope.Meta.TraceID) == "" {
		envelope.Meta.TraceID = strings.TrimSpace(contextx.GetTraceId(ctx))
	}
	if strings.TrimSpace(envelope.Meta.ClientSource) == "" {
		envelope.Meta.ClientSource = strings.TrimSpace(contextx.GetClientSource(ctx))
	}
	if strings.TrimSpace(envelope.Meta.SourceType) == "" {
		envelope.Meta.SourceType = strings.TrimSpace(contextx.GetSourceType(ctx))
	}
	if strings.TrimSpace(envelope.Meta.SourceRef) == "" {
		envelope.Meta.SourceRef = strings.TrimSpace(contextx.GetSourceRef(ctx))
	}

	if err := h.consumer.Consume(ctx, &envelope); err != nil {
		logger.Errorf(ctx, "[MessageConsumer] Consume failed: %v", err)
	}
}
