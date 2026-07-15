import { describe, expect, it } from 'vitest'
import { extractApiMessage, isAuthExpiredBusinessResponse, isRefreshRequestUrl } from './authSession'

describe('authSession', () => {
  it('识别 refresh 路由', () => {
    expect(isRefreshRequestUrl('/hr/api/v1/auth/refresh')).toBe(true)
    expect(isRefreshRequestUrl('http://localhost:8080/hr/api/v1/auth/refresh')).toBe(true)
    expect(isRefreshRequestUrl('/hr/api/v1/auth/login')).toBe(false)
  })

  it('识别当前后端返回的 token 过期业务错误', () => {
    expect(isAuthExpiredBusinessResponse({ code: 7, msg: '认证令牌无效或已过期' })).toBe(true)
    expect(isAuthExpiredBusinessResponse({ code: 7, msg: '未提供认证令牌' })).toBe(true)
    expect(isAuthExpiredBusinessResponse({ code: 7, msg: '刷新Token失败: RefreshToken无效或已过期' })).toBe(true)
  })

  it('识别标准化 token 错误码', () => {
    expect(isAuthExpiredBusinessResponse({ code: 'TOKEN_EXPIRED', msg: 'Token 已过期，请重新登录' })).toBe(true)
    expect(isAuthExpiredBusinessResponse({ code: 'TOKEN_INVALID', msg: 'Token 无效，请重新登录' })).toBe(true)
  })

  it('忽略普通业务错误', () => {
    expect(isAuthExpiredBusinessResponse({ code: 7, msg: '参数错误' })).toBe(false)
    expect(isAuthExpiredBusinessResponse({ code: 7, msg: '无权限查询该表格' })).toBe(false)
  })

  it('优先读取 msg，其次兼容 message', () => {
    expect(extractApiMessage({ msg: '主消息', message: '备用消息' })).toBe('主消息')
    expect(extractApiMessage({ message: '备用消息' })).toBe('备用消息')
    expect(extractApiMessage(undefined)).toBe('')
  })
})
