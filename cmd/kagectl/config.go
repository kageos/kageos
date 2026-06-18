package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func loadConfig(paths Paths) (Config, error) {
	data, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", paths.ConfigPath, err)
	}
	cfg := Config{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", paths.ConfigPath, err)
	}
	applyDefaults(&cfg)
	applyEnvOverrides(&cfg)
	return cfg, nil
}

func loadRuntimeConfig(paths Paths) (RuntimeConfig, error) {
	cfg, err := loadConfig(paths)
	if err != nil {
		return RuntimeConfig{}, err
	}
	return buildRuntimeConfig(paths, cfg)
}

func ensureRuntimeLayout(rt RuntimeConfig) error {
	dirs := []string{
		filepath.Join(rt.Storage.Root, "mysql"),
		filepath.Join(rt.Storage.Root, "minio"),
		filepath.Join(rt.Storage.Root, "podman_storage"),
		filepath.Join(rt.Storage.Root, "logs"),
		filepath.Join(rt.Storage.Root, "namespace"),
		filepath.Join(rt.Storage.Root, "data", "runtime", "app-runtime"),
		filepath.Join(rt.Storage.Root, "data", "tmp"),
		rt.TLSCertsHostDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create runtime directory %s: %w", dir, err)
		}
	}
	return nil
}

func requireGeneratedCompose(paths Paths) error {
	composePath := filepath.Join(paths.GeneratedDir, "docker-compose.yaml")
	if !fileExists(composePath) {
		return fmt.Errorf("generated compose not found: %s; run `kagectl render` or `kagectl up` first", composePath)
	}
	return nil
}

func defaultConfig() (Config, error) {
	jwt, err := randomHex(32)
	if err != nil {
		return Config{}, err
	}
	appDBSecret, err := randomHex(32)
	if err != nil {
		return Config{}, err
	}
	mysqlPass, err := randomHex(32)
	if err != nil {
		return Config{}, err
	}
	natsPass, err := randomHex(24)
	if err != nil {
		return Config{}, err
	}
	minioPass, err := randomHex(32)
	if err != nil {
		return Config{}, err
	}
	systemUserPass, err := randomHex(24)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Timezone: defaultTimezone,
		Site: SiteConfig{
			TLSMode:  "http",
			CertFile: "/app/tls/fullchain.pem",
			KeyFile:  "/app/tls/privkey.pem",
		},
		Images: ImageConfig{
			Main:    defaultMainImage,
			AppBase: defaultAppBaseImage,
			MySQL:   defaultMySQLImage,
			NATS:    defaultNATSImage,
			MinIO:   defaultMinIOImage,
		},
		Storage: StorageConfig{Root: defaultStorageRoot()},
		MySQL: MySQLConfig{
			Mode:              "bundled",
			Host:              "127.0.0.1",
			Port:              3306,
			User:              "root",
			Password:          mysqlPass,
			AppDatabase:       "app-server",
			AgentDatabase:     "agent-server",
			ConnectorDatabase: "connector-server",
			StorageDatabase:   "app-storage",
			HRDatabase:        "hr-server",
			TimerDatabase:     "timer-scheduler",
			MessageDatabase:   "message-server",
			CreateBundledSQL:  true,
		},
		NATS: NATSConfig{
			Mode:        "bundled",
			Host:        "127.0.0.1",
			Port:        4222,
			AuthEnabled: true,
			User:        "aos",
			Password:    natsPass,
		},
		MinIO: MinIOConfig{
			Mode:         "bundled",
			Endpoint:     "127.0.0.1:9000",
			RootUser:     "minioadmin",
			RootPassword: minioPass,
			AccessKey:    "minioadmin",
			SecretKey:    minioPass,
			UseSSL:       false,
			Region:       "us-east-1",
			Bucket:       "kageos",
		},
		Company: CompanyConfig{
			Code: "default",
			Name: "Default",
		},
		Auth: AuthConfig{
			RegistrationMode: "admin_only",
		},
		Secrets: SecretsConfig{
			JWTSecret:              jwt,
			AppDBSecret:            appDBSecret,
			GeneratedByKageCtl:     true,
			GeneratedAtUnixSeconds: time.Now().Unix(),
		},
		SystemUser: SystemUserConfig{
			Password: systemUserPass,
		},
		SMTP: SMTPConfig{
			Mode:     "smtp",
			Host:     "smtp.qq.com",
			Port:     587,
			FromName: "Kageos",
		},
	}
	return cfg, nil
}

