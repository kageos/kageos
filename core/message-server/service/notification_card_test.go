package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	msgmodel "github.com/kageos/kageos/core/message-server/model"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDefaultNotificationCardBuilderIncludesSourceAndActions(t *testing.T) {
	entry := &msgmodel.MessageEntry{
		ID:                    42,
		CreatedAt:             time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC),
		From:                  "scheduler",
		FullCodePath:          "/alice/demo/ops/daily_report.form",
		TraceID:               "trace-1",
		SourceType:            "agent_tool",
		SourcePath:            "/alice/demo/ops/daily_report.form",
		SourceTitle:           "每日运营日报",
		SourceParentPath:      "/alice/demo/ops",
		SourceParentTitle:     "运营目录",
		WorkspaceSessionID:    "session-1",
		WorkspaceSessionTitle: "每日线索巡检",
		WorkspaceRole:         "automation_operator",
		Title:                 "【提醒】每日线索巡检完成",
		Content:               "发现 3 条高优先级线索，已写入线索表。",
		ContentType:           "markdown",
	}

	card := DefaultNotificationCardBuilder{}.BuildNotificationCard(context.Background(), entry, dto.MessageSendPayload{}, NotificationTarget{
		Recipient: ResolvedRecipient{Username: "bob"},
		Channel:   NotificationChannelFeishu,
	}, NotificationCardBuildOptions{BaseURL: "https://kageos.example"})

	if card.ToUser != "bob" {
		t.Fatalf("to user = %q, want bob", card.ToUser)
	}
	if card.Level != NotificationLevelWarning {
		t.Fatalf("level = %q, want warning", card.Level)
	}
	if card.Source.Workspace != "alice/demo" {
		t.Fatalf("workspace = %q, want alice/demo", card.Source.Workspace)
	}
	if card.Source.Title != "每日运营日报" {
		t.Fatalf("source title = %q", card.Source.Title)
	}
	if !strings.Contains(card.Summary, "3 条高优先级线索") {
		t.Fatalf("summary = %q", card.Summary)
	}

	actions := map[string]string{}
	for _, action := range card.Actions {
		actions[action.Kind] = action.URL
	}
	if !strings.Contains(actions[NotificationActionDetail], "https://kageos.example/workspace/alice/demo/ops/daily_report.form") ||
		!strings.Contains(actions[NotificationActionDetail], "_open=inbox") ||
		!strings.Contains(actions[NotificationActionDetail], "_message_id=42") {
		t.Fatalf("detail action url = %q", actions[NotificationActionDetail])
	}
	if !strings.Contains(actions[NotificationActionSession], "_session_id=session-1") ||
		!strings.Contains(actions[NotificationActionSession], "_mws_sid=session-1") {
		t.Fatalf("session action url = %q", actions[NotificationActionSession])
	}
}

func TestFeishuCardRendererProducesInteractiveCard(t *testing.T) {
	card := sampleNotificationCard()
	card.Actions = append([]NotificationAction{
		{Kind: NotificationActionProcess, Label: "处理消息", URL: "https://kageos.example/m/action?t=kat_sample"},
	}, card.Actions...)
	payload, err := FeishuCardRenderer{}.Render(card)
	if err != nil {
		t.Fatalf("render feishu: %v", err)
	}
	if payload["msg_type"] != "interactive" {
		t.Fatalf("msg_type = %v", payload["msg_type"])
	}
	cardPayload, ok := payload["card"].(map[string]interface{})
	if !ok {
		t.Fatalf("card payload missing: %#v", payload["card"])
	}
	header := cardPayload["header"].(map[string]interface{})
	if header["template"] != "blue" {
		t.Fatalf("header template = %v, want blue", header["template"])
	}
	if cardPayload["schema"] != "2.0" {
		t.Fatalf("schema = %v, want 2.0", cardPayload["schema"])
	}
	if _, ok := cardPayload["card_link"]; !ok {
		t.Fatalf("expected card_link in feishu card")
	}
	body := cardPayload["body"].(map[string]interface{})
	elements := body["elements"].([]interface{})
	if len(elements) == 0 {
		t.Fatalf("expected feishu card elements")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	for _, want := range []string{"schema\":\"2.0", "normal_v2", "回复消息", "查看详情", "primary_filled", "具体内容"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("feishu card missing %q:\n%s", want, string(raw))
		}
	}
	for _, unwanted := range []string{"工作空间", "目录路径", "任务/会话", "发起人", "完整内容已保存"} {
		if strings.Contains(string(raw), unwanted) {
			t.Fatalf("feishu card should not include noisy context %q:\n%s", unwanted, string(raw))
		}
	}
	if strings.Contains(string(raw), "{{") {
		t.Fatalf("feishu card still contains template placeholders:\n%s", string(raw))
	}
}

