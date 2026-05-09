package repository

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/message-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListInboxScansMessageInboxDTO(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitModels(db); err != nil {
		t.Fatalf("migrate message models: %v", err)
	}

	repo := NewMessageRepository(db)
	_, err = repo.Create(context.Background(), dto.MessageSendMeta{
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
	if list[0].SourceDisplay != nil {
		t.Fatalf("source_display = %#v, want nil", list[0].SourceDisplay)
	}
}

func TestGetInboxMessageDoesNotRequireSourceTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitModels(db); err != nil {
		t.Fatalf("migrate message models: %v", err)
	}

	repo := NewMessageRepository(db)
	entry, err := repo.Create(context.Background(), dto.MessageSendMeta{
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

	item, err := repo.GetInboxMessage(context.Background(), "bob", entry.ID)
	if err != nil {
		t.Fatalf("get inbox message: %v", err)
	}
	if item.SourceDisplay != nil {
		t.Fatalf("source_display = %#v, want nil", item.SourceDisplay)
	}
}
