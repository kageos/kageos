package apicall

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

// NewContext 创建一个包含 token 和 traceId 的 context
// 用于 SDK 等场景，当原始 context 不包含这些信息时
// token: 认证 token（可选）
// traceId: 追踪 ID（可选）
// 返回: 包含 token 和 traceId 的 context.Context
// ⭐ 只使用 contextx 包中的常量 key（TokenHeader、TraceIdHeader）
func NewContext(token, traceId string) context.Context {
	ctx := context.Background()

	// 设置 token（只使用 TokenHeader 常量）
	if token != "" {
		ctx = context.WithValue(ctx, contextx.TokenHeader, token)
	}

	// 设置 traceId（只使用 TraceIdHeader 常量）
	if traceId != "" {
		ctx = context.WithValue(ctx, contextx.TraceIdHeader, traceId)
	}

	return ctx
}

// NewContextWithParent 基于父 context 创建一个包含 token 和 traceId 的新 context
// 如果父 context 中已有 token 或 traceId，则优先使用父 context 的值
// 如果提供了新的 token 或 traceId，则覆盖父 context 的值
// parent: 父 context（可选，如果为 nil 则使用 context.Background()）
// token: 认证 token（可选，如果为空则尝试从 parent 中获取）
// traceId: 追踪 ID（可选，如果为空则尝试从 parent 中获取）
// 返回: 包含 token 和 traceId 的 context.Context
// ⭐ 只使用 contextx 包中的常量 key（TokenHeader、TraceIdHeader）
func NewContextWithParent(parent context.Context, token, traceId string) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	// 如果未提供 token，尝试从 parent 中获取
	if token == "" {
		token = contextx.GetToken(parent)
	}

	// 如果未提供 traceId，尝试从 parent 中获取
	if traceId == "" {
		traceId = contextx.GetTraceId(parent)
	}

	ctx := parent

	// 设置 token（只使用 TokenHeader 常量）
	if token != "" {
		ctx = context.WithValue(ctx, contextx.TokenHeader, token)
	}

	// 设置 traceId（只使用 TraceIdHeader 常量）
	if traceId != "" {
		ctx = context.WithValue(ctx, contextx.TraceIdHeader, traceId)
	}

	return ctx
}
