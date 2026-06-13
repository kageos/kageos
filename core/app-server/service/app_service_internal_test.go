package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestSyncUpdatedAppMetadataSkipsWhenDiffMissing(t *testing.T) {
	svc := &AppService{}

	if warning := svc.syncUpdatedAppMetadata(context.Background(), 1, nil); warning != "" {
		t.Fatalf("expected empty warning for nil diff, got %q", warning)
	}
}

func TestFinalizeReleasedAppMetadataRequiresApp(t *testing.T) {
	svc := &AppService{}

	_, err := svc.finalizeReleasedAppMetadata(context.Background(), "test", nil, "alice", "demo", "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "应用不存在" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestBuildFormScheduleTaskRequestSupportsCron(t *testing.T) {
	req, err := buildFormScheduleTaskRequest(context.Background(), &appMetadataSyncState{
		app:         &model.App{User: "system", Code: "demo"},
		requestUser: "system",
	}, &dto.ApiInfo{
		Name:         "会议即将开始提醒",
		TemplateType: "form",
		FullCodePath: "/system/demo/meeting/meeting_room_notify_soon.form",
	}, dto.FormScheduleConfig{
		Code:     "meeting_reminder_soon",
		Title:    "会议即将开始提醒",
		CronExpr: "*/2 * * * *",
		Timezone: "Asia/Shanghai",
		MaxRuns:  3,
		Body:     json.RawMessage(`{"lead_minutes":5}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.ExecutorKey != ScheduledFunctionExecutorKey || req.Schedule.Type != scheduledsdk.ScheduleCron {
		t.Fatalf("unexpected scheduled request: %#v", req)
	}
	if req.Schedule.CronExpr != "*/2 * * * *" || req.Schedule.Timezone != "Asia/Shanghai" || req.Schedule.MaxRuns != 3 {
		t.Fatalf("unexpected cron schedule: %#v", req.Schedule)
	}
	if req.IdempotencyKey == "" || req.SourceRef != "/system/demo/meeting/meeting_room_notify_soon.form" {
		t.Fatalf("missing stable identity/source: %#v", req)
	}
	var payload struct {
		FullCodePath string          `json:"full_code_path"`
		TemplateType string          `json:"template_type"`
		Action       string          `json:"action"`
		Method       string          `json:"method"`
		Payload      json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(req.ExecutorPayload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.FullCodePath != req.SourceRef || payload.TemplateType != "form" || payload.Action != "execute" || payload.Method != "POST" {
		t.Fatalf("unexpected executor payload: %#v", payload)
	}
	if string(payload.Payload) != `{"lead_minutes":5}` {
		t.Fatalf("payload body = %s", payload.Payload)
	}
}

func TestScheduledSDKScheduleFromFormScheduleRejectsAmbiguousInput(t *testing.T) {
	_, err := scheduledSDKScheduleFromFormSchedule(dto.FormScheduleConfig{
		Code:         "bad",
		CronExpr:     "*/2 * * * *",
		EverySeconds: 120,
	})
	if err == nil {
		t.Fatal("expected ambiguous schedule error")
	}
}

func TestApplyFunctionNodeMetadataOverwritesStaleFields(t *testing.T) {
	svc := &AppService{}
	tree := &model.ServiceTree{
		Code:         "old_code",
		Name:         "Old Name",
		Description:  "old desc",
		TemplateType: "form",
		RefID:        1,
		Tags:         "legacy,stale",
		Connectors:   "legacy",
	}
	api := &dto.ApiInfo{
		Code:         "ticket_list",
		Name:         "工单列表",
		Desc:         `新的描述`,
		TemplateType: "table",
		Connectors:   []string{"GitHub", "github", "slack"},
		ConnectorEndpoints: []dto.ConnectorEndpoint{
			{Provider: "GitHub", Method: "get", URL: "/user", Name: "当前用户", RequiredScopes: []string{"read:user"}},
			{Provider: "github", Method: "GET", URL: "/user", Name: "duplicate", RequiredScopes: []string{"user:email"}},
		},
	}

	svc.applyFunctionNodeMetadata(tree, api, 42)

	if tree.Code != "ticket_list" || tree.Name != "工单列表" {
		t.Fatalf("unexpected code/name after apply: %+v", tree)
	}
	if tree.Description != "新的描述" || tree.TemplateType != "table" {
		t.Fatalf("unexpected description/template type after apply: %+v", tree)
	}
	if tree.RefID != 42 {
		t.Fatalf("expected ref id to be updated, got %d", tree.RefID)
	}
	if tree.Tags != "" {
		t.Fatalf("expected stale tags to be cleared, got %q", tree.Tags)
	}
	if tree.Connectors != "github,slack" {
		t.Fatalf("expected connectors to be normalized, got %q", tree.Connectors)
	}
	endpoints := splitConnectorEndpoints(tree.ConnectorEndpoints)
	if len(endpoints) != 1 {
		t.Fatalf("expected one normalized endpoint, got %#v", endpoints)
	}
	if endpoint := endpoints[0]; endpoint.Provider != "github" || endpoint.Method != "GET" || endpoint.URL != "/user" || endpoint.Name != "当前用户" {
		t.Fatalf("unexpected endpoint after apply: %#v", endpoint)
	}
	if got := endpoints[0].RequiredScopes; len(got) != 2 || got[0] != "read:user" || got[1] != "user:email" {
		t.Fatalf("unexpected required scopes after apply: %#v", got)
	}
}
