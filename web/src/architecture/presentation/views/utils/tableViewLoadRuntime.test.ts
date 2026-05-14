import { describe, expect, it } from 'vitest'
import type { TableState } from '@/architecture/domain/types'
import {
  buildTableLoadRequest,
  buildTableLoadingState,
  decideTableLoadGuard,
  getTableLoadErrorMessage
} from './tableViewLoadRuntime'

function createState(overrides: Partial<TableState> = {}): TableState {
  return {
    data: [],
    loading: false,
    searchParams: {},
    searchForm: {
      keyword: 'alice',
      status: 'open'
    },
    sortParams: null,
    sorts: [],
    hasManualSort: false,
    pagination: {
      currentPage: 3,
      pageSize: 50,
      total: 99
    },
    ...overrides
  }
}

describe('tableViewLoadRuntime', () => {
  it('skips load when component is unmounted', () => {
    expect(
      decideTableLoadGuard({
        isMounted: false,
        skipNextTableLoad: false
      })
    ).toBe('skip-unmounted')
  })

  it('skips the next load when detail open guard is set', () => {
    expect(
      decideTableLoadGuard({
        isMounted: true,
        skipNextTableLoad: true
      })
    ).toBe('skip-next-load')
  })

  it('marks table state loading flag without changing other fields', () => {
    expect(buildTableLoadingState(createState(), true)).toEqual({
      ...createState(),
      loading: true
    })
  })

  it('builds request payload from search form, default sort and pagination', () => {
    const request = buildTableLoadRequest({
      functionDetail: { router: '/users' } as any,
      state: createState(),
      buildDefaultSorts: () => [{ field: 'id', order: 'desc' }],
      buildSearchParams: (_detail, searchForm) => ({
        like: `keyword:${searchForm.keyword}`,
        status: searchForm.status
      })
    })

    expect(request).toEqual({
      searchParams: {
        like: 'keyword:alice',
        status: 'open'
      },
      sortParams: {
        field: 'id',
        order: 'desc'
      },
      pagination: {
        page: 3,
        pageSize: 50
      }
    })
  })

  it('drops sort params when user manually cleared all sorts', () => {
    const request = buildTableLoadRequest({
      functionDetail: { router: '/users' } as any,
      state: createState({
        hasManualSort: true
      }),
      buildDefaultSorts: () => [{ field: 'id', order: 'desc' }],
      buildSearchParams: () => ({})
    })

    expect(request.sortParams).toBeUndefined()
  })

  it('prefers backend msg and falls back to generic error text', () => {
    expect(getTableLoadErrorMessage({ response: { data: { msg: '接口异常' } } })).toBe('接口异常')
    expect(getTableLoadErrorMessage({ message: '网络错误' })).toBe('网络错误')
    expect(getTableLoadErrorMessage({})).toBe('加载数据失败，请稍后重试')
  })
})
