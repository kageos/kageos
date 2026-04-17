import { describe, expect, it } from 'vitest'
import {
  deleteScopedFieldQueryKey,
  getFormDraftQueryKey,
  getScopedFieldQueryValue,
  getSearchFieldQueryKey,
  shouldUseRawFormQueryKeys,
  shouldAllowLegacyFormDraftFallback
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

  it('allows legacy form draft fallback for link navigation add-dialog URLs', () => {
    expect(
      shouldAllowLegacyFormDraftFallback({
        _tab: 'OnTableAddRow',
        _link_type: 'table',
        job_id: '6'
      })
    ).toBe(true)
  })

  it('allows legacy form draft fallback for normal add-dialog URLs', () => {
    expect(
      shouldAllowLegacyFormDraftFallback({
        _tab: 'OnTableAddRow',
        job_id: '6'
      })
    ).toBe(true)
  })

  it('uses raw form query keys for add-dialog URLs', () => {
    const query = {
      _tab: 'OnTableAddRow',
      job_id: '6'
    }

    expect(shouldUseRawFormQueryKeys(query)).toBe(true)
    expect(shouldAllowLegacyFormDraftFallback(query)).toBe(true)
  })
})
