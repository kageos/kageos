package repository

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListInboxScansMessageInboxDTO(t *testing.T) {
	repo := newTestMessageRepo(t)

	_, err := repo.Create(context.Background(), dto.MessageSendMeta{
		From:         "alice",
		FullCodePath: "/alice/hr/leave.form",
		SourceType:   "function",
	}, dto.MessageSendPayload{
		Title:   "请假审批",
		Content: "请审批",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	list, total, err := repo.ListInbox(context.Background(), "bob", "", 0, 20)
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("got total=%d len=%d, want 1", total, len(list))
	}
	if got := list[0].FullCodePath; got != "/alice/hr/leave.form" {
		t.Fatalf("full_code_path = %q, want /alice/hr/leave.form", got)
	}
}

func TestMarkReadAndUnreadCount(t *testing.T) {
	repo := newTestMessageRepo(t)
	entry, err := repo.Create(context.Background(), dto.MessageSendMeta{From: "alice"}, dto.MessageSendPayload{
		Title:   "hello",
		Content: "world",
	}, []string{"bob", "bob"})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	count, err := repo.CountUnread(context.Background(), "bob")
	if err != nil {
		t.Fatalf("count unread: %v", err)
	}
	if count != 1 {
		t.Fatalf("unread count = %d, want 1", count)
	}
	if err := repo.MarkRead(context.Background(), "bob", entry.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	count, err = repo.CountUnread(context.Background(), "bob")
	if err != nil {
		t.Fatalf("count unread after mark: %v", err)
	}
	if count != 0 {
		t.Fatalf("unread count after mark = %d, want 0", count)
	}
}

func newTestMessageRepo(t *testing.T) *MessageRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitModels(db); err != nil {
		t.Fatalf("migrate message models: %v", err)
	}
	return NewMessageRepository(db)
}
