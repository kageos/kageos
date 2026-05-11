import { describe, expect, it } from 'vitest'
import {
  DEFAULT_TABLE_PAGE_SIZE,
  getTablePageSizePreferenceKey,
  resolveTablePageSizeForRestore
} from './tablePageSizePreference'

describe('tablePageSizePreference', () => {
  it('normalizes legacy page_size=20 to default 10 when no user preference exists', () => {
    expect(
      resolveTablePageSizeForRestore({
        queryPageSize: '20',
        preferredPageSize: null,
        isLinkNavigation: false
      })
    ).toEqual({
      pageSize: DEFAULT_TABLE_PAGE_SIZE,
      shouldSyncToURL: true
    })
  })

  it('uses stored page size preference for the table page', () => {
    expect(
      resolveTablePageSizeForRestore({
        queryPageSize: undefined,
        preferredPageSize: 20,
        isLinkNavigation: false
      })
    ).toEqual({
      pageSize: 20,
      shouldSyncToURL: true
    })
  })

  it('keeps matching stored page size without rewriting the URL', () => {
    expect(
      resolveTablePageSizeForRestore({
        queryPageSize: '20',
        preferredPageSize: 20,
        isLinkNavigation: false
      })
    ).toEqual({
      pageSize: 20,
      shouldSyncToURL: false
    })
  })

  it('respects explicit link navigation page size', () => {
    expect(
      resolveTablePageSizeForRestore({
        queryPageSize: '20',
        preferredPageSize: null,
        isLinkNavigation: true
      })
    ).toEqual({
      pageSize: 20,
      shouldSyncToURL: false
    })
  })

  it('uses router path as the page-level preference key', () => {
    expect(getTablePageSizePreferenceKey({ id: 1, code: 'list', router: '/app/orders.table' } as any))
      .toBe('aos:table-page-size:/app/orders.table')
  })
})
