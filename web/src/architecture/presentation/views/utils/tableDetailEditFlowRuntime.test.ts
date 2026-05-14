import { describe, expect, it } from 'vitest'
import type { IEventBus } from '@/architecture/domain/interfaces/IEventBus'
import type { IFormGateway } from '@/architecture/domain/interfaces/IFormGateway'
import { TEMPLATE_TYPE } from '@/architecture/runtime/utils/functionTypes'
import { getChangedFields } from '@/architecture/runtime/utils/objectDiff'
import { TableDomainService } from '@/architecture/domain/services/TableDomainService'
import type { TableState } from '@/architecture/domain/types'
import { TableEvent } from '@/architecture/domain/interfaces/IEventBus'
import { buildEditFunctionDetail, filterDetailInitialData, getEditableFieldCodes, buildDetailBaseQuery } from '@/architecture/presentation/composables/utils/workspaceDetailRuntime'
import { createFormViewRuntime, buildInitialDataFromFormDataStore } from './formViewRuntime'
import { buildTableDetailRowPayload } from './tableViewRouteRuntime'
import { buildNextTableSyncQuery } from './tableViewURLRuntime'
import { getFormRequestFields } from '@/architecture/domain/utils/functionSchemaSelectors'

function createMockEventBus(): IEventBus {
  const listeners = new Map<string, Set<(payload?: any) => void>>()

  return {
    emit(event: string, payload?: any) {
      listeners.get(event)?.forEach(handler => handler(payload))
    },
    on(event: string, handler: (payload?: any) => void) {
      if (!listeners.has(event)) {
        listeners.set(event, new Set())
      }
      listeners.get(event)!.add(handler)
      return () => {
        listeners.get(event)?.delete(handler)
      }
    },
    off(event: string, handler: (payload?: any) => void) {
      listeners.get(event)?.delete(handler)
    },
    once(event: string, handler: (payload?: any) => void) {
      const unsubscribe = this.on(event, (payload?: any) => {
        unsubscribe()
        handler(payload)
      })
    }
  }
}

const formGatewayStub: IFormGateway = {
  submitForm: async () => ({})
}

function createMutableTableStateManager(initialState?: Partial<TableState>) {
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
    setState: (nextState: TableState) => {
      state = nextState
    }
  }
}

