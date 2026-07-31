package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

func renderAll(rt RuntimeConfig) error {
	if err := os.RemoveAll(rt.Paths.GeneratedDir); err != nil {
		return err
	}
	dirs := []string{
		rt.Paths.GeneratedDir,
		filepath.Join(rt.Paths.GeneratedDir, "config"),
		filepath.Join(rt.Paths.GeneratedDir, "env"),
		filepath.Join(rt.Paths.GeneratedDir, "infra"),
		rt.TLSCertsHostDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"docker-compose.yaml":          renderTemplate(composeTemplate, rt),
		".env":                         renderTemplate(envTemplate, rt),
		"env/kageos.env":               renderTemplate(envTemplate, rt),
		"infra/nats-server.conf":       renderTemplate(natsConfigTemplate, rt),
		"infra/mysql-init.sql":         renderTemplate(mysqlInitTemplate, rt),
		"config/global.yaml":           renderTemplate(globalConfigTemplate, rt),
		"config/api-gateway.yaml":      renderTemplate(apiGatewayConfigTemplate, rt),
		"config/timer-scheduler.yaml":  renderTemplate(timerSchedulerConfigTemplate, rt),
		"config/message-server.yaml":   renderTemplate(messageServerConfigTemplate, rt),
		"config/app-runtime.yaml":      renderTemplate(appRuntimeConfigTemplate, rt),
		"config/app-server.yaml":       renderTemplate(appServerConfigTemplate, rt),
		"config/app-storage.yaml":      renderTemplate(appStorageConfigTemplate, rt),
		"config/connector-server.yaml": renderTemplate(connectorServerConfigTemplate, rt),
		"config/agent-server.yaml":     renderTemplate(agentServerConfigTemplate, rt),
		"config/hr-server.yaml":        renderTemplate(hrServerConfigTemplate, rt),
	}
	for rel, content := range files {
		mode := os.FileMode(0644)
		if rel == ".env" || strings.HasPrefix(rel, "env/") {
			mode = 0600
		}
		if err := os.WriteFile(filepath.Join(rt.Paths.GeneratedDir, rel), []byte(content), mode); err != nil {
			return err
		}
	}
	if err := renderTLSFiles(rt); err != nil {
		return err
	}
	return nil
}

func renderDevConfig(paths Paths, regenSecrets bool) error {
	stateDir := filepath.Join(paths.RepoRoot, ".kageos", "dev")
	envDir := filepath.Join(paths.RepoRoot, ".kageos", "dev", "env")
	envPath := filepath.Join(envDir, "kageos.env")
	secrets, err := loadOrCreateDevSecrets(stateDir, envPath, regenSecrets)
	if err != nil {
		return err
	}
	if regenSecrets {
		fmt.Println("==> dev secrets regenerated")
	} else if fileExists(envPath) && hasWeakDevSecrets(secrets) {
		fmt.Println("WARN: existing dev secrets contain old fixed defaults; keep them to avoid breaking existing local volumes")
		fmt.Println("WARN: rotate with `kagectl init --dev --regen-secrets` after clearing old dev infra volumes")
	}

	cfg := defaultDevDeploymentConfig(secrets)
	applyEnvOverrides(&cfg)
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		return err
	}
	rt.AppContainerNetworkMode = ""
	rt.SDKGatewayURL = "http://host.containers.internal:9090"
	rt.SDKNATSURL = devSDKNATSURL(rt.NATSURL)
	rt.SDKMinIOEndpoint = devSDKMinIOEndpoint(rt.MinIOEndpoint)
	rt.AppRuntimeBasePath = filepath.Join(paths.RepoRoot, ".kageos", "dev", "namespace")

	configDir := filepath.Join(paths.RepoRoot, defaultDevConfig)
	for _, dir := range []string{configDir, envDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"global.yaml":           renderTemplate(globalConfigTemplate, rt),
		"api-gateway.yaml":      renderTemplate(apiGatewayConfigTemplate, rt),
		"timer-scheduler.yaml":  renderTemplate(timerSchedulerConfigTemplate, rt),
		"message-server.yaml":   renderTemplate(messageServerConfigTemplate, rt),
		"app-runtime.yaml":      renderTemplate(appRuntimeConfigTemplate, rt),
		"app-server.yaml":       renderTemplate(appServerConfigTemplate, rt),
		"app-storage.yaml":      renderTemplate(appStorageConfigTemplate, rt),
		"connector-server.yaml": renderTemplate(connectorServerConfigTemplate, rt),
		"agent-server.yaml":     renderTemplate(agentServerConfigTemplate, rt),
		"hr-server.yaml":        renderTemplate(hrServerConfigTemplate, rt),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(configDir, name), []byte(content), 0644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(envPath, []byte(renderTemplate(envTemplate, rt)), 0600); err != nil {
		return err
	}
	fmt.Printf("==> dev config rendered: %s\n", configDir)
	return nil
}

