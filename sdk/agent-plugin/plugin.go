package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/nats-io/nats.go"
)

// Plugin 插件实例
type Plugin struct {
	conn         *nats.Conn
	sub          *nats.Subscription
	handler      HandlerFunc
	subject      string // 订阅的主题
	shutdownOnce sync.Once
}

// HandlerFunc 插件处理函数类型
// ctx: 上下文信息（包含 trace_id, user 等）
// req: 插件执行请求
// 返回: 插件执行响应和错误
type HandlerFunc func(ctx *Context, req *dto.PluginRunReq) (*dto.PluginRunResp, error)

// NewPlugin 创建插件实例
// natsURL: NATS 服务器地址（如 "nats://127.0.0.1:4222"）
// subject: 订阅的主题（如 "agent.function_gen.beiluo.1.run"）
// handler: 处理函数
func NewPlugin(natsURL, subject string, handler HandlerFunc) (*Plugin, error) {
	// 1. 连接 NATS
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("连接 NATS 失败: %w", err)
	}

	plugin := &Plugin{
		conn:    conn,
		handler: handler,
		subject: subject,
	}

	// 2. 订阅主题
	sub, err := conn.Subscribe(subject, plugin.handleMessage)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("订阅主题失败: %w", err)
	}

	plugin.sub = sub

	logger.Infof(context.Background(), "[Plugin] 插件初始化成功, Subject: %s", subject)

	return plugin, nil
}

// Run 运行插件（阻塞，直到收到退出信号）
func (p *Plugin) Run() error {
	logger.Infof(context.Background(), "[Plugin] 插件已启动，等待消息... Subject: %s", p.subject)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Infof(context.Background(), "[Plugin] 收到退出信号，正在关闭...")

	return p.Close()
}

// Close 关闭插件
func (p *Plugin) Close() error {
	var err error
	p.shutdownOnce.Do(func() {
		if p.sub != nil {
			if err = p.sub.Unsubscribe(); err != nil {
				logger.Errorf(context.Background(), "[Plugin] 取消订阅失败: %v", err)
			}
		}
		if p.conn != nil {
			p.conn.Close()
		}
		logger.Infof(context.Background(), "[Plugin] 插件已关闭")
	})
	return err
}

// handleMessage 处理接收到的消息
func (p *Plugin) handleMessage(msg *nats.Msg) {
	ctx := context.Background()

	// 解析请求
	var req dto.PluginRunReq
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logger.Errorf(ctx, "[Plugin] 解析请求失败: %v", err)
		msgx.RespFailMsg(msg, fmt.Errorf("解析请求失败: %w", err))
		return
	}

	// 创建上下文
	pluginCtx := &Context{
		TraceID:     msg.Header.Get("X-Trace-Id"),
		RequestUser: msg.Header.Get("X-Request-User"),
	}

	logger.Infof(ctx, "[Plugin] 收到请求, TraceID: %s, User: %s, Message: %s, Files: %d",
		pluginCtx.TraceID, pluginCtx.RequestUser, req.Message, len(req.Files))

	// 调用处理函数
	resp, err := p.handler(pluginCtx, &req)
	if err != nil {
		logger.Errorf(ctx, "[Plugin] 处理失败: %v, TraceID: %s", err, pluginCtx.TraceID)
		msgx.RespFailMsg(msg, err)
		return
	}

	// 返回成功响应
	if err := msgx.RespSuccessMsg(msg, resp); err != nil {
		logger.Errorf(ctx, "[Plugin] 发送响应失败: %v, TraceID: %s", err, pluginCtx.TraceID)
		return
	}

	logger.Infof(ctx, "[Plugin] 处理成功, TraceID: %s, DataLength: %d", pluginCtx.TraceID, len(resp.Data))
}

