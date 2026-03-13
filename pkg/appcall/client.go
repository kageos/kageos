// Package appcall 提供调用 app / app-runtime 的 SDK 风格客户端（NATS request-reply + RequestApp）。
// 与 pkg/apicall（HTTP 调 API）对称：apicall 调 API，appcall 调 app。
package appcall

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/nats-io/nats.go"
)

// ConnProvider 按 hostId / natsId 提供 NATS 连接；HostIds 用于订阅时遍历所有连接。
type ConnProvider interface {
	GetNatsByHost(hostId int64) (*nats.Conn, error)
	GetNatsByNatsId(natsId int64) (*nats.Conn, error)
	HostIds() []int64
}

// Waiter RequestApp 发 Publish 后 Wait 等响应，响应回调里 Notify；需同一实例。
type Waiter interface {
	Wait(ctx context.Context, key string, timeout time.Duration) (*dto.RequestAppResp, error)
	Notify(key string, resp *dto.RequestAppResp) bool
}

// Options 创建 Client 时注入的依赖。
type Options struct {
	ConnProvider       ConnProvider
	NatsRequestTimeout time.Duration // 普通 request-reply 超时
	AppRequestTimeout  time.Duration // RequestApp 等待响应的超时
	Waiter             Waiter
}

// Client app-server 调用 app-runtime 的 SDK 风格客户端（NATS request-reply + RequestApp + 响应订阅）。
type Client struct {
	connProvider       ConnProvider
	natsRequestTimeout time.Duration
	appRequestTimeout  time.Duration
	waiter             Waiter
	subs               []*nats.Subscription
}

// New 创建 Client，并初始化响应主题订阅。
func New(opts Options) *Client {
	c := &Client{
		connProvider:       opts.ConnProvider,
		natsRequestTimeout: opts.NatsRequestTimeout,
		appRequestTimeout:  opts.AppRequestTimeout,
		waiter:             opts.Waiter,
		subs:               make([]*nats.Subscription, 0),
	}
	c.initSubscriptions()
	return c
}

func (c *Client) requestByHost(ctx context.Context, hostId int64, subject string, req, resp interface{}) error {
	conn, err := c.connProvider.GetNatsByHost(hostId)
	if err != nil {
		return err
	}
	_, err = msgx.RequestMsgWithTimeout(ctx, conn, subject, req, resp, c.natsRequestTimeout)
	return err
}

