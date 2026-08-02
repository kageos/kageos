package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type recordingPermissionNotifier struct {
	notifications []PermissionNotification
	err           error
}

func (n *recordingPermissionNotifier) Notify(_ context.Context, notification PermissionNotification) error {
	n.notifications = append(n.notifications, notification)
	return n.err
}

type recordingPermissionMessagePublisher struct {
	msg *nats.Msg
	err error
}

func (p *recordingPermissionMessagePublisher) PublishMsg(msg *nats.Msg) error {
	p.msg = msg
	return p.err
}

func TestNATSPermissionNotifierPublishesPlatformMessage(t *testing.T) {
	publisher := &recordingPermissionMessagePublisher{}
	notifier := NewNATSPermissionNotifier(publisher)
	err := notifier.Notify(actorContext("alice"), PermissionNotification{
		ToUser:       "Bob",
		Actor:        "Alice",
		TenantUser:   "alice",
		App:          "ops",
		ResourcePath: "/alice/ops/tickets",
		Title:        "权限申请已通过",
		Message:      "你现在可以使用该目录。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if publisher.msg == nil || publisher.msg.Subject != subjects.MessageSendCommandSubject {
		t.Fatalf("published message = %#v", publisher.msg)
	}

	var envelope dto.MessageSendEnvelope
	if err := json.Unmarshal(publisher.msg.Data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Message.ToUsers != "bob" || envelope.Message.ContentType != "markdown" {
		t.Fatalf("message payload = %#v", envelope.Message)
	}
	if envelope.Meta.From != "alice" || envelope.Meta.SourceType != "permission" {
		t.Fatalf("message meta = %#v", envelope.Meta)
	}
	if envelope.Meta.FullCodePath != "/alice/ops/tickets" || envelope.Meta.SourcePath != "/alice/ops/tickets" {
		t.Fatalf("message source = %#v", envelope.Meta)
	}
	if envelope.Meta.ThreadKey != "permission:alice:ops:bob" {
		t.Fatalf("thread_key = %q", envelope.Meta.ThreadKey)
	}
}

func TestPermissionNotificationFailureDoesNotFailGrant(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	notifier := &recordingPermissionNotifier{err: errors.New("message service unavailable")}
	service.permissionNotifier = notifier

	err := service.GrantRole(actorContext("alice"), accessGrantForNotificationTest("bob"))
	if err != nil {
		t.Fatalf("grant should succeed when notification fails: %v", err)
	}
	if len(notifier.notifications) != 1 {
		t.Fatalf("notifications = %#v", notifier.notifications)
	}
	var count int64
	if err := db.Table("workspace_role_assignments").Where("principal_key = ?", "bob").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("assignment count = %d", count)
	}
}

func TestPermissionGrantNotificationSkipsOrganizationPrincipals(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	notifier := &recordingPermissionNotifier{}
	service.permissionNotifier = notifier
	service.departmentLookup = func(context.Context, string) (bool, error) { return true, nil }

	req := accessGrantForNotificationTest("bob")
	req.Principal = departmentPrincipal("/org")
	if err := service.GrantRole(actorContext("alice"), req); err != nil {
		t.Fatal(err)
	}
	if len(notifier.notifications) != 0 {
		t.Fatalf("organization grant should not fan out notifications: %#v", notifier.notifications)
	}
}

func accessGrantForNotificationTest(username string) access.GrantRoleRequest {
	return access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal(username),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}
}

func requireNotificationContains(t *testing.T, notification PermissionNotification, values ...string) {
	t.Helper()
	content := notification.Title + "\n" + notification.Message
	for _, value := range values {
		if !strings.Contains(content, value) {
			t.Fatalf("notification %q does not contain %q", content, value)
		}
	}
}
