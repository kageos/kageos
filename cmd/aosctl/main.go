package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultProdDir     = "deploy/prod"
	defaultConfigName  = "aos.yaml"
	defaultGenerated   = ".generated"
	defaultStorageRoot = "/data/ai-agent-os"

	defaultMainImage    = "localhost/agentos-main:latest"
	defaultAppBaseImage = "localhost/agentos-app-runtime-base:latest"
	defaultMySQLImage   = "docker.io/library/mysql:8.0"
	defaultNATSImage    = "docker.io/library/nats:2.10-alpine"
	defaultMinIOImage   = "docker.io/minio/minio:RELEASE.2025-09-07T16-13-09Z"
)

type Config struct {
	Site      SiteConfig      `yaml:"site"`
	Images    ImageConfig     `yaml:"images"`
	Storage   StorageConfig   `yaml:"storage"`
	MySQL     MySQLConfig     `yaml:"mysql"`
	NATS      NATSConfig      `yaml:"nats"`
	MinIO     MinIOConfig     `yaml:"minio"`
	Secrets   SecretsConfig   `yaml:"secrets"`
	SMTP      SMTPConfig      `yaml:"smtp"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
}

type SiteConfig struct {
	BaseURL      string `yaml:"base_url"`
	TLSMode      string `yaml:"tls_mode"`
	CertsHostDir string `yaml:"certs_host_dir"`
	CertFile     string `yaml:"cert_file"`
	KeyFile      string `yaml:"key_file"`
}

type ImageConfig struct {
	Main    string `yaml:"main"`
	AppBase string `yaml:"app_base"`
	MySQL   string `yaml:"mysql"`
	NATS    string `yaml:"nats"`
	MinIO   string `yaml:"minio"`
}

type StorageConfig struct {
	Root string `yaml:"root"`
}

type MySQLConfig struct {
	Mode              string `yaml:"mode"`
	Host              string `yaml:"host"`
	Port              int    `yaml:"port"`
	User              string `yaml:"user"`
	Password          string `yaml:"password"`
	AppDatabase       string `yaml:"app_database"`
	SchedulerDatabase string `yaml:"scheduler_database"`
	AgentDatabase     string `yaml:"agent_database"`
	StorageDatabase   string `yaml:"storage_database"`
	HRDatabase        string `yaml:"hr_database"`
	BackupAdminUser   string `yaml:"backup_admin_user"`
	BackupAdminPass   string `yaml:"backup_admin_password"`
	CreateBundledSQL  bool   `yaml:"create_bundled_sql"`
}

type NATSConfig struct {
	Mode        string `yaml:"mode"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	AuthEnabled bool   `yaml:"auth_enabled"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	URL         string `yaml:"url"`
}

type MinIOConfig struct {
	Mode         string `yaml:"mode"`
	Endpoint     string `yaml:"endpoint"`
	RootUser     string `yaml:"root_user"`
	RootPassword string `yaml:"root_password"`
	AccessKey    string `yaml:"access_key"`
	SecretKey    string `yaml:"secret_key"`
	UseSSL       bool   `yaml:"use_ssl"`
	Region       string `yaml:"region"`
	Bucket       string `yaml:"bucket"`
}

type SecretsConfig struct {
	JWTSecret              string `yaml:"jwt_secret"`
	ControlEncKey          string `yaml:"control_enc_key"`
	BackupBasicAuthPass    string `yaml:"backup_basic_auth_password"`
	BackupBasicAuthUser    string `yaml:"backup_basic_auth_username"`
	GeneratedByAOSCtl      bool   `yaml:"generated_by_aosctl"`
	GeneratedAtUnixSeconds int64  `yaml:"generated_at_unix_seconds"`
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	FromName string `yaml:"from_name"`
}

type SchedulerConfig struct {
	HealthPort int `yaml:"health_port"`
}

type Paths struct {
	RepoRoot     string
	ProdDir      string
	ConfigPath   string
	GeneratedDir string
}

type RuntimeConfig struct {
	Config
	Paths              Paths
	MySQLHostForMain   string
	MySQLPortForMain   int
	NATSHostForMain    string
	NATSPortForMain    int
	MinIOHostForMain   string
	MinIOPortForMain   int
	MySQLAddress       string
	BackupMySQLAddress string
	BackupMySQLHost    string
	BackupMySQLPort    int
	NATSURL            string
	SDKNATSURL         string
	MinIOEndpoint      string
	BackupMinIOAddress string
	BackupMinIOHost    string
	BackupMinIOPort    int
	TLSCertsHostDir    string
	BackupListenHost   string
	IncludeMySQL       bool
	IncludeNATS        bool
	IncludeMinIO       bool
	NATSAuthUser       string
	NATSAuthPassword   string
	ComposeConfigPath  string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	cmd := args[0]
	opts, rest, err := parseCommonFlags(args[1:])
	if err != nil {
		return err
	}

	paths, err := resolvePaths(opts)
	if err != nil {
		return err
	}

	switch cmd {
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "init":
		return cmdInit(paths, rest)
	case "render":
		return cmdRender(paths)
	case "doctor":
		return cmdDoctor(paths)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

type commonOptions struct {
	ConfigPath string
	ProdDir    string
}

func parseCommonFlags(args []string) (commonOptions, []string, error) {
	opts := commonOptions{}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--config":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--config requires a path")
			}
			opts.ConfigPath = args[i]
		case "--prod-dir":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--prod-dir requires a path")
			}
			opts.ProdDir = args[i]
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

func printUsage() {
	fmt.Println(`aosctl manages AI-Agent-OS production deployment files.

Usage:
  aosctl init [--force] [--base-url URL] [--mysql-mode bundled|external]
  aosctl render [--config deploy/prod/aos.yaml]
  aosctl doctor [--config deploy/prod/aos.yaml]

The first version renders Compose/config files into deploy/prod/.generated.
Compose remains the container orchestration engine.`)
}

func resolvePaths(opts commonOptions) (Paths, error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return Paths{}, err
	}

	prodDir := opts.ProdDir
	if prodDir == "" {
		prodDir = filepath.Join(repoRoot, defaultProdDir)
	} else if !filepath.IsAbs(prodDir) {
		prodDir = filepath.Join(repoRoot, prodDir)
	}

	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = filepath.Join(prodDir, defaultConfigName)
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(repoRoot, configPath)
	}

	return Paths{
		RepoRoot:     repoRoot,
		ProdDir:      prodDir,
		ConfigPath:   configPath,
		GeneratedDir: filepath.Join(prodDir, defaultGenerated),
	}, nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if fileExists(filepath.Join(dir, "go.mod")) && dirExists(filepath.Join(dir, defaultProdDir)) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot find repository root from %s", wd)
		}
		dir = parent
	}
}

func cmdInit(paths Paths, args []string) error {
	force := false
	baseURL := ""
	mysqlMode := "bundled"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force":
			force = true
		case "--base-url":
			i++
			if i >= len(args) {
				return fmt.Errorf("--base-url requires a value")
			}
			baseURL = args[i]
		case "--mysql-mode":
			i++
			if i >= len(args) {
				return fmt.Errorf("--mysql-mode requires a value")
			}
			mysqlMode = args[i]
		default:
			return fmt.Errorf("init does not support argument %q", args[i])
		}
	}
	if err := validateMode("mysql.mode", mysqlMode); err != nil {
		return err
	}

	if fileExists(paths.ConfigPath) && !force {
		fmt.Printf("config already exists: %s\n", paths.ConfigPath)
		fmt.Println("use --force to overwrite it")
		return nil
	}

	cfg, err := defaultConfig()
	if err != nil {
		return err
	}
	cfg.Site.BaseURL = baseURL
	cfg.MySQL.Mode = mysqlMode
	if mysqlMode == "external" {
		cfg.MySQL.Host = ""
		cfg.MySQL.User = ""
		cfg.MySQL.Password = ""
	}

	if err := os.MkdirAll(filepath.Dir(paths.ConfigPath), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(paths.ConfigPath, data, 0600); err != nil {
		return err
	}

	fmt.Printf("created config: %s\n", paths.ConfigPath)
	fmt.Println("next: edit site.base_url and run `aosctl doctor`, then `aosctl render`")
	return nil
}

func cmdRender(paths Paths) error {
	cfg, err := loadConfig(paths)
	if err != nil {
		return err
	}
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}
	if err := renderAll(rt); err != nil {
		return err
	}
	fmt.Printf("rendered deployment files: %s\n", paths.GeneratedDir)
	fmt.Printf("compose file: %s\n", filepath.Join(paths.GeneratedDir, "docker-compose.yaml"))
	return nil
}

func cmdDoctor(paths Paths) error {
	cfg, err := loadConfig(paths)
	if err != nil {
		return err
	}
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		return err
	}

	checks := []doctorCheck{
		{name: "config validation", fn: func() error { return validateConfig(rt) }},
		{name: "compose command", fn: checkComposeCommand},
		{name: "linux host", fn: checkLinuxHost},
		{name: "storage root", fn: func() error { return checkStorageRoot(rt.Storage.Root) }},
		{name: "external dependencies", fn: func() error { return checkExternalDependencies(rt) }},
	}

	failures := 0
	for _, check := range checks {
		if err := check.fn(); err != nil {
			failures++
			fmt.Printf("[FAIL] %s: %v\n", check.name, err)
		} else {
			fmt.Printf("[ OK ] %s\n", check.name)
		}
	}
	if failures > 0 {
		return fmt.Errorf("doctor failed with %d issue(s)", failures)
	}
	fmt.Println("doctor passed")
	return nil
}

type doctorCheck struct {
	name string
	fn   func() error
}

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
	return cfg, nil
}

func defaultConfig() (Config, error) {
	jwt, err := randomHex(32)
	if err != nil {
		return Config{}, err
	}
	control, err := randomHex(16)
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
	backupPass, err := randomHex(48)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Site: SiteConfig{
			TLSMode:      "http",
			CertsHostDir: "./certs",
			CertFile:     "/app/tls/fullchain.pem",
			KeyFile:      "/app/tls/privkey.pem",
		},
		Images: ImageConfig{
			Main:    defaultMainImage,
			AppBase: defaultAppBaseImage,
			MySQL:   defaultMySQLImage,
			NATS:    defaultNATSImage,
			MinIO:   defaultMinIOImage,
		},
		Storage: StorageConfig{Root: defaultStorageRoot},
		MySQL: MySQLConfig{
			Mode:              "bundled",
			Host:              "127.0.0.1",
			Port:              3306,
			User:              "root",
			Password:          mysqlPass,
			AppDatabase:       "app_db",
			SchedulerDatabase: "app-scheduler",
			AgentDatabase:     "agent-server",
			StorageDatabase:   "app-storage",
			HRDatabase:        "hr-server",
			BackupAdminUser:   "root",
			BackupAdminPass:   mysqlPass,
			CreateBundledSQL:  true,
		},
		NATS: NATSConfig{
			Mode:        "bundled",
			Host:        "127.0.0.1",
			Port:        4222,
			AuthEnabled: false,
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
			Bucket:       "ai-agent-os",
		},
		Secrets: SecretsConfig{
			JWTSecret:              jwt,
			ControlEncKey:          control,
			BackupBasicAuthUser:    "admin",
			BackupBasicAuthPass:    backupPass,
			GeneratedByAOSCtl:      true,
			GeneratedAtUnixSeconds: time.Now().Unix(),
		},
		SMTP: SMTPConfig{
			Host:     "smtp.qq.com",
			Port:     587,
			FromName: "AI Agent OS",
		},
		Scheduler: SchedulerConfig{HealthPort: 9098},
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Site.TLSMode == "" {
		cfg.Site.TLSMode = "http"
	}
	if cfg.Site.CertsHostDir == "" {
		cfg.Site.CertsHostDir = "./certs"
	}
	if cfg.Site.CertFile == "" {
		cfg.Site.CertFile = "/app/tls/fullchain.pem"
	}
	if cfg.Site.KeyFile == "" {
		cfg.Site.KeyFile = "/app/tls/privkey.pem"
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
		cfg.Storage.Root = defaultStorageRoot
	}
	if cfg.MySQL.Mode == "" {
		cfg.MySQL.Mode = "bundled"
	}
	if cfg.MySQL.Port == 0 {
		cfg.MySQL.Port = 3306
	}
	if cfg.MySQL.AppDatabase == "" {
		cfg.MySQL.AppDatabase = "app_db"
	}
	if cfg.MySQL.SchedulerDatabase == "" {
		cfg.MySQL.SchedulerDatabase = "app-scheduler"
	}
	if cfg.MySQL.AgentDatabase == "" {
		cfg.MySQL.AgentDatabase = "agent-server"
	}
	if cfg.MySQL.StorageDatabase == "" {
		cfg.MySQL.StorageDatabase = "app-storage"
	}
	if cfg.MySQL.HRDatabase == "" {
		cfg.MySQL.HRDatabase = "hr-server"
	}
	if cfg.MySQL.BackupAdminUser == "" {
		cfg.MySQL.BackupAdminUser = cfg.MySQL.User
	}
	if cfg.MySQL.BackupAdminPass == "" {
		cfg.MySQL.BackupAdminPass = cfg.MySQL.Password
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
		cfg.MinIO.Bucket = "ai-agent-os"
	}
	if cfg.SMTP.Host == "" {
		cfg.SMTP.Host = "smtp.qq.com"
	}
	if cfg.SMTP.Port == 0 {
		cfg.SMTP.Port = 587
	}
	if cfg.SMTP.FromName == "" {
		cfg.SMTP.FromName = "AI Agent OS"
	}
	if cfg.Scheduler.HealthPort == 0 {
		cfg.Scheduler.HealthPort = 9098
	}
}

func buildRuntimeConfig(paths Paths, cfg Config) (RuntimeConfig, error) {
	rt := RuntimeConfig{
		Config:           cfg,
		Paths:            paths,
		IncludeMySQL:     cfg.MySQL.Mode == "bundled",
		IncludeNATS:      cfg.NATS.Mode == "bundled",
		IncludeMinIO:     cfg.MinIO.Mode == "bundled",
		BackupListenHost: "0.0.0.0",
	}

	rt.TLSCertsHostDir = resolveRelativePath(paths.ProdDir, cfg.Site.CertsHostDir)

	rt.MySQLHostForMain = cfg.MySQL.Host
	rt.MySQLPortForMain = cfg.MySQL.Port
	if cfg.MySQL.Mode == "bundled" {
		rt.MySQLHostForMain = "127.0.0.1"
		rt.MySQLPortForMain = 3306
	}
	rt.MySQLAddress = net.JoinHostPort(rt.MySQLHostForMain, strconv.Itoa(rt.MySQLPortForMain))
	rt.BackupMySQLAddress = rt.MySQLAddress
	rt.BackupMySQLHost = rt.MySQLHostForMain
	rt.BackupMySQLPort = rt.MySQLPortForMain
	if cfg.MySQL.Mode == "bundled" {
		rt.BackupMySQLAddress = "mysql:3306"
		rt.BackupMySQLHost = "mysql"
		rt.BackupMySQLPort = 3306
	}

	rt.NATSHostForMain, rt.NATSPortForMain = natsHostPort(cfg)
	rt.NATSURL = natsURLForMain(cfg)
	rt.SDKNATSURL = rt.NATSURL

	rt.MinIOEndpoint = cfg.MinIO.Endpoint
	rt.BackupMinIOAddress = cfg.MinIO.Endpoint
	if cfg.MinIO.Mode == "bundled" {
		rt.MinIOEndpoint = "127.0.0.1:9000"
		rt.BackupMinIOAddress = "minio:9000"
	}
	minioHost, minioPort, err := splitHostPortDefault(rt.MinIOEndpoint, 9000)
	if err != nil {
		return RuntimeConfig{}, err
	}
	rt.MinIOHostForMain = minioHost
	rt.MinIOPortForMain = minioPort
	backupMinIOHost, backupMinIOPort, err := splitHostPortDefault(rt.BackupMinIOAddress, 9000)
	if err != nil {
		return RuntimeConfig{}, err
	}
	rt.BackupMinIOHost = backupMinIOHost
	rt.BackupMinIOPort = backupMinIOPort

	if cfg.NATS.AuthEnabled {
		rt.NATSAuthUser = cfg.NATS.User
		rt.NATSAuthPassword = cfg.NATS.Password
	}
	rt.ComposeConfigPath = filepath.Join(paths.GeneratedDir, "docker-compose.yaml")
	return rt, nil
}

func validateConfig(rt RuntimeConfig) error {
	var errs []error
	if !strings.HasPrefix(rt.Site.BaseURL, "http://") && !strings.HasPrefix(rt.Site.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("site.base_url must start with http:// or https://"))
	}
	switch rt.Site.TLSMode {
	case "http", "https", "redirect", "external":
	default:
		errs = append(errs, fmt.Errorf("site.tls_mode must be http, https, redirect, or external"))
	}
	if rt.Site.TLSMode == "redirect" && !strings.HasPrefix(rt.Site.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("site.tls_mode=redirect requires https site.base_url"))
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
		if rt.MySQL.BackupAdminUser != rt.MySQL.User || rt.MySQL.BackupAdminPass != rt.MySQL.Password {
			errs = append(errs, fmt.Errorf("mysql.mode=bundled requires backup_admin_user/password to match mysql.user/password"))
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
	if len(rt.Secrets.JWTSecret) < 32 {
		errs = append(errs, fmt.Errorf("secrets.jwt_secret must be at least 32 chars"))
	}
	if len(rt.Secrets.ControlEncKey) != 32 {
		errs = append(errs, fmt.Errorf("secrets.control_enc_key must be exactly 32 chars"))
	}
	if len(rt.Secrets.BackupBasicAuthPass) < 16 {
		errs = append(errs, fmt.Errorf("secrets.backup_basic_auth_password must be at least 16 chars"))
	}
	return errors.Join(errs...)
}

func validateMode(name, value string) error {
	switch value {
	case "bundled", "external":
		return nil
	default:
		return fmt.Errorf("%s must be bundled or external", name)
	}
}

func renderAll(rt RuntimeConfig) error {
	if err := os.RemoveAll(rt.Paths.GeneratedDir); err != nil {
		return err
	}
	dirs := []string{
		rt.Paths.GeneratedDir,
		filepath.Join(rt.Paths.GeneratedDir, "config"),
		filepath.Join(rt.Paths.GeneratedDir, "infra"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"docker-compose.yaml":         renderTemplate(composeTemplate, rt),
		".env":                        renderTemplate(envTemplate, rt),
		"infra/nats-server.conf":      renderTemplate(natsConfigTemplate, rt),
		"infra/mysql-init.sql":        renderTemplate(mysqlInitTemplate, rt),
		"config/global.yaml":          renderTemplate(globalConfigTemplate, rt),
		"config/api-gateway.yaml":     renderTemplate(apiGatewayConfigTemplate, rt),
		"config/app-runtime.yaml":     renderTemplate(appRuntimeConfigTemplate, rt),
		"config/app-server.yaml":      renderTemplate(appServerConfigTemplate, rt),
		"config/app-storage.yaml":     renderTemplate(appStorageConfigTemplate, rt),
		"config/agent-server.yaml":    renderTemplate(agentServerConfigTemplate, rt),
		"config/hr-server.yaml":       renderTemplate(hrServerConfigTemplate, rt),
		"config/control-service.yaml": renderTemplate(controlServiceConfigTemplate, rt),
		"config/backup-service.yaml":  renderTemplate(backupServiceConfigTemplate, rt),
	}
	for rel, content := range files {
		mode := os.FileMode(0644)
		if rel == ".env" {
			mode = 0600
		}
		if err := os.WriteFile(filepath.Join(rt.Paths.GeneratedDir, rel), []byte(content), mode); err != nil {
			return err
		}
	}
	return nil
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

func checkComposeCommand() error {
	if _, err := exec.LookPath("podman"); err == nil {
		if err := exec.Command("podman", "compose", "version").Run(); err == nil {
			return nil
		}
	}
	if _, err := exec.LookPath("docker"); err == nil {
		if err := exec.Command("docker", "compose", "version").Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("podman compose or docker compose is required")
}

func checkLinuxHost() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("prod single-node deployment currently requires Linux, current=%s", runtime.GOOS)
	}
	return nil
}

func checkStorageRoot(root string) error {
	if fileExists(root) {
		return nil
	}
	parent := filepath.Dir(root)
	info, err := os.Stat(parent)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", parent)
	}
	return nil
}

func checkExternalDependencies(rt RuntimeConfig) error {
	var errs []error
	if rt.MySQL.Mode == "external" {
		errs = append(errs, checkTCP("mysql", rt.MySQL.Host, rt.MySQL.Port))
	}
	if rt.NATS.Mode == "external" {
		host, port := natsHostPort(rt.Config)
		errs = append(errs, checkTCP("nats", host, port))
	}
	if rt.MinIO.Mode == "external" {
		host, port, err := splitHostPortDefault(rt.MinIO.Endpoint, 9000)
		if err != nil {
			errs = append(errs, err)
		} else {
			errs = append(errs, checkTCP("minio", host, port))
		}
	}
	return errors.Join(errs...)
}

func checkTCP(label, host string, port int) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return fmt.Errorf("%s %s:%d not reachable: %w", label, host, port, err)
	}
	_ = conn.Close()
	return nil
}

func randomHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func resolveRelativePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(base, path))
}

func natsURLForMain(cfg Config) string {
	if cfg.NATS.Mode == "external" && cfg.NATS.URL != "" {
		return cfg.NATS.URL
	}
	userInfo := ""
	if cfg.NATS.AuthEnabled {
		userInfo = url.UserPassword(cfg.NATS.User, cfg.NATS.Password).String() + "@"
	}
	host := cfg.NATS.Host
	port := cfg.NATS.Port
	if cfg.NATS.Mode == "bundled" {
		host = "127.0.0.1"
		port = 4222
	}
	return fmt.Sprintf("nats://%s%s", userInfo, net.JoinHostPort(host, strconv.Itoa(port)))
}

func natsHostPort(cfg Config) (string, int) {
	if cfg.NATS.Mode == "external" && cfg.NATS.URL != "" {
		parsed, err := url.Parse(cfg.NATS.URL)
		if err == nil && parsed.Host != "" {
			host, port, err := splitHostPortDefault(parsed.Host, 4222)
			if err == nil {
				return host, port
			}
		}
	}
	host := cfg.NATS.Host
	port := cfg.NATS.Port
	if cfg.NATS.Mode == "bundled" {
		host = "127.0.0.1"
		port = 4222
	}
	return host, port
}

func splitHostPortDefault(value string, defaultPort int) (string, int, error) {
	host, portStr, err := net.SplitHostPort(value)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			return value, defaultPort, nil
		}
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
