import { describe, expect, it } from 'vitest'
import { isWorkspaceForbiddenError } from './apiError'

describe('apiError', () => {
  it('recognizes workspace permission business errors', () => {
    expect(isWorkspaceForbiddenError({
      response: {
        status: 200,
        data: {
          code: 7,
          msg: '无权限查看该 workspace'
        }
      }
    })).toBe(true)
  })

  it('recognizes http forbidden errors', () => {
    expect(isWorkspaceForbiddenError({
      response: {
        status: 403,
        data: {
          msg: '请求被拒绝'
        }
      }
    })).toBe(true)
  })

  it('does not treat auth expiration or unrelated business errors as workspace forbidden', () => {
    expect(isWorkspaceForbiddenError({
      response: {
        status: 200,
        data: {
          code: 7,
          msg: '认证令牌无效或已过期'
        }
      }
    })).toBe(false)

    expect(isWorkspaceForbiddenError({
      response: {
        status: 200,
        data: {
          code: 7,
          msg: '无权限查询该表格'
        }
      }
    })).toBe(false)
  })
})
