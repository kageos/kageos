package query

import "testing"

func TestPageSortReqDefaults(t *testing.T) {
	req := &PageSortReq{}

	if got := req.GetLimit(); got != 20 {
		t.Fatalf("GetLimit() = %d, want 20", got)
	}
	if got := req.GetLimit(50); got != 50 {
		t.Fatalf("GetLimit(50) = %d, want 50", got)
	}
	if got := req.GetPage(); got != 1 {
		t.Fatalf("GetPage() = %d, want 1", got)
	}
	if got := req.GetOffset(); got != 0 {
		t.Fatalf("GetOffset() = %d, want 0", got)
	}
}

func TestPageSortReqPagingAndOrder(t *testing.T) {
	req := &PageSortReq{Page: 3, PageSize: 25, Sorts: `[{"field":"created_at","order":"desc"},{"field":"name","order":"asc"}]`}

	if got := req.GetLimit(); got != 25 {
		t.Fatalf("GetLimit() = %d, want 25", got)
	}
	if got := req.GetOffset(); got != 50 {
		t.Fatalf("GetOffset() = %d, want 50", got)
	}
	if got := req.GetOrder(); got != "`created_at` DESC, `name` ASC" {
		t.Fatalf("GetOrder() = %q", got)
	}
}

func TestPageSortReqSortsJSON(t *testing.T) {
	req := &PageSortReq{Sorts: `[{"field":"score","order":"desc"},{"field":"id","order":"asc"}]`}

	if got := req.GetOrder(); got != "`score` DESC, `id` ASC" {
		t.Fatalf("GetOrder() = %q", got)
	}
}