func TestWeComMarkdownRendererIncludesContextAndLinks(t *testing.T) {
	payload, err := WeComMarkdownRenderer{}.Render(sampleNotificationCard())
	if err != nil {
		t.Fatalf("render wecom: %v", err)
	}
	if payload["msgtype"] != "markdown" {
		t.Fatalf("msgtype = %v", payload["msgtype"])
	}
	markdown := payload["markdown"].(map[string]interface{})
	content := markdown["content"].(string)
	for _, want := range []string{"【高优先级】线索巡检异常", "来源目录：线索巡检", "alice/demo", "[查看详情](https://kageos.example/workspace/alice/demo/leads?_open=inbox)"} {
		if !strings.Contains(content, want) {
			t.Fatalf("wecom content missing %q:\n%s", want, content)
		}
	}
}

func TestWeComTemplateCardRendererIncludesContextAndLinks(t *testing.T) {
	payload, err := WeComTemplateCardRenderer{}.Render(sampleNotificationCard())
	if err != nil {
		t.Fatalf("render wecom template card: %v", err)
	}
	if payload["msgtype"] != "template_card" {
		t.Fatalf("msgtype = %v", payload["msgtype"])
	}
	templateCard := payload["template_card"].(map[string]interface{})
	if templateCard["card_type"] != "text_notice" {
		t.Fatalf("card_type = %v", templateCard["card_type"])
	}
	mainTitle := templateCard["main_title"].(map[string]interface{})
	if mainTitle["title"] != "【高优先级】线索巡检异常" {
		t.Fatalf("main title = %#v", mainTitle)
	}
	horizontal := templateCard["horizontal_content_list"].([]interface{})
	if len(horizontal) == 0 {
		t.Fatalf("expected horizontal content list")
	}
	jumps := templateCard["jump_list"].([]interface{})
	if len(jumps) == 0 {
		t.Fatalf("expected jump list")
	}
	action := templateCard["card_action"].(map[string]interface{})
	if !strings.Contains(action["url"].(string), "https://kageos.example/workspace/alice/demo/leads?_open=inbox") {
		t.Fatalf("card action url = %#v", action)
	}
	raw, _ := json.Marshal(payload)
	for _, want := range []string{"来源目录", "线索巡检", "打开会话", "Kageos 自动通知"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("wecom template card missing %q:\n%s", want, string(raw))
		}
	}
}

func TestDingTalkActionCardRendererIncludesContextAndButtons(t *testing.T) {
	payload, err := DingTalkActionCardRenderer{}.Render(sampleNotificationCard())
	if err != nil {
		t.Fatalf("render dingtalk: %v", err)
	}
	if payload["msgtype"] != "actionCard" {
		t.Fatalf("msgtype = %v", payload["msgtype"])
	}
	actionCard := payload["actionCard"].(map[string]interface{})
	if actionCard["title"] != "【高优先级】线索巡检异常" {
		t.Fatalf("title = %v", actionCard["title"])
	}
	text := actionCard["text"].(string)
	for _, want := range []string{"Kageos 自动通知", "来源目录：线索巡检", "alice/demo", "完整内容已保存到 Kageos 站内信"} {
		if !strings.Contains(text, want) {
			t.Fatalf("dingtalk content missing %q:\n%s", want, text)
		}
	}
	for _, rawMarkdown := range []string{"**", "###", "[查看详情]", "`/alice/demo/leads`"} {
		if strings.Contains(text, rawMarkdown) {
			t.Fatalf("dingtalk content should avoid raw markdown %q:\n%s", rawMarkdown, text)
		}
	}
	buttons := actionCard["btns"].([]interface{})
	if len(buttons) == 0 {
		t.Fatalf("expected dingtalk action buttons")
	}
	first := buttons[0].(map[string]interface{})
	if first["title"] != "查看详情" || !strings.Contains(first["actionURL"].(string), "https://kageos.example/workspace/alice/demo/leads?_open=inbox") {
		t.Fatalf("unexpected first dingtalk button: %#v", first)
	}
}

