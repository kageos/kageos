import { get, post } from '@/architecture/infrastructure/apiClient/request'
import type { UserInfo, LoginRequest, RegisterRequest } from '@/architecture/domain/types'

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

export interface OAuthRegistrationIntent {
  ticket: string
  provider_code: string
  provider_name: string
  email: string
  nickname: string
  avatar: string
  suggested_code: string
  code_suggestions: string[]
  redirect_after: string
  expires_at: string
}

export interface ConfirmOAuthRegistrationRequest {
  username: string
  nickname: string
  accepted_terms: boolean
  terms_version: string
  accepted_privacy: boolean
  privacy_version: string
}

export interface ConfirmOAuthRegistrationResponse {
  token: string
  refresh_token: string
  user: UserInfo
  redirect_after: string
}

export interface WechatLoginAttempt {
  attempt_token: string
  qr_code_url: string
  expires_at: string
  poll_after_ms: number
}

export interface WechatLoginCompletion {
  status: 'pending' | 'complete'
  token?: string
  refresh_token?: string
  redirect_after?: string
  registration_required?: boolean
  registration_ticket?: string
}

// 用户注册
export function register(data: RegisterRequest) {
  return post('/hr/api/v1/auth/register', data)
}

// 用户登录
export function login(data: LoginRequest) {
  return post<{
    token: string
    refresh_token: string
    user: UserInfo
  }>('/hr/api/v1/auth/login', data)
}

export function listLoginMethods() {
  return get<ListLoginMethodsResp>('/hr/api/v1/auth/methods')
}

export function createWechatLoginAttempt(redirectAfter: string) {
  return post<WechatLoginAttempt>('/hr/api/v1/auth/wechat/attempts', {
    redirect_after: redirectAfter,
  })
}

export function completeWechatLoginAttempt(attemptToken: string) {
  return post<WechatLoginCompletion>('/hr/api/v1/auth/wechat/attempts/complete', {
    attempt_token: attemptToken,
  })
}

export function getOAuthRegistrationIntent(ticket: string) {
  return get<OAuthRegistrationIntent>(`/hr/api/v1/auth/oauth/registration/${encodeURIComponent(ticket)}`)
}

export function confirmOAuthRegistration(ticket: string, data: ConfirmOAuthRegistrationRequest) {
  return post<ConfirmOAuthRegistrationResponse>(
    `/hr/api/v1/auth/oauth/registration/${encodeURIComponent(ticket)}/confirm`,
    data
  )
}

// 刷新token（传入 refresh_token，返回新 token 与 refresh_token）
export function refreshToken(refreshTokenValue: string) {
  return post<{
    token: string
    refresh_token: string
  }>('/hr/api/v1/auth/refresh', { refresh_token: refreshTokenValue })
}

// 用户登出
export function logout(token?: string) {
  return post('/hr/api/v1/auth/logout', token ? { token } : undefined)
}

// 获取用户信息
export function getUserInfo() {
  return get<UserInfo>('/hr/api/v1/user/info')
}

// 发送邮箱验证码
export function sendEmailCode(email: string, codeType: 'register' | 'forgot_password' = 'register') {
  // 将 codeType 作为查询参数传递
  const url = `/hr/api/v1/auth/send_email_code?type=${codeType}`
  return post<{ debug_code?: string }>(url, { email })
}

// 忘记密码（简化版：直接通过验证码重置密码）
export function forgotPassword(data: { email: string; code: string; password: string }) {
  return post('/hr/api/v1/auth/forgot_password', data)
}
