package infra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setKageosWorkspaceEnv(t *testing.T, content string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("KAGEOS_ROOT", root)
	t.Setenv("KAGEOS_DEV_ENGINE", "")
	if err := os.MkdirAll(filepath.Join(root, ".kageos"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".kageos", "kageos.env"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func stubPodmanClockSync(t *testing.T, fn func(context.Context) error) *int {
	t.Helper()
	oldSync := syncPodmanMachineClock
	oldAvailable := podmanCommandAvailable
	calls := 0
	syncPodmanMachineClock = func(ctx context.Context) error {
		calls++
		return fn(ctx)
	}
	podmanCommandAvailable = func() bool { return true }
	t.Cleanup(func() {
		syncPodmanMachineClock = oldSync
		podmanCommandAvailable = oldAvailable
	})
	return &calls
}

func TestCheckMinIOClockSkewWithDevPodmanRepairRetriesAfterRepair(t *testing.T) {
	setKageosWorkspaceEnv(t, "KAGEOS_MODE=dev\nKAGEOS_DEV_ENGINE=podman\n")

	localTime := time.Date(2026, 6, 6, 8, 40, 0, 0, time.UTC)
	serverTime := localTime.Add(-16 * time.Minute)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", serverTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	calls := stubPodmanClockSync(t, func(context.Context) error {
		serverTime = localTime
		return nil
	})

	err := CheckMinIOClockSkewWithDevPodmanRepair(context.Background(), ts.URL, 15*time.Minute, func() time.Time {
		return localTime
	})
	if err != nil {
		t.Fatalf("CheckMinIOClockSkewWithDevPodmanRepair() unexpected error: %v", err)
	}
	if *calls != 1 {
		t.Fatalf("podman clock sync calls = %d, want 1", *calls)
	}
}

func TestCheckMinIOClockSkewWithDevPodmanRepairSkipsDockerDev(t *testing.T) {
	setKageosWorkspaceEnv(t, "KAGEOS_MODE=dev\nKAGEOS_DEV_ENGINE=docker\n")

	serverTime := time.Date(2026, 6, 6, 8, 0, 0, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", serverTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	calls := stubPodmanClockSync(t, func(context.Context) error {
		t.Fatal("podman clock sync should not run for docker dev")
		return nil
	})

	err := CheckMinIOClockSkewWithDevPodmanRepair(context.Background(), ts.URL, 15*time.Minute, func() time.Time {
		return serverTime.Add(16 * time.Minute)
	})
	if !IsMinIOClockSkewError(err) {
		t.Fatalf("expected MinIOClockSkewError, got %T: %v", err, err)
	}
	if *calls != 0 {
		t.Fatalf("podman clock sync calls = %d, want 0", *calls)
	}
}

func TestShouldAttemptDevPodmanClockRepairRequiresLocalURL(t *testing.T) {
	setKageosWorkspaceEnv(t, "KAGEOS_MODE=dev\nKAGEOS_DEV_ENGINE=podman\n")
	calls := stubPodmanClockSync(t, func(context.Context) error { return nil })

	if shouldAttemptDevPodmanClockRepair("https://minio.example.com/minio/health/ready") {
		t.Fatal("expected no repair attempt for non-local MinIO URL")
	}
	if *calls != 0 {
		t.Fatalf("podman clock sync calls = %d, want 0", *calls)
	}
}
