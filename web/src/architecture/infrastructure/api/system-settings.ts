import { get, put, post } from '@/architecture/infrastructure/apiClient/request'

export type RegistrationMode = 'admin_only' | 'email_code' | 'debug_code'
export type EmailMode = 'smtp' | 'log'

export interface EmailSettings {
  mode: EmailMode
  host: string
  port: number
  username: string
  password?: string
  password_set: boolean
  from: string
  from_name: string
}

export interface SystemSettings {
  registration_mode: RegistrationMode
  email: EmailSettings
}

export interface TLSCertificateInfo {
  subject: string
  issuer: string
  dns_names: string[]
  ip_addresses: string[]
  not_before: string
  not_after: string
  is_self_signed: boolean
}

export interface TLSSettings {
  mode: string
  base_url: string
  cert_file: string
  key_file: string
  cert_exists: boolean
  key_exists: boolean
  ready: boolean
  writable: boolean
  reload_supported: boolean
  certificate?: TLSCertificateInfo
  message?: string
}

export interface UpdateTLSCertificatePayload {
  certificate_pem: string
  private_key_pem: string
  reload: boolean
}

export interface AuthLoginProviderField {
  key: string
  label: string
  type: string
  required: boolean
  secret: boolean
  help?: string
  placeholder?: string
  value?: string
  value_set: boolean
}

export interface AuthLoginProviderInfo {
  code: string
  name: string
  description: string
  action: string
  enabled: boolean
  configured: boolean
  status: 'unconfigured' | 'disabled' | 'enabled' | string
  callback_path?: string
  docs_url?: string
  fields: AuthLoginProviderField[]
  updated_by?: string
  updated_at?: string
}

export interface ListAuthLoginProvidersResp {
  providers: AuthLoginProviderInfo[]
}

export interface LoginMethodInfo {
  provider: string
  label: string
  action: string
  description?: string
  authorize_path?: string
}

export interface ListLoginMethodsResp {
  methods: LoginMethodInfo[]
}

export interface LoginAnnouncement {
  enabled: boolean
  markdown: string
}

export interface LogArchiveResourceSummary {
  resource_path: string
  count: number
}

export interface LogArchiveBatch {
  id: number
  archive_key: string
  archive_type: string
  tenant_user: string
  app: string
  range_started_at: string
  range_ended_at: string
  record_count: number
  object_ref: string
  file_name: string
  file_size: number
  sha256: string
  status: 'exporting' | 'uploaded' | 'completed' | 'failed' | string
  summary_json?: { top_resource_paths?: LogArchiveResourceSummary[] }
  error_message?: string
  archived_at?: string
  deleted_at_source?: string
  created_at: string
}

export interface ListLogArchiveBatchesResp {
  list: LogArchiveBatch[]
  total: number
  retention_days: number
  cron_expr: string
  timezone: string
}

export interface SystemResourceComponent {
  key: string
  name: string
  pool_key: string
  used_bytes: number
  available: boolean
}

export interface SystemEnvironmentInfo {
  mode: 'development' | 'production' | string
  deployment: 'source' | 'aio' | 'container' | 'host' | string
  containerized: boolean
  container_engine: string
  container_remote: boolean
  storage_root_source: string
}

export interface SystemStoragePool {
  key: string
  name: string
  path?: string
  total_bytes: number
  used_bytes: number
  free_bytes: number
  used_percent: number
  primary: boolean
  available: boolean
}

export interface SystemResourceSnapshot {
  collected_at: string
  hostname: string
  operating_system: string
  architecture: string
  cpu_cores: number
  cpu_used_percent: number
  cpu_available: boolean
  load_1: number
  load_5: number
  load_15: number
  load_available: boolean
  uptime_seconds: number
  memory_total_bytes: number
  memory_used_bytes: number
  memory_used_percent: number
  memory_available: boolean
  disk_mount: string
  disk_total_bytes: number
  disk_used_bytes: number
  disk_free_bytes: number
  disk_used_percent: number
  environment: SystemEnvironmentInfo
  storage_pools: SystemStoragePool[]
  components: SystemResourceComponent[]
}

export interface SystemResourceHistoryPoint {
  collected_at: string
  disk_used_bytes: number
  disk_used_percent: number
  memory_used_percent: number
  load_1: number
}

export interface StorageExpansionForecast {
  status: 'healthy' | 'warning' | 'critical' | string
  pool_key: string
  current_used_percent: number
  daily_growth_bytes: number
  target_percent: number
  days_to_target?: number
  message: string
}

export interface SystemResourceOverview {
  current: SystemResourceSnapshot
  history: SystemResourceHistoryPoint[]
  history_hours: number
  sample_interval_minutes: number
  forecast: StorageExpansionForecast
}

export function getSystemSettings() {
  return get<SystemSettings>('/hr/api/v1/system/settings')
}

export function updateSystemSettings(data: SystemSettings) {
  return put<SystemSettings>('/hr/api/v1/system/settings', data)
}

export function testSystemEmail(to: string) {
  return post('/hr/api/v1/system/settings/email/test', { to })
}

export function getTLSSettings() {
  return get<TLSSettings>('/hr/api/v1/system/settings/tls')
}

export function updateTLSCertificate(data: UpdateTLSCertificatePayload) {
  return put<TLSSettings>('/hr/api/v1/system/settings/tls', data)
}

export function reloadTLSCertificate() {
  return post<TLSSettings>('/hr/api/v1/system/settings/tls/reload')
}

export function listAuthLoginProviders() {
  return get<ListAuthLoginProvidersResp>('/hr/api/v1/system/auth/providers')
}

export function getLoginAnnouncementConfig() {
  return get<LoginAnnouncement>('/hr/api/v1/system/auth/login-announcement')
}

export function updateLoginAnnouncementConfig(data: LoginAnnouncement) {
  return put<LoginAnnouncement>('/hr/api/v1/system/auth/login-announcement', data)
}

export function updateAuthLoginProviderConfig(code: string, config: Record<string, string>) {
  return put<AuthLoginProviderInfo>(`/hr/api/v1/system/auth/providers/${encodeURIComponent(code)}/config`, {
    config
  })
}

export function updateAuthLoginProviderEnabled(code: string, enabled: boolean) {
  return put<AuthLoginProviderInfo>(`/hr/api/v1/system/auth/providers/${encodeURIComponent(code)}/enabled`, {
    enabled
  })
}

export function listLoginMethods() {
  return get<ListLoginMethodsResp>('/hr/api/v1/auth/methods')
}

export function listLogArchiveBatches(page = 1, pageSize = 20) {
  return get<ListLogArchiveBatchesResp>('/workspace/api/v1/system/log_archives', { page, page_size: pageSize })
}

export function getSystemResourceOverview(hours = 24 * 7) {
  return get<SystemResourceOverview>('/hr/api/v1/system/settings/resources', { hours })
}
