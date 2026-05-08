package service

import (
	"context"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestShouldSuggestExecuteGuide(t *testing.T) {
	if !shouldSuggestExecuteGuide("run_form_submit") {
		t.Fatal("expected run_form_submit to suggest execute guide")
	}
	if !shouldSuggestExecuteGuide("run_table_search") {
		t.Fatal("expected run_table_search to suggest execute guide")
	}
	if !shouldSuggestExecuteGuide("run_table_batch_create") {
		t.Fatal("expected run_table_batch_create to suggest execute guide")
	}
	if !shouldSuggestExecuteGuide("run_table_delete") {
		t.Fatal("expected run_table_delete to suggest execute guide")
	}
	if shouldSuggestExecuteGuide("read_doc") {
		t.Fatal("did not expect read_doc to suggest execute guide")
	}
}

func TestAppendExecuteGuideHint(t *testing.T) {
	msg := appendExecuteGuideHint("run_table_create", "创建失败：字段缺失")
	if msg != "创建失败：字段缺失" {
		t.Fatalf("expected original error unchanged, got %q", msg)
	}

	plain := appendExecuteGuideHint("read_doc", "读取失败")
	if plain != "读取失败" {
		t.Fatalf("expected non execute tool error unchanged, got %q", plain)
	}
}

func TestGuideDocPathsFromReadDocArgsSupportsCommaSeparatedPaths(t *testing.T) {
	paths := guideDocPathsFromReadDocArgs(map[string]interface{}{
		"directory": "/system/prompt/platform-capability-boundaries,/system/prompt/sdk/agent-app-sdk-readme/",
	})
	loaded := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		loaded[path] = struct{}{}
	}

	if _, ok := loaded["/system/prompt/platform-capability-boundaries"]; !ok {
		t.Fatalf("expected platform capability boundaries to be marked loaded: %#v", loaded)
	}
	if _, ok := loaded["/system/prompt/sdk/agent-app-sdk-readme"]; !ok {
		t.Fatalf("expected sdk readme to be marked loaded: %#v", loaded)
	}
}

func TestWithAgentToolExecutionContextMarksSourceAndSession(t *testing.T) {
	base := contextx.WithClientSource(context.Background(), "browser")

	ctx := withAgentToolExecutionContext(base, "session-1")

	if got := contextx.GetClientSource(ctx); got != "agent" {
		t.Fatalf("client source = %q, want agent", got)
	}
	if got := getWorkspaceSessionID(ctx); got != "session-1" {
		t.Fatalf("session id = %q, want session-1", got)
	}
}

func TestCreateWorkspaceHandoffArchivesSourceAndCreatesArtifactSession(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sessionRepo := repository.NewChatSessionRepository(db)
	source := &model.AgentChatSession{
		TreeID:        7,
		FullCodePath:  "/liubeiluo/demo",
		Source:        SourceWorkspace,
		SessionID:     "source-session",
		Title:         "PRD 讨论",
		ModeCode:      "dev",
		Status:        model.ChatSessionStatusActive,
		ContextPolicy: ContextPolicyFull,
		User:          "tester",
	}
	if err := sessionRepo.Create(source); err != nil {
		t.Fatalf("create source: %v", err)
	}

	svc := &WorkspaceChatService{sessionRepo: sessionRepo}
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "tester")
	resp, err := svc.CreateWorkspaceHandoff(ctx, &dto.WorkspaceHandoffReq{
		SourceSessionID: "source-session",
		FullCodePath:    "/liubeiluo/demo",
		TargetRole:      "app.create",
		ArtifactKind:    "agent_app_prd",
		Artifact:        []byte(`{"kind":"agent_app_prd","project":{"name":"工单管理"}}`),
		Remark:          "优先做列表",
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	if resp.SessionID == "" || resp.SessionID == source.SessionID {
		t.Fatalf("unexpected target session id: %q", resp.SessionID)
	}
	if resp.ContextPolicy != ContextPolicyArtifactOnly {
		t.Fatalf("context policy=%q want %q", resp.ContextPolicy, ContextPolicyArtifactOnly)
	}
	if !strings.Contains(resp.Content, "target_role 固定为 app.create") {
		t.Fatalf("content should include target role, got %q", resp.Content)
	}
	if !strings.Contains(resp.Content, `"kind": "agent_app_prd"`) {
		t.Fatalf("content should include formatted artifact JSON, got %q", resp.Content)
	}

	archived, err := sessionRepo.GetBySessionID("source-session")
	if err != nil {
		t.Fatalf("get source: %v", err)
	}
	if !archived.ArchivedForModel || archived.ContextPolicy != ContextPolicyDisplayOnly {
		t.Fatalf("source not archived for model: %#v", archived)
	}
	target, err := sessionRepo.GetBySessionID(resp.SessionID)
	if err != nil {
		t.Fatalf("get target: %v", err)
	}
	if target.ParentSessionID != source.SessionID || target.HandoffKind != "agent_app_prd" || target.HandoffTargetRole != "app.create" {
		t.Fatalf("target handoff metadata wrong: %#v", target)
	}
}
