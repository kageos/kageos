package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
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

func TestBuildAppVersionSpecInjectsPinnedStartupEnv(t *testing.T) {
	t.Parallel()

	service := &AppManageService{
		config: &appconfig.AppManageServiceConfig{
			Build: appconfig.BuildConfig{
				BinaryNameFormat: "{user}-{app}-{version}",
			},
		},
		runtimeConfig: &appconfig.AppRuntimeConfig{
			Runtime: appconfig.RuntimeConfig{
				InstanceID: "rt-dev",
			},
			Container: appconfig.ContainerServiceConfig{
				Image: appconfig.ImageConfig{
					BaseImage:     "kagebase:test",
					ContainerPath: "/srv/app",
				},
			},
		},
	}

	spec, err := service.buildAppVersionSpec(context.Background(), AppVersionRef{User: "alice", App: "demo", Version: "v9"}, ".")
	if err != nil {
		t.Fatalf("buildAppVersionSpec: %v", err)
	}

	requireEnv(t, spec.EnvVars, "KAGEOS_APP_USER=alice")
	requireEnv(t, spec.EnvVars, "KAGEOS_APP_NAME=demo")
	requireEnv(t, spec.EnvVars, "APP_VERSION=v9")
	requireEnv(t, spec.EnvVars, "APP_BINARY_NAME=alice-demo-v9")
	requireEnv(t, spec.EnvVars, "KAGEOS_APP_WORK_DIR=/srv/app/workplace/bin")
	requireEnv(t, spec.EnvVars, "KAGEOS_APP_BIN_DIR=/srv/app/workplace/bin/releases")
	requireEnv(t, spec.EnvVars, "KAGEOS_RUNTIME_INSTANCE_ID=rt-dev")
}

func TestExtractAppRuntimeNATSSecretRemovesURLUserInfoFromEnv(t *testing.T) {
	t.Parallel()

	rawURL := "nats://app-user:app-password@host.containers.internal:4222"
	envVars := map[string]string{"NATS_URL": rawURL}
	secrets, err := extractAppRuntimeNATSSecret(
		AppVersionRef{User: "alice", App: "demo", Version: "v9"},
		envVars,
	)
	if err != nil {
		t.Fatalf("extractAppRuntimeNATSSecret() error = %v", err)
	}
	if got := envVars["NATS_URL"]; got != "nats://host.containers.internal:4222" {
		t.Fatalf("NATS_URL = %q, want credential-free endpoint", got)
	}
	if len(secrets) != 1 {
		t.Fatalf("secret count = %d, want 1", len(secrets))
	}
	secret := secrets[0]
	if secret.Target != appNATSCredentialsSecretTarget {
		t.Fatalf("secret target = %q, want %q", secret.Target, appNATSCredentialsSecretTarget)
	}
	if string(secret.Data) != rawURL {
		t.Fatal("runtime secret did not preserve the private NATS URL")
	}
	if strings.Contains(envVars["NATS_URL"], "app-user") || strings.Contains(envVars["NATS_URL"], "app-password") {
		t.Fatalf("NATS_URL leaked credential material: %q", envVars["NATS_URL"])
	}
}

func TestExtractAppRuntimeNATSSecretKeepsNoAuthCompatibility(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{"NATS_URL": "nats://host.containers.internal:4222"}
	secrets, err := extractAppRuntimeNATSSecret(
		AppVersionRef{User: "alice", App: "demo", Version: "v9"},
		envVars,
	)
	if err != nil {
		t.Fatalf("extractAppRuntimeNATSSecret() error = %v", err)
	}
	if len(secrets) != 0 {
		t.Fatalf("secret count = %d, want 0 for no-auth endpoint", len(secrets))
	}
	if got := envVars["NATS_URL"]; got != "nats://host.containers.internal:4222" {
		t.Fatalf("NATS_URL changed unexpectedly: %q", got)
	}
}

func TestExtractAppRuntimeNATSSecretHandlesServerList(t *testing.T) {
	t.Parallel()

	rawURL := "nats://user:pass@nats-a:4222,tls://token@nats-b:4222"
	envVars := map[string]string{"NATS_URL": rawURL}
	secrets, err := extractAppRuntimeNATSSecret(
		AppVersionRef{User: "alice", App: "demo", Version: "v9"},
		envVars,
	)
	if err != nil {
		t.Fatalf("extractAppRuntimeNATSSecret() error = %v", err)
	}
	if got, want := envVars["NATS_URL"], "nats://nats-a:4222,tls://nats-b:4222"; got != want {
		t.Fatalf("NATS_URL = %q, want %q", got, want)
	}
	if len(secrets) != 1 || string(secrets[0].Data) != rawURL {
		t.Fatal("expected one private URL secret for the server list")
	}
}

