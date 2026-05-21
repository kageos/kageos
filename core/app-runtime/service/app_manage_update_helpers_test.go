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

func TestCreateVersionContainerReusesRunningRuntime(t *testing.T) {
	t.Parallel()

	driver := &stubAppRuntimeDriver{running: true}
	service := &AppManageService{runtimeDriver: driver}

	if err := service.createVersionContainer(context.Background(), "system", "tools", "v3", "."); err != nil {
		t.Fatalf("expected running runtime to be reused, got error: %v", err)
	}
	if driver.createCalled {
		t.Fatal("expected create to be skipped when runtime is already running")
	}
}

type stubAppRuntimeDriver struct {
	running      bool
	createCalled bool
}

func (d *stubAppRuntimeDriver) IsAvailable() bool {
	return true
}

func (d *stubAppRuntimeDriver) CreateAppVersion(context.Context, AppVersionSpec) error {
	d.createCalled = true
	return nil
}

func (d *stubAppRuntimeDriver) StartAppVersion(context.Context, AppVersionSpec) error {
	return nil
}

func (d *stubAppRuntimeDriver) StopAppVersion(context.Context, AppVersionRef) error {
	return nil
}

func (d *stubAppRuntimeDriver) RemoveAppVersion(context.Context, AppVersionRef) error {
	return nil
}

func (d *stubAppRuntimeDriver) IsAppVersionRunning(context.Context, AppVersionRef) (bool, error) {
	return d.running, nil
}

func (d *stubAppRuntimeDriver) ListAppVersions(context.Context) ([]AppRuntimeInstance, error) {
	return nil, nil
}
