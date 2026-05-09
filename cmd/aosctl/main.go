package main

import (
	"bytes"
	"crypto/rand"
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
	Secrets    SecretsConfig    `yaml:"secrets"`
	SystemUser SystemUserConfig `yaml:"system_user"`
	LLMs       LLMSeedsConfig   `yaml:"llms"`
	SMTP       SMTPConfig       `yaml:"smtp"`
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
	Mode                   string `yaml:"mode"`
	Host                   string `yaml:"host"`
	Port                   int    `yaml:"port"`
	User                   string `yaml:"user"`
	Password               string `yaml:"password"`
	AppDatabase            string `yaml:"app_database"`
	ScheduledTaskDatabase  string `yaml:"scheduled_task_database"`
	TimerSchedulerDatabase string `yaml:"timer_scheduler_database"`
	AgentDatabase          string `yaml:"agent_database"`
	StorageDatabase        string `yaml:"storage_database"`
	HRDatabase             string `yaml:"hr_database"`
	BackupAdminUser        string `yaml:"backup_admin_user"`
	BackupAdminPass        string `yaml:"backup_admin_password"`
	CreateBundledSQL       bool   `yaml:"create_bundled_sql"`
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
	Provider    string `yaml:"provider"`
	Model       string `yaml:"model"`
	APIKey      string `yaml:"api_key"`
	APIKeyEnv   string `yaml:"api_key_env"`
	APIBase     string `yaml:"api_base"`
	Timeout     int    `yaml:"timeout"`
	MaxTokens   int    `yaml:"max_tokens"`
	ExtraConfig string `yaml:"extra_config"`
	UseThinking bool   `yaml:"use_thinking"`
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
	SDKGatewayURL      string
	MinIOEndpoint      string
	SDKMinIOEndpoint   string
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
	LLMSeedEnvVars     []string
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
	case "bootstrap":
		return cmdBootstrap(paths, rest)
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
	Force     bool
	BaseURL   string
	MySQLMode string
}

