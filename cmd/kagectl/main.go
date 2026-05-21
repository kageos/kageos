package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

var defaultCompanyCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const (
	defaultProdDir     = ".kageos/prod"
	defaultConfigName  = "kage.yaml"
	defaultGenerated   = "generated"
	defaultDevConfig   = ".kageos/dev/config"
	defaultStorageRoot = "/data/kageos"

	defaultMainImage    = "localhost/kageos-main:latest"
	defaultAppBaseImage = "kagebase:latest"
	defaultMySQLImage   = "docker.io/library/mysql:8.0"
	defaultNATSImage    = "docker.io/library/nats:2.10-alpine"
	defaultMinIOImage   = "docker.io/minio/minio:RELEASE.2025-09-07T16-13-09Z"

	defaultUpVerifyTimeout  = 5 * time.Minute
	defaultUpVerifyInterval = 5 * time.Second
)

type Config struct {
	Site       SiteConfig       `yaml:"site"`
	Images     ImageConfig      `yaml:"images"`
	Storage    StorageConfig    `yaml:"storage"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	NATS       NATSConfig       `yaml:"nats"`
	MinIO      MinIOConfig      `yaml:"minio"`
	Company    CompanyConfig    `yaml:"company"`
	Secrets    SecretsConfig    `yaml:"secrets"`
	SystemUser SystemUserConfig `yaml:"system_user"`
	LLMs       LLMSeedsConfig   `yaml:"llms"`
	SMTP       SMTPConfig       `yaml:"smtp"`
}

type SiteConfig struct {
	BaseURL       string `yaml:"base_url"`
	TLSMode       string `yaml:"tls_mode"`
	CertFile      string `yaml:"cert_file"`
	KeyFile       string `yaml:"key_file"`
	TLSCertPEMB64 string `yaml:"tls_cert_pem_b64,omitempty"`
	TLSKeyPEMB64  string `yaml:"tls_key_pem_b64,omitempty"`
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
	Mode             string `yaml:"mode"`
	Host             string `yaml:"host"`
	Port             int    `yaml:"port"`
	User             string `yaml:"user"`
	Password         string `yaml:"password"`
	AppDatabase      string `yaml:"app_database"`
	AgentDatabase    string `yaml:"agent_database"`
	StorageDatabase  string `yaml:"storage_database"`
	HRDatabase       string `yaml:"hr_database"`
	CreateBundledSQL bool   `yaml:"create_bundled_sql"`
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

type CompanyConfig struct {
	Code    string `yaml:"code"`
	Name    string `yaml:"name"`
	LogoURL string `yaml:"logo_url"`
}

type SecretsConfig struct {
	JWTSecret              string `yaml:"jwt_secret"`
	GeneratedByKageCtl     bool   `yaml:"generated_by_kagectl"`
	GeneratedAtUnixSeconds int64  `yaml:"generated_at_unix_seconds"`
}

type SystemUserConfig struct {
	Password string `yaml:"password"`
}

type LLMSeedsConfig struct {
	Default string          `yaml:"default"`
	Configs []LLMSeedConfig `yaml:"configs"`
}

type LLMSeedConfig struct {
	Code        string `yaml:"code"`
	Name        string `yaml:"name"`
	Model       string `yaml:"model"`
	APIKey      string `yaml:"api_key"`
	APIKeyEnv   string `yaml:"api_key_env"`
	APIBase     string `yaml:"api_base"`
	Timeout     int    `yaml:"timeout"`
	MaxTokens   int    `yaml:"max_tokens"`
	ExtraConfig string `yaml:"extra_config"`
	IsDefault   bool   `yaml:"is_default"`
	Visibility  int    `yaml:"visibility"`
	Admin       string `yaml:"admin"`
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	FromName string `yaml:"from_name"`
}

type Paths struct {
	RepoRoot     string
	ProdDir      string
	ConfigPath   string
	GeneratedDir string
}

type RuntimeConfig struct {
	Config
	Paths             Paths
	MySQLHostForMain  string
	MySQLPortForMain  int
	NATSHostForMain   string
	NATSPortForMain   int
	MinIOHostForMain  string
	MinIOPortForMain  int
	MySQLAddress      string
	NATSURL           string
	SDKNATSURL        string
	SDKGatewayURL     string
	MinIOEndpoint     string
	SDKMinIOEndpoint  string
	TLSCertsHostDir   string
	IncludeMySQL      bool
	IncludeNATS       bool
	IncludeMinIO      bool
	NATSAuthUser      string
	NATSAuthPassword  string
	ComposeConfigPath string
	LLMSeedEnvVars    []string
	EnvFilePath       string
	SummaryPath       string
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
	case "init-dev", "dev-init", "init-local":
		return cmdInitDev(paths, rest)
	case "bootstrap":
		return cmdBootstrap(paths, rest)
	case "build-app-base":
		return cmdBuildAppBase(paths, rest)
	case "render":
		return cmdRender(paths)
	case "layers", "topology":
		return cmdLayers(paths, rest)
	case "doctor":
		return cmdDoctor(paths, rest)
	case "up":
		return cmdUp(paths, rest)
	case "verify":
		return cmdVerify(paths, rest)
	case "status", "ps":
		return cmdStatus(paths, rest)
	case "logs":
		return cmdLogs(paths, rest)
	case "down":
		return cmdDown(paths)
	case "uninstall", "reset":
		return cmdUninstall(paths, rest)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

type commonOptions struct {
	ConfigPath string
	ProdDir    string
}

type outputOptions struct {
	JSON bool
}

type upOptions struct {
	UseImage      bool
	NoBuild       bool
	SkipVerify    bool
	VerifyTimeout time.Duration
}

type uninstallOptions struct {
	PurgeData          bool
	PurgePodmanStorage bool
	PurgeImages        bool
	KeepGenerated      bool
	PurgePrivateConfig bool
	Force              bool
	DryRun             bool
}

type initOptions struct {
	Force       bool
	BaseURL     string
	MySQLMode   string
	CompanyCode string
	CompanyName string
}

type bootstrapOptions struct {
	Init   initOptions
	UpArgs []string
}

type buildAppBaseOptions struct {
	Image   string
	Force   bool
	NoCache bool
}

type initDevOptions struct {
	Engine       string
	SkipBase     bool
	BaseImage    string
	BaseForce    bool
	BaseNoCache  bool
	RegenSecrets bool
	CompanyCode  string
	CompanyName  string
}

type devSecrets struct {
	MySQLRootPassword  string
	NATSUser           string
	NATSPassword       string
	MinIORootUser      string
	MinIORootPassword  string
	JWTSecret          string
	SystemUserPassword string
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

func parseOutputFlags(command string, args []string) (outputOptions, []string, error) {
	opts := outputOptions{}
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--json":
			opts.JSON = true
		default:
			rest = append(rest, arg)
		}
	}
	if len(rest) > 0 {
		return opts, rest, fmt.Errorf("%s does not support argument %q", command, rest[0])
	}
	return opts, rest, nil
}

func parseUpFlags(args []string) (upOptions, error) {
	opts := upOptions{VerifyTimeout: defaultUpVerifyTimeout}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--image":
			opts.UseImage = true
		case "--no-build":
			opts.NoBuild = true
		case "--skip-verify":
			opts.SkipVerify = true
		case "--wait-timeout":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--wait-timeout requires a duration, e.g. 5m or 30s")
			}
			timeout, err := time.ParseDuration(args[i])
			if err != nil {
				return opts, fmt.Errorf("parse --wait-timeout: %w", err)
			}
			if timeout <= 0 {
				return opts, fmt.Errorf("--wait-timeout must be greater than 0")
			}
			opts.VerifyTimeout = timeout
		default:
			return opts, fmt.Errorf("up does not support argument %q", args[i])
		}
	}
	if opts.UseImage && opts.NoBuild {
		return opts, fmt.Errorf("--image and --no-build cannot be used together")
	}
	return opts, nil
}

func parseUninstallFlags(args []string) (uninstallOptions, error) {
	opts := uninstallOptions{}
	for _, arg := range args {
		switch arg {
		case "--purge-data":
			opts.PurgeData = true
		case "--purge-podman-storage":
			opts.PurgePodmanStorage = true
		case "--purge-images":
			opts.PurgeImages = true
		case "--keep-generated":
			opts.KeepGenerated = true
		case "--purge-private-config":
			opts.PurgePrivateConfig = true
		case "--force":
			opts.Force = true
		case "--dry-run":
			opts.DryRun = true
		default:
			return opts, fmt.Errorf("uninstall does not support argument %q", arg)
		}
	}
	if opts.PurgePodmanStorage && !opts.PurgeData {
		return opts, fmt.Errorf("--purge-podman-storage requires --purge-data")
	}
	if uninstallRequiresForce(opts) && !opts.Force && !opts.DryRun {
		return opts, fmt.Errorf("destructive uninstall options require --force; preview with --dry-run first")
	}
	return opts, nil
}

func parseInitFlags(command string, args []string) (initOptions, error) {
	opts := initOptions{MySQLMode: "bundled"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force":
			opts.Force = true
		case "--base-url":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--base-url requires a value")
			}
			opts.BaseURL = args[i]
		case "--mysql-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--mysql-mode requires a value")
			}
			opts.MySQLMode = args[i]
		case "--company-code":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-code requires a value")
			}
			opts.CompanyCode = strings.TrimSpace(args[i])
		case "--company-name":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-name requires a value")
			}
			opts.CompanyName = strings.TrimSpace(args[i])
		default:
			return opts, fmt.Errorf("%s does not support argument %q", command, args[i])
		}
	}
	if err := validateMode("mysql.mode", opts.MySQLMode); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseBootstrapFlags(args []string) (bootstrapOptions, error) {
	opts := bootstrapOptions{Init: initOptions{MySQLMode: "bundled"}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force":
			opts.Init.Force = true
		case "--base-url":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--base-url requires a value")
			}
			opts.Init.BaseURL = args[i]
		case "--mysql-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--mysql-mode requires a value")
			}
			opts.Init.MySQLMode = args[i]
		case "--image", "--no-build", "--skip-verify":
			opts.UpArgs = append(opts.UpArgs, args[i])
		case "--wait-timeout":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--wait-timeout requires a duration, e.g. 5m or 30s")
			}
			opts.UpArgs = append(opts.UpArgs, "--wait-timeout", args[i])
		default:
			return opts, fmt.Errorf("bootstrap does not support argument %q", args[i])
		}
	}
	if err := validateMode("mysql.mode", opts.Init.MySQLMode); err != nil {
		return opts, err
	}
	if _, err := parseUpFlags(opts.UpArgs); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseBuildAppBaseFlags(args []string) (buildAppBaseOptions, error) {
	opts := buildAppBaseOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--image":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--image requires a value")
			}
			opts.Image = strings.TrimSpace(args[i])
			if opts.Image == "" {
				return opts, fmt.Errorf("--image cannot be empty")
			}
		case "--force":
			opts.Force = true
		case "--no-cache":
			opts.NoCache = true
		default:
			return opts, fmt.Errorf("build-app-base does not support argument %q", args[i])
		}
	}
	return opts, nil
}

func parseInitDevFlags(args []string) (initDevOptions, error) {
	opts := initDevOptions{Engine: "podman"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--engine":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--engine requires auto, docker, or podman")
			}
			opts.Engine = strings.TrimSpace(args[i])
		case "--skip-base":
			opts.SkipBase = true
		case "--base-image":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--base-image requires a value")
			}
			opts.BaseImage = strings.TrimSpace(args[i])
			if opts.BaseImage == "" {
				return opts, fmt.Errorf("--base-image cannot be empty")
			}
		case "--base-force":
			opts.BaseForce = true
		case "--base-no-cache":
			opts.BaseNoCache = true
		case "--regen-secrets":
			opts.RegenSecrets = true
		case "--company-code":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-code requires a value")
			}
			opts.CompanyCode = strings.TrimSpace(args[i])
		case "--company-name":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-name requires a value")
			}
			opts.CompanyName = strings.TrimSpace(args[i])
		default:
			if args[i] == "auto" || args[i] == "docker" || args[i] == "podman" {
				opts.Engine = args[i]
				continue
			}
			return opts, fmt.Errorf("init-dev does not support argument %q", args[i])
		}
	}
	if opts.Engine != "auto" && opts.Engine != "docker" && opts.Engine != "podman" {
		return opts, fmt.Errorf("--engine requires auto, docker, or podman")
	}
	return opts, nil
}

func printUsage() {
	fmt.Println(`kagectl manages Kageos production deployment files.

Usage:
  kagectl init [--force] [--base-url URL] [--mysql-mode bundled|external] [--company-code CODE] [--company-name NAME]
  kagectl init-dev [--engine podman|docker|auto] [--skip-base] [--regen-secrets] [--base-image IMAGE] [--base-force] [--base-no-cache] [--company-code CODE] [--company-name NAME]
  kagectl bootstrap --base-url URL [--mysql-mode bundled|external] [--image|--no-build] [--skip-verify] [--wait-timeout 5m]
  kagectl build-app-base [--image IMAGE] [--force] [--no-cache]
  kagectl render [--config .kageos/prod/kage.yaml]
  kagectl layers [--config .kageos/prod/kage.yaml] [--json]
  kagectl doctor [--config .kageos/prod/kage.yaml] [--json]
  kagectl up [--config .kageos/prod/kage.yaml] [--image|--no-build] [--skip-verify] [--wait-timeout 5m]
  kagectl verify [--config .kageos/prod/kage.yaml] [--json]
  kagectl status [--config .kageos/prod/kage.yaml] [--json]
  kagectl logs [--config .kageos/prod/kage.yaml] [service|layer] [--layer L0-L5]
  kagectl down [--config .kageos/prod/kage.yaml]
  kagectl uninstall [--config .kageos/prod/kage.yaml] [--purge-data] [--purge-podman-storage] [--purge-images] [--keep-generated] [--purge-private-config] [--force] [--dry-run]

Compose remains the container execution engine; kagectl owns layered config rendering, deployment orchestration, and diagnostics.`)
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
		if fileExists(filepath.Join(dir, "go.mod")) && dirExists(filepath.Join(dir, "deploy", "prod")) {
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
	opts, err := parseInitFlags("init", args)
	if err != nil {
		return err
	}
	created, err := writeInitialConfig(paths, opts)
	if err != nil {
		return err
	}
	if created {
		fmt.Println("next: edit site.base_url if needed and run `kagectl up`")
	}
	return nil
}

func cmdInitDev(paths Paths, args []string) error {
	opts, err := parseInitDevFlags(args)
	if err != nil {
		return err
	}
	if err := renderDevConfig(paths, opts.RegenSecrets, opts.CompanyCode, opts.CompanyName); err != nil {
		return err
	}
	if err := runDevInfraScript(paths, opts); err != nil {
		return err
	}
	if opts.SkipBase {
		fmt.Println("==> skip app base image (--skip-base)")
		printDevInitSummary(paths, opts)
		return nil
	}
	fmt.Println("==> ensure app base image")
	if err := runBuildAppBaseScript(paths, buildAppBaseOptions{
		Image:   opts.BaseImage,
		Force:   opts.BaseForce,
		NoCache: opts.BaseNoCache,
	}); err != nil {
		return err
	}
	printDevInitSummary(paths, opts)
	return nil
}

func cmdBootstrap(paths Paths, args []string) error {
	opts, err := parseBootstrapFlags(args)
	if err != nil {
		return err
	}
	if fileExists(paths.ConfigPath) && !opts.Init.Force {
		fmt.Printf("using existing config: %s\n", paths.ConfigPath)
	} else {
		if opts.Init.BaseURL == "" {
			return fmt.Errorf("bootstrap requires --base-url when creating config")
		}
		if _, err := writeInitialConfig(paths, opts.Init); err != nil {
			return err
		}
	}
	return cmdUp(paths, opts.UpArgs)
}

func cmdBuildAppBase(paths Paths, args []string) error {
	opts, err := parseBuildAppBaseFlags(args)
	if err != nil {
		return err
	}
	return runBuildAppBaseScript(paths, opts)
}

func writeInitialConfig(paths Paths, opts initOptions) (bool, error) {
	if fileExists(paths.ConfigPath) && !opts.Force {
		fmt.Printf("config already exists: %s\n", paths.ConfigPath)
		fmt.Println("use --force to overwrite it")
		return false, nil
	}

	cfg, err := defaultConfig()
	if err != nil {
		return false, err
	}
	cfg.Site.BaseURL = opts.BaseURL
	cfg.MySQL.Mode = opts.MySQLMode
	if opts.CompanyCode != "" {
		cfg.Company.Code = opts.CompanyCode
	}
	if opts.CompanyName != "" {
		cfg.Company.Name = opts.CompanyName
	}
	applyEnvOverrides(&cfg)
	if opts.MySQLMode == "external" {
		cfg.MySQL.Host = ""
		cfg.MySQL.User = ""
		cfg.MySQL.Password = ""
	}

	if err := os.MkdirAll(filepath.Dir(paths.ConfigPath), 0755); err != nil {
		return false, err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(paths.ConfigPath, data, 0600); err != nil {
		return false, err
	}

	fmt.Printf("created config: %s\n", paths.ConfigPath)
	return true, nil
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

func cmdLayers(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("layers", args)
	if err != nil {
		return err
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}
	if opts.JSON {
		return writeJSON(buildDeploymentReport(rt))
	}
	printDeploymentLayers(rt)
	return nil
}

func cmdDoctor(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("doctor", args)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(paths)
	if err != nil {
		return err
	}
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		return err
	}

	if opts.JSON {
		return runLayerChecksJSON("doctor", doctorLayerChecks(rt))
	}
	return runLayerChecks("doctor", doctorLayerChecks(rt))
}

func cmdUp(paths Paths, args []string) error {
	opts, err := parseUpFlags(args)
	if err != nil {
		return err
	}
	fmt.Println("[L0 部署控制层] 检查宿主机和读取配置")
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	fmt.Println("[L0-L1] 运行分层部署预检")
	if err := runLayerChecks("doctor", doctorLayerChecks(rt)); err != nil {
		return err
	}
	fmt.Println("[L0 部署控制层] 校验配置、准备运行目录、渲染 Compose/config")
	if err := ensureRuntimeLayout(rt); err != nil {
		return err
	}
	if err := renderAll(rt); err != nil {
		return err
	}

	switch {
	case opts.UseImage:
		fmt.Println("[L3 平台服务层] 拉取主镜像")
		if err := runCompose(rt.Paths.GeneratedDir, "pull", "main"); err != nil {
			return err
		}
	case opts.NoBuild:
		fmt.Println("[L3 平台服务层] 跳过主镜像构建/拉取 (--no-build)")
	default:
		fmt.Println("[L3 平台服务层] 构建主镜像")
		if err := runCompose(rt.Paths.GeneratedDir, "build", "main"); err != nil {
			return err
		}
	}

	fmt.Println("[L4 运行时管理层] 准备用户应用基础镜像")
	if err := runCompose(rt.Paths.GeneratedDir, "run", "--rm", "--no-deps", "-e", "KAGEOS_APP_BASE_ACTION=ensure", "-e", "KAGEOS_APP_BASE_BUILD_NO_CACHE=0", "--entrypoint", "/app/entrypoint-app-base.sh", "main"); err != nil {
		return err
	}

	if infraServices := composeServicesForLayer(rt, layerInfra); len(infraServices) > 0 {
		fmt.Printf("[L1 基础设施层] 启动基础设施服务: %s\n", strings.Join(infraServices, ", "))
		args := append([]string{"up", "-d", "--no-build"}, infraServices...)
		if err := runCompose(rt.Paths.GeneratedDir, args...); err != nil {
			return err
		}
	}
	if checks := startupDependencyChecks(rt); len(checks) > 0 {
		fmt.Printf("[L1 基础设施层] 等待基础设施可用（timeout %s）\n", opts.VerifyTimeout)
		if err := waitLayerChecks("startup dependencies", checks, opts.VerifyTimeout, defaultUpVerifyInterval); err != nil {
			return err
		}
	}

	fmt.Println("[L2-L4] 启动应用服务栈")
	if err := runCompose(rt.Paths.GeneratedDir, "up", "-d", "--no-build"); err != nil {
		return err
	}
	if opts.SkipVerify {
		fmt.Println("deployment started; layered verify skipped (--skip-verify)")
		if err := finishDeploymentSummary(rt, "started (verification skipped)"); err != nil {
			return err
		}
		return nil
	}
	fmt.Printf("[L0-L5] 等待分层健康检查通过（timeout %s）\n", opts.VerifyTimeout)
	if err := waitLayerChecks("verify", verifyLayerChecks(rt), opts.VerifyTimeout, defaultUpVerifyInterval); err != nil {
		return err
	}
	fmt.Println("deployment ready")
	if err := finishDeploymentSummary(rt, "ready"); err != nil {
		return err
	}
	return nil
}

func cmdVerify(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("verify", args)
	if err != nil {
		return err
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if opts.JSON {
		return runLayerChecksJSON("verify", verifyLayerChecks(rt))
	}
	return runLayerChecks("verify", verifyLayerChecks(rt))
}

func cmdStatus(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("status", args)
	if err != nil {
		return err
	}
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}
	if opts.JSON {
		composePS, err := runComposeOutput(paths.GeneratedDir, "ps")
		if err != nil {
			return err
		}
		return writeJSON(statusReport{
			Deployment:        buildDeploymentReport(rt),
			ComposeConfigPath: rt.ComposeConfigPath,
			ComposePS:         composePS,
		})
	}
	fmt.Println("Deployment layers")
	printDeploymentLayers(rt)
	fmt.Println("\nCompose service ownership")
	printComposeServiceOwnership(rt)
	fmt.Println("\nCompose services")
	return runCompose(paths.GeneratedDir, "ps")
}

func cmdLogs(paths Paths, args []string) error {
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	layerArg := ""
	targets := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--layer":
			i++
			if i >= len(args) {
				return fmt.Errorf("--layer requires L0-L5, edge, infra, platform, runtime, or apps")
			}
			layerArg = args[i]
		default:
			targets = append(targets, args[i])
		}
	}
	if layerArg != "" && len(targets) > 0 {
		return fmt.Errorf("logs cannot combine --layer with a service name")
	}
	if len(targets) > 1 {
		return fmt.Errorf("logs accepts at most one service name or layer")
	}

	target := "main"
	if len(targets) == 1 {
		target = targets[0]
	}
	if layerArg == "" {
		if _, ok := parseDeploymentLayer(target); !ok {
			return runCompose(paths.GeneratedDir, "logs", "-f", target)
		}
	}

	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}

	if layerArg != "" {
		return runLayerLogs(rt, layerArg)
	}
	return runLayerLogs(rt, target)
}

func cmdDown(paths Paths) error {
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	return runCompose(paths.GeneratedDir, "down")
}

func cmdUninstall(paths Paths, args []string) error {
	opts, err := parseUninstallFlags(args)
	if err != nil {
		return err
	}

	rt, rtErr := loadRuntimeConfig(paths)
	if rtErr != nil && uninstallNeedsRuntimeConfig(opts, paths) {
		return rtErr
	}

	printUninstallPlan(paths, rt, rtErr, opts)
	if opts.DryRun {
		return nil
	}

	composeReady, err := ensureGeneratedComposeForUninstall(paths, rt, rtErr)
	if err != nil {
		return err
	}
	if composeReady {
		downArgs := []string{"down"}
		if opts.PurgeImages {
			downArgs = append(downArgs, "--rmi", "all")
		}
		if err := runCompose(paths.GeneratedDir, downArgs...); err != nil {
			return err
		}
	}

	if opts.PurgeData {
		for _, target := range uninstallDataTargets(rt, opts) {
			if err := removePath(target.Path, target.Label, false); err != nil {
				return err
			}
		}
	}
	if !opts.KeepGenerated {
		if err := removePath(paths.GeneratedDir, "generated deployment files", false); err != nil {
			return err
		}
	}
	if opts.PurgePrivateConfig {
		if err := removePath(paths.ConfigPath, "private deploy config", false); err != nil {
			return err
		}
	}
	fmt.Println("uninstall completed")
	return nil
}

func runLayerLogs(rt RuntimeConfig, value string) error {
	layer, ok := parseDeploymentLayer(value)
	if !ok {
		return fmt.Errorf("unknown deployment layer %q", value)
	}
	services := composeServicesForLayer(rt, layer)
	if len(services) == 0 {
		return fmt.Errorf("%s has no Compose service logs; %s", layerTitle(layer), layerLogHint(layer))
	}
	fmt.Printf("%s logs: %s\n", layerTitle(layer), strings.Join(services, ", "))
	args := append([]string{"logs", "-f"}, services...)
	return runCompose(rt.Paths.GeneratedDir, args...)
}

func layerLogHint(layer deploymentLayerID) string {
	switch layer {
	case layerControl:
		return "kagectl runs on the host and logs to the current terminal"
	case layerApps:
		return "user App containers are managed by app-runtime; use platform app log APIs"
	default:
		return "check `kagectl status` for layer to service ownership"
	}
}

func runLayerChecksJSON(name string, checks []layerCheck) error {
	report := runLayerChecksReport(name, checks)
	if err := writeJSON(report); err != nil {
		return err
	}
	if !report.OK {
		return fmt.Errorf("%s failed with %d issue(s)", name, report.Failures)
	}
	return nil
}

func waitLayerChecks(name string, checks []layerCheck, timeout, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	started := time.Now()
	deadline := started.Add(timeout)
	attempt := 0
	var last checkReport
	for {
		attempt++
		last = runLayerChecksReport(name, checks)
		if last.OK {
			fmt.Printf("%s passed after %s\n", name, time.Since(started).Round(time.Second))
			return nil
		}
		if time.Now().After(deadline) {
			fmt.Printf("\n%s final failure report\n", name)
			printLayerCheckReport(last)
			return fmt.Errorf("%s did not pass within %s (%d issue(s) remaining)", name, timeout, last.Failures)
		}
		remaining := time.Until(deadline)
		sleep := interval
		if remaining < sleep {
			sleep = remaining
		}
		fmt.Printf("  waiting for %s: %d issue(s) remaining (attempt %d, next check in %s)\n", name, last.Failures, attempt, sleep.Round(time.Second))
		time.Sleep(sleep)
	}
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func doctorLayerChecks(rt RuntimeConfig) []layerCheck {
	checks := []layerCheck{
		{Layer: layerControl, Name: "config validation", Target: rt.Paths.ConfigPath, Fn: func() error { return validateConfig(rt) }},
		{Layer: layerControl, Name: "compose command", Target: "podman compose / docker compose", Fn: checkComposeCommand},
		{Layer: layerControl, Name: "linux host", Target: runtime.GOOS, Fn: checkLinuxHost},
		{Layer: layerInfra, Name: "storage root parent", Target: rt.Storage.Root, Fn: func() error { return checkStorageRoot(rt.Storage.Root) }},
	}
	checks = appendExternalDependencyChecks(checks, rt)
	return checks
}

func appendExternalDependencyChecks(checks []layerCheck, rt RuntimeConfig) []layerCheck {
	if rt.MySQL.Mode == "external" {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "external mysql",
			Target: tcpTarget(rt.MySQL.Host, rt.MySQL.Port),
			Fn:     func() error { return checkTCP("mysql", rt.MySQL.Host, rt.MySQL.Port) },
		})
	}
	if rt.NATS.Mode == "external" {
		host, port := natsHostPort(rt.Config)
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "external nats",
			Target: tcpTarget(host, port),
			Fn:     func() error { return checkTCP("nats", host, port) },
		})
	}
	if rt.MinIO.Mode == "external" {
		host, port, err := splitHostPortDefault(rt.MinIO.Endpoint, 9000)
		check := layerCheck{
			Layer:  layerInfra,
			Name:   "external minio",
			Target: rt.MinIO.Endpoint,
		}
		if err != nil {
			check.Fn = func() error { return err }
		} else {
			check.Target = tcpTarget(host, port)
			check.Fn = func() error { return checkTCP("minio", host, port) }
		}
		checks = append(checks, check)
	}
	return checks
}

func startupDependencyChecks(rt RuntimeConfig) []layerCheck {
	checks := make([]layerCheck, 0, 3)
	if rt.IncludeMySQL {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "mysql initialized",
			Target: "compose exec mysql SELECT required databases",
			Fn:     func() error { return checkBundledMySQLInitialized(rt) },
		})
	} else {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "mysql tcp",
			Target: rt.MySQLAddress,
			Fn:     func() error { return checkTCP("mysql", rt.MySQLHostForMain, rt.MySQLPortForMain) },
		})
	}
	if rt.IncludeNATS {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "nats tcp",
			Target: tcpTarget(rt.NATSHostForMain, rt.NATSPortForMain),
			Fn:     func() error { return checkTCP("nats", rt.NATSHostForMain, rt.NATSPortForMain) },
		})
	}
	if rt.IncludeMinIO {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "minio tcp",
			Target: tcpTarget(rt.MinIOHostForMain, rt.MinIOPortForMain),
			Fn:     func() error { return checkTCP("minio", rt.MinIOHostForMain, rt.MinIOPortForMain) },
		})
	}
	return checks
}

func verifyLayerChecks(rt RuntimeConfig) []layerCheck {
	checks := []layerCheck{
		{Layer: layerControl, Name: "config validation", Target: rt.Paths.ConfigPath, Fn: func() error { return validateConfig(rt) }},
		{Layer: layerControl, Name: "rendered compose", Target: rt.ComposeConfigPath, Fn: func() error { return requireGeneratedCompose(rt.Paths) }},
		{Layer: layerEdge, Name: "nginx http listener", Target: "127.0.0.1:80", Fn: func() error { return checkTCP("nginx", "127.0.0.1", 80) }},
		{Layer: layerEdge, Name: "main edge probe", Target: "compose exec main /app/health/edge.sh", Fn: func() error {
			return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "/app/health/edge.sh")
		}},
	}
	checks = append(checks[:2], append(startupDependencyChecks(rt), checks[2:]...)...)
	if rt.Site.TLSMode == "https" || rt.Site.TLSMode == "redirect" {
		checks = append(checks, layerCheck{Layer: layerEdge, Name: "nginx https listener", Target: "127.0.0.1:443", Fn: func() error { return checkTCP("nginx", "127.0.0.1", 443) }})
	}
	checks = append(checks,
		layerCheck{Layer: layerPlatform, Name: "api-gateway", Target: "http://127.0.0.1:9090/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9090/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-server", Target: "http://127.0.0.1:9091/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9091/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-storage", Target: "http://127.0.0.1:9092/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9092/health") }},
		layerCheck{Layer: layerPlatform, Name: "agent-server", Target: "http://127.0.0.1:9095/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9095/health") }},
		layerCheck{Layer: layerPlatform, Name: "hr-server", Target: "http://127.0.0.1:9097/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9097/health") }},
		layerCheck{Layer: layerPlatform, Name: "main platform probe", Target: "compose exec main /app/health/platform.sh", Fn: func() error {
			return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "/app/health/platform.sh")
		}},
		layerCheck{Layer: layerRuntime, Name: "app-runtime", Target: "http://127.0.0.1:9093/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9093/health") }},
		layerCheck{Layer: layerRuntime, Name: "main runtime probe", Target: "compose exec main /app/health/runtime.sh", Fn: func() error {
			return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "/app/health/runtime.sh")
		}},
	)
	checks = appendSDKEndpointChecks(checks, rt)
	return checks
}

func appendSDKEndpointChecks(checks []layerCheck, rt RuntimeConfig) []layerCheck {
	checks = append(checks, layerCheck{
		Layer:  layerApps,
		Name:   "sdk gateway endpoint",
		Target: rt.SDKGatewayURL,
		Fn:     func() error { return requireContains(rt.SDKGatewayURL, "127.0.0.1") },
	})
	if rt.NATS.Mode == "bundled" {
		checks = append(checks, layerCheck{
			Layer:  layerApps,
			Name:   "sdk nats endpoint",
			Target: redactURLCredentials(rt.SDKNATSURL),
			Fn:     func() error { return requireContains(rt.SDKNATSURL, "127.0.0.1") },
		})
	}
	if rt.MinIO.Mode == "bundled" {
		checks = append(checks, layerCheck{
			Layer:  layerApps,
			Name:   "sdk minio endpoint",
			Target: rt.SDKMinIOEndpoint,
			Fn:     func() error { return requireContains(rt.SDKMinIOEndpoint, "host.containers.internal") },
		})
	}
	return checks
}

func requireContains(value, needle string) error {
	if !strings.Contains(value, needle) {
		return fmt.Errorf("expected %q to contain %q", value, needle)
	}
	return nil
}

func uninstallRequiresForce(opts uninstallOptions) bool {
	return opts.PurgeData || opts.PurgePodmanStorage || opts.PurgeImages || opts.PurgePrivateConfig
}

func uninstallNeedsRuntimeConfig(opts uninstallOptions, paths Paths) bool {
	if opts.PurgeData || opts.PurgePodmanStorage {
		return true
	}
	return !fileExists(filepath.Join(paths.GeneratedDir, "docker-compose.yaml"))
}

func ensureGeneratedComposeForUninstall(paths Paths, rt RuntimeConfig, rtErr error) (bool, error) {
	if fileExists(filepath.Join(paths.GeneratedDir, "docker-compose.yaml")) {
		return true, nil
	}
	if rtErr != nil {
		fmt.Printf("generated compose not found and config unavailable; skipping compose down: %v\n", rtErr)
		return false, nil
	}
	fmt.Println("generated compose not found; rendering it from config before uninstall")
	if err := renderAll(rt); err != nil {
		return false, err
	}
	return true, nil
}

type uninstallTarget struct {
	Label string
	Path  string
}

func uninstallDataTargets(rt RuntimeConfig, opts uninstallOptions) []uninstallTarget {
	root := filepath.Clean(rt.Storage.Root)
	targets := []uninstallTarget{
		{Label: "mysql data", Path: filepath.Join(root, "mysql")},
		{Label: "minio data", Path: filepath.Join(root, "minio")},
		{Label: "user namespace", Path: filepath.Join(root, "namespace")},
		{Label: "app data", Path: filepath.Join(root, "data")},
		{Label: "logs", Path: filepath.Join(root, "logs")},
	}
	if opts.PurgePodmanStorage {
		targets = append(targets, uninstallTarget{Label: "podman storage and app-base image", Path: filepath.Join(root, "podman_storage")})
	}
	return targets
}

func printUninstallPlan(paths Paths, rt RuntimeConfig, rtErr error, opts uninstallOptions) {
	fmt.Println("uninstall plan")
	if fileExists(filepath.Join(paths.GeneratedDir, "docker-compose.yaml")) {
		if opts.PurgeImages {
			fmt.Println("  - compose down --rmi all (stop/remove services and host engine images)")
		} else {
			fmt.Println("  - compose down (stop/remove services; keep host engine images)")
		}
	} else if rtErr == nil {
		fmt.Println("  - render generated compose, then compose down")
	} else {
		fmt.Printf("  - skip compose down: generated compose missing and config unavailable (%v)\n", rtErr)
	}
	if opts.PurgeData {
		for _, target := range uninstallDataTargets(rt, opts) {
			fmt.Printf("  - remove %s: %s\n", target.Label, target.Path)
		}
		if !opts.PurgePodmanStorage {
			fmt.Printf("  - keep podman storage/app-base image: %s\n", filepath.Join(filepath.Clean(rt.Storage.Root), "podman_storage"))
		}
	} else if rtErr == nil {
		fmt.Printf("  - keep runtime data: %s\n", rt.Storage.Root)
	}
	if opts.KeepGenerated {
		fmt.Printf("  - keep generated files: %s\n", paths.GeneratedDir)
	} else {
		fmt.Printf("  - remove generated files: %s\n", paths.GeneratedDir)
	}
	if opts.PurgePrivateConfig {
		fmt.Printf("  - remove private config: %s\n", paths.ConfigPath)
	} else {
		fmt.Printf("  - keep private config: %s\n", paths.ConfigPath)
	}
	if opts.DryRun {
		fmt.Println("dry-run only; no changes made")
	}
}

func removePath(path, label string, dryRun bool) error {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." || clean == string(filepath.Separator) {
		return fmt.Errorf("refuse to remove unsafe %s path: %q", label, path)
	}
	if dryRun {
		fmt.Printf("[dry-run] remove %s: %s\n", label, clean)
		return nil
	}
	if !fileExists(clean) && !dirExists(clean) {
		fmt.Printf("skip missing %s: %s\n", label, clean)
		return nil
	}
	fmt.Printf("remove %s: %s\n", label, clean)
	return os.RemoveAll(clean)
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
		Storage: StorageConfig{Root: defaultStorageRoot},
		MySQL: MySQLConfig{
			Mode:             "bundled",
			Host:             "127.0.0.1",
			Port:             3306,
			User:             "root",
			Password:         mysqlPass,
			AppDatabase:      "app-server",
			AgentDatabase:    "agent-server",
			StorageDatabase:  "app-storage",
			HRDatabase:       "hr-server",
			CreateBundledSQL: true,
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
		Secrets: SecretsConfig{
			JWTSecret:              jwt,
			GeneratedByKageCtl:     true,
			GeneratedAtUnixSeconds: time.Now().Unix(),
		},
		SystemUser: SystemUserConfig{
			Password: systemUserPass,
		},
		SMTP: SMTPConfig{
			Host:     "smtp.qq.com",
			Port:     587,
			FromName: "Kageos",
		},
	}
	return cfg, nil
}

func defaultDevDeploymentConfig(secrets devSecrets) Config {
	return Config{
		Site: SiteConfig{
			BaseURL:  "http://127.0.0.1:9090",
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
		Storage: StorageConfig{Root: defaultStorageRoot},
		MySQL: MySQLConfig{
			Mode:             "external",
			Host:             "127.0.0.1",
			Port:             3318,
			User:             "root",
			Password:         secrets.MySQLRootPassword,
			AppDatabase:      "app-server",
			AgentDatabase:    "agent-server",
			StorageDatabase:  "app-storage",
			HRDatabase:       "hr-server",
			CreateBundledSQL: true,
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
		Secrets: SecretsConfig{
			JWTSecret:              secrets.JWTSecret,
			GeneratedByKageCtl:     true,
			GeneratedAtUnixSeconds: time.Now().Unix(),
		},
		SystemUser: SystemUserConfig{Password: secrets.SystemUserPassword},
		SMTP: SMTPConfig{
			Host:     "smtp.qq.com",
			Port:     587,
			FromName: "Kageos",
		},
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Site.TLSMode == "" {
		cfg.Site.TLSMode = "http"
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
	if cfg.Company.Code == "" {
		cfg.Company.Code = "default"
	}
	if cfg.Company.Name == "" {
		cfg.Company.Name = "Default"
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
	if cfg.MySQL.StorageDatabase == "" {
		cfg.MySQL.StorageDatabase = "app-storage"
	}
	if cfg.MySQL.HRDatabase == "" {
		cfg.MySQL.HRDatabase = "hr-server"
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
	if cfg.SMTP.Port == 0 {
		cfg.SMTP.Port = 587
	}
	if cfg.SMTP.FromName == "" {
		cfg.SMTP.FromName = "Kageos"
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("KAGEOS_BASE_URL")); v != "" {
		cfg.Site.BaseURL = v
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

func buildRuntimeConfig(paths Paths, cfg Config) (RuntimeConfig, error) {
	rt := RuntimeConfig{
		Config:       cfg,
		Paths:        paths,
		IncludeMySQL: cfg.MySQL.Mode == "bundled",
		IncludeNATS:  cfg.NATS.Mode == "bundled",
		IncludeMinIO: cfg.MinIO.Mode == "bundled",
	}

	rt.TLSCertsHostDir = filepath.Join(paths.GeneratedDir, "tls")

	rt.MySQLHostForMain = cfg.MySQL.Host
	rt.MySQLPortForMain = cfg.MySQL.Port
	if cfg.MySQL.Mode == "bundled" {
		rt.MySQLHostForMain = "127.0.0.1"
		rt.MySQLPortForMain = 3306
	}
	rt.MySQLAddress = net.JoinHostPort(rt.MySQLHostForMain, strconv.Itoa(rt.MySQLPortForMain))

	rt.NATSHostForMain, rt.NATSPortForMain = natsHostPort(cfg)
	rt.NATSURL = natsURLForMain(cfg)
	rt.SDKNATSURL = natsURLForSDK(cfg)
	rt.SDKGatewayURL = "http://127.0.0.1:9090"

	rt.MinIOEndpoint = cfg.MinIO.Endpoint
	rt.SDKMinIOEndpoint = sdkMinIOEndpoint(cfg.MinIO.Endpoint)
	if cfg.MinIO.Mode == "bundled" {
		rt.MinIOEndpoint = "127.0.0.1:9000"
		rt.SDKMinIOEndpoint = "host.containers.internal:9000"
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

func sdkMinIOEndpoint(endpoint string) string {
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
	if len(rt.Secrets.JWTSecret) < 32 {
		errs = append(errs, fmt.Errorf("secrets.jwt_secret must be at least 32 chars"))
	}
	if rt.SystemUser.Password == "" {
		errs = append(errs, fmt.Errorf("system_user.password is required"))
	}
	if err := validateLLMSeeds(rt.LLMs); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
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
		"docker-compose.yaml":      renderTemplate(composeTemplate, rt),
		".env":                     renderTemplate(envTemplate, rt),
		"env/kageos.env":           renderTemplate(envTemplate, rt),
		"infra/nats-server.conf":   renderTemplate(natsConfigTemplate, rt),
		"infra/mysql-init.sql":     renderTemplate(mysqlInitTemplate, rt),
		"config/global.yaml":       renderTemplate(globalConfigTemplate, rt),
		"config/api-gateway.yaml":  renderTemplate(apiGatewayConfigTemplate, rt),
		"config/app-runtime.yaml":  renderTemplate(appRuntimeConfigTemplate, rt),
		"config/app-server.yaml":   renderTemplate(appServerConfigTemplate, rt),
		"config/app-storage.yaml":  renderTemplate(appStorageConfigTemplate, rt),
		"config/agent-server.yaml": renderTemplate(agentServerConfigTemplate, rt),
		"config/hr-server.yaml":    renderTemplate(hrServerConfigTemplate, rt),
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

func renderDevConfig(paths Paths, regenSecrets bool, companyCode string, companyName string) error {
	stateDir := filepath.Join(paths.RepoRoot, ".kageos")
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
		fmt.Println("WARN: rotate with `kagectl init-dev --regen-secrets` after clearing old dev infra volumes")
	}

	cfg := defaultDevDeploymentConfig(secrets)
	if companyCode != "" {
		cfg.Company.Code = companyCode
	}
	if companyName != "" {
		cfg.Company.Name = companyName
	}
	applyEnvOverrides(&cfg)
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		return err
	}

	configDir := filepath.Join(paths.RepoRoot, defaultDevConfig)
	for _, dir := range []string{configDir, envDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"global.yaml":       renderTemplate(globalConfigTemplate, rt),
		"api-gateway.yaml":  renderTemplate(apiGatewayConfigTemplate, rt),
		"app-runtime.yaml":  renderTemplate(appRuntimeConfigTemplate, rt),
		"app-server.yaml":   renderTemplate(appServerConfigTemplate, rt),
		"app-storage.yaml":  renderTemplate(appStorageConfigTemplate, rt),
		"agent-server.yaml": renderTemplate(agentServerConfigTemplate, rt),
		"hr-server.yaml":    renderTemplate(hrServerConfigTemplate, rt),
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
	if strings.TrimSpace(values["SMTP_HOST"]) != "" &&
		strings.TrimSpace(values["SMTP_USERNAME"]) != "" &&
		strings.TrimSpace(values["SMTP_PASSWORD"]) != "" &&
		strings.TrimSpace(values["SMTP_FROM"]) != "" {
		smtpStatus = "configured"
	}

	rows := [][2]string{
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
		{"MySQL databases", "app-server, agent-server, app-storage, hr-server"},
		{"NATS URL", values["NATS_URL"]},
		{"NATS user", values["NATS_SEED_USER"]},
		{"NATS password", values["NATS_SEED_PASSWORD"]},
		{"MinIO endpoint", values["MINIO_ENDPOINT"]},
		{"MinIO root user", values["MINIO_ROOT_USER"]},
		{"MinIO root password", values["MINIO_ROOT_PASSWORD"]},
		{"JWT secret", values["JWT_SECRET"]},
		{"Company code", values["KAGEOS_COMPANY_CODE"]},
		{"Company name", strings.Trim(values["KAGEOS_COMPANY_NAME"], `"'`)},
		{"SMTP status", smtpStatus},
		{"SMTP host", values["SMTP_HOST"]},
		{"SMTP username", values["SMTP_USERNAME"]},
	}
	fmt.Println()
	fmt.Println("Kageos dev initialization summary")
	printPlainTable("Item", "Value", rows)
	fmt.Println()
	fmt.Println("Tip: SMTP is optional for local startup. Configure SMTP_* in the env file only when email verification must send real messages.")
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
	return devSecrets{
		MySQLRootPassword:  strings.TrimSpace(values["MYSQL_ROOT_PASSWORD"]),
		NATSUser:           strings.TrimSpace(values["NATS_SEED_USER"]),
		NATSPassword:       strings.TrimSpace(values["NATS_SEED_PASSWORD"]),
		MinIORootUser:      strings.TrimSpace(values["MINIO_ROOT_USER"]),
		MinIORootPassword:  strings.TrimSpace(values["MINIO_ROOT_PASSWORD"]),
		JWTSecret:          strings.TrimSpace(values["JWT_SECRET"]),
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
		{"Admin username", "system"},
		{"Initial password", rt.SystemUser.Password},
		{"Main config", rt.Paths.ConfigPath},
		{"Compose file", rt.ComposeConfigPath},
		{"Generated config dir", filepath.Join(rt.Paths.GeneratedDir, "config")},
		{"Environment file", rt.EnvFilePath},
		{"TLS directory", rt.TLSCertsHostDir},
		{"Summary file", rt.SummaryPath},
		{"Status command", fmt.Sprintf("go run ./cmd/kagectl status --config %s", rt.Paths.ConfigPath)},
		{"Logs command", fmt.Sprintf("go run ./cmd/kagectl logs --config %s main", rt.Paths.ConfigPath)},
		{"Stop command", fmt.Sprintf("go run ./cmd/kagectl down --config %s", rt.Paths.ConfigPath)},
	}
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

