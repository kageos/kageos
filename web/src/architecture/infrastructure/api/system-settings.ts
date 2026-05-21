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

export function getSystemSettings() {
  return get<SystemSettings>('/hr/api/v1/system/settings')
}

export function updateSystemSettings(data: SystemSettings) {
  return put<SystemSettings>('/hr/api/v1/system/settings', data)
}

export function testSystemEmail(to: string) {
  return post('/hr/api/v1/system/settings/email/test', { to })
}
