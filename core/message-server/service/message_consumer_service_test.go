package service

import (
	"context"
	"testing"

	msgmodel "github.com/kageos/kageos/core/message-server/model"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConsumeCreatesInboxRecipients(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := msgmodel.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := msgrepo.NewMessageRepository(db)
	svc := NewMessageConsumerService(repo)

	err = svc.Consume(context.Background(), &dto.MessageSendEnvelope{
		Meta: dto.MessageSendMeta{From: "alice"},
		Message: dto.MessageSendPayload{
			ToUsers: "bob,carol",
			Title:   "任务完成",
			Content: "已经处理",
		},
	})
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	for _, username := range []string{"bob", "carol"} {
		count, err := repo.CountUnread(context.Background(), username)
		if err != nil {
			t.Fatalf("count unread %s: %v", username, err)
		}
		if count != 1 {
			t.Fatalf("unread count for %s = %d, want 1", username, count)
		}
	}
}

func TestConsumeValidatesPayload(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := msgmodel.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewMessageConsumerService(msgrepo.NewMessageRepository(db))
	err = svc.Consume(context.Background(), &dto.MessageSendEnvelope{
		Message: dto.MessageSendPayload{Content: "missing recipients"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestConsumeCreatesRecipientWithoutUserLookup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := msgmodel.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := msgrepo.NewMessageRepository(db)
	svc := NewMessageConsumerService(repo)
	err = svc.Consume(context.Background(), &dto.MessageSendEnvelope{
		Meta: dto.MessageSendMeta{From: "alice"},
		Message: dto.MessageSendPayload{
			ToUsers: "missing-user",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("consume missing-user as recipient key: %v", err)
	}
	count, err := repo.CountUnread(context.Background(), "missing-user")
	if err != nil {
		t.Fatalf("count unread missing-user: %v", err)
	}
	if count != 1 {
		t.Fatalf("unread count for missing-user = %d, want 1", count)
	}
}

func TestConsumeAllowsSystemRecipient(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := msgmodel.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := msgrepo.NewMessageRepository(db)
	svc := NewMessageConsumerService(repo)
	err = svc.Consume(context.Background(), &dto.MessageSendEnvelope{
		Meta: dto.MessageSendMeta{From: "scheduler"},
		Message: dto.MessageSendPayload{
			ToUsers: "system",
			Title:   "会议即将开始提醒",
			Content: "hello",
		},
	})
	if err != nil {
		t.Fatalf("consume system recipient: %v", err)
	}

	count, err := repo.CountUnread(context.Background(), "system")
	if err != nil {
		t.Fatalf("count unread system: %v", err)
	}
	if count != 1 {
		t.Fatalf("unread count for system = %d, want 1", count)
	}
}
