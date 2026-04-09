package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type BackupServiceConfig struct {
	Server struct {
		Port     int    `mapstructure:"port"`
		LogLevel string `mapstructure:"log_level"`
		Debug    bool   `mapstructure:"debug"`
	} `mapstructure:"server"`

	Storage struct {
		Root              string `mapstructure:"root"`
		NamespacePath     string `mapstructure:"namespace_path"`
		DataPath          string `mapstructure:"data_path"`
		LogsPath          string `mapstructure:"logs_path"`
		MySQLPath         string `mapstructure:"mysql_path"`
		MinIOPath         string `mapstructure:"minio_path"`
		PodmanStoragePath string `mapstructure:"podman_storage_path"`
	} `mapstructure:"storage"`

	Repository struct {
		RootPath    string `mapstructure:"root_path"`
		StatePath   string `mapstructure:"state_path"`
		StagingPath string `mapstructure:"staging_path"`
	} `mapstructure:"repository"`

	Database struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"database"`

	Maintenance struct {
		MarkerPath   string `mapstructure:"marker_path"`
		PagePath     string `mapstructure:"page_path"`
		MetadataPath string `mapstructure:"metadata_path"`
	} `mapstructure:"maintenance"`

	Dependencies struct {
		MySQLAddress string `mapstructure:"mysql_address"`
		MinIOAddress string `mapstructure:"minio_address"`
	} `mapstructure:"dependencies"`

	Tooling struct {
		MySQLBinary       string `mapstructure:"mysql_binary"`
		MySQLDumpBinary   string `mapstructure:"mysqldump_binary"`
		ResticBinary      string `mapstructure:"restic_binary"`
		MinIOClientBinary string `mapstructure:"minio_client_binary"`
	} `mapstructure:"tooling"`

	MySQL struct {
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"mysql"`

	MinIO struct {
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
	} `mapstructure:"minio"`

	Auth struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Realm    string `mapstructure:"realm"`
	} `mapstructure:"auth"`
}

var (
	backupServiceConfig     *BackupServiceConfig
	backupServiceConfigOnce sync.Once
	backupServiceConfigMu   sync.RWMutex
)

func GetBackupServiceConfig() *BackupServiceConfig {
	backupServiceConfigOnce.Do(func() {
		bootstrapBackupServiceLocalEnv()

		cfg := &BackupServiceConfig{}
		if err := loadYAMLConfig("backup-service.yaml", cfg); err != nil {
			fmt.Printf("Failed to load backup-service config: %v\n", err)
			cfg = &BackupServiceConfig{}
		}
		cfg.normalizeForLocalHost()
		backupServiceConfigMu.Lock()
		backupServiceConfig = cfg
		backupServiceConfigMu.Unlock()
	})

	backupServiceConfigMu.RLock()
	defer backupServiceConfigMu.RUnlock()
	return backupServiceConfig
}

func (c *BackupServiceConfig) GetPort() int {
	if c.Server.Port == 0 {
		return 19088
	}
	return c.Server.Port
}

