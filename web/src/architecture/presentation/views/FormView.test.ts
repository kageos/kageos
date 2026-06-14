import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { computed, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const formGatewaySubmitMock = vi.hoisted(() => vi.fn(async () => ({})))

vi.mock('vue-router', () => ({
  createRouter: () => ({
    beforeEach: vi.fn(),
    afterEach: vi.fn(),
    push: vi.fn(),
    replace: vi.fn()
  }),
  createWebHistory: () => ({}),
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push: vi.fn() })
}))

vi.mock('../../infrastructure/eventBus', () => {
  const listeners = new Map<string, Set<(payload?: any) => void>>()

  return {
    eventBus: {
      emit(event: string, payload?: any) {
        listeners.get(event)?.forEach((handler) => handler(payload))
      },
      on(event: string, handler: (payload?: any) => void) {
        if (!listeners.has(event)) {
          listeners.set(event, new Set())
        }
        listeners.get(event)!.add(handler)
        return () => listeners.get(event)?.delete(handler)
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
    },
    WorkspaceEvent: {},
    FormEvent: {}
  }
})

vi.mock('../../infrastructure/factories', () => ({
  serviceFactory: {
    getWorkspaceStateManager: () => ({
      getCurrentFunction: () => null
    }),
    getWorkspaceDomainService: () => ({
      loadFunction: vi.fn()
    }),
    getFormGateway: () => ({
      submitForm: formGatewaySubmitMock
    })
  }
}))

vi.mock('../composables/useFormDebug', () => ({
  useFormDebug: () => ({
    showDebugDialog: ref(false),
    debugActiveTab: ref('request'),
    debugRequestData: ref(''),
    debugResponseData: ref(''),
    debugRawData: ref(''),
    copyToClipboard: vi.fn()
  })
}))

vi.mock('../composables/useFunctionParamInitialization', () => ({
  useFunctionParamInitialization: () => ({
    initialize: vi.fn(async () => undefined)
  })
}))

vi.mock('../composables/useFormParamURLSync', () => ({
  useFormParamURLSync: () => ({
    watchFormData: vi.fn()
  })
}))

vi.mock('../composables/useFormViewLifecycle', () => ({
  useFormViewLifecycle: () => ({
    resetFormRuntimeState: vi.fn()
  })
}))

import FormView from './FormView.vue'

const sliderField = {
  code: 'progress',
  field_name: 'Progress',
  name: '完成进度',
  data: {
    type: 'int'
  },
  widget: {
    type: 'slider',
    config: {
      default: 50,
      min: 0,
      max: 100,
      step: 5,
      unit: '%'
    }
  }
} as any

function createFormDetail() {
  return {
    id: 1,
    router: '/demo/form',
    template_type: 'form',
    method: 'POST',
    schema: {
      version: 1,
      type: 'form',
      form: {
        request: [sliderField],
        response: []
      }
    }
  } as any
}

describe('FormView', () => {
  it('renders slider fields in the request form', () => {
    const wrapper = mount(FormView, {
      props: {
        functionDetail: createFormDetail()
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          ElAlert: { template: '<div class="el-alert">{{ title }}</div>', props: ['title', 'type'] },
          ElForm: { template: '<form><slot /></form>' },
          ElFormItem: { template: '<div class="form-item"><slot /></div>' },
          ElButton: { template: '<button><slot /></button>' },
          ElDialog: { template: '<div><slot /></div>' },
          ElTabs: { template: '<div><slot /></div>' },
          ElTabPane: { template: '<div><slot /></div>' },
          ElInput: { template: '<input />' },
          ElEmpty: { template: '<div />' },
          ElTag: { template: '<span><slot /></span>' },
          ElIcon: { template: '<i><slot /></i>' }
        }
      }
    })

    expect(wrapper.find('[data-testid="form-request"]').exists()).toBe(true)
    expect(wrapper.find('.slider-widget').exists()).toBe(true)
    expect(wrapper.find('.el-slider').exists()).toBe(true)
  })

  it('shows only the primary submit button', () => {
    const wrapper = mount(FormView, {
      props: {
        functionDetail: createFormDetail()
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          ElAlert: { template: '<div class="el-alert">{{ title }}</div>', props: ['title', 'type'] },
          ElForm: { template: '<form><slot /></form>' },
          ElFormItem: { template: '<div class="form-item"><slot /></div>' },
          ElButton: { template: '<button><slot /></button>' },
          ElDialog: { template: '<div><slot /></div>' },
          ElTabs: { template: '<div><slot /></div>' },
          ElTabPane: { template: '<div><slot /></div>' },
          ElInput: { template: '<input />' },
          ElEmpty: { template: '<div />' },
          ElTag: { template: '<span><slot /></span>' },
          ElIcon: { template: '<i><slot /></i>' }
        }
      }
    })

    const submitButtons = wrapper
      .findAll('button')
      .filter((button) => button.text().includes('提交'))

    expect(submitButtons).toHaveLength(1)
    expect(submitButtons.at(0)?.text()).toBe('提交')
  })

  it('shows compact inline error feedback when submit returns a business error', async () => {
    formGatewaySubmitMock.mockResolvedValueOnce({
      code: -1,
      data: null,
      msg: '余额不足，请充值后重试'
    } as any)

    const wrapper = mount(FormView, {
      props: {
        functionDetail: createFormDetail()
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          ElForm: { template: '<form><slot /></form>' },
          ElFormItem: { template: '<div class="form-item"><slot /></div>' },
          ElButton: { template: '<button><slot /></button>' },
          ElDialog: { template: '<div><slot /></div>' },
          ElTabs: { template: '<div><slot /></div>' },
          ElTabPane: { template: '<div><slot /></div>' },
          ElInput: { template: '<input />' },
          ElEmpty: { template: '<div />' },
          ElTag: { template: '<span><slot /></span>' },
          ElIcon: { template: '<i><slot /></i>' }
        }
      }
    })

    await wrapper.find('[data-testid="form-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('.submit-feedback').text()).toContain('余额不足，请充值后重试')
    expect(wrapper.find('.submit-feedback').text()).not.toContain('提交失败')
  })
})