func defaultDevDeploymentConfig(secrets devSecrets) Config {
	return Config{
		Timezone: defaultTimezone,
		Site: SiteConfig{
			BaseURL:  "http://localhost:5173",
			TLSMode:  "http",
			CertFile: "/app/tls/fullchain.pem",
			KeyFile:  "/app/tls/privkey.pem",
		},
		Images: ImageConfig{
			Main:    defaultMainImage,
			AppBase: defaultAppBaseImage,
			MySQL:   defaultMySQLImage,
			NATS:    defaultNATSImage,
			MinIO:   defaultMinIOImage,
		},
		Storage: StorageConfig{Root: defaultStorageRoot()},
		MySQL: MySQLConfig{
			Mode:              "external",
			Host:              "127.0.0.1",
			Port:              3318,
			User:              "root",
			Password:          secrets.MySQLRootPassword,
			AppDatabase:       "app-server",
			AgentDatabase:     "agent-server",
			ConnectorDatabase: "connector-server",
			StorageDatabase:   "app-storage",
			HRDatabase:        "hr-server",
			TimerDatabase:     "timer-scheduler",
			MessageDatabase:   "message-server",
			CreateBundledSQL:  true,
		},
		NATS: NATSConfig{
			Mode:        "external",
			Host:        "127.0.0.1",
			Port:        4222,
			AuthEnabled: true,
			User:        secrets.NATSUser,
			Password:    secrets.NATSPassword,
			URL:         fmt.Sprintf("nats://%s:%s@127.0.0.1:4222", url.QueryEscape(secrets.NATSUser), url.QueryEscape(secrets.NATSPassword)),
		},
		MinIO: MinIOConfig{
			Mode:         "external",
			Endpoint:     "127.0.0.1:9000",
			RootUser:     secrets.MinIORootUser,
			RootPassword: secrets.MinIORootPassword,
			AccessKey:    secrets.MinIORootUser,
			SecretKey:    secrets.MinIORootPassword,
			UseSSL:       false,
			Region:       "us-east-1",
			Bucket:       "kageos",
		},
		Company: CompanyConfig{
			Code: "default",
			Name: "Default",
		},
		Auth: AuthConfig{
			RegistrationMode: "debug_code",
		},
		Secrets: SecretsConfig{
			JWTSecret:              secrets.JWTSecret,
			AppDBSecret:            secrets.AppDBSecret,
			GeneratedByKageCtl:     true,
			GeneratedAtUnixSeconds: time.Now().Unix(),
		},
		SystemUser: SystemUserConfig{Password: secrets.SystemUserPassword},
		SMTP: SMTPConfig{
			Mode:     "log",
			Host:     "smtp.qq.com",
			Port:     587,
			FromName: "Kageos",
		},
	}
}

