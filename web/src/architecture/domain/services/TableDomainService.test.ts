import { describe, expect, it, vi } from 'vitest'
import { TableDomainService } from './TableDomainService'
import type { TableListResponse, TableState } from '../types'
import { TableEvent } from '../interfaces/IEventBus'

function createService() {
  return new TableDomainService(
    {} as any,
    {
      getState: () => ({
        data: [],
        loading: false,
        searchParams: {},
        searchForm: {},
        sortParams: null,
        sorts: [],
        hasManualSort: false,
        pagination: {
          currentPage: 1,
          pageSize: 10,
          total: 0
        }
      }),
      setState: () => {}
    } as any,
    {
      emit: () => {},
      on: () => () => {},
      off: () => {}
    } as any
  )
}

function createStateManager(initialState?: Partial<TableState>) {
  let state: TableState = {
    data: [],
    loading: false,
    searchParams: {},
    searchForm: {},
    sortParams: null,
    sorts: [],
    hasManualSort: false,
    pagination: {
      currentPage: 1,
      pageSize: 10,
      total: 0
    },
    ...initialState
  }

  return {
    getState: () => state,
    setState: vi.fn((nextState: TableState) => {
      state = nextState
    })
  }
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })

  return {
    promise,
    resolve,
    reject
  }
}

