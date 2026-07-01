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
		Files:       "kageos/a.pdf，kageos/a.pdf;kageos/b.xlsx",
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
	if envelope.Message.Files != "kageos/a.pdf,kageos/b.xlsx" {
		t.Fatalf("files = %q", envelope.Message.Files)
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

func TestSendNotificationAllowsFilesOnly(t *testing.T) {
	publisher := &fakeNotificationPublisher{}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		RequestUser: "alice",
	})

	result := runSendNotificationTool(ctx, publisher, sendNotificationArgs{
		Title: "报告已生成",
		Files: "kageos/reports/a.pdf，kageos/reports/a.pdf;kageos/reports/b.xlsx",
	}, "/system/test22/reports")

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
	if envelope.Message.Content != "" {
		t.Fatalf("content = %q, want empty", envelope.Message.Content)
	}
	if envelope.Message.Files != "kageos/reports/a.pdf,kageos/reports/b.xlsx" {
		t.Fatalf("files = %q", envelope.Message.Files)
	}
}

func TestSendNotificationPublishesWorkspaceRouteSourcePath(t *testing.T) {
	publisher := &fakeNotificationPublisher{}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		RequestUser:  "alice",
		ClientSource: contextx.ClientSourceAgent,
		SourceType:   contextx.SourceTypeAgentTool,
		SourceRef:    "session-1",
	})
	ctx = contextx.WithWorkspaceSession(ctx, "session-1", "订单处理", "app_operator")

	result := runSendNotificationTool(ctx, publisher, sendNotificationArgs{
		Title:   "订单处理完成",
		Message: "已经处理完成",
	}, "/alice/sales/orders")

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
	if envelope.Meta.FullCodePath != "/alice/sales/orders" {
		t.Fatalf("full_code_path = %q", envelope.Meta.FullCodePath)
	}
	if envelope.Meta.SourcePath != "/alice/sales/orders" {
		t.Fatalf("source_path = %q", envelope.Meta.SourcePath)
	}
}

func TestSendNotificationUsesContextSourcePathWhenToolFullCodePathMissing(t *testing.T) {
	publisher := &fakeNotificationPublisher{}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		RequestUser:  "alice",
		ClientSource: contextx.ClientSourceAgent,
		SourceType:   contextx.SourceTypeAgentTool,
		SourceRef:    "session-1",
		SourcePath:   "/alice/sales/orders",
	})
	ctx = contextx.WithWorkspaceSession(ctx, "session-1", "订单处理", "app_operator")

	result := runSendNotificationTool(ctx, publisher, sendNotificationArgs{
		Title:   "订单处理完成",
		Message: "已经处理完成",
	}, "")

	if result.IsError {
		t.Fatalf("send_notification returned error: %s", result.Content)
	}
	envelope, err := decodeNotifyEnvelope(publisher.msgs[0].Data)
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Meta.FullCodePath != "/alice/sales/orders" {
		t.Fatalf("full_code_path = %q", envelope.Meta.FullCodePath)
	}
	if envelope.Meta.SourcePath != "/alice/sales/orders" {
		t.Fatalf("source_path = %q", envelope.Meta.SourcePath)
	}
}

func TestWorkspaceToolSourceDisplayInjectsActiveFullCodePath(t *testing.T) {
	ctx := contextx.WithSourceInfo(context.Background(), contextx.SourceTypeAgentTool, "session-1")
	ctx = withWorkspaceToolSourceDisplay(ctx, "/alice/sales/orders")

	if got := contextx.GetSourcePath(ctx); got != "/alice/sales/orders" {
		t.Fatalf("source_path = %q", got)
	}

	ctx = withWorkspaceToolSourceDisplay(ctx, "/alice/sales/customers")
	if got := contextx.GetSourcePath(ctx); got != "/alice/sales/orders" {
		t.Fatalf("existing source_path should be preserved, got %q", got)
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
	if containsInterfaceString(required, "message") {
		t.Fatalf("send_notification should allow files-only notifications, required=%#v", required)
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
	for _, want := range []string{"请求用户", "会话创建人", "没有默认用户时才必须显式填写"} {
		if !strings.Contains(desc, want) {
			t.Fatalf("to_users description should contain %q, got %q", want, desc)
		}
	}
}

func TestSendNotificationDefaultsRecipientWhenRequestUserIsSystem(t *testing.T) {
	publisher := &fakeNotificationPublisher{}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		RequestUser: "system",
	})

	result := runSendNotificationTool(ctx, publisher, sendNotificationArgs{
		Title:   "提醒",
		Message: "需要通知",
	}, "/system/test22/hot_news")

	if result.IsError {
		t.Fatalf("send_notification should accept system as a recipient: %s", result.Content)
	}
	if len(publisher.msgs) != 1 {
		t.Fatalf("published messages = %d, want 1", len(publisher.msgs))
	}
	envelope, err := decodeNotifyEnvelope(publisher.msgs[0].Data)
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Message.ToUsers != "system" {
		t.Fatalf("to_users = %q, want system", envelope.Message.ToUsers)
	}
}

func TestSendNotificationRequiresRecipientWhenRequestUserIsEmpty(t *testing.T) {
	publisher := &fakeNotificationPublisher{}

	result := runSendNotificationTool(context.Background(), publisher, sendNotificationArgs{
		Title:   "提醒",
		Message: "需要通知",
	}, "/system/test22/hot_news")

	if !result.IsError {
		t.Fatalf("send_notification should fail without explicit recipient under empty context")
	}
	if len(publisher.msgs) != 0 {
		t.Fatalf("published messages = %d, want 0", len(publisher.msgs))
	}
	for _, want := range []string{"to_users", "context_user=<empty>", "client_source", "workspace_session_id"} {
		if !strings.Contains(result.Content, want) {
			t.Fatalf("error should contain context hint %q, got %q", want, result.Content)
		}
	}
}