func detectComposeCommand() ([]string, error) {
	if _, err := exec.LookPath("podman"); err == nil {
		if err := exec.Command("podman", "compose", "version").Run(); err == nil {
			return []string{"podman", "compose"}, nil
		}
	}
	if _, err := exec.LookPath("docker"); err == nil {
		if err := exec.Command("docker", "compose", "version").Run(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}
	return nil, fmt.Errorf("podman compose or docker compose is required")
}

func checkComposeCommand() error {
	_, err := detectComposeCommand()
	return err
}

func runCompose(workDir string, args ...string) error {
	compose, err := detectComposeCommand()
	if err != nil {
		return err
	}
	cmdArgs := append(compose[1:], args...)
	cmd := exec.Command(compose[0], cmdArgs...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runComposeCapture(workDir string, args ...string) error {
	_, err := runComposeOutput(workDir, args...)
	return err
}

func runComposeOutput(workDir string, args ...string) (string, error) {
	compose, err := detectComposeCommand()
	if err != nil {
		return "", err
	}
	cmdArgs := append(compose[1:], args...)
	cmd := exec.Command(compose[0], cmdArgs...)
	cmd.Dir = workDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, message)
	}
	return strings.TrimSpace(output.String()), nil
}

func runBuildAppBaseScript(paths Paths, opts buildAppBaseOptions) error {
	scriptPath := filepath.Join(paths.RepoRoot, "deploy", "base", "scripts", "build-app-base-image.sh")
	if !fileExists(scriptPath) {
		return fmt.Errorf("app-base build script not found: %s", scriptPath)
	}

	args := []string{scriptPath}
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	cmd := exec.Command("bash", args...)
	cmd.Dir = paths.RepoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	if opts.Image != "" {
		cmd.Env = append(cmd.Env, "KAGEOS_APP_BASE_IMAGE="+opts.Image)
	}
	return cmd.Run()
}

func runDevInfraScript(paths Paths, opts initDevOptions) error {
	scriptPath := filepath.Join(paths.RepoRoot, "deploy", "dev", "scripts", "infra.sh")
	if !fileExists(scriptPath) {
		return fmt.Errorf("dev infra script not found: %s", scriptPath)
	}

	args := []string{scriptPath}
	if opts.Engine != "" && opts.Engine != "auto" {
		args = append(args, opts.Engine)
	}
	args = append(args, "up", "-d")

	cmd := exec.Command("bash", args...)
	cmd.Dir = paths.RepoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	return cmd.Run()
}

func checkHTTP(rawURL string) error {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned HTTP %d", rawURL, resp.StatusCode)
	}
	return nil
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

func checkTCP(label, host string, port int) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return fmt.Errorf("%s %s:%d not reachable: %w", label, host, port, err)
	}
	_ = conn.Close()
	return nil
}

