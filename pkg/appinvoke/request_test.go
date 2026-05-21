package appinvoke

import (
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
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
