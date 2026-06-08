import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SelectWidget from './SelectWidget.vue'
import { WidgetType } from '@/architecture/domain/constants/widget'

const { selectFuzzyMock } = vi.hoisted(() => ({
  selectFuzzyMock: vi.fn()
}))

vi.mock('@/architecture/presentation/context/api/function', () => ({
  selectFuzzy: (...args: any[]) => selectFuzzyMock(...args)
}))

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  props: {
    modelValue: {
      type: null,
      default: null
    }
  },
  emits: ['update:modelValue', 'clear'],
  template: '<div data-testid="inline-select" :data-model-value="JSON.stringify(modelValue)"><slot /></div>'
})

const ElOptionStub = defineComponent({
  name: 'ElOption',
  props: {
    label: {
      type: String,
      default: ''
    },
    value: {
      type: null,
      default: null
    }
  },
  template: '<div class="el-option-stub" :data-label="label" :data-value="String(value)"><slot /></div>'
})

const FuzzySearchDialogStub = defineComponent({
  name: 'FuzzySearchDialog',
  template: '<div data-testid="fuzzy-dialog" />'
})

const ElTagStub = defineComponent({
  name: 'ElTag',
  props: {
    style: {
      type: [String, Array, Object],
      default: undefined
    }
  },
  setup(props, { attrs, slots }) {
    const styleValue = props.style ?? attrs.style ?? {}
    return () => h('span', {
      class: 'el-tag-stub',
      'data-style': typeof styleValue === 'string' ? styleValue : JSON.stringify(styleValue)
    }, slots.default?.())
  }
})

describe('SelectWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    selectFuzzyMock.mockReset()
  })

  it('uses inline select for static options in search mode', async () => {
    const wrapper = mount(SelectWidget, {
      props: {
        field: {
          code: 'status',
          name: '状态',
          widget: {
            type: WidgetType.SELECT,
            config: {
              options: [
                { label: '开启', value: 'open' },
                { label: '关闭', value: 'closed' }
              ]
            }
          }
        } as any,
        value: {
          raw: 'open',
          display: '',
          meta: {}
        },
        mode: 'search',
        fieldPath: 'status'
      },
      global: {
        stubs: {
          ElSelect: ElSelectStub,
          ElOption: ElOptionStub,
          FuzzySearchDialog: FuzzySearchDialogStub,
          FieldStatistics: true,
          ElIcon: true,
          ElTag: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('[data-testid="inline-select"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="fuzzy-dialog"]').exists()).toBe(false)
    expect(wrapper.findAll('.el-option-stub')).toHaveLength(2)

    await wrapper.getComponent(ElSelectStub).vm.$emit('update:modelValue', 'closed')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.at(-1)?.[0]).toMatchObject({
      raw: 'closed',
      display: '关闭'
    })
  })

  it('shows callback option label instead of raw value in search mode', async () => {
    selectFuzzyMock.mockResolvedValue({
      error_msg: '',
      items: [
        {
          label: '前端开发工程师 - 技术 (北京, 20000-35000元)',
          value: 1,
          display_info: {
            部门: '技术'
          }
        }
      ]
    })

    const wrapper = mount(SelectWidget, {
      props: {
        field: {
          code: 'job_id',
          name: '投递职位',
          callbacks: ['OnSelectFuzzy'],
          widget: {
            type: WidgetType.SELECT,
            config: {}
          },
          data: {
            type: 'int'
          }
        } as any,
        value: {
          raw: 1,
          display: '1',
          meta: {}
        },
        mode: 'search',
        fieldPath: 'job_id',
        functionMethod: 'GET',
        functionRouter: '/jobs'
      },
      global: {
        stubs: {
          ElSelect: ElSelectStub,
          ElOption: ElOptionStub,
          FuzzySearchDialog: FuzzySearchDialogStub,
          FieldStatistics: true,
          ElIcon: true,
          ElTag: true
        }
      }
    })

    await flushPromises()

    expect(selectFuzzyMock).toHaveBeenCalled()
    expect(wrapper.find('[data-testid="inline-select"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="fuzzy-dialog"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('前端开发工程师 - 技术 (北京, 20000-35000元)')
    expect(wrapper.text()).not.toContain('>1<')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toMatchObject({
      raw: 1,
      display: '前端开发工程师 - 技术 (北京, 20000-35000元)',
      meta: {
        displayInfo: {
          部门: '技术'
        }
      }
    })
  })

  it('allows clearing callback selections in search mode even when the field is required', async () => {
    selectFuzzyMock.mockResolvedValue({
      error_msg: '',
      items: [
        {
          label: '前端开发工程师 - 技术 (北京, 20000-35000元)',
          value: 1
        }
      ]
    })

    const wrapper = mount(SelectWidget, {
      props: {
        field: {
          code: 'job_id',
          name: '投递职位',
          validation: 'required',
          callbacks: ['OnSelectFuzzy'],
          widget: {
            type: WidgetType.SELECT,
            config: {}
          },
          data: {
            type: 'int'
          }
        } as any,
        value: {
          raw: 1,
          display: '1',
          meta: {}
        },
        mode: 'search',
        fieldPath: 'job_id',
        functionMethod: 'GET',
        functionRouter: '/jobs'
      },
      global: {
        stubs: {
          ElSelect: ElSelectStub,
          ElOption: ElOptionStub,
          FuzzySearchDialog: FuzzySearchDialogStub,
          FieldStatistics: true,
          ElIcon: true,
          ElTag: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('.search-selected-value').exists()).toBe(true)
    expect(wrapper.find('.selected-value-remove').exists()).toBe(true)

    await wrapper.get('.selected-value-remove').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.at(-1)?.[0]).toMatchObject({
      raw: null,
      display: ''
    })
  })

  it('renders options_colors as a colored tag in table-cell mode', async () => {
    const wrapper = mount(SelectWidget, {
      props: {
        field: {
          code: 'status',
          name: '状态',
          widget: {
            type: WidgetType.SELECT,
            config: {
              options: ['开启', '关闭'],
              options_colors: ['67C23A', 'F56C6C']
            }
          }
        } as any,
        value: {
          raw: '关闭',
          display: '',
          meta: {}
        },
        mode: 'table-cell',
        fieldPath: 'status'
      },
      global: {
        stubs: {
          ElTag: ElTagStub
        }
      }
    })

    await flushPromises()

    const tag = wrapper.find('.el-tag-stub')
    expect(tag.exists()).toBe(true)
    expect(tag.text()).toContain('关闭')
    expect(tag.attributes('data-style')).toContain('rgb(254, 237, 237)')
    expect(tag.attributes('data-style')).toContain('#B04E4E')
  })
})
