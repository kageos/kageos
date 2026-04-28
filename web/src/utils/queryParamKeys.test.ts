import { describe, expect, it } from 'vitest'
import {
  deleteFieldQueryKey,
  getLegacyFieldQueryKeys,
  isBackendSearchOperatorQueryKey,
  isDisplayCompanionQueryKey,
  isPersistentPlatformStateQueryKey,
  isPlatformStateQueryKey,
  isTableControlQueryKey,
  isUnsupportedGeneratedFieldQueryKey
} from './queryParamKeys'

describe('queryParamKeys', () => {
  it('classifies sdk-app params, backend operators and platform state separately', () => {
    expect(isPlatformStateQueryKey('_tab')).toBe(true)
    expect(isPersistentPlatformStateQueryKey('_tab')).toBe(true)
    expect(isPersistentPlatformStateQueryKey('_link_type')).toBe(false)
    expect(isPersistentPlatformStateQueryKey('_genre__display')).toBe(false)

    expect(isTableControlQueryKey('page')).toBe(true)
    expect(isTableControlQueryKey('genre')).toBe(false)
    expect(isBackendSearchOperatorQueryKey('in')).toBe(true)
    expect(isBackendSearchOperatorQueryKey('genre')).toBe(false)
  })

  it('recognizes old generated field params without treating all s_/f_ keys as aliases', () => {
    const fieldCodes = new Set(['genre', 'style'])

    expect(isUnsupportedGeneratedFieldQueryKey('s_genre', fieldCodes)).toBe(true)
    expect(isUnsupportedGeneratedFieldQueryKey('f_genre', fieldCodes)).toBe(true)
    expect(isUnsupportedGeneratedFieldQueryKey('s_style__display', fieldCodes)).toBe(true)
    expect(isUnsupportedGeneratedFieldQueryKey('_style__display', fieldCodes)).toBe(true)
    expect(isUnsupportedGeneratedFieldQueryKey('s_external', fieldCodes)).toBe(false)
    expect(isDisplayCompanionQueryKey('any__display')).toBe(true)
  })

  it('deletes raw field params and stale generated aliases for the same field', () => {
    const query: Record<string, any> = {
      genre: '诗',
      s_genre: '旧搜索值',
      f_genre: '旧表单值',
      s_genre__display: '旧显示值',
      _genre__display: '旧显示值',
      topic_id: '42'
    }

    deleteFieldQueryKey(query, 'genre')

    expect(query).toEqual({
      topic_id: '42'
    })
    expect(getLegacyFieldQueryKeys('genre')).toContain('s_genre')
  })

  it('can preserve a raw field while still removing stale generated aliases', () => {
    const query: Record<string, any> = {
      genre: '诗',
      s_genre: '旧搜索值',
      _genre__display: '旧显示值'
    }

    deleteFieldQueryKey(query, 'genre', { deleteRaw: false })

    expect(query).toEqual({
      genre: '诗'
    })
  })
})
