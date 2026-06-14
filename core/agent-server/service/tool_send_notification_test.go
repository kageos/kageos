package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type fakeNotificationPublisher struct {
	msgs []*nats.Msg
}

func (p *fakeNotificationPublisher) PublishMsg(msg *nats.Msg) error {
	p.msgs = append(p.msgs, msg)
	return nil
}

func TestSendNotificationPublishesWithScheduledSource(t *testing.T) {
	publisher := &fakeNotificationPublisher{}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		RequestUser:  "alice",
		ClientSource: contextx.ClientSourceScheduledTask,
		SourceType:   contextx.SourceTypeScheduledTask,
		SourceRef:    "timer_task:7:execution:9",
		SourcePath:   "/system/test22/hot_news",
		SourceTitle:  "热点情报定时推送",
	})
	ctx = contextx.WithWorkspaceSession(ctx, "session-1", "热点情报定时推送", "app_operator")

	result := runSendNotificationTool(ctx, publisher, sendNotificationArgs{
		ToUsers:     "bob，carol,bob",
		Title:       "发现重要情报",
		Message:     "<b>需要关注</b>",
		ContentType: "html",
		Level:       "critical",
	}, "/system/test22/hot_news")

	if result.IsError {
		t.Fatalf("send_notification returned error: %s", result.Content)
	}
	if len(publisher.msgs) != 1 {
		t.Fatalf("published messages = %d, want 1", len(publisher.msgs))
	}
	msg := publisher.msgs[0]
	if msg.Subject != subjects.MessageSendCommandSubject {
		t.Fatalf("subject = %q", msg.Subject)
	}
	envelope, err := decodeNotifyEnvelope(msg.Data)
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Message.ToUsers != "bob,carol" {
		t.Fatalf("to_users = %q", envelope.Message.ToUsers)
	}
	if envelope.Message.ContentType != "html" {
		t.Fatalf("content_type = %q", envelope.Message.ContentType)
	}
	if !strings.HasPrefix(envelope.Message.Title, "【高优先级】") {
		t.Fatalf("title should be prefixed for critical notifications, got %q", envelope.Message.Title)
	}
	if envelope.Meta.SourceType != contextx.SourceTypeScheduledTask {
		t.Fatalf("source_type = %q", envelope.Meta.SourceType)
	}
	if envelope.Meta.SourceRef != "timer_task:7:execution:9" {
		t.Fatalf("source_ref = %q", envelope.Meta.SourceRef)
	}
	if envelope.Meta.SourcePath != "/system/test22/hot_news" {
		t.Fatalf("source_path = %q", envelope.Meta.SourcePath)
	}
	if envelope.Meta.ThreadKey != "scheduled_task:timer_task:7" {
		t.Fatalf("thread_key = %q", envelope.Meta.ThreadKey)
	}
	if envelope.Meta.WorkspaceSessionID != "session-1" {
		t.Fatalf("workspace_session_id = %q", envelope.Meta.WorkspaceSessionID)
	}
}

func TestSendNotificationDefaultsRecipientToRequestUser(t *testing.T) {
	publisher := &fakeNotificationPublisher{}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		RequestUser:  "alice",
		ClientSource: contextx.ClientSourceAgent,
		SourceType:   contextx.SourceTypeAgentTool,
		SourceRef:    "session-1",
	})
	ctx = contextx.WithWorkspaceSession(ctx, "session-1", "情报巡检", "app_operator")

	result := runSendNotificationTool(ctx, publisher, sendNotificationArgs{
		Title:   "发现重要情报",
		Message: "需要关注",
	}, "/system/test22/hot_news")

	if result.IsError {
		t.Fatalf("send_notification returned error: %s", result.Content)
	}
	if len(publisher.msgs) != 1 {
		t.Fatalf("published messages = %d, want 1", len(publisher.msgs))
	}
	envelope, err := decodeNotifyEnvelope(publisher.msgs[0].Data)
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Message.ToUsers != "alice" {
		t.Fatalf("to_users = %q, want alice", envelope.Message.ToUsers)
	}
}

func TestSendNotificationSchemaDoesNotRequireToUsers(t *testing.T) {
	def := (&SendNotificationTool{}).Definition()
	required, ok := def.InputSchema["required"].([]interface{})
	if !ok {
		t.Fatalf("input schema required missing: %#v", def.InputSchema)
	}
	if containsInterfaceString(required, "to_users") {
		t.Fatalf("send_notification should not require to_users, required=%#v", required)
	}
	properties, ok := def.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("input schema properties missing: %#v", def.InputSchema)
	}
	toUsers, ok := properties["to_users"].(map[string]interface{})
	if !ok {
		t.Fatalf("to_users schema missing: %#v", properties)
	}
	desc, _ := toUsers["description"].(string)
	for _, want := range []string{"普通工作台会话", "定时会话或后台任务必须显式填写", "任务创建人/请求用户"} {
		if !strings.Contains(desc, want) {
			t.Fatalf("to_users description should contain %q, got %q", want, desc)
		}
	}
}

func TestSendNotificationRequiresRecipientWhenRequestUserIsSystem(t *testing.T) {
	publisher := &fakeNotificationPublisher{}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		RequestUser: "system",
	})

	result := runSendNotificationTool(ctx, publisher, sendNotificationArgs{
		Title:   "提醒",
		Message: "需要通知",
	}, "/system/test22/hot_news")

	if !result.IsError {
		t.Fatalf("send_notification should fail without explicit recipient under system context")
	}
	if len(publisher.msgs) != 0 {
		t.Fatalf("published messages = %d, want 0", len(publisher.msgs))
	}
	if !strings.Contains(result.Content, "to_users") {
		t.Fatalf("error should mention to_users, got %q", result.Content)
	}
}
