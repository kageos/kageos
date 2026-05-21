package main

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	cfg.LLMs = LLMSeedsConfig{
		Default: "main",
		Configs: []LLMSeedConfig{
			{
				Code:      "main",
				Name:      "默认模型",
				Model:     "gpt-4o-mini",
				APIKeyEnv: "OPENAI_API_KEY",
				Timeout:   300,
				MaxTokens: 8192,
				Admin:     "system",
			},
		},
	}

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
		`MYSQL_HOST: "127.0.0.1"`,
		`MINIO_HOST: "127.0.0.1"`,
		`NATS_URL: "nats://aos:`,
		`NATS_SEED_USER: "aos"`,
		`NATS_SEED_PASSWORD: "`,
		`SYSTEM_USER_PASSWORD: "` + cfg.SystemUser.Password + `"`,
		`OPENAI_API_KEY: "${OPENAI_API_KEY:-}"`,
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("generated compose missing %q", want)
		}
	}

	mysqlInit := mustReadFile(t, filepath.Join(paths.GeneratedDir, "infra", "mysql-init.sql"))
	if !strings.Contains(mysqlInit, "CREATE DATABASE IF NOT EXISTS `app_db`") {
		t.Fatalf("mysql init should quote database identifiers, got:\n%s", mysqlInit)
	}

	appServerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "app-server.yaml"))
	if strings.Contains(appServerConfig, `scheduled_task_db`) {
		t.Fatalf("generated app-server config should not include scheduled task database, got:\n%s", appServerConfig)
	}

	appRuntimeConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "app-runtime.yaml"))
	if !strings.Contains(appRuntimeConfig, `app_startup_notification: 300`) {
		t.Fatalf("generated app-runtime config should include startup notification timeout, got:\n%s", appRuntimeConfig)
	}

	globalConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "global.yaml"))
	if strings.Contains(globalConfig, `timer_scheduler`) {
		t.Fatalf("generated global config should not include timer scheduler config, got:\n%s", globalConfig)
	}
	for _, want := range []string{
		`nats_url: "nats://aos:`,
		`@127.0.0.1:4222"`,
		`gateway_url: "http://127.0.0.1:9090"`,
	} {
		if !strings.Contains(globalConfig, want) {
			t.Fatalf("generated global SDK config missing %q, got:\n%s", want, globalConfig)
		}
	}

	agentServerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "agent-server.yaml"))
	for _, want := range []string{
		`llms:`,
		`default: "main"`,
		`code: "main"`,
		`api_key_env: "OPENAI_API_KEY"`,
		`api_base: ""`,
	} {
		if !strings.Contains(agentServerConfig, want) {
			t.Fatalf("generated agent-server config missing %q, got:\n%s", want, agentServerConfig)
		}
	}

	appStorageConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "app-storage.yaml"))
	if !strings.Contains(appStorageConfig, `server_endpoint: "127.0.0.1:9000"`) {
		t.Fatalf("generated app-storage config should include container MinIO endpoint, got:\n%s", appStorageConfig)
	}

	hrServerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "hr-server.yaml"))
	for _, want := range []string{
		`system_user:`,
		`password: "` + cfg.SystemUser.Password + `"`,
	} {
		if !strings.Contains(hrServerConfig, want) {
			t.Fatalf("generated hr-server config missing %q, got:\n%s", want, hrServerConfig)
		}
	}

	apiGatewayConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "api-gateway.yaml"))
	for _, retired := range []string{`path: "/message"`, `path: "/control"`} {
		if strings.Contains(apiGatewayConfig, retired) {
			t.Fatalf("generated api-gateway config should not include retired route %q, got:\n%s", retired, apiGatewayConfig)
		}
	}
}

