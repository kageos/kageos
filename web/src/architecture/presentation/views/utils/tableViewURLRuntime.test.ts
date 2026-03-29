import { describe, expect, it } from 'vitest'
import type { TableState } from '@/architecture/domain/services/TableDomainService'
import {
  buildNextTableSyncQuery,
  buildTableURLQueryParams,
  getTableRequestFieldCodes,
  preserveExistingTableQueryParams
} from './tableViewURLRuntime'

function createState(overrides: Partial<TableState> = {}): TableState {
  return {
    data: [],
    loading: false,
    searchParams: {},
    searchForm: {
      status: 'open',
      keyword: 'alice'
    },
    sortParams: null,
    sorts: [],
    hasManualSort: false,
    pagination: {
      currentPage: 2,
      pageSize: 50,
      total: 0
    },
    ...overrides
  }
}

function createFunctionDetail() {
  return {
    request: [
      {
        code: 'status',
        name: '状态',
        widget: { type: 'select' }
      }
    ],
    response: [
      {
        code: 'keyword',
        name: '关键词',
        search: 'like',
        widget: { type: 'input' }
      },
      {
        code: 'created_at',
        name: '创建时间',
        search: 'gte,lte',
        widget: { type: 'timestamp' }
      }
    ]
  } as any
}

describe('tableViewURLRuntime', () => {
  it('collects request field codes for filtering and namespaced search params', () => {
    expect(Array.from(getTableRequestFieldCodes(createFunctionDetail()))).toEqual(['status'])
  })

  it('builds table query params from scoped state, default sorts and namespaced request search', () => {
    expect(
      buildTableURLQueryParams({
        functionDetail: createFunctionDetail(),
        state: createState({
          searchForm: {
            status: 'open',
            keyword: 'alice',
            created_at: ['1700000000', '1800000000']
          }
        }),
        buildDefaultSorts: () => [{ field: 'id', order: 'desc' }]
      })
    ).toEqual({
      page: '2',
      page_size: '50',
      sorts: 'id:desc',
      like: 'keyword:alice',
      gte: 'created_at:1700000000',
      lte: 'created_at:1800000000',
      s_status: 'open'
    })
  })

  it('preserves only state and custom params on non-link sync', () => {
    expect(
      preserveExistingTableQueryParams({
        routeQuery: {
          _tab: 'detail',
          _link_type: 'table',
          page: '9',
          like: 'keyword:legacy',
          s_status: 'legacy-open',
          status: 'legacy-open',
          topic_id: '42'
        },
        requestFieldCodes: new Set(['status']),
        isLinkNavigation: false
      })
    ).toEqual({
      _tab: 'detail',
      topic_id: '42'
    })
  })

  it('keeps backend search params from link navigation while removing _link_type', () => {
    expect(
      preserveExistingTableQueryParams({
        routeQuery: {
          _tab: 'detail',
          _link_type: 'table',
          eq: 'owner:alice',
          in: 'status:open,closed',
          topic_id: '42'
        },
        requestFieldCodes: new Set(['status']),
        isLinkNavigation: true
      })
    ).toEqual({
      _tab: 'detail',
      eq: 'owner:alice',
      in: 'status:open,closed',
      topic_id: '42'
    })
  })

  it('uses only fresh table query when current route is empty', () => {
    expect(
      buildNextTableSyncQuery({
        routeQuery: {},
        functionDetail: createFunctionDetail(),
        state: createState(),
        buildDefaultSorts: () => [{ field: 'id', order: 'desc' }],
        isLinkNavigation: false
      })
    ).toEqual({
      page: '2',
      page_size: '50',
      sorts: 'id:desc',
      like: 'keyword:alice',
      s_status: 'open'
    })
  })

  it('merges preserved params with the latest table query when route already has params', () => {
    expect(
      buildNextTableSyncQuery({
        routeQuery: {
          _tab: 'detail',
          topic_id: '42',
          eq: 'owner:bob',
          page: '9'
        },
        functionDetail: createFunctionDetail(),
        state: createState(),
        buildDefaultSorts: () => [{ field: 'id', order: 'desc' }],
        isLinkNavigation: true
      })
    ).toEqual({
      _tab: 'detail',
      topic_id: '42',
      eq: 'owner:bob',
      page: '2',
      page_size: '50',
      sorts: 'id:desc',
      like: 'keyword:alice',
      s_status: 'open'
    })
  })
})