func checkBundledMySQLInitialized(rt RuntimeConfig) error {
	databases := requiredMySQLDatabases(rt)
	if len(databases) == 0 {
		return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "mysql", "mysqladmin", "ping", "-h", "127.0.0.1", "--connect-timeout=3", "-u"+rt.MySQL.User, "-p"+rt.MySQL.Password)
	}

	sql := fmt.Sprintf(
		"SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME IN (%s);",
		mysqlStringList(databases),
	)
	output, err := runComposeOutput(rt.Paths.GeneratedDir, "exec", "-T", "mysql", "mysql", "-h", "127.0.0.1", "--connect-timeout=3", "-u"+rt.MySQL.User, "-p"+rt.MySQL.Password, "-N", "-B", "-e", sql)
	if err != nil {
		return err
	}
	count, err := parseMySQLCountOutput(output)
	if err != nil {
		return err
	}
	if count != len(databases) {
		return fmt.Errorf("mysql initialized databases not ready: got %d/%d (%s)", count, len(databases), strings.Join(databases, ", "))
	}
	return nil
}

func requiredMySQLDatabases(rt RuntimeConfig) []string {
	return uniqueNonEmptyStrings([]string{
		rt.MySQL.AppDatabase,
		rt.MySQL.StorageDatabase,
		rt.MySQL.AgentDatabase,
		rt.MySQL.HRDatabase,
	})
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func mysqlStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, mysqlStringLiteral(value))
	}
	return strings.Join(quoted, ", ")
}

