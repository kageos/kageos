package main

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var defaultCompanyCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const (
	defaultProdDir    = ".kageos/prod"
	defaultConfigName = "kage.yaml"
	defaultGenerated  = "generated"
	defaultDevConfig  = ".kageos/dev/config"
	defaultTimezone   = "Asia/Shanghai"
	composeEngineEnv  = "KAGEOS_COMPOSE_ENGINE"

	networkProfileAIOBridge  = "aio-bridge"
	networkProfileDevHost    = "dev-host"
	networkProfileLegacyHost = "legacy-host"

	defaultMainImage           = "localhost/kageos-main:latest"
	defaultAppBaseBuilderImage = "localhost/kageos-app-base-builder:latest"
	defaultAppBaseImage        = "kagebase:latest"
	defaultMySQLImage          = "docker.io/library/mysql:latest"
	defaultNATSImage           = "docker.io/library/nats:latest"
	defaultMinIOImage          = "docker.io/minio/minio:latest"

	defaultUpVerifyTimeout  = 5 * time.Minute
	defaultUpVerifyInterval = 5 * time.Second
)

const defaultStorageRootFallback = "/data/kageos"

var (
	execLookPath = exec.LookPath
	execRun      = func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	}
	execOutput = func(name string, args ...string) (string, error) {
		output, err := exec.Command(name, args...).CombinedOutput()
		return string(output), err
	}
)

func defaultStorageRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return defaultStorageRootFallback
	}
	return defaultStorageRootForHome(home)
}

func defaultStorageRootForHome(home string) string {
	return filepath.Join(home, ".kageos", "storage", "prod")
}

func defaultSiteHTTPPort(site SiteConfig) int {
	if p := explicitURLPort(site.BaseURL, "http"); p > 0 {
		return p
	}
	return 80
}

func defaultSiteHTTPSPort(site SiteConfig) int {
	if p := explicitURLPort(site.BaseURL, "https"); p > 0 {
		return p
	}
	return 443
}

func explicitURLPort(rawURL string, scheme string) int {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme != scheme {
		return 0
	}
	port := strings.TrimSpace(u.Port())
	if port == "" {
		return 0
	}
	n, err := strconv.Atoi(port)
	if err != nil || n <= 0 || n > 65535 {
		return 0
	}
	return n
}

type Config struct {
	Timezone   string           `yaml:"timezone"`
	Network    NetworkConfig    `yaml:"network"`
	Site       SiteConfig       `yaml:"site"`
	Images     ImageConfig      `yaml:"images"`
	Storage    StorageConfig    `yaml:"storage"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	NATS       NATSConfig       `yaml:"nats"`
	MinIO      MinIOConfig      `yaml:"minio"`
	Company    CompanyConfig    `yaml:"company"`
	Auth       AuthConfig       `yaml:"auth"`
	Secrets    SecretsConfig    `yaml:"secrets"`
	SystemUser SystemUserConfig `yaml:"system_user"`
	LLMs       LLMSeedsConfig   `yaml:"llms"`
	SMTP       SMTPConfig       `yaml:"smtp"`
}

type NetworkConfig struct {
	// profile selects the address matrix rendered into compose/runtime configs.
	// aio-bridge keeps main on the compose bridge network; legacy-host preserves
	// the older host-network deployment for emergency rollback.
	Profile string `yaml:"profile"`
}

type SiteConfig struct {
	BaseURL                  string `yaml:"base_url"`
	TLSMode                  string `yaml:"tls_mode"`
	AllowSelfSignedBootstrap bool   `yaml:"allow_self_signed_bootstrap,omitempty"`
	HTTPPort                 int    `yaml:"http_port,omitempty"`
	HTTPSPort                int    `yaml:"https_port,omitempty"`
	CertFile                 string `yaml:"cert_file"`
	KeyFile                  string `yaml:"key_file"`
	TLSCertPEMB64            string `yaml:"tls_cert_pem_b64,omitempty"`
	TLSKeyPEMB64             string `yaml:"tls_key_pem_b64,omitempty"`
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
	AgentDatabase     string `yaml:"agent_database"`
	ConnectorDatabase string `yaml:"connector_database"`
	StorageDatabase   string `yaml:"storage_database"`
	HRDatabase        string `yaml:"hr_database"`
	TimerDatabase     string `yaml:"timer_database"`
	MessageDatabase   string `yaml:"message_database"`
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

type CompanyConfig struct {
	Code    string `yaml:"code"`
	Name    string `yaml:"name"`
	LogoURL string `yaml:"logo_url"`
}

type AuthConfig struct {
	RegistrationMode string `yaml:"registration_mode"`
}

type SecretsConfig struct {
	JWTSecret              string `yaml:"jwt_secret"`
	AppDBSecret            string `yaml:"app_db_secret"`
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
	Code         string `yaml:"code"`
	Name         string `yaml:"name"`
	Provider     string `yaml:"provider"`
	Protocol     string `yaml:"protocol"`
	Model        string `yaml:"model"`
	APIKey       string `yaml:"api_key"`
	APIKeyEnv    string `yaml:"api_key_env"`
	APIBase      string `yaml:"api_base"`
	EndpointPath string `yaml:"endpoint_path"`
	APIVersion   string `yaml:"api_version"`
	AuthScheme   string `yaml:"auth_scheme"`
	Headers      string `yaml:"headers"`
	Timeout      int    `yaml:"timeout"`
	MaxTokens    int    `yaml:"max_tokens"`
	ExtraConfig  string `yaml:"extra_config"`
	Capabilities string `yaml:"capabilities"`
	IsDefault    bool   `yaml:"is_default"`
	Visibility   int    `yaml:"visibility"`
	Admin        string `yaml:"admin"`
}

type SMTPConfig struct {
	Mode     string `yaml:"mode"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	FromName string `yaml:"from_name"`
}

type Paths struct {
	RepoRoot     string
	StateDir     string
	StateEnvPath string
	ProdDir      string
	ConfigPath   string
	GeneratedDir string
}

type RuntimeConfig struct {
	Config
	Paths                   Paths
	MySQLHostForMain        string
	MySQLPortForMain        int
	NATSHostForMain         string
	NATSPortForMain         int
	MinIOHostForMain        string
	MinIOPortForMain        int
	MySQLAddress            string
	AppDBClusterKey         string
	NATSURL                 string
	SDKNATSURL              string
	SDKGatewayURL           string
	MinIOEndpoint           string
	SDKMinIOEndpoint        string
	TLSCertsHostDir         string
	IncludeMySQL            bool
	IncludeNATS             bool
	IncludeMinIO            bool
	NATSAuthUser            string
	NATSAuthPassword        string
	ComposeConfigPath       string
	AppBaseBuilderImage     string
	AppContainerNetworkMode string
	AppRuntimeBasePath      string
	UseHostNetwork          bool
	LLMSeedEnvVars          []string
	EnvFilePath             string
	SummaryPath             string
}

type devSecrets struct {
	MySQLRootPassword  string
	NATSUser           string
	NATSPassword       string
	MinIORootUser      string
	MinIORootPassword  string
	JWTSecret          string
	AppDBSecret        string
	SystemUserPassword string
}
