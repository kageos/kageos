package service

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/agent-server/prompt"
)

func TestAllListedToolsHaveHandlers(t *testing.T) {
	reg := NewToolRegistry()
	defs, err := reg.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	if len(defs) == 0 {
		t.Fatal("expected builtin tools, got none")
	}

	for _, def := range defs {
		tool, ok := reg.tools[def.Name]
		if !ok {
			t.Fatalf("listed tool %q has no registered handler", def.Name)
		}
		if tool.Definition().Name != def.Name {
			t.Fatalf("listed tool %q returned mismatched definition %q", def.Name, tool.Definition().Name)
		}
	}
}

func TestRetiredRouterToolsAreNotInMainRegistry(t *testing.T) {
	reg := NewToolRegistry()
	retiredSuffix := "inte" + "nt"
	for _, name := range []string{"classify_" + retiredSuffix, "handoff_" + retiredSuffix} {
		if _, ok := reg.tools[name]; ok {
			t.Fatalf("%s should be folded into change_role, not exposed as a standalone tool", name)
		}
	}
	if _, ok := reg.tools["change_role"]; !ok {
		t.Fatal("change_role should be registered")
	}
	if _, ok := reg.tools["write_"+"doc"]; ok {
		t.Fatal("the standalone document writer should be retired in favor of write_file/edit_file")
	}
}

func TestCodeEditToolsAreRegistered(t *testing.T) {
	reg := NewToolRegistry()
	for _, name := range []string{"read_file", "edit_file", "write_file"} {
		if _, ok := reg.tools[name]; !ok {
			t.Fatalf("%s should be registered", name)
		}
	}
}

func TestModeToolNamesResolveInRegistry(t *testing.T) {
	reg := NewToolRegistry()

	defs, err := reg.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	specs := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		specs[def.Name] = struct{}{}
	}

	provider := prompt.GetModeProvider("dev")
	if provider == nil {
		t.Fatal("mode provider dev is nil")
	}
	for _, name := range provider.ToolNames() {
		if _, ok := specs[name]; !ok {
			t.Fatalf("dev mode references unknown tool spec %q", name)
		}
		if _, ok := reg.tools[name]; !ok {
			t.Fatalf("dev mode references missing tool handler %q", name)
		}
	}
}

func TestScheduledTaskToolsAreRegistered(t *testing.T) {
	reg := NewToolRegistry()
	for _, name := range []string{
		"create_scheduled_function_task",
		"create_scheduled_agent_task",
		"list_scheduled_tasks",
		"manage_scheduled_task",
		"list_scheduled_task_executions",
	} {
		if _, ok := reg.tools[name]; !ok {
			t.Fatalf("%s should be registered", name)
		}
	}
}

func TestWebSearchToolIsRegistered(t *testing.T) {
	reg := NewToolRegistry()
	if _, ok := reg.tools["web_search"]; !ok {
		t.Fatal("web_search should be registered")
	}
}

func TestSendNotificationToolIsRegistered(t *testing.T) {
	reg := NewToolRegistry()
	if _, ok := reg.tools["send_notification"]; !ok {
		t.Fatal("send_notification should be registered")
	}
}

func TestToolSchemasAreWellFormed(t *testing.T) {
	reg := NewToolRegistry()
	defs, err := reg.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	for _, def := range defs {
		schema := def.InputSchema
		if got := schema["type"]; got != "object" {
			t.Fatalf("tool %q schema type = %v, want object", def.Name, got)
		}
		properties, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("tool %q schema properties missing or invalid", def.Name)
		}
		required, ok := schema["required"].([]interface{})
		if !ok {
			t.Fatalf("tool %q schema required missing or invalid", def.Name)
		}
		for _, item := range required {
			name, ok := item.(string)
			if !ok {
				t.Fatalf("tool %q schema required item has invalid type %T", def.Name, item)
			}
			if _, exists := properties[name]; !exists {
				t.Fatalf("tool %q schema required %q not found in properties", def.Name, name)
			}
		}
	}
}

func TestScheduledTaskCreateSchemasExposeScheduleFields(t *testing.T) {
	reg := NewToolRegistry()
	for _, name := range []string{"create_scheduled_function_task", "create_scheduled_agent_task"} {
		schema := reg.tools[name].Definition().InputSchema
		properties := schema["properties"].(map[string]interface{})
		for _, field := range []string{"schedule_type", "run_at", "cron_expr", "interval_seconds", "timezone", "max_runs"} {
			if _, ok := properties[field]; !ok {
				t.Fatalf("%s schema should expose %s, properties=%v", name, field, properties)
			}
		}
		required := schema["required"].([]interface{})
		if containsInterfaceString(required, "schedule_type") {
			t.Fatalf("%s schema should allow schedule_type inference, required=%v", name, required)
		}
	}
}

func TestDerivedSchemasHideCompatFields(t *testing.T) {
	reg := NewToolRegistry()

	readDoc := reg.tools["read_doc"].Definition().InputSchema
	readDocProps := readDoc["properties"].(map[string]interface{})
	if _, ok := readDocProps["directory"]; !ok {
		t.Fatal("read_doc schema should expose directory")
	}
	if _, ok := readDocProps["full_code_path"]; ok {
		t.Fatal("read_doc schema should not expose full_code_path")
	}

	writeFile := reg.tools["write_file"].Definition().InputSchema
	writeFileProps := writeFile["properties"].(map[string]interface{})
	if _, ok := writeFileProps["content"]; !ok {
		t.Fatal("write_file schema should expose content")
	}
	if _, ok := writeFileProps["name"]; !ok {
		t.Fatal("write_file schema should expose docs display name")
	}
	if _, ok := writeFileProps["source_code"]; ok {
		t.Fatal("write_file schema should not expose source_code")
	}
}

