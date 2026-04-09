package apicall

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

// NewContext 创建一个包含 token 和 traceId 的 context。
func NewContext(token, traceId string) context.Context {
	return NewContextWithClientSource(token, traceId, "")
}

// NewContextWithClientSource 创建一个包含 token、traceId 和 clientSource 的 context。
// 用于 SDK 等场景，当原始 context 不包含这些信息时。
func NewContextWithClientSource(token, traceId, clientSource string) context.Context {
	return buildAPIContext(context.Background(), token, traceId, clientSource)
}

// NewContextWithParent 基于父 context 创建一个包含 token 和 traceId 的新 context。
func NewContextWithParent(parent context.Context, token, traceId string) context.Context {
	return NewContextWithParentAndClientSource(parent, token, traceId, "")
}

// NewContextWithParentAndClientSource 基于父 context 创建一个包含 token、traceId 和 clientSource 的新 context。
// 如果对应值为空，则优先沿用父 context 中已有的值。
func NewContextWithParentAndClientSource(parent context.Context, token, traceId, clientSource string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	if token == "" {
		token = contextx.GetToken(parent)
	}
	if traceId == "" {
		traceId = contextx.GetTraceId(parent)
	}
	if clientSource == "" {
		clientSource = contextx.GetClientSource(parent)
	}
	return buildAPIContext(parent, token, traceId, clientSource)
}

func buildAPIContext(parent context.Context, token, traceId, clientSource string) context.Context {
	ctx := parent
	if token != "" {
		ctx = context.WithValue(ctx, contextx.TokenHeader, token)
	}
	if traceId != "" {
		ctx = context.WithValue(ctx, contextx.TraceIdHeader, traceId)
	}
	if clientSource != "" {
		ctx = context.WithValue(ctx, contextx.ClientSourceHeader, clientSource)
	}
	return ctx
}
