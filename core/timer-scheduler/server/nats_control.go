package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func startTimerNATSControl(conn *nats.Conn, service *timerservice.Service) ([]*nats.Subscription, error) {
	if conn == nil {
		return nil, fmt.Errorf("timer-scheduler nats control requires nats connection")
	}
	if service == nil {
		return nil, fmt.Errorf("timer-scheduler nats control requires service")
	}

	routes := []struct {
		subject string
		handler nats.MsgHandler
	}{
		{
			subject: subjects.TimerExecutionStartedCommandSubject,
			handler: func(msg *nats.Msg) {
				var req scheduledsdk.MarkExecutionStartedRequest
				handleTimerNATSCommand(msg, &req, func(ctx context.Context) error {
					return service.MarkExecutionStarted(ctx, req)
				})
			},
		},
		{
			subject: subjects.TimerExecutionHeartbeatCommandSubject,
			handler: func(msg *nats.Msg) {
				var req scheduledsdk.MarkExecutionHeartbeatRequest
				handleTimerNATSCommand(msg, &req, func(ctx context.Context) error {
					return service.MarkExecutionHeartbeat(ctx, req)
				})
			},
		},
		{
			subject: subjects.TimerExecutionFinishedCommandSubject,
			handler: func(msg *nats.Msg) {
				var req scheduledsdk.MarkExecutionFinishedRequest
				handleTimerNATSCommand(msg, &req, func(ctx context.Context) error {
					return service.MarkExecutionFinished(ctx, req)
				})
			},
		},
	}

	subs := make([]*nats.Subscription, 0, len(routes))
	for _, route := range routes {
		sub, err := conn.QueueSubscribe(route.subject, subjects.TimerExecutionControlQueueGroup, route.handler)
		if err != nil {
			for _, existing := range subs {
				_ = existing.Unsubscribe()
			}
			return nil, err
		}
		subs = append(subs, sub)
	}
	if err := conn.FlushTimeout(2 * time.Second); err != nil {
		for _, sub := range subs {
			_ = sub.Unsubscribe()
		}
		return nil, err
	}
	return subs, nil
}

func handleTimerNATSCommand(msg *nats.Msg, req interface{}, run func(context.Context) error) {
	ctx := timerNATSContext(msg)
	err := json.Unmarshal(msg.Data, req)
	if err == nil {
		err = run(ctx)
	}
	if err != nil {
		logger.Warnf(ctx, "[timer-scheduler] nats control command failed subject=%s err=%v", msg.Subject, err)
	}
	respondTimerNATSCommand(msg, err)
}

type timerNATSCommandResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func respondTimerNATSCommand(msg *nats.Msg, err error) {
	if msg == nil || msg.Reply == "" {
		return
	}
	resp := timerNATSCommandResponse{OK: err == nil}
	if err != nil {
		resp.Error = err.Error()
	}
	data, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		data = []byte(`{"ok":false,"error":"failed to encode nats response"}`)
	}
	_ = msg.Respond(data)
}

func timerNATSContext(msg *nats.Msg) context.Context {
	if msg == nil || len(msg.Header) == 0 {
		return context.Background()
	}
	return contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:            msg.Header.Get(contextx.TraceIdHeader),
		RequestUser:        msg.Header.Get(contextx.RequestUserHeader),
		Token:              msg.Header.Get(contextx.TokenHeader),
		DepartmentFullPath: msg.Header.Get(contextx.DepartmentFullPathHeader),
		ClientSource:       msg.Header.Get(contextx.ClientSourceHeader),
		SourceType:         msg.Header.Get(contextx.SourceTypeHeader),
		SourceRef:          msg.Header.Get(contextx.SourceRefHeader),
	})
}
