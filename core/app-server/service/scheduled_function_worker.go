package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/nats-io/nats.go"
)

const (
	ScheduledFunctionExecutorKey   = "app.function"
	internalTableGetRowsCallback   = "__table_get_rows"
	tableGetRowsCallbackHTTPMethod = http.MethodPost
)

type scheduledFunctionPayload struct {
	FullCodePath string          `json:"full_code_path"`
	TemplateType string          `json:"template_type,omitempty"`
	Action       string          `json:"action,omitempty"`
	Method       string          `json:"method,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Body         json.RawMessage `json:"body,omitempty"`
}

type scheduledFunctionRunResult struct {
	Content            string
	Data               interface{}
	IsError            bool
	OperateLogRecorded bool
}

type scheduledCallbackRequestEnvelope struct {
	Method string `json:"method"`
	Router string `json:"router"`
	Body   []byte `json:"body"`
	Type   string `json:"type"`
}

type scheduledTableGetRowsCallbackReq struct {
	IDs []int64 `json:"ids"`
}

func NewScheduledFunctionWorker(natsConn *nats.Conn, appService *AppService, controlPlaneSecret string) (*scheduledsdk.Worker, error) {
	if natsConn == nil {
		return nil, fmt.Errorf("scheduled function worker requires nats connection")
	}
	if appService == nil {
		return nil, fmt.Errorf("scheduled function worker requires app service")
	}
	natsAuth, err := scheduledsdk.NewWorkerNATSAuth(controlPlaneSecret)
	if err != nil {
		return nil, fmt.Errorf("scheduled function worker control auth: %w", err)
	}
	client := scheduledsdk.NewClient(scheduledsdk.Options{
		Adapter: scheduledsdk.NewNATSAdapter(natsConn, scheduledsdk.NATSAdapterOptions{
			CommandSigner:    natsAuth.CommandSigner,
			ResponseVerifier: natsAuth.ResponseVerifier,
		}),
	})
	return scheduledsdk.NewWorker(scheduledsdk.WorkerOptions{
		Client:          client,
		NATSConn:        natsConn,
		ExecutorKey:     ScheduledFunctionExecutorKey,
		MessageVerifier: natsAuth.MessageVerifier,
		Handler: func(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
			if strings.TrimSpace(event.RequestUser) == "" {
				return nil, fmt.Errorf("scheduled function request user is missing")
			}
			return appService.RunScheduledFunction(ctx, event)
		},
		OnError: func(ctx context.Context, err error) {
			logger.Warnf(ctx, "[ScheduledFunctionWorker] %v", err)
		},
	})
}
