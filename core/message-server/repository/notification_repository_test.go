package repository

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/message-server/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationChannelRepositoryUpsertListAndResolve(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewMessageRepository(db)

	row, err := repo.UpsertNotificationChannel(context.Background(), &model.NotificationChannelSetting{
		OwnerUsername:    "alice",
		Channel:          "feishu",
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "飞书群",
		WebhookURLCipher: "cipher-url",
		SecretCipher:     "cipher-secret",
		Metadata:         `{"tenant":"demo"}`,
	})
	if err != nil {
		t.Fatalf("upsert notification channel: %v", err)
	}
	if row.ID == 0 {
		t.Fatal("expected persisted row id")
	}

	list, err := repo.ListNotificationChannels(context.Background(), "alice")
	if err != nil {
		t.Fatalf("list notification channels: %v", err)
	}
	if len(list) != 1 || list[0].Channel != "feishu" || list[0].DisplayName != "飞书群" {
		t.Fatalf("unexpected list: %#v", list)
	}

	enabled, err := repo.ListEnabledNotificationChannels(context.Background(), []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("list enabled notification channels: %v", err)
	}
	if len(enabled) != 1 || enabled[0].OwnerUsername != "alice" {
		t.Fatalf("unexpected enabled list: %#v", enabled)
	}

	_, err = repo.UpsertNotificationChannel(context.Background(), &model.NotificationChannelSetting{
		OwnerUsername:    "alice",
		Channel:          "feishu",
		Enabled:          false,
		DeliveryType:     "webhook",
		WebhookURLCipher: "new-cipher-url",
	})
	if err != nil {
		t.Fatalf("update notification channel: %v", err)
	}
	got, err := repo.GetNotificationChannel(context.Background(), "alice", "feishu")
	if err != nil {
		t.Fatalf("get notification channel: %v", err)
	}
	if got.Enabled || got.WebhookURLCipher != "new-cipher-url" {
		t.Fatalf("updated row = %#v", got)
	}

	if err := repo.DeleteNotificationChannel(context.Background(), "alice", "feishu"); err != nil {
		t.Fatalf("delete notification channel: %v", err)
	}
	list, err = repo.ListNotificationChannels(context.Background(), "alice")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list after delete, got %#v", list)
	}
}

func TestNotificationChannelRepositoryRecordsDeliveryStatus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewMessageRepository(db)
	if _, err := repo.UpsertNotificationChannel(context.Background(), &model.NotificationChannelSetting{
		OwnerUsername:    "alice",
		Channel:          "wecom",
		Enabled:          true,
		DeliveryType:     "webhook",
		WebhookURLCipher: "cipher-url",
	}); err != nil {
		t.Fatalf("create notification channel: %v", err)
	}

	if err := repo.RecordNotificationChannelDeliveryFailure(context.Background(), "alice", "wecom", "webhook returned 400", true); err != nil {
		t.Fatalf("record failure: %v", err)
	}
	got, err := repo.GetNotificationChannel(context.Background(), "alice", "wecom")
	if err != nil {
		t.Fatalf("get after failure: %v", err)
	}
	if got.FailCount != 1 || got.LastFailedAt == nil || got.LastTestAt == nil || got.LastError == "" {
		t.Fatalf("unexpected failure status: %#v", got)
	}

	if err := repo.RecordNotificationChannelDeliverySuccess(context.Background(), "alice", "wecom", false); err != nil {
		t.Fatalf("record success: %v", err)
	}
	got, err = repo.GetNotificationChannel(context.Background(), "alice", "wecom")
	if err != nil {
		t.Fatalf("get after success: %v", err)
	}
	if got.FailCount != 0 || got.LastSuccessAt == nil || got.LastError != "" {
		t.Fatalf("unexpected success status: %#v", got)
	}
}
