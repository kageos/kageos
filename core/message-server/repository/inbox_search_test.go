package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kageos/kageos/dto"
)

func TestInboxSearchScopesAndPagination(t *testing.T) {
	repo := newTestMessageRepo(t)
	ctx := context.Background()
	// The oldest matching message must remain searchable beyond the first 100 rows.
	for i := 0; i < 105; i++ {
		_, err := repo.Create(ctx, dto.MessageSendMeta{From: "system", SourcePath: "/owner/app/orders/notify.form", SourceTitle: "库存巡检"}, dto.MessageSendPayload{Title: fmt.Sprintf("库存 %03d", i), Content: "补货 100% SKU_A !"}, []string{"alice"})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := repo.Create(ctx, dto.MessageSendMeta{From: "system", SourcePath: "/owner/secret/notify.form"}, dto.MessageSendPayload{Title: "库存 confidential", Content: "库存"}, []string{"bob"})
	if err != nil {
		t.Fatal(err)
	}
	for _, keyword := range []string{"库存", "库存巡检", "100%", "SKU_A", "!"} {
		rows, total, err := repo.ListInbox(ctx, "alice", InboxListFilter{Query: keyword, SourcePath: "/owner/app", IncludeChildren: true}, 100, 20)
		if err != nil || total != 105 || len(rows) != 5 {
			t.Fatalf("query %q: total=%d len=%d err=%v", keyword, total, len(rows), err)
		}
	}
	rows, total, err := repo.ListInbox(ctx, "alice", InboxListFilter{Query: "库存 000"}, 0, 20)
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("old message: total=%d rows=%d err=%v", total, len(rows), err)
	}
	if err := repo.MarkRead(ctx, "alice", rows[0].ID); err != nil {
		t.Fatal(err)
	}
	_, total, err = repo.ListInbox(ctx, "alice", InboxListFilter{Query: "库存 000", Status: "unread"}, 0, 20)
	if err != nil || total != 0 {
		t.Fatalf("unread search total=%d err=%v", total, err)
	}
	for _, keyword := range []string{"confidential", "100_", "SKU%Z", "' OR 1=1 --"} {
		_, total, err := repo.ListInbox(ctx, "alice", InboxListFilter{Query: keyword}, 0, 20)
		if err != nil || total != 0 {
			t.Fatalf("literal/private query %q: total=%d err=%v", keyword, total, err)
		}
	}
	future := time.Now().Add(24 * time.Hour)
	_, total, err = repo.ListInbox(ctx, "alice", InboxListFilter{Query: "库存", Since: &future}, 0, 20)
	if err != nil || total != 0 {
		t.Fatalf("time search total=%d err=%v", total, err)
	}
	// Tied timestamps must still have a stable order across pages.
	first, _, err := repo.ListInbox(ctx, "alice", InboxListFilter{Query: "库存"}, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := repo.ListInbox(ctx, "alice", InboxListFilter{Query: "库存"}, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	for _, row := range first {
		seen[row.ID] = true
	}
	for _, row := range second {
		if seen[row.ID] {
			t.Fatalf("duplicate id across pages: %d", row.ID)
		}
	}
}
