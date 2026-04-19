package service

import (
	"context"
	"strings"
	"testing"
)

func TestBuildWriteOnlyUpdateRespKeepsCurrentVersion(t *testing.T) {
	t.Parallel()

	service := &AppManageService{}
	resp := service.buildWriteOnlyUpdateResp(context.Background(), "alice", "demo", "v3")
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.User != "alice" || resp.App != "demo" {
		t.Fatalf("unexpected identity: %+v", resp)
	}
	if resp.OldVersion != "v3" || resp.NewVersion != "v3" {
		t.Fatalf("expected old/new version to stay on v3, got old=%s new=%s", resp.OldVersion, resp.NewVersion)
	}
}

func TestNoteUnknownUpdateVersionAppendsLogMarker(t *testing.T) {
	t.Parallel()

	service := &AppManageService{}
	var logStr strings.Builder

	service.noteUnknownUpdateVersion(&updateAppState{oldVersion: "unknown"}, &logStr)
	if got := logStr.String(); !strings.Contains(got, "Failed to get current version") {
		t.Fatalf("expected unknown-version marker in log, got %q", got)
	}

	logStr.Reset()
	service.noteUnknownUpdateVersion(&updateAppState{oldVersion: "v2"}, &logStr)
	if got := logStr.String(); got != "" {
		t.Fatalf("expected no marker for known version, got %q", got)
	}
}
