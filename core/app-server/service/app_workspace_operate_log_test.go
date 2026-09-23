package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
)

func TestWorkspaceUpdateLogPersistsFunctionSnapshots(t *testing.T) {
	service, db := newAppServiceOperateLogTest(t)
	var app model.App
	if err := db.First(&app).Error; err != nil {
		t.Fatal(err)
	}
	deleted := &dto.ApiInfo{Name: "旧统计", FullCodePath: "/alice/ops/old.chart", TemplateType: "chart", Desc: "private source detail"}
	resp := &dto.UpdateAppResp{NewVersion: "v10", Diff: &dto.DiffData{
		Add:      []*dto.ApiInfo{nil, {Name: "导入", User: "alice", App: "ops", Router: "import.form", TemplateType: "form"}},
		Update:   []*dto.ApiInfo{{Name: "客户", FullCodePath: "/alice/ops/customers.table", TemplateType: "table"}},
		Delete:   []*dto.ApiInfo{deleted, deleted},
		Packages: []*dto.PackageInfo{{Name: "Existing directory"}},
	}}
	details := workspaceUpdateOperateLogDetailsFromRequest(&dto.UpdateAppReq{}, resp, time.Now(), nil)
	// The log owns a snapshot even if the source object is later changed/deleted.
	deleted.Name = "changed later"
	service.writeWorkspaceOperateLog(context.Background(), &app, workspaceUpdatedAction, "success", "updated", details, workspaceOperateLogSnapshot(&app), workspaceUpdateResponseSnapshot(&app, resp))
	var log model.OperateLog
	deadline := time.Now().Add(3 * time.Second)
	for {
		if err := db.Where("action = ?", workspaceUpdatedAction).First(&log).Error; err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("workspace log was not persisted")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var got workspaceUpdateOperateLogDetails
	if err := json.Unmarshal(log.DetailsJSON, &got); err != nil {
		t.Fatal(err)
	}
	if got.Changes == nil || len(got.Changes.Added) != 1 || len(got.Changes.Updated) != 1 || len(got.Changes.Deleted) != 1 {
		t.Fatalf("unexpected changes: %+v", got.Changes)
	}
	if got.Changes.Deleted[0].Name != "旧统计" || got.Changes.Deleted[0].Path != "/alice/ops/old.chart" || got.Changes.Deleted[0].Type != "chart" {
		t.Fatalf("deleted snapshot lost: %+v", got.Changes.Deleted)
	}
	if got.Changes.Added[0].Path != "/alice/ops/import.form" {
		t.Fatalf("path fallback: %+v", got.Changes.Added)
	}
	if strings.Contains(string(log.DetailsJSON), "private source detail") || strings.Contains(string(log.DetailsJSON), "Existing directory") {
		t.Fatal("log must only include function identity snapshots")
	}
	if !strings.Contains(log.Summary, "1 added, 1 updated, 1 deleted") {
		t.Fatalf("summary: %s", log.Summary)
	}
}

func TestWorkspaceUpdateLogDistinguishesMissingAndEmptyDiff(t *testing.T) {
	missing := workspaceUpdateOperateLogDetailsFromRequest(nil, &dto.UpdateAppResp{}, time.Now(), nil)
	empty := workspaceUpdateOperateLogDetailsFromRequest(nil, &dto.UpdateAppResp{Diff: &dto.DiffData{}}, time.Now(), nil)
	if missing.Changes != nil || empty.Changes == nil || empty.Changes.Added == nil {
		t.Fatal("missing diff must not be represented as an empty diff")
	}
}

func TestWorkspaceUpdateLogForceDiffIsNotIncremental(t *testing.T) {
	api := &dto.ApiInfo{Name: "Existing", FullCodePath: "/alice/ops/existing.form"}
	resp := &dto.UpdateAppResp{Diff: &dto.DiffData{Add: []*dto.ApiInfo{api}, Update: []*dto.ApiInfo{api}}}
	got := workspaceUpdateOperateLogDetailsFromRequest(&dto.UpdateAppReq{ForceDiff: true}, resp, time.Now(), nil)
	if got.Changes.Mode != "resync" || len(got.Changes.Synced) != 1 || len(got.Changes.Added) != 0 || len(got.Changes.Updated) != 0 || len(got.Changes.Deleted) != 0 {
		t.Fatalf("force diff misrepresented: %+v", got.Changes)
	}
	if len(resp.Diff.Add) != 1 {
		t.Fatal("logging must not mutate the runtime diff")
	}
}

func TestWorkspaceUpdateLogOutcomes(t *testing.T) {
	for _, tt := range []struct {
		name    string
		req     *dto.UpdateAppReq
		resp    *dto.UpdateAppResp
		err     error
		outcome string
	}{
		{"runtime failure", nil, nil, errors.New("build failed"), "runtime_failed"},
		{"version persistence failure", nil, &dto.UpdateAppResp{NewVersion: "v10"}, errors.New("database unavailable"), "finalization_failed"},
		{"metadata warning", nil, &dto.UpdateAppResp{Warnings: []string{"sync failed"}}, nil, "metadata_warning"},
		{"callback warning", nil, &dto.UpdateAppResp{Error: "callback failed"}, nil, "metadata_warning"},
		{"source only", &dto.UpdateAppReq{WriteOnly: true}, &dto.UpdateAppResp{Diff: &dto.DiffData{}}, nil, "write_only"},
		{"completed", nil, &dto.UpdateAppResp{}, nil, "completed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := workspaceUpdateOperateLogDetailsFromRequest(tt.req, tt.resp, time.Now(), tt.err)
			if got.Outcome != tt.outcome {
				t.Fatalf("outcome: %s", got.Outcome)
			}
			if tt.err != nil && got.Error != tt.err.Error() {
				t.Fatal("error was not preserved")
			}
			if tt.outcome == "metadata_warning" && len(got.Warnings) != 1 {
				t.Fatal("warning was not preserved")
			}
			if tt.outcome == "write_only" && got.Changes != nil {
				t.Fatal("source-only writes must not report published changes")
			}
		})
	}
}
