package model

import (
	"time"

	"github.com/kageos/kageos/pkg/gormx/models"
)

const (
	ConnectorStatusActive  = "active"
	ConnectorStatusRevoked = "revoked"
)

// ConnectorConnection is a user-owned connector account reference.
type ConnectorConnection struct {
	models.Base
	ConnectionID      string `json:"connection_id" gorm:"column:connection_id;type:varchar(64);not null;uniqueIndex"`
	OwnerUsername     string `json:"owner_username" gorm:"column:owner_username;type:varchar(100);not null;index:idx_connector_owner_provider"`
	Provider          string `json:"provider" gorm:"column:provider;type:varchar(80);not null;index:idx_connector_owner_provider"`
	DisplayName       string `json:"display_name" gorm:"column:display_name;type:varchar(200);not null"`
	ExternalAccountID string `json:"external_account_id" gorm:"column:external_account_id;type:varchar(200)"`
	Status            string `json:"status" gorm:"column:status;type:varchar(30);not null;default:active;index"`
	Metadata          string `json:"metadata" gorm:"column:metadata;type:text"`
}

func (ConnectorConnection) TableName() string {
	return "connector_connections"
}

// ConnectorDirectoryBinding binds one user's connector to one resource path.
//
// The owner_username column is part of the lookup key. This is what prevents
// user A from resolving or using user B's directory binding.
type ConnectorDirectoryBinding struct {
	models.Base
	OwnerUsername string `json:"owner_username" gorm:"column:owner_username;type:varchar(100);not null;uniqueIndex:idx_connector_binding_scope;index:idx_connector_binding_owner"`
	TenantUser    string `json:"tenant_user" gorm:"column:tenant_user;type:varchar(100);not null;index:idx_connector_binding_workspace"`
	App           string `json:"app" gorm:"column:app;type:varchar(100);not null;index:idx_connector_binding_workspace"`
	ResourcePath  string `json:"resource_path" gorm:"column:resource_path;type:varchar(500);not null;uniqueIndex:idx_connector_binding_scope;index:idx_connector_binding_workspace"`
	Provider      string `json:"provider" gorm:"column:provider;type:varchar(80);not null;uniqueIndex:idx_connector_binding_scope"`
	ConnectionID  string `json:"connection_id" gorm:"column:connection_id;type:varchar(64);not null;index"`
}

func (ConnectorDirectoryBinding) TableName() string {
	return "connector_directory_bindings"
}

const (
	OAuthStateStatusPending = "pending"
	OAuthStateStatusUsed    = "used"
)

// ConnectorOAuthState stores the server-side state for one OAuth authorization
// attempt. Keeping state server-side prevents callback tampering and lets one
// callback endpoint handle every provider.
type ConnectorOAuthState struct {
	models.Base
	State         string     `json:"state" gorm:"column:state;type:varchar(128);not null;uniqueIndex"`
	OwnerUsername string     `json:"owner_username" gorm:"column:owner_username;type:varchar(100);not null;index"`
	Provider      string     `json:"provider" gorm:"column:provider;type:varchar(80);not null;index"`
	ResourcePath  string     `json:"resource_path" gorm:"column:resource_path;type:varchar(500);index"`
	Scopes        string     `json:"scopes" gorm:"column:scopes;type:text"`
	DisplayName   string     `json:"display_name" gorm:"column:display_name;type:varchar(200)"`
	RedirectAfter string     `json:"redirect_after" gorm:"column:redirect_after;type:varchar(1000)"`
	CodeVerifier  string     `json:"-" gorm:"column:code_verifier;type:varchar(256)"`
	Status        string     `json:"status" gorm:"column:status;type:varchar(30);not null;default:pending;index"`
	ExpiresAt     time.Time  `json:"expires_at" gorm:"column:expires_at;not null;index"`
	UsedAt        *time.Time `json:"used_at" gorm:"column:used_at"`
}