func TestRenderTLSFromBase64Config(t *testing.T) {
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
	certPEM := "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----\n"
	keyPEM := "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----\n"
	cfg.Site.BaseURL = "https://example.com"
	cfg.Site.TLSMode = "redirect"
	cfg.Site.TLSCertPEMB64 = base64.StdEncoding.EncodeToString([]byte(certPEM))
	cfg.Site.TLSKeyPEMB64 = base64.StdEncoding.EncodeToString([]byte(keyPEM))

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
	if !strings.Contains(compose, filepath.Join(paths.GeneratedDir, "tls")+":/app/tls:ro") {
		t.Fatalf("generated compose should mount generated tls dir, got:\n%s", compose)
	}

	if got := mustReadFile(t, filepath.Join(paths.GeneratedDir, "tls", "fullchain.pem")); got != certPEM {
		t.Fatalf("generated cert = %q, want %q", got, certPEM)
	}
	if got := mustReadFile(t, filepath.Join(paths.GeneratedDir, "tls", "privkey.pem")); got != keyPEM {
		t.Fatalf("generated key = %q, want %q", got, keyPEM)
	}

	envFile := mustReadFile(t, filepath.Join(paths.GeneratedDir, "env", "kageos.env"))
	for _, want := range []string{
		"CANONICAL_BASE_URL=https://example.com",
		"TLS_MODE=redirect",
		"KAGEOS_TLS_CERT_PEM_B64=" + cfg.Site.TLSCertPEMB64,
		"KAGEOS_TLS_KEY_PEM_B64=" + cfg.Site.TLSKeyPEMB64,
	} {
		if !strings.Contains(envFile, want) {
			t.Fatalf("generated env file missing %q, got:\n%s", want, envFile)
		}
	}
}

func TestRenderExternalNATSKeepsSDKURL(t *testing.T) {
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
	cfg.NATS.Mode = "external"
	cfg.NATS.URL = "nats://nats.example.com:4222"

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

	globalConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "global.yaml"))
	if !strings.Contains(globalConfig, `nats_url: "nats://nats.example.com:4222"`) {
		t.Fatalf("generated SDK config should preserve external NATS URL, got:\n%s", globalConfig)
	}
}

func TestDeploymentLayersExposeRuntimeBoundary(t *testing.T) {
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

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}

	components := deploymentComponents(rt)
	for _, want := range []struct {
		layer deploymentLayerID
		name  string
	}{
		{layer: layerEdge, name: "nginx"},
		{layer: layerPlatform, name: "core-server"},
		{layer: layerRuntime, name: "podman-api"},
		{layer: layerRuntime, name: defaultAppBaseImage},
		{layer: layerApps, name: "user-app containers"},
	} {
		if !hasDeploymentComponent(components, want.layer, want.name) {
			t.Fatalf("deployment components missing %s/%s: %#v", want.layer, want.name, components)
		}
	}
}

func TestComposeServicesByLayer(t *testing.T) {
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

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		layer deploymentLayerID
		want  []string
	}{
		{layer: layerInfra, want: []string{"mysql", "nats", "minio"}},
		{layer: layerEdge, want: []string{"main"}},
		{layer: layerPlatform, want: []string{"main"}},
		{layer: layerRuntime, want: []string{"main"}},
		{layer: layerApps, want: nil},
	} {
		got := composeServicesForLayer(rt, tc.layer)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Fatalf("unexpected compose services for %s: got=%v want=%v", tc.layer, got, tc.want)
		}
	}
}

func TestDeploymentReportIncludesComposeOwnership(t *testing.T) {
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

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}

	report := buildDeploymentReport(rt)
	if len(report.Layers) != 6 {
		t.Fatalf("unexpected layer count: %d", len(report.Layers))
	}
	for _, layer := range report.Layers {
		if layer.ID == layerPlatform {
			if strings.Join(layer.ComposeServices, ",") != "main" {
				t.Fatalf("unexpected platform compose services: %#v", layer.ComposeServices)
			}
			return
		}
	}
	t.Fatal("platform layer not found")
}

func TestParseDeploymentLayerAliases(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		value string
		want  deploymentLayerID
	}{
		{value: "L0", want: layerControl},
		{value: "infra", want: layerInfra},
		{value: "edge", want: layerEdge},
		{value: "platform", want: layerPlatform},
		{value: "runtime", want: layerRuntime},
		{value: "apps", want: layerApps},
	} {
		got, ok := parseDeploymentLayer(tc.value)
		if !ok || got != tc.want {
			t.Fatalf("parseDeploymentLayer(%q)=%q,%v want=%q,true", tc.value, got, ok, tc.want)
		}
	}
}

