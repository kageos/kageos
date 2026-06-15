package service

import (
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestRequestFunctionFullCodePathUsesTargetRouterForCallback(t *testing.T) {
	req := &dto.RequestAppReq{
		User:         "alice",
		App:          "crm",
		Router:       "/_callback",
		TargetRouter: "sales/leads.table",
		Body:         []byte(`{"router":"wrong/path.form"}`),
	}

	got := requestFunctionFullCodePath(req)
	if got != "/alice/crm/sales/leads.table" {
		t.Fatalf("full code path = %q, want /alice/crm/sales/leads.table", got)
	}
}
