package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSessionHistoryToolsSearchAndReadExactRawMessages(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatalf("migrate sessions: %v", err)
	}
	if err := createSQLiteAgentChatMessagesTable(db); err != nil {
		t.Fatalf("migrate messages: %v", err)
	}
	sessionRepo := repository.NewChatSessionRepository(db)
	messageRepo := repository.NewChatMessageRepository(db)
	current := &model.AgentChatSession{SessionID: "history-current", Source: SourceWorkspace, Status: model.ChatSessionStatusActive, User: "tester"}
	foreign := &model.AgentChatSession{SessionID: "history-foreign", Source: SourceWorkspace, Status: model.ChatSessionStatusActive, User: "tester"}
	if err := sessionRepo.Create(current); err != nil {
		t.Fatalf("create current session: %v", err)
	}
	if err := sessionRepo.Create(foreign); err != nil {
		t.Fatalf("create foreign session: %v", err)
	}
	raw := &model.AgentChatMessage{SessionID: current.SessionID, Role: RoleUser, Content: "alpha-decision: 报表必须使用自然月，不能使用滚动 30 天。", User: "tester"}
	longRaw := &model.AgentChatMessage{SessionID: current.SessionID, Role: RoleAssistant, Content: strings.Repeat("x", 12500) + "tail-exact-marker", User: "tester"}
	foreignRaw := &model.AgentChatMessage{SessionID: foreign.SessionID, Role: RoleUser, Content: "foreign-secret-marker", User: "tester"}
	if err := messageRepo.Create(raw); err != nil {
		t.Fatalf("create raw message: %v", err)
	}
	if err := messageRepo.Create(longRaw); err != nil {
		t.Fatalf("create long raw message: %v", err)
	}
	if err := messageRepo.Create(foreignRaw); err != nil {
		t.Fatalf("create foreign message: %v", err)
	}
	svc := &WorkspaceChatService{sessionRepo: sessionRepo, messageRepo: messageRepo}
	ctx := contextx.WithWorkspaceSession(contextx.WithRequestUser(context.Background(), "tester"), current.SessionID, "current", WorkspaceRoleReviewer)

	searchResult := svc.searchSessionHistoryTool(ctx, map[string]interface{}{"query": "alpha-decision"})
	if searchResult.IsError {
		t.Fatalf("search returned error: %s", searchResult.Content)
	}
	searchData, ok := searchResult.Data.(searchSessionHistoryData)
	if !ok || searchData.Count != 1 || searchData.Hits[0].MessageID != raw.ID {
		t.Fatalf("search result = %#v", searchResult.Data)
	}

	readResult := svc.readSessionMessagesTool(ctx, map[string]interface{}{"message_ids": []interface{}{float64(raw.ID)}})
	if readResult.IsError {
		t.Fatalf("read returned error: %s", readResult.Content)
	}
	readData, ok := readResult.Data.(readSessionMessagesData)
	if !ok || readData.Count != 1 || !strings.Contains(readData.Messages[0].Content, "自然月") {
		t.Fatalf("exact read result = %#v", readResult.Data)
	}

	foreignResult := svc.readSessionMessagesTool(ctx, map[string]interface{}{"message_ids": []interface{}{float64(foreignRaw.ID)}})
	if foreignResult.IsError {
		t.Fatalf("foreign ID should be invisible rather than leak an authorization oracle: %s", foreignResult.Content)
	}
	foreignData, ok := foreignResult.Data.(readSessionMessagesData)
	if !ok || foreignData.Count != 0 {
		t.Fatalf("cross-session raw message leaked: %#v", foreignResult.Data)
	}

	firstPageResult := svc.readSessionMessagesTool(ctx, map[string]interface{}{"message_ids": []interface{}{float64(longRaw.ID)}, "max_chars": float64(12000)})
	firstPage, ok := firstPageResult.Data.(readSessionMessagesData)
	if firstPageResult.IsError || !ok || !firstPage.Truncated || firstPage.NextOffset != 12000 {
		t.Fatalf("first exact page = %#v", firstPageResult.Data)
	}
	secondPageResult := svc.readSessionMessagesTool(ctx, map[string]interface{}{"message_ids": []interface{}{float64(longRaw.ID)}, "offset_chars": float64(firstPage.NextOffset), "max_chars": float64(12000)})
	secondPage, ok := secondPageResult.Data.(readSessionMessagesData)
	if secondPageResult.IsError || !ok || secondPage.Truncated || !strings.Contains(secondPage.Messages[0].Content, "tail-exact-marker") {
		t.Fatalf("second exact page = %#v", secondPageResult.Data)
	}
}

func TestCheckpointCandidateProtectsCurrentTurnAndUsesTokenTail(t *testing.T) {
	messages := make([]*model.AgentChatMessage, 0, 21)
	for i := int64(1); i <= 20; i++ {
		messages = append(messages, &model.AgentChatMessage{Role: RoleUser, Content: strings.Repeat("历史", 800), User: "tester"})
		messages[len(messages)-1].ID = i
	}
	current := &model.AgentChatMessage{Role: RoleUser, Content: "当前任务不能进入检查点", User: "tester"}
	current.ID = 21
	messages = append(messages, current)

	candidate := selectWorkspaceCheckpointCandidate(messages, current.ID, 16000, nil)
	if len(candidate) == 0 {
		t.Fatal("expected an older checkpoint candidate")
	}
	if candidate[len(candidate)-1].ID >= current.ID {
		t.Fatalf("current turn was selected for checkpoint: last=%d current=%d", candidate[len(candidate)-1].ID, current.ID)
	}
	largerMessages := make([]*model.AgentChatMessage, 0, len(messages))
	for _, message := range messages {
		copyMessage := *message
		if copyMessage.ID < current.ID {
			copyMessage.Content = strings.Repeat("更长历史", 1600)
		}
		largerMessages = append(largerMessages, &copyMessage)
	}
	largerCandidate := selectWorkspaceCheckpointCandidate(largerMessages, current.ID, 16000, nil)
	if len(largerCandidate) == len(candidate) {
		t.Fatalf("candidate should change with token volume instead of a fixed count: normal=%d larger=%d", len(candidate), len(largerCandidate))
	}
}
