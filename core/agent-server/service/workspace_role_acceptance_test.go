package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	workspaceroles "github.com/kageos/kageos/core/agent-server/workspace/roles"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWorkspaceRoleAcceptanceExistingVoteRequestPrefersAppOperator(t *testing.T) {
	workspaceCtx := &dto.GetWorkspaceContextResp{
		User: "tester",
		Directory: dto.WorkspaceContextDirectory{
			Name:         "投票系统",
			Code:         "vote",
			FullCodePath: "/system/x_world/vote",
			Description:  "用于创建投票主题、提交投票并查看统计结果的现有应用",
			Type:         "package",
		},
		Children: []dto.WorkspaceContextNode{
			{
				Name:         "投票主题",
				Code:         "vote_topic_list",
				Type:         "function",
				TemplateType: "table",
				FullCodePath: "/system/x_world/vote/vote_topic_list.table",
				Description:  "新增和查询投票主题、选项、单选/多选和状态",
			},
			{
				Name:         "提交投票",
				Code:         "submit_vote",
				Type:         "function",
				TemplateType: "form",
				FullCodePath: "/system/x_world/vote/submit_vote.form",
				Description:  "用户选择投票主题和选项后提交投票记录",
			},
		},
	}

	hint := workspaceFirstTurnDirectoryRAGHint([]*model.AgentChatMessage{
		{Role: RoleUser, Content: "创建一个四大古都投票，北京南京西安洛阳单选", User: "tester"},
	}, workspaceCtx)
	for _, want := range []string{
		"首轮目录理解要求",
		"当前软件",
		"app_operator",
		"不要先写 PRD 或进入开发",
	} {
		if !strings.Contains(hint, want) {
			t.Fatalf("first-turn directory hint should contain %q, got:\n%s", want, hint)
		}
	}

	routing := workspaceroles.RoutingMarkdown()
	appOperatorIndex := strings.Index(routing, "### `app_operator`")
	productManagerIndex := strings.Index(routing, "### `product_manager`")
	if appOperatorIndex < 0 || productManagerIndex < 0 || appOperatorIndex > productManagerIndex {
		t.Fatalf("routing should evaluate app_operator before product_manager, got:\n%s", routing)
	}
	operatorSpec, _ := workspaceroles.SpecFor(workspaceroles.AppOperator)
	productSpec, _ := workspaceroles.SpecFor(workspaceroles.ProductManager)
	if !strings.Contains(operatorSpec.RouteDescription, "新增投票主题和选项") ||
		!strings.Contains(operatorSpec.RouteDescription, "优先于 `product_manager`") {
		t.Fatalf("app_operator route should explicitly cover voting business operations, got: %s", operatorSpec.RouteDescription)
	}
	if !strings.Contains(productSpec.RouteDescription, "/system/x_world/vote") ||
		!strings.Contains(productSpec.RouteDescription, "“创建一个投票”是业务操作") {
		t.Fatalf("product_manager route should exclude existing vote-app business operation, got: %s", productSpec.RouteDescription)
	}
}

