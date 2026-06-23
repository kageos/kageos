package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/discovery"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// AppDiscoveryBroadcaster 负责向 app 广播 discovery 请求。
type AppDiscoveryBroadcaster struct {
	conn *nats.Conn
}

// NewAppDiscoveryBroadcaster 创建 AppDiscoveryBroadcaster。
func NewAppDiscoveryBroadcaster(conn *nats.Conn) *AppDiscoveryBroadcaster {
	return &AppDiscoveryBroadcaster{conn: conn}
}

// PublishDiscoveryRequest 广播 discovery 请求。
func (b *AppDiscoveryBroadcaster) PublishDiscoveryRequest(ctx context.Context, msg *discovery.DiscoveryMessage) error {
	if b == nil || b.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discovery message: %w", err)
	}

	if err := b.conn.Publish(subjects.AppDiscoveryRequestSubject, data); err != nil {
		return fmt.Errorf("publish discovery request: %w", err)
	}

	logger.Infof(ctx, "[AppDiscoveryBroadcaster] Published discovery request: runtime_id=%s", msg.RuntimeID)
	return nil
}

// AppControlClient 负责 runtime -> app 的控制类 publish / request-reply。
type AppControlClient struct {
	conn *nats.Conn
}

// NewAppControlClient 创建 AppControlClient。
func NewAppControlClient(conn *nats.Conn) *AppControlClient {
	return &AppControlClient{conn: conn}
}

// RequestUpdateCallback 请求 app 执行 onAppUpdate 并等待响应。
func (c *AppControlClient) RequestUpdateCallback(ctx context.Context, user, app, version string, request *subjects.Message, timeout time.Duration) (*subjects.Message, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}

	subject := subjects.BuildAppControlSubject(user, app, version)
	logger.Infof(ctx, "[AppControlClient] Sending update callback request to subject: %s", subject)
	logger.Debugf(ctx, "[AppControlClient] Request data: %+v", request)

	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	msg := nats.NewMsg(subject)
	msg.Data = requestData

	responseMsg, err := c.conn.RequestMsg(msg, timeout)
	if err != nil {
		return nil, fmt.Errorf("update callback request failed: %w", err)
	}

	var rsp subjects.Message
	if err := json.Unmarshal(responseMsg.Data, &rsp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if rsp.ErrorMsg != "" {
		return &rsp, fmt.Errorf("SDK onAppUpdate error: %s", rsp.ErrorMsg)
	}

	return &rsp, nil
}

// PublishShutdown 发布 shutdown 控制命令。
func (c *AppControlClient) PublishShutdown(ctx context.Context, user, app, version string, message *subjects.Message) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal shutdown command: %w", err)
	}

	subject := subjects.BuildAppControlSubject(user, app, version)
	if err := c.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish shutdown command to %s: %w", subject, err)
	}

	logger.Infof(ctx, "[AppControlClient] Published shutdown command to %s", subject)
	return nil
}
