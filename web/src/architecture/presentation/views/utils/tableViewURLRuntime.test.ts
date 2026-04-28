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
    schema: {
      version: 1,
      type: 'table',
      table: {
        request: [
          {
            code: 'status',
            name: '状态',
            widget: { type: 'select' }
          }
        ],
        fields: [
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
            widget: { type: 'datetime' }
          }
        ]
      }
    }
  } as any
}

describe('tableViewURLRuntime', () => {
  it('collects request field codes for filtering raw request search params', () => {
    expect(Array.from(getTableRequestFieldCodes(createFunctionDetail()))).toEqual(['status'])
  })

  it('builds table query params from scoped state, default sorts and raw request search', () => {
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
      status: 'open'
    })
  })

  it('keeps request fields aligned with backend form tags when a table field has the same code', () => {
    expect(
      buildTableURLQueryParams({
        functionDetail: {
          schema: {
            version: 1,
            type: 'table',
            table: {
              request: [
                {
                  code: 'genre',
                  name: '体裁',
                  widget: { type: 'select' }
                }
              ],
              fields: [
                {
                  code: 'genre',
                  name: '体裁',
                  search: 'in',
                  widget: { type: 'select' }
                },
                {
                  code: 'style',
                  name: '格律形式',
                  search: 'in',
                  widget: { type: 'select' }
                }
              ]
            }
          }
        } as any,
        state: createState({
          searchForm: {
            genre: '诗',
            style: '律诗'
          }
        }),
        buildDefaultSorts: () => [{ field: 'id', order: 'desc' }]
      })
    ).toEqual({
      page: '2',
      page_size: '50',
      sorts: 'id:desc',
      genre: '诗',
      in: 'style:律诗'
    })
  })

  it('builds only backend search params for stored search field values', () => {
    expect(
      buildTableURLQueryParams({
        functionDetail: {
          schema: {
            version: 1,
            type: 'table',
            table: {
              request: [],
              fields: [
                {
                  code: 'job_id',
                  name: '投递职位',
                  search: 'eq',
                  callbacks: ['OnSelectFuzzy'],
                  widget: { type: 'select' }
                }
              ]
            }
          }
        } as any,
        state: createState({
          searchForm: {
            job_id: {
              raw: '1',
              display: '前端开发工程师 - 技术 (北京, 20000-35000元)',
              meta: {}
            }
          }
        }),
        buildDefaultSorts: () => [{ field: 'id', order: 'desc' }]
      })
    ).toEqual({
      page: '2',
      page_size: '50',
      sorts: 'id:desc',
      eq: 'job_id:1'
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

  it('drops stale generated field params instead of preserving old URL aliases', () => {
    expect(
      preserveExistingTableQueryParams({
        routeQuery: {
          _tab: 'detail',
          s_genre: '诗',
          f_genre: '诗',
          s_style: '律诗',
          s_style__display: '律诗',
          _genre__display: '诗',
          genre: 'legacy-raw',
          topic_id: '42'
        },
        requestFieldCodes: new Set(['genre']),
        generatedFieldCodes: new Set(['genre', 'style']),
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
      status: 'open'
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
      status: 'open'
    })
  })
})
