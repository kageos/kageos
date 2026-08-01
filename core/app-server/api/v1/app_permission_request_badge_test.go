package v1

import (
	"testing"

	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
)

func TestAnnotatePermissionRequestBadges(t *testing.T) {
	nodes := []*dto.GetServiceTreeResp{
		{
			FullCodePath: "/alice/ops",
			Children: []*dto.GetServiceTreeResp{
				{FullCodePath: "/alice/ops/orders"},
				{FullCodePath: "/alice/ops/reports"},
			},
		},
	}
	summary := &service.PermissionRequestWorkspaceSummary{
		Paths: map[string]service.PermissionRequestPathSummary{
			"/alice/ops/orders": {
				OwnPendingCount:    1,
				ReviewPendingCount: 2,
			},
		},
	}

	annotatePermissionRequestBadges(nodes, summary)

	orders := nodes[0].Children[0]
	if orders.PermissionRequests == nil || orders.PermissionRequests.OwnPendingCount != 1 || orders.PermissionRequests.ReviewPendingCount != 2 {
		t.Fatalf("unexpected orders badge counts: %#v", orders)
	}
	reports := nodes[0].Children[1]
	if reports.PermissionRequests == nil || reports.PermissionRequests.OwnPendingCount != 0 || reports.PermissionRequests.ReviewPendingCount != 0 {
		t.Fatalf("unexpected reports badge counts: %#v", reports)
	}
}
