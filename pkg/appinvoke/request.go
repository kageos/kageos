// Package appinvoke 收敛 runtime <-> app 调用链里的主题与请求头语义。
package appinvoke

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

const TargetRouterHeader = "X-kageos-Target-Router"

// RequestMeta 描述 runtime -> app invoke 请求的路由与调用头信息。
type RequestMeta struct {
	TraceID               string
	RequestUser           string
	RequestUserDept       string
	Token                 string
	AnonymousToken        string
	ClientSource          string
	SourceType            string
	SourceRef             string
	SourcePath            string
	SourceTitle           string
	SourceParentPath      string
	SourceParentTitle     string
	SourceTemplateType    string
	WorkspaceSessionID    string
	WorkspaceSessionTitle string
	WorkspaceRole         string
	InitiatorUser         string
	WorkspaceMessageID    int64
	ToolCallID            string
	ToolName              string
	User                  string
	App                   string
	Version               string
	Method                string
	Router                string
	TargetRouter          string
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
		TraceID:               req.TraceId,
		RequestUser:           req.RequestUser,
		RequestUserDept:       req.RequestUserDept,
		Token:                 req.Token,
		AnonymousToken:        req.AnonymousToken,
		ClientSource:          req.ClientSource,
		SourceType:            req.SourceType,
		SourceRef:             req.SourceRef,
		SourcePath:            req.SourcePath,
		SourceTitle:           req.SourceTitle,
		SourceParentPath:      req.SourceParentPath,
		SourceParentTitle:     req.SourceParentTitle,
		SourceTemplateType:    req.SourceTemplateType,
		WorkspaceSessionID:    req.WorkspaceSessionID,
		WorkspaceSessionTitle: req.WorkspaceSessionTitle,
		WorkspaceRole:         req.WorkspaceRole,
		InitiatorUser:         req.InitiatorUser,
		WorkspaceMessageID:    req.WorkspaceMessageID,
		ToolCallID:            req.ToolCallID,
		ToolName:              req.ToolName,
		User:                  req.User,
		App:                   req.App,
		Version:               req.Version,
		Method:                req.Method,
		Router:                req.Router,
		TargetRouter:          req.TargetRouter,
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
		TraceID:               msg.Header.Get(contextx.TraceIdHeader),
		RequestUser:           msg.Header.Get(contextx.RequestUserHeader),
		RequestUserDept:       msg.Header.Get(contextx.DepartmentFullPathHeader),
		Token:                 msg.Header.Get(contextx.TokenHeader),
		AnonymousToken:        msg.Header.Get("X-Public-Anonymous-Token"),
		ClientSource:          msg.Header.Get(contextx.ClientSourceHeader),
		SourceType:            msg.Header.Get(contextx.SourceTypeHeader),
		SourceRef:             msg.Header.Get(contextx.SourceRefHeader),
		SourcePath:            msg.Header.Get(contextx.SourcePathHeader),
		SourceTitle:           msg.Header.Get(contextx.SourceTitleHeader),
		SourceParentPath:      msg.Header.Get(contextx.SourceParentPathHeader),
		SourceParentTitle:     msg.Header.Get(contextx.SourceParentTitleHeader),
		SourceTemplateType:    msg.Header.Get(contextx.SourceTemplateTypeHeader),
		WorkspaceSessionID:    msg.Header.Get(contextx.WorkspaceSessionIDHeader),
		WorkspaceSessionTitle: msg.Header.Get(contextx.WorkspaceSessionTitleHeader),
		WorkspaceRole:         msg.Header.Get(contextx.WorkspaceRoleHeader),
		InitiatorUser:         msg.Header.Get(contextx.InitiatorUserHeader),
		WorkspaceMessageID:    parseInt64Header(msg.Header.Get(contextx.WorkspaceMessageIDHeader)),
		ToolCallID:            msg.Header.Get(contextx.ToolCallIDHeader),
		ToolName:              msg.Header.Get(contextx.ToolNameHeader),
		User:                  msg.Header.Get("user"),
		App:                   msg.Header.Get("app"),
		Version:               msg.Header.Get("version"),
		Method:                msg.Header.Get("method"),
		Router:                msg.Header.Get("router"),
		TargetRouter:          msg.Header.Get(TargetRouterHeader),
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
	if m.TargetRouter != "" {
		header.Set(TargetRouterHeader, m.TargetRouter)
	}
	if m.Token != "" {
		header.Set(contextx.TokenHeader, m.Token)
	}
	if m.AnonymousToken != "" {
		header.Set("X-Public-Anonymous-Token", m.AnonymousToken)
	}
	if m.ClientSource != "" {
		header.Set(contextx.ClientSourceHeader, m.ClientSource)
	}
	if m.SourceType != "" {
		header.Set(contextx.SourceTypeHeader, m.SourceType)
	}
	if m.SourceRef != "" {
		header.Set(contextx.SourceRefHeader, m.SourceRef)
	}
	for _, item := range []struct {
		key   string
		value string
	}{
		{contextx.SourcePathHeader, m.SourcePath},
		{contextx.SourceTitleHeader, m.SourceTitle},
		{contextx.SourceParentPathHeader, m.SourceParentPath},
		{contextx.SourceParentTitleHeader, m.SourceParentTitle},
		{contextx.SourceTemplateTypeHeader, m.SourceTemplateType},
		{contextx.WorkspaceSessionIDHeader, m.WorkspaceSessionID},
		{contextx.WorkspaceSessionTitleHeader, m.WorkspaceSessionTitle},
		{contextx.WorkspaceRoleHeader, m.WorkspaceRole},
		{contextx.InitiatorUserHeader, m.InitiatorUser},
		{contextx.WorkspaceMessageIDHeader, formatInt64Header(m.WorkspaceMessageID)},
		{contextx.ToolCallIDHeader, m.ToolCallID},
		{contextx.ToolNameHeader, m.ToolName},
	} {
		if item.value != "" {
			header.Set(item.key, item.value)
		}
	}
}

func parseInt64Header(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func formatInt64Header(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}
