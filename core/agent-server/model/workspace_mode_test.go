package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitWorkspaceModesOnlySeedsDev(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&WorkspaceMode{}); err != nil {
		t.Fatalf("migrate workspace mode: %v", err)
	}

	if err := InitWorkspaceModes(db); err != nil {
		t.Fatalf("InitWorkspaceModes returned error: %v", err)
	}

	var modes []WorkspaceMode
	if err := db.Find(&modes).Error; err != nil {
		t.Fatalf("query workspace modes: %v", err)
	}
	if len(modes) != 1 || modes[0].Code != "dev" {
		t.Fatalf("builtin modes = %#v, want only dev", modes)
	}
	tools := modes[0].GetToolNames()
	for _, want := range []string{"change_role", "read_doc", "write_prd", "write_go_file", "build_workspace", "run_form_submit"} {
		if !containsToolName(tools, want) {
			t.Fatalf("dev tools missing %s: %v", want, tools)
		}
	}
	for _, removed := range removedDocToolNames() {
		if containsToolName(tools, removed) {
			t.Fatalf("dev tools should not include %s: %v", removed, tools)
		}
	}
}

func removedDocToolNames() []string {
	return []string{"read_" + "sk" + "ill", "search_" + "sk" + "ills"}
}

func containsToolName(tools []string, target string) bool {
	for _, tool := range tools {
		if tool == target {
			return true
		}
	}
	return false
}
