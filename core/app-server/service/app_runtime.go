package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/waiter"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type AppRuntime struct {
	waiter      *waiter.ResponseWaiter
	config      *config.AppServerConfig
	natsService *NatsService
	subs        []*nats.Subscription // 添加订阅管理
}

// NewAppRuntimeService 创建 AppRuntime 服务（依赖注入）
func NewAppRuntimeService(cfg *config.AppServerConfig, natsService *NatsService) *AppRuntime {
	appRuntime := &AppRuntime{
		waiter:      waiter.GetDefaultWaiter(), // 内部初始化 waiter
		config:      cfg,
		natsService: natsService,
		subs:        []*nats.Subscription{},
	}

	// 初始化订阅
	appRuntime.initSubscriptions()

	return appRuntime
}

// CreateApp 创建应用
func (a *AppRuntime) CreateApp(ctx context.Context, hostId int64, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	var resp dto.CreateAppResp
	timeout := time.Duration(a.config.GetNatsRequestTimeout()) * time.Second

	conn, err := a.natsService.GetNatsByHost(hostId)
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

	conn, err := a.natsService.GetNatsByHost(hostId)
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
	conn, err := a.natsService.GetNatsByHost(natsId)
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

	conn, err := a.natsService.GetNatsByHost(hostId)
	if err != nil {
		return nil, err
	}

	_, err = msgx.RequestMsgWithTimeout(ctx, conn, subjects.GetAppServer2AppRuntimeDeleteRequestSubject(), req, &resp, timeout)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateServiceTree 创建服务目录
func (a *AppRuntime) CreateServiceTree(ctx context.Context, hostId int64, req *dto.CreateServiceTreeRuntimeReq) (*dto.CreateServiceTreeRuntimeResp, error) {
	var resp dto.CreateServiceTreeRuntimeResp
	timeout := time.Duration(a.config.GetNatsRequestTimeout()) * time.Second

	conn, err := a.natsService.GetNatsByHost(hostId)
	if err != nil {
		return nil, err
	}

	_, err = msgx.RequestMsgWithTimeout(ctx, conn, subjects.GetAppRuntime2ServiceTreeCreateRequestSubject(), req, &resp, timeout)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// initSubscriptions 初始化 NATS 订阅
func (a *AppRuntime) initSubscriptions() {
	// 获取所有可用的 NATS 连接
	for hostId := range a.natsService.hostIdMap {
		conn, err := a.natsService.GetNatsByHost(hostId)
		if err != nil {
			continue
		}

		// 订阅应用响应主题
		sub, err := conn.Subscribe(subjects.GetApp2FunctionServerResponseSubject(), a.HandleApp2FunctionServerResponse)
		if err != nil {
			fmt.Printf("[AppRuntime] Failed to subscribe to response subject on host %d: %v\n", hostId, err)
			continue
		}

		a.subs = append(a.subs, sub)
	}
}

// HandleApp2FunctionServerResponse 处理应用返回的响应
func (a *AppRuntime) HandleApp2FunctionServerResponse(msg *nats.Msg) {
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
	if !a.waiter.Notify(resp.TraceId, &resp) {
		// 如果没有找到等待的请求，记录日志
		fmt.Printf("[AppRuntime] No waiting request found for traceId: %s\n", resp.TraceId)
	}
}

// Close 关闭 AppRuntime 服务
func (a *AppRuntime) Close() error {
	// 取消所有订阅
	for _, sub := range a.subs {
		if sub != nil {
			sub.Unsubscribe()
		}
	}
	a.subs = []*nats.Subscription{}
	return nil
}
