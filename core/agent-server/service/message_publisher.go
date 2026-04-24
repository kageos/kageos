package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type scheduledAgentMessagePublisher interface {
	PublishMessage(ctx context.Context, payload *dto.MessageSendPayload) error
}

type NATSMessagePublisher struct {
	conn *nats.Conn
}

func NewNATSMessagePublisher(conn *nats.Conn) *NATSMessagePublisher {
	return &NATSMessagePublisher{conn: conn}
}

func (p *NATSMessagePublisher) PublishMessage(ctx context.Context, payload *dto.MessageSendPayload) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	msg, err := msgx.BuildJSONRequest(ctx, subjects.MessageSendCommandSubject, payload)
	if err != nil {
		return err
	}
	return p.conn.PublishMsg(msg)
}