func TestPrepareAppRuntimeNATSSecretUsesSecretForCurrentSDKBinary(t *testing.T) {
	t.Parallel()

	binaryPath := filepath.Join(t.TempDir(), "current-sdk-app")
	binaryContents := append([]byte("compiled-app-prefix\x00"), []byte(appNATSCredentialsSDKMarker)...)
	if err := os.WriteFile(binaryPath, binaryContents, 0o600); err != nil {
		t.Fatalf("write current SDK binary fixture: %v", err)
	}

	rawURL := "nats://app-user:app-password@nats.internal:4222"
	envVars := map[string]string{"NATS_URL": rawURL}
	secrets, err := prepareAppRuntimeNATSSecret(
		context.Background(),
		AppVersionRef{User: "alice", App: "demo", Version: "v9"},
		binaryPath,
		envVars,
	)
	if err != nil {
		t.Fatalf("prepareAppRuntimeNATSSecret() error = %v", err)
	}
	if got := envVars["NATS_URL"]; got != "nats://nats.internal:4222" {
		t.Fatalf("NATS_URL = %q, want credential-free endpoint", got)
	}
	if len(secrets) != 1 || string(secrets[0].Data) != rawURL {
		t.Fatal("current SDK binary should receive one private NATS secret")
	}
}

func TestPrepareAppRuntimeNATSSecretKeepsLegacyURLForOldBinary(t *testing.T) {
	t.Parallel()

	binaryPath := filepath.Join(t.TempDir(), "old-sdk-app")
	if err := os.WriteFile(binaryPath, []byte("compiled-with-an-old-sdk"), 0o600); err != nil {
		t.Fatalf("write old SDK binary fixture: %v", err)
	}

	rawURL := "nats://legacy-user:legacy-password@nats.internal:4222"
	envVars := map[string]string{"NATS_URL": rawURL}
	secrets, err := prepareAppRuntimeNATSSecret(
		context.Background(),
		AppVersionRef{User: "alice", App: "legacy", Version: "v1"},
		binaryPath,
		envVars,
	)
	if err != nil {
		t.Fatalf("prepareAppRuntimeNATSSecret() error = %v", err)
	}
	if len(secrets) != 0 {
		t.Fatalf("legacy binary secret count = %d, want 0", len(secrets))
	}
	if envVars["NATS_URL"] != rawURL {
		t.Fatal("legacy NATS_URL changed unexpectedly")
	}
}

func TestPrepareAppRuntimeNATSSecretKeepsLegacyURLWhenBinaryUnreadable(t *testing.T) {
	t.Parallel()

	rawURL := "nats://legacy-user:legacy-password@nats.internal:4222"
	envVars := map[string]string{"NATS_URL": rawURL}
	secrets, err := prepareAppRuntimeNATSSecret(
		context.Background(),
		AppVersionRef{User: "alice", App: "legacy", Version: "v1"},
		filepath.Join(t.TempDir(), "missing-app-binary"),
		envVars,
	)
	if err != nil {
		t.Fatalf("prepareAppRuntimeNATSSecret() error = %v", err)
	}
	if len(secrets) != 0 || envVars["NATS_URL"] != rawURL {
		t.Fatal("unreadable binary should retain legacy NATS_URL without a secret")
	}
}

func TestAppBinaryContainsMarkerAcrossScanBoundary(t *testing.T) {
	t.Parallel()

	binaryPath := filepath.Join(t.TempDir(), "boundary-app")
	prefix := bytes.Repeat([]byte{'x'}, appBinaryMarkerScanBufferSize-5)
	contents := append(prefix, []byte(appNATSCredentialsSDKMarker)...)
	if err := os.WriteFile(binaryPath, contents, 0o600); err != nil {
		t.Fatalf("write boundary binary fixture: %v", err)
	}

	found, err := appBinaryContainsMarker(binaryPath, []byte(appNATSCredentialsSDKMarker))
	if err != nil {
		t.Fatalf("appBinaryContainsMarker() error = %v", err)
	}
	if !found {
		t.Fatal("expected marker split across scan chunks to be found")
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

func requireEnv(t *testing.T, envVars []string, want string) {
	t.Helper()

	for _, envVar := range envVars {
		if envVar == want {
			return
		}
	}
	t.Fatalf("missing env %q in %#v", want, envVars)
}
