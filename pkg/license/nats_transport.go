package license

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// NATSTransport 收敛 License 跨服务同步链路的 NATS 协议细节。
type NATSTransport struct {
	conn *nats.Conn
}

func NewNATSTransport(conn *nats.Conn) *NATSTransport {
	return &NATSTransport{conn: conn}
}

func (t *NATSTransport) RequestKey(ctx context.Context, timeout time.Duration) (*LicenseKeyMessage, error) {
	if err := t.ensureConnected(); err != nil {
		return nil, err
	}

	req := LicenseKeyRequestMessage{Request: "license_key"}
	var resp LicenseKeyMessage
	if _, err := msgx.RequestJSON(ctx, t.conn, subjects.ControlLicenseKeyGetQuerySubject, req, &resp, timeout); err != nil {
		return nil, fmt.Errorf("request license key: %w", err)
	}
	return &resp, nil
}

func (t *NATSTransport) SubscribeKeyUpdates(handler nats.MsgHandler) (*nats.Subscription, error) {
	if err := t.ensureConnected(); err != nil {
		return nil, err
	}
	sub, err := t.conn.Subscribe(subjects.LicenseKeyUpdatedEventSubject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribe license key updates: %w", err)
	}
	return sub, nil
}

func (t *NATSTransport) SubscribeRefreshInstructions(handler nats.MsgHandler) (*nats.Subscription, error) {
	if err := t.ensureConnected(); err != nil {
		return nil, err
	}
	sub, err := t.conn.Subscribe(subjects.LicenseKeyRefreshEventSubject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribe license refresh instructions: %w", err)
	}
	return sub, nil
}

func (t *NATSTransport) PublishKeyUpdate(msg *LicenseKeyMessage) error {
	if err := t.ensureConnected(); err != nil {
		return err
	}
	return t.publish(subjects.LicenseKeyUpdatedEventSubject, msg)
}

func (t *NATSTransport) PublishInstruction(msg *LicenseInstructionMessage) error {
	if err := t.ensureConnected(); err != nil {
		return err
	}
	return t.publish(subjects.LicenseKeyRefreshEventSubject, msg)
}

func (t *NATSTransport) publish(subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal license NATS payload: %w", err)
	}
	if err := t.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}

func (t *NATSTransport) ensureConnected() error {
	if t == nil || t.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	if !t.conn.IsConnected() {
		return fmt.Errorf("NATS connection is not connected")
	}
	return nil
}