func TestDerivedSchemaNestedArrayItems(t *testing.T) {
	reg := NewToolRegistry()
	schema := reg.tools["edit_file"].Definition().InputSchema
	properties := schema["properties"].(map[string]interface{})
	if _, ok := properties["name"]; !ok {
		t.Fatal("edit_file schema should expose docs display name")
	}
	for _, name := range []string{"search_edits", "line_edits"} {
		if _, ok := properties[name]; !ok {
			t.Fatalf("edit_file schema should expose %s", name)
		}
	}
	lineEdits := properties["line_edits"].(map[string]interface{})
	if lineEdits["type"] != "array" {
		t.Fatalf("edit_file line_edits type = %v, want array", lineEdits["type"])
	}
	items, ok := lineEdits["items"].(map[string]interface{})
	if !ok {
		t.Fatal("edit_file line_edits items missing or invalid")
	}
	if items["type"] != "object" {
		t.Fatalf("edit_file line_edits item type = %v, want object", items["type"])
	}
	itemProps, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("edit_file line_edits item properties missing or invalid")
	}
	if _, ok := itemProps["start_line"]; !ok {
		t.Fatal("edit_file line_edits item should expose start_line")
	}
	required, ok := items["required"].([]interface{})
	if !ok || len(required) != 3 || required[0] != "start_line" || required[1] != "end_line" || required[2] != "replacement" {
		t.Fatalf("edit_file line_edits item required = %v, want [start_line end_line replacement]", items["required"])
	}

	searchEdits := properties["search_edits"].(map[string]interface{})
	if searchEdits["type"] != "array" {
		t.Fatalf("edit_file search_edits type = %v, want array", searchEdits["type"])
	}
	searchItems, ok := searchEdits["items"].(map[string]interface{})
	if !ok {
		t.Fatal("edit_file search_edits items missing or invalid")
	}
	if searchItems["type"] != "object" {
		t.Fatalf("edit_file search_edits item type = %v, want object", searchItems["type"])
	}
	searchProps, ok := searchItems["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("edit_file search_edits item properties missing or invalid")
	}
	if _, ok := searchProps["old_text"]; !ok {
		t.Fatal("edit_file search_edits item should expose old_text")
	}
	searchRequired, ok := searchItems["required"].([]interface{})
	if !ok || len(searchRequired) != 2 || searchRequired[0] != "old_text" || searchRequired[1] != "new_text" {
		t.Fatalf("edit_file search_edits item required = %v, want [old_text new_text]", searchItems["required"])
	}
}

func TestBuiltinToolOutputSchemasAreWellFormed(t *testing.T) {
	reg := NewToolRegistry()
	defs, err := reg.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	for _, def := range defs {
		schema := def.OutputSchema
		if schema == nil {
			t.Fatalf("tool %q output schema is nil", def.Name)
		}
		if got := schema["type"]; got != "object" {
			t.Fatalf("tool %q output schema type = %v, want object", def.Name, got)
		}
		properties, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("tool %q output schema properties missing or invalid", def.Name)
		}
		if _, ok := properties["content"]; !ok {
			t.Fatalf("tool %q output schema should expose content", def.Name)
		}
		if _, ok := properties["is_error"]; !ok {
			t.Fatalf("tool %q output schema should expose is_error", def.Name)
		}
		required, ok := schema["required"].([]interface{})
		if !ok {
			t.Fatalf("tool %q output schema required missing or invalid", def.Name)
		}
		if len(required) != 2 || required[0] != "content" || required[1] != "is_error" {
			t.Fatalf("tool %q output schema required = %v, want [content is_error]", def.Name, required)
		}
	}
}

func TestReadGoFileOutputSchemaExposesStructuredData(t *testing.T) {
	reg := NewToolRegistry()
	schema := reg.tools["read_file"].Definition().OutputSchema
	properties := schema["properties"].(map[string]interface{})
	data, ok := properties["data"].(map[string]interface{})
	if !ok {
		t.Fatal("read_file output schema should expose data")
	}
	if data["type"] != "object" {
		t.Fatalf("read_file data schema type = %v, want object", data["type"])
	}
	dataProps, ok := data["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("read_file data schema properties missing or invalid")
	}
	for _, name := range []string{"target_path", "file_name", "content_sha", "content", "numbered_content"} {
		if _, ok := dataProps[name]; !ok {
			t.Fatalf("read_file data schema should expose %q", name)
		}
	}
}

func TestReadDirOutputSchemaExposesStructuredData(t *testing.T) {
	reg := NewToolRegistry()
	schema := reg.tools["read_dir"].Definition().OutputSchema
	properties := schema["properties"].(map[string]interface{})
	data, ok := properties["data"].(map[string]interface{})
	if !ok {
		t.Fatal("read_dir output schema should expose data")
	}
	if data["type"] != "object" {
		t.Fatalf("read_dir data schema type = %v, want object", data["type"])
	}
	dataProps, ok := data["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("read_dir data schema properties missing or invalid")
	}
	for _, name := range []string{"resolved_path", "directory", "summary"} {
		if _, ok := dataProps[name]; !ok {
			t.Fatalf("read_dir data schema should expose %q", name)
		}
	}
	summary, ok := dataProps["summary"].(map[string]interface{})
	if !ok || summary["type"] != "object" {
		t.Fatalf("read_dir summary schema = %v, want object", dataProps["summary"])
	}
	summaryProps, ok := summary["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("read_dir summary properties missing or invalid")
	}
	for _, name := range []string{"directory_count", "document_count", "function_count", "file_count"} {
		if _, ok := summaryProps[name]; !ok {
			t.Fatalf("read_dir summary schema should expose %q", name)
		}
	}
}