func applyDefaults(cfg *Config) {
	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.Timezone == "" {
		cfg.Timezone = defaultTimezone
	}
	if cfg.Site.TLSMode == "" {
		cfg.Site.TLSMode = "http"
	}
	if cfg.Site.CertFile == "" {
		cfg.Site.CertFile = "/app/tls/fullchain.pem"
	}
	if cfg.Site.KeyFile == "" {
		cfg.Site.KeyFile = "/app/tls/privkey.pem"
	}
	if cfg.Site.HTTPPort == 0 {
		cfg.Site.HTTPPort = defaultSiteHTTPPort(cfg.Site)
	}
	if cfg.Site.HTTPSPort == 0 {
		cfg.Site.HTTPSPort = defaultSiteHTTPSPort(cfg.Site)
	}
	if cfg.Images.Main == "" {
		cfg.Images.Main = defaultMainImage
	}
	if cfg.Images.AppBase == "" {
		cfg.Images.AppBase = defaultAppBaseImage
	}
	if cfg.Images.MySQL == "" {
		cfg.Images.MySQL = defaultMySQLImage
	}
	if cfg.Images.NATS == "" {
		cfg.Images.NATS = defaultNATSImage
	}
	if cfg.Images.MinIO == "" {
		cfg.Images.MinIO = defaultMinIOImage
	}
	if cfg.Storage.Root == "" {
		cfg.Storage.Root = defaultStorageRoot()
	}
	if cfg.Company.Code == "" {
		cfg.Company.Code = "default"
	}
	if cfg.Company.Name == "" {
		cfg.Company.Name = "Default"
	}
	if cfg.Auth.RegistrationMode == "" {
		if cfg.SMTP.Mode == "log" {
			cfg.Auth.RegistrationMode = "debug_code"
		} else {
			cfg.Auth.RegistrationMode = "admin_only"
		}
	}
	if cfg.Secrets.AppDBSecret == "" {
		cfg.Secrets.AppDBSecret = cfg.Secrets.JWTSecret
	}
	if cfg.MySQL.Mode == "" {
		cfg.MySQL.Mode = "bundled"
	}
	if cfg.MySQL.Port == 0 {
		cfg.MySQL.Port = 3306
	}
	if cfg.MySQL.AppDatabase == "" {
		cfg.MySQL.AppDatabase = "app-server"
	}
	if cfg.MySQL.AgentDatabase == "" {
		cfg.MySQL.AgentDatabase = "agent-server"
	}
	if cfg.MySQL.ConnectorDatabase == "" {
		cfg.MySQL.ConnectorDatabase = "connector-server"
	}
	if cfg.MySQL.StorageDatabase == "" {
		cfg.MySQL.StorageDatabase = "app-storage"
	}
	if cfg.MySQL.HRDatabase == "" {
		cfg.MySQL.HRDatabase = "hr-server"
	}
	if cfg.MySQL.TimerDatabase == "" {
		cfg.MySQL.TimerDatabase = "timer-scheduler"
	}
	if cfg.MySQL.MessageDatabase == "" {
		cfg.MySQL.MessageDatabase = "message-server"
	}
	if cfg.NATS.Mode == "" {
		cfg.NATS.Mode = "bundled"
	}
	if cfg.NATS.Port == 0 {
		cfg.NATS.Port = 4222
	}
	if cfg.NATS.User == "" {
		cfg.NATS.User = "aos"
	}
	if cfg.MinIO.Mode == "" {
		cfg.MinIO.Mode = "bundled"
	}
	if cfg.MinIO.RootUser == "" {
		cfg.MinIO.RootUser = "minioadmin"
	}
	if cfg.MinIO.AccessKey == "" {
		cfg.MinIO.AccessKey = cfg.MinIO.RootUser
	}
	if cfg.MinIO.SecretKey == "" {
		cfg.MinIO.SecretKey = cfg.MinIO.RootPassword
	}
	if cfg.MinIO.Region == "" {
		cfg.MinIO.Region = "us-east-1"
	}
	if cfg.MinIO.Bucket == "" {
		cfg.MinIO.Bucket = "kageos"
	}
	if cfg.SMTP.Host == "" {
		cfg.SMTP.Host = "smtp.qq.com"
	}
	if cfg.SMTP.Mode == "" {
		cfg.SMTP.Mode = "smtp"
	}
	if cfg.SMTP.Port == 0 {
		cfg.SMTP.Port = 587
	}
	if cfg.SMTP.FromName == "" {
		cfg.SMTP.FromName = "Kageos"
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("KAGEOS_TIMEZONE")); v != "" {
		cfg.Timezone = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_BASE_URL")); v != "" {
		cfg.Site.BaseURL = v
	}
	if port, ok, err := parsePortEnv("KAGEOS_HTTP_PORT"); err != nil {
		fmt.Printf("WARN: %v\n", err)
	} else if ok {
		cfg.Site.HTTPPort = port
	}
	if port, ok, err := parsePortEnv("KAGEOS_HTTPS_PORT"); err != nil {
		fmt.Printf("WARN: %v\n", err)
	} else if ok {
		cfg.Site.HTTPSPort = port
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_BASE_IMAGE")); v != "" {
		cfg.Images.AppBase = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_COMPANY_CODE")); v != "" {
		cfg.Company.Code = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_COMPANY_NAME")); v != "" {
		cfg.Company.Name = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_COMPANY_LOGO_URL")); v != "" {
		cfg.Company.LogoURL = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_REGISTRATION_MODE")); v != "" {
		cfg.Auth.RegistrationMode = v
	}
	if v := strings.TrimSpace(os.Getenv("SMTP_MODE")); v != "" {
		cfg.SMTP.Mode = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_TLS_MODE")); v != "" {
		cfg.Site.TLSMode = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_TLS_CERT_PEM_B64")); v != "" {
		cfg.Site.TLSCertPEMB64 = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_TLS_KEY_PEM_B64")); v != "" {
		cfg.Site.TLSKeyPEMB64 = v
	}
}

