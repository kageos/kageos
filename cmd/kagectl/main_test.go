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
		`SMTP_MODE: "smtp"`,
		`OPENAI_API_KEY: "${OPENAI_API_KEY:-}"`,
		`app-base-builder:`,
		`profiles: ["build"]`,
		`dockerfile: deploy/prod/app-base-builder.Dockerfile`,
		`image: "localhost/kageos-app-base-builder:latest"`,
		`KAGEOS_APP_BASE_IMAGE: "kagebase:latest"`,
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("generated compose missing %q", want)
		}
	}

	mysqlInit := mustReadFile(t, filepath.Join(paths.GeneratedDir, "infra", "mysql-init.sql"))
	if !strings.Contains(mysqlInit, "CREATE DATABASE IF NOT EXISTS `app-server`") {
		t.Fatalf("mysql init should quote database identifiers, got:\n%s", mysqlInit)
	}
	if !strings.Contains(mysqlInit, "CREATE DATABASE IF NOT EXISTS `connector-server`") {
		t.Fatalf("mysql init should create connector-server database, got:\n%s", mysqlInit)
	}
	if !strings.Contains(mysqlInit, "CREATE DATABASE IF NOT EXISTS `timer-scheduler`") {
		t.Fatalf("mysql init should create timer-scheduler database, got:\n%s", mysqlInit)
	}
	if !strings.Contains(mysqlInit, "CREATE DATABASE IF NOT EXISTS `message-server`") {
		t.Fatalf("mysql init should create message-server database, got:\n%s", mysqlInit)
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

	hrServerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "hr-server.yaml"))
	for _, want := range []string{
		`registration_mode: "admin_only"`,
		`mode: "smtp"`,
	} {
		if !strings.Contains(hrServerConfig, want) {
			t.Fatalf("generated hr-server config missing %q, got:\n%s", want, hrServerConfig)
		}
	}

	appStorageConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "app-storage.yaml"))
	if !strings.Contains(appStorageConfig, `server_endpoint: "127.0.0.1:9000"`) {
		t.Fatalf("generated app-storage config should include prod runtime MinIO endpoint, got:\n%s", appStorageConfig)
	}

	connectorServerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "connector-server.yaml"))
	for _, want := range []string{
		`port: 9096`,
		`name: "connector-server"`,
		`callback_base_url: "http://127.0.0.1"`,
	} {
		if !strings.Contains(connectorServerConfig, want) {
			t.Fatalf("generated connector-server config missing %q, got:\n%s", want, connectorServerConfig)
		}
	}

	timerSchedulerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "timer-scheduler.yaml"))
	for _, want := range []string{
		`port: 9098`,
		`name: "timer-scheduler"`,
		`poll_interval_millis: 1000`,
	} {
		if !strings.Contains(timerSchedulerConfig, want) {
			t.Fatalf("generated timer-scheduler config missing %q, got:\n%s", want, timerSchedulerConfig)
		}
	}

	messageServerConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "message-server.yaml"))
	for _, want := range []string{
		`port: 9099`,
		`name: "message-server"`,
		`allow_nats_degraded_startup: false`,
	} {
		if !strings.Contains(messageServerConfig, want) {
			t.Fatalf("generated message-server config missing %q, got:\n%s", want, messageServerConfig)
		}
	}

	for _, want := range []string{
		`system_user:`,
		`password: "` + cfg.SystemUser.Password + `"`,
	} {
		if !strings.Contains(hrServerConfig, want) {
			t.Fatalf("generated hr-server config missing %q, got:\n%s", want, hrServerConfig)
		}
	}

	apiGatewayConfig := mustReadFile(t, filepath.Join(paths.GeneratedDir, "config", "api-gateway.yaml"))
	if !strings.Contains(apiGatewayConfig, `path: "/public/api"`) {
		t.Fatalf("generated api-gateway config should proxy public share APIs, got:\n%s", apiGatewayConfig)
	}
	if !strings.Contains(apiGatewayConfig, `path: "/connector"`) {
		t.Fatalf("generated api-gateway config should proxy connector APIs, got:\n%s", apiGatewayConfig)
	}
	if !strings.Contains(apiGatewayConfig, `path: "/timer"`) {
		t.Fatalf("generated api-gateway config should proxy timer APIs, got:\n%s", apiGatewayConfig)
	}
	if !strings.Contains(apiGatewayConfig, `path: "/message"`) {
		t.Fatalf("generated api-gateway config should proxy message APIs, got:\n%s", apiGatewayConfig)
	}
	for _, retired := range []string{`path: "/control"`} {
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
		"KAGEOS_REGISTRATION_MODE=admin_only",
		"SMTP_MODE=smtp",
		"KAGEOS_TLS_CERT_PEM_B64=" + cfg.Site.TLSCertPEMB64,
		"KAGEOS_TLS_KEY_PEM_B64=" + cfg.Site.TLSKeyPEMB64,
	} {
		if !strings.Contains(envFile, want) {
			t.Fatalf("generated env file missing %q, got:\n%s", want, envFile)
		}
	}
}

