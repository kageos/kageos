import { describe, expect, it } from 'vitest'
import type { TableState } from '@/architecture/domain/types'
import {
  buildClearedTableState,
  buildRestoredTableState,
  decideTableRestoreStrategy,
  normalizeTableRouteQuery,
  shouldSkipTableReloadOnRouteChange,
  shouldSyncTableURLAfterRestore
} from './tableInitializationRuntime'

function createTableState(overrides: Partial<TableState> = {}): TableState {
  return {
    data: [],
    loading: false,
    searchParams: { status: 'open' },
    searchForm: { status: 'open' },
    sortParams: { field: 'created_at', order: 'desc' },
    sorts: [{ field: 'created_at', order: 'desc' }],
    hasManualSort: true,
    pagination: {
      currentPage: 3,
      pageSize: 50,
      total: 99
    },
    ...overrides
  }
}

describe('tableInitializationRuntime', () => {
  it('normalizes route query into domain restore payload', () => {
    expect(
      normalizeTableRouteQuery({
        page: 2,
        sorts: [JSON.stringify([{ field: 'id', order: 'desc' }]), null],
        empty: null,
        keyword: 'alice'
      })
    ).toEqual({
      page: '2',
      sorts: [JSON.stringify([{ field: 'id', order: 'desc' }])],
      keyword: 'alice'
    })
  })

  it('prefers URL restore for link navigation even when tab state exists', () => {
    expect(
      decideTableRestoreStrategy({
        pathMatches: true,
        query: { _link_type: 'table', page: '2' },
        searchForm: { status: 'open' },
        isLinkNavigation: true
      })
    ).toBe('restore-from-url')
  })

  it('prefers tab state over plain URL params when navigation is not from link', () => {
    expect(
      decideTableRestoreStrategy({
        pathMatches: true,
        query: { page: '2' },
        searchForm: { status: 'open' },
        isLinkNavigation: false
      })
    ).toBe('sync-tab-state')
  })

  it('clears table state while preserving page size', () => {
    expect(buildClearedTableState(createTableState())).toEqual({
      data: [],
      loading: false,
      searchParams: {},
      searchForm: {},
      sortParams: null,
      sorts: [],
      hasManualSort: false,
      pagination: {
        currentPage: 1,
        pageSize: 50,
        total: 0
      }
    })
  })

  it('clears table state with a page-specific preferred page size', () => {
    expect(buildClearedTableState(createTableState(), 10).pagination).toEqual({
      currentPage: 1,
      pageSize: 10,
      total: 0
    })
  })

  it('applies restored search, sorts and pagination into current state', () => {
    expect(
      buildRestoredTableState(createTableState(), {
        searchForm: { owner: 'alice' },
        sorts: [{ field: 'updated_at', order: 'asc' }],
        pagination: {
          page: 5,
          pageSize: 100
        }
      })
    ).toMatchObject({
      searchForm: { owner: 'alice' },
      sorts: [{ field: 'updated_at', order: 'asc' }],
      hasManualSort: true,
      sortParams: { field: 'updated_at', order: 'asc' },
      pagination: {
        currentPage: 5,
        pageSize: 100,
        total: 99
      }
    })
  })

  it('only syncs default pagination after restore for non-link URLs without page params', () => {
    expect(
      shouldSyncTableURLAfterRestore({
        query: { like: 'name:alice' },
        isLinkNavigation: false
      })
    ).toBe(true)

    expect(
      shouldSyncTableURLAfterRestore({
        query: { page: '1', page_size: '10' },
        isLinkNavigation: false
      })
    ).toBe(false)

    expect(
      shouldSyncTableURLAfterRestore({
        query: { page: '1', page_size: '20' },
        isLinkNavigation: false,
        shouldSyncPageSize: true
      })
    ).toBe(true)

    expect(
      shouldSyncTableURLAfterRestore({
        query: { like: 'name:alice' },
        isLinkNavigation: true
      })
    ).toBe(false)
  })

  it('skips table reload for detail-only route changes', () => {
    expect(
      shouldSkipTableReloadOnRouteChange({
        source: 'router-change',
        pathMatches: true,
        isMounted: true,
        isSyncingToURL: false,
        isRestoringFromURL: false,
        isInitializing: false,
        newQuery: { page: '1', _tab: 'detail', _id: '42' },
        oldQuery: { page: '1' }
      })
    ).toBe(true)

    expect(
      shouldSkipTableReloadOnRouteChange({
        source: 'router-change',
        pathMatches: true,
        isMounted: true,
        isSyncingToURL: false,
        isRestoringFromURL: false,
        isInitializing: false,
        newQuery: { page: '1' },
        oldQuery: { page: '1', _tab: 'detail', _id: '42' }
      })
    ).toBe(true)
  })

  it('skips table reload when only internal panel route state changes', () => {
    expect(
      shouldSkipTableReloadOnRouteChange({
        source: 'router-change',
        pathMatches: true,
        isMounted: true,
        isSyncingToURL: false,
        isRestoringFromURL: false,
        isInitializing: false,
        newQuery: {
          page: '1',
          page_size: '20',
          _open: 'operate_log',
          _panel: 'operateLog',
          _focus: 'operate_log',
          _log_id: '856',
          _trace_id: '1f9ba85c-08bc-418e-8d45-5771077d4366',
          _source_path: '/system/demos/inventory/supplier_list.table'
        },
        oldQuery: {
          page: '1',
          page_size: '20'
        }
      })
    ).toBe(true)
  })

  it('skips table reload when only generated display params change', () => {
    expect(
      shouldSkipTableReloadOnRouteChange({
        source: 'router-change',
        pathMatches: true,
        isMounted: true,
        isSyncingToURL: false,
        isRestoringFromURL: false,
        isInitializing: false,
        newQuery: { page: '1', supplier_id: '12', supplier_id__display: 'Acme' },
        oldQuery: { page: '1', supplier_id: '12' }
      })
    ).toBe(true)
  })

  it('allows table reload for real route changes', () => {
    expect(
      shouldSkipTableReloadOnRouteChange({
        source: 'router-change',
        pathMatches: true,
        isMounted: true,
        isSyncingToURL: false,
        isRestoringFromURL: false,
        isInitializing: false,
        newQuery: { page: '2' },
        oldQuery: { page: '1' }
      })
    ).toBe(false)

    expect(
      shouldSkipTableReloadOnRouteChange({
        source: 'router-change',
        pathMatches: true,
        isMounted: true,
        isSyncingToURL: false,
        isRestoringFromURL: false,
        isInitializing: false,
        newQuery: { page: '1', page_size: '20', supplier_id: '12' },
        oldQuery: { page: '1', page_size: '20', supplier_id: '11' }
      })
    ).toBe(false)
  })
})
