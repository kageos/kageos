package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	agentosskills "github.com/ai-agent-os/ai-agent-os/core/agent-server/skills"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
)

func TestWorkspaceSkillToolGateResult(t *testing.T) {
	registry := agentosskills.LoadEmbedded()
	createSkill, ok := registry.Get("sop.create-project")
	if !ok {
		t.Fatal("expected sop.create-project")
	}
	comboSkill, ok := registry.Get("sdk.combo-table-form-chart")
	if !ok {
		t.Fatal("expected sdk.combo-table-form-chart")
	}
	executeSkill, ok := registry.Get("sop.execute-function")
	if !ok {
		t.Fatal("expected sop.execute-function")
	}
	runtimeSkill, ok := registry.Get("system.tools.runtime")
	if !ok {
		t.Fatal("expected system.tools.runtime")
	}
	messageSkill, ok := registry.Get("system.openapi.message")
	if !ok {
		t.Fatal("expected system.openapi.message")
	}
	loaded := map[string]*agentosskills.Skill{createSkill.Meta.ID: createSkill}
	loadedWithSDKPackage := map[string]*agentosskills.Skill{
		createSkill.Meta.ID: createSkill,
		comboSkill.Meta.ID:  comboSkill,
	}
	loadedDocs := loadedDocsForSkill(createSkill)
	loadedDocsWithSDKPackage := loadedDocsForSkill(createSkill)
	for doc := range loadedDocsForSkill(comboSkill) {
		loadedDocsWithSDKPackage[doc] = struct{}{}
	}
	executeLoadedDocs := loadedDocsForSkill(executeSkill)
	messageLoadedDocs := loadedDocsForSkill(messageSkill)

	if _, blocked := workspaceSkillToolGateResult("web_search", map[string]interface{}{"keyword": "Claude source leak"}, nil, nil); blocked {
		t.Fatal("skills should not hard-block ordinary web_search")
	}
	if _, blocked := workspaceSkillToolGateResult("read_doc", map[string]interface{}{"directory": "/system/prompt/case_catalog/table/ticket"}, nil, nil); blocked {
		t.Fatal("skills should not hard-block ordinary reference docs when no skill is loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("read_doc", map[string]interface{}{"directory": "/system/prompt/workspace/create-project"}, nil, nil); !blocked {
		t.Fatal("skills should block legacy workspace SOP docs")
	}
	if _, blocked := workspaceSkillToolGateResult("read_doc", map[string]interface{}{"directory": "/system/prompt/platform-capability-boundaries"}, nil, nil); !blocked {
		t.Fatal("skills should block direct platform guide docs until a matching skill is loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("read_dir", nil, nil, nil); blocked {
		t.Fatal("skills should not hard-block read_dir when no skill is loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("run_table_search", nil, nil, nil); !blocked {
		t.Fatal("skills should block execution tools until a matching skill is loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("run_table_search", nil, loaded, nil); !blocked {
		t.Fatal("create-project skill should not authorize validation tools before required_docs are loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("run_table_search", nil, loaded, loadedDocs); blocked {
		t.Fatal("create-project skill should authorize run_table_search after required_docs are loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("read_dir", nil, loaded, loadedDocs); blocked {
		t.Fatal("create-project skill should authorize read_dir because it is in allowed_tools")
	}
	if res, blocked := workspaceSkillToolGateResult("web_search", nil, loaded, loadedDocs); !blocked {
		t.Fatal("loaded skill should block tools outside allowed_tools")
	} else if !strings.Contains(res.Content, "allowed_tools") || !strings.Contains(res.Content, "提示词以外") {
		t.Fatalf("expected allowed_tools hard boundary, got %q", res.Content)
	}
	if _, blocked := workspaceSkillToolGateResult("run_table_search", nil, map[string]*agentosskills.Skill{executeSkill.Meta.ID: executeSkill}, executeLoadedDocs); blocked {
		t.Fatal("execute-function skill should authorize run_table_search")
	}
	if _, blocked := workspaceSkillToolGateResult("run_form_submit", nil, map[string]*agentosskills.Skill{messageSkill.Meta.ID: messageSkill}, messageLoadedDocs); blocked {
		t.Fatal("system.openapi.message should authorize run_form_submit")
	}
	if _, blocked := workspaceSkillToolGateResult("build_workspace", nil, nil, nil); !blocked {
		t.Fatal("skills should block build_workspace until a create/modify/build skill is loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("build_workspace", nil, loaded, nil); !blocked {
		t.Fatal("create-project skill should not authorize build_workspace before required_docs are loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("read_dir", nil, loaded, nil); !blocked {
		t.Fatal("skills should block read_dir after skill is loaded until required_docs are loaded")
	}
	firstRequiredDoc := createSkill.Meta.RequiredDocs[0]
	if _, blocked := workspaceSkillToolGateResult("read_doc", map[string]interface{}{"directory": firstRequiredDoc}, loaded, nil); blocked {
		t.Fatal("skills should not hard-block read_doc for required_docs")
	}
	if _, blocked := workspaceSkillToolGateResult("read_doc", map[string]interface{}{"directory": "/system/prompt/case_catalog/table/ticket"}, loaded, nil); !blocked {
		t.Fatal("skills should block unrelated read_doc until required_docs are loaded")
	}
	if res, blocked := workspaceSkillToolGateResult("build_workspace", nil, loaded, loadedDocs); !blocked {
		t.Fatal("create-project should require a concrete sdk task package before build_workspace")
	} else if !strings.Contains(res.Content, "具体 SDK 任务包") || !strings.Contains(res.Content, "sdk.combo-table-form-chart") {
		t.Fatalf("expected sdk task package guidance, got %q", res.Content)
	}
	if _, blocked := workspaceSkillToolGateResult("build_workspace", nil, loadedWithSDKPackage, loadedDocsWithSDKPackage); blocked {
		t.Fatal("skills should not hard-block build_workspace after create skill, required docs, and sdk task package are loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("write_go_file", nil, loadedWithSDKPackage, loadedDocsWithSDKPackage); blocked {
		t.Fatal("skills should not hard-block write_go_file after create skill, required docs, and sdk task package are loaded")
	}
	if _, blocked := workspaceSkillToolGateResult("read_skill", nil, loaded, nil); blocked {
		t.Fatal("read_skill should always be allowed")
	}
	if _, blocked := workspaceSkillToolGateResult("run_official_python", nil, loaded, loadedDocs); !blocked {
		t.Fatal("create-project skill should not authorize run_official_python")
	}
	if _, blocked := workspaceSkillToolGateResult("run_official_python", nil, map[string]*agentosskills.Skill{runtimeSkill.Meta.ID: runtimeSkill}, nil); blocked {
		t.Fatal("system.tools.runtime should authorize run_official_python")
	}
}

func TestWorkspaceSkillGateForExplainProjectFlow(t *testing.T) {
	registry := agentosskills.LoadEmbedded()
	explainSkill, ok := registry.Get("sop.explain-project")
	if !ok {
		t.Fatal("expected sop.explain-project")
	}
	loaded := map[string]*agentosskills.Skill{explainSkill.Meta.ID: explainSkill}
	if len(explainSkill.Meta.RequiredDocs) != 0 {
		t.Fatalf("explain-project should not depend on legacy docs, got %#v", explainSkill.Meta.RequiredDocs)
	}

	if _, blocked := workspaceSkillToolGateResult("read_doc", map[string]interface{}{"directory": "/system/prompt/workspace/explain-project"}, loaded, nil); !blocked {
		t.Fatal("legacy explain-project doc should stay blocked even after skill is loaded")
	}
	loadedDocs := loadedDocsForSkill(explainSkill)
	if _, blocked := workspaceSkillToolGateResult("read_dir", map[string]interface{}{"recursive": true}, loaded, loadedDocs); blocked {
		t.Fatal("read_dir should be allowed after explain-project skill and required doc are loaded")
	}
}

func TestWorkspaceSkillGateGuidesSingleNextStep(t *testing.T) {
	res, blocked := workspaceSkillToolGateResult("run_table_search", nil, nil, nil)
	if !blocked {
		t.Fatal("expected run_table_search to be blocked")
	}
	if !strings.Contains(res.Content, "下一步只调用") {
		t.Fatalf("expected gate message to require one next step, got %q", res.Content)
	}
	if !strings.Contains(res.Content, "read_skill(\"sop.execute-function\")") {
		t.Fatalf("expected execute skill hint, got %q", res.Content)
	}
	if !strings.Contains(res.Content, "不要继续调用") {
		t.Fatalf("expected repeated execution calls to be discouraged, got %q", res.Content)
	}
}

func TestWriteMutationBatchGuards(t *testing.T) {
	if shouldSkipAfterWriteMutationFailure("read_go_file") {
		t.Fatal("read_go_file should still be allowed after a write failure")
	}
	for _, toolName := range []string{"write_go_file", "search_replace_file", "build_workspace"} {
		if !shouldSkipAfterWriteMutationFailure(toolName) {
			t.Fatalf("%s should be skipped after a write mutation failure", toolName)
		}
		res := writeMutationFailureBatchSkipResult(toolName)
		for _, want := range []string{"已有写入/替换/删除/创建目录工具失败", "未落盘", "不继续执行"} {
			if !strings.Contains(res.Content, want) {
				t.Fatalf("expected %q in %q", want, res.Content)
			}
		}
	}
	if shouldGateWriteAfterBatchLimit("write_go_file", maxWriteGoFilesPerToolBatch-1) {
		t.Fatal("should allow writes below staged write limit")
	}
	if !shouldGateWriteAfterBatchLimit("write_go_file", maxWriteGoFilesPerToolBatch) {
		t.Fatal("should gate write_go_file at staged write limit")
	}
	if shouldGateWriteAfterBatchLimit("build_workspace", maxWriteGoFilesPerToolBatch) {
		t.Fatal("should not gate build_workspace by write file count")
	}
	res := writeGoFileBatchLimitResult("write_go_file", maxWriteGoFilesPerToolBatch)
	for _, want := range []string{"达到分阶段上限", "先对当前阶段调用 `build_workspace`", "不继续执行"} {
		if !strings.Contains(res.Content, want) {
			t.Fatalf("expected %q in %q", want, res.Content)
		}
	}
}

func TestWorkspaceSkillGateRequiresRequiredDocs(t *testing.T) {
	registry := agentosskills.LoadEmbedded()
	createSkill, ok := registry.Get("sop.create-project")
	if !ok {
		t.Fatal("expected sop.create-project")
	}
	loaded := map[string]*agentosskills.Skill{createSkill.Meta.ID: createSkill}

	res, blocked := workspaceSkillToolGateResult("read_dir", nil, loaded, nil)
	if !blocked {
		t.Fatal("expected read_dir to be blocked until required_docs are loaded")
	}
	if !strings.Contains(res.Content, "required_docs") || !strings.Contains(res.Content, "read_doc") {
		t.Fatalf("expected missing docs guidance, got %q", res.Content)
	}
}

func TestWorkspaceSkillGateBatchSkipResult(t *testing.T) {
	if !isSkillBootstrapTool("read_skill") {
		t.Fatal("read_skill should be a skill bootstrap tool")
	}
	if shouldSkipAfterSkillGateBlock("read_skill") {
		t.Fatal("read_skill should still be allowed after a skill gate block")
	}
	if !shouldSkipAfterSkillGateBlock("web_search") {
		t.Fatal("web_search should be skipped after a skill gate block when a skill is active")
	}
	if !shouldSkipAfterSkillGateBlock("run_table_search") {
		t.Fatal("run_table_search should be skipped after a skill gate block in the same batch")
	}
	res := skillGateBatchSkipResult("run_table_search")
	if !res.IsError {
		t.Fatal("expected skip result to be an error tool result")
	}
	if !strings.Contains(res.Content, "已跳过工具调用") || !strings.Contains(res.Content, "本批次不继续执行") {
		t.Fatalf("expected concise batch skip message, got %q", res.Content)
	}
	if !strings.Contains(res.Content, "read_skill(\"sop.execute-function\")") {
		t.Fatalf("expected execute skill hint, got %q", res.Content)
	}
}

func loadedDocsForSkill(skill *agentosskills.Skill) map[string]struct{} {
	out := make(map[string]struct{})
	if skill == nil {
		return out
	}
	for _, doc := range skill.Meta.RequiredDocs {
		if doc = normalizeGuideDocPath(doc); doc != "" {
			out[doc] = struct{}{}
		}
	}
	return out
}

func TestLoadedSkillsFromMessagesPersistsAcrossUserTurns(t *testing.T) {
	registry := agentosskills.LoadEmbedded()
	messages := []*model.AgentChatMessage{
		{Role: RoleUser, Content: "帮我创建一个系统"},
		assistantReadSkillMessage("call-create", "sop.create-project"),
		{Role: RoleTool, ToolCallID: "call-create", ToolStatus: ToolCallStatusOK},
		{Role: RoleUser, Content: "现在帮我执行已有函数"},
		assistantReadSkillMessage("call-execute", "sop.execute-function"),
		{Role: RoleTool, ToolCallID: "call-execute", ToolStatus: ToolCallStatusError},
	}

	loaded := loadedSkillsFromMessages(context.Background(), messages, registry)
	if _, ok := loaded["sop.create-project"]; !ok {
		t.Fatalf("expected skill from previous user turn to remain active: %#v", loaded)
	}
	if _, ok := loaded["sop.execute-function"]; ok {
		t.Fatalf("failed read_skill should not activate execute skill: %#v", loaded)
	}

	messages = append(messages, &model.AgentChatMessage{Role: RoleTool, ToolCallID: "call-execute", ToolStatus: ToolCallStatusOK})
	loaded = loadedSkillsFromMessages(context.Background(), messages, registry)
	if _, ok := loaded["sop.execute-function"]; !ok {
		t.Fatalf("expected execute skill to be active: %#v", loaded)
	}
	if _, ok := loaded["sop.create-project"]; !ok {
		t.Fatalf("skill from previous user turn should remain active: %#v", loaded)
	}
}

func TestLoadedGuideDocsFromMessagesMarksReadSkillRequiredDocs(t *testing.T) {
	registry := agentosskills.LoadEmbedded()
	createSkill, ok := registry.Get("sop.create-project")
	if !ok {
		t.Fatal("expected sop.create-project")
	}
	messages := []*model.AgentChatMessage{
		{Role: RoleUser, Content: "帮我创建一个系统"},
		assistantReadSkillMessage("call-create", "sop.create-project"),
		{Role: RoleTool, ToolCallID: "call-create", ToolStatus: ToolCallStatusOK},
	}

	loaded := loadedGuideDocsFromMessages(context.Background(), messages, registry)
	for _, doc := range createSkill.Meta.RequiredDocs {
		doc = normalizeGuideDocPath(doc)
		if _, ok := loaded[doc]; !ok {
			t.Fatalf("expected required doc %s to be marked loaded: %#v", doc, loaded)
		}
	}

	messages = append(messages, &model.AgentChatMessage{Role: RoleUser, Content: "下一轮新需求"})
	loaded = loadedGuideDocsFromMessages(context.Background(), messages, registry)
	for _, doc := range createSkill.Meta.RequiredDocs {
		doc = normalizeGuideDocPath(doc)
		if _, ok := loaded[doc]; !ok {
			t.Fatalf("expected loaded docs from previous user turn to remain active, missing %s in %#v", doc, loaded)
		}
	}
}

func TestWorkspaceSkillGateAllowsCreateAfterPriorTurnReadSkill(t *testing.T) {
	registry := agentosskills.LoadEmbedded()
	messages := []*model.AgentChatMessage{
		{Role: RoleUser, Content: "帮我做一个评价系统"},
		assistantReadSkillMessage("call-create", "sop.create-project"),
		{Role: RoleTool, ToolCallID: "call-create", ToolStatus: ToolCallStatusOK},
		{Role: RoleUser, Content: "确认，开始创建"},
	}
	loadedSkills := loadedSkillsFromMessages(context.Background(), messages, registry)
	loadedDocs := loadedGuideDocsFromMessages(context.Background(), messages, registry)

	if _, blocked := workspaceSkillToolGateResult("create_directory", nil, loadedSkills, loadedDocs); blocked {
		t.Fatalf("create_directory should be allowed after create-project was read in a previous user turn")
	}
}

func assistantReadSkillMessage(toolCallID string, skillID string) *model.AgentChatMessage {
	tc := llms.ToolCall{ID: toolCallID, Type: "function"}
	tc.Function.Name = "read_skill"
	tc.Function.Arguments = `{"id":` + strconvQuote(skillID) + `}`
	raw, _ := json.Marshal([]llms.ToolCall{tc})
	s := string(raw)
	return &model.AgentChatMessage{
		Role:      RoleAssistant,
		ToolCalls: &s,
	}
}

func strconvQuote(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}