func printDevInitSummary(paths Paths, opts initDevOptions) {
	envPath := filepath.Join(paths.RepoRoot, ".kageos", "dev", "env", "kageos.env")
	values, err := readEnvFile(envPath)
	if err != nil {
		fmt.Printf("WARN: unable to read dev summary env file: %v\n", err)
		return
	}

	baseImage := opts.BaseImage
	if baseImage == "" {
		baseImage = values["KAGEOS_APP_BASE_IMAGE"]
	}
	smtpStatus := "not configured"
	if strings.TrimSpace(values["SMTP_MODE"]) == "log" {
		smtpStatus = "log mode"
	} else if strings.TrimSpace(values["SMTP_HOST"]) != "" &&
		strings.TrimSpace(values["SMTP_USERNAME"]) != "" &&
		strings.TrimSpace(values["SMTP_PASSWORD"]) != "" &&
		strings.TrimSpace(values["SMTP_FROM"]) != "" {
		smtpStatus = "configured"
	}

	rows := [][2]string{
		{"Mode env", workspaceEnvPath(paths)},
		{"Config dir", filepath.Join(paths.RepoRoot, defaultDevConfig)},
		{"Env file", envPath},
		{"Engine", opts.Engine},
		{"App base image", baseImage},
		{"Admin username", "system"},
		{"Admin password", values["SYSTEM_USER_PASSWORD"]},
		{"MySQL host", values["MYSQL_HOST"]},
		{"MySQL port", values["MYSQL_PORT"]},
		{"MySQL user", "root"},
		{"MySQL password", values["MYSQL_ROOT_PASSWORD"]},
		{"MySQL databases", "app-server, agent-server, app-storage, connector-server, hr-server, timer-scheduler, message-server"},
		{"NATS URL", values["NATS_URL"]},
		{"NATS user", values["NATS_SEED_USER"]},
		{"NATS password", values["NATS_SEED_PASSWORD"]},
		{"MinIO endpoint", values["MINIO_ENDPOINT"]},
		{"MinIO root user", values["MINIO_ROOT_USER"]},
		{"MinIO root password", values["MINIO_ROOT_PASSWORD"]},
		{"JWT secret", values["JWT_SECRET"]},
		{"SMTP mode", values["SMTP_MODE"]},
		{"SMTP status", smtpStatus},
		{"SMTP host", values["SMTP_HOST"]},
		{"SMTP username", values["SMTP_USERNAME"]},
	}
	fmt.Println()
	fmt.Println("Kageos dev initialization summary")
	printPlainTable("Item", "Value", rows)
	fmt.Println()
	fmt.Println("Next: run `go run ./cmd/kagectl up` to start the local backend.")
	fmt.Println("Tip: local dev uses SMTP_MODE=log, so verification codes are printed in logs and returned as debug_code. Set SMTP_MODE=smtp and configure SMTP_* when real email delivery is required.")
}

func printPlainTable(leftHeader, rightHeader string, rows [][2]string) {
	width := len(leftHeader)
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	fmt.Printf("%-*s  %s\n", width, leftHeader, rightHeader)
	fmt.Printf("%s  %s\n", strings.Repeat("-", width), strings.Repeat("-", 48))
	for _, row := range rows {
		fmt.Printf("%-*s  %s\n", width, row[0], row[1])
	}
}