func (c *BackupServiceConfig) GetLogLevel() string {
	if c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

func (c *BackupServiceConfig) IsDebug() bool {
	return c.Server.Debug
}

func (c *BackupServiceConfig) GetDatabasePath() string {
	if c.Database.Path != "" {
		return c.Database.Path
	}
	return filepath.Join(c.Repository.StatePath, "backup-service.db")
}

func (c *BackupServiceConfig) GetMaintenanceMarkerPath() string {
	if c.Maintenance.MarkerPath != "" {
		return c.Maintenance.MarkerPath
	}
	return filepath.Join(c.Repository.StatePath, "maintenance.flag")
}

func (c *BackupServiceConfig) GetMaintenancePagePath() string {
	if c.Maintenance.PagePath != "" {
		return c.Maintenance.PagePath
	}
	return filepath.Join(c.Repository.StatePath, "maintenance.html")
}

func (c *BackupServiceConfig) GetMaintenanceMetadataPath() string {
	if c.Maintenance.MetadataPath != "" {
		return c.Maintenance.MetadataPath
	}
	return filepath.Join(c.Repository.StatePath, "maintenance.json")
}

func (c *BackupServiceConfig) GetMySQLAddress() string {
	if c.Dependencies.MySQLAddress != "" {
		return c.Dependencies.MySQLAddress
	}
	return "mysql:3306"
}

func (c *BackupServiceConfig) GetMySQLUser() string {
	if value := sanitizeConfigValue(c.MySQL.User); value != "" {
		return value
	}
	if value := sanitizeConfigValue(GetAppStorageConfig().DB.User); value != "" {
		return value
	}
	if value := sanitizeConfigValue(GetAppServerConfig().DB.User); value != "" {
		return value
	}
	return "root"
}

func (c *BackupServiceConfig) GetMySQLPassword() string {
	if value := sanitizeConfigValue(c.MySQL.Password); value != "" {
		return value
	}
	if value := sanitizeConfigValue(GetAppStorageConfig().DB.Password); value != "" {
		return value
	}
	if value := sanitizeConfigValue(GetAppServerConfig().DB.Password); value != "" {
		return value
	}
	return sanitizeConfigValue(os.Getenv("MYSQL_ROOT_PASSWORD"))
}

func (c *BackupServiceConfig) GetMySQLBinary() string {
	if c.Tooling.MySQLBinary != "" {
		return c.Tooling.MySQLBinary
	}
	return "mysql"
}

func (c *BackupServiceConfig) GetMinIOAddress() string {
	if c.Dependencies.MinIOAddress != "" {
		return c.Dependencies.MinIOAddress
	}
	return "minio:9000"
}

func (c *BackupServiceConfig) GetMinIOAccessKey() string {
	if value := sanitizeConfigValue(c.MinIO.AccessKey); value != "" {
		return value
	}
	if value := sanitizeConfigValue(GetAppStorageConfig().Storage.MinIO.AccessKey); value != "" {
		return value
	}
	return sanitizeConfigValue(os.Getenv("MINIO_ROOT_USER"))
}

func (c *BackupServiceConfig) GetMinIOSecretKey() string {
	if value := sanitizeConfigValue(c.MinIO.SecretKey); value != "" {
		return value
	}
	if value := sanitizeConfigValue(GetAppStorageConfig().Storage.MinIO.SecretKey); value != "" {
		return value
	}
	return sanitizeConfigValue(os.Getenv("MINIO_ROOT_PASSWORD"))
}

func (c *BackupServiceConfig) GetBasicAuthUsername() string {
	if value := sanitizeConfigValue(c.Auth.Username); value != "" {
		return value
	}
	return sanitizeConfigValue(os.Getenv("BACKUP_BASIC_AUTH_USERNAME"))
}

func (c *BackupServiceConfig) GetBasicAuthPassword() string {
	if value := sanitizeConfigValue(c.Auth.Password); value != "" {
		return value
	}
	return sanitizeConfigValue(os.Getenv("BACKUP_BASIC_AUTH_PASSWORD"))
}

func (c *BackupServiceConfig) GetBasicAuthRealm() string {
	if value := sanitizeConfigValue(c.Auth.Realm); value != "" {
		return value
	}
	if value := sanitizeConfigValue(os.Getenv("BACKUP_BASIC_AUTH_REALM")); value != "" {
		return value
	}
	return "Backup Control Plane"
}

func (c *BackupServiceConfig) IsBasicAuthEnabled() bool {
	return c.GetBasicAuthUsername() != "" && c.GetBasicAuthPassword() != ""
}

func sanitizeConfigValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		return ""
	}
	return value
}

func (c *BackupServiceConfig) GetMySQLDumpBinary() string {
	if c.Tooling.MySQLDumpBinary != "" {
		return c.Tooling.MySQLDumpBinary
	}
	return "mysqldump"
}

func (c *BackupServiceConfig) GetResticBinary() string {
	if c.Tooling.ResticBinary != "" {
		return c.Tooling.ResticBinary
	}
	return "restic"
}

func (c *BackupServiceConfig) GetMinIOClientBinary() string {
	if c.Tooling.MinIOClientBinary != "" {
		return c.Tooling.MinIOClientBinary
	}
	return "mc"
}

func (c *BackupServiceConfig) normalizeForLocalHost() {
	root := GetAgentOSRoot()
	if root == "" {
		return
	}

	if getConfigEnv() == "dev" {
		c.normalizeForDevRoot(root)
		return
	}

	if filepath.Clean(root) == "/app" {
		return
	}

	storageRoot := resolveBackupServiceStorageRoot(root)
	if storageRoot == "" {
		return
	}

	c.normalizeForHostStorageRoot(storageRoot)
}