describe('TableDomainService URL restore', () => {
  it('keeps integer request fields searchable for table search bars', () => {
    const service = createService()
    const functionDetail = {
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [
            {
              code: 'id',
              name: '会议室ID',
              widget: { type: 'integer' },
              data: { type: 'int' }
            },
            {
              code: 'name',
              name: '会议室名称',
              widget: { type: 'input' }
            }
          ],
          fields: []
        }
      }
    } as any

    expect(service.getSearchableFields(functionDetail).map(field => field.code)).toEqual(['id', 'name'])
  })

  it('restores request search fields from raw query keys', () => {
    const service = createService()
    const functionDetail = {
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
          fields: []
        }
      }
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      status: 'open',
      page: '2',
      page_size: '50'
    })

    expect(restored.searchForm).toEqual({ status: 'open' })
    expect(restored.pagination).toEqual({ page: 2, pageSize: 50 })
  })

  it('restores explicit request field values and builds direct params', () => {
    const service = createService()
    const functionDetail = {
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [
            {
              code: 'job_id',
              name: '投递职位',
              callbacks: ['OnSelectFuzzy'],
              widget: { type: 'select' },
              data: { type: 'int' }
            }
          ]
        }
      }
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      job_id: '1'
    })

    expect(restored.searchForm).toEqual({
      job_id: '1'
    })
    expect(service.buildSearchParams(functionDetail, restored.searchForm)).toEqual({
      job_id: '1'
    })
  })

  it('restores range request fields as direct values', () => {
    const service = createService()
    const functionDetail = {
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [
            {
              code: 'created_start',
              name: '创建开始时间',
              widget: { type: 'datetime' },
              data: { type: 'string' }
            },
            {
              code: 'created_end',
              name: '创建结束时间',
              widget: { type: 'datetime' },
              data: { type: 'string' }
            }
          ],
          fields: []
        }
      }
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      created_start: '2026-04-21 00:00:00',
      created_end: '2026-04-21 23:59:59'
    })

    expect(restored.searchForm).toEqual({
      created_start: '2026-04-21 00:00:00',
      created_end: '2026-04-21 23:59:59'
    })
    expect(service.buildSearchParams(functionDetail, restored.searchForm)).toEqual({
      created_start: '2026-04-21 00:00:00',
      created_end: '2026-04-21 23:59:59'
    })
  })

  it('uses raw request params for request fields that share a code with searchable table fields', () => {
    const service = createService()
    const functionDetail = {
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [
            {
              code: 'genre',
              name: '体裁',
              widget: { type: 'select' }
            },
            {
              code: 'style',
              name: '格律形式',
              widget: { type: 'select' }
            }
          ],
          fields: []
        }
      }
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      genre: '诗',
      style: '律诗'
    })

    expect(restored.searchForm).toEqual({
      genre: '诗',
      style: '律诗'
    })
    expect(service.buildSearchParams(functionDetail, restored.searchForm)).toEqual({
      genre: '诗',
      style: '律诗'
    })
  })

  it('restores request field values containing commas', () => {
    const service = createService()
    const functionDetail = {
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [
            {
              code: 'title',
              name: '标题',
              widget: { type: 'input' }
            },
            {
              code: 'author',
              name: '作者',
              widget: { type: 'input' }
            }
          ],
          fields: []
        }
      }
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      title: '春风,又绿江南岸',
      author: '王安石'
    })

    expect(restored.searchForm).toEqual({
      title: '春风,又绿江南岸',
      author: '王安石'
    })
  })

  it('keeps only the latest load result when earlier requests finish later', async () => {
    const firstResponse = createDeferred<TableListResponse>()
    const secondResponse = createDeferred<TableListResponse>()
    const stateManager = createStateManager()
    const eventBus = {
      emit: vi.fn(),
      on: () => () => {},
      off: () => {},
      once: () => {}
    }
    const tableGateway = {
      loadRows: vi.fn()
        .mockReturnValueOnce(firstResponse.promise)
        .mockReturnValueOnce(secondResponse.promise),
      addRow: vi.fn(),
      updateRow: vi.fn(),
      deleteRow: vi.fn()
    }

    const service = new TableDomainService(tableGateway as any, stateManager as any, eventBus as any)
    const functionDetail = {
      router: '/orders',
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [],
          fields: []
        }
      }
    } as any

    const oldLoad = service.loadData(functionDetail, { status: 'old' }, undefined, {
      page: 1,
      pageSize: 20
    })
    const latestLoad = service.loadData(functionDetail, { status: 'new' }, undefined, {
      page: 2,
      pageSize: 20
    })

    secondResponse.resolve({
      items: [{ id: 2, status: 'new' }],
      paginated: {
        current_page: 2,
        page_size: 20,
        total_count: 1,
        total_pages: 1
      }
    })

    await latestLoad

    expect(stateManager.getState().data).toEqual([{ id: 2, status: 'new' }])
    expect(stateManager.getState().searchParams).toEqual({ status: 'new' })
    expect(stateManager.getState().pagination.currentPage).toBe(2)

    firstResponse.resolve({
      items: [{ id: 1, status: 'old' }],
      paginated: {
        current_page: 1,
        page_size: 20,
        total_count: 1,
        total_pages: 1
      }
    })

    await oldLoad

    expect(stateManager.getState().data).toEqual([{ id: 2, status: 'new' }])
    expect(stateManager.getState().searchParams).toEqual({ status: 'new' })
    expect(eventBus.emit).toHaveBeenCalledTimes(1)
    expect(eventBus.emit).toHaveBeenCalledWith(TableEvent.dataLoaded, {
      data: [{ id: 2, status: 'new' }],
      pagination: {
        current_page: 2,
        page_size: 20,
        total_count: 1,
        total_pages: 1
      }
    })
  })

  it('sends multi-column sorts as structured JSON in table search params', async () => {
    const stateManager = createStateManager({
      sorts: [
        { field: 'created_at', order: 'desc' },
        { field: 'name', order: 'asc' }
      ]
    })
    const eventBus = {
      emit: vi.fn(),
      on: () => () => {},
      off: () => {},
      once: () => {}
    }
    const tableGateway = {
      loadRows: vi.fn().mockResolvedValue({
        items: [],
        paginated: {
          current_page: 2,
          page_size: 50,
          total_count: 0,
          total_pages: 0
        }
      }),
      addRow: vi.fn(),
      updateRow: vi.fn(),
      deleteRow: vi.fn()
    }

    const service = new TableDomainService(tableGateway as any, stateManager as any, eventBus as any)

    await service.loadData(
      { router: '/orders' } as any,
      { status: 'open' },
      undefined,
      { page: 2, pageSize: 50 }
    )

    expect(tableGateway.loadRows).toHaveBeenCalledWith({
      functionDetail: { router: '/orders' },
      params: {
        status: 'open',
        page: 2,
        page_size: 50,
        sorts: JSON.stringify([
          { field: 'created_at', order: 'desc' },
          { field: 'name', order: 'asc' }
        ])
      }
    })
  })

  it('sends single sort params as structured JSON in table search params', async () => {
    const stateManager = createStateManager()
    const eventBus = {
      emit: vi.fn(),
      on: () => () => {},
      off: () => {},
      once: () => {}
    }
    const tableGateway = {
      loadRows: vi.fn().mockResolvedValue({
        items: [],
        paginated: {
          current_page: 1,
          page_size: 20,
          total_count: 0,
          total_pages: 0
        }
      }),
      addRow: vi.fn(),
      updateRow: vi.fn(),
      deleteRow: vi.fn()
    }

    const service = new TableDomainService(tableGateway as any, stateManager as any, eventBus as any)

    await service.loadData(
      { router: '/orders' } as any,
      {},
      { field: 'id', order: 'desc' },
      { page: 1, pageSize: 20 }
    )

    expect(tableGateway.loadRows).toHaveBeenCalledWith({
      functionDetail: { router: '/orders' },
      params: {
        page: 1,
        page_size: 20,
        sorts: JSON.stringify([{ field: 'id', order: 'desc' }])
      }
    })
  })

  it('restores structured sorts from URL query JSON', () => {
    const service = createService()
    const functionDetail = {
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [],
          fields: [
            { code: 'created_at', name: '创建时间' },
            { code: 'name', name: '姓名' }
          ]
        }
      }
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      sorts: JSON.stringify([
        { field: 'created_at', order: 'desc' },
        { field: 'name', order: 'asc' }
      ])
    })

    expect(restored.sorts).toEqual([
      { field: 'created_at', order: 'desc' },
      { field: 'name', order: 'asc' }
    ])
  })
})