func parsePortEnv(name string) (int, bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return 0, false, nil
	}
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0, false, fmt.Errorf("%s must be a TCP port between 1 and 65535, got %q", name, value)
	}
	return port, true, nil
}

func buildRuntimeConfig(paths Paths, cfg Config) (RuntimeConfig, error) {
	applyDefaults(&cfg)
	rt := RuntimeConfig{
		Config:                  cfg,
		Paths:                   paths,
		IncludeMySQL:            cfg.MySQL.Mode == "bundled",
		IncludeNATS:             cfg.NATS.Mode == "bundled",
		IncludeMinIO:            cfg.MinIO.Mode == "bundled",
		AppBaseBuilderImage:     defaultAppBaseBuilderImage,
		AppContainerNetworkMode: "host",
	}

	rt.TLSCertsHostDir = filepath.Join(paths.GeneratedDir, "tls")

	rt.MySQLHostForMain = cfg.MySQL.Host
	rt.MySQLPortForMain = cfg.MySQL.Port
	if cfg.MySQL.Mode == "bundled" {
		rt.MySQLHostForMain = "127.0.0.1"
		rt.MySQLPortForMain = 3306
	}
	rt.MySQLAddress = net.JoinHostPort(rt.MySQLHostForMain, strconv.Itoa(rt.MySQLPortForMain))
	rt.AppDBClusterKey = appDBClusterKey(rt.MySQLHostForMain, rt.MySQLPortForMain)

	rt.NATSHostForMain, rt.NATSPortForMain = natsHostPort(cfg)
	rt.NATSURL = natsURLForMain(cfg)
	rt.SDKNATSURL = natsURLForSDK(cfg)
	rt.SDKGatewayURL = "http://127.0.0.1:9090"

	rt.MinIOEndpoint = cfg.MinIO.Endpoint
	rt.SDKMinIOEndpoint = sdkMinIOEndpoint(cfg.MinIO.Endpoint)
	if cfg.MinIO.Mode == "bundled" {
		rt.MinIOEndpoint = "127.0.0.1:9000"
		rt.SDKMinIOEndpoint = "127.0.0.1:9000"
	}
	minioHost, minioPort, err := splitHostPortDefault(rt.MinIOEndpoint, 9000)
	if err != nil {
		return RuntimeConfig{}, err
	}
	rt.MinIOHostForMain = minioHost
	rt.MinIOPortForMain = minioPort

	if cfg.NATS.AuthEnabled {
		rt.NATSAuthUser = cfg.NATS.User
		rt.NATSAuthPassword = cfg.NATS.Password
	}
	rt.ComposeConfigPath = filepath.Join(paths.GeneratedDir, "docker-compose.yaml")
	rt.EnvFilePath = filepath.Join(paths.GeneratedDir, "env", "kageos.env")
	rt.SummaryPath = filepath.Join(paths.GeneratedDir, "kageos-deployment-summary.md")
	rt.LLMSeedEnvVars = uniqueLLMSeedEnvVars(cfg.LLMs.Configs)
	return rt, nil
}

func appDBClusterKey(host string, port int) string {
	value := strings.ToLower(strings.TrimSpace(host)) + ":" + strconv.Itoa(port)
	sum := sha256.Sum256([]byte(value))
	return "mysql_" + hex.EncodeToString(sum[:])[:12]
}