// CreateApp 创建应用（subject: app_runtime.app.create）
func (c *Client) CreateApp(ctx context.Context, hostId int64, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	var resp dto.CreateAppResp
	if err := c.requestByHost(ctx, hostId, "app_runtime.app.create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateApp 更新应用（subject: app_runtime.app.update）
func (c *Client) UpdateApp(ctx context.Context, hostId int64, req *dto.UpdateAppReq) (*dto.UpdateAppResp, error) {
	var resp dto.UpdateAppResp
	if err := c.requestByHost(ctx, hostId, "app_runtime.app.update", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteApp 删除应用（subject: app_server.app_runtime.delete）
func (c *Client) DeleteApp(ctx context.Context, hostId int64, req *dto.DeleteAppReq) (*dto.DeleteAppResp, error) {
	var resp dto.DeleteAppResp
	if err := c.requestByHost(ctx, hostId, "app_server.app_runtime.delete", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateServiceTree 创建服务目录（subject: app_runtime.service_tree.create）
func (c *Client) CreateServiceTree(ctx context.Context, hostId int64, req *dto.CreateServiceTreeRuntimeReq) (*dto.CreateServiceTreeRuntimeResp, error) {
	var resp dto.CreateServiceTreeRuntimeResp
	if err := c.requestByHost(ctx, hostId, "app_runtime.service_tree.create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteServiceTree 删除服务目录（subject: app_server.app_runtime.delete_service_tree，删磁盘并从 main.go 移除 import）
func (c *Client) DeleteServiceTree(ctx context.Context, hostId int64, req *dto.DeleteServiceTreeRuntimeReq) (*dto.DeleteServiceTreeRuntimeResp, error) {
	var resp dto.DeleteServiceTreeRuntimeResp
	if err := c.requestByHost(ctx, hostId, "app_server.app_runtime.delete_service_tree", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReadDirectoryFiles 读取目录文件（subject: app_server.app_runtime.read_directory_files）
func (c *Client) ReadDirectoryFiles(ctx context.Context, hostId int64, req *dto.ReadDirectoryFilesRuntimeReq) (*dto.ReadDirectoryFilesRuntimeResp, error) {
	var resp dto.ReadDirectoryFilesRuntimeResp
	if err := c.requestByHost(ctx, hostId, "app_server.app_runtime.read_directory_files", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReplaceInFileBatch 文件批量 search-replace（subject: app_server.app_runtime.replace_in_file_batch）
func (c *Client) ReplaceInFileBatch(ctx context.Context, hostId int64, req *dto.ReplaceInFileBatchReq) (*dto.ReplaceInFileBatchResp, error) {
	var resp dto.ReplaceInFileBatchResp
	if err := c.requestByHost(ctx, hostId, "app_server.app_runtime.replace_in_file_batch", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteFile 删除磁盘文件（subject: app_server.app_runtime.delete_file）
func (c *Client) DeleteFile(ctx context.Context, hostId int64, req *dto.DeleteFileRuntimeReq) (*dto.DeleteFileRuntimeResp, error) {
	var resp dto.DeleteFileRuntimeResp
	if err := c.requestByHost(ctx, hostId, "app_server.app_runtime.delete_file", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchCreateDirectoryTree 批量创建目录树（subject: app_server.app_runtime.batch_create_directory_tree）
func (c *Client) BatchCreateDirectoryTree(ctx context.Context, hostId int64, req *dto.BatchCreateDirectoryTreeRuntimeReq) (*dto.BatchCreateDirectoryTreeRuntimeResp, error) {
	var resp dto.BatchCreateDirectoryTreeRuntimeResp
	if err := c.requestByHost(ctx, hostId, "app_server.app_runtime.batch_create_directory_tree", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchWriteFiles 批量写文件（subject: app_server.app_runtime.batch_write_files）
func (c *Client) BatchWriteFiles(ctx context.Context, hostId int64, req *dto.BatchWriteFilesRuntimeReq) (*dto.BatchWriteFilesRuntimeResp, error) {
	var resp dto.BatchWriteFilesRuntimeResp
	if err := c.requestByHost(ctx, hostId, "app_server.app_runtime.batch_write_files", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RequestApp 请求应用（Publish 到 function_server.app_runtime.{user}.{app}.{version}，通过 waiter 等响应）
func (c *Client) RequestApp(ctx context.Context, natsId int64, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	subject := fmt.Sprintf("function_server.app_runtime.%s.%s.%s", req.User, req.App, req.Version)
	conn, err := c.connProvider.GetNatsByNatsId(natsId)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}
	msg := &nats.Msg{Subject: subject, Data: data, Header: make(nats.Header)}
	msg.Header.Set(contextx.TraceIdHeader, req.TraceId)
	msg.Header.Set(contextx.RequestUserHeader, req.RequestUser)
	if req.RequestUserDept != "" {
		msg.Header.Set(contextx.DepartmentFullPathHeader, req.RequestUserDept)
	}
	msg.Header.Set("method", req.Method)
	msg.Header.Set("router", req.Router)
	msg.Header.Set("app", req.App)
	msg.Header.Set("user", req.User)
	msg.Header.Set("version", req.Version)
	if req.Token != "" {
		msg.Header.Set(contextx.TokenHeader, req.Token)
	}
	if err := conn.PublishMsg(msg); err != nil {
		return nil, fmt.Errorf("publish request failed: %w", err)
	}
	resp, err := c.waiter.Wait(ctx, req.TraceId, c.appRequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("wait response timeout: %w", err)
	}
	return resp, nil
}

func (c *Client) initSubscriptions() {
	for _, hostId := range c.connProvider.HostIds() {
		conn, err := c.connProvider.GetNatsByHost(hostId)
		if err != nil {
			continue
		}
		sub, err := conn.Subscribe("app.function_server.*.*.*", c.handleApp2FunctionServerResponse)
		if err != nil {
			logger.Warnf(context.Background(), "[appcall] Failed to subscribe to response subject on host %d: %v", hostId, err)
			continue
		}
		c.subs = append(c.subs, sub)
	}
}

func (c *Client) handleApp2FunctionServerResponse(msg *nats.Msg) {
	var resp dto.RequestAppResp
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return
	}
	if traceId := msg.Header.Get(contextx.TraceIdHeader); traceId != "" {
		resp.TraceId = traceId
	}
	if !c.waiter.Notify(resp.TraceId, &resp) {
		logger.Warnf(context.Background(), "[appcall] No waiting request found for traceId: %s", resp.TraceId)
	}
}

// Close 取消所有订阅，释放资源（不关闭 conn，由 ConnProvider 持有方负责）。
func (c *Client) Close() error {
	for _, sub := range c.subs {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	c.subs = nil
	return nil
}
