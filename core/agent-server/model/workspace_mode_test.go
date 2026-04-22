package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitWorkspaceModesRefreshesExistingBuiltinTools(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&WorkspaceMode{}); err != nil {
		t.Fatalf("migrate workspace mode: %v", err)
	}
	old := &WorkspaceMode{
		Code:      "execute",
		Name:      "旧执行模式",
		ToolNames: "run_table_search;run_table_create",
		IsBuiltin: false,
	}
	if err := db.Create(old).Error; err != nil {
		t.Fatalf("seed old mode: %v", err)
	}

	if err := InitWorkspaceModes(db); err != nil {
		t.Fatalf("InitWorkspaceModes returned error: %v", err)
	}

	var got WorkspaceMode
	if err := db.Where("code = ?", "execute").First(&got).Error; err != nil {
		t.Fatalf("query execute mode: %v", err)
	}
	tools := got.GetToolNames()
	if !containsToolName(tools, "run_table_delete") {
		t.Fatalf("execute tools missing run_table_delete: %v", tools)
	}
	if !containsToolName(tools, "run_table_batch_create") {
		t.Fatalf("execute tools missing run_table_batch_create: %v", tools)
	}
	if !got.IsBuiltin {
		t.Fatal("execute mode should be marked builtin after refresh")
	}
}

func containsToolName(tools []string, target string) bool {
	for _, tool := range tools {
		if tool == target {
			return true
		}
	}
	return false
}