func sdkMinIOEndpoint(endpoint string) string {
	host, port, err := splitHostPortDefault(endpoint, 9000)
	if err != nil {
		return endpoint
	}
	// Production app containers are started by the nested Podman inside main.
	// deploy/prod/Dockerfile sets containers.conf netns=host, so local MinIO is
	// reachable on loopback. host.containers.internal resolves to a bridge
	// gateway and cannot reach services bound to 127.0.0.1.
	if isLocalHostForContainer(host) {
		return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	return endpoint
}

func devSDKMinIOEndpoint(endpoint string) string {
	host, port, err := splitHostPortDefault(endpoint, 9000)
	if err != nil {
		return endpoint
	}
	if isLocalHostForContainer(host) {
		return net.JoinHostPort("host.containers.internal", strconv.Itoa(port))
	}
	return endpoint
}

func isLocalHostForContainer(host string) bool {
	host = strings.ToLower(strings.Trim(strings.TrimSpace(host), "[]"))
	if host == "localhost" || host == "host.containers.internal" || host == "host.docker.internal" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateConfig(rt RuntimeConfig) error {
	var errs []error
	if err := validateTimezoneValue("timezone", rt.Timezone); err != nil {
		errs = append(errs, err)
	}
	if !strings.HasPrefix(rt.Site.BaseURL, "http://") && !strings.HasPrefix(rt.Site.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("site.base_url must start with http:// or https://"))
	}
	if err := validateTCPPort("site.http_port", rt.Site.HTTPPort); err != nil {
		errs = append(errs, err)
	}
	if err := validateTCPPort("site.https_port", rt.Site.HTTPSPort); err != nil {
		errs = append(errs, err)
	}
	switch rt.Site.TLSMode {
	case "http", "https", "redirect", "external":
	default:
		errs = append(errs, fmt.Errorf("site.tls_mode must be http, https, redirect, or external"))
	}
	if rt.Site.TLSMode == "redirect" && !strings.HasPrefix(rt.Site.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("site.tls_mode=redirect requires https site.base_url"))
	}
	if rt.Site.TLSMode == "https" || rt.Site.TLSMode == "redirect" {
		if strings.TrimSpace(rt.Site.TLSCertPEMB64) == "" || strings.TrimSpace(rt.Site.TLSKeyPEMB64) == "" {
			errs = append(errs, fmt.Errorf("site.tls_mode=%s requires KAGEOS_TLS_CERT_PEM_B64 and KAGEOS_TLS_KEY_PEM_B64, or site.tls_cert_pem_b64/site.tls_key_pem_b64", rt.Site.TLSMode))
		}
	}
	if !filepath.IsAbs(rt.Storage.Root) {
		errs = append(errs, fmt.Errorf("storage.root must be absolute"))
	}
	if err := validateMode("mysql.mode", rt.MySQL.Mode); err != nil {
		errs = append(errs, err)
	}
	if err := validateMode("nats.mode", rt.NATS.Mode); err != nil {
		errs = append(errs, err)
	}
	if err := validateMode("minio.mode", rt.MinIO.Mode); err != nil {
		errs = append(errs, err)
	}
	if rt.MySQL.Port <= 0 {
		errs = append(errs, fmt.Errorf("mysql.port is required"))
	}
	if rt.MySQL.Mode == "external" && rt.MySQL.Host == "" {
		errs = append(errs, fmt.Errorf("mysql.host is required when mysql.mode=external"))
	}
	if rt.MySQL.User == "" || rt.MySQL.Password == "" {
		errs = append(errs, fmt.Errorf("mysql.user/mysql.password are required"))
	}
	if rt.MySQL.Mode == "bundled" {
		if rt.MySQL.User != "root" {
			errs = append(errs, fmt.Errorf("mysql.mode=bundled currently requires mysql.user=root"))
		}
	}
	if rt.NATS.AuthEnabled && (rt.NATS.User == "" || rt.NATS.Password == "") {
		errs = append(errs, fmt.Errorf("nats auth requires nats.user and nats.password"))
	}
	if rt.NATS.Mode == "external" && rt.NATS.URL == "" && rt.NATS.Host == "" {
		errs = append(errs, fmt.Errorf("nats.host or nats.url is required when nats.mode=external"))
	}
	if rt.MinIO.Endpoint == "" {
		errs = append(errs, fmt.Errorf("minio.endpoint is required"))
	}
	if rt.MinIO.AccessKey == "" || rt.MinIO.SecretKey == "" {
		errs = append(errs, fmt.Errorf("minio.access_key/minio.secret_key are required"))
	}
	if rt.MinIO.Mode == "bundled" {
		if rt.MinIO.RootUser == "" || rt.MinIO.RootPassword == "" {
			errs = append(errs, fmt.Errorf("minio.mode=bundled requires minio.root_user/minio.root_password"))
		}
		if rt.MinIO.RootUser != rt.MinIO.AccessKey || rt.MinIO.RootPassword != rt.MinIO.SecretKey {
			errs = append(errs, fmt.Errorf("minio.mode=bundled requires root credentials to match access_key/secret_key"))
		}
	}
	if strings.TrimSpace(rt.Company.Code) == "" {
		errs = append(errs, fmt.Errorf("company.code is required"))
	} else if !defaultCompanyCodePattern.MatchString(rt.Company.Code) {
		errs = append(errs, fmt.Errorf("company.code can only contain letters, numbers, underscores, and hyphens"))
	}
	if strings.TrimSpace(rt.Company.Name) == "" {
		errs = append(errs, fmt.Errorf("company.name is required"))
	}
	if err := validateRegistrationMode(rt.Auth.RegistrationMode); err != nil {
		errs = append(errs, err)
	}
	if err := validateSMTPMode(rt.SMTP.Mode); err != nil {
		errs = append(errs, err)
	}
	if rt.Auth.RegistrationMode == "email_code" && rt.SMTP.Mode != "smtp" {
		errs = append(errs, fmt.Errorf("auth.registration_mode=email_code requires smtp.mode=smtp"))
	}
	if rt.Auth.RegistrationMode == "email_code" {
		if strings.TrimSpace(rt.SMTP.Host) == "" ||
			rt.SMTP.Port == 0 ||
			strings.TrimSpace(rt.SMTP.Username) == "" ||
			strings.TrimSpace(rt.SMTP.Password) == "" ||
			strings.TrimSpace(rt.SMTP.From) == "" {
			errs = append(errs, fmt.Errorf("auth.registration_mode=email_code requires complete smtp host/port/username/password/from"))
		}
	}
	if len(rt.Secrets.JWTSecret) < 32 {
		errs = append(errs, fmt.Errorf("secrets.jwt_secret must be at least 32 chars"))
	}
	if len(rt.Secrets.AppDBSecret) < 32 {
		errs = append(errs, fmt.Errorf("secrets.app_db_secret must be at least 32 chars"))
	}
	if rt.SystemUser.Password == "" {
		errs = append(errs, fmt.Errorf("system_user.password is required"))
	}
	if err := validateLLMSeeds(rt.LLMs); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateTCPPort(name string, port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", name)
	}
	return nil
}