describe('table detail edit flow runtime', () => {
  it('keeps table search query while preparing detail drawer edit state', () => {
    const currentFunctionDetail = {
      id: 101,
      name: '用户列表',
      router: '/workspace/demo/users',
      template_type: TEMPLATE_TYPE.TABLE,
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
              code: 'id',
              name: 'ID',
              widget: { type: 'ID' },
              hide: { scenes: ['create', 'update'] }
            },
            {
              code: 'name',
              name: '姓名',
              widget: { type: 'input', config: {} },
              data: { type: 'string' },
              hide: { scenes: ['list', 'create'] }
            },
            {
              code: 'status',
              name: '状态',
              widget: { type: 'select', config: {} },
              data: { type: 'string' },
              hide: { scenes: ['list', 'create'] }
            },
            {
              code: 'created_by',
              name: '创建人',
              widget: { type: 'input', config: {} },
              data: { type: 'string' },
              hide: { scenes: ['create', 'update'] }
            }
          ]
        }
      }
    } as any

    const tableData = [
      { id: 1, name: 'Alice', status: 'open', created_by: 'system' },
      { id: 2, name: 'Bob', status: 'closed', created_by: 'admin' }
    ]

    const detailPayload = buildTableDetailRowPayload({
      row: tableData[1]!,
      tableData,
      initialMode: 'edit'
    })

    expect(detailPayload.index).toBe(1)
    expect(detailPayload.initialMode).toBe('edit')

    const editFunctionDetail = buildEditFunctionDetail(currentFunctionDetail)
    expect(getEditableFieldCodes(editFunctionDetail)).toEqual(['name', 'status'])

    const filteredInitialData = filterDetailInitialData({
      rowData: detailPayload.row,
      editFunctionDetail
    })

    expect(filteredInitialData).toEqual({
      name: 'Bob',
      status: 'closed'
    })

    expect(
      buildDetailBaseQuery({
        query: {
          page: '2',
          page_size: '20',
          status: 'closed',
          like: 'name:Bob',
          _tab: 'detail',
          _id: '2',
          name: 'draft-name'
        },
        editableFieldCodes: getEditableFieldCodes(editFunctionDetail),
        preserveRawFieldCodes: ['status']
      })
    ).toEqual({
      page: '2',
      page_size: '20',
      status: 'closed',
      like: 'name:Bob'
    })
  })

  it('hydrates edit form state from detail row and keeps only changed fields for update submit', () => {
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      formGateway: formGatewayStub
    })

    const editFunctionDetail = {
      id: 101,
      name: '用户编辑',
      router: '/workspace/demo/users',
      template_type: TEMPLATE_TYPE.FORM,
      schema: {
        version: 1,
        type: 'form',
        form: {
          request: [
            {
              code: 'name',
              name: '姓名',
              widget: { type: 'input', config: {} },
              data: { type: 'string' }
            },
            {
              code: 'status',
              name: '状态',
              widget: { type: 'select', config: {} },
              data: { type: 'string' }
            }
          ],
          response: []
        }
      }
    } as any
    const editRequestFields = getFormRequestFields(editFunctionDetail)

    const initialData = {
      name: 'Bob',
      status: 'closed'
    }

    runtime.applicationService.initializeForm(editRequestFields, initialData, true)

    expect(runtime.domainService.getSubmitData(editRequestFields)).toEqual(initialData)

    runtime.applicationService.updateFieldValue('name', {
      raw: 'Bobby',
      display: 'Bobby',
      meta: {}
    })

    const currentSubmitData = runtime.domainService.getSubmitData(editRequestFields)

    expect(buildInitialDataFromFormDataStore({
      fields: editRequestFields,
      formDataStore: runtime.formDataStore
    })).toEqual({
      name: 'Bobby',
      status: 'closed'
    })

    expect(getChangedFields(initialData, currentSubmitData).updates).toEqual({
      name: 'Bobby'
    })
  })

  it('round-trips search query into table request params and saves only changed fields from detail edit', async () => {
    const currentFunctionDetail = {
      id: 101,
      name: '用户列表',
      router: '/workspace/demo/users',
      method: 'PUT',
      template_type: TEMPLATE_TYPE.TABLE,
      schema: {
        version: 1,
        type: 'table',
        table: {
          request: [
            {
              code: 'status',
              name: '状态',
              widget: { type: 'select', config: {} }
            },
            {
              code: 'name',
              name: '姓名',
              widget: { type: 'input', config: {} }
            }
          ],
          fields: [
            {
              code: 'id',
              name: 'ID',
              widget: { type: 'ID' },
              hide: { scenes: ['create', 'update'] }
            },
            {
              code: 'name',
              name: '姓名',
              widget: { type: 'input', config: {} },
              data: { type: 'string' },
              hide: { scenes: ['list', 'create'] }
            },
            {
              code: 'status',
              name: '状态',
              widget: { type: 'select', config: {} },
              data: { type: 'string' },
              hide: { scenes: ['list', 'create'] }
            }
          ]
        }
      }
    } as any

    const tableState = {
      data: [
        { id: 1, name: 'Alice', status: 'open' },
        { id: 2, name: 'Bob', status: 'closed' }
      ],
      loading: false,
      searchParams: {},
      searchForm: {
        status: 'closed',
        name: 'Bob'
      },
      sortParams: null,
      sorts: [],
      hasManualSort: false,
      pagination: {
        currentPage: 2,
        pageSize: 20,
        total: 2
      }
    } satisfies TableState

    const query = buildNextTableSyncQuery({
      routeQuery: {
        topic_id: '42'
      },
      functionDetail: currentFunctionDetail,
      state: tableState,
      buildDefaultSorts: () => [{ field: 'id', order: 'desc' }],
      isLinkNavigation: false
    })

    expect(query).toEqual({
      topic_id: '42',
      page: '2',
      page_size: '20',
      sorts: JSON.stringify([{ field: 'id', order: 'desc' }]),
      status: 'closed',
      name: 'Bob'
    })

    const restoreService = new TableDomainService(
      {} as any,
      createMutableTableStateManager() as any,
      createMockEventBus() as any
    )

    const restored = restoreService.restoreFromURL(currentFunctionDetail, query)
    expect(restored.searchForm).toEqual({
      status: 'closed',
      name: 'Bob'
    })

    expect(restoreService.buildSearchParams(currentFunctionDetail, restored.searchForm)).toEqual({
      status: 'closed',
      name: 'Bob'
    })

    const detailPayload = buildTableDetailRowPayload({
      row: tableState.data[1]!,
      tableData: tableState.data,
      initialMode: 'edit'
    })
    const editFunctionDetail = buildEditFunctionDetail(currentFunctionDetail)
    const filteredInitialData = filterDetailInitialData({
      rowData: detailPayload.row,
      editFunctionDetail
    })
    const editRequest = getFormRequestFields(editFunctionDetail)

    const formRuntime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      formGateway: formGatewayStub
    })
    formRuntime.applicationService.initializeForm(editRequest, filteredInitialData, true)
    formRuntime.applicationService.updateFieldValue('name', {
      raw: 'Bobby',
      display: 'Bobby',
      meta: {}
    })

    const submitData = formRuntime.domainService.getSubmitData(editRequest)
    expect(submitData).toEqual({
      name: 'Bobby',
      status: 'closed'
    })

    const updateCalls: Array<Record<string, any>> = []
    const updateEventBus = createMockEventBus()
    let updatedEventPayload: any = null
    updateEventBus.on(TableEvent.rowUpdated, payload => {
      updatedEventPayload = payload
    })

    const updateService = new TableDomainService(
      {
        loadRows: async () => ({ items: [] }),
        addRow: async () => ({}),
        updateRow: async (request: Record<string, any>) => {
          updateCalls.push(request)
          return {
            id: 2,
            name: 'Bobby',
            status: 'closed'
          }
        },
        deleteRow: async () => {}
      } as any,
      createMutableTableStateManager() as any,
      updateEventBus as any
    )

    const updatedRow = await updateService.updateRow(
      currentFunctionDetail,
      2,
      submitData,
      detailPayload.row
    )

    expect(updateCalls).toEqual([
      {
        functionDetail: currentFunctionDetail,
        id: 2,
        data: {
          name: 'Bobby',
          status: 'closed'
        },
        oldData: detailPayload.row
      }
    ])

    expect(updatedRow).toEqual({
      id: 2,
      name: 'Bobby',
      status: 'closed'
    })

    expect(updatedEventPayload).toEqual({
      id: 2,
      row: {
        id: 2,
        name: 'Bobby',
        status: 'closed'
      }
    })
  })
})
