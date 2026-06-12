package service

import (
	"encoding/json"
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