type bootstrapOptions struct {
	Init   initOptions
	UpArgs []string
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

func printUsage() {
	fmt.Println(`aosctl manages AI-Agent-OS production deployment files.

Usage:
  aosctl init [--force] [--base-url URL] [--mysql-mode bundled|external]
  aosctl bootstrap --base-url URL [--mysql-mode bundled|external] [--image|--no-build] [--skip-verify] [--wait-timeout 5m]
  aosctl render [--config deploy/prod/aos.yaml]
  aosctl layers [--config deploy/prod/aos.yaml] [--json]
  aosctl doctor [--config deploy/prod/aos.yaml] [--json]
  aosctl up [--config deploy/prod/aos.yaml] [--image|--no-build] [--skip-verify] [--wait-timeout 5m]
  aosctl verify [--config deploy/prod/aos.yaml] [--json]
  aosctl status [--config deploy/prod/aos.yaml] [--json]
  aosctl logs [--config deploy/prod/aos.yaml] [service|layer] [--layer L0-L5]
  aosctl down [--config deploy/prod/aos.yaml]
  aosctl uninstall [--config deploy/prod/aos.yaml] [--purge-data] [--purge-podman-storage] [--purge-images] [--keep-generated] [--purge-private-config] [--force] [--dry-run]

Compose remains the container execution engine; aosctl owns layered config rendering, deployment orchestration, and diagnostics.`)
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
	opts, err := parseInitFlags("init", args)
	if err != nil {
		return err
	}
	created, err := writeInitialConfig(paths, opts)
	if err != nil {
		return err
	}
	if created {
		fmt.Println("next: edit site.base_url if needed and run `aosctl up`")
	}
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
	if err := runCompose(rt.Paths.GeneratedDir, "run", "--rm", "--no-deps", "-e", "APP_BASE_ACTION=ensure", "-e", "APP_BASE_BUILD_NO_CACHE=0", "--entrypoint", "/app/entrypoint-app-base.sh", "main"); err != nil {
		return err
	}
	fmt.Println("[L1-L4] 启动 Compose 服务栈")
	if err := runCompose(rt.Paths.GeneratedDir, "up", "-d", "--no-build"); err != nil {
		return err
	}
	if opts.SkipVerify {
		fmt.Println("deployment started; layered verify skipped (--skip-verify)")
		return nil
	}
	fmt.Printf("[L0-L5] 等待分层健康检查通过（timeout %s）\n", opts.VerifyTimeout)
	if err := waitLayerChecks("verify", verifyLayerChecks(rt), opts.VerifyTimeout, defaultUpVerifyInterval); err != nil {
		return err
	}
	fmt.Println("deployment ready")
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
		return "aosctl runs on the host and logs to the current terminal"
	case layerApps:
		return "user App containers are managed by app-runtime; use platform app log APIs"
	default:
		return "check `aosctl status` for layer to service ownership"
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

func verifyLayerChecks(rt RuntimeConfig) []layerCheck {
	checks := []layerCheck{
		{Layer: layerControl, Name: "config validation", Target: rt.Paths.ConfigPath, Fn: func() error { return validateConfig(rt) }},
		{Layer: layerControl, Name: "rendered compose", Target: rt.ComposeConfigPath, Fn: func() error { return requireGeneratedCompose(rt.Paths) }},
		{Layer: layerInfra, Name: "mysql tcp", Target: rt.MySQLAddress, Fn: func() error { return checkTCP("mysql", rt.MySQLHostForMain, rt.MySQLPortForMain) }},
		{Layer: layerInfra, Name: "nats tcp", Target: tcpTarget(rt.NATSHostForMain, rt.NATSPortForMain), Fn: func() error { return checkTCP("nats", rt.NATSHostForMain, rt.NATSPortForMain) }},
		{Layer: layerInfra, Name: "minio tcp", Target: tcpTarget(rt.MinIOHostForMain, rt.MinIOPortForMain), Fn: func() error { return checkTCP("minio", rt.MinIOHostForMain, rt.MinIOPortForMain) }},
		{Layer: layerEdge, Name: "nginx http listener", Target: "127.0.0.1:80", Fn: func() error { return checkTCP("nginx", "127.0.0.1", 80) }},
		{Layer: layerEdge, Name: "main edge probe", Target: "compose exec main /app/health/edge.sh", Fn: func() error {
			return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "/app/health/edge.sh")
		}},
	}
	if rt.Site.TLSMode == "https" || rt.Site.TLSMode == "redirect" {
		checks = append(checks, layerCheck{Layer: layerEdge, Name: "nginx https listener", Target: "127.0.0.1:443", Fn: func() error { return checkTCP("nginx", "127.0.0.1", 443) }})
	}
	checks = append(checks,
		layerCheck{Layer: layerPlatform, Name: "api-gateway", Target: "http://127.0.0.1:9090/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9090/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-server", Target: "http://127.0.0.1:9091/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9091/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-storage", Target: "http://127.0.0.1:9092/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9092/health") }},
		layerCheck{Layer: layerPlatform, Name: "agent-server", Target: "http://127.0.0.1:9095/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9095/health") }},
		layerCheck{Layer: layerPlatform, Name: "control-service", Target: "http://127.0.0.1:9096/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9096/health") }},
		layerCheck{Layer: layerPlatform, Name: "hr-server", Target: "http://127.0.0.1:9097/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9097/health") }},
		layerCheck{Layer: layerPlatform, Name: "message-server", Target: "http://127.0.0.1:9109/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9109/health") }},
		layerCheck{Layer: layerPlatform, Name: "timer-scheduler", Target: "http://127.0.0.1:9108/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9108/health") }},
		layerCheck{Layer: layerPlatform, Name: "backup-service", Target: "http://127.0.0.1:19088/health", Fn: func() error { return checkHTTP("http://127.0.0.1:19088/health") }},
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
			Fn:     func() error { return requireContains(rt.SDKMinIOEndpoint, "127.0.0.1") },
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
		filepath.Join(rt.Storage.Root, "data", "license"),
		filepath.Join(rt.Storage.Root, "data", "backup", "repo"),
		filepath.Join(rt.Storage.Root, "data", "backup", "state"),
		filepath.Join(rt.Storage.Root, "data", "backup", "staging"),
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
		return fmt.Errorf("generated compose not found: %s; run `aosctl render` or `aosctl up` first", composePath)
	}
	return nil
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
	systemUserPass, err := randomHex(24)
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
			Mode:                   "bundled",
			Host:                   "127.0.0.1",
			Port:                   3306,
			User:                   "root",
			Password:               mysqlPass,
			AppDatabase:            "app_db",
			ScheduledTaskDatabase:  "app-scheduled-task",
			TimerSchedulerDatabase: "timer-scheduler",
			AgentDatabase:          "agent-server",
			StorageDatabase:        "app-storage",
			HRDatabase:             "hr-server",
			BackupAdminUser:        "root",
			BackupAdminPass:        mysqlPass,
			CreateBundledSQL:       true,
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
		SystemUser: SystemUserConfig{
			Password: systemUserPass,
		},
		SMTP: SMTPConfig{
			Host:     "smtp.qq.com",
			Port:     587,
			FromName: "AI Agent OS",
		},
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
	if cfg.MySQL.ScheduledTaskDatabase == "" {
		cfg.MySQL.ScheduledTaskDatabase = "app-scheduled-task"
	}
	if cfg.MySQL.TimerSchedulerDatabase == "" {
		cfg.MySQL.TimerSchedulerDatabase = "timer-scheduler"
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
	rt.SDKNATSURL = natsURLForSDK(cfg)
	rt.SDKGatewayURL = "http://127.0.0.1:9090"

	rt.MinIOEndpoint = cfg.MinIO.Endpoint
	rt.SDKMinIOEndpoint = cfg.MinIO.Endpoint
	rt.BackupMinIOAddress = cfg.MinIO.Endpoint
	if cfg.MinIO.Mode == "bundled" {
		rt.MinIOEndpoint = "127.0.0.1:9000"
		rt.SDKMinIOEndpoint = "127.0.0.1:9000"
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
	rt.LLMSeedEnvVars = uniqueLLMSeedEnvVars(cfg.LLMs.Configs)
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
		if strings.TrimSpace(cfg.Provider) == "" {
			errs = append(errs, fmt.Errorf("%s.provider is required", prefix))
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
		"config/message-server.yaml":  renderTemplate(messageServerConfigTemplate, rt),
		"config/control-service.yaml": renderTemplate(controlServiceConfigTemplate, rt),
		"config/timer-scheduler.yaml": renderTemplate(timerSchedulerConfigTemplate, rt),
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