func TestWorkspaceRoleAcceptancePRDConfirmHandoffCarriesDeveloperPacket(t *testing.T) {
	const votePRD = `{
  "kind": "agent_app_prd",
  "project": {
    "name": "投票系统",
    "code": "vote",
    "summary": "创建投票主题、配置候选项、收集用户投票并统计结果"
  },
  "tables": [
    {
      "name": "投票主题",
      "code": "vote_topic",
      "fields": [{"name": "主题标题"}, {"name": "投票类型"}, {"name": "候选项"}, {"name": "状态"}],
      "search_fields": [{"name": "主题标题"}, {"name": "状态"}, {"name": "创建开始时间"}, {"name": "创建结束时间"}],
      "handlers": ["OnTableAddRow", "OnTableUpdateRow", "OnTableDeleteRow"]
    },
    {
      "name": "投票记录",
      "code": "vote_record",
      "fields": [{"name": "投票主题"}, {"name": "投票选项"}, {"name": "投票用户"}],
      "search_fields": [{"name": "投票主题"}, {"name": "投票用户"}],
      "handlers": []
    }
  ],
  "forms": [
    {
      "name": "提交投票",
      "code": "submit_vote",
      "target_table": "投票记录",
      "request_fields": [{"name": "投票主题"}, {"name": "投票选项"}],
      "response_fields": [{"name": "提交结果"}]
    }
  ],
  "charts": [
    {
      "name": "投票结果统计",
      "code": "vote_result_chart",
      "source_table": "投票记录",
      "chart_type": "bar",
      "dimension": "投票选项",
      "metrics": ["票数"],
      "filters": ["投票主题"]
    }
  ],
  "rules": ["每个用户在同一投票主题下只能投一次", "投票主题关闭后不允许提交"]
}`

	source := &model.AgentChatSession{
		SessionID:       "prd-session",
		Title:           "投票系统 PRD",
		RoleID:          WorkspaceRoleProductManager,
		RoleDisplayName: "产品经理",
	}
	ctx := buildWorkspaceHandoffContext(workspaceHandoffContextInput{
		Source:        source,
		FullCodePath:  "/system/x_world",
		TargetRole:    WorkspaceRoleAppDeveloper,
		ArtifactKind:  "agent_app_prd",
		ArtifactJSON:  votePRD,
		ContextPolicy: ContextPolicyArtifactOnly,
		Messages: []*model.AgentChatMessage{
			{Role: RoleUser, Content: "我要一个投票系统，可以创建主题、配置候选项、收集投票并看统计", User: "tester"},
		},
	})

	packet := ctx.HandoffPacket
	if packet == nil {
		t.Fatal("confirmed PRD handoff should expose a typed handoff packet")
	}
	if packet.SourceSessionID != "prd-session" ||
		packet.SourceRole != WorkspaceRoleProductManager ||
		packet.TargetRole != WorkspaceRoleAppDeveloper ||
		packet.ExecuteDirectory != "/system/x_world" ||
		packet.TargetAppDirectory != "/system/x_world/vote" {
		t.Fatalf("developer handoff packet metadata is incomplete: %#v", packet)
	}
	if packet.Artifact == nil || !packet.Artifact.Included || packet.Artifact.Kind != "agent_app_prd" || packet.Artifact.Source != "AGENT_APP_PRD JSON" {
		t.Fatalf("developer packet should reference the full PRD artifact block, got %#v", packet.Artifact)
	}
	if packet.ArtifactDigest == nil ||
		packet.ArtifactDigest.ProjectCode != "vote" ||
		len(packet.ArtifactDigest.Tables) != 2 ||
		len(packet.ArtifactDigest.Forms) != 1 ||
		len(packet.ArtifactDigest.Charts) != 1 {
		t.Fatalf("developer packet should carry structured PRD digest, got %#v", packet.ArtifactDigest)
	}
	keyInfo := strings.Join(packet.KeyInformation, "；")
	for _, want := range []string{
		"主执行目录/绑定目录：/system/x_world",
		"目标应用目录：/system/x_world/vote",
		"实现重点",
		"Table=投票主题、投票记录",
	} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("developer packet key_information should contain %q, got %#v", want, packet.KeyInformation)
		}
	}
	for _, want := range []string{
		"AGENT_APP_PRD JSON（本消息完整产物块）",
		"/system/prompt/roles/app-developer",
		"/system/prompt/sdk/agent-app-sdk-readme",
		"/system/prompt/case_catalog",
		"/system/x_world",
		"/system/x_world/vote",
	} {
		if !containsWorkspaceRoleString(packet.References, want) {
			t.Fatalf("developer packet references should include %q, got %#v", want, packet.References)
		}
	}
	if packet.Validation.Status == "error" {
		t.Fatalf("developer handoff packet should validate, got %#v", packet.Validation)
	}
	if !containsWorkspaceRoleHook(ctx.ExecutedHooks, workspaceRoleHookProductManagerToDeveloper) {
		t.Fatalf("PRD confirm handoff should execute product-manager hook, got %#v", ctx.ExecutedHooks)
	}
	if !strings.Contains(ctx.PRDExecutionMarkdown, "## Table：投票主题") ||
		!strings.Contains(ctx.PRDExecutionMarkdown, "每个用户在同一投票主题下只能投一次") {
		t.Fatalf("developer handoff should carry PRD execution markdown, got:\n%s", ctx.PRDExecutionMarkdown)
	}

	content := buildWorkspaceHandoffContent(workspaceHandoffContentInput{
		ArtifactKind:         "agent_app_prd",
		ArtifactJSON:         votePRD,
		HandoffPacketJSON:    formatWorkspaceRoleHandoffPacketJSON(packet),
		HandoffContextJSON:   formatWorkspaceHandoffContextJSON(workspaceHandoffContextForMessage(ctx)),
		PRDExecutionMarkdown: ctx.PRDExecutionMarkdown,
		ExecuteDirectory:     ctx.ExecuteDirectory,
		TargetRole:           WorkspaceRoleAppDeveloper,
		ContextPolicy:        ContextPolicyArtifactOnly,
	})
	for _, want := range []string{
		"target_role 固定为 app_developer",
		"change_role.execute_directory 必须固定为 /system/x_world",
		"HANDOFF_PACKET JSON",
		"PRD_EXECUTION_MARKDOWN",
		"不要重新输出 PRD",
		"tables.fields 是业务模型字段",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("developer handoff content should contain %q, got:\n%s", want, content)
		}
	}
}

