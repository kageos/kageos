package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAppServiceOperateLogTest(t *testing.T) (*AppService, *gorm.DB) {
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
	if err := db.AutoMigrate(&model.App{}, &model.OperateLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	appRepo := repository.NewAppRepository(db)
	if err := appRepo.CreateApp(&model.App{User: "alice", Code: "ops", Name: "Ops", Version: "v9"}); err != nil {
		t.Fatalf("create app: %v", err)
	}
	return &AppService{
		appRepo:        appRepo,
		operateLogRepo: repository.NewOperateLogRepository(db),
	}, db
}

func TestRecordTableActionLogPersistsFailedResultAndDuration(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{CompanyCode: "acme"})

	err := service.RecordTableActionLog(ctx, &dto.RecordTableActionLogReq{
		TenantUser:     "alice",
		RequestUser:    "bob",
		App:            "ops",
		Router:         "tickets.table",
		Action:         "OnTableUpdateRow",
		RowID:          42,
		Updates:        json.RawMessage(`{"status":"closed","password":"new-password"}`),
		OldValues:      json.RawMessage(`{"status":"open","password":"old-password"}`),
		ResponseBody:   json.RawMessage(`{"code":500,"error":"boom","secret":"response-secret","total_cost_mill":35}`),
		TraceID:        "trace-1",
		Version:        "v10",
		DurationMillis: 35,
		Status:         "failed",
		Summary:        "boom",
	})
	if err != nil {
		t.Fatalf("record table action log: %v", err)
	}

	var log model.OperateLog
	deadline := time.Now().Add(time.Second)
	for {
		queryErr := db.Where("target_id = ?", "42").First(&log).Error
		if queryErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("log was not persisted: %v", queryErr)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if log.Status != "failed" {
		t.Fatalf("status = %q", log.Status)
	}
	if log.CompanyCode != "acme" {
		t.Fatalf("company_code = %q", log.CompanyCode)
	}
	if log.Summary != "boom" {
		t.Fatalf("summary = %q", log.Summary)
	}
	if log.ResourceType != "table" || log.ResourcePath != "/alice/ops/tickets.table" {
		t.Fatalf("unexpected resource: type=%q path=%q", log.ResourceType, log.ResourcePath)
	}

	var details dto.TableActionLogDetails
	if err := json.Unmarshal(log.DetailsJSON, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details.RowID != 42 || details.Version != "v10" || details.DurationMillis != 35 {
		t.Fatalf("unexpected details: %+v", details)
	}

	var oldValues map[string]interface{}
	if err := json.Unmarshal(log.OldValuesJSON, &oldValues); err != nil {
		t.Fatalf("unmarshal old values: %v", err)
	}
	if _, exists := oldValues["password"]; exists {
		t.Fatalf("password should be removed from old values: %+v", oldValues)
	}
	if oldValues["status"] != "open" {
		t.Fatalf("old values were not redacted correctly: %+v", oldValues)
	}

	var newValues map[string]interface{}
	if err := json.Unmarshal(log.NewValuesJSON, &newValues); err != nil {
		t.Fatalf("unmarshal new values: %v", err)
	}
	if _, exists := newValues["password"]; exists {
		t.Fatalf("password should be removed from new values: %+v", newValues)
	}
	if newValues["status"] != "closed" {
		t.Fatalf("new values were not redacted correctly: %+v", newValues)
	}

	responseBody, ok := details.ResponseBody.(map[string]interface{})
	if !ok {
		t.Fatalf("response body should be an object: %+v", details.ResponseBody)
	}
	if _, exists := responseBody["secret"]; exists {
		t.Fatalf("secret should be removed from response body: %+v", responseBody)
	}
	if responseBody["error"] != "boom" {
		t.Fatalf("response body was not redacted correctly: %+v", responseBody)
	}
}
