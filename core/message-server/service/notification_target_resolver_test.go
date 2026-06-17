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

func TestUserNotificationTargetResolverResolvesEnabledWebhookTargets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := msgmodel.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := msgrepo.NewMessageRepository(db)
	vault, err := NewNotificationSecretVault("test-secret")
	if err != nil {
		t.Fatalf("new vault: %v", err)
	}
	webhookCipher, err := vault.Seal("https://open.feishu.cn/open-apis/bot/v2/hook/test")
	if err != nil {
		t.Fatalf("seal webhook: %v", err)
	}
	secretCipher, err := vault.Seal("sign-secret")
	if err != nil {
		t.Fatalf("seal secret: %v", err)
	}
	if _, err := repo.UpsertNotificationChannel(context.Background(), &msgmodel.NotificationChannelSetting{
		OwnerUsername:    "bob",
		Channel:          NotificationChannelFeishu,
		Enabled:          true,
		DeliveryType:     "webhook",
		WebhookURLCipher: webhookCipher,
		SecretCipher:     secretCipher,
		Metadata:         `{"scope":"personal"}`,
	}); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}
	if _, err := repo.UpsertNotificationChannel(context.Background(), &msgmodel.NotificationChannelSetting{
		OwnerUsername:    "alice",
		Channel:          NotificationChannelFeishu,
		Enabled:          false,
		DeliveryType:     "webhook",
		WebhookURLCipher: webhookCipher,
	}); err != nil {
		t.Fatalf("upsert disabled channel: %v", err)
	}

	resolver := NewUserNotificationTargetResolver(repo, vault)
	targets, err := resolver.ResolveNotificationTargets(context.Background(), []ResolvedRecipient{
		{Username: "alice"},
		{Username: "bob"},
	}, &msgmodel.MessageEntry{}, dto.MessageSendPayload{})
	if err != nil {
		t.Fatalf("resolve targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets len = %d, want 1: %#v", len(targets), targets)
	}
	target := targets[0]
	if target.Recipient.Username != "bob" || target.Channel != NotificationChannelFeishu {
		t.Fatalf("unexpected target = %#v", target)
	}
	if target.WebhookURL != "https://open.feishu.cn/open-apis/bot/v2/hook/test" || target.Secret != "sign-secret" {
		t.Fatalf("unexpected webhook/secret: %#v", target)
	}
	if target.Metadata["scope"] != "personal" {
		t.Fatalf("metadata = %#v", target.Metadata)
	}
}
