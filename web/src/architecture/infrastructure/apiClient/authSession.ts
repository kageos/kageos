import type { ApiResponse } from '@/architecture/shared/apiTypes'

const AUTH_ERROR_CODES = new Set(['unauthenticated', 'TOKEN_EXPIRED', 'TOKEN_INVALID', 'TOKEN_BLACKLISTED'])
const AUTH_ERROR_MESSAGE_KEYWORDS = [
  '认证令牌无效或已过期',
  '未提供认证令牌',
  'RefreshToken无效或已过期',
  '刷新Token失败',
  'Token 已过期',
  'Token 无效',
  'Token 已失效',
  '登录已过期',
]
const LEGACY_AUTH_ERROR_CODE = 7
const AUTH_SUBJECT_PATTERN = /(token|refresh\s*token|refreshtoken|令牌|认证|登录)/i
const AUTH_STATE_PATTERN = /(过期|无效|失效|未提供|重新登录|blacklist|expired|invalid)/i

export function isRefreshRequestUrl(url?: string): boolean {
  return typeof url === 'string' && url.includes('/auth/refresh')
}

// These endpoints establish identity and must not inherit a stale access token.
// In particular, a standards-compliant 401 from login means bad credentials,
// not that the client should try to refresh an existing session.
export function isPublicAuthRequestUrl(url?: string): boolean {
  if (typeof url !== 'string') {
    return false
  }
  const path = url.split(/[?#]/, 1)[0] ?? ''
  return /\/auth\/(?:login|register|send-email-code|forgot-password|companies\/search)(?:\/|$)/.test(path) ||
    path.includes('/auth/oauth/')
}

export function extractApiMessage(payload?: Partial<ApiResponse> | Record<string, unknown> | null): string {
  if (!payload || typeof payload !== 'object') {
    return ''
  }

  const rawMessage = (payload as Record<string, unknown>).msg ?? (payload as Record<string, unknown>).message
  return typeof rawMessage === 'string' ? rawMessage : ''
}

export function isAuthExpiredBusinessResponse(
  payload?: Partial<ApiResponse> | Record<string, unknown> | null
): boolean {
  if (!payload || typeof payload !== 'object') {
    return false
  }

  const code = (payload as Record<string, unknown>).code
  if (typeof code === 'string' && AUTH_ERROR_CODES.has(code)) {
    return true
  }

  const message = extractApiMessage(payload)
  if (AUTH_ERROR_MESSAGE_KEYWORDS.some((keyword) => message.includes(keyword))) {
    return true
  }

  if (code === LEGACY_AUTH_ERROR_CODE && AUTH_SUBJECT_PATTERN.test(message) && AUTH_STATE_PATTERN.test(message)) {
    return true
  }

  return false
}
