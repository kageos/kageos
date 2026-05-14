import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getRequestComponent, getResponseComponent } = vi.hoisted(() => ({
  getRequestComponent: vi.fn(),
  getResponseComponent: vi.fn()
}))

vi.mock('@/architecture/presentation/widgets/registry', () => ({
  widgetComponentFactory: {
    getRequestComponent,
    getResponseComponent
  }
}))

const RequestRendererStub = defineComponent({
  name: 'RequestRendererStub',
  props: {
    searchType: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h(
        'button',
        {
          'data-testid': 'request-renderer',
          'data-search-type': props.searchType,
          onClick: () =>
            emit('update:modelValue', {
              raw: 'next',
              display: 'Next',
              meta: {}
            })
        },
        props.searchType
      )
  }
})

const ResponseRendererStub = defineComponent({
  name: 'ResponseRendererStub',
  setup() {
    return () => h('div', { 'data-testid': 'response-renderer' }, 'response')
  }
})

import WidgetComponent from './WidgetComponent.vue'

describe('WidgetComponent', () => {
  beforeEach(() => {
    getRequestComponent.mockReset()
    getResponseComponent.mockReset()
    getRequestComponent.mockReturnValue(RequestRendererStub)
    getResponseComponent.mockReturnValue(ResponseRendererStub)
  })

  it('forwards searchType to request widgets and re-emits updates', async () => {
    const wrapper = mount(WidgetComponent, {
      props: {
        field: {
          code: 'status',
          name: '状态',
          widget: { type: 'select' }
        },
        value: {
          raw: null,
          display: '',
          meta: {}
        },
        mode: 'search',
        fieldPath: 'status',
        searchType: 'in'
      }
    })

    expect(getRequestComponent).toHaveBeenCalledWith('select')
    expect(wrapper.get('[data-testid="request-renderer"]').attributes('data-search-type')).toBe('in')

    await wrapper.get('[data-testid="request-renderer"]').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([
      [{ raw: 'next', display: 'Next', meta: {} }]
    ])
  })

  it('uses response renderer in response mode', () => {
    const wrapper = mount(WidgetComponent, {
      props: {
        field: {
          code: 'status',
          name: '状态',
          widget: { type: 'select' }
        },
        value: {
          raw: 'enabled',
          display: '启用',
          meta: {}
        },
        mode: 'response',
        fieldPath: 'status'
      }
    })

    expect(getResponseComponent).toHaveBeenCalledWith('select')
    expect(wrapper.get('[data-testid="response-renderer"]').text()).toBe('response')
  })
})
