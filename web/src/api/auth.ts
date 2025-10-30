import { get, post } from '@/utils/request'
import type { UserInfo, LoginRequest, RegisterRequest } from '@/types'

// 用户注册
export function register(data: RegisterRequest) {
  return post('/api/v1/auth/register', data)
}

// 用户登录
export function login(data: LoginRequest) {
  return post<{
    token: string
    user: UserInfo
  }>('/api/v1/auth/login', data)
}

// 刷新token
export function refreshToken() {
  return post<{
    token: string
  }>('/api/v1/auth/refresh')
}

// 用户登出
export function logout() {
  return post('/api/v1/auth/logout')
}

// 获取用户信息
export function getUserInfo() {
  return get<UserInfo>('/api/v1/auth/userinfo')
}

// 发送邮箱验证码
export function sendEmailCode(email: string) {
  return post('/api/v1/auth/send_email_code', { email })
}

// 验证邮箱
export function verifyEmail(email: string, code: string) {
  return post('/api/v1/auth/verify-email', { email, code })
}