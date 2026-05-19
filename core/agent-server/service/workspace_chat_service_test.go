package service

import (
	"context"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/streamloop"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
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

func TestWithAgentToolExecutionContextMarksSource(t *testing.T) {
	base := contextx.WithClientSource(context.Background(), "browser")

	ctx := withAgentToolExecutionContext(base, "session-1")

	if got := contextx.GetClientSource(ctx); got != "agent" {
		t.Fatalf("client source = %q, want agent", got)
	}
}

func TestSaveAssistantMessageStoresLLMMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := createSQLiteAgentChatMessagesTable(db); err != nil {
		t.Fatalf("migrate messages: %v", err)
	}
	messageRepo := repository.NewChatMessageRepository(db)
	svc := &WorkspaceChatService{messageRepo: messageRepo}
	meta := messageLLMMetadata{
		ConfigID:   12,
		ConfigName: "OpenAI Mini",
		Provider:   "openai",
		Model:      "gpt-4o-mini",
	}

	if err := svc.saveAssistantMessage(context.Background(), "session-llm", "ok", "tester", meta); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}
	messages, err := messageRepo.ListBySessionID("session-llm")
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	got := messages[0]
	if got.LLMConfigID != meta.ConfigID || got.LLMConfigName != meta.ConfigName || got.LLMProvider != meta.Provider || got.LLMModel != meta.Model {
		t.Fatalf("LLM metadata not stored: %#v", got)
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
	if err := createSQLiteAgentChatMessagesTable(db); err != nil {
		t.Fatalf("migrate messages: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkspaceHandoffPacket{}); err != nil {
		t.Fatalf("migrate handoff packets: %v", err)
	}
	sessionRepo := repository.NewChatSessionRepository(db)
	messageRepo := repository.NewChatMessageRepository(db)
	handoffRepo := repository.NewWorkspaceHandoffPacketRepository(db)
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

	svc := &WorkspaceChatService{sessionRepo: sessionRepo, messageRepo: messageRepo}
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "tester")
	resp, err := svc.CreateWorkspaceHandoff(ctx, &dto.WorkspaceHandoffReq{
		SourceSessionID: "source-session",
		FullCodePath:    "/liubeiluo/demo",
		TargetRole:      WorkspaceRoleAppDeveloper,
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
	if resp.MessageID == 0 {
		t.Fatal("expected handoff response to include initial message id")
	}
	if resp.HandoffPacketID == 0 {
		t.Fatal("expected handoff response to include handoff packet id")
	}
	if !strings.Contains(resp.Content, "target_role 固定为 app_developer") {
		t.Fatalf("content should include target role, got %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "tables.search_fields 是查询请求字段") {
		t.Fatalf("content should include PRD v2 search field handoff rule, got %q", resp.Content)
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
	if archived.Status != model.ChatSessionStatusDone {
		t.Fatalf("source status = %q, want %q", archived.Status, model.ChatSessionStatusDone)
	}
	target, err := sessionRepo.GetBySessionID(resp.SessionID)
	if err != nil {
		t.Fatalf("get target: %v", err)
	}
	if target.ParentSessionID != source.SessionID || target.HandoffKind != "agent_app_prd" || target.HandoffTargetRole != WorkspaceRoleAppDeveloper {
		t.Fatalf("target handoff metadata wrong: %#v", target)
	}
	if target.RoleID != WorkspaceRoleAppDeveloper || target.RoleDisplayName != "应用开发工程师" {
		t.Fatalf("target role metadata wrong: %#v", target)
	}
	messages, err := messageRepo.ListBySessionID(resp.SessionID)
	if err != nil {
		t.Fatalf("list target messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected one initial handoff message, got %d", len(messages))
	}
	if messages[0].ID != resp.MessageID || messages[0].Role != RoleUser || messages[0].ContextUsage != MessageContextArtifact || messages[0].ArtifactKind != "agent_app_prd" {
		t.Fatalf("initial message metadata wrong: %#v", messages[0])
	}
	if !strings.Contains(messages[0].Content, `"kind": "agent_app_prd"`) || !strings.Contains(messages[0].DisplayContent, "优先做列表") {
		t.Fatalf("initial handoff message content wrong: %#v", messages[0])
	}
	packet, err := handoffRepo.GetByTargetSessionID(resp.SessionID)
	if err != nil {
		t.Fatalf("get handoff packet: %v", err)
	}
	if packet.ID != resp.HandoffPacketID || packet.SourceSessionID != source.SessionID || packet.TargetSessionID != resp.SessionID || packet.InitialMessageID != resp.MessageID {
		t.Fatalf("handoff packet metadata wrong: %#v", packet)
	}
	if packet.TargetRole != WorkspaceRoleAppDeveloper || packet.ArtifactKind != "agent_app_prd" || !strings.Contains(packet.ArtifactJSON, `"project"`) {
		t.Fatalf("handoff packet payload wrong: %#v", packet)
	}
}

func TestPersistWorkspaceSessionInteractionStatusMarksPending(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sessionRepo := repository.NewChatSessionRepository(db)
	session := &model.AgentChatSession{
		TreeID:        7,
		FullCodePath:  "/liubeiluo/demo",
		Source:        SourceWorkspace,
		SessionID:     "pending-session",
		Title:         "PRD 讨论",
		ModeCode:      "dev",
		Status:        model.ChatSessionStatusGenerating,
		ContextPolicy: ContextPolicyFull,
		User:          "tester",
	}
	if err := sessionRepo.Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	svc := &WorkspaceChatService{sessionRepo: sessionRepo}
	svc.persistWorkspaceSessionInteractionStatus(context.Background(), "pending-session", []streamloop.ToolCallSummary{
		{
			Name:   "write_prd",
			Status: ToolCallStatusOK,
			ResultData: map[string]interface{}{
				"kind": "agent_app_prd",
				"interaction": map[string]interface{}{
					"status": model.ChatSessionStatusPendingConfirmation,
				},
			},
		},
	}, "tester")

	updated, err := sessionRepo.GetBySessionID("pending-session")
	if err != nil {
		t.Fatalf("get updated session: %v", err)
	}
	if updated.Status != model.ChatSessionStatusPendingConfirmation {
		t.Fatalf("status = %q, want %q", updated.Status, model.ChatSessionStatusPendingConfirmation)
	}
}

func TestPersistWorkspaceSessionInteractionStatusMarksOutput(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sessionRepo := repository.NewChatSessionRepository(db)
	session := &model.AgentChatSession{
		TreeID:        7,
		FullCodePath:  "/liubeiluo/demo",
		Source:        SourceWorkspace,
		SessionID:     "output-session",
		Title:         "生成代码",
		ModeCode:      "dev",
		Status:        model.ChatSessionStatusGenerating,
		ContextPolicy: ContextPolicyFull,
		User:          "tester",
	}
	if err := sessionRepo.Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	svc := &WorkspaceChatService{sessionRepo: sessionRepo}
	svc.persistWorkspaceSessionInteractionStatus(context.Background(), "output-session", []streamloop.ToolCallSummary{
		{Name: "write_go_file", Status: ToolCallStatusOK},
	}, "tester")

	updated, err := sessionRepo.GetBySessionID("output-session")
	if err != nil {
		t.Fatalf("get updated session: %v", err)
	}
	if updated.Status != model.ChatSessionStatusOutput {
		t.Fatalf("status = %q, want %q", updated.Status, model.ChatSessionStatusOutput)
	}
}

func TestResolveWorkspacePendingInteractionClearsPending(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sessionRepo := repository.NewChatSessionRepository(db)
	session := &model.AgentChatSession{
		TreeID:        7,
		FullCodePath:  "/liubeiluo/demo",
		Source:        SourceWorkspace,
		SessionID:     "pending-test-session",
		Title:         "构建结果",
		ModeCode:      "dev",
		Status:        model.ChatSessionStatusPendingTest,
		ContextPolicy: ContextPolicyFull,
		User:          "tester",
	}
	if err := sessionRepo.Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	svc := &WorkspaceChatService{sessionRepo: sessionRepo}
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "tester")
	if err := svc.ResolveWorkspacePendingInteraction(ctx, "pending-test-session"); err != nil {
		t.Fatalf("resolve pending interaction: %v", err)
	}

	updated, err := sessionRepo.GetBySessionID("pending-test-session")
	if err != nil {
		t.Fatalf("get updated session: %v", err)
	}
	if updated.Status != model.ChatSessionStatusActive {
		t.Fatalf("status = %q, want %q", updated.Status, model.ChatSessionStatusActive)
	}
}

func TestBuildWorkspaceHandoffContentForQA(t *testing.T) {
	got := buildWorkspaceHandoffContent(workspaceHandoffContentInput{
		TargetRole:    WorkspaceRoleQAEngineer,
		ArtifactKind:  workspaceBuildArtifactKind,
		ArtifactJSON:  `{"kind":"agent_app_build","workspace_path":"/liubeiluo/nps","new_version":"v4"}`,
		ContextPolicy: ContextPolicyArtifactOnly,
	})
	for _, want := range []string{
		"target_role 固定为 qa_engineer",
		"测试阶段要求",
		"search_tools/read_dir",
		"创建开始时间/创建结束时间",
		`"kind":"agent_app_build"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("content should include %q, got %q", want, got)
		}
	}
}

func TestExecuteToolCallsPersistsRoleAfterChangeRole(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentChatSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := createSQLiteAgentChatMessagesTable(db); err != nil {
		t.Fatalf("migrate messages: %v", err)
	}
	sessionRepo := repository.NewChatSessionRepository(db)
	messageRepo := repository.NewChatMessageRepository(db)
	session := &model.AgentChatSession{
		TreeID:        7,
		FullCodePath:  "/liubeiluo/demo",
		Source:        SourceWorkspace,
		SessionID:     "role-session",
		Title:         "角色切换",
		ModeCode:      "dev",
		Status:        model.ChatSessionStatusGenerating,
		ContextPolicy: ContextPolicyFull,
		User:          "tester",
	}
	if err := sessionRepo.Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	svc := &WorkspaceChatService{
		toolReg:     NewToolRegistry(),
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
	call := llms.ToolCall{ID: "call-change-role", Type: "function"}
	call.Function.Name = "change_role"
	call.Function.Arguments = `{"target_role":"product_manager","user_input":"帮我做个系统"}`

	summaries, err := svc.executeToolCalls(context.Background(), []llms.ToolCall{call}, "role-session", "/liubeiluo/demo", "tester", "", func(string, interface{}) {})
	if err != nil {
		t.Fatalf("execute tool calls: %v", err)
	}
	if len(summaries) != 1 || summaries[0].Status != ToolCallStatusOK {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}
	updated, err := sessionRepo.GetBySessionID("role-session")
	if err != nil {
		t.Fatalf("get updated session: %v", err)
	}
	if updated.RoleID != WorkspaceRoleProductManager || updated.RoleDisplayName != "产品经理" {
		t.Fatalf("session role not persisted: %#v", updated)
	}
}

func createSQLiteAgentChatMessagesTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE agent_chat_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at datetime,
	updated_at datetime,
	deleted_at datetime,
	created_by text,
	updated_by text,
	deleted_by text,
	session_id text NOT NULL,
	role text NOT NULL,
	content text,
	display_content text,
	files text,
	tool_calls text,
	tool_call_id text,
	tool_status text,
	result_data text,
	result_metadata text,
	llm_config_id integer,
	llm_config_name text,
	llm_provider text,
	llm_model text,
	context_usage text DEFAULT 'include',
	artifact_kind text,
	user text NOT NULL
)`).Error
}
