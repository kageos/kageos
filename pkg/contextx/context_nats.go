package contextx

import (
	"context"

	"github.com/nats-io/nats.go"
)

func NatsTraceContext(msg *nats.Msg) context.Context {
	ctx := context.Background()
	if msg == nil {
		return ctx
	}

	// NATS is a process boundary, so rebuild the complete identity and audit
	// context from the same centralized list used by HTTP trust boundaries.
	// Callers that require authenticated identity must verify the message before
	// using this context.
	for _, key := range append(TrustedIdentityHeaderNames(), TokenHeader, TraceIdHeader) {
		if value := msg.Header.Get(key); value != "" {
			ctx = context.WithValue(ctx, key, value)
		}
	}

	return ctx
}

func CtxToTraceNats(c context.Context, subject string) *nats.Msg {
	msg := nats.NewMsg(subject)
	for _, key := range TrustedIdentityHeaderNames() {
		if value := getStringFromContextOrHeader(c, key); value != "" {
			msg.Header.Set(key, value)
		}
	}
	if token := GetToken(c); token != "" {
		msg.Header.Set(TokenHeader, token)
	}
	if trace := GetTraceId(c); trace != "" {
		msg.Header.Set(TraceIdHeader, trace)
	}
	return msg
}
