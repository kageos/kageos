package v1

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/nats-io/nats.go"
)

func handlerContext(msg *nats.Msg) context.Context {
	return contextx.NatsTraceContext(msg)
}

func decodeRequest[T any](ctx context.Context, msg *nats.Msg, op string) (*T, bool) {
	msgInfo, err := msgx.DecodeJSON[T](msg)
	if err != nil {
		logger.Errorf(ctx, "[%s] Failed to decode message: %v", op, err)
		respondFailure(ctx, msg, op, err)
		return nil, false
	}
	return &msgInfo.Data, true
}

func respondSuccess(ctx context.Context, msg *nats.Msg, op string, data interface{}) bool {
	if err := msgx.RespondJSONSuccess(msg, data); err != nil {
		logger.Errorf(ctx, "[%s] Failed to respond success: %v", op, err)
		return false
	}
	return true
}

func respondFailure(ctx context.Context, msg *nats.Msg, op string, err error) {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	if respErr := msgx.RespondJSONFailure(msg, err); respErr != nil {
		logger.Errorf(ctx, "[%s] Failed to respond failure: %v", op, respErr)
	}
}
