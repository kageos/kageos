package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

func TestTaskAuditConsumerIsIdempotentAndPreservesOccurrence(t *testing.T) {
	_, db := newAppServiceOperateLogTest(t)
	event := dto.TaskAuditEvent{EventID: "timer-event-1", OccurredAt: time.Now().Add(-time.Hour).Truncate(time.Second), Action: "timer.task.updated", TaskID: 7, ResourcePath: "/alice/ops/job.form", ResourceName: "Job", Actor: "bob", Status: "success", Before: json.RawMessage(`{"title":"Before"}`), After: json.RawMessage(`{"title":"After"}`)}
	for i := 0; i < 2; i++ {
		if err := RecordTaskAudit(context.Background(), db, event); err != nil {
			t.Fatal(err)
		}
	}
	var logs []model.OperateLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ActorUser != "bob" || !time.Time(logs[0].CreatedAt).Equal(event.OccurredAt) {
		t.Fatalf("duplicate or incorrect audit: %+v", logs)
	}
	var after map[string]any
	if err := json.Unmarshal(logs[0].NewValuesJSON, &after); err != nil || after["title"] != "After" {
		t.Fatalf("after values must be JSON object: %s", logs[0].NewValuesJSON)
	}
	event.Action = "permission.role.granted"
	if err := RecordTaskAudit(context.Background(), db, event); err == nil {
		t.Fatal("consumer accepted unrelated action")
	}
}

func TestManagementAuditSurvivesCancellationAndPreservesFailureStage(t *testing.T) {
	_, db := newAppServiceOperateLogTest(t)
	ctx, cancel := context.WithCancel(contextx.WithRequestUser(context.Background(), "alice"))
	audit, err := beginManagementAudit(ctx, db, "directory.installed", "directory", "/alice/ops", "Bundle", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var row model.OperateLog
	if err := db.First(&row, audit.log.ID).Error; err != nil || row.Status != "pending" {
		t.Fatal("start not durable")
	}
	if err := audit.stage(ctx, "writing_files"); err != nil {
		t.Fatal(err)
	}
	cancel()
	operationErr := errors.New("download https://user:pass@example.test/bundle?token=secret failed")
	audit.finish(ctx, nil, &operationErr)
	if err := db.First(&row, audit.log.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "failed" || !strings.Contains(string(row.DetailsJSON), "writing_files") || strings.Contains(string(row.DetailsJSON), "secret") || strings.Contains(string(row.DetailsJSON), "user:pass") {
		t.Fatalf("incorrect failure audit: %s", row.DetailsJSON)
	}
}

func TestPublicShareManagementAuditExcludesAccessCredential(t *testing.T) {
	_, db := newAppServiceOperateLogTest(t)
	if err := db.AutoMigrate(&model.PublicShare{}, &model.Function{}, &model.ServiceTree{}); err != nil {
		t.Fatal(err)
	}
	path := "/alice/ops/submit.form"
	function := &model.Function{Router: path, TemplateType: "form", Schema: json.RawMessage(`{"version":1,"type":"form","form":{}}`)}
	if err := db.Create(function).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewPublicShareService(repository.NewPublicShareRepository(db), repository.NewFunctionRepository(db), repository.NewServiceTreeRepository(db), repository.NewOperateLogRepository(db))
	resp, err := svc.Create(context.Background(), &dto.CreatePublicShareReq{FullCodePath: path, Title: "Public form", MaxUses: 10}, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Disable(context.Background(), resp.ShareID, "bob"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Disable(context.Background(), resp.ShareID, "bob"); err != nil {
		t.Fatal(err)
	}
	var logs []model.OperateLog
	if err := db.Order("id").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 || logs[0].Status != "success" || logs[1].ActorUser != "bob" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
	raw, _ := json.Marshal(logs)
	if strings.Contains(string(raw), resp.ShareID) {
		t.Fatal("public access credential leaked into audit")
	}
	var before, after map[string]any
	_ = json.Unmarshal(logs[1].OldValuesJSON, &before)
	_ = json.Unmarshal(logs[1].NewValuesJSON, &after)
	if before["enabled"] != true || after["enabled"] != false {
		t.Fatal("share status change missing")
	}
}

func TestCapabilityAuditCapturesDownloadFailureAndSourceWithoutSecrets(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	svc := &serviceTreeCapabilityBundleService{appRepo: service.appRepo}
	_, err := svc.InstallCapabilityBundleFromURL(context.Background(), &dto.InstallCapabilityOptions{TargetDirectoryPath: "/alice/ops", Overwrite: true}, "invalid://host/bundle?token=secret", "private-install-key")
	if err == nil {
		t.Fatal("expected invalid URL failure")
	}
	var log model.OperateLog
	if err := db.First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.Action != "directory.installed" || log.Status != "failed" {
		t.Fatalf("missing install failure: %+v", log)
	}
	raw, _ := json.Marshal(log)
	if strings.Contains(string(raw), "token=secret") || strings.Contains(string(raw), "private-install-key") {
		t.Fatal("install credential leaked")
	}
}

func TestWorkspaceDeletionKeepsAuditAfterResourceIsGone(t *testing.T) {
	appRepo, _, db := newAppServiceRequestDeleteTestDeps(t)
	createRequestDeleteTestApp(t, appRepo)
	svc := NewAppService(AppServiceDependencies{AppRepository: appRepo, AppRuntimeClient: &fakeAppRuntimeClient{deleteResp: &dto.DeleteAppResp{User: "alice", App: "demo"}}})
	_, err := svc.DeleteApp(contextx.WithRequestUser(context.Background(), "alice"), &dto.DeleteAppReq{ResourcePath: "/alice/demo"})
	if err != nil {
		t.Fatal(err)
	}
	var log model.OperateLog
	if err := db.First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.Action != "workspace.deleted" || log.Status != "success" || !strings.Contains(string(log.OldValuesJSON), "Demo") || !strings.Contains(string(log.DetailsJSON), `"runtime_deleted":true`) {
		t.Fatalf("deletion history incomplete: %+v", log)
	}
}

func TestManagementAuditPanicIsNeverRecordedAsSuccess(t *testing.T) {
	_, db := newAppServiceOperateLogTest(t)
	ctx := context.Background()
	audit, err := beginManagementAudit(ctx, db, "workspace.deleted", "workspace", "/alice/ops", "Ops", "1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected original panic to propagate")
			}
		}()
		func() (resultErr error) {
			defer func() { audit.finish(ctx, nil, &resultErr) }()
			defer audit.capturePanic(&resultErr)
			panic("sensitive panic detail")
		}()
	}()
	var log model.OperateLog
	if err := db.First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.Status != "failed" || strings.Contains(string(log.DetailsJSON), "sensitive panic detail") {
		t.Fatalf("panic audit incorrect: %+v", log)
	}
}
