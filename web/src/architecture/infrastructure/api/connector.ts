import { del, get, post, put } from '@/architecture/infrastructure/apiClient/request'

export const CONNECTOR_GLOBAL_RESOURCE_PATH = '/'

export interface ConnectorResourceSummary {
  page_count?: number
  database_count?: number
  samples?: string[]
}

export interface ConnectorConnectionProfile {
  provider?: string
  display_name?: string
  account_id?: string
  account_name?: string
  avatar_url?: string
  account_url?: string
  workspace_id?: string
  workspace_name?: string
  workspace_icon?: string
  resource_summary?: ConnectorResourceSummary
  last_enriched_at?: string
}

export interface ConnectorProviderCapabilities {
  oauth_supported?: boolean
  proxy_supported?: boolean
  profile_supported?: boolean
  resource_summary_supported?: boolean
}

export interface ConnectorConnectionInfo {
  id: number
  connection_id: string
  owner_username: string
  provider: string
  auth_type: string
  display_name: string
  external_account_id?: string
  status: string
  metadata?: string
  profile?: ConnectorConnectionProfile
  created_at: string
  updated_at: string
}

export interface ConnectorDirectoryBindingInfo {
  id: number
  owner_username: string
  tenant_user: string
  app: string
  resource_path: string
  provider: string
  connection_id: string
  connection?: ConnectorConnectionInfo
  created_at: string
  updated_at: string
}

export interface StartConnectorOAuthReq {
  provider: string
  resource_path?: string
  connection_id?: string
  scopes?: string[]
  display_name?: string
  redirect_after?: string
}

export interface StartConnectorOAuthResp {
  provider: string
  authorize_url: string
  state: string
  expires_at: string
  callback_url: string
}

export interface ListConnectorConnectionsResp {
  connections: ConnectorConnectionInfo[]
}

export interface ResolveConnectorBindingResp {
  binding: ConnectorDirectoryBindingInfo
  connection: ConnectorConnectionInfo
  token?: {
    connection_id: string
    provider: string
    token_type?: string
    scopes?: string
    expiry?: string
    last_refresh_at?: string
    has_access: boolean
    has_refresh: boolean
  }
  resolved_from: string
  requested_path: string
  required_scopes?: string[]
  granted_scopes?: string[]
  missing_scopes?: string[]
  scope_satisfied?: boolean
}

export interface ConnectorOAuthProviderInfo {
  id?: number
  code: string
  name: string
  auth_type: string
  client_id?: string
  has_client_secret: boolean
  auth_url?: string
  token_url?: string
  scopes?: string[]
  provider_account_url?: string
  logo_url?: string
  brand_color?: string
  enabled: boolean
  active: boolean
  managed: boolean
  capabilities?: ConnectorProviderCapabilities
  created_at?: string
  updated_at?: string
}

export interface UpsertConnectorOAuthProviderReq {
  code: string
  name: string
  auth_type?: string
  client_id?: string
  client_secret?: string
  auth_url?: string
  token_url?: string
  scopes?: string[]
  provider_account_url?: string
  logo_url?: string
  brand_color?: string
  enabled?: boolean
}

export interface ListConnectorOAuthProvidersResp {
  providers: ConnectorOAuthProviderInfo[]
}

export interface GetConnectorOAuthProviderResp {
  provider: ConnectorOAuthProviderInfo
}

export interface UpsertConnectorOAuthProviderResp {
  provider: ConnectorOAuthProviderInfo
}

export function listConnectorOAuthProviders() {
  return get<ListConnectorOAuthProvidersResp>('/connector/api/v1/oauth/providers')
}

export function getConnectorOAuthProvider(provider: string) {
  return get<GetConnectorOAuthProviderResp>(`/connector/api/v1/oauth/providers/${encodeURIComponent(provider)}`)
}

export function upsertConnectorOAuthProvider(provider: string, data: UpsertConnectorOAuthProviderReq) {
  return put<UpsertConnectorOAuthProviderResp>(
    `/connector/api/v1/oauth/providers/${encodeURIComponent(provider)}`,
    data
  )
}

export function deleteConnectorOAuthProvider(provider: string) {
  return del(`/connector/api/v1/oauth/providers/${encodeURIComponent(provider)}`)
}

export function startConnectorOAuth(data: StartConnectorOAuthReq) {
  return post<StartConnectorOAuthResp>('/connector/api/v1/oauth/authorize', {
    ...data,
    resource_path: data.resource_path || CONNECTOR_GLOBAL_RESOURCE_PATH
  })
}

export function listConnectorConnections(provider?: string) {
  return get<ListConnectorConnectionsResp>('/connector/api/v1/connections', { provider })
}

export function resolveConnectorBinding(provider: string, resourcePath: string = CONNECTOR_GLOBAL_RESOURCE_PATH, requiredScopes: string[] = []) {
  return get<ResolveConnectorBindingResp>('/connector/api/v1/resolve', {
    provider,
    resource_path: resourcePath || CONNECTOR_GLOBAL_RESOURCE_PATH,
    required_scopes: requiredScopes.join(',')
  })
}