func TestParseOutputFlags(t *testing.T) {
	t.Parallel()

	opts, rest, err := parseOutputFlags("verify", []string{"--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.JSON || len(rest) != 0 {
		t.Fatalf("unexpected output opts: opts=%#v rest=%#v", opts, rest)
	}

	if _, _, err := parseOutputFlags("verify", []string{"--bad"}); err == nil {
		t.Fatal("expected unknown output flag to fail")
	}
}

func TestParseUpFlags(t *testing.T) {
	t.Parallel()

	opts, err := parseUpFlags([]string{"--image", "--skip-verify", "--wait-timeout", "30s"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.UseImage || !opts.SkipVerify || opts.VerifyTimeout != 30*time.Second {
		t.Fatalf("unexpected up opts: %#v", opts)
	}

	if _, err := parseUpFlags([]string{"--image", "--no-build"}); err == nil {
		t.Fatal("expected --image and --no-build conflict")
	}
	if _, err := parseUpFlags([]string{"--wait-timeout", "0s"}); err == nil {
		t.Fatal("expected non-positive timeout to fail")
	}
}

func TestParseUninstallFlags(t *testing.T) {
	t.Parallel()

	opts, err := parseUninstallFlags([]string{"--purge-data", "--force", "--keep-generated"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.PurgeData || !opts.Force || !opts.KeepGenerated {
		t.Fatalf("unexpected uninstall opts: %#v", opts)
	}

	if _, err := parseUninstallFlags([]string{"--purge-data"}); err == nil {
		t.Fatal("expected destructive uninstall to require --force")
	}
	if _, err := parseUninstallFlags([]string{"--purge-data", "--dry-run"}); err != nil {
		t.Fatalf("dry-run should allow destructive preview: %v", err)
	}
	if _, err := parseUninstallFlags([]string{"--purge-podman-storage", "--force"}); err == nil {
		t.Fatal("expected podman storage purge to require --purge-data")
	}
}

func TestUninstallDataTargetsKeepPodmanStorageByDefault(t *testing.T) {
	t.Parallel()

	rt := RuntimeConfig{
		Config: Config{
			Storage: StorageConfig{Root: "/data/kageos"},
		},
	}
	targets := uninstallDataTargets(rt, uninstallOptions{PurgeData: true})
	got := make([]string, 0, len(targets))
	for _, target := range targets {
		got = append(got, target.Path)
	}
	joined := strings.Join(got, ",")
	if strings.Contains(joined, "podman_storage") {
		t.Fatalf("podman storage should be kept by default, got %v", got)
	}

	targets = uninstallDataTargets(rt, uninstallOptions{PurgeData: true, PurgePodmanStorage: true})
	got = got[:0]
	for _, target := range targets {
		got = append(got, target.Path)
	}
	if !strings.Contains(strings.Join(got, ","), "/data/kageos/podman_storage") {
		t.Fatalf("podman storage should be included when explicitly requested, got %v", got)
	}
}

func TestParseInitAndBootstrapFlags(t *testing.T) {
	t.Parallel()

	initOpts, err := parseInitFlags("init", []string{"--force", "--base-url", "http://example.com", "--mysql-mode", "bundled"})
	if err != nil {
		t.Fatal(err)
	}
	if !initOpts.Force || initOpts.BaseURL != "http://example.com" || initOpts.MySQLMode != "bundled" {
		t.Fatalf("unexpected init opts: %#v", initOpts)
	}

	bootstrapOpts, err := parseBootstrapFlags([]string{"--base-url", "http://example.com", "--skip-verify", "--wait-timeout", "30s"})
	if err != nil {
		t.Fatal(err)
	}
	if bootstrapOpts.Init.BaseURL != "http://example.com" || strings.Join(bootstrapOpts.UpArgs, " ") != "--skip-verify --wait-timeout 30s" {
		t.Fatalf("unexpected bootstrap opts: %#v", bootstrapOpts)
	}

	if _, err := parseBootstrapFlags([]string{"--image", "--no-build"}); err == nil {
		t.Fatal("expected bootstrap to reject conflicting up flags")
	}
}

func TestWriteInitialConfig(t *testing.T) {
	t.Parallel()

	prodDir := t.TempDir()
	paths := Paths{
		RepoRoot:     filepath.Dir(filepath.Dir(prodDir)),
		ProdDir:      prodDir,
		ConfigPath:   filepath.Join(prodDir, defaultConfigName),
		GeneratedDir: filepath.Join(prodDir, defaultGenerated),
	}
	created, err := writeInitialConfig(paths, initOptions{BaseURL: "http://example.com", MySQLMode: "bundled"})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected config to be created")
	}
	cfg, err := loadConfig(paths)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Site.BaseURL != "http://example.com" {
		t.Fatalf("unexpected base url: %s", cfg.Site.BaseURL)
	}
	if cfg.SystemUser.Password == "" {
		t.Fatal("expected generated config to include system_user.password")
	}
	created, err = writeInitialConfig(paths, initOptions{BaseURL: "http://other.example.com", MySQLMode: "bundled"})
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("expected existing config to be preserved")
	}
}

func TestValidateConfigRequiresSystemUserPassword(t *testing.T) {
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
	cfg.SystemUser.Password = ""

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateConfig(rt); err == nil || !strings.Contains(err.Error(), "system_user.password is required") {
		t.Fatalf("expected system_user.password validation error, got: %v", err)
	}
}

func TestRunLayerChecksReport(t *testing.T) {
	t.Parallel()

	report := runLayerChecksReport("unit", []layerCheck{
		{Layer: layerControl, Name: "pass", Fn: func() error { return nil }},
		{Layer: layerRuntime, Name: "fail", Target: "target", Fn: func() error { return errors.New("boom") }},
	})
	if report.OK {
		t.Fatal("expected report to fail")
	}
	if report.Failures != 1 {
		t.Fatalf("unexpected failure count: %d", report.Failures)
	}
	if len(report.Checks) != 2 || report.Checks[1].Error != "boom" {
		t.Fatalf("unexpected checks: %#v", report.Checks)
	}
}

func TestWaitLayerChecks(t *testing.T) {
	t.Parallel()

	attempts := 0
	if err := waitLayerChecks("unit", []layerCheck{{
		Layer: layerControl,
		Name:  "eventual",
		Fn: func() error {
			attempts++
			if attempts < 2 {
				return errors.New("not yet")
			}
			return nil
		},
	}}, 100*time.Millisecond, time.Millisecond); err != nil {
		t.Fatal(err)
	}

	err := waitLayerChecks("unit", []layerCheck{{
		Layer: layerControl,
		Name:  "always fail",
		Fn:    func() error { return errors.New("boom") },
	}}, time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatal("expected waitLayerChecks to time out")
	}
}

func TestVerifyLayerChecksIncludeBundledSDKEndpoints(t *testing.T) {
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

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}

	checks := verifyLayerChecks(rt)
	for _, want := range []string{
		"mysql initialized",
		"main edge probe",
		"main platform probe",
		"main runtime probe",
		"sdk gateway endpoint",
		"sdk nats endpoint",
		"sdk minio endpoint",
	} {
		if !hasLayerCheckByName(checks, want) {
			t.Fatalf("verify checks missing %s: %#v", want, checks)
		}
	}
}

func TestRequiredMySQLDatabases(t *testing.T) {
	t.Parallel()

	rt := RuntimeConfig{
		Config: Config{
			MySQL: MySQLConfig{
				AppDatabase:     "app_db",
				StorageDatabase: "app-storage",
				AgentDatabase:   "agent-server",
				HRDatabase:      "hr-server",
			},
		},
	}

	got := requiredMySQLDatabases(rt)
	for _, want := range []string{"app_db", "app-storage", "agent-server", "hr-server"} {
		if !containsString(got, want) {
			t.Fatalf("required MySQL databases missing %q: %#v", want, got)
		}
	}
}

func TestParseMySQLCountOutputIgnoresClientWarnings(t *testing.T) {
	t.Parallel()

	got, err := parseMySQLCountOutput("mysql: [Warning] Using a password on the command line interface can be insecure.\n7\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != 7 {
		t.Fatalf("count = %d, want 7", got)
	}
}

func TestDeploymentLayersRedactSDKCredentials(t *testing.T) {
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
	cfg.NATS.User = "aos"
	cfg.NATS.Password = "super-secret"

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}

	components := deploymentComponents(rt)
	for _, component := range components {
		if strings.Contains(component.Role, "super-secret") {
			t.Fatalf("deployment component leaked SDK credentials: %#v", component)
		}
	}
	if got := redactURLCredentials("nats://aos:super-secret@host.containers.internal:4222"); got != "nats://aos:redacted@host.containers.internal:4222" {
		t.Fatalf("unexpected redacted url: %s", got)
	}
}

func hasDeploymentComponent(components []deploymentComponent, layer deploymentLayerID, name string) bool {
	for _, component := range components {
		if component.Layer == layer && component.Name == name {
			return true
		}
	}
	return false
}

func hasLayerCheckByName(checks []layerCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
