package subject

import (
	"encoding/json"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/nats-io/nats.go"
)

// ResponseNotifier 响应通知接口
type ResponseNotifier interface {
	Notify(traceId string, resp *dto.RequestAppResp) bool
}

// responseNotifier 全局响应通知器（由 global 层注入）
var responseNotifier ResponseNotifier

// SetResponseNotifier 设置响应通知器
func SetResponseNotifier(notifier ResponseNotifier) {
	responseNotifier = notifier
}

// HandleApp2FunctionServerResponse 处理应用返回的响应
func HandleApp2FunctionServerResponse(msg *nats.Msg) {
	// 解析响应
	var resp dto.RequestAppResp
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return
	}

	// 从消息头获取 traceId（如果有）
	if traceId := msg.Header.Get("trace_id"); traceId != "" {
		resp.TraceId = traceId
	}

	// 通知等待的请求
	if responseNotifier != nil {
		if !responseNotifier.Notify(resp.TraceId, &resp) {
		}
	}
}
