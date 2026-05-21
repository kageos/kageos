// Package appinvoke 收敛 runtime <-> app 调用链里的主题与请求头语义。
package appinvoke

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// RequestMeta 描述 runtime -> app invoke 请求的路由与调用头信息。
type RequestMeta struct {
	TraceID         string
	RequestUser     string
	RequestUserDept string
	Token           string
	ClientSource    string
	User            string
	App             string
	Version         string
	Method          string
	Router          string
}

// BuildRuntimeRequestMsg 构建发往 runtime 的 invoke 请求消息。
func BuildRuntimeRequestMsg(req *dto.RequestAppReq) (*nats.Msg, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	meta := RequestMeta{
		TraceID:         req.TraceId,
		RequestUser:     req.RequestUser,
		RequestUserDept: req.RequestUserDept,
		Token:           req.Token,
		ClientSource:    req.ClientSource,
		User:            req.User,
		App:             req.App,
		Version:         req.Version,
		Method:          req.Method,
		Router:          req.Router,
	}

	msg := &nats.Msg{
		Subject: meta.RuntimeSubject(),
		Data:    data,
		Header:  make(nats.Header),
	}
	meta.ApplyHeaders(msg.Header)
	return msg, nil
}

// ParseRuntimeRequest 从 runtime 收到的 NATS 消息里提取 invoke 请求头信息。
func ParseRuntimeRequest(msg *nats.Msg) (*RequestMeta, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	meta := &RequestMeta{
		TraceID:         msg.Header.Get(contextx.TraceIdHeader),
		RequestUser:     msg.Header.Get(contextx.RequestUserHeader),
		RequestUserDept: msg.Header.Get(contextx.DepartmentFullPathHeader),
		Token:           msg.Header.Get(contextx.TokenHeader),
		ClientSource:    msg.Header.Get(contextx.ClientSourceHeader),
		User:            msg.Header.Get("user"),
		App:             msg.Header.Get("app"),
		Version:         msg.Header.Get("version"),
		Method:          msg.Header.Get("method"),
		Router:          msg.Header.Get("router"),
	}
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	return meta, nil
}

// Validate 校验 invoke 请求最小必需的目标路由信息。
func (m *RequestMeta) Validate() error {
	if m == nil {
		return fmt.Errorf("request meta is nil")
	}

	missing := make([]string, 0, 3)
	if m.User == "" {
		missing = append(missing, "user")
	}
	if m.App == "" {
		missing = append(missing, "app")
	}
	if m.Version == "" {
		missing = append(missing, "version")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing invoke headers: %s", strings.Join(missing, ", "))
	}
	return nil
}

// RuntimeSubject 返回发往 app-runtime 的 invoke 主题。
func (m *RequestMeta) RuntimeSubject() string {
	return subjects.BuildRuntimeAppInvokeCommandSubject(m.User, m.App, m.Version)
}

// AppSubject 返回 runtime 转发给 app 的 invoke 主题。
func (m *RequestMeta) AppSubject() string {
	return subjects.BuildAppInvokeSubject(m.User, m.App, m.Version)
}

// ApplyHeaders 将 invoke 请求头写入 NATS header。
func (m *RequestMeta) ApplyHeaders(header nats.Header) {
	if header == nil || m == nil {
		return
	}

	header.Set(contextx.TraceIdHeader, m.TraceID)
	header.Set(contextx.RequestUserHeader, m.RequestUser)
	if m.RequestUserDept != "" {
		header.Set(contextx.DepartmentFullPathHeader, m.RequestUserDept)
	}
	header.Set("method", m.Method)
	header.Set("router", m.Router)
	header.Set("app", m.App)
	header.Set("user", m.User)
	header.Set("version", m.Version)
	if m.Token != "" {
		header.Set(contextx.TokenHeader, m.Token)
	}
	if m.ClientSource != "" {
		header.Set(contextx.ClientSourceHeader, m.ClientSource)
	}
}
