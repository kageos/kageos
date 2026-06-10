package app

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
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

func TestResolvedDownloadFileDownloadCandidatesPreferServer(t *testing.T) {
	file := resolvedDownloadFile{
		downloadURL:       "http://browser.example/file",
		serverDownloadURL: "http://server.example/file",
	}
	candidates := file.downloadCandidates()
	if len(candidates) != 2 {
		t.Fatalf("expected two candidates, got %#v", candidates)
	}
	if candidates[0].label != "server" || candidates[0].url != "http://server.example/file" {
		t.Fatalf("unexpected first candidate: %#v", candidates[0])
	}
	if candidates[1].label != "browser" || candidates[1].url != "http://browser.example/file" {
		t.Fatalf("unexpected second candidate: %#v", candidates[1])
	}

	file.downloadURL = file.serverDownloadURL
	candidates = file.downloadCandidates()
	if len(candidates) != 1 || candidates[0].label != "server" {
		t.Fatalf("expected duplicate URL to be de-duplicated, got %#v", candidates)
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

func TestDownloadResolvedFilesReportsMissingURL(t *testing.T) {
	fs := &FS{
		ctx: &Context{
			msg: &trace.Msg{TraceId: "trace-1"},
		},
		fileCache: GetFileCache(),
	}

	paths, stats, issues := fs.downloadResolvedFiles([]resolvedDownloadFile{{
		ref:          "kageos/workspace/chat/a.png",
		key:          "workspace/chat/a.png",
		errorMessage: "object not found",
	}}, t.TempDir())
	if got := compactNonEmptyStrings(paths); len(got) != 0 {
		t.Fatalf("expected no paths, got %#v", paths)
	}
	if stats.skipCount != 1 || stats.downloadCount != 0 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if len(issues) != 1 || issues[0] == "" {
		t.Fatalf("expected one issue, got %#v", issues)
	}
}

func TestSummarizeDownloadErrorRedactsPresignedURL(t *testing.T) {
	err := fmt.Errorf("下载请求失败: %w", &url.Error{
		Op:  "Get",
		URL: "http://storage.example/kageos/a.png?X-Amz-Signature=secret",
		Err: errors.New("connection refused"),
	})
	got := summarizeDownloadError(err)
	if strings.Contains(got, "X-Amz-Signature") || strings.Contains(got, "storage.example") {
		t.Fatalf("expected redacted error, got %q", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Fatalf("expected root error, got %q", got)
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