func loadOrCreateDevSecrets(stateDir, envPath string, regen bool) (devSecrets, error) {
	if !regen {
		if values, err := readEnvFile(envPath); err == nil {
			return mergeDevSecrets(values)
		} else if !errors.Is(err, os.ErrNotExist) {
			return devSecrets{}, err
		}
		if dirExists(stateDir) {
			return devSecrets{}, fmt.Errorf("dev state exists at %s but %s is missing; refusing to generate new secrets implicitly (use --regen-secrets only after clearing old dev infra volumes)", stateDir, envPath)
		}
	}
	return generateDevSecrets()
}

func generateDevSecrets() (devSecrets, error) {
	mysqlPass, err := randomHex(32)
	if err != nil {
		return devSecrets{}, err
	}
	natsPass, err := randomHex(24)
	if err != nil {
		return devSecrets{}, err
	}
	minioPass, err := randomHex(32)
	if err != nil {
		return devSecrets{}, err
	}
	jwt, err := randomHex(32)
	if err != nil {
		return devSecrets{}, err
	}
	appDBSecret, err := randomHex(32)
	if err != nil {
		return devSecrets{}, err
	}
	systemPass, err := randomHex(24)
	if err != nil {
		return devSecrets{}, err
	}
	return devSecrets{
		MySQLRootPassword:  mysqlPass,
		NATSUser:           "kageos",
		NATSPassword:       natsPass,
		MinIORootUser:      "minioadmin",
		MinIORootPassword:  minioPass,
		JWTSecret:          jwt,
		AppDBSecret:        appDBSecret,
		SystemUserPassword: systemPass,
	}, nil
}

func mergeDevSecrets(values map[string]string) (devSecrets, error) {
	required := []string{
		"MYSQL_ROOT_PASSWORD",
		"NATS_SEED_USER",
		"NATS_SEED_PASSWORD",
		"MINIO_ROOT_USER",
		"MINIO_ROOT_PASSWORD",
		"JWT_SECRET",
		"SYSTEM_USER_PASSWORD",
	}
	missing := make([]string, 0)
	for _, key := range required {
		if strings.TrimSpace(values[key]) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return devSecrets{}, fmt.Errorf("dev env is incomplete, missing %s; refusing to generate replacement secrets implicitly", strings.Join(missing, ", "))
	}
	appDBSecret := strings.TrimSpace(values["KAGEOS_APP_DB_SECRET_KEY"])
	if appDBSecret == "" {
		generated, err := randomHex(32)
		if err != nil {
			return devSecrets{}, err
		}
		appDBSecret = generated
	}
	return devSecrets{
		MySQLRootPassword:  strings.TrimSpace(values["MYSQL_ROOT_PASSWORD"]),
		NATSUser:           strings.TrimSpace(values["NATS_SEED_USER"]),
		NATSPassword:       strings.TrimSpace(values["NATS_SEED_PASSWORD"]),
		MinIORootUser:      strings.TrimSpace(values["MINIO_ROOT_USER"]),
		MinIORootPassword:  strings.TrimSpace(values["MINIO_ROOT_PASSWORD"]),
		JWTSecret:          strings.TrimSpace(values["JWT_SECRET"]),
		AppDBSecret:        appDBSecret,
		SystemUserPassword: strings.TrimSpace(values["SYSTEM_USER_PASSWORD"]),
	}, nil
}

func readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values, nil
}

func hasWeakDevSecrets(secrets devSecrets) bool {
	return secrets.MySQLRootPassword == "root" ||
		secrets.MinIORootPassword == "minioadmin123" ||
		strings.HasPrefix(secrets.JWTSecret, "dev-jwt-secret") ||
		secrets.SystemUserPassword == "kageos-dev-password"
}

func renderTLSFiles(rt RuntimeConfig) error {
	certB64 := strings.TrimSpace(rt.Site.TLSCertPEMB64)
	keyB64 := strings.TrimSpace(rt.Site.TLSKeyPEMB64)
	if certB64 == "" && keyB64 == "" {
		if (rt.Site.TLSMode == "https" || rt.Site.TLSMode == "redirect") && rt.Site.AllowSelfSignedBootstrap {
			if tlsFilesExist(rt) {
				return nil
			}
			cert, key, err := generateSelfSignedTLSPEM(rt.Site.BaseURL)
			if err != nil {
				return err
			}
			return writeTLSFiles(rt, cert, key)
		}
		return nil
	}
	if certB64 == "" || keyB64 == "" {
		return fmt.Errorf("TLS cert and key must be provided together")
	}
	cert, err := decodeBase64PEM("TLS certificate", certB64)
	if err != nil {
		return err
	}
	key, err := decodeBase64PEM("TLS private key", keyB64)
	if err != nil {
		return err
	}
	return writeTLSFiles(rt, cert, key)
}

