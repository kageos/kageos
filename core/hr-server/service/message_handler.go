package service

import (
	"context"
	"encoding/json"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// MessageCommandHandler 负责把 NATS 消息解码后转交给 MessageConsumerService。
type MessageCommandHandler struct {
	consumer *MessageConsumerService
}

// NewMessageCommandHandler 创建 MessageCommandHandler。
func NewMessageCommandHandler(consumer *MessageConsumerService) *MessageCommandHandler {
	return &MessageCommandHandler{consumer: consumer}
}

// HandleMessageSend 处理 message send 命令。
func (h *MessageCommandHandler) HandleMessageSend(msg *nats.Msg) {
	ctx := context.Background()

	var payload dto.MessageSendPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		logger.Errorf(ctx, "[MessageConsumer] Unmarshal failed: %v", err)
		return
	}

	if err := h.consumer.Consume(ctx, &payload); err != nil {
		logger.Errorf(ctx, "[MessageConsumer] Consume failed: %v", err)
	}
}
