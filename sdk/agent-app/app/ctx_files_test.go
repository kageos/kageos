package app

import (
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/storage"
	"github.com/kageos/kageos/pkg/trace"
)

func TestResolvedDownloadFilePreferredDownloadURL(t *testing.T) {
	file := resolvedDownloadFile{
		downloadURL:       "http://browser.example/file",
		serverDownloadURL: "http://server.example/file",
	}
	if got := file.preferredDownloadURL(); got != "http://server.example/file" {
		t.Fatalf("expected server download URL, got %s", got)
	}

	file.serverDownloadURL = ""
	if got := file.preferredDownloadURL(); got != "http://browser.example/file" {
		t.Fatalf("expected browser download URL fallback, got %s", got)
	}
}

func TestResolvedDownloadFileTargetFileName(t *testing.T) {
	if got := (resolvedDownloadFile{name: "report.xlsx", key: "objects/fallback.txt"}).targetFileName(); got != "report.xlsx" {
		t.Fatalf("expected explicit name, got %s", got)
	}
	if got := (resolvedDownloadFile{key: "objects/fallback.txt"}).targetFileName(); got != "fallback.txt" {
		t.Fatalf("expected key basename fallback, got %s", got)
	}
}

func TestCompactNonEmptyStrings(t *testing.T) {
	got := compactNonEmptyStrings([]string{"a", "", "b", ""})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected compact result: %#v", got)
	}
}

func TestBuildBatchUploadTokenReq(t *testing.T) {
	ctx := &Context{
		msg: &trace.Msg{
			User:   "alice",
			App:    "demo",
			Router: "/tools/export",
		},
	}
	req := ctx.buildBatchUploadTokenReq([]*FileInfo{{
		FileName:    "report.csv",
		FileSize:    42,
		ContentType: "text/csv",
		Hash:        "sha256",
	}})

	if req.UploadSource != dto.UploadSourceServer {
		t.Fatalf("expected server upload source, got %s", req.UploadSource)
	}
	if len(req.Files) != 1 {
		t.Fatalf("expected one file request, got %d", len(req.Files))
	}
	fileReq := req.Files[0]
	if fileReq.Router != "/alice/demo/tools/export" {
		t.Fatalf("unexpected router: %s", fileReq.Router)
	}
	if fileReq.UploadSource != dto.UploadSourceServer {
		t.Fatalf("expected file server upload source, got %s", fileReq.UploadSource)
	}
	if fileReq.FileName != "report.csv" || fileReq.FileSize != 42 || fileReq.ContentType != "text/csv" || fileReq.Hash != "sha256" {
		t.Fatalf("unexpected file request: %#v", fileReq)
	}
}

func TestUploadRefFallbacks(t *testing.T) {
	withRef := &fileUploadResult{
		cred: &dto.GetUploadTokenResp{
			Ref:    "bucket/custom-ref",
			Bucket: "bucket",
			Key:    "object",
		},
	}
	if got := refFromUploadResult(withRef); got != "bucket/custom-ref" {
		t.Fatalf("expected credential ref, got %s", got)
	}

	withoutRef := &fileUploadResult{
		cred: &dto.GetUploadTokenResp{
			Bucket: "bucket",
			Key:    "object",
		},
	}
	if got := refFromUploadResult(withoutRef); got != "bucket/object" {
		t.Fatalf("expected joined ref fallback, got %s", got)
	}
}

func TestAppendCompletedUploadRefs(t *testing.T) {
	uploadResultMap := map[string]*fileUploadResult{
		"completed-key": {
			cred: &dto.GetUploadTokenResp{
				Ref:    "bucket/fallback",
				Bucket: "bucket",
				Key:    "completed-key",
			},
			result: &storage.UploadResult{Key: "completed-key"},
		},
	}

	got := appendCompletedUploadRefs(nil, []dto.BatchUploadCompleteResult{
		{Key: "completed-key", Status: "failed", Ref: "bucket/ignored"},
		{Key: "missing-key", Status: "completed", Ref: "bucket/missing"},
		{Key: "completed-key", Status: "completed", Ref: "bucket/from-api"},
	}, uploadResultMap)

	if len(got) != 1 || got[0] != "bucket/from-api" {
		t.Fatalf("unexpected completed refs: %#v", got)
	}
}
