package service

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
)

func TestAllListedToolsHaveHandlers(t *testing.T) {
	reg := NewToolRegistry(nil)
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

func TestModeToolNamesResolveInRegistry(t *testing.T) {
	reg := NewToolRegistry(nil)

	defs, err := reg.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	specs := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		specs[def.Name] = struct{}{}
	}

	for _, mode := range []string{"qa", "dev", "modify", "execute", "agent"} {
		provider := prompt.GetModeProvider(mode)
		if provider == nil {
			t.Fatalf("mode provider %q is nil", mode)
		}
		for _, name := range provider.ToolNames() {
			if _, ok := specs[name]; !ok {
				t.Fatalf("mode %q references unknown tool spec %q", mode, name)
			}
			if _, ok := reg.tools[name]; !ok {
				t.Fatalf("mode %q references missing tool handler %q", mode, name)
			}
		}
	}
}

func TestToolSchemasAreWellFormed(t *testing.T) {
	reg := NewToolRegistry(nil)
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

func TestDerivedSchemasHideCompatFields(t *testing.T) {
	reg := NewToolRegistry(nil)

	readDoc := reg.tools["read_doc"].Definition().InputSchema
	readDocProps := readDoc["properties"].(map[string]interface{})
	if _, ok := readDocProps["directory"]; !ok {
		t.Fatal("read_doc schema should expose directory")
	}
	if _, ok := readDocProps["full_code_path"]; ok {
		t.Fatal("read_doc schema should not expose full_code_path")
	}

	writeGoFile := reg.tools["write_go_file"].Definition().InputSchema
	writeGoFileProps := writeGoFile["properties"].(map[string]interface{})
	if _, ok := writeGoFileProps["content"]; !ok {
		t.Fatal("write_go_file schema should expose content")
	}
	if _, ok := writeGoFileProps["source_code"]; ok {
		t.Fatal("write_go_file schema should not expose source_code")
	}
}

func TestDerivedSchemaNestedArrayItems(t *testing.T) {
	reg := NewToolRegistry(nil)
	schema := reg.tools["search_replace_file"].Definition().InputSchema
	properties := schema["properties"].(map[string]interface{})
	replacements := properties["replacements"].(map[string]interface{})
	if replacements["type"] != "array" {
		t.Fatalf("search_replace_file replacements type = %v, want array", replacements["type"])
	}
	items, ok := replacements["items"].(map[string]interface{})
	if !ok {
		t.Fatal("search_replace_file replacements items missing or invalid")
	}
	if items["type"] != "object" {
		t.Fatalf("search_replace_file replacements item type = %v, want object", items["type"])
	}
	itemProps, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("search_replace_file replacements item properties missing or invalid")
	}
	if _, ok := itemProps["search_string"]; !ok {
		t.Fatal("search_replace_file replacements item should expose search_string")
	}
	required, ok := items["required"].([]interface{})
	if !ok || len(required) != 1 || required[0] != "search_string" {
		t.Fatalf("search_replace_file replacements item required = %v, want [search_string]", items["required"])
	}
}

func TestBuiltinToolOutputSchemasAreWellFormed(t *testing.T) {
	reg := NewToolRegistry(nil)
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
	reg := NewToolRegistry(nil)
	schema := reg.tools["read_go_file"].Definition().OutputSchema
	properties := schema["properties"].(map[string]interface{})
	data, ok := properties["data"].(map[string]interface{})
	if !ok {
		t.Fatal("read_go_file output schema should expose data")
	}
	if data["type"] != "object" {
		t.Fatalf("read_go_file data schema type = %v, want object", data["type"])
	}
	dataProps, ok := data["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("read_go_file data schema properties missing or invalid")
	}
	for _, name := range []string{"target_path", "file_count", "files"} {
		if _, ok := dataProps[name]; !ok {
			t.Fatalf("read_go_file data schema should expose %q", name)
		}
	}
	files, ok := dataProps["files"].(map[string]interface{})
	if !ok || files["type"] != "array" {
		t.Fatalf("read_go_file files schema = %v, want array", dataProps["files"])
	}
	items, ok := files["items"].(map[string]interface{})
	if !ok {
		t.Fatal("read_go_file files items missing or invalid")
	}
	itemProps, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("read_go_file files item properties missing or invalid")
	}
	for _, name := range []string{"file_name", "full_path", "content"} {
		if _, ok := itemProps[name]; !ok {
			t.Fatalf("read_go_file files item should expose %q", name)
		}
	}
}

func TestReadDirOutputSchemaExposesStructuredData(t *testing.T) {
	reg := NewToolRegistry(nil)
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
	for _, name := range []string{"directory_count", "function_count", "file_count"} {
		if _, ok := summaryProps[name]; !ok {
			t.Fatalf("read_dir summary schema should expose %q", name)
		}
	}
}
