package service

import (
	"context"
	"strings"
	"testing"
	"time"
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

func TestWaitForStartupNotificationReturnsNotification(t *testing.T) {
	t.Parallel()

	service := &AppManageService{runtimeDriver: &stubAppRuntimeDriver{running: true}}
	waiterChan := make(chan *StartupNotification, 1)
	want := &StartupNotification{User: "system", App: "demo", Version: "v1", Status: "running", StartTime: time.Now()}
	waiterChan <- want

	got, err := service.waitForStartupNotificationOrRuntimeExit(
		context.Background(),
		AppVersionRef{User: "system", App: "demo", Version: "v1"},
		waiterChan,
		50*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("waitForStartupNotificationOrRuntimeExit() error = %v", err)
	}
	if got != want {
		t.Fatalf("expected original notification, got %#v", got)
	}
}

func TestWaitForStartupNotificationFailsWhenRuntimeExitsFirst(t *testing.T) {
	t.Parallel()

	service := &AppManageService{runtimeDriver: &stubAppRuntimeDriver{running: false}}
	_, err := service.waitForStartupNotificationOrRuntimeExit(
		context.Background(),
		AppVersionRef{User: "system", App: "demo", Version: "v1"},
		make(chan *StartupNotification),
		50*time.Millisecond,
	)
	if err == nil {
		t.Fatal("expected runtime exit error, got nil")
	}
	if !strings.Contains(err.Error(), "exited before startup notification") {
		t.Fatalf("expected runtime exit error, got %v", err)
	}
}

func TestWaitForStartupNotificationTimesOutWithoutTreatingRunningRuntimeAsStarted(t *testing.T) {
	t.Parallel()

	service := &AppManageService{runtimeDriver: &stubAppRuntimeDriver{running: true}}
	_, err := service.waitForStartupNotificationOrRuntimeExit(
		context.Background(),
		AppVersionRef{User: "system", App: "demo", Version: "v1"},
		make(chan *StartupNotification),
		20*time.Millisecond,
	)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout waiting for app startup notification") {
		t.Fatalf("expected startup timeout, got %v", err)
	}
	if !strings.Contains(err.Error(), "runtime still running") {
		t.Fatalf("expected runtime-still-running detail, got %v", err)
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
