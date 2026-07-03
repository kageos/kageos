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

func TestRecordTableActionLogUsesContextAuditSource(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		ClientSource:          contextx.ClientSourceAgent,
		SourceType:            contextx.SourceTypeAgentTool,
		SourceRef:             "session-1",
		WorkspaceSessionID:    "session-1",
		WorkspaceSessionTitle: "订单处理",
		WorkspaceRole:         "app_operator",
	})

	err := service.RecordTableActionLog(ctx, &dto.RecordTableActionLogReq{
		TenantUser:  "alice",
		RequestUser: "bob",
		App:         "ops",
		Router:      "tickets.table",
		Action:      "OnTableAddRow",
		Body:        json.RawMessage(`{"title":"demo"}`),
		TraceID:     "trace-agent",
	})
	if err != nil {
		t.Fatalf("record table action log: %v", err)
	}

	var log model.OperateLog
	deadline := time.Now().Add(time.Second)
	for {
		queryErr := db.Where("trace_id = ?", "trace-agent").First(&log).Error
		if queryErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("log was not persisted: %v", queryErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if log.Source != contextx.ClientSourceAgent {
		t.Fatalf("source = %q, want agent", log.Source)
	}
	if log.SourceType != contextx.SourceTypeAgentTool || log.SourceRef != "session-1" {
		t.Fatalf("source fields mismatch: type=%q ref=%q", log.SourceType, log.SourceRef)
	}
	if log.ExecutorType != operateLogExecutorAgent {
		t.Fatalf("executor_type = %q, want agent", log.ExecutorType)
	}
	if log.WorkspaceSessionID != "session-1" || log.WorkspaceSessionTitle != "订单处理" || log.WorkspaceRole != "app_operator" {
		t.Fatalf("workspace session fields mismatch: id=%q title=%q role=%q", log.WorkspaceSessionID, log.WorkspaceSessionTitle, log.WorkspaceRole)
	}
	var details dto.TableActionLogDetails
	if err := json.Unmarshal(log.DetailsJSON, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details.SourceType != contextx.SourceTypeAgentTool || details.SourceRef != "session-1" {
		t.Fatalf("source details mismatch: %+v", details)
	}
}

func TestRecordTableActionLogUsesScheduledTaskAuditSource(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		ClientSource: contextx.ClientSourceScheduledTask,
		SourceType:   contextx.SourceTypeScheduledTask,
		SourceRef:    "timer_task:7:execution:9",
	})

	err := service.RecordTableActionLog(ctx, &dto.RecordTableActionLogReq{
		TenantUser:  "alice",
		RequestUser: "system",
		App:         "ops",
		Router:      "tickets.table",
		Action:      "OnTableAddRow",
		Body:        json.RawMessage(`{"title":"scheduled"}`),
		TraceID:     "trace-scheduled-table",
	})
	if err != nil {
		t.Fatalf("record scheduled table action log: %v", err)
	}

	var log model.OperateLog
	deadline := time.Now().Add(time.Second)
	for {
		queryErr := db.Where("trace_id = ?", "trace-scheduled-table").First(&log).Error
		if queryErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("log was not persisted: %v", queryErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if log.Source != contextx.ClientSourceScheduledTask {
		t.Fatalf("source = %q, want scheduled_task", log.Source)
	}
	if log.ExecutorType != operateLogExecutorScheduledFunction {
		t.Fatalf("executor_type = %q, want scheduled_function", log.ExecutorType)
	}
	var details dto.TableActionLogDetails
	if err := json.Unmarshal(log.DetailsJSON, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details.SourceType != contextx.SourceTypeScheduledTask || details.SourceRef != "timer_task:7:execution:9" {
		t.Fatalf("source details mismatch: %+v", details)
	}
}

func TestRecordTableActionLogMarksScheduledAgentAsAgentExecutor(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		ClientSource:          contextx.ClientSourceScheduledTask,
		SourceType:            contextx.SourceTypeScheduledTask,
		SourceRef:             "timer_task:7:execution:9",
		WorkspaceSessionID:    "session-scheduled-agent",
		WorkspaceSessionTitle: "物流巡检",
		WorkspaceRole:         "automation_operator",
	})

	err := service.RecordTableActionLog(ctx, &dto.RecordTableActionLogReq{
		TenantUser:  "alice",
		RequestUser: "system",
		App:         "ops",
		Router:      "tickets.table",
		Action:      "OnTableAddRow",
		Body:        json.RawMessage(`{"title":"scheduled-agent"}`),
		TraceID:     "trace-scheduled-agent-table",
	})
	if err != nil {
		t.Fatalf("record scheduled agent table action log: %v", err)
	}

	var log model.OperateLog
	deadline := time.Now().Add(time.Second)
	for {
		queryErr := db.Where("trace_id = ?", "trace-scheduled-agent-table").First(&log).Error
		if queryErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("log was not persisted: %v", queryErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if log.Source != contextx.ClientSourceScheduledTask {
		t.Fatalf("source = %q, want scheduled_task", log.Source)
	}
	if log.ExecutorType != operateLogExecutorAgent {
		t.Fatalf("executor_type = %q, want agent", log.ExecutorType)
	}
	if log.WorkspaceSessionID != "session-scheduled-agent" || log.WorkspaceSessionTitle != "物流巡检" {
		t.Fatalf("workspace session fields mismatch: id=%q title=%q", log.WorkspaceSessionID, log.WorkspaceSessionTitle)
	}
}

func TestRecordFormOperateLogInfersOpenAPISource(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		SourceType: contextx.SourceTypeOpenAPIToken,
		SourceRef:  "bob",
	})

	err := service.RecordFormOperateLog(ctx, &dto.RecordFormOperateLogReq{
		TenantUser:     "alice",
		RequestUser:    "bob",
		App:            "ops",
		Router:         "tools/export.form",
		Action:         "form_submit",
		FunctionMethod: "POST",
		RequestBody:    json.RawMessage(`{"format":"csv"}`),
		ResponseBody:   json.RawMessage(`{"ok":true}`),
		TraceID:        "trace-openapi",
		Status:         "success",
	})
	if err != nil {
		t.Fatalf("record form operate log: %v", err)
	}

	var log model.OperateLog
	deadline := time.Now().Add(time.Second)
	for {
		queryErr := db.Where("trace_id = ?", "trace-openapi").First(&log).Error
		if queryErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("log was not persisted: %v", queryErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if log.Source != contextx.ClientSourceOpenAPI {
		t.Fatalf("source = %q, want openapi", log.Source)
	}
	if log.SourceType != contextx.SourceTypeOpenAPIToken || log.SourceRef != "bob" {
		t.Fatalf("source fields mismatch: type=%q ref=%q", log.SourceType, log.SourceRef)
	}
	if log.ExecutorType != operateLogExecutorOpenAPI {
		t.Fatalf("executor_type = %q, want openapi", log.ExecutorType)
	}
	var details dto.FormOperateLogDetails
	if err := json.Unmarshal(log.DetailsJSON, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details.SourceType != contextx.SourceTypeOpenAPIToken || details.SourceRef != "bob" {
		t.Fatalf("source details mismatch: %+v", details)
	}
}

func TestRecordFormOperateLogUsesScheduledTaskAuditSource(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		ClientSource: contextx.ClientSourceScheduledTask,
		SourceType:   contextx.SourceTypeScheduledTask,
		SourceRef:    "timer_task:8:execution:10",
	})

	err := service.RecordFormOperateLog(ctx, &dto.RecordFormOperateLogReq{
		TenantUser:     "alice",
		RequestUser:    "system",
		App:            "ops",
		Router:         "tools/remind.form",
		Action:         "form_submit",
		FunctionMethod: "POST",
		RequestBody:    json.RawMessage(`{"lead_minutes":5}`),
		ResponseBody:   json.RawMessage(`{"ok":true}`),
		TraceID:        "trace-scheduled-form",
		Status:         "success",
	})
	if err != nil {
		t.Fatalf("record scheduled form operate log: %v", err)
	}

	var log model.OperateLog
	deadline := time.Now().Add(time.Second)
	for {
		queryErr := db.Where("trace_id = ?", "trace-scheduled-form").First(&log).Error
		if queryErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("log was not persisted: %v", queryErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if log.Source != contextx.ClientSourceScheduledTask {
		t.Fatalf("source = %q, want scheduled_task", log.Source)
	}
	if log.ExecutorType != operateLogExecutorScheduledFunction {
		t.Fatalf("executor_type = %q, want scheduled_function", log.ExecutorType)
	}
	var details dto.FormOperateLogDetails
	if err := json.Unmarshal(log.DetailsJSON, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details.SourceType != contextx.SourceTypeScheduledTask || details.SourceRef != "timer_task:8:execution:10" {
		t.Fatalf("source details mismatch: %+v", details)
	}
}

func TestRecordScheduledFunctionOperateLogPersistsGenericEntry(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:      "trace-scheduled-function",
		RequestUser:  "system",
		ClientSource: contextx.ClientSourceScheduledTask,
		SourceType:   contextx.SourceTypeScheduledTask,
		SourceRef:    "timer_task:9:execution:11",
	})

	err := service.recordScheduledFunctionOperateLog(ctx, scheduledFunctionPayload{
		FullCodePath: "/alice/ops/reports.chart",
		TemplateType: "chart",
		Action:       "execute",
		Payload:      json.RawMessage(`{"page":1}`),
	}, scheduledFunctionRunResult{
		Content: `{"ok":true}`,
		Data: map[string]interface{}{
			"ok": true,
		},
	}, nil, 42)
	if err != nil {
		t.Fatalf("record scheduled function operate log: %v", err)
	}

	var log model.OperateLog
	if err := db.Where("trace_id = ?", "trace-scheduled-function").First(&log).Error; err != nil {
		t.Fatalf("log was not persisted: %v", err)
	}
	if log.Action != "scheduled_function_execute" || log.ResourceType != "function" {
		t.Fatalf("unexpected action/resource: action=%q resource_type=%q", log.Action, log.ResourceType)
	}
	if log.Source != contextx.ClientSourceScheduledTask {
		t.Fatalf("source = %q, want scheduled_task", log.Source)
	}
	if log.SourceType != contextx.SourceTypeScheduledTask || log.SourceRef != "timer_task:9:execution:11" {
		t.Fatalf("source fields mismatch: type=%q ref=%q", log.SourceType, log.SourceRef)
	}
	if log.ExecutorType != operateLogExecutorScheduledFunction {
		t.Fatalf("executor_type = %q, want scheduled_function", log.ExecutorType)
	}
	if log.ResourcePath != "/alice/ops/reports.chart" || log.Status != "success" {
		t.Fatalf("unexpected log path/status: path=%q status=%q", log.ResourcePath, log.Status)
	}
	var details dto.FunctionExecutionLogDetails
	if err := json.Unmarshal(log.DetailsJSON, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details.SourceType != contextx.SourceTypeScheduledTask || details.SourceRef != "timer_task:9:execution:11" {
		t.Fatalf("source details mismatch: %+v", details)
	}
	if details.TemplateType != "chart" || details.Method != "GET" || details.DurationMillis != 42 {
		t.Fatalf("unexpected function details: %+v", details)
	}
}