func TestWriteDeploymentSummary(t *testing.T) {
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
	if err := renderAll(rt); err != nil {
		t.Fatal(err)
	}
	if err := writeDeploymentSummary(rt, "ready"); err != nil {
		t.Fatal(err)
	}

	summary := mustReadFile(t, rt.SummaryPath)
	for _, want := range []string{
		"# Kageos Deployment Summary",
		"| Access URL | `http://127.0.0.1` |",
		"| Admin username | `system` |",
		"| Initial password | `" + cfg.SystemUser.Password + "` |",
		"| Registration mode | `admin_only` |",
		"| SMTP status | `not configured` |",
		"| Environment file | `" + rt.EnvFilePath + "` |",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("deployment summary missing %q, got:\n%s", want, summary)
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

func TestAppBaseImageEnvOverride(t *testing.T) {
	t.Setenv("KAGEOS_APP_BASE_IMAGE", "registry.example.com/kagebase:stable")

	cfg, err := defaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	applyEnvOverrides(&cfg)

	if got := cfg.Images.AppBase; got != "registry.example.com/kagebase:stable" {
		t.Fatalf("Images.AppBase = %q, want env override", got)
	}
}

func TestParseBuildAppBaseFlags(t *testing.T) {
	opts, err := parseBuildAppBaseFlags([]string{"--image", "registry.example.com/kagebase:stable", "--force", "--no-cache"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Image != "registry.example.com/kagebase:stable" {
		t.Fatalf("Image = %q", opts.Image)
	}
	if !opts.Force {
		t.Fatal("Force = false, want true")
	}
	if !opts.NoCache {
		t.Fatal("NoCache = false, want true")
	}
}

func TestParseBuildAppBaseFlagsRejectsEmptyImage(t *testing.T) {
	if _, err := parseBuildAppBaseFlags([]string{"--image", " "}); err == nil {
		t.Fatal("parseBuildAppBaseFlags() error = nil, want error")
	}
}

func TestParseInitDevFlags(t *testing.T) {
	opts, err := parseInitDevFlags([]string{"--engine", "podman", "--base-image", "registry.example.com/kagebase:stable", "--base-force", "--base-no-cache", "--company-code", "acme", "--company-name", "Acme Inc"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Engine != "podman" {
		t.Fatalf("Engine = %q, want podman", opts.Engine)
	}
	if opts.BaseImage != "registry.example.com/kagebase:stable" {
		t.Fatalf("BaseImage = %q", opts.BaseImage)
	}
	if !opts.BaseForce {
		t.Fatal("BaseForce = false, want true")
	}
	if !opts.BaseNoCache {
		t.Fatal("BaseNoCache = false, want true")
	}
	if opts.CompanyCode != "acme" || opts.CompanyName != "Acme Inc" {
		t.Fatalf("unexpected company opts: %#v", opts)
	}
}

func TestParseInitDevFlagsDefaultsToPodman(t *testing.T) {
	opts, err := parseInitDevFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Engine != "podman" {
		t.Fatalf("Engine = %q, want podman", opts.Engine)
	}
}

func TestParseInitDevFlagsAcceptsPositionalEngine(t *testing.T) {
	opts, err := parseInitDevFlags([]string{"docker"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Engine != "docker" {
		t.Fatalf("Engine = %q, want docker", opts.Engine)
	}
}

func TestParseInitDevFlagsRejectsUnknownEngine(t *testing.T) {
	if _, err := parseInitDevFlags([]string{"--engine", "containerd"}); err == nil {
		t.Fatal("parseInitDevFlags() error = nil, want error")
	}
}

func TestParseInitDevFlagsAcceptsSkipBase(t *testing.T) {
	opts, err := parseInitDevFlags([]string{"--skip-base"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.SkipBase {
		t.Fatal("SkipBase = false, want true")
	}
}

func TestTakeDevFlag(t *testing.T) {
	dev, rest := takeDevFlag([]string{"--dev", "--engine", "docker"})
	if !dev {
		t.Fatal("dev = false, want true")
	}
	if strings.Join(rest, " ") != "--engine docker" {
		t.Fatalf("rest = %q", strings.Join(rest, " "))
	}
}

func TestWorkspaceConfigRoundTrip(t *testing.T) {
	repoRoot := t.TempDir()
	paths := Paths{RepoRoot: repoRoot}

	if got := currentWorkspaceMode(paths); got != workspaceModeProd {
		t.Fatalf("currentWorkspaceMode() = %q, want prod default", got)
	}
	if err := writeWorkspaceConfig(paths, workspaceModeDev, workspaceDevConfig{Engine: "docker"}); err != nil {
		t.Fatal(err)
	}
	cfg := loadWorkspaceConfig(paths)
	if cfg.Mode != workspaceModeDev || cfg.Dev.Engine != "docker" {
		t.Fatalf("unexpected workspace config: %#v", cfg)
	}
	envContent := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "kageos.env"))
	if !strings.Contains(envContent, "KAGEOS_MODE=dev") || !strings.Contains(envContent, "KAGEOS_DEV_ENGINE=docker") {
		t.Fatalf("workspace env file missing dev engine, got:\n%s", envContent)
	}
}

func TestSDKMinIOEndpointUsesLoopbackForProdHostNetworkContainers(t *testing.T) {
	t.Parallel()

	for _, endpoint := range []string{"127.0.0.1:9000", "localhost:9000", "host.containers.internal:9000"} {
		if got := sdkMinIOEndpoint(endpoint); got != "127.0.0.1:9000" {
			t.Fatalf("sdkMinIOEndpoint(%q) = %q", endpoint, got)
		}
	}
	if got := sdkMinIOEndpoint("minio.example.com:9000"); got != "minio.example.com:9000" {
		t.Fatalf("sdkMinIOEndpoint(remote) = %q", got)
	}
}

func TestRenderDevConfigUsesKageosDir(t *testing.T) {
	repoRoot := t.TempDir()
	paths := Paths{
		RepoRoot:     repoRoot,
		ProdDir:      filepath.Join(repoRoot, defaultProdDir),
		ConfigPath:   filepath.Join(repoRoot, defaultProdDir, defaultConfigName),
		GeneratedDir: filepath.Join(repoRoot, defaultProdDir, defaultGenerated),
	}

	if err := renderDevConfig(paths, false, "", ""); err != nil {
		t.Fatal(err)
	}

	appServerConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "app-server.yaml"))
	if strings.Contains(appServerConfig, `password: "root"`) {
		t.Fatalf("dev app-server config should not use fixed root password, got:\n%s", appServerConfig)
	}
	if !strings.Contains(appServerConfig, `port: 3318`) {
		t.Fatalf("dev app-server config should use isolated mysql port 3318, got:\n%s", appServerConfig)
	}
	apiGatewayConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "api-gateway.yaml"))
	if !strings.Contains(apiGatewayConfig, `path: "/connector"`) {
		t.Fatalf("dev api-gateway config should proxy connector APIs, got:\n%s", apiGatewayConfig)
	}
	if !strings.Contains(apiGatewayConfig, `path: "/timer"`) {
		t.Fatalf("dev api-gateway config should proxy timer APIs, got:\n%s", apiGatewayConfig)
	}
	if !strings.Contains(apiGatewayConfig, `path: "/message"`) {
		t.Fatalf("dev api-gateway config should proxy message APIs, got:\n%s", apiGatewayConfig)
	}
	connectorServerConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "connector-server.yaml"))
	for _, want := range []string{
		`port: 9096`,
		`port: 3318`,
		`name: "connector-server"`,
	} {
		if !strings.Contains(connectorServerConfig, want) {
			t.Fatalf("dev connector-server config missing %q, got:\n%s", want, connectorServerConfig)
		}
	}
	appRuntimeConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "app-runtime.yaml"))
	if !strings.Contains(appRuntimeConfig, `base_image: "kagebase:latest"`) {
		t.Fatalf("dev app-runtime config should use default base image, got:\n%s", appRuntimeConfig)
	}
	timerSchedulerConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "timer-scheduler.yaml"))
	for _, want := range []string{
		`port: 9098`,
		`port: 3318`,
		`name: "timer-scheduler"`,
	} {
		if !strings.Contains(timerSchedulerConfig, want) {
			t.Fatalf("dev timer-scheduler config missing %q, got:\n%s", want, timerSchedulerConfig)
		}
	}
	messageServerConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "message-server.yaml"))
	for _, want := range []string{
		`port: 9099`,
		`port: 3318`,
		`name: "message-server"`,
	} {
		if !strings.Contains(messageServerConfig, want) {
			t.Fatalf("dev message-server config missing %q, got:\n%s", want, messageServerConfig)
		}
	}
	envFile := filepath.Join(repoRoot, ".kageos", "dev", "env", "kageos.env")
	if !fileExists(envFile) {
		t.Fatal("dev env file was not rendered")
	}
	envContent := mustReadFile(t, envFile)
	if strings.Contains(envContent, "MYSQL_ROOT_PASSWORD=root") || strings.Contains(envContent, "MINIO_ROOT_PASSWORD=minioadmin123") {
		t.Fatalf("dev env should not use fixed passwords, got:\n%s", envContent)
	}
	if !strings.Contains(envContent, "MYSQL_PORT=3318") {
		t.Fatalf("dev env should use isolated mysql port 3318, got:\n%s", envContent)
	}
}

