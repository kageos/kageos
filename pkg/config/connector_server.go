package config

import (
	"fmt"
	"sync"
)

var (
	connectorServerConfig *ConnectorServerConfig
	connectorServerOnce   sync.Once
	connectorServerMu     sync.RWMutex
)

func GetConnectorServerConfig() *ConnectorServerConfig {
	connectorServerOnce.Do(func() {
		cfg := &ConnectorServerConfig{}
		if err := loadYAMLConfig("connector-server.yaml", cfg); err != nil {
			fmt.Printf("Failed to load connector-server config: %v\n", err)
			cfg = &ConnectorServerConfig{}
		}
		connectorServerMu.Lock()
		connectorServerConfig = cfg
		connectorServerMu.Unlock()
	})

	connectorServerMu.RLock()
	defer connectorServerMu.RUnlock()
	return connectorServerConfig
}

type ConnectorServerConfig struct {
	Server ConnectorServerServerConfig `mapstructure:"server"`
	DB     DBConfig                    `mapstructure:"db"`
	OAuth  ConnectorOAuthConfig        `mapstructure:"oauth"`
}

type ConnectorServerServerConfig struct {
	Port        int    `mapstructure:"port"`
	ListenHost  string `mapstructure:"listen_host"`
	LogLevel    string `mapstructure:"log_level"`
	Debug       bool   `mapstructure:"debug"`
	EnablePprof *bool  `mapstructure:"enable_pprof"`
}

type ConnectorOAuthConfig struct {
	CallbackBaseURL       string                         `mapstructure:"callback_base_url"`
	TokenEncryptionSecret string                         `mapstructure:"token_encryption_secret"`
	StateTTLSeconds       int                            `mapstructure:"state_ttl_seconds"`
	ProviderAdmins        []string                       `mapstructure:"provider_admins"`
	Providers             []ConnectorOAuthProviderConfig `mapstructure:"providers"`
}

type ConnectorOAuthProviderConfig struct {
	Code               string            `mapstructure:"code"`
	Name               string            `mapstructure:"name"`
	AuthType           string            `mapstructure:"auth_type"`
	ClientID           string            `mapstructure:"client_id"`
	ClientSecret       string            `mapstructure:"client_secret"`
	ClientIDEnv        string            `mapstructure:"client_id_env"`
	ClientSecretEnv    string            `mapstructure:"client_secret_env"`
	AuthURL            string            `mapstructure:"auth_url"`
	TokenURL           string            `mapstructure:"token_url"`
	UserInfoURL        string            `mapstructure:"user_info_url"`
	Scopes             []string          `mapstructure:"scopes"`
	UsePKCE            *bool             `mapstructure:"use_pkce"`
	TokenRequestMode   string            `mapstructure:"token_request_mode"`
	ClientIDParam      string            `mapstructure:"client_id_param"`
	ClientSecretParam  string            `mapstructure:"client_secret_param"`
	GrantTypeParam     string            `mapstructure:"grant_type_param"`
	CodeParam          string            `mapstructure:"code_param"`
	RefreshTokenParam  string            `mapstructure:"refresh_token_param"`
	RedirectURIParam   string            `mapstructure:"redirect_uri_param"`
	ExtraAuthParams    map[string]string `mapstructure:"extra_auth_params"`
	ExtraTokenParams   map[string]string `mapstructure:"extra_token_params"`
	ExternalIDField    string            `mapstructure:"external_id_field"`
	DisplayNameField   string            `mapstructure:"display_name_field"`
	ProviderAccountURL string            `mapstructure:"provider_account_url"`
	LogoURL            string            `mapstructure:"logo_url"`
	BrandColor         string            `mapstructure:"brand_color"`
}

func (c *ConnectorServerConfig) GetPort() int {
	if c == nil || c.Server.Port == 0 {
		return 9096
	}
	return c.Server.Port
}

func (c *ConnectorServerConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Server.ListenHost)
}

func (c *ConnectorServerConfig) GetLogLevel() string {
	if c == nil || c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

func (c *ConnectorServerConfig) IsDebug() bool {
	return c != nil && c.Server.Debug
}

func (c *ConnectorServerConfig) IsPprofEnabled() bool {
	if c == nil {
		return true
	}
	return boolConfigValue(c.Server.EnablePprof, true)
}

func (c *ConnectorServerConfig) GetDB() DBConfig {
	if c == nil {
		return DBConfig{}
	}
	return c.DB
}

func (c *ConnectorServerConfig) GetOAuth() ConnectorOAuthConfig {
	if c == nil {
		return ConnectorOAuthConfig{}
	}
	return c.OAuth
}
