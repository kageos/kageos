package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWorkspaceSessionFiltersKeepDirectoryAndResourceSeparate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatal(err)
	}
	rows := []*model.AgentChatSession{
		{SessionID: "dir-human", FullCodePath: "/alice/home/tools", Source: model.ChatSessionSourceWorkspace, User: "alice"},
		{SessionID: "resource-human", FullCodePath: "/alice/home/tools", ResourceFullCodePath: "/alice/home/tools/run", Source: model.ChatSessionSourceWorkspace, User: "alice"},
		{SessionID: "legacy-resource", FullCodePath: "/alice/home/tools/run", Source: model.ChatSessionSourceWorkspace, User: "alice"},
		{SessionID: "agent-7", FullCodePath: "/alice/home/tools", ResourceFullCodePath: "/alice/home/tools/run", Source: model.ChatSessionSourceAutomationAgent, AutomationTaskID: 7, AutomationTaskTitle: "巡检", User: "alice"},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewChatSessionRepository(db)
	human, total, err := repo.ListWorkspaceSessions(context.Background(), WorkspaceSessionListOptions{
		FullCodePath:         "/alice/home/tools",
		ResourceFullCodePath: "/alice/home/tools/run",
		User:                 "alice",
		SessionScope:         "human",
		Limit:                20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(human) != 2 {
		t.Fatalf("resource human sessions = %d/%d, want 2", len(human), total)
	}
	agents, err := repo.ListWorkspaceAutomationAgents(context.Background(), "/alice/home/tools", "/alice/home/tools/run", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].TaskID != 7 || agents[0].TaskTitle != "巡检" {
		t.Fatalf("unexpected automation agents: %#v", agents)
	}
}
