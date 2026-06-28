package buildtrace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildTraceRecordsAndPersistsSpans(t *testing.T) {
	ctx, trace := Ensure(context.Background(), "runtime.update_app", "alice", "demo")

	span := Start(ctx, "builder.go_build", String("package", "./code/cmd/app"))
	span.Finish(nil)
	trace.Finalize(nil)

	dir := t.TempDir()
	path, err := Persist(trace, dir)
	if err != nil {
		t.Fatalf("Persist error = %v", err)
	}

	if filepath.Base(path) == "latest.json" {
		t.Fatalf("expected trace-specific file, got %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected trace file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "latest.json")); err != nil {
		t.Fatalf("expected latest trace file: %v", err)
	}

	snapshot := trace.Snapshot()
	if snapshot.StoragePath != path {
		t.Fatalf("storage path = %q, want %q", snapshot.StoragePath, path)
	}
	if snapshot.DurationMS <= 0 {
		t.Fatalf("expected positive duration: %+v", snapshot)
	}
	if len(snapshot.Spans) != 1 || snapshot.Spans[0].Name != "builder.go_build" {
		t.Fatalf("unexpected spans: %+v", snapshot.Spans)
	}
	if got := Summary(snapshot, 1); !strings.Contains(got, "builder.go_build=") {
		t.Fatalf("summary should include slowest span, got %q", got)
	}
}

func TestBuildTraceRecordsErrors(t *testing.T) {
	ctx, trace := Ensure(context.Background(), "runtime.update_app", "alice", "demo")

	errBoom := errors.New("boom")
	span := Start(ctx, "builder.go_mod_tidy")
	span.Finish(errBoom)
	snapshot := trace.Finalize(errBoom)

	if snapshot.Status != statusError || snapshot.Error != "boom" {
		t.Fatalf("unexpected trace error: %+v", snapshot)
	}
	if len(snapshot.Spans) != 1 || snapshot.Spans[0].Status != statusError || snapshot.Spans[0].Error != "boom" {
		t.Fatalf("unexpected span error: %+v", snapshot.Spans)
	}
}
