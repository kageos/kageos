package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderBundledConfig(t *testing.T) {
	t.Parallel()

	prodDir := t.TempDir()
	paths := Paths{
		RepoRoot:     filepath.Dir(filepath.Dir(prodDir)),
		ProdDir:      prodDir,
		ConfigPath:   filepath.Join(prodDir, defaultConfigName),
		GeneratedDir: filepath.Join(prodDir, defaultGenerated),
	}
	cfg, err := defaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Site.BaseURL = "http://127.0.0.1"
	cfg.NATS.AuthEnabled = true

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateConfig(rt); err != nil {
		t.Fatal(err)
	}
	if err := renderAll(rt); err != nil {
		t.Fatal(err)
	}

	compose := mustReadFile(t, filepath.Join(paths.GeneratedDir, "docker-compose.yaml"))
	for _, want := range []string{
		`image: "docker.io/minio/minio:RELEASE.2025-09-07T16-13-09Z"`,
		`MYSQL_HOST: "mysql"`,
		`MINIO_HOST: "minio"`,
		`NATS_SEED_USER: "aos"`,
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("generated compose missing %q", want)
		}
	}

	mysqlInit := mustReadFile(t, filepath.Join(paths.GeneratedDir, "infra", "mysql-init.sql"))
	if !strings.Contains(mysqlInit, "CREATE DATABASE IF NOT EXISTS `app-scheduler`") {
		t.Fatalf("mysql init should quote database identifiers, got:\n%s", mysqlInit)
	}

	backupConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "backup-service.yaml"))
	for _, want := range []string{
		`listen_host: "0.0.0.0"`,
		`mysql_address: "mysql:3306"`,
		`minio_address: "minio:9000"`,
	} {
		if !strings.Contains(backupConfig, want) {
			t.Fatalf("generated backup config missing %q", want)
		}
	}

	appServerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "app-server.yaml"))
	if !strings.Contains(appServerConfig, `name: "app-scheduler"`) {
		t.Fatalf("generated app-server config should include scheduler database, got:\n%s", appServerConfig)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
