package app

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

func TestNewContextCarriesClientSource(t *testing.T) {
	a := &App{}

	ctx, err := a.NewContext(context.Background(), &dto.RequestAppReq{
		Method:       "POST",
		Router:       "/demo",
		ClientSource: "agent",
	})
	if err != nil {
		t.Fatalf("NewContext returned error: %v", err)
	}

	if got := ctx.GetClientSource(); got != "agent" {
		t.Fatalf("ctx.GetClientSource() = %q, want agent", got)
	}
	if got := contextx.GetClientSource(ctx); got != "agent" {
		t.Fatalf("contextx.GetClientSource(ctx) = %q, want agent", got)
	}
}
