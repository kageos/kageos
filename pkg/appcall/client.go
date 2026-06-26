// Package appcall 提供调用 app / app-runtime 的 SDK 风格客户端（NATS request-reply + RequestApp）。
// 与 pkg/apicall（HTTP 调 API）对称：apicall 调 API，appcall 调 app。
package appcall

import (
	"context"
	"time"

	"github.com/kageos/kageos/dto"
	waiterpkg "github.com/kageos/kageos/pkg/waiter"
	"github.com/nats-io/nats.go"
)

// ConnProvider 按 hostID / natsID 提供 NATS 连接；HostIDs 用于订阅时遍历所有连接。
type ConnProvider interface {
	GetConnByHost(hostID int64) (*nats.Conn, error)
	GetConnByNATSID(natsID int64) (*nats.Conn, error)
	HostIDs() []int64
}

// Waiter RequestApp 发 Publish 后 Wait 等响应，响应回调里 Notify；需同一实例。
type Waiter interface {
	Register(key string) waiterpkg.Registration
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

// Client app-server 调用 app-runtime 的 SDK 风格客户端。
// 内部按两条链拆分：
// 1. runtimeRequester：普通 request-reply 管理类调用
// 2. appInvokeTransport：RequestApp 的 publish + reply + waiter 链路
type Client struct {
	runtime *runtimeRequester
	invoke  *appInvokeTransport
}

// New 创建 Client，并初始化响应主题订阅。
func New(opts Options) *Client {
	runtime := newRuntimeRequester(opts.ConnProvider, opts.NatsRequestTimeout)
	invoke := newAppInvokeTransport(opts.ConnProvider, opts.Waiter, opts.AppRequestTimeout)
	invoke.initSubscriptions()

	return &Client{
		runtime: runtime,
		invoke:  invoke,
	}
}

// Close 取消所有订阅，释放资源（不关闭 conn，由 ConnProvider 持有方负责）。
func (c *Client) Close() error {
	if c.invoke == nil {
		return nil
	}
	return c.invoke.close()
}
