package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/discovery"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/msgx"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/kageos/kageos/sdk/agent-app/env"
	"github.com/nats-io/nats.go"
)

// AppTransport 收敛 App 主动发出的 NATS 消息，避免业务主体直接操作底层连接。
type AppTransport struct {
	conn     *nats.Conn
	subjects *Subjects
}

func NewAppTransport(conn *nats.Conn, subjects *Subjects) *AppTransport {
	return &AppTransport{
		conn:     conn,
		subjects: subjects,
	}
}

func (t *AppTransport) PublishInvokeResponse(resp *dto.RequestAppResp, success bool) error {
	if t.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal invoke response: %w", err)
	}

	msg := &nats.Msg{
		Subject: t.subjects.InvokeReply,
		Data:    data,
		Header:  make(nats.Header),
	}
	if resp.TraceId != "" {
		msg.Header.Set(contextx.TraceIdHeader, resp.TraceId)
	}

	if success {
		msg.Header.Set("code", "0")
	} else {
		msg.Header.Set("code", "-1")
		msg.Header.Set("msg", resp.Error)
	}

	if err := t.conn.PublishMsg(msg); err != nil {
		return fmt.Errorf("publish invoke response to %s: %w", t.subjects.InvokeReply, err)
	}
	return nil
}

func (t *AppTransport) PublishMessageCommand(ctx context.Context, envelope *dto.MessageSendEnvelope) error {
	if t.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	if envelope == nil {
		return fmt.Errorf("message envelope is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	msg, err := msgx.BuildJSONRequest(ctx, subjects.MessageSendCommandSubject, envelope)
	if err != nil {
		return fmt.Errorf("build message command: %w", err)
	}
	if err := t.conn.PublishMsg(msg); err != nil {
		return fmt.Errorf("publish message command to %s: %w", subjects.MessageSendCommandSubject, err)
	}
	logger.Infof(ctx, "[PublishMessage] published async to %s, from=%s full_code_path=%s to_users=%s title=%s",
		subjects.MessageSendCommandSubject, envelope.Meta.From, envelope.Meta.FullCodePath, envelope.Message.ToUsers, envelope.Message.Title)
	return nil
}

func (t *AppTransport) PublishDiscoveryResponse(runtimeID string, startTime time.Time) error {
	responseData := map[string]interface{}{
		"type":       "response",
		"status":     "running",
		"runtime_id": runtimeID,
		"start_time": startTime,
		"timestamp":  time.Now(),
	}

	return t.publishLifecycleEvent(subjects.MessageTypeStatusDiscovery, responseData)
}

func (t *AppTransport) PublishStartup(startTime time.Time) error {
	return t.publishLifecycleEvent(subjects.MessageTypeStatusStartup, map[string]interface{}{
		"status":     "running",
		"start_time": startTime,
	})
}

func (t *AppTransport) PublishStartupFailure(startTime time.Time, message string) error {
	return t.publishLifecycleEvent(subjects.MessageTypeStatusStartup, map[string]interface{}{
		"status":        "failed",
		"start_time":    startTime,
		"error_message": message,
	})
}

func (t *AppTransport) PublishClose(startTime, closeTime time.Time) error {
	return t.publishLifecycleEvent(subjects.MessageTypeStatusClose, map[string]interface{}{
		"status":     "closed",
		"start_time": startTime,
		"close_time": closeTime,
	})
}

func (t *AppTransport) RespondUpdateSuccess(msg *nats.Msg, data *DiffData) error {
	rsp := subjects.Message{
		Type:      subjects.MessageTypeStatusOnAppUpdate,
		Data:      data,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Timestamp: time.Now(),
	}

	return msgx.RespondJSONSuccess(msg, rsp)
}

func (t *AppTransport) RespondUpdateError(msg *nats.Msg, message string) error {
	rsp := subjects.Message{
		ErrorMsg:  message,
		Type:      subjects.MessageTypeStatusOnAppUpdate,
		Data:      nil,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Timestamp: time.Now(),
	}

	responseData, err := json.Marshal(rsp)
	if err != nil {
		return fmt.Errorf("marshal update error response: %w", err)
	}

	responseMsg := nats.NewMsg(msg.Subject)
	responseMsg.Header = msg.Header
	responseMsg.Data = responseData
	if err := msg.RespondMsg(responseMsg); err != nil {
		return fmt.Errorf("respond update error message: %w", err)
	}
	return nil
}

func (t *AppTransport) publishLifecycleEvent(messageType string, data map[string]interface{}) error {
	if t.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	message := subjects.Message{
		Type:      messageType,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Data:      data,
		Timestamp: time.Now(),
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal lifecycle message %s: %w", messageType, err)
	}

	if err := t.conn.Publish(t.subjects.LifecycleEvent, messageData); err != nil {
		return fmt.Errorf("publish lifecycle event %s to %s: %w", messageType, t.subjects.LifecycleEvent, err)
	}

	logger.Infof(context.Background(), "Lifecycle event %s sent to subject: %s", messageType, t.subjects.LifecycleEvent)
	return nil
}

func (t *AppTransport) ParseDiscoveryRequest(msg *nats.Msg) (*discovery.DiscoveryMessage, error) {
	var discoveryMsg discovery.DiscoveryMessage
	if err := json.Unmarshal(msg.Data, &discoveryMsg); err != nil {
		return nil, fmt.Errorf("unmarshal discovery request: %w", err)
	}
	return &discoveryMsg, nil
}
