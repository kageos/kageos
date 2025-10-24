package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/upstrem"
	"github.com/ai-agent-os/ai-agent-os/pkg/waiter"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type AppRuntime struct {
	waiter *waiter.ResponseWaiter
	config *config.AppServerConfig
}

// NewDefaultAppRuntimeService 创建 AppRuntime 服务（默认，内部获取依赖）
func NewDefaultAppRuntimeService() *AppRuntime {
	cfg := config.GetAppServerConfig()
	return NewAppRuntimeService(waiter.GetDefaultWaiter(), cfg)
}

// NewAppRuntimeService 创建 AppRuntime 服务（依赖注入）
func NewAppRuntimeService(waiter *waiter.ResponseWaiter, cfg *config.AppServerConfig) *AppRuntime {
	return &AppRuntime{
		waiter: waiter,
		config: cfg,
	}
}

// CreateApp 创建应用
func (a *AppRuntime) CreateApp(ctx context.Context, hostId int64, req interface{}) (*dto.CreateAppResp, error) {
	var resp dto.CreateAppResp
	timeout := time.Duration(a.config.GetNatsRequestTimeout()) * time.Second

	conn, err := upstrem.GetNatsService().GetNatsByHost(hostId)
	if err != nil {
		return nil, err
	}

	_, err = msgx.RequestMsgWithTimeout(ctx, conn, subjects.GetAppRuntime2AppCreateRequestSubject(), req, &resp, timeout)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateApp 更新应用
func (a *AppRuntime) UpdateApp(ctx context.Context, hostId int64, req *dto.UpdateAppReq) (*dto.UpdateAppResp, error) {
	var resp dto.UpdateAppResp
	timeout := time.Duration(a.config.GetNatsRequestTimeout()) * time.Second

	conn, err := upstrem.GetNatsService().GetNatsByHost(hostId)
	if err != nil {
		return nil, err
	}

	_, err = msgx.RequestMsgWithTimeout(ctx, conn, subjects.GetAppRuntime2AppUpdateRequestSubject(), req, &resp, timeout)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RequestApp 请求应用（异步等待响应）
func (a *AppRuntime) RequestApp(ctx context.Context, natsId int64, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {

	// 发送到 app-runtime，由 app-runtime 转发给具体的 app
	subject := subjects.BuildFunctionServer2AppRuntimeSubject(req.User, req.App, req.Version)
	conn, err := upstrem.GetNatsService().GetNatsByHost(natsId)
	if err != nil {
		return nil, err
	}
	// 序列化请求
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 创建带 header 的消息
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  make(nats.Header),
	}
	msg.Header.Set("trace_id", req.TraceId)
	msg.Header.Set("request_user", req.RequestUser)
	msg.Header.Set("user", req.RequestUser)
	msg.Header.Set("method", req.Method)
	msg.Header.Set("router", req.Router)
	msg.Header.Set("app", req.App)
	msg.Header.Set("user", req.User)
	msg.Header.Set("version", req.Version)
	// 发送消息（不等待响应）
	if err := conn.PublishMsg(msg); err != nil {
		return nil, fmt.Errorf("publish request failed: %w", err)
	}

	// 使用注入的配置获取超时时间
	timeout := time.Duration(a.config.GetAppRequestTimeout()) * time.Second

	resp, err := a.waiter.Wait(ctx, req.TraceId, timeout)
	if err != nil {
		return nil, fmt.Errorf("wait response timeout: %w", err)
	}

	return resp, nil
}

// DeleteApp 删除应用
func (a *AppRuntime) DeleteApp(ctx context.Context, hostId int64, req *dto.DeleteAppReq) (*dto.DeleteAppResp, error) {
	var resp dto.DeleteAppResp
	timeout := time.Duration(a.config.GetNatsRequestTimeout()) * time.Second

	conn, err := upstrem.GetNatsService().GetNatsByHost(hostId)
	if err != nil {
		return nil, err
	}

	_, err = msgx.RequestMsgWithTimeout(ctx, conn, subjects.GetAppServer2AppRuntimeDeleteRequestSubject(), req, &resp, timeout)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