func tlsFilesExist(rt RuntimeConfig) bool {
	return fileExists(filepath.Join(rt.TLSCertsHostDir, "fullchain.pem")) && fileExists(filepath.Join(rt.TLSCertsHostDir, "privkey.pem"))
}

func writeTLSFiles(rt RuntimeConfig, cert []byte, key []byte) error {
	if err := os.WriteFile(filepath.Join(rt.TLSCertsHostDir, "fullchain.pem"), cert, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(rt.TLSCertsHostDir, "privkey.pem"), key, 0600); err != nil {
		return err
	}
	return nil
}

func decodeBase64PEM(label, value string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		if raw, rawErr := base64.RawStdEncoding.DecodeString(value); rawErr == nil {
			data = raw
		} else {
			return nil, fmt.Errorf("decode %s base64: %w", label, err)
		}
	}
	text := strings.TrimSpace(string(data))
	if !strings.Contains(text, "-----BEGIN ") || !strings.Contains(text, "-----END ") {
		return nil, fmt.Errorf("%s does not look like PEM data after base64 decode", label)
	}
	return []byte(text + "\n"), nil
}

func finishDeploymentSummary(rt RuntimeConfig, status string) error {
	if err := writeDeploymentSummary(rt, status); err != nil {
		return err
	}
	printDeploymentSummary(rt, status)
	return nil
}

func writeDeploymentSummary(rt RuntimeConfig, status string) error {
	content := deploymentSummaryMarkdown(rt, status)
	return os.WriteFile(rt.SummaryPath, []byte(content), 0600)
}

func deploymentSummaryMarkdown(rt RuntimeConfig, status string) string {
	rows := deploymentSummaryRows(rt, status)
	var b strings.Builder
	b.WriteString("# Kageos Deployment Summary\n\n")
	b.WriteString("| Item | Value |\n")
	b.WriteString("|---|---|\n")
	for _, row := range rows {
		b.WriteString("| ")
		b.WriteString(row[0])
		b.WriteString(" | `")
		b.WriteString(strings.ReplaceAll(row[1], "`", "\\`"))
		b.WriteString("` |\n")
	}
	return b.String()
}

func printDeploymentSummary(rt RuntimeConfig, status string) {
	rows := deploymentSummaryRows(rt, status)
	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}

	fmt.Println()
	fmt.Println("Kageos deployment summary")
	fmt.Printf("%-*s  %s\n", width, "Item", "Value")
	fmt.Printf("%s  %s\n", strings.Repeat("-", width), strings.Repeat("-", 48))
	for _, row := range rows {
		fmt.Printf("%-*s  %s\n", width, row[0], row[1])
	}
}

