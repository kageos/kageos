package msgx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"time"

	"github.com/nats-io/nats.go"
)

func RequestMsg(conn *nats.Conn, subject string, data interface{}, resp interface{}) (rsp *nats.Msg, err error) {
	// 使用默认超时时间
	timeout := time.Duration(100) * time.Second
	return RequestMsgWithTimeout(context.Background(), conn, subject, data, resp, timeout)
}

func RequestMsgWithTimeout(ctx context.Context, conn *nats.Conn, subject string, data interface{}, resp interface{}, timeout time.Duration) (rsp *nats.Msg, err error) {

	msg := contextx.CtxToTraceNats(ctx, subject)
	//msg := nats.NewMsg(subject)
	marshal, _ := json.Marshal(data)
	msg.Data = marshal

	//// 从 context 中获取请求用户信息并添加到 NATS header
	//requestUser := contextx.GetRequestUser(ctx)
	//if requestUser != "" {
	//	msg.Header.Set(contextx.RequestUserHeader, requestUser)
	//}
	//
	//token := contextx.GetRequestUser(ctx)
	//if requestUser != "" {
	//	msg.Header.Set(contextx.RequestUserHeader, requestUser)
	//}
	//
	//// 从 context 中获取追踪ID并添加到 NATS header
	//if traceId := contextx.GetTraceId(ctx); traceId != "" {
	//	msg.Header.Set(contextx.TraceIdHeader, traceId)
	//}

	//contextx.ToContext()

	requestMsg, err := conn.RequestMsg(msg, timeout)
	if err != nil {
		return requestMsg, err
	}
	if requestMsg.Header.Get("code") != "0" {
		return requestMsg, fmt.Errorf("%s", requestMsg.Header.Get("msg"))
	}
	err = json.Unmarshal(requestMsg.Data, resp)
	if err != nil {
		return requestMsg, err
	}
	return requestMsg, nil
}

func RespSuccessMsg(rsp *nats.Msg, data interface{}) error {

	msg := nats.NewMsg(rsp.Subject)
	msg.Header.Set("code", "0")
	msg.Header.Set("msg", "ok")
	marshal, _ := json.Marshal(data)
	msg.Data = marshal
	return rsp.RespondMsg(msg)
}

func RespFailMsg(rsp *nats.Msg, err error) error {

	msg := nats.NewMsg(rsp.Subject)
	msg.Header.Set("code", "-1")
	msg.Header.Set("msg", err.Error())
	return rsp.RespondMsg(msg)
}

// NatsMsgInfo NATS 消息解析结果
type NatsMsgInfo[T any] struct {
	RequestUser string // 请求用户（实际发起请求的用户）
	TraceId     string // 追踪ID
	Data        T      // 解析后的结构体数据
}

// DecodeNatsMsg 统一解析 NATS 消息，提取 header 信息和 body 数据
func DecodeNatsMsg[T any](msg *nats.Msg) (*NatsMsgInfo[T], error) {
	var data T

	// 解析 body 数据
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message data: %w", err)
	}

	// 提取 header 信息
	info := &NatsMsgInfo[T]{
		RequestUser: msg.Header.Get("X-Request-User"), // 请求用户
		TraceId:     msg.Header.Get("X-Trace-Id"),     // 追踪ID
		Data:        data,
	}

	return info, nil
}
