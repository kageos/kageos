import { get, post } from '@/utils/request'
import type { UserInfo, LoginRequest, RegisterRequest } from '@/types'

// 用户注册
export function register(data: RegisterRequest) {
  return post('/hr/api/v1/auth/register', data)
}

// 用户登录
export function login(data: LoginRequest) {
  return post<{
    token: string
    user: UserInfo
  }>('/hr/api/v1/auth/login', data)
}

// 刷新token
export function refreshToken() {
  return post<{
    token: string
  }>('/hr/api/v1/auth/refresh')
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
export function sendEmailCode(email: string) {
  return post('/hr/api/v1/auth/send_email_code', { email })
}

// 验证邮箱
// 注意：hr-server 中没有独立的 verify-email API，验证码验证在 register 接口中完成
// 如果需要独立的验证邮箱功能，需要在 hr-server 中添加对应的 API
export function verifyEmail(email: string, code: string) {
  // TODO: hr-server 中暂无独立的 verify-email API，暂时保留此函数以保持兼容性
  // 实际验证在 register 接口中完成
  return post('/hr/api/v1/auth/verify-email', { email, code })
}