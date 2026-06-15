package appinvoke

import (
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/publicshare"
)

func TestRuntimeRequestCarriesClientSource(t *testing.T) {
	req := &dto.RequestAppReq{
		TraceId:      "trace-1",
		User:         "alice",
		App:          "demo",
		Version:      "v1",
		Method:       "POST",
		Router:       "/tools/run",
		ClientSource: "agent",
	}

	msg, err := BuildRuntimeRequestMsg(req)
	if err != nil {
		t.Fatalf("BuildRuntimeRequestMsg returned error: %v", err)
	}
	if got := msg.Header.Get(contextx.ClientSourceHeader); got != "agent" {
		t.Fatalf("client source header = %q, want agent", got)
	}

	meta, err := ParseRuntimeRequest(msg)
	if err != nil {
		t.Fatalf("ParseRuntimeRequest returned error: %v", err)
	}
	if meta.ClientSource != "agent" {
		t.Fatalf("meta.ClientSource = %q, want agent", meta.ClientSource)
	}
}

func TestRuntimeRequestCarriesPublicShareContext(t *testing.T) {
	req := &dto.RequestAppReq{
		TraceId:        "trace-1",
		User:           "system",
		App:            "tools",
		Version:        "v1",
		Method:         "POST",
		Router:         "/image/convert.form",
		AnonymousToken: "anon-token",
		ClientSource:   "public_share",
		SourceType:     "public_share",
		SourceRef:      "ps_123",
	}

	msg, err := BuildRuntimeRequestMsg(req)
	if err != nil {
		t.Fatalf("BuildRuntimeRequestMsg returned error: %v", err)
	}
	if got := msg.Header.Get(publicshare.AnonymousTokenHeader); got != "anon-token" {
		t.Fatalf("anonymous token header = %q, want anon-token", got)
	}
	if got := msg.Header.Get(contextx.SourceTypeHeader); got != "public_share" {
		t.Fatalf("source type header = %q, want public_share", got)
	}
	if got := msg.Header.Get(contextx.SourceRefHeader); got != "ps_123" {
		t.Fatalf("source ref header = %q, want ps_123", got)
	}

	meta, err := ParseRuntimeRequest(msg)
	if err != nil {
		t.Fatalf("ParseRuntimeRequest returned error: %v", err)
	}
	if meta.AnonymousToken != "anon-token" || meta.SourceType != "public_share" || meta.SourceRef != "ps_123" {
		t.Fatalf("parsed public share meta mismatch: %+v", meta)
	}
}

func TestRuntimeRequestCarriesTargetRouter(t *testing.T) {
	req := &dto.RequestAppReq{
		TraceId:      "trace-1",
		User:         "alice",
		App:          "crm",
		Version:      "v1",
		Method:       "POST",
		Router:       "/_callback",
		TargetRouter: "sales/leads.table",
	}

	msg, err := BuildRuntimeRequestMsg(req)
	if err != nil {
		t.Fatalf("BuildRuntimeRequestMsg returned error: %v", err)
	}
	if got := msg.Header.Get(TargetRouterHeader); got != "sales/leads.table" {
		t.Fatalf("target router header = %q, want sales/leads.table", got)
	}

	meta, err := ParseRuntimeRequest(msg)
	if err != nil {
		t.Fatalf("ParseRuntimeRequest returned error: %v", err)
	}
	if meta.TargetRouter != "sales/leads.table" {
		t.Fatalf("meta.TargetRouter = %q, want sales/leads.table", meta.TargetRouter)
	}
}
