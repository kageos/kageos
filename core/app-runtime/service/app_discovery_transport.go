package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/discovery"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// AppDiscoveryTransport 管理 app discovery 的 NATS 订阅与广播。
type AppDiscoveryTransport struct {
	conn        *nats.Conn
	broadcaster *AppDiscoveryBroadcaster
	sub         *nats.Subscription
}

func NewAppDiscoveryTransport(conn *nats.Conn) *AppDiscoveryTransport {
	return &AppDiscoveryTransport{
		conn:        conn,
		broadcaster: NewAppDiscoveryBroadcaster(conn),
	}
}

func (t *AppDiscoveryTransport) SubscribeRuntimeLifecycleEvents(handler nats.MsgHandler) error {
	if t == nil || t.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	sub, err := t.conn.Subscribe(subjects.RuntimeLifecycleEventSubjectPattern, handler)
	if err != nil {
		return fmt.Errorf("subscribe runtime lifecycle events: %w", err)
	}
	t.sub = sub
	return nil
}

func (t *AppDiscoveryTransport) PublishDiscoveryRequest(ctx context.Context, msg *discovery.DiscoveryMessage) error {
	if t == nil || t.broadcaster == nil {
		return fmt.Errorf("app discovery broadcaster is nil")
	}
	return t.broadcaster.PublishDiscoveryRequest(ctx, msg)
}

func (t *AppDiscoveryTransport) Close() error {
	if t == nil || t.sub == nil {
		return nil
	}
	err := t.sub.Unsubscribe()
	t.sub = nil
	return err
}