func TestRenderDevConfigAcceptsCompanyOverrides(t *testing.T) {
	repoRoot := t.TempDir()
	paths := Paths{
		RepoRoot:     repoRoot,
		ProdDir:      filepath.Join(repoRoot, defaultProdDir),
		ConfigPath:   filepath.Join(repoRoot, defaultProdDir, defaultConfigName),
		GeneratedDir: filepath.Join(repoRoot, defaultProdDir, defaultGenerated),
	}

	if err := renderDevConfig(paths, false, "acme", "Acme Inc"); err != nil {
		t.Fatal(err)
	}

	hrConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "hr-server.yaml"))
	for _, want := range []string{`code: "acme"`, `name: "Acme Inc"`} {
		if !strings.Contains(hrConfig, want) {
			t.Fatalf("dev hr config missing %q, got:\n%s", want, hrConfig)
		}
	}
	envContent := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "env", "kageos.env"))
	if !strings.Contains(envContent, "KAGEOS_COMPANY_CODE=acme") || !strings.Contains(envContent, `KAGEOS_COMPANY_NAME="Acme Inc"`) {
		t.Fatalf("dev env missing company values, got:\n%s", envContent)
	}
}

func TestRenderDevConfigPreservesExistingSecrets(t *testing.T) {
	repoRoot := t.TempDir()
	paths := Paths{
		RepoRoot:     repoRoot,
		ProdDir:      filepath.Join(repoRoot, defaultProdDir),
		ConfigPath:   filepath.Join(repoRoot, defaultProdDir, defaultConfigName),
		GeneratedDir: filepath.Join(repoRoot, defaultProdDir, defaultGenerated),
	}

	if err := os.MkdirAll(filepath.Join(repoRoot, ".kageos", "dev", "env"), 0755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(repoRoot, ".kageos", "dev", "env", "kageos.env")
	initialEnv := strings.Join([]string{
		"MYSQL_ROOT_PASSWORD=existing-mysql",
		"NATS_SEED_USER=existing-nats",
		"NATS_SEED_PASSWORD=existing-nats-pass",
		"MINIO_ROOT_USER=existing-minio",
		"MINIO_ROOT_PASSWORD=existing-minio-pass",
		"JWT_SECRET=existing-jwt-secret-existing-jwt-secret",
		"SYSTEM_USER_PASSWORD=existing-system-pass",
		"",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(initialEnv), 0600); err != nil {
		t.Fatal(err)
	}

	if err := renderDevConfig(paths, false, "", ""); err != nil {
		t.Fatal(err)
	}

	appServerConfig := mustReadFile(t, filepath.Join(repoRoot, ".kageos", "dev", "config", "app-server.yaml"))
	if !strings.Contains(appServerConfig, `password: "existing-mysql"`) {
		t.Fatalf("dev config should preserve existing mysql password, got:\n%s", appServerConfig)
	}
}

func TestRenderDevConfigRefusesImplicitSecretsWhenStateExistsWithoutEnv(t *testing.T) {
	repoRoot := t.TempDir()
	paths := Paths{
		RepoRoot:     repoRoot,
		ProdDir:      filepath.Join(repoRoot, defaultProdDir),
		ConfigPath:   filepath.Join(repoRoot, defaultProdDir, defaultConfigName),
		GeneratedDir: filepath.Join(repoRoot, defaultProdDir, defaultGenerated),
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, ".kageos", "dev"), 0755); err != nil {
		t.Fatal(err)
	}

	err := renderDevConfig(paths, false, "", "")
	if err == nil || !strings.Contains(err.Error(), "refusing to generate new secrets implicitly") {
		t.Fatalf("expected implicit secret generation refusal, got: %v", err)
	}
}

func TestRenderDevConfigRefusesIncompleteExistingEnv(t *testing.T) {
	repoRoot := t.TempDir()
	paths := Paths{
		RepoRoot:     repoRoot,
		ProdDir:      filepath.Join(repoRoot, defaultProdDir),
		ConfigPath:   filepath.Join(repoRoot, defaultProdDir, defaultConfigName),
		GeneratedDir: filepath.Join(repoRoot, defaultProdDir, defaultGenerated),
	}
	envDir := filepath.Join(repoRoot, ".kageos", "dev", "env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "kageos.env"), []byte("MYSQL_ROOT_PASSWORD=existing-mysql\n"), 0600); err != nil {
		t.Fatal(err)
	}

	err := renderDevConfig(paths, false, "", "")
	if err == nil || !strings.Contains(err.Error(), "dev env is incomplete") {
		t.Fatalf("expected incomplete env refusal, got: %v", err)
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

func TestDefaultStorageRootUsesUserHome(t *testing.T) {
	t.Parallel()

	got := defaultStorageRootForHome("/home/ubuntu")
	want := filepath.Join("/home/ubuntu", ".kageos", "storage", "prod")
	if got != want {
		t.Fatalf("defaultStorageRootForHome() = %q, want %q", got, want)
	}
}

func TestDetectComposeCommandHonorsForcedEngine(t *testing.T) {
	t.Setenv(composeEngineEnv, "docker")
	restoreExecMocks := mockExecForComposeTests(t)
	defer restoreExecMocks()

	execLookPath = func(file string) (string, error) {
		if file == "docker" {
			return "/usr/bin/docker", nil
		}
		return "", errors.New("not found")
	}
	execRun = func(name string, args ...string) error {
		if name == "docker" && strings.Join(args, " ") == "compose version" {
			return nil
		}
		t.Fatalf("unexpected command: %s %s", name, strings.Join(args, " "))
		return nil
	}

	compose, err := detectComposeCommand()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(compose, " ") != "docker compose" {
		t.Fatalf("compose command = %v, want docker compose", compose)
	}
}

func TestCheckComposeRuntimeReportsPodmanSocketHint(t *testing.T) {
	t.Setenv("USER", "ubuntu")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	t.Setenv("DOCKER_HOST", "")
	restoreExecMocks := mockExecForComposeTests(t)
	defer restoreExecMocks()

	execLookPath = func(file string) (string, error) {
		if file == "podman" {
			return "/usr/bin/podman", nil
		}
		return "", errors.New("not found")
	}
	execRun = func(name string, args ...string) error {
		if name == "podman" && strings.Join(args, " ") == "compose version" {
			return nil
		}
		t.Fatalf("unexpected command: %s %s", name, strings.Join(args, " "))
		return nil
	}
	execOutput = func(name string, args ...string) (string, error) {
		if name == "podman" && strings.Join(args, " ") == "compose ls" {
			return "failed to connect to the docker API at unix:///run/user/1000/podman/podman.sock", errors.New("exit status 1")
		}
		t.Fatalf("unexpected output command: %s %s", name, strings.Join(args, " "))
		return "", nil
	}

	err := checkComposeRuntime()
	if err == nil {
		t.Fatal("expected compose runtime check to fail")
	}
	for _, want := range []string{
		"podman.socket",
		"loginctl enable-linger ubuntu",
		"/run/user/1000/podman/podman.sock",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("compose runtime error missing %q: %v", want, err)
		}
	}
}

func TestCheckComposeRuntimeFallsBackToEngineInfoWhenComposeLsUnsupported(t *testing.T) {
	restoreExecMocks := mockExecForComposeTests(t)
	defer restoreExecMocks()

	execLookPath = func(file string) (string, error) {
		if file == "podman" {
			return "/usr/bin/podman", nil
		}
		return "", errors.New("not found")
	}
	execRun = func(name string, args ...string) error {
		if name != "podman" {
			t.Fatalf("unexpected command name: %s", name)
		}
		switch strings.Join(args, " ") {
		case "compose version", "info":
			return nil
		default:
			t.Fatalf("unexpected command args: %s", strings.Join(args, " "))
			return nil
		}
	}
	execOutput = func(name string, args ...string) (string, error) {
		if name == "podman" && strings.Join(args, " ") == "compose ls" {
			return "unknown command \"ls\"", errors.New("exit status 1")
		}
		t.Fatalf("unexpected output command: %s %s", name, strings.Join(args, " "))
		return "", nil
	}

	if err := checkComposeRuntime(); err != nil {
		t.Fatalf("compose runtime should fall back to podman info: %v", err)
	}
}

func mockExecForComposeTests(t *testing.T) func() {
	t.Helper()
	oldLookPath := execLookPath
	oldRun := execRun
	oldOutput := execOutput
	return func() {
		execLookPath = oldLookPath
		execRun = oldRun
		execOutput = oldOutput
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

	initOpts, err := parseInitFlags("init", []string{"--force", "--base-url", "http://example.com", "--mysql-mode", "bundled", "--company-code", "acme", "--company-name", "Acme Inc"})
	if err != nil {
		t.Fatal(err)
	}
	if !initOpts.Force || initOpts.BaseURL != "http://example.com" || initOpts.MySQLMode != "bundled" || initOpts.CompanyCode != "acme" || initOpts.CompanyName != "Acme Inc" {
		t.Fatalf("unexpected init opts: %#v", initOpts)
	}

	bootstrapOpts, err := parseBootstrapFlags([]string{"--base-url", "http://example.com", "--skip-verify", "--wait-timeout", "30s"})
	if err != nil {
		t.Fatal(err)
	}
	if bootstrapOpts.Init.BaseURL != "http://example.com" || strings.Join(bootstrapOpts.UpArgs, " ") != "--skip-verify --wait-timeout 30s" {
		t.Fatalf("unexpected bootstrap opts: %#v", bootstrapOpts)
	}

	devBootstrapOpts, err := parseBootstrapDevFlags([]string{"--engine", "docker", "--skip-base", "--skip-verify", "--wait-timeout", "30s"})
	if err != nil {
		t.Fatal(err)
	}
	if devBootstrapOpts.Init.Engine != "docker" || !devBootstrapOpts.Init.SkipBase || strings.Join(devBootstrapOpts.UpArgs, " ") != "--skip-verify --wait-timeout 30s" {
		t.Fatalf("unexpected dev bootstrap opts: %#v", devBootstrapOpts)
	}

	if _, err := parseBootstrapFlags([]string{"--image", "--no-build"}); err == nil {
		t.Fatal("expected bootstrap to reject conflicting up flags")
	}
	if _, err := parseBootstrapDevFlags([]string{"--image"}); err == nil {
		t.Fatal("expected bootstrap --dev to reject --image")
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
	created, err := writeInitialConfig(paths, initOptions{BaseURL: "http://example.com", MySQLMode: "bundled", CompanyCode: "acme", CompanyName: "Acme Inc", RegistrationMode: "email_code", SMTPMode: "smtp"})
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
	if cfg.Company.Code != "acme" || cfg.Company.Name != "Acme Inc" {
		t.Fatalf("unexpected company config: %#v", cfg.Company)
	}
	if cfg.Auth.RegistrationMode != "email_code" || cfg.SMTP.Mode != "smtp" {
		t.Fatalf("unexpected auth/smtp config: auth=%#v smtp=%#v", cfg.Auth, cfg.SMTP)
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

func TestValidateConfigRequiresSMTPForEmailRegistration(t *testing.T) {
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
	cfg.Auth.RegistrationMode = "email_code"
	cfg.SMTP.Username = ""
	cfg.SMTP.Password = ""
	cfg.SMTP.From = ""

	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateConfig(rt); err == nil || !strings.Contains(err.Error(), "requires complete smtp") {
		t.Fatalf("expected smtp validation error, got: %v", err)
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
		"minio clock",
		"main edge probe",
		"main platform probe",
		"main runtime probe",
		"connector-server",
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
				AppDatabase:       "app-server",
				StorageDatabase:   "app-storage",
				AgentDatabase:     "agent-server",
				ConnectorDatabase: "connector-server",
				HRDatabase:        "hr-server",
				TimerDatabase:     "timer-scheduler",
				MessageDatabase:   "message-server",
			},
		},
	}

	got := requiredMySQLDatabases(rt)
	for _, want := range []string{"app-server", "app-storage", "agent-server", "connector-server", "hr-server", "timer-scheduler", "message-server"} {
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
