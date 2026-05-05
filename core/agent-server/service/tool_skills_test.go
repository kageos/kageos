package service

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestSkillsToolsRegistered(t *testing.T) {
	registry := NewToolRegistry(nil)
	defs, err := registry.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}
	names := map[string]bool{}
	for _, def := range defs {
		names[def.Name] = true
	}
	for _, name := range workspaceSkillToolNames {
		if !names[name] {
			t.Fatalf("expected tool %s to be registered", name)
		}
	}
}

func TestAppendWorkspaceSkillToolNamesAlwaysOn(t *testing.T) {
	base := []string{"read_dir"}
	if got := appendWorkspaceSkillToolNames(base); strings.Join(got, ",") != "read_dir,read_skill,search_skills" {
		t.Fatalf("skills tool names = %#v", got)
	}

	got := appendWorkspaceSkillToolNames([]string{"read_dir", "search_skills"})
	if strings.Join(got, ",") != "read_dir,search_skills,read_skill" {
		t.Fatalf("deduped skills tool names = %#v", got)
	}
	if workspaceSkillsPrompt("execute") == "" {
		t.Fatal("expected skills prompt")
	}
}

func TestWorkspaceSkillsPromptContainsDirectReadCatalog(t *testing.T) {
	prompt := workspaceSkillsPrompt("dev")
	for _, want := range []string{"### Skills 目录", "`sop.create-project`", "`sop.explain-project`", "`sdk.form-submit-basic`", "`sdk.table-crud-basic`", "`sdk.combo-table-form`", "`sdk.combo-table-form-chart`", "`sdk.widget-selection`", "`system.openapi.message`", "`system.tools`", "直接 `read_skill"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("workspace skills prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "先用 search_skills") {
		t.Fatalf("workspace skills prompt should not require search first:\n%s", prompt)
	}
	if !strings.Contains(prompt, "不能直接继续") || !strings.Contains(prompt, "allowed_tools") {
		t.Fatalf("workspace skills prompt should enforce allowed_tools boundary:\n%s", prompt)
	}
}

func TestSearchAndReadSkillsToolExecute(t *testing.T) {
	searchTool := &SearchSkillsTool{}
	searchResult := searchTool.Execute(context.Background(), ToolCall{
		Args: map[string]interface{}{
			"keyword": "创建项目",
			"mode":    "execute",
			"limit":   5,
		},
	})
	if searchResult.IsError {
		t.Fatalf("search_skills returned error: %s", searchResult.Content)
	}
	data, ok := searchResult.Data.(searchSkillsResultData)
	if !ok {
		t.Fatalf("search data type = %T, want searchSkillsResultData", searchResult.Data)
	}
	if data.Count == 0 || data.Skills[0].Meta.ID != "sop.create-project" {
		t.Fatalf("unexpected search result: %#v", data)
	}

	readTool := &ReadSkillTool{}
	readResult := readTool.Execute(context.Background(), ToolCall{
		Args: map[string]interface{}{
			"id": data.Skills[0].Meta.ID,
		},
	})
	if readResult.IsError {
		t.Fatalf("read_skill returned error: %s", readResult.Content)
	}
	readData, ok := readResult.Data.(readSkillResultData)
	if !ok {
		t.Fatalf("read data type = %T, want readSkillResultData", readResult.Data)
	}
	if readData.Skill == nil || !strings.Contains(readData.Skill.Body, "创建项目 SOP") {
		t.Fatalf("unexpected read skill data: %#v", readData.Skill)
	}
	for _, want := range []string{"落地目录和函数清单", "示例数据", "确认后我将创建目录"} {
		if !strings.Contains(readData.Skill.Body, want) {
			t.Fatalf("create-project skill body missing %q", want)
		}
	}
	wantRequiredDocs := []string{
		"/system/prompt/platform-capability-boundaries",
		"/system/prompt/platform-overview",
		"/system/prompt/platform-function-architecture",
		"/system/prompt/platform-cross-cutting-capabilities",
		"/system/prompt/sdk/agent-app-sdk-readme",
	}
	if !reflect.DeepEqual(readData.Skill.Meta.RequiredDocs, wantRequiredDocs) {
		t.Fatalf("skill meta required_docs = %#v, want %#v", readData.Skill.Meta.RequiredDocs, wantRequiredDocs)
	}
	if len(readData.RequiredDocs) == 0 {
		t.Fatal("expected read_skill to auto-load required_docs")
	}
	if requiredDocsPos, skillPos := strings.Index(readResult.Content, `"required_docs"`), strings.Index(readResult.Content, `"skill"`); requiredDocsPos < 0 || skillPos < 0 || requiredDocsPos > skillPos {
		t.Fatalf("read_skill content should put required_docs before large skill body, required_docs=%d skill=%d content prefix=%s", requiredDocsPos, skillPos, readResult.Content[:min(len(readResult.Content), 200)])
	}
	seenArchitecture := false
	seenSDKReadme := false
	for _, doc := range readData.RequiredDocs {
		if doc.Path == "/system/prompt/platform-function-architecture" {
			seenArchitecture = true
			if doc.IsError || !strings.Contains(doc.Content, "Form/Table/Chart 组合架构") {
				t.Fatalf("unexpected architecture required_doc content: %#v", doc)
			}
		}
		if doc.Path == "/system/prompt/sdk/agent-app-sdk-readme" {
			seenSDKReadme = true
			if doc.IsError || !strings.Contains(doc.Content, "Agent-App SDK 使用说明") {
				t.Fatalf("unexpected sdk readme required_doc content: %#v", doc)
			}
		}
	}
	if !seenArchitecture {
		t.Fatalf("expected architecture required_doc in read_skill result: %#v", readData.RequiredDocs)
	}
	if !seenSDKReadme {
		t.Fatalf("expected sdk readme required_doc in read_skill result: %#v", readData.RequiredDocs)
	}
}

func TestScenarioSkillsRouteToTaskPackages(t *testing.T) {
	searchTool := &SearchSkillsTool{}
	for _, tt := range []struct {
		keyword string
		wantID  string
		wantDoc string
	}{
		{keyword: "文件处理", wantID: "sdk.form-submit-basic", wantDoc: "/system/prompt/sdk/form-submit-basic"},
		{keyword: "CRUD", wantID: "sdk.table-crud-basic", wantDoc: "/system/prompt/sdk/table-crud-basic"},
		{keyword: "评价系统", wantID: "sdk.combo-table-form", wantDoc: "/system/prompt/sdk/combo-table-form"},
		{keyword: "收银系统", wantID: "sdk.combo-table-form-chart", wantDoc: "/system/prompt/sdk/combo-table-form-chart"},
	} {
		searchResult := searchTool.Execute(context.Background(), ToolCall{
			Args: map[string]interface{}{
				"keyword": tt.keyword,
				"mode":    "dev",
				"limit":   5,
			},
		})
		if searchResult.IsError {
			t.Fatalf("search_skills(%q) returned error: %s", tt.keyword, searchResult.Content)
		}
		data, ok := searchResult.Data.(searchSkillsResultData)
		if !ok {
			t.Fatalf("search data type = %T, want searchSkillsResultData", searchResult.Data)
		}
		if data.Count == 0 || data.Skills[0].Meta.ID != tt.wantID {
			t.Fatalf("unexpected search result for %q: %#v", tt.keyword, data)
		}

		readResult := (&ReadSkillTool{}).Execute(context.Background(), ToolCall{
			Args: map[string]interface{}{"id": tt.wantID},
		})
		if readResult.IsError {
			t.Fatalf("read_skill(%s) returned error: %s", tt.wantID, readResult.Content)
		}
		readData, ok := readResult.Data.(readSkillResultData)
		if !ok {
			t.Fatalf("read data type = %T, want readSkillResultData", readResult.Data)
		}
		if !containsString(readData.Skill.Meta.RequiredDocs, tt.wantDoc) {
			t.Fatalf("skill %s required_docs = %#v, want %s", tt.wantID, readData.Skill.Meta.RequiredDocs, tt.wantDoc)
		}
		if !containsString(readData.Skill.Meta.RequiredDocs, "/system/prompt/sdk/common-runtime-capabilities") {
			t.Fatalf("skill %s required_docs = %#v, want common runtime capabilities", tt.wantID, readData.Skill.Meta.RequiredDocs)
		}
		foundDoc := false
		foundCommonRuntime := false
		for _, doc := range readData.RequiredDocs {
			if doc.Path == tt.wantDoc && !doc.IsError && strings.TrimSpace(doc.Content) != "" {
				foundDoc = true
			}
			if doc.Path == "/system/prompt/sdk/common-runtime-capabilities" && !doc.IsError && strings.Contains(doc.Content, "ctx.SendMessage") {
				foundCommonRuntime = true
			}
		}
		if !foundDoc {
			t.Fatalf("expected auto-loaded doc %s in %#v", tt.wantDoc, readData.RequiredDocs)
		}
		if !foundCommonRuntime {
			t.Fatalf("expected auto-loaded common runtime doc in %#v", readData.RequiredDocs)
		}
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
