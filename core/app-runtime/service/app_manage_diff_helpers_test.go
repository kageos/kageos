package service

import (
	"context"
	"testing"
)

func TestParseDiffDataParsesCallbackPayload(t *testing.T) {
	t.Parallel()

	service := &AppManageService{}
	diff := service.parseDiffData(context.Background(), map[string]interface{}{
		"add": []interface{}{
			map[string]interface{}{
				"router":  "/ticket/list",
				"method":  "GET",
				"name":    "ticket_list",
				"summary": "list tickets",
			},
		},
		"packages": []interface{}{
			map[string]interface{}{
				"code":         "ticket",
				"name":         "Ticket",
				"desc":         "ticket package",
				"router_group": "/ticket",
				"full_path":    "/alice/demo/ticket",
			},
		},
	}, "Test")
	if diff == nil {
		t.Fatalf("expected diff data to be parsed")
	}
	if len(diff.Add) != 1 || diff.Add[0].Router != "/ticket/list" {
		t.Fatalf("unexpected add diff: %+v", diff.Add)
	}
	if len(diff.Packages) != 1 || diff.Packages[0].RouterGroup != "/ticket" {
		t.Fatalf("unexpected packages diff: %+v", diff.Packages)
	}
}
