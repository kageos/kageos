import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { IApiClient } from '@/architecture/domain/interfaces/IApiClient'
import type { IEventBus } from '@/architecture/domain/interfaces/IEventBus'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/architecture/domain/types'
import {
  buildInitialDataFromFormDataStore,
  createFormViewRuntime,
  syncFormDataStoreToStateManager
} from './formViewRuntime'

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

const apiClientStub = {
  get: async () => ({}),
  post: async () => ({}),
  put: async () => ({}),
  delete: async () => ({})
} as IApiClient

const fields: FieldConfig[] = [
  {
    code: 'name',
    name: '姓名',
    widget: {
      type: 'input',
      config: {
        default: 'DEFAULT'
      }
    },
    data: {
      type: 'string'
    }
  },
  {
    code: 'title',
    name: '标题',
    widget: {
      type: 'input',
      config: {}
    },
    data: {
      type: 'string'
    }
  }
]

const submitFunctionDetail: FunctionDetail = {
  method: 'POST',
  router: '/test/form-submit',
  request: fields
}

describe('formViewRuntime', () => {
  beforeEach(() => {
    // 每个 runtime 都有独立 pinia，不需要额外全局清理
  })

  it('keeps isolated submit data across multiple runtimes', () => {
    const runtimeA = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: apiClientStub
    })
    const runtimeB = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: apiClientStub
    })

    runtimeA.applicationService.initializeForm(fields, { name: 'Alice' }, true)
    runtimeB.applicationService.initializeForm(fields, { name: 'Bob' }, true)

    expect(runtimeA.domainService.getSubmitData(fields)).toEqual({
      name: 'Alice',
      title: ''
    })
    expect(runtimeB.domainService.getSubmitData(fields)).toEqual({
      name: 'Bob',
      title: ''
    })

    runtimeA.formDataStore.clear()

    expect(runtimeB.domainService.getSubmitData(fields)).toEqual({
      name: 'Bob',
      title: ''
    })
  })

  it('preserves empty initialData in update mode instead of falling back to defaults', () => {
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: apiClientStub
    })

    runtime.applicationService.initializeForm(fields, { name: '' }, true)

    expect(runtime.domainService.getSubmitData(fields)).toEqual({
      name: '',
      title: ''
    })
  })

  it('omits excluded fields from submit data when the exclusion condition is active', () => {
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: apiClientStub
    })

    const exclusionFields: FieldConfig[] = [
      {
        code: 'invoice_type',
        field_name: 'InvoiceType',
        name: '发票类型',
        widget: { type: 'input', config: {} },
        data: { type: 'string' }
      },
      {
        code: 'tax_no',
        name: '税号',
        widget: { type: 'input', config: {} },
        data: { type: 'string' },
        validation: 'excluded_unless=InvoiceType company'
      }
    ]

    runtime.applicationService.initializeForm(exclusionFields, {
      invoice_type: 'personal',
      tax_no: 'T-001'
    }, true)

    expect(runtime.domainService.getSubmitData(exclusionFields)).toEqual({
      invoice_type: 'personal'
    })
  })

  it('builds raw initialData from the scoped form store', () => {
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: apiClientStub
    })

    runtime.formDataStore.setValue('name', {
      raw: 'Alice',
      display: 'Alice Display',
      meta: { source: 'test' }
    })
    runtime.formDataStore.setValue('title', {
      raw: 'Engineer',
      display: 'Engineer',
      meta: {}
    })

    expect(buildInitialDataFromFormDataStore({
      fields,
      formDataStore: runtime.formDataStore
    })).toEqual({
      name: 'Alice',
      title: 'Engineer'
    })
  })

  it('syncs scoped form store values into the state manager without losing display metadata', () => {
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: apiClientStub
    })

    const fieldValue: FieldValue = {
      raw: 'Alice',
      display: 'Alice Display',
      meta: { source: 'test' }
    }

    runtime.formDataStore.setValue('name', fieldValue)

    syncFormDataStoreToStateManager({
      fields,
      formDataStore: runtime.formDataStore,
      stateManager: runtime.stateManager
    })

    expect(runtime.stateManager.getState().data.get('name')).toEqual(fieldValue)
    expect(runtime.stateManager.getState().data.get('title')).toEqual({
      raw: null,
      display: '',
      meta: {}
    })
  })

  it('rejects with backend msg when api client returns a raw business error envelope', async () => {
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: {
        get: async () => ({}),
        post: async () => ({
          code: -1,
          data: null,
          msg: '您不在本次活动参与名单中，无法参与抽奖'
        }),
        put: async () => ({}),
        delete: async () => ({})
      } as IApiClient
    })

    runtime.applicationService.initializeForm(fields, { name: 'Alice' }, true)

    await expect(runtime.applicationService.submitForm(submitFunctionDetail)).rejects.toMatchObject({
      message: '您不在本次活动参与名单中，无法参与抽奖',
      response: {
        data: {
          msg: '您不在本次活动参与名单中，无法参与抽奖'
        }
      }
    })
  })

  it('blocks submit when conditional required fields fail validation', async () => {
    const post = vi.fn(async () => ({}))
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: {
        get: async () => ({}),
        post,
        put: async () => ({}),
        delete: async () => ({})
      } as IApiClient
    })

    const conditionalFields: FieldConfig[] = [
      {
        code: 'member_type',
        field_name: 'MemberType',
        name: '会员类型',
        widget: { type: 'input', config: {} },
        data: { type: 'string' }
      },
      {
        code: 'card_no',
        name: '卡号',
        widget: { type: 'input', config: {} },
        data: { type: 'string' },
        validation: 'required_if=MemberType vip'
      }
    ]

    runtime.applicationService.initializeForm(conditionalFields, {
      member_type: 'vip',
      card_no: ''
    }, true)

    await expect(runtime.applicationService.submitForm({
      method: 'POST',
      router: '/test/conditional-submit',
      request: conditionalFields
    })).rejects.toThrow('请先修正表单校验错误')

    expect(post).not.toHaveBeenCalled()
    expect(runtime.domainService.getFieldError('card_no')[0]?.message).toBe('卡号必填')
  })

  it('submits sanitized payload with excluded fields removed', async () => {
    const post = vi.fn(async (_url: string, payload: Record<string, any>) => payload)
    const runtime = createFormViewRuntime({
      eventBus: createMockEventBus(),
      apiClient: {
        get: async () => ({}),
        post,
        put: async () => ({}),
        delete: async () => ({})
      } as IApiClient
    })

    const exclusionFields: FieldConfig[] = [
      {
        code: 'invoice_type',
        field_name: 'InvoiceType',
        name: '发票类型',
        widget: { type: 'input', config: {} },
        data: { type: 'string' }
      },
      {
        code: 'tax_no',
        name: '税号',
        widget: { type: 'input', config: {} },
        data: { type: 'string' },
        validation: 'excluded_unless=InvoiceType company'
      }
    ]

    runtime.applicationService.initializeForm(exclusionFields, {
      invoice_type: 'personal',
      tax_no: 'T-001'
    }, true)

    const response = await runtime.applicationService.submitForm({
      method: 'POST',
      router: '/test/excluded-submit',
      request: exclusionFields
    })

    expect(post).toHaveBeenCalledWith(
      '/workspace/api/v1/form/submit/test/excluded-submit',
      { invoice_type: 'personal' }
    )
    expect(response).toEqual({ invoice_type: 'personal' })
  })
})
