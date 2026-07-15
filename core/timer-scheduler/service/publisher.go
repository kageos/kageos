package service

import (
	"context"
	"fmt"
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
	if err := p.conn.Publish(subject, payload); err != nil {
		return err
	}
	return p.conn.FlushTimeout(2 * time.Second)
}
