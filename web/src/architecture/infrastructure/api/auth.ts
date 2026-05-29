import { get, post } from '@/architecture/infrastructure/apiClient/request'
import type { UserInfo, LoginRequest, RegisterRequest, CompanyOption } from '@/architecture/domain/types'

// 用户注册
export function register(data: RegisterRequest) {
  return post('/hr/api/v1/auth/register', data)
}

export function searchCompanies(keyword: string, limit = 10) {
  return get<{ companies: CompanyOption[] }>('/hr/api/v1/auth/companies/search', { keyword, limit })
}

interface PublicLogoUploadToken {
  key: string
  bucket?: string
  upload_url: string
  download_url?: string
  headers?: Record<string, string>
}

interface PublicLogoUploadComplete {
  download_url: string
  key: string
  bucket?: string
}

export async function uploadCompanyLogo(file: File): Promise<string> {
  const token = await post<PublicLogoUploadToken>('/storage/api/v1/public/company_logo/upload_token', {
    file_name: file.name,
    content_type: file.type || 'application/octet-stream',
    file_size: file.size,
  })
  if (!token.upload_url) {
    throw new Error('未获取到企业 Logo 上传地址')
  }

  const uploadResp = await fetch(token.upload_url, {
    method: 'PUT',
    headers: {
      'Content-Type': file.type || 'application/octet-stream',
      ...(token.headers || {}),
    },
    body: file,
  })
  if (!uploadResp.ok) {
    throw new Error('企业 Logo 上传失败')
  }

  const complete = await post<PublicLogoUploadComplete>('/storage/api/v1/public/company_logo/upload_complete', {
    key: token.key,
    bucket: token.bucket,
    success: true,
    router: 'public/company-logos',
    file_name: file.name,
    file_size: file.size,
    content_type: file.type || 'application/octet-stream',
  })
  if (!complete.download_url) {
    throw new Error('企业 Logo 上传完成后未返回访问链接')
  }
  return complete.download_url
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
  return post<{ debug_code?: string }>(url, { email })
}

// 忘记密码（简化版：直接通过验证码重置密码）
export function forgotPassword(data: { email: string; code: string; password: string }) {
  return post('/hr/api/v1/auth/forgot_password', data)
}

/** 超管一键创建用户（免邮箱验证，仅已登录的 system 用户可调用） */
export function createUserBySecret(data: { username: string; password: string }) {
  return post<{ user_id: number }>('/hr/api/v1/user/create_user_by_secret', data)
}