func TestWorkspaceRoleAcceptanceBuildFailureEntersRepairFlowWithDiagnostics(t *testing.T) {
	const buildErr = `app startup failed: SDK schema compile failed: router /vote/vote_topic_list.table table schema decode failed
field Status (status): widget "select" requires options or OnSelectFuzzyMap entry
field CreatedBy (created_by): audit field "created_by" hide tag must be "create,update", got ""`

	res := (&ChangeRoleTool{}).Execute(context.Background(), ToolCall{
		FullCodePath: "/system/x_world/vote",
		Args: map[string]interface{}{
			"current_role":      WorkspaceRoleAppDeveloper,
			"target_role":       WorkspaceRoleBuildEngineer,
			"execute_directory": "/system/x_world/vote",
			"task_context": []string{
				"build_workspace 失败，不能交接测试",
				"用户目标是完成投票系统开发后的构建修复",
			},
			"key_information": []string{
				"完整构建错误：" + buildErr,
			},
			"references": []string{
				"/system/prompt/sdk/reference/build-validation",
				"/system/prompt/sdk/agent-app-sdk-readme",
				"/system/x_world/vote/vote_topic.go",
			},
			"reset_context": true,
		},
	})
	if res.IsError {
		t.Fatalf("change_role to build_engineer should succeed, got: %s", res.Content)
	}
	data, ok := res.Data.(changeRoleData)
	if !ok {
		t.Fatalf("change_role data type = %T, want changeRoleData", res.Data)
	}
	diagnostics := data.BuildDiagnostics
	if diagnostics == nil || diagnostics.Status != "error" {
		t.Fatalf("build_engineer handoff should include diagnostics, got %#v", diagnostics)
	}
	for _, want := range []string{"schema_validation", "select_options", "audit_field"} {
		if !containsWorkspaceRoleString(diagnostics.Categories, want) {
			t.Fatalf("diagnostics should include category %q, got %#v", want, diagnostics.Categories)
		}
	}
	if !containsWorkspaceRoleString(diagnostics.Routers, "/vote/vote_topic_list.table") {
		t.Fatalf("diagnostics should include failing router, got %#v", diagnostics.Routers)
	}
	if len(diagnostics.FieldIssues) < 2 {
		t.Fatalf("diagnostics should preserve field-level issues, got %#v", diagnostics.FieldIssues)
	}
	for _, want := range []string{
		"/system/prompt/sdk/reference/build-validation",
		"/system/prompt/sdk/agent-app-sdk-readme",
		"/system/prompt/case_catalog",
	} {
		if !containsWorkspaceRoleString(diagnostics.RequiredDocs, want) {
			t.Fatalf("diagnostics required_docs should include %q, got %#v", want, diagnostics.RequiredDocs)
		}
		if !containsWorkspaceRoleString(data.HandoffPacket.References, want) {
			t.Fatalf("handoff packet references should include diagnostic doc %q, got %#v", want, data.HandoffPacket.References)
		}
	}
	if data.HandoffPacket.BuildDiagnostics == nil {
		t.Fatalf("build_engineer packet should carry build_diagnostics, got %#v", data.HandoffPacket)
	}
	if !containsWorkspaceRoleHook(data.ExecutedHooks, workspaceRoleHookBuildEngineerDiagnostics) {
		t.Fatalf("build_engineer before_enter diagnostics hook should run, got %#v", data.ExecutedHooks)
	}
	keyInfo := strings.Join(data.HandoffPacket.KeyInformation, "；")
	for _, want := range []string{"构建诊断", "构建修复必读资料", "构建修复策略", "不要继续同一方案重试"} {
		if !strings.Contains(keyInfo, want) {
			t.Fatalf("build repair packet key_information should contain %q, got %#v", want, data.HandoffPacket.KeyInformation)
		}
	}

	failureCtx := buildWorkspaceHandoffContext(workspaceHandoffContextInput{
		FullCodePath:  "/system/x_world/vote",
		TargetRole:    WorkspaceRoleBuildEngineer,
		ArtifactKind:  workspaceBuildFailureKind,
		ArtifactJSON:  `{"kind":"agent_app_build_failure","workspace_path":"/system/x_world/vote","error":` + quoteJSON(buildErr) + `}`,
		ContextPolicy: ContextPolicyArtifactOnly,
	})
	failureContent := buildWorkspaceHandoffContent(workspaceHandoffContentInput{
		ArtifactKind:       workspaceBuildFailureKind,
		ArtifactJSON:       `{"kind":"agent_app_build_failure"}`,
		HandoffPacketJSON:  formatWorkspaceRoleHandoffPacketJSON(failureCtx.HandoffPacket),
		HandoffContextJSON: formatWorkspaceHandoffContextJSON(workspaceHandoffContextForMessage(failureCtx)),
		ExecuteDirectory:   failureCtx.ExecuteDirectory,
		TargetRole:         WorkspaceRoleBuildEngineer,
		ContextPolicy:      ContextPolicyArtifactOnly,
	})
	for _, want := range []string{
		"构建修复阶段要求",
		"不要继续同一方案反复重写",
		"先读 /system/prompt/sdk/reference/build-validation",
		"build_diagnostics",
	} {
		if !strings.Contains(failureContent, want) {
			t.Fatalf("build failure handoff content should contain %q, got:\n%s", want, failureContent)
		}
	}
}

