package msgx

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/nats-io/nats.go"
)

const (
	ResponseCodeHeader    = "code"
	ResponseMessageHeader = "msg"
	ResponseCodeSuccess   = "0"
	ResponseCodeFailure   = "-1"
)

// NatsMsgInfo NATS 消息解析结果。
type NatsMsgInfo[T any] struct {
	RequestUser string // 请求用户（实际发起请求的用户）
	TraceId     string // 追踪ID
	Data        T      // 解析后的结构体数据
}

// BuildJSONRequest 基于 context 构建带 trace/request-user 透传的 JSON 请求消息。
func BuildJSONRequest(ctx context.Context, subject string, data interface{}) (*nats.Msg, error) {
	msg := contextx.CtxToTraceNats(ctx, subject)

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal request payload: %w", err)
	}
	msg.Data = payload
	return msg, nil
}

// RequestJSON 发送 JSON request-reply，并按统一 code/msg 头解析响应。
func RequestJSON(ctx context.Context, conn *nats.Conn, subject string, data interface{}, resp interface{}, timeout time.Duration) (*nats.Msg, error) {
	if conn == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}

	msg, err := BuildJSONRequest(ctx, subject, data)
	if err != nil {
		return nil, err
	}

	requestMsg, err := conn.RequestMsg(msg, timeout)
	if err != nil {
		return requestMsg, err
	}

	if requestMsg.Header.Get(ResponseCodeHeader) != ResponseCodeSuccess {
		message := requestMsg.Header.Get(ResponseMessageHeader)
		if message == "" {
			message = "request failed"
		}
		return requestMsg, fmt.Errorf("%s", message)
	}

	if resp == nil {
		return requestMsg, nil
	}
	if err := json.Unmarshal(requestMsg.Data, resp); err != nil {
		return requestMsg, err
	}
	return requestMsg, nil
}

// RespondJSONSuccess 返回统一的成功响应。
func RespondJSONSuccess(rsp *nats.Msg, data interface{}) error {
	msg := nats.NewMsg(rsp.Subject)
	msg.Header.Set(ResponseCodeHeader, ResponseCodeSuccess)
	msg.Header.Set(ResponseMessageHeader, "ok")

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal success response: %w", err)
	}
	msg.Data = payload
	return rsp.RespondMsg(msg)
}

// RespondJSONFailure 返回统一的失败响应。
func RespondJSONFailure(rsp *nats.Msg, err error) error {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}

	msg := nats.NewMsg(rsp.Subject)
	msg.Header.Set(ResponseCodeHeader, ResponseCodeFailure)
	msg.Header.Set(ResponseMessageHeader, err.Error())
	return rsp.RespondMsg(msg)
}

// DecodeJSON 统一解析 NATS 消息，提取 header 信息和 body 数据。
func DecodeJSON[T any](msg *nats.Msg) (*NatsMsgInfo[T], error) {
	var data T

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message data: %w", err)
	}

	info := &NatsMsgInfo[T]{
		RequestUser: msg.Header.Get(contextx.RequestUserHeader),
		TraceId:     msg.Header.Get(contextx.TraceIdHeader),
		Data:        data,
	}

	return info, nil
}
