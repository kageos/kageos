package dto

import "encoding/json"

type ConnectorConnectionInfo struct {
	ID                int64  `json:"id"`
	ConnectionID      string `json:"connection_id"`
	OwnerUsername     string `json:"owner_username"`
	Provider          string `json:"provider"`
	DisplayName       string `json:"display_name"`
	ExternalAccountID string `json:"external_account_id,omitempty"`
	Status            string `json:"status"`
	Metadata          string `json:"metadata,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type CreateConnectorConnectionReq struct {
	Provider          string                 `json:"provider" binding:"required" example:"github"`
	DisplayName       string                 `json:"display_name" example:"GitHub - beiluo"`
	ExternalAccountID string                 `json:"external_account_id,omitempty" example:"beiluo"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

type CreateConnectorConnectionResp struct {
	Connection ConnectorConnectionInfo `json:"connection"`
}

type ListConnectorConnectionsResp struct {
	Connections []ConnectorConnectionInfo `json:"connections"`
}

type BindConnectorDirectoryReq struct {
	ResourcePath string `json:"resource_path" binding:"required" example:"/"`
	Provider     string `json:"provider" binding:"required" example:"github"`
	ConnectionID string `json:"connection_id" binding:"required" example:"conn_abc123"`
}

type ConnectorDirectoryBindingInfo struct {
	ID            int64                    `json:"id"`
	OwnerUsername string                   `json:"owner_username"`
	TenantUser    string                   `json:"tenant_user"`
	App           string                   `json:"app"`
	ResourcePath  string                   `json:"resource_path"`
	Provider      string                   `json:"provider"`
	ConnectionID  string                   `json:"connection_id"`
	Connection    *ConnectorConnectionInfo `json:"connection,omitempty"`
	CreatedAt     string                   `json:"created_at"`
	UpdatedAt     string                   `json:"updated_at"`
}

type BindConnectorDirectoryResp struct {
	Binding ConnectorDirectoryBindingInfo `json:"binding"`
}

type ListConnectorDirectoryBindingsResp struct {
	Bindings []ConnectorDirectoryBindingInfo `json:"bindings"`
}

type ResolveConnectorBindingResp struct {
	Binding        ConnectorDirectoryBindingInfo `json:"binding"`
	Connection     ConnectorConnectionInfo       `json:"connection"`
	Token          *ConnectorTokenInfo           `json:"token,omitempty"`
	ResolvedFrom   string                        `json:"resolved_from"`
	RequestedPath  string                        `json:"requested_path"`
	RequiredScopes []string                      `json:"required_scopes,omitempty"`
	GrantedScopes  []string                      `json:"granted_scopes,omitempty"`
	MissingScopes  []string                      `json:"missing_scopes,omitempty"`
	ScopeSatisfied bool                          `json:"scope_satisfied"`
}

type StartConnectorOAuthReq struct {
	Provider      string   `json:"provider" binding:"required" example:"github"`
	ResourcePath  string   `json:"resource_path,omitempty" example:"/"`
	Scopes        []string `json:"scopes,omitempty" example:"repo,user:email"`
	DisplayName   string   `json:"display_name,omitempty" example:"GitHub - beiluo"`
	RedirectAfter string   `json:"redirect_after,omitempty" example:"/settings/connectors"`
}

type StartConnectorOAuthResp struct {
	Provider     string `json:"provider"`
	AuthorizeURL string `json:"authorize_url"`
	State        string `json:"state"`
	ExpiresAt    string `json:"expires_at"`
	CallbackURL  string `json:"callback_url"`
}

type ConnectorTokenInfo struct {
	ConnectionID  string `json:"connection_id"`
	Provider      string `json:"provider"`
	TokenType     string `json:"token_type,omitempty"`
	Scopes        string `json:"scopes,omitempty"`
	Expiry        string `json:"expiry,omitempty"`
	LastRefreshAt string `json:"last_refresh_at,omitempty"`
	HasAccess     bool   `json:"has_access"`
	HasRefresh    bool   `json:"has_refresh"`
}

type ConnectorOAuthCallbackResp struct {
	Connection ConnectorConnectionInfo        `json:"connection"`
	Token      ConnectorTokenInfo             `json:"token"`
	Binding    *ConnectorDirectoryBindingInfo `json:"binding,omitempty"`
}

type RefreshConnectorOAuthTokenResp struct {
	Connection ConnectorConnectionInfo `json:"connection"`
	Token      ConnectorTokenInfo      `json:"token"`
}

type ConnectorOAuthProviderInfo struct {
	ID              int64    `json:"id,omitempty"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	ClientID        string   `json:"client_id,omitempty"`
	HasClientSecret bool     `json:"has_client_secret"`
	AuthURL         string   `json:"auth_url,omitempty"`
	TokenURL        string   `json:"token_url,omitempty"`
	Scopes          []string `json:"scopes,omitempty"`
	Enabled         bool     `json:"enabled"`
	Active          bool     `json:"active"`
	Managed         bool     `json:"managed"`
	CreatedAt       string   `json:"created_at,omitempty"`
	UpdatedAt       string   `json:"updated_at,omitempty"`
}

type UpsertConnectorOAuthProviderReq struct {
	Code         string   `json:"code" example:"github"`
	Name         string   `json:"name" example:"GitHub"`
	ClientID     string   `json:"client_id,omitempty"`
	ClientSecret string   `json:"client_secret,omitempty"`
	AuthURL      string   `json:"auth_url,omitempty"`
	TokenURL     string   `json:"token_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

type ListConnectorOAuthProvidersResp struct {
	Providers []ConnectorOAuthProviderInfo `json:"providers"`
}

type GetConnectorOAuthProviderResp struct {
	Provider ConnectorOAuthProviderInfo `json:"provider"`
}

type UpsertConnectorOAuthProviderResp struct {
	Provider ConnectorOAuthProviderInfo `json:"provider"`
}

type ConnectorProxyReq struct {
	Provider     string            `json:"provider" binding:"required" example:"github"`
	ResourcePath string            `json:"resource_path,omitempty" example:"/system/connector/github/me.form"`
	Method       string            `json:"method" binding:"required" example:"GET"`
	Path         string            `json:"path" binding:"required" example:"/user"`
	Query        map[string]string `json:"query,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         json.RawMessage   `json:"body,omitempty"`
}

type ConnectorProxyResp struct {
	Provider      string            `json:"provider"`
	StatusCode    int               `json:"status_code"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          json.RawMessage   `json:"body,omitempty"`
	ResolvedFrom  string            `json:"resolved_from,omitempty"`
	RequestedPath string            `json:"requested_path,omitempty"`
}
