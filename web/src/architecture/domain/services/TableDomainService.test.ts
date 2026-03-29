import { describe, expect, it, vi } from 'vitest'
import { TableDomainService, type TableResponse, type TableState } from './TableDomainService'
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
          pageSize: 20,
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
      pageSize: 20,
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
  it('restores request search fields from namespaced query keys', () => {
    const service = createService()
    const functionDetail = {
      request: [
        {
          code: 'status',
          name: '状态',
          widget: { type: 'select' }
        }
      ],
      response: []
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      s_status: 'open',
      page: '2',
      page_size: '50'
    })

    expect(restored.searchForm).toEqual({ status: 'open' })
    expect(restored.pagination).toEqual({ page: 2, pageSize: 50 })
  })

  it('keeps compatibility with legacy raw request search keys', () => {
    const service = createService()
    const functionDetail = {
      request: [
        {
          code: 'status',
          name: '状态',
          widget: { type: 'select' }
        }
      ],
      response: []
    } as any

    const restored = service.restoreFromURL(functionDetail, {
      status: 'legacy-open'
    })

    expect(restored.searchForm).toEqual({ status: 'legacy-open' })
  })

  it('keeps only the latest load result when earlier requests finish later', async () => {
    const firstResponse = createDeferred<TableResponse>()
    const secondResponse = createDeferred<TableResponse>()
    const stateManager = createStateManager()
    const eventBus = {
      emit: vi.fn(),
      on: () => () => {},
      off: () => {},
      once: () => {}
    }
    const apiClient = {
      get: vi.fn()
        .mockReturnValueOnce(firstResponse.promise)
        .mockReturnValueOnce(secondResponse.promise),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn()
    }

    const service = new TableDomainService(apiClient as any, stateManager as any, eventBus as any)
    const functionDetail = {
      router: '/orders',
      request: [],
      response: []
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
})
