import { describe, expect, it } from 'vitest'
import {
  deleteScopedFieldQueryKey,
  getFormDraftQueryKey,
  getScopedFieldQueryValue,
  getSearchFieldQueryKey
} from './queryFieldNamespace'

describe('queryFieldNamespace', () => {
  it('prefers namespaced query keys for form draft values', () => {
    const query = {
      [getFormDraftQueryKey('title')]: 'draft-title',
      title: 'legacy-title'
    }

    expect(getScopedFieldQueryValue(query, 'title', 'form')).toBe('draft-title')
  })

  it('can disable raw fallback for form draft values', () => {
    const query = {
      title: 'legacy-title'
    }

    expect(
      getScopedFieldQueryValue(query, 'title', 'form', {
        fallbackToLegacyRaw: false
      })
    ).toBeUndefined()
  })

  it('supports legacy raw fallback for search values and deletes both key shapes', () => {
    const query: Record<string, any> = {
      status: 'legacy-open',
      [getSearchFieldQueryKey('status')]: 'open'
    }

    expect(getScopedFieldQueryValue(query, 'status', 'search')).toBe('open')

    deleteScopedFieldQueryKey(query, 'status', 'search')
    expect(query).toEqual({})
  })
})