func (ConnectorOAuthState) TableName() string {
	return "connector_oauth_states"
}

// ConnectorOAuthToken stores encrypted OAuth tokens for one connection.
// Access/refresh token plaintext should never leave connector-server storage.
type ConnectorOAuthToken struct {
	models.Base
	ConnectionID       string     `json:"connection_id" gorm:"column:connection_id;type:varchar(64);not null;uniqueIndex"`
	OwnerUsername      string     `json:"owner_username" gorm:"column:owner_username;type:varchar(100);not null;index"`
	Provider           string     `json:"provider" gorm:"column:provider;type:varchar(80);not null;index"`
	AccessTokenCipher  string     `json:"-" gorm:"column:access_token_cipher;type:text;not null"`
	RefreshTokenCipher string     `json:"-" gorm:"column:refresh_token_cipher;type:text"`
	TokenType          string     `json:"token_type" gorm:"column:token_type;type:varchar(50)"`
	Scopes             string     `json:"scopes" gorm:"column:scopes;type:text"`
	Expiry             *time.Time `json:"expiry" gorm:"column:expiry;index"`
	LastRefreshAt      *time.Time `json:"last_refresh_at" gorm:"column:last_refresh_at"`
	RawResponse        string     `json:"-" gorm:"column:raw_response;type:text"`
}

func (ConnectorOAuthToken) TableName() string {
	return "connector_oauth_tokens"
}

// ConnectorOAuthProviderSetting stores platform-managed OAuth provider config.
// client_secret is encrypted; responses should expose only a has_secret flag.
type ConnectorOAuthProviderSetting struct {
	models.Base
	Code               string `json:"code" gorm:"column:code;type:varchar(80);not null;uniqueIndex"`
	Name               string `json:"name" gorm:"column:name;type:varchar(120);not null"`
	ClientID           string `json:"client_id" gorm:"column:client_id;type:varchar(300)"`
	ClientSecretCipher string `json:"-" gorm:"column:client_secret_cipher;type:text"`
	AuthURL            string `json:"auth_url" gorm:"column:auth_url;type:varchar(1000)"`
	TokenURL           string `json:"token_url" gorm:"column:token_url;type:varchar(1000)"`
	UserInfoURL        string `json:"user_info_url" gorm:"column:user_info_url;type:varchar(1000)"`
	Scopes             string `json:"scopes" gorm:"column:scopes;type:text"`
	UsePKCE            *bool  `json:"use_pkce" gorm:"column:use_pkce"`
	TokenRequestMode   string `json:"token_request_mode" gorm:"column:token_request_mode;type:varchar(30)"`
	ClientIDParam      string `json:"client_id_param" gorm:"column:client_id_param;type:varchar(80)"`
	ClientSecretParam  string `json:"client_secret_param" gorm:"column:client_secret_param;type:varchar(80)"`
	GrantTypeParam     string `json:"grant_type_param" gorm:"column:grant_type_param;type:varchar(80)"`
	CodeParam          string `json:"code_param" gorm:"column:code_param;type:varchar(80)"`
	RefreshTokenParam  string `json:"refresh_token_param" gorm:"column:refresh_token_param;type:varchar(80)"`
	RedirectURIParam   string `json:"redirect_uri_param" gorm:"column:redirect_uri_param;type:varchar(80)"`
	ExtraAuthParams    string `json:"extra_auth_params" gorm:"column:extra_auth_params;type:text"`
	ExtraTokenParams   string `json:"extra_token_params" gorm:"column:extra_token_params;type:text"`
	ExternalIDField    string `json:"external_id_field" gorm:"column:external_id_field;type:varchar(120)"`
	DisplayNameField   string `json:"display_name_field" gorm:"column:display_name_field;type:varchar(120)"`
	Enabled            bool   `json:"enabled" gorm:"column:enabled;not null;default:true;index"`
}

func (ConnectorOAuthProviderSetting) TableName() string {
	return "connector_oauth_provider_settings"
}
