import { get, post } from '@/architecture/infrastructure/apiClient/request'
import type { UserInfo, LoginRequest, RegisterRequest } from '@/architecture/domain/types'

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

// 刷新token（传入 refresh_token，返回新 token 与 refresh_token）
export function refreshToken(refreshTokenValue: string) {
  return post<{
    token: string
    refresh_token: string
  }>('/hr/api/v1/auth/refresh', { refresh_token: refreshTokenValue })
}

// 用户登出
export function logout() {
  return post('/hr/api/v1/auth/logout')
}

// 获取用户信息
export function getUserInfo() {
  return get<UserInfo>('/hr/api/v1/user/info')
}

// 发送邮箱验证码
export function sendEmailCode(email: string, codeType: 'register' | 'forgot_password' = 'register') {
  // 将 codeType 作为查询参数传递
  const url = `/hr/api/v1/auth/send_email_code?type=${codeType}`
  return post(url, { email })
}

// 忘记密码（简化版：直接通过验证码重置密码）
export function forgotPassword(data: { email: string; code: string; password: string }) {
  return post('/hr/api/v1/auth/forgot_password', data)
}

/** 超管一键创建用户（免邮箱验证，仅已登录的 system 用户可调用） */
export function createUserBySecret(data: { username: string; password: string }) {
  return post<{ user_id: number }>('/hr/api/v1/user/create_user_by_secret', data)
}
