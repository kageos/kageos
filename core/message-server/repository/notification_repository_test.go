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

func TestNotificationRouteRepositoryMatchesNearestScopeAndRecordsStatus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewMessageRepository(db)
	if _, err := repo.UpsertNotificationRoute(context.Background(), &model.NotificationRouteSetting{
		ScopePath:        "/alice/sales",
		Channel:          "feishu",
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "销售默认群",
		RequireAuth:      true,
		WebhookURLCipher: "sales-cipher-url",
	}); err != nil {
		t.Fatalf("create parent route: %v", err)
	}
	child, err := repo.UpsertNotificationRoute(context.Background(), &model.NotificationRouteSetting{
		ScopePath:        "/alice/sales/orders",
		Channel:          "wecom",
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "订单群",
		RequireAuth:      true,
		WebhookURLCipher: "orders-cipher-url",
	})
	if err != nil {
		t.Fatalf("create child route: %v", err)
	}
	if _, err := repo.UpsertNotificationRoute(context.Background(), &model.NotificationRouteSetting{
		ScopePath:        "/alice/support",
		Channel:          "dingtalk",
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "支持群",
		RequireAuth:      true,
		WebhookURLCipher: "support-cipher-url",
	}); err != nil {
		t.Fatalf("create unrelated route: %v", err)
	}
	summaryRoutes, err := repo.ListNotificationRoutesByRoot(context.Background(), "/alice/sales")
	if err != nil {
		t.Fatalf("list route summary by root: %v", err)
	}
	summaryPaths := map[string]bool{}
	for _, route := range summaryRoutes {
		summaryPaths[route.ScopePath] = true
	}
	if len(summaryRoutes) != 2 || !summaryPaths["/alice/sales"] || !summaryPaths["/alice/sales/orders"] {
		t.Fatalf("unexpected summary routes: %#v", summaryRoutes)
	}
	if _, err := repo.UpsertNotificationRoute(context.Background(), &model.NotificationRouteSetting{
		ScopePath:        "/alice/sales_ops",
		Channel:          "feishu",
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "销售运维群",
		RequireAuth:      true,
		WebhookURLCipher: "sales-ops-cipher-url",
	}); err != nil {
		t.Fatalf("create underscore route: %v", err)
	}
	if _, err := repo.UpsertNotificationRoute(context.Background(), &model.NotificationRouteSetting{
		ScopePath:        "/alice/salesXops",
		Channel:          "feishu",
		Enabled:          true,
		DeliveryType:     "webhook",
		DisplayName:      "不应命中的群",
		RequireAuth:      true,
		WebhookURLCipher: "sales-xops-cipher-url",
	}); err != nil {
		t.Fatalf("create like-wildcard route: %v", err)
	}
	underscoreRoutes, err := repo.ListNotificationRoutesByRoot(context.Background(), "/alice/sales_ops")
	if err != nil {
		t.Fatalf("list underscore route summary: %v", err)
	}
	if len(underscoreRoutes) != 1 || underscoreRoutes[0].ScopePath != "/alice/sales_ops" {
		t.Fatalf("unexpected underscore summary routes: %#v", underscoreRoutes)
	}

	candidates := NotificationRouteCandidatePaths("/alice/sales/orders/notify.form")
	wantCandidates := []string{"/alice/sales/orders/notify.form", "/alice/sales/orders", "/alice/sales"}
	if len(candidates) != len(wantCandidates) {
		t.Fatalf("candidates = %#v", candidates)
	}
	for i := range wantCandidates {
		if candidates[i] != wantCandidates[i] {
			t.Fatalf("candidate[%d]=%q want %q all=%#v", i, candidates[i], wantCandidates[i], candidates)
		}
	}
	routes, err := repo.ListEnabledNotificationRoutesByPaths(context.Background(), candidates)
	if err != nil {
		t.Fatalf("list enabled routes: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("routes = %#v", routes)
	}

	if err := repo.RecordNotificationRouteDeliveryFailure(context.Background(), child.ID, "webhook returned 400", true); err != nil {
		t.Fatalf("record route failure: %v", err)
	}
	got, err := repo.GetNotificationRoute(context.Background(), "/alice/sales/orders", "wecom")
	if err != nil {
		t.Fatalf("get route: %v", err)
	}
	if got.FailCount != 1 || got.LastFailedAt == nil || got.LastTestAt == nil || got.LastError == "" {
		t.Fatalf("unexpected route failure status: %#v", got)
	}
	if err := repo.RecordNotificationRouteDeliverySuccess(context.Background(), child.ID, false); err != nil {
		t.Fatalf("record route success: %v", err)
	}
	got, err = repo.GetNotificationRoute(context.Background(), "/alice/sales/orders", "wecom")
	if err != nil {
		t.Fatalf("get route after success: %v", err)
	}
	if got.FailCount != 0 || got.LastSuccessAt == nil || got.LastError != "" {
		t.Fatalf("unexpected route success status: %#v", got)
	}
}
