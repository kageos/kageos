package chart

import (
	"context"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

// logViz 记录数据可视化各阶段耗时，便于排查网关/proxy 长时间无响应（如 context deadline exceeded）。
// op 为函数名（如 pie_chart），phase 为阶段说明。
// ctx 可以是 *app.Context（自动提取 traceId）或普通 context.Context。
func logViz(ctx context.Context, op, phase string, start time.Time) {
	traceId := ""
	if appCtx, ok := ctx.(*app.Context); ok {
		traceId = appCtx.GetTraceId()
	}
	logger.Infof(ctx, "[data_visualization:%s] traceId=%s, %s, elapsed=%s", op, traceId, phase, time.Since(start).Truncate(time.Millisecond))
}