func TestNotificationChannelProviderSupportsDingTalk(t *testing.T) {
	if !IsSupportedNotificationChannel(NotificationChannelDingTalk) {
		t.Fatalf("dingtalk should be supported")
	}
	provider, err := NewNotificationChannelProvider(NotificationChannelDingTalk, time.Second)
	if err != nil {
		t.Fatalf("new dingtalk provider: %v", err)
	}
	if provider.Channel() != NotificationChannelDingTalk {
		t.Fatalf("provider channel = %q", provider.Channel())
	}
}

func TestRegisterNotificationChannelDefinitionIncludesWebhookValidation(t *testing.T) {
	restoreNotificationChannelRegistryForTest(t)

	RegisterNotificationChannel(NotificationChannelDefinition{
		Channel: "slack_test",
		ProviderFactory: func(time.Duration) ChannelProvider {
			return &recordingChannelProvider{channel: "slack_test"}
		},
		ValidateWebhookURL: func(raw string) error {
			if raw != "https://hooks.slack.com/services/test" {
				return errors.New("invalid slack webhook")
			}
			return nil
		},
	})

	if !IsSupportedNotificationChannel("slack_test") {
		t.Fatal("slack_test should be supported")
	}
	provider, err := NewNotificationChannelProvider("slack_test", time.Second)
	if err != nil {
		t.Fatalf("new slack_test provider: %v", err)
	}
	if provider.Channel() != "slack_test" {
		t.Fatalf("provider channel = %q", provider.Channel())
	}
	if err := ValidateNotificationWebhookURL("slack_test", "https://hooks.slack.com/services/test"); err != nil {
		t.Fatalf("valid webhook rejected: %v", err)
	}
	if err := ValidateNotificationWebhookURL("slack_test", "https://example.com/hook"); err == nil {
		t.Fatal("invalid webhook accepted")
	}
}

func TestRegisterNotificationChannelRejectsDuplicateChannel(t *testing.T) {
	restoreNotificationChannelRegistryForTest(t)

	definition := NotificationChannelDefinition{
		Channel: "slack_test",
		ProviderFactory: func(time.Duration) ChannelProvider {
			return &recordingChannelProvider{channel: "slack_test"}
		},
	}
	RegisterNotificationChannel(definition)
	mustPanic(t, func() {
		RegisterNotificationChannel(definition)
	})
}

