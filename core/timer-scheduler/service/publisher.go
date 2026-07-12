package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type OutboxPublisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

type NATSOutboxPublisher struct {
	conn   *nats.Conn
	signer *controlauth.Signer
}

func NewNATSOutboxPublisher(conn *nats.Conn, signer *controlauth.Signer) *NATSOutboxPublisher {
	return &NATSOutboxPublisher{conn: conn, signer: signer}
}

func (p *NATSOutboxPublisher) Publish(ctx context.Context, subject string, payload []byte) error {
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("timer-scheduler: outbox subject is empty")
	}
	if err := validateTimerRequestedPayload(subject, payload); err != nil {
		return err
	}
	if p == nil || p.conn == nil {
		return fmt.Errorf("timer-scheduler: nats connection is nil")
	}
	msg := nats.NewMsg(subject)
	msg.Data = payload
	if err := controlauth.SignNATSMessage(msg, p.signer); err != nil {
		return fmt.Errorf("timer-scheduler: authenticate outbox message: %w", err)
	}
	if err := p.conn.PublishMsg(msg); err != nil {
		return err
	}
	return p.conn.FlushTimeout(2 * time.Second)
}

func validateTimerRequestedPayload(subject string, payload []byte) error {
	if !strings.HasPrefix(subject, subjects.TimerExecutionRequestedSubjectPrefix) {
		return nil
	}
	var event scheduledsdk.ExecutionRequestedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("timer-scheduler: decode execution request outbox payload: %w", err)
	}
	if strings.TrimSpace(event.Token) != "" {
		return fmt.Errorf("timer-scheduler: refuse execution request containing bearer token")
	}
	if subject != subjects.TimerExecutionRequestedSubject(event.ExecutorKey) {
		return fmt.Errorf("timer-scheduler: execution request subject does not match executor_key")
	}
	return nil
}
