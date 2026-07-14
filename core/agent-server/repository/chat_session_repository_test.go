package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newChatSessionRepositoryTestRepo(t *testing.T) *ChatSessionRepository {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatalf("migrate chat sessions: %v", err)
	}
	return NewChatSessionRepository(db)
}

func TestListWorkspaceSessionsMatchesSuffixlessResourcePath(t *testing.T) {
	repo := newChatSessionRepositoryTestRepo(t)
	ctx := context.Background()
	sessions := []*model.AgentChatSession{
		{SessionID: "resource", User: "alice", FullCodePath: "/system/info", ResourceFullCodePath: "/system/info/site_monitor"},
		{SessionID: "directory", User: "alice", FullCodePath: "/system/info/site_monitor"},
		{SessionID: "other", User: "alice", FullCodePath: "/system/info/other"},
		{SessionID: "other-user", User: "bob", FullCodePath: "/system/info", ResourceFullCodePath: "/system/info/site_monitor"},
	}
	for _, session := range sessions {
		if err := repo.Create(ctx, session); err != nil {
			t.Fatalf("create session %s: %v", session.SessionID, err)
		}
	}

	got, total, err := repo.ListWorkspaceSessions(ctx, WorkspaceSessionListOptions{
		FullCodePath: "/system/info/site_monitor",
		User:         "alice",
		SessionScope: "all",
		Limit:        20,
	})
	if err != nil {
		t.Fatalf("ListWorkspaceSessions: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Fatalf("sessions total=%d len=%d, want both resource and legacy directory sessions", total, len(got))
	}
	seen := map[string]bool{}
	for _, session := range got {
		seen[session.SessionID] = true
	}
	if !seen["resource"] || !seen["directory"] || seen["other"] || seen["other-user"] {
		t.Fatalf("unexpected sessions: %#v", seen)
	}
}