func TestMessageConsumerDeliversExternalNotificationTargets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := msgmodel.InitModels(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := msgrepo.NewMessageRepository(db)
	if _, err := repo.UpsertNotificationChannel(context.Background(), &msgmodel.NotificationChannelSetting{
		OwnerUsername:    "bob",
		Channel:          NotificationChannelFeishu,
		Enabled:          true,
		DeliveryType:     "webhook",
		WebhookURLCipher: "cipher-url",
	}); err != nil {
		t.Fatalf("create notification channel: %v", err)
	}

	provider := &recordingChannelProvider{channel: NotificationChannelFeishu}
	resolver := NotificationTargetResolverFunc(func(ctx context.Context, recipients []ResolvedRecipient, entry *msgmodel.MessageEntry, payload dto.MessageSendPayload) ([]NotificationTarget, error) {
		if entry == nil || entry.ID == 0 {
			t.Fatalf("resolver entry not persisted: %#v", entry)
		}
		if len(recipients) != 1 || recipients[0].Username != "bob" {
			t.Fatalf("recipients = %#v", recipients)
		}
		return []NotificationTarget{{
			Recipient:  recipients[0],
			Channel:    NotificationChannelFeishu,
			WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/test",
		}}, nil
	})
	svc := NewMessageConsumerService(
		repo,
		WithChannelProviders(provider),
		WithNotificationTargetResolver(resolver),
		WithNotificationCardBaseURL("https://kageos.example"),
	)

	err = svc.Consume(context.Background(), &dto.MessageSendEnvelope{
		Meta: dto.MessageSendMeta{
			From:                  "alice",
			FullCodePath:          "/alice/demo/leads",
			SourcePath:            "/alice/demo/leads",
			SourceTitle:           "线索巡检",
			WorkspaceSessionID:    "session-1",
			WorkspaceSessionTitle: "线索巡检任务",
		},
		Message: dto.MessageSendPayload{
			ToUsers: "bob",
			Title:   "线索巡检完成",
			Content: "已发现 2 条新线索。",
		},
	})
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if len(provider.calls) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(provider.calls))
	}
	call := provider.calls[0]
	if call.target.Recipient.Username != "bob" || call.card.ToUser != "bob" {
		t.Fatalf("provider call target/card = %#v / %#v", call.target, call.card)
	}
	processAction := notificationActionByKind(call.card.Actions, NotificationActionProcess)
	if !strings.Contains(processAction.URL, "https://kageos.example/m/action?t=kat_") {
		t.Fatalf("process action = %#v, actions = %#v", processAction, call.card.Actions)
	}
	detailAction := notificationActionByKind(call.card.Actions, NotificationActionDetail)
	if !strings.Contains(detailAction.URL, "https://kageos.example/workspace/alice/demo/leads") {
		t.Fatalf("card actions = %#v", call.card.Actions)
	}
	askAction := notificationActionByKind(call.card.Actions, NotificationActionAsk)
	if !strings.Contains(askAction.URL, "https://kageos.example/m?source_path=%2Falice%2Fdemo%2Fleads") {
		t.Fatalf("ask action = %#v, actions = %#v", askAction, call.card.Actions)
	}
	status, err := repo.GetNotificationChannel(context.Background(), "bob", NotificationChannelFeishu)
	if err != nil {
		t.Fatalf("get notification status: %v", err)
	}
	if status.LastSuccessAt == nil || status.FailCount != 0 || status.LastError != "" {
		t.Fatalf("unexpected delivery status: %#v", status)
	}
}

func sampleNotificationCard() NotificationCard {
	return NotificationCard{
		MessageID:   1,
		Title:       "【高优先级】线索巡检异常",
		Level:       NotificationLevelCritical,
		Summary:     "发现 2 条高优先级线索需要处理",
		Content:     "请尽快查看线索详情，并跟进负责人。",
		ContentType: "markdown",
		FromUser:    "scheduler",
		ToUser:      "bob",
		CreatedAt:   time.Date(2026, 6, 17, 10, 30, 0, 0, time.UTC),
		Source: NotificationCardSource{
			Workspace: "alice/demo",
			Path:      "/alice/demo/leads",
			Title:     "线索巡检",
		},
		Task: NotificationCardTask{
			SessionID:    "session-1",
			SessionTitle: "线索巡检任务",
		},
		Actions: []NotificationAction{
			{Kind: NotificationActionDetail, Label: "查看详情", URL: "https://kageos.example/workspace/alice/demo/leads?_open=inbox"},
			{Kind: NotificationActionSource, Label: "打开目录", URL: "https://kageos.example/workspace/alice/demo/leads"},
			{Kind: NotificationActionSession, Label: "打开会话", URL: "https://kageos.example/workspace/alice/demo/leads?_open=session&_session_id=session-1"},
		},
	}
}

type recordingChannelProvider struct {
	channel string
	calls   []recordingChannelCall
}

type recordingChannelCall struct {
	target NotificationTarget
	card   NotificationCard
}

func (p *recordingChannelProvider) Channel() string {
	return p.channel
}

func (p *recordingChannelProvider) Deliver(_ context.Context, target NotificationTarget, card NotificationCard) error {
	p.calls = append(p.calls, recordingChannelCall{target: target, card: card})
	return nil
}

func restoreNotificationChannelRegistryForTest(t *testing.T) {
	t.Helper()

	notificationChannelRegistry.RLock()
	order := append([]string(nil), notificationChannelRegistry.order...)
	channels := make(map[string]NotificationChannelDefinition, len(notificationChannelRegistry.channels))
	for channel, definition := range notificationChannelRegistry.channels {
		channels[channel] = definition
	}
	notificationChannelRegistry.RUnlock()

	t.Cleanup(func() {
		notificationChannelRegistry.Lock()
		notificationChannelRegistry.order = order
		notificationChannelRegistry.channels = channels
		notificationChannelRegistry.Unlock()
	})
}

func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