func deploymentSummaryRows(rt RuntimeConfig, status string) [][2]string {
	return [][2]string{
		{"Status", status},
		{"Access URL", rt.Site.BaseURL},
		{"TLS mode", rt.Site.TLSMode},
		{"Self-signed TLS bootstrap", fmt.Sprintf("%t", rt.Site.AllowSelfSignedBootstrap)},
		{"Timezone", rt.Timezone},
		{"Network profile", rt.Network.Profile},
		{"Admin username", "system"},
		{"Initial password", rt.SystemUser.Password},
		{"Registration mode", rt.Auth.RegistrationMode},
		{"SMTP mode", rt.SMTP.Mode},
		{"SMTP status", smtpStatus(rt.SMTP)},
		{"MySQL mode", rt.MySQL.Mode},
		{"MySQL address", rt.MySQLAddress},
		{"MySQL user", rt.MySQL.User},
		{"MySQL password", rt.MySQL.Password},
		{"NATS URL", redactURLCredentials(rt.NATSURL)},
		{"NATS user", rt.NATSAuthUser},
		{"NATS password", rt.NATSAuthPassword},
		{"MinIO endpoint", rt.MinIOEndpoint},
		{"MinIO access key", rt.MinIO.AccessKey},
		{"MinIO secret key", rt.MinIO.SecretKey},
		{"JWT secret", rt.Secrets.JWTSecret},
		{"App base image", rt.Images.AppBase},
		{"Main config", rt.Paths.ConfigPath},
		{"Compose file", rt.ComposeConfigPath},
		{"Generated config dir", filepath.Join(rt.Paths.GeneratedDir, "config")},
		{"Environment file", rt.EnvFilePath},
		{"TLS directory", rt.TLSCertsHostDir},
		{"Summary file", rt.SummaryPath},
		{"Status command", "go run ./cmd/kagectl status"},
		{"Logs command", "go run ./cmd/kagectl logs main"},
		{"Stop command", "go run ./cmd/kagectl down"},
	}
}

func printProdInitSummary(paths Paths, cfg Config) {
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		fmt.Printf("WARN: unable to build prod init summary: %v\n", err)
		return
	}
	rows := [][2]string{
		{"Mode env", workspaceEnvPath(paths)},
		{"Config file", paths.ConfigPath},
		{"Access URL", cfg.Site.BaseURL},
		{"TLS mode", rt.Site.TLSMode},
		{"Self-signed TLS bootstrap", fmt.Sprintf("%t", rt.Site.AllowSelfSignedBootstrap)},
		{"Timezone", rt.Timezone},
		{"Network profile", rt.Network.Profile},
		{"HTTP port", strconv.Itoa(rt.Site.HTTPPort)},
		{"HTTPS port", strconv.Itoa(rt.Site.HTTPSPort)},
		{"Admin username", "system"},
		{"Initial password", cfg.SystemUser.Password},
		{"Registration mode", cfg.Auth.RegistrationMode},
		{"SMTP mode", cfg.SMTP.Mode},
		{"SMTP status", smtpStatus(cfg.SMTP)},
		{"MySQL mode", cfg.MySQL.Mode},
		{"MySQL address", rt.MySQLAddress},
		{"MySQL user", cfg.MySQL.User},
		{"MySQL password", cfg.MySQL.Password},
		{"NATS URL", redactURLCredentials(rt.NATSURL)},
		{"NATS user", cfg.NATS.User},
		{"NATS password", cfg.NATS.Password},
		{"MinIO endpoint", rt.MinIOEndpoint},
		{"MinIO access key", cfg.MinIO.AccessKey},
		{"MinIO secret key", cfg.MinIO.SecretKey},
		{"JWT secret", cfg.Secrets.JWTSecret},
		{"Storage root", cfg.Storage.Root},
	}
	fmt.Println()
	fmt.Println("Kageos production initialization summary")
	printPlainTable("Item", "Value", rows)
	fmt.Println()
	fmt.Println("Next: run `go run ./cmd/kagectl doctor`, then `go run ./cmd/kagectl up`.")
	fmt.Println("Tip: production defaults to auth.registration_mode=admin_only. Log in as system, configure SMTP in System settings, then enable email_code registration if public signup is required.")
}

func smtpStatus(cfg SMTPConfig) string {
	if cfg.Mode == "log" {
		return "log mode"
	}
	if strings.TrimSpace(cfg.Host) != "" &&
		cfg.Port > 0 &&
		strings.TrimSpace(cfg.Username) != "" &&
		strings.TrimSpace(cfg.Password) != "" &&
		strings.TrimSpace(cfg.From) != "" {
		return "configured"
	}
	return "not configured"
}

func renderTemplate(text string, data any) string {
	tpl := template.Must(template.New("tpl").Funcs(template.FuncMap{
		"q":          func(v any) string { return strconv.Quote(fmt.Sprint(v)) },
		"mysqlIdent": mysqlIdent,
	}).Parse(text))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return strings.TrimSpace(buf.String()) + "\n"
}

func mysqlIdent(v string) string {
	return "`" + strings.ReplaceAll(v, "`", "``") + "`"
}
