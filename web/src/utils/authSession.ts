import type { ApiResponse } from '@/types'

const AUTH_ERROR_CODES = new Set(['TOKEN_EXPIRED', 'TOKEN_INVALID', 'TOKEN_BLACKLISTED'])
const AUTH_ERROR_MESSAGE_KEYWORDS = [
  '认证令牌无效或已过期',
  '未提供认证令牌',
  'Token 已过期',
  'Token 无效',
  'Token 已失效',
  '登录已过期',
]

export function isRefreshRequestUrl(url?: string): boolean {
  return typeof url === 'string' && url.includes('/auth/refresh')
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
  return AUTH_ERROR_MESSAGE_KEYWORDS.some((keyword) => message.includes(keyword))
}
