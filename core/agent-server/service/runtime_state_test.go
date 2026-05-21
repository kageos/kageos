package service

import (
	"context"
	"testing"
	"time"

	"github.com/kageos/kageos/dto"
)

func TestInMemoryRuntimeStateStoreSummaryAggregatesToAncestors(t *testing.T) {
	store := NewInMemoryRuntimeStateStore()
	now := time.Now()
	err := store.Upsert(context.Background(), dto.RuntimeStateItem{
		Key:          "workspace_session:s1",
		Kind:         RuntimeStateKindWorkspaceSession,
		Status:       RuntimeStateStatusThinking,
		FullCodePath: "/u/app/ticket",
		StartedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	err = store.Upsert(context.Background(), dto.RuntimeStateItem{
		Key:          "workspace_session:s2",
		Kind:         RuntimeStateKindWorkspaceSession,
		Status:       RuntimeStateStatusToolRunning,
		FullCodePath: "/u/app/ticket/report",
		StartedAt:    now,
		UpdatedAt:    now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	summaries, err := store.Summary(context.Background(), RuntimeStateFilter{RootFullCodePath: "/u/app"})
	if err != nil {
		t.Fatalf("Summary returned error: %v", err)
	}

	root := summaries["/u/app"]
	if root.RunningCount != 2 || root.ManualRunningCount != 2 {
		t.Fatalf("root summary = %+v, want running=2 manual=2", root)
	}
	if root.BadgeText != "2" || root.BadgeTone != "tool" || root.DominantStatus != RuntimeStateStatusToolRunning {
		t.Fatalf("root display summary = %+v, want badge_text=2 badge_tone=tool dominant=tool_running", root)
	}
	ticket := summaries["/u/app/ticket"]
	if ticket.ThinkingCount != 1 || ticket.ToolRunningCount != 1 {
		t.Fatalf("ticket summary = %+v, want thinking=1 tool_running=1", ticket)
	}
	report := summaries["/u/app/ticket/report"]
	if report.RunningCount != 1 || report.ManualRunningCount != 1 {
		t.Fatalf("report summary = %+v, want running=1 manual=1", report)
	}
}

func TestInMemoryRuntimeStateStoreExpiresItems(t *testing.T) {
	store := NewInMemoryRuntimeStateStore()
	expiredAt := time.Now().Add(-time.Second)
	if err := store.Upsert(context.Background(), dto.RuntimeStateItem{
		Key:          "workspace_session:expired",
		Kind:         RuntimeStateKindWorkspaceSession,
		Status:       RuntimeStateStatusFailed,
		FullCodePath: "/u/app",
		ExpiresAt:    &expiredAt,
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	items, err := store.List(context.Background(), RuntimeStateFilter{RootFullCodePath: "/u/app"})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items len = %d, want 0", len(items))
	}
}
