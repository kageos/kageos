package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFunctionRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.Function{}); err != nil {
		t.Fatalf("migrate function: %v", err)
	}
	return db
}

func TestCreateFunctionsUpdatesExistingActiveFunction(t *testing.T) {
	db := newFunctionRepositoryTestDB(t)
	repo := NewFunctionRepository(db)

	first := &model.Function{
		AppID:        42,
		Method:       "POST",
		Router:       "/alice/helpdesk/ticket/list.table",
		Schema:       json.RawMessage(`{"version":1}`),
		CreateTables: "tickets",
		Connectors:   "github",
		TemplateType: "table",
	}
	first.CreatedBy = "alice"
	if err := repo.CreateFunctions(context.Background(), []*model.Function{first}); err != nil {
		t.Fatalf("create first function: %v", err)
	}
	if first.ID == 0 {
		t.Fatalf("expected first function id to be hydrated")
	}

	second := &model.Function{
		AppID:              first.AppID,
		Method:             first.Method,
		Router:             first.Router,
		Schema:             json.RawMessage(`{"version":2}`),
		HasConfig:          true,
		CreateTables:       "tickets,ticket_comments",
		Connectors:         "github,notion",
		ConnectorEndpoints: `[{"provider":"github","method":"GET","path":"/user"}]`,
		TemplateType:       "form",
	}
	second.CreatedBy = "bob"
	if err := repo.CreateFunctions(context.Background(), []*model.Function{second}); err != nil {
		t.Fatalf("upsert second function: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected upsert to hydrate existing id %d, got %d", first.ID, second.ID)
	}

	var count int64
	if err := db.Model(&model.Function{}).Where("app_id = ? AND method = ? AND router = ?", first.AppID, first.Method, first.Router).Count(&count).Error; err != nil {
		t.Fatalf("count functions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one function row after upsert, got %d", count)
	}

	var got model.Function
	if err := db.First(&got, first.ID).Error; err != nil {
		t.Fatalf("load function: %v", err)
	}
	if string(got.Schema) != string(second.Schema) || got.TemplateType != second.TemplateType || got.Connectors != second.Connectors || !got.HasConfig {
		t.Fatalf("function was not updated: %+v", got)
	}
	if got.CreatedBy != "alice" {
		t.Fatalf("created_by should keep original creator, got %q", got.CreatedBy)
	}
}

func TestCreateFunctionsKeepsSoftDeletedHistory(t *testing.T) {
	db := newFunctionRepositoryTestDB(t)
	repo := NewFunctionRepository(db)

	function := &model.Function{
		AppID:        7,
		Method:       "GET",
		Router:       "/alice/helpdesk/ticket/list.table",
		Schema:       json.RawMessage(`{"version":1}`),
		TemplateType: "table",
	}
	if err := repo.CreateFunctions(context.Background(), []*model.Function{function}); err != nil {
		t.Fatalf("create function: %v", err)
	}

	if err := db.Delete(&model.Function{}, function.ID).Error; err != nil {
		t.Fatalf("soft delete function: %v", err)
	}

	restored := &model.Function{
		AppID:        function.AppID,
		Method:       function.Method,
		Router:       function.Router,
		Schema:       json.RawMessage(`{"version":2}`),
		TemplateType: "form",
	}
	if err := repo.CreateFunctions(context.Background(), []*model.Function{restored}); err != nil {
		t.Fatalf("create active function with soft-deleted history: %v", err)
	}
	if restored.ID == function.ID {
		t.Fatalf("expected a new active function while keeping soft-deleted history, got reused id %d", restored.ID)
	}

	var activeCount int64
	if err := db.Model(&model.Function{}).Where("app_id = ? AND method = ? AND router = ?", function.AppID, function.Method, function.Router).Count(&activeCount).Error; err != nil {
		t.Fatalf("count active functions: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("expected one active function, active count=%d", activeCount)
	}

	var totalCount int64
	if err := db.Unscoped().Model(&model.Function{}).Where("app_id = ? AND method = ? AND router = ?", function.AppID, function.Method, function.Router).Count(&totalCount).Error; err != nil {
		t.Fatalf("count all functions: %v", err)
	}
	if totalCount != 2 {
		t.Fatalf("expected soft-deleted history plus new active function, total count=%d", totalCount)
	}
}

func TestCreateFunctionsSoftDeletesDuplicateActiveFunctions(t *testing.T) {
	db := newFunctionRepositoryTestDB(t)
	if err := db.AutoMigrate(&model.ServiceTree{}); err != nil {
		t.Fatalf("migrate service tree: %v", err)
	}
	repo := NewFunctionRepository(db)

	oldDuplicate := &model.Function{
		AppID:        7,
		Method:       "GET",
		Router:       "/alice/helpdesk/ticket/list.table",
		Schema:       json.RawMessage(`{"version":1}`),
		TemplateType: "table",
	}
	newDuplicate := &model.Function{
		AppID:        oldDuplicate.AppID,
		Method:       oldDuplicate.Method,
		Router:       oldDuplicate.Router,
		Schema:       json.RawMessage(`{"version":2}`),
		TemplateType: "table",
	}
	if err := db.Create(oldDuplicate).Error; err != nil {
		t.Fatalf("create old duplicate: %v", err)
	}
	if err := db.Create(newDuplicate).Error; err != nil {
		t.Fatalf("create new duplicate: %v", err)
	}
	tree := &model.ServiceTree{
		AppID: oldDuplicate.AppID,
		RefID: oldDuplicate.ID,
		Type:  model.ServiceTreeTypeFunction,
	}
	if err := db.Create(tree).Error; err != nil {
		t.Fatalf("create service tree: %v", err)
	}

	incoming := &model.Function{
		AppID:        oldDuplicate.AppID,
		Method:       oldDuplicate.Method,
		Router:       oldDuplicate.Router,
		Schema:       json.RawMessage(`{"version":3}`),
		TemplateType: "form",
	}
	if err := repo.CreateFunctions(context.Background(), []*model.Function{incoming}); err != nil {
		t.Fatalf("create/update function: %v", err)
	}
	if incoming.ID != newDuplicate.ID {
		t.Fatalf("expected newest active function id %d to be kept, got %d", newDuplicate.ID, incoming.ID)
	}

	var activeCount int64
	if err := db.Model(&model.Function{}).Where("app_id = ? AND method = ? AND router = ?", oldDuplicate.AppID, oldDuplicate.Method, oldDuplicate.Router).Count(&activeCount).Error; err != nil {
		t.Fatalf("count active functions: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("expected duplicate active functions to be healed to one active row, got %d", activeCount)
	}

	var totalCount int64
	if err := db.Unscoped().Model(&model.Function{}).Where("app_id = ? AND method = ? AND router = ?", oldDuplicate.AppID, oldDuplicate.Method, oldDuplicate.Router).Count(&totalCount).Error; err != nil {
		t.Fatalf("count all functions: %v", err)
	}
	if totalCount != 2 {
		t.Fatalf("expected duplicate row to remain as soft-deleted history, total count=%d", totalCount)
	}

	var gotTree model.ServiceTree
	if err := db.First(&gotTree, tree.ID).Error; err != nil {
		t.Fatalf("load service tree: %v", err)
	}
	if gotTree.RefID != newDuplicate.ID {
		t.Fatalf("expected service_tree ref to be repointed to %d, got %d", newDuplicate.ID, gotTree.RefID)
	}
}