func mysqlStringLiteral(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "'", "''")
	return "'" + replacer.Replace(value) + "'"
}

func parseMySQLCountOutput(output string) (int, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		count, err := strconv.Atoi(line)
		if err == nil {
			return count, nil
		}
	}
	return 0, fmt.Errorf("parse mysql database count from %q: no numeric line found", output)
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
	host, port := natsHostPortForMain(cfg)
	return buildNATSURL(cfg, host, port)
}

func natsURLForSDK(cfg Config) string {
	if cfg.NATS.Mode == "external" && cfg.NATS.URL != "" {
		return cfg.NATS.URL
	}
	if cfg.NATS.Mode == "bundled" {
		return buildNATSURL(cfg, "127.0.0.1", 4222)
	}
	return buildNATSURL(cfg, cfg.NATS.Host, cfg.NATS.Port)
}

func buildNATSURL(cfg Config, host string, port int) string {
	userInfo := ""
	if cfg.NATS.AuthEnabled {
		userInfo = url.UserPassword(cfg.NATS.User, cfg.NATS.Password).String() + "@"
	}
	return fmt.Sprintf("nats://%s%s", userInfo, net.JoinHostPort(host, strconv.Itoa(port)))
}

func natsHostPortForMain(cfg Config) (string, int) {
	host := cfg.NATS.Host
	port := cfg.NATS.Port
	if cfg.NATS.Mode == "bundled" {
		host = "127.0.0.1"
		port = 4222
	}
	return host, port
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
