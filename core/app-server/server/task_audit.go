package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

func (s *Server) startTaskAuditConsumer() error {
	if s.natsConn == nil {
		return fmt.Errorf("task audit NATS connection unavailable")
	}
	sub, err := s.natsConn.QueueSubscribe(dto.TaskAuditSubject, "app-server-task-audit", func(msg *nats.Msg) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var event dto.TaskAuditEvent
		err := json.Unmarshal(msg.Data, &event)
		if err == nil {
			err = service.RecordTaskAudit(ctx, s.db, event)
		}
		if err != nil {
			logger.Warnf(ctx, "[TaskAudit] persist failed: %v", err)
			_ = msg.Respond([]byte("retry"))
			return
		}
		_ = msg.Respond([]byte("ok"))
	})
	if err != nil {
		return err
	}
	s.taskAuditSub = sub
	if err := s.natsConn.Flush(); err != nil {
		_ = sub.Unsubscribe()
		s.taskAuditSub = nil
		return err
	}
	return nil
}
