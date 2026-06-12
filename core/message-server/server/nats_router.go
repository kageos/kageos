package server

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/message-server/service"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func RegisterNATS(ctx context.Context, conn *nats.Conn, subs *[]*nats.Subscription, messageHandler *service.MessageCommandHandler) error {
	if conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	sub, err := conn.QueueSubscribe(subjects.MessageSendCommandSubject, subjects.MessageSendQueueGroup, messageHandler.HandleMessageSend)
	if err != nil {
		return fmt.Errorf("subscribe message send: %w", err)
	}
	*subs = append(*subs, sub)
	logger.Infof(ctx, "[message-server] Registered NATS subscription: %s", subjects.MessageSendCommandSubject)
	return nil
}
