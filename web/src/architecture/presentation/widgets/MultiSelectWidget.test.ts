import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MultiSelectWidget from './MultiSelectWidget.vue'
import { WidgetType } from '@/core/constants/widget'

const { selectFuzzyMock } = vi.hoisted(() => ({
  selectFuzzyMock: vi.fn()
}))

vi.mock('@/api/function', () => ({
  selectFuzzy: (...args: any[]) => selectFuzzyMock(...args)
}))

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue', 'clear'],
  template: '<div data-testid="inline-multiselect" :data-model-value="JSON.stringify(modelValue)"><slot name="tag" /><slot /></div>'
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
    type: {
      type: String,
      default: ''
    },
    color: {
      type: String,
      default: ''
    },
    style: {
      type: [String, Array, Object],
      default: undefined
    }
  },
  setup(props, { attrs, slots }) {
    const styleValue = props.style ?? attrs.style ?? {}
    return () => h('span', {
      class: 'el-tag-stub',
      'data-type': props.type || '',
      'data-color': props.color || '',
      'data-style': typeof styleValue === 'string' ? styleValue : JSON.stringify(styleValue)
    }, slots.default?.())
  }
})

describe('MultiSelectWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    selectFuzzyMock.mockReset()
  })

  it('uses inline multiselect for static options in search mode', async () => {
    const wrapper = mount(MultiSelectWidget, {
      props: {
        field: {
          code: 'status_list',
          name: '状态',
          widget: {
            type: WidgetType.MULTI_SELECT,
            config: {
              options: [
                { label: '开启', value: 'open' },
                { label: '关闭', value: 'closed' }
              ]
            }
          }
        } as any,
        value: {
          raw: ['open'],
          display: '开启',
          meta: {}
        },
        mode: 'search',
        fieldPath: 'status_list'
      },
      global: {
        stubs: {
          ElSelect: ElSelectStub,
          ElOption: ElOptionStub,
          FuzzySearchDialog: FuzzySearchDialogStub,
          ElTag: ElTagStub,
          ElIcon: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('[data-testid="inline-multiselect"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="fuzzy-dialog"]').exists()).toBe(false)
    expect(wrapper.findAll('.el-option-stub')).toHaveLength(2)

    await wrapper.getComponent(ElSelectStub).vm.$emit('update:modelValue', ['open', 'closed'])

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.at(-1)?.[0]).toMatchObject({
      raw: ['open', 'closed'],
      display: '开启, 关闭'
    })
  })

  it('renders high-contrast colored tags for static multiselect selections', async () => {
    const wrapper = mount(MultiSelectWidget, {
      props: {
        field: {
          code: 'status_list',
          name: '状态',
          widget: {
            type: WidgetType.MULTI_SELECT,
            config: {
              options: [
                { label: '开启', value: 'open' },
                { label: '关闭', value: 'closed' }
              ],
              options_colors: ['F56C6C', '67C23A']
            }
          }
        } as any,
        value: {
          raw: ['open'],
          display: '开启',
          meta: {}
        },
        mode: 'edit',
        fieldPath: 'status_list'
      },
      global: {
        stubs: {
          ElSelect: ElSelectStub,
          ElOption: ElOptionStub,
          FuzzySearchDialog: FuzzySearchDialogStub,
          ElTag: ElTagStub,
          ElIcon: true
        }
      }
    })

    await flushPromises()

    const tag = wrapper.find('.el-tag-stub')
    expect(tag.exists()).toBe(true)
    expect(tag.attributes('data-type')).toBe('')
    expect(tag.attributes('data-style')).toContain('backgroundColor')
    expect(tag.attributes('data-style')).toContain('rgb(254, 237, 237)')
    expect(tag.attributes('data-style')).toContain('#B04E4E')
  })

  it('falls back to neutral tag style when options_colors uses invalid color', async () => {
    const wrapper = mount(MultiSelectWidget, {
      props: {
        field: {
          code: 'status_list',
          name: '状态',
          widget: {
            type: WidgetType.MULTI_SELECT,
            config: {
              options: [
                { label: '开启', value: 'open' },
                { label: '关闭', value: 'closed' }
              ],
              options_colors: ['not-a-real-color', '12345G']
            }
          }
        } as any,
        value: {
          raw: ['open', 'closed'],
          display: '开启, 关闭',
          meta: {}
        },
        mode: 'edit',
        fieldPath: 'status_list'
      },
      global: {
        stubs: {
          ElSelect: ElSelectStub,
          ElOption: ElOptionStub,
          FuzzySearchDialog: FuzzySearchDialogStub,
          ElTag: ElTagStub,
          ElIcon: true
        }
      }
    })

    await flushPromises()

    const tags = wrapper.findAll('.el-tag-stub')
    expect(tags[0]?.attributes('data-type')).toBe('')
    expect(tags[0]?.attributes('data-style')).toBe('{}')
    expect(tags[1]?.attributes('data-style')).toBe('{}')
  })

  it('keeps fuzzy dialog for callback-driven multiselect search', async () => {
    selectFuzzyMock.mockResolvedValue({
      error_msg: '',
      items: [
        {
          label: '前端',
          value: 1
        },
        {
          label: '后端',
          value: 2
        }
      ]
    })

    const wrapper = mount(MultiSelectWidget, {
      props: {
        field: {
          code: 'job_ids',
          name: '职位',
          callbacks: ['OnSelectFuzzy'],
          widget: {
            type: WidgetType.MULTI_SELECT,
            config: {}
          },
          data: {
            type: '[]int'
          }
        } as any,
        value: {
          raw: [1],
          display: '前端',
          meta: {}
        },
        mode: 'search',
        fieldPath: 'job_ids',
        functionMethod: 'GET',
        functionRouter: '/jobs'
      },
      global: {
        stubs: {
          ElSelect: ElSelectStub,
          ElOption: ElOptionStub,
          FuzzySearchDialog: FuzzySearchDialogStub,
          ElTag: ElTagStub,
          ElIcon: true
        }
      }
    })

    await flushPromises()

    expect(selectFuzzyMock).toHaveBeenCalled()
    expect(wrapper.find('[data-testid="inline-multiselect"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="fuzzy-dialog"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('前端')
  })
})
