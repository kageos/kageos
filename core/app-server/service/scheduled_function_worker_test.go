package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestDecodeScheduledFunctionPayloadDefaultsActionAndTemplate(t *testing.T) {
	raw := json.RawMessage(`{
		"full_code_path": "/system/test22/ticket_management/ticket_submit.form",
		"payload": {"title": "自动工单"}
	}`)
	got, err := decodeScheduledFunctionPayload(scheduledsdk.ExecutionRequestedEvent{ExecutorPayload: raw})
	if err != nil {
		t.Fatalf("decodeScheduledFunctionPayload() error = %v", err)
	}
	if got.Action != "execute" || got.TemplateType != "form" {
		t.Fatalf("unexpected defaults: %#v", got)
	}
	if string(got.Payload) == "" {
		t.Fatal("payload should be retained")
	}
}

func TestScheduledFunctionPayloadURLQueryFromObject(t *testing.T) {
	got, err := scheduledFunctionPayloadURLQuery(json.RawMessage(`{"status":"open","ids":[1,2]}`))
	if err != nil {
		t.Fatalf("scheduledFunctionPayloadURLQuery() error = %v", err)
	}
	for _, want := range []string{"status=open", "ids=1", "ids=2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("query %q missing %q", got, want)
		}
	}
}

func TestScheduledFunctionPayloadURLQueryFromString(t *testing.T) {
	got, err := scheduledFunctionPayloadURLQuery(json.RawMessage(`"?status=open&page=1"`))
	if err != nil {
		t.Fatalf("scheduledFunctionPayloadURLQuery() error = %v", err)
	}
	if got != "status=open&page=1" {
		t.Fatalf("query = %q", got)
	}
}

func TestBuildScheduledCallbackAppReqSetsTargetRouter(t *testing.T) {
	req, err := buildScheduledCallbackAppReq(
		context.Background(),
		"/alice/crm/sales/leads.table",
		http.MethodPost,
		"OnTableAddRow",
		[]byte(`{"name":"Ada"}`),
		"",
	)
	if err != nil {
		t.Fatalf("buildScheduledCallbackAppReq() error = %v", err)
	}
	if req.Router != "/_callback" {
		t.Fatalf("Router = %q, want /_callback", req.Router)
	}
	if req.TargetRouter != "sales/leads.table" {
		t.Fatalf("TargetRouter = %q, want sales/leads.table", req.TargetRouter)
	}

	var envelope struct {
		Router string `json:"router"`
		Type   string `json:"type"`
	}
	if err := json.Unmarshal(req.Body, &envelope); err != nil {
		t.Fatalf("unmarshal callback envelope: %v", err)
	}
	if envelope.Router != req.TargetRouter || envelope.Type != "OnTableAddRow" {
		t.Fatalf("callback envelope mismatch: %#v target=%q", envelope, req.TargetRouter)
	}
}
