package scheduledsdk

import (
	"context"
	"net/http"
	"testing"

	"github.com/kageos/kageos/pkg/contextx"
)

func TestExecutionRequestedEventWithAuditContext(t *testing.T) {
	event := ExecutionRequestedEvent{
		TaskID:          123,
		ExecutionID:     456,
		TraceID:         "trace-1",
		RequestUser:     "alice",
		RequestUserDept: "/org/dev",
		Metadata: map[string]string{
			MetadataCompanyCode:    "acme",
			MetadataCompanyName:    "Acme",
			MetadataCompanyLogoURL: "https://example.com/logo.png",
		},
	}

	ctx := event.WithAuditContext(context.Background())

	if got := contextx.GetAuditClientSource(ctx); got != contextx.ClientSourceScheduledTask {
		t.Fatalf("audit source = %q, want %s", got, contextx.ClientSourceScheduledTask)
	}
	if got := contextx.GetSourceType(ctx); got != contextx.SourceTypeScheduledTask {
		t.Fatalf("source type = %q, want %s", got, contextx.SourceTypeScheduledTask)
	}
	if got := contextx.GetSourceRef(ctx); got != "timer_task:123:execution:456" {
		t.Fatalf("source ref = %q, want timer_task:123:execution:456", got)
	}
	if got := contextx.GetTraceId(ctx); got != "trace-1" {
		t.Fatalf("trace id = %q, want trace-1", got)
	}
	if got := contextx.GetRequestUser(ctx); got != "alice" {
		t.Fatalf("request user = %q, want alice", got)
	}
	if got := contextx.GetRequestDepartmentFullPath(ctx); got != "/org/dev" {
		t.Fatalf("request dept = %q, want /org/dev", got)
	}
	if got := contextx.GetRequestCompanyCode(ctx); got != "acme" {
		t.Fatalf("company code = %q, want acme", got)
	}
	if got := contextx.GetRequestCompanyName(ctx); got != "Acme" {
		t.Fatalf("company name = %q, want Acme", got)
	}
	if got := contextx.GetRequestCompanyLogoURL(ctx); got != "https://example.com/logo.png" {
		t.Fatalf("company logo = %q", got)
	}
}

func TestExecutionRequestedEventApplyAuditHeaders(t *testing.T) {
	event := ExecutionRequestedEvent{
		TaskID:      123,
		ExecutionID: 456,
		TraceID:     "trace-1",
		RequestUser: "alice",
		Metadata: map[string]string{
			MetadataCompanyCode: "acme",
		},
	}
	header := http.Header{}

	event.ApplyAuditHeaders(header)

	if got := header.Get(contextx.ClientSourceHeader); got != contextx.ClientSourceScheduledTask {
		t.Fatalf("client source header = %q, want %s", got, contextx.ClientSourceScheduledTask)
	}
	if got := header.Get(contextx.SourceTypeHeader); got != contextx.SourceTypeScheduledTask {
		t.Fatalf("source type header = %q, want %s", got, contextx.SourceTypeScheduledTask)
	}
	if got := header.Get(contextx.SourceRefHeader); got != "timer_task:123:execution:456" {
		t.Fatalf("source ref header = %q, want timer_task:123:execution:456", got)
	}
	if got := header.Get(contextx.TraceIdHeader); got != "trace-1" {
		t.Fatalf("trace header = %q, want trace-1", got)
	}
	if got := header.Get(contextx.RequestUserHeader); got != "alice" {
		t.Fatalf("request user header = %q, want alice", got)
	}
	if got := header.Get(contextx.CompanyCodeHeader); got != "acme" {
		t.Fatalf("company code header = %q, want acme", got)
	}
}
