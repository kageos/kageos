package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newServiceTreeAuditTestService(t *testing.T) (*ServiceTreeService, *gorm.DB, *model.ServiceTree) {
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
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}, &model.WorkspaceRoleAssignment{}, &model.OperateLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	appRepo := repository.NewAppRepository(db)
	app := &model.App{User: "alice", Code: "ops", Name: "Ops", Version: "v1"}
	if err := appRepo.CreateApp(app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	serviceTreeRepo := repository.NewServiceTreeRepository(db)
	functionNode := &model.ServiceTree{
		Name:         "Ticket List",
		Code:         "ticket_list.table",
		Type:         model.ServiceTreeTypeFunction,
		Description:  "old description",
		Tags:         "old",
		AppID:        app.ID,
		FullCodePath: "/alice/ops/ticket_list.table",
		TemplateType: "table",
		Version:      "v1",
		VersionNum:   1,
	}
	if err := serviceTreeRepo.Create(functionNode); err != nil {
		t.Fatalf("create service tree: %v", err)
	}

	operateLogRepo := repository.NewOperateLogRepository(db)
	teamAccess := NewTeamAccessService(repository.NewTeamAccessRepository(db), operateLogRepo, appRepo)
	service := NewServiceTreeService(serviceTreeRepo, appRepo, nil, nil, nil, nil, teamAccess)
	return service, db, functionNode
}

func TestServiceTreeUpdateWritesOperateLog(t *testing.T) {
	service, db, functionNode := newServiceTreeAuditTestService(t)
	ctx := actorContext("alice")

	nextName := "Ticket Table"
	nextDescription := "new description"
	if err := service.UpdateFunction(ctx, &dto.UpdateFunctionReq{
		ID:          functionNode.ID,
		Name:        &nextName,
		Description: &nextDescription,
	}); err != nil {
		t.Fatal(err)
	}

	log := waitOperateLog(t, db, "service_tree.node.updated")
	if log.ActorUser != "alice" || log.ResourceType != "function" || log.ResourcePath != "/alice/ops/ticket_list.table" {
		t.Fatalf("unexpected update log: %+v", log)
	}

	var oldValues dto.ServiceTreeNodeLogValues
	var newValues dto.ServiceTreeNodeLogValues
	if err := json.Unmarshal(log.OldValuesJSON, &oldValues); err != nil {
		t.Fatalf("unmarshal old values: %v", err)
	}
	if err := json.Unmarshal(log.NewValuesJSON, &newValues); err != nil {
		t.Fatalf("unmarshal new values: %v", err)
	}
	if oldValues.Name != "Ticket List" || newValues.Name != "Ticket Table" {
		t.Fatalf("unexpected name diff: old=%+v new=%+v", oldValues, newValues)
	}
	if oldValues.Description != "old description" || newValues.Description != "new description" {
		t.Fatalf("unexpected description diff: old=%+v new=%+v", oldValues, newValues)
	}
}

func TestServiceTreeDeleteWritesOperateLog(t *testing.T) {
	service, db, functionNode := newServiceTreeAuditTestService(t)
	ctx := actorContext("alice")

	if err := service.DeleteFunction(ctx, functionNode.ID); err != nil {
		t.Fatal(err)
	}

	log := waitOperateLog(t, db, "service_tree.node.deleted")
	if log.ActorUser != "alice" || log.ResourceType != "function" || log.ResourcePath != "/alice/ops/ticket_list.table" {
		t.Fatalf("unexpected delete log: %+v", log)
	}
	if len(log.NewValuesJSON) != 0 {
		t.Fatalf("delete log should not have new values: %s", string(log.NewValuesJSON))
	}

	var oldValues dto.ServiceTreeNodeLogValues
	if err := json.Unmarshal(log.OldValuesJSON, &oldValues); err != nil {
		t.Fatalf("unmarshal old values: %v", err)
	}
	if oldValues.ID != functionNode.ID || oldValues.FullCodePath != "/alice/ops/ticket_list.table" {
		t.Fatalf("unexpected delete old values: %+v", oldValues)
	}
}