func (c *BackupServiceConfig) normalizeForDevRoot(root string) {
	root = filepath.Clean(root)

	abs := func(value string) string {
		value = strings.TrimSpace(value)
		if value == "" {
			return ""
		}
		if filepath.IsAbs(value) {
			return filepath.Clean(value)
		}
		return filepath.Join(root, value)
	}

	if strings.TrimSpace(c.Storage.Root) == "" || c.Storage.Root == "." {
		c.Storage.Root = root
	} else {
		c.Storage.Root = abs(c.Storage.Root)
	}

	c.Storage.NamespacePath = abs(c.Storage.NamespacePath)
	c.Storage.DataPath = abs(c.Storage.DataPath)
	c.Storage.LogsPath = abs(c.Storage.LogsPath)
	c.Storage.MySQLPath = abs(c.Storage.MySQLPath)
	c.Storage.MinIOPath = abs(c.Storage.MinIOPath)
	c.Storage.PodmanStoragePath = abs(c.Storage.PodmanStoragePath)

	c.Repository.RootPath = abs(c.Repository.RootPath)
	c.Repository.StatePath = abs(c.Repository.StatePath)
	c.Repository.StagingPath = abs(c.Repository.StagingPath)
	c.Database.Path = abs(c.Database.Path)

	c.Maintenance.MarkerPath = abs(c.Maintenance.MarkerPath)
	c.Maintenance.PagePath = abs(c.Maintenance.PagePath)
	c.Maintenance.MetadataPath = abs(c.Maintenance.MetadataPath)

	if strings.TrimSpace(c.Dependencies.MySQLAddress) == "" {
		c.Dependencies.MySQLAddress = "127.0.0.1:3306"
	}
	if strings.TrimSpace(c.Dependencies.MinIOAddress) == "" {
		c.Dependencies.MinIOAddress = "127.0.0.1:9000"
	}
}

func (c *BackupServiceConfig) normalizeForHostStorageRoot(storageRoot string) {
	storageRoot = filepath.Clean(storageRoot)
	c.Storage.Root = storageRoot
	c.Storage.NamespacePath = filepath.Join(storageRoot, "namespace")
	c.Storage.DataPath = filepath.Join(storageRoot, "data")
	c.Storage.LogsPath = filepath.Join(storageRoot, "logs")
	c.Storage.MySQLPath = filepath.Join(storageRoot, "mysql")
	c.Storage.MinIOPath = filepath.Join(storageRoot, "minio")
	c.Storage.PodmanStoragePath = filepath.Join(storageRoot, "podman_storage")

	c.Repository.RootPath = filepath.Join(storageRoot, "data", "backup", "repo")
	c.Repository.StatePath = filepath.Join(storageRoot, "data", "backup", "state")
	c.Repository.StagingPath = filepath.Join(storageRoot, "data", "backup", "staging")
	c.Database.Path = filepath.Join(storageRoot, "data", "backup", "state", "backup-service.db")

	c.Maintenance.MarkerPath = filepath.Join(storageRoot, "data", "backup", "state", "maintenance.flag")
	c.Maintenance.PagePath = filepath.Join(storageRoot, "data", "backup", "state", "maintenance.html")
	c.Maintenance.MetadataPath = filepath.Join(storageRoot, "data", "backup", "state", "maintenance.json")

	if strings.TrimSpace(c.Dependencies.MySQLAddress) == "" || c.Dependencies.MySQLAddress == "mysql:3306" {
		c.Dependencies.MySQLAddress = "127.0.0.1:3306"
	}
	if strings.TrimSpace(c.Dependencies.MinIOAddress) == "" || c.Dependencies.MinIOAddress == "minio:9000" {
		c.Dependencies.MinIOAddress = "127.0.0.1:9000"
	}
}

func bootstrapBackupServiceLocalEnv() {
	if getConfigEnv() != "prod" {
		return
	}

	root := GetAgentOSRoot()
	if root == "" || filepath.Clean(root) == "/app" {
		return
	}

	envMap := readLocalProdEnvFile(root)
	for _, key := range []string{
		"STORAGE_ROOT",
		"MYSQL_ROOT_PASSWORD",
		"MINIO_ROOT_USER",
		"MINIO_ROOT_PASSWORD",
		"BACKUP_BASIC_AUTH_USERNAME",
		"BACKUP_BASIC_AUTH_PASSWORD",
		"BACKUP_BASIC_AUTH_REALM",
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}
		if value := strings.TrimSpace(envMap[key]); value != "" {
			_ = os.Setenv(key, value)
		}
	}
}

func resolveBackupServiceStorageRoot(root string) string {
	if value := strings.TrimSpace(os.Getenv("STORAGE_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
	}

	envMap := readLocalProdEnvFile(root)
	if value := strings.TrimSpace(envMap["STORAGE_ROOT"]); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
	}

	return filepath.Join(root, ".local", "ai-agent-os")
}

func readLocalProdEnvFile(root string) map[string]string {
	values := map[string]string{}

	data, err := os.ReadFile(filepath.Join(root, "deploy", "prod", ".env"))
	if err != nil {
		return values
	}

	lines := strings.Split(string(data), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		values[key] = value
	}

	return values
}