func TestWorkspaceRoleAcceptanceArchivedHistoryExcludedFromModelContext(t *testing.T) {
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
	session := &model.AgentChatSession{
		TreeID:          42,
		FullCodePath:    "/system/x_world/vote",
		Source:          SourceWorkspace,
		SessionID:       "acceptance-archive-session",
		Title:           "投票系统开发",
		ModeCode:        "dev",
		Status:          model.ChatSessionStatusGenerating,
		RoleID:          WorkspaceRoleAppDeveloper,
		RoleDisplayName: "应用开发工程师",
		ContextPolicy:   ContextPolicyArtifactOnly,
		ParentSessionID: "prd-source-session",
		User:            "tester",
	}
	if err := sessionRepo.Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}
	oldMsg := &model.AgentChatMessage{
		SessionID: session.SessionID,
		Role:      RoleUser,
		Content:   "旧会话关键约束：保留投票主题、提交投票、统计结果三块能力",
		User:      "tester",
	}
	if err := messageRepo.Create(oldMsg); err != nil {
		t.Fatalf("create old message: %v", err)
	}
	session.ModelContextAnchorMessageID = oldMsg.ID
	if err := sessionRepo.Update(session); err != nil {
		t.Fatalf("update anchor: %v", err)
	}
	packet := workspaceRoleHandoffPacket{
		Version:            workspaceRoleHandoffPacketVersion,
		SourceSessionID:    "prd-source-session",
		SourceRole:         WorkspaceRoleProductManager,
		TargetRole:         WorkspaceRoleAppDeveloper,
		ArtifactKind:       "agent_app_prd",
		ExecuteDirectory:   "/system/x_world",
		WorkspaceDirectory: "/system/x_world",
		TargetAppDirectory: "/system/x_world/vote",
		TaskContext:        []string{"投票系统 PRD 已确认，进入开发阶段"},
		KeyInformation:     []string{"目标应用目录：/system/x_world/vote", "实现重点：投票主题、提交投票、统计结果"},
		References:         []string{"AGENT_APP_PRD JSON（本消息完整产物块）", "/system/prompt/roles/app-developer", "/system/prompt/sdk/agent-app-sdk-readme"},
		ContextPolicy:      ContextPolicyArtifactOnly,
		Artifact:           &workspaceRolePacketArtifact{Kind: "agent_app_prd", Included: true, Source: "AGENT_APP_PRD JSON"},
	}
	normalizeAndValidateWorkspaceRoleHandoffPacket(&packet)
	handoffMsg := &model.AgentChatMessage{
		SessionID:    session.SessionID,
		Role:         RoleUser,
		Content:      "HANDOFF_PACKET JSON:\n```json\n" + formatWorkspaceRoleHandoffPacketJSON(&packet) + "\n```",
		ContextUsage: MessageContextArtifact,
		ArtifactKind: "agent_app_prd",
		User:         "tester",
	}
	if err := messageRepo.Create(handoffMsg); err != nil {
		t.Fatalf("create handoff message: %v", err)
	}
	displayOnlyMsg := &model.AgentChatMessage{
		SessionID:    session.SessionID,
		Role:         RoleUser,
		Content:      "旧会话展示卡片：用户仍可看到，也要作为历史背景保留",
		ContextUsage: MessageContextDisplayOnly,
		User:         "tester",
	}
	if err := messageRepo.Create(displayOnlyMsg); err != nil {
		t.Fatalf("create display-only message: %v", err)
	}
	currentMsg := &model.AgentChatMessage{
		SessionID: session.SessionID,
		Role:      RoleUser,
		Content:   "按确认后的 PRD 开始开发投票系统",
		User:      "tester",
	}
	if err := messageRepo.Create(currentMsg); err != nil {
		t.Fatalf("create current message: %v", err)
	}

	svc := &WorkspaceChatService{
		toolReg:     NewToolRegistry(),
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
	workspaceCtx := &dto.GetWorkspaceContextResp{
		Directory: dto.WorkspaceContextDirectory{
			Name:         "投票系统",
			Code:         "vote",
			FullCodePath: "/system/x_world/vote",
			Type:         "package",
		},
	}
	msgs, _, plan, err := svc.buildLLMMessagesWithPlan(context.Background(), session.SessionID, "/system/x_world/vote", "投票系统", workspaceCtx, nil, []string{"read_doc", "read_dir"}, "fallback", 1)
	if err != nil {
		t.Fatalf("build llm messages: %v", err)
	}
	joined := joinLLMMessageContents(msgs)
	for _, required := range []string{
		"旧会话关键约束",
		"保留投票主题、提交投票、统计结果三块能力",
		"旧会话展示卡片",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("historical content should remain in model context; missing %q in:\n%s", required, joined)
		}
	}
	if !strings.Contains(joined, "HANDOFF_PACKET JSON") ||
		!strings.Contains(joined, "按确认后的 PRD 开始开发投票系统") {
		t.Fatalf("model context should include handoff packet and current user request, got:\n%s", joined)
	}
	if plan == nil {
		t.Fatal("model context plan is nil")
	}
	if plan.Messages.ExcludedByAnchor != 0 ||
		plan.Messages.ExcludedDisplayOnly != 0 ||
		plan.Messages.SourceHistoryPolicy != "same_session_full_with_parent_reference" {
		t.Fatalf("model context plan should preserve legacy anchor/display-tagged history, got %#v", plan.Messages)
	}
	if plan.Handoff == nil ||
		plan.Handoff.TargetRole != WorkspaceRoleAppDeveloper ||
		plan.Handoff.ExecuteDirectory != "/system/x_world" {
		t.Fatalf("model context plan should expose the typed handoff packet, got %#v", plan.Handoff)
	}
}

func containsWorkspaceRoleHook(items []workspaceExecutedRoleHook, hookID string) bool {
	for _, item := range items {
		if item.ID == hookID {
			return true
		}
	}
	return false
}

func joinLLMMessageContents(messages []llms.Message) string {
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		parts = append(parts, msg.Content)
	}
	return strings.Join(parts, "\n")
}

func workspaceContextPlanHasExcludedReason(refs []dto.WorkspaceModelContextMessageRef, id int64, reason string) bool {
	for _, ref := range refs {
		if ref.ID == id && ref.Reason == reason {
			return true
		}
	}
	return false
}

func quoteJSON(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return `"` + value + `"`
}
