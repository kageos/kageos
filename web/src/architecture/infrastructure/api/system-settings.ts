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
