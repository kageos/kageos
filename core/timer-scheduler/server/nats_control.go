package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func startTimerNATSControl(conn *nats.Conn, service *timerservice.Service, commandVerifier *controlauth.Verifier, responseSigner *controlauth.Signer) ([]*nats.Subscription, error) {
	if conn == nil {
		return nil, fmt.Errorf("timer-scheduler nats control requires nats connection")
	}
	if service == nil {
		return nil, fmt.Errorf("timer-scheduler nats control requires service")
	}
	if commandVerifier == nil || responseSigner == nil {
		return nil, fmt.Errorf("timer-scheduler nats control requires authentication")
	}

	routes := []struct {
		subject string
		handler nats.MsgHandler
	}{
		{
			subject: subjects.TimerExecutionStartedCommandSubject,
			handler: func(msg *nats.Msg) {
				var req scheduledsdk.MarkExecutionStartedRequest
				handleTimerNATSCommand(msg, commandVerifier, responseSigner, &req, func(ctx context.Context) error {
					return service.MarkExecutionStarted(ctx, req)
				})
			},
		},
		{
			subject: subjects.TimerExecutionHeartbeatCommandSubject,
			handler: func(msg *nats.Msg) {
				var req scheduledsdk.MarkExecutionHeartbeatRequest
				handleTimerNATSCommand(msg, commandVerifier, responseSigner, &req, func(ctx context.Context) error {
					return service.MarkExecutionHeartbeat(ctx, req)
				})
			},
		},
		{
			subject: subjects.TimerExecutionFinishedCommandSubject,
			handler: func(msg *nats.Msg) {
				var req scheduledsdk.MarkExecutionFinishedRequest
				handleTimerNATSCommand(msg, commandVerifier, responseSigner, &req, func(ctx context.Context) error {
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

func handleTimerNATSCommand(msg *nats.Msg, commandVerifier *controlauth.Verifier, responseSigner *controlauth.Signer, req interface{}, run func(context.Context) error) {
	handleTimerNATSCommandWithResponder(msg, commandVerifier, responseSigner, req, run, respondTimerNATSCommand)
}

type timerNATSCommandResponder func(*nats.Msg, *controlauth.Signer, string, string, error) error

func handleTimerNATSCommandWithResponder(
	msg *nats.Msg,
	commandVerifier *controlauth.Verifier,
	responseSigner *controlauth.Signer,
	req interface{},
	run func(context.Context) error,
	respond timerNATSCommandResponder,
) {
	ctx := context.Background()
	if err := controlauth.VerifyNATSMessage(msg, commandVerifier); err != nil {
		// Reply is attacker-controlled until the command is authenticated. Never
		// turn the scheduler into a signed-response oracle or reflection source.
		logger.Warnf(ctx, "[timer-scheduler] rejected unauthenticated nats control command subject=%s err=%v", natsMessageSubject(msg), err)
		return
	}
	requestNonce := msg.Header.Get(controlauth.NATSNonceHeader)
	requestSubject := msg.Subject
	ctx = timerNATSContext(msg)
	err := json.Unmarshal(msg.Data, req)
	if err == nil {
		err = run(ctx)
	}
	if err != nil {
		logger.Warnf(ctx, "[timer-scheduler] nats control command failed subject=%s err=%v", msg.Subject, err)
	}
	if respond == nil {
		return
	}
	if responseErr := respond(msg, responseSigner, requestNonce, requestSubject, err); responseErr != nil {
		logger.Warnf(ctx, "[timer-scheduler] nats control response failed subject=%s err=%v", msg.Subject, responseErr)
	}
}

func natsMessageSubject(msg *nats.Msg) string {
	if msg == nil {
		return ""
	}
	return msg.Subject
}

type timerNATSCommandResponse struct {
	OK             bool   `json:"ok"`
	Error          string `json:"error,omitempty"`
	RequestNonce   string `json:"request_nonce"`
	RequestSubject string `json:"request_subject"`
}

func respondTimerNATSCommand(msg *nats.Msg, signer *controlauth.Signer, requestNonce, requestSubject string, err error) error {
	if msg == nil || msg.Reply == "" {
		return nil
	}
	response, buildErr := buildTimerNATSCommandResponse(msg.Reply, signer, requestNonce, requestSubject, err)
	if buildErr != nil {
		return buildErr
	}
	return msg.RespondMsg(response)
}

func buildTimerNATSCommandResponse(reply string, signer *controlauth.Signer, requestNonce, requestSubject string, err error) (*nats.Msg, error) {
	if reply == "" {
		return nil, fmt.Errorf("timer nats response reply subject is required")
	}
	if requestNonce == "" || requestSubject == "" {
		return nil, fmt.Errorf("timer nats response request binding is required")
	}
	resp := timerNATSCommandResponse{
		OK:             err == nil,
		RequestNonce:   requestNonce,
		RequestSubject: requestSubject,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	data, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		data = []byte(`{"ok":false,"error":"failed to encode nats response"}`)
	}
	response := nats.NewMsg(reply)
	response.Data = data
	if signErr := controlauth.SignNATSMessage(response, signer); signErr != nil {
		return nil, fmt.Errorf("authenticate timer nats response: %w", signErr)
	}
	return response, nil
}

func timerNATSContext(msg *nats.Msg) context.Context {
	if msg == nil || len(msg.Header) == 0 {
		return context.Background()
	}
	return contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:            msg.Header.Get(contextx.TraceIdHeader),
		RequestUser:        msg.Header.Get(contextx.RequestUserHeader),
		DepartmentFullPath: msg.Header.Get(contextx.DepartmentFullPathHeader),
		CompanyCode:        msg.Header.Get(contextx.CompanyCodeHeader),
		CompanyName:        msg.Header.Get(contextx.CompanyNameHeader),
		CompanyLogoURL:     msg.Header.Get(contextx.CompanyLogoURLHeader),
		ClientSource:       msg.Header.Get(contextx.ClientSourceHeader),
		SourceType:         msg.Header.Get(contextx.SourceTypeHeader),
		SourceRef:          msg.Header.Get(contextx.SourceRefHeader),
	})
}
