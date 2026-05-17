import { describe, expect, it } from 'vitest'
import {
  deleteFieldQueryKey,
  isDisplayCompanionQueryKey,
  isGeneratedFieldQueryKey,
  isPersistentPlatformStateQueryKey,
  isPlatformStateQueryKey,
  isStaleTableFilterQueryKey,
  isTableControlQueryKey
} from './queryParamKeys'

describe('queryParamKeys', () => {
  it('classifies sdk-app params and platform state separately', () => {
    expect(isPlatformStateQueryKey('_tab')).toBe(true)
    expect(isPersistentPlatformStateQueryKey('_tab')).toBe(true)
    expect(isPersistentPlatformStateQueryKey('_link_type')).toBe(false)
    expect(isPersistentPlatformStateQueryKey('_genre__display')).toBe(false)

    expect(isTableControlQueryKey('page')).toBe(true)
    expect(isTableControlQueryKey('genre')).toBe(false)
    expect(isStaleTableFilterQueryKey('in')).toBe(true)
    expect(isStaleTableFilterQueryKey('genre')).toBe(false)
  })

  it('recognizes generated field params as unsupported URL keys', () => {
    expect(isGeneratedFieldQueryKey('s_genre')).toBe(true)
    expect(isGeneratedFieldQueryKey('f_genre')).toBe(true)
    expect(isGeneratedFieldQueryKey('s_style__display')).toBe(true)
    expect(isGeneratedFieldQueryKey('_style__display')).toBe(true)
    expect(isGeneratedFieldQueryKey('genre')).toBe(false)
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
