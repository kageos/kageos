package systembackup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigRoundTripSealsSecret(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Bucket = "backup-bucket"
	cfg.AccessKeyID = "access"
	cfg.SecretAccessKey = "very-secret"
	if err := SaveConfig(dir, "jwt-secret", cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "very-secret") {
		t.Fatalf("stored config contains plaintext secret: %s", raw)
	}
	loaded, err := LoadConfig(dir, "jwt-secret")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.SecretAccessKey != "very-secret" || !loaded.SecretAccessKeySet {
		t.Fatalf("unexpected loaded secret state: %#v", loaded)
	}
	if PublicConfig(loaded).SecretAccessKey != "" {
		t.Fatal("public config must not expose the secret")
	}
}

func TestDueRunsOncePerDayAndHonorsManualRequest(t *testing.T) {
	now := time.Date(2026, 9, 1, 4, 0, 0, 0, time.Local)
	cfg := DefaultConfig()
	cfg.Enabled = true
	if due, trigger := Due(cfg, State{}, now); !due || trigger != "schedule" {
		t.Fatalf("expected scheduled run, got due=%v trigger=%q", due, trigger)
	}
	state := State{Records: []Record{{TriggeredBy: "schedule", StartedAt: now.Add(-time.Hour).UTC().Format(time.RFC3339)}}}
	if due, _ := Due(cfg, state, now); due {
		t.Fatal("same-day scheduled run must not repeat")
	}
	cfg.RunNowRequestedAt = now.UTC().Format(time.RFC3339Nano)
	if due, trigger := Due(cfg, state, now); !due || trigger != "manual" {
		t.Fatalf("manual request should override daily schedule, got due=%v trigger=%q", due, trigger)
	}
}

func TestValidateConfigAllowsAWSWithoutEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Bucket = "backup-bucket"
	cfg.AccessKeyID = "access"
	cfg.SecretAccessKey = "secret"
	if err := ValidateConfig(cfg, true); err != nil {
		t.Fatalf("AWS config without endpoint should be valid: %v", err)
	}
	endpoint, secure, err := endpointFor(cfg)
	if err != nil || endpoint != "s3.amazonaws.com" || !secure {
		t.Fatalf("unexpected AWS endpoint: endpoint=%q secure=%v err=%v", endpoint, secure, err)
	}
}

func TestPruneLocalKeepsNewestArchives(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"kageos-backup-20260901T010000Z.tar.gz", "kageos-backup-20260902T010000Z.tar.gz", "kageos-backup-20260903T010000Z.tar.gz"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := PruneLocal(dir, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "kageos-backup-20260901T010000Z.tar.gz")); !os.IsNotExist(err) {
		t.Fatal("oldest backup should be deleted")
	}
}
