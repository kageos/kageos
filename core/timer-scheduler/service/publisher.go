package service

import (
	"context"
	"fmt"
	"github.com/kageos/kageos/dto"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type OutboxPublisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

type NATSOutboxPublisher struct {
	conn *nats.Conn
}

func NewNATSOutboxPublisher(conn *nats.Conn) *NATSOutboxPublisher {
	return &NATSOutboxPublisher{conn: conn}
}

func (p *NATSOutboxPublisher) Publish(ctx context.Context, subject string, payload []byte) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("timer-scheduler: nats connection is nil")
	}
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("timer-scheduler: outbox subject is empty")
	}
	if subject == dto.TaskAuditSubject {
		requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		resp, err := p.conn.RequestWithContext(requestCtx, subject, payload)
		if err != nil {
			return err
		}
		if string(resp.Data) != "ok" {
			return fmt.Errorf("task audit not acknowledged")
		}
		return nil
	}
	if err := p.conn.Publish(subject, payload); err != nil {
		return err
	}
	return p.conn.FlushTimeout(2 * time.Second)
}
