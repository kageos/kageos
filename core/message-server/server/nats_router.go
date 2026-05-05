package server

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/message-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func RegisterNATS(ctx context.Context, conn *nats.Conn, subs *[]*nats.Subscription, messageHandler *service.MessageCommandHandler) error {
	sub, err := conn.QueueSubscribe(subjects.MessageSendCommandSubject, subjects.MessageSendQueueGroup, messageHandler.HandleMessageSend)
	if err != nil {
		return fmt.Errorf("subscribe message send: %w", err)
	}
	*subs = append(*subs, sub)
	logger.Infof(ctx, "[message-server] Registered NATS subscription: %s", subjects.MessageSendCommandSubject)
	return nil
}