func validateRegistrationMode(mode string) error {
	switch strings.TrimSpace(mode) {
	case "", "admin_only", "email_code", "debug_code":
		return nil
	default:
		return fmt.Errorf("auth.registration_mode must be admin_only, email_code, or debug_code")
	}
}

func validateSMTPMode(mode string) error {
	switch strings.TrimSpace(mode) {
	case "", "smtp", "log":
		return nil
	default:
		return fmt.Errorf("smtp.mode must be smtp or log")
	}
}

func validateTimezoneValue(name string, timezone string) error {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return nil
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return fmt.Errorf("%s must be a valid IANA timezone, got %q", name, timezone)
	}
	return nil
}

func uniqueLLMSeedEnvVars(configs []LLMSeedConfig) []string {
	seen := map[string]struct{}{}
	var names []string
	for _, cfg := range configs {
		name := strings.TrimSpace(cfg.APIKeyEnv)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func validateLLMSeeds(seeds LLMSeedsConfig) error {
	var errs []error
	codes := map[string]struct{}{}
	defaultCode := strings.TrimSpace(seeds.Default)
	defaultFound := defaultCode == ""
	defaultCount := 0
	for i, cfg := range seeds.Configs {
		prefix := fmt.Sprintf("llms.configs[%d]", i)
		code := strings.TrimSpace(cfg.Code)
		if code == "" {
			errs = append(errs, fmt.Errorf("%s.code is required", prefix))
			continue
		}
		if _, ok := codes[code]; ok {
			errs = append(errs, fmt.Errorf("llms.configs code %q is duplicated", code))
		}
		codes[code] = struct{}{}
		if strings.TrimSpace(cfg.Name) == "" {
			errs = append(errs, fmt.Errorf("%s.name is required", prefix))
		}
		if strings.TrimSpace(cfg.Model) == "" {
			errs = append(errs, fmt.Errorf("%s.model is required", prefix))
		}
		if envName := strings.TrimSpace(cfg.APIKeyEnv); envName != "" && !isValidEnvName(envName) {
			errs = append(errs, fmt.Errorf("%s.api_key_env must be a valid environment variable name", prefix))
		}
		if cfg.IsDefault {
			defaultCount++
		}
		if defaultCode != "" && code == defaultCode {
			defaultFound = true
		}
	}
	if defaultCount > 1 && defaultCode == "" {
		errs = append(errs, fmt.Errorf("llms.configs can contain at most one is_default=true when llms.default is empty"))
	}
	if !defaultFound {
		errs = append(errs, fmt.Errorf("llms.default %q must match one llms.configs[].code", defaultCode))
	}
	return errors.Join(errs...)
}

func isValidEnvName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func validateMode(name, value string) error {
	switch value {
	case "bundled", "external":
		return nil
	default:
		return fmt.Errorf("%s must be bundled or external", name)
	}
}
