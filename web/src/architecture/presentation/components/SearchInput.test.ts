import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SearchComponent, SearchConfig, SearchType } from '@/architecture/runtime/constants/search'
import { WidgetType } from '@/architecture/runtime/constants/widget'

const { hasRequestComponent, createSearchComponentConfigMock } = vi.hoisted(() => ({
  hasRequestComponent: vi.fn(() => false),
  createSearchComponentConfigMock: vi.fn()
}))

const ElInputStub = defineComponent({
  name: 'ElInput',
  props: {
    modelValue: {
      type: [String, Number, Array, Object, Boolean, null],
      default: null
    }
  },
  emits: ['update:modelValue', 'input', 'clear'],
  setup(props, { emit }) {
    return () =>
      h('div', [
        h('input', {
          'data-testid': 'search-input',
          value: props.modelValue ?? '',
          onInput: (event: Event) => {
            const nextValue = (event.target as HTMLInputElement).value
            emit('update:modelValue', nextValue)
            emit('input', nextValue)
          }
        }),
        h(
          'button',
          {
            type: 'button',
            'data-testid': 'clear-input',
            onClick: () => {
              emit('update:modelValue', null)
              emit('clear')
            }
          },
          'clear'
        )
      ])
  }
})

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  props: {
    modelValue: {
      type: [String, Number, Array, Object, Boolean, null],
      default: null
    },
    loading: {
      type: Boolean,
      default: false
    },
    multiple: {
      type: Boolean,
      default: false
    },
    remoteMethod: {
      type: Function,
      default: undefined
    }
  },
  emits: ['update:modelValue', 'change', 'clear', 'visible-change'],
  setup(props, { emit, slots }) {
    const stringify = (value: unknown) => JSON.stringify(value ?? null)

    return () =>
      h(
        'div',
        {
          'data-testid': 'el-select',
          'data-model-value': stringify(props.modelValue),
          'data-loading': String(props.loading),
          'data-multiple': String(props.multiple)
        },
        [
          slots.tag?.(),
          h(
            'button',
            {
              type: 'button',
              'data-testid': 'remote-query',
              onClick: async () => {
                if (props.remoteMethod) {
                  await props.remoteMethod('alice')
                }
              }
            },
            'remote-query'
          ),
          h(
            'button',
            {
              type: 'button',
              'data-testid': 'open-select',
              onClick: () => emit('visible-change', true)
            },
            'open-select'
          ),
          slots.default?.()
        ]
      )
  }
})

const ElTagStub = defineComponent({
  name: 'ElTag',
  props: {
    type: {
      type: String,
      default: undefined
    },
    color: {
      type: String,
      default: undefined
    },
    style: {
      type: [String, Array, Object],
      default: undefined
    },
    closable: {
      type: Boolean,
      default: false
    }
  },
  emits: ['close'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'div',
        {
          class: 'el-tag-stub',
          'data-type': props.type ?? '',
          'data-color': props.color ?? '',
          'data-style': JSON.stringify(props.style ?? attrs.style ?? null),
          'data-closable': String(props.closable)
        },
        [
          slots.default?.(),
          props.closable
            ? h(
                'button',
                {
                  type: 'button',
                  'data-testid': 'close-tag',
                  onClick: () => emit('close')
                },
                'close'
              )
            : null
        ]
      )
  }
})

const ElOptionStub = defineComponent({
  name: 'ElOption',
  props: {
    label: {
      type: [String, Number, Boolean, Object, Array, null],
      default: null
    },
    value: {
      type: [String, Number, Boolean, Object, Array, null],
      default: null
    }
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          class: 'el-option-stub',
          'data-label': String(props.label ?? ''),
          'data-value': String(props.value ?? '')
        },
        slots.default?.()
      )
  }
})

const ElInputNumberStub = defineComponent({
  name: 'ElInputNumber',
  props: {
    modelValue: {
      type: [String, Number, null],
      default: null
    },
    placeholder: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit }) {
    const nextValue = () => (props.placeholder.includes('最大') ? 20 : 10)

    return () =>
      h(
        'div',
        {
          class: 'el-input-number-stub',
          'data-testid': 'input-number',
          'data-placeholder': props.placeholder,
          'data-model-value': String(props.modelValue ?? '')
        },
        [
          h(
            'button',
            {
              type: 'button',
              'data-testid': 'apply-input-number',
              onClick: () => {
                const value = nextValue()
                emit('update:modelValue', value)
                emit('change', value)
              }
            },
            'apply-input-number'
          )
        ]
      )
  }
})

const ElDatePickerStub = defineComponent({
  name: 'ElDatePicker',
  props: {
    modelValue: {
      type: [Array, null],
      default: null
    }
  },
  emits: ['update:modelValue', 'change', 'clear'],
  setup(props, { emit }) {
    const nextRange = ['1711000000000', '1711086400000']

    return () =>
      h(
        'div',
        {
          'data-testid': 'date-picker',
          'data-model-value': JSON.stringify(props.modelValue ?? null)
        },
        [
          h(
            'button',
            {
              type: 'button',
              'data-testid': 'apply-date-range',
              onClick: () => {
                emit('update:modelValue', nextRange)
                emit('change', nextRange)
              }
            },
            'apply-date-range'
          ),
          h(
            'button',
            {
              type: 'button',
              'data-testid': 'clear-date-range',
              onClick: () => {
                emit('update:modelValue', null)
                emit('clear')
              }
            },
            'clear-date-range'
          )
        ]
      )
  }
})

vi.mock('@/architecture/presentation/widgets/registry', () => ({
  widgetComponentFactory: {
    hasRequestComponent
  }
}))

vi.mock('@/architecture/presentation/components/utils/searchComponentConfig', () => ({
  createSearchComponentConfig: (...args: any[]) => createSearchComponentConfigMock(...args)
}))

vi.mock('@/architecture/presentation/widgets/WidgetComponent.vue', () => ({
  default: {
    name: 'WidgetComponent',
    props: {
      searchType: {
        type: String,
        default: ''
      }
    },
    emits: ['update:modelValue'],
    template: `
      <button
        type="button"
        data-testid="widget-search"
        :data-search-type="searchType"
        @click="$emit('update:modelValue', { raw: 'open', display: '开启', meta: { source: 'widget' } })"
      >
        widget-search
      </button>
    `
  }
}))

import SearchInput from './SearchInput.vue'

function createField(widgetType: string, overrides: Record<string, any> = {}) {
  return {
    code: 'status',
    name: '状态',
    widget: {
      type: widgetType
    },
    ...overrides
  } as any
}

function mountSearchInput(options?: {
  field?: any
  searchType?: string
  modelValue?: any
}) {
  return mount(SearchInput, {
    props: {
      field: options?.field ?? createField(WidgetType.INPUT),
      searchType: options?.searchType ?? SearchType.LIKE,
      modelValue: options?.modelValue ?? null
    },
    global: {
      stubs: {
        ElInput: ElInputStub,
        ElSelect: ElSelectStub,
        ElOption: ElOptionStub,
        ElInputNumber: ElInputNumberStub,
        ElDatePicker: ElDatePickerStub,
        ElAvatar: true,
        ElIcon: true,
        ElTag: ElTagStub,
        UserFilterChip: true,
        Close: true
      }
    }
  })
}

describe('SearchInput', () => {
  beforeEach(() => {
    hasRequestComponent.mockReset()
    hasRequestComponent.mockReturnValue(false)
    createSearchComponentConfigMock.mockReset()
    createSearchComponentConfigMock.mockImplementation((field: any) => ({
      component: field.widget?.type === WidgetType.MULTI_SELECT ? SearchComponent.EL_SELECT : SearchComponent.EL_INPUT,
      props: {
        clearable: true,
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH },
        ...(field.widget?.type === WidgetType.MULTI_SELECT ? { multiple: true } : {})
      }
    }))
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
  })

  it('emits debounced updates for plain text search inputs', async () => {
    const wrapper = mountSearchInput({
      field: createField(WidgetType.INPUT),
      searchType: SearchType.LIKE
    })

    await wrapper.get('[data-testid="search-input"]').setValue('alice')

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    vi.advanceTimersByTime(SearchConfig.DEBOUNCE_DELAY)

    expect(wrapper.emitted('update:modelValue')).toEqual([['alice']])
  })

  it('clears immediately without waiting for debounce', async () => {
    const wrapper = mountSearchInput({
      field: createField(WidgetType.INPUT),
      searchType: SearchType.LIKE,
      modelValue: 'seed'
    })

    await wrapper.get('[data-testid="clear-input"]').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([[null]])
  })

  it('uses inline fallback select for static multiselect contains search', async () => {
    hasRequestComponent.mockReturnValue(true)

    const wrapper = mountSearchInput({
      field: createField(WidgetType.MULTI_SELECT),
      searchType: SearchType.CONTAINS
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="el-select"]').exists()).toBe(true)
  })

  it('uses widget renderer for user eq search in search bar', () => {
    hasRequestComponent.mockReturnValue(true)

    const wrapper = mountSearchInput({
      field: createField(WidgetType.USER),
      searchType: SearchType.EQ
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(true)
  })

  it('uses widget renderer for department eq search in search bar', () => {
    hasRequestComponent.mockReturnValue(true)

    const wrapper = mountSearchInput({
      field: createField(WidgetType.DEPARTMENT),
      searchType: SearchType.EQ
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(true)
  })

  it('uses widget renderer for users contains search in search bar', () => {
    hasRequestComponent.mockReturnValue(true)

    const wrapper = mountSearchInput({
      field: createField(WidgetType.USERS),
      searchType: SearchType.CONTAINS
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="el-select"]').exists()).toBe(false)
  })

  it('uses widget renderer for departments contains search in search bar', () => {
    hasRequestComponent.mockReturnValue(true)

    const wrapper = mountSearchInput({
      field: createField(WidgetType.DEPARTMENTS),
      searchType: SearchType.CONTAINS
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="el-select"]').exists()).toBe(false)
  })

  it('uses inline fallback select for static select options even when a widget renderer exists', () => {
    hasRequestComponent.mockReturnValue(true)
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        options: [
          { label: '开启', value: 'open' },
          { label: '关闭', value: 'closed' }
        ]
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.SELECT),
      searchType: SearchType.EQ
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="el-select"]').exists()).toBe(true)
  })

  it('uses inline fallback select for radio in search', () => {
    hasRequestComponent.mockReturnValue(true)
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        multiple: true,
        options: [
          { label: '是', value: '是' },
          { label: '否', value: '否' },
          { label: '不确定', value: '不确定' }
        ]
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.RADIO),
      searchType: SearchType.IN
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="el-select"]').exists()).toBe(true)
    expect(wrapper.findAll('.el-option-stub').map(option => option.attributes('data-value'))).toEqual([
      '是',
      '否',
      '不确定'
    ])
  })

  it('renders high-contrast colored tags for radio in search when options_colors exist', () => {
    hasRequestComponent.mockReturnValue(true)
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        multiple: true,
        options: [
          { label: '是', value: '是' },
          { label: '否', value: '否' },
          { label: '不确定', value: '不确定' }
        ]
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.RADIO, {
        widget: {
          type: WidgetType.RADIO,
          config: {
            options: ['是', '否', '不确定'],
            options_colors: ['F56C6C', '67C23A', 'E6A23C']
          }
        }
      }),
      searchType: SearchType.IN,
      modelValue: '是'
    })

    const tag = wrapper.find('.multiselect-tag.el-tag-stub')

    expect(tag.exists()).toBe(true)
    expect(tag.attributes('data-type')).toBe('')
    expect(tag.attributes('data-style')).toContain('rgb(254, 237, 237)')
    expect(tag.attributes('data-style')).toContain('#B04E4E')
  })

  it('falls back to neutral tag style for unsupported search option colors', () => {
    hasRequestComponent.mockReturnValue(true)
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        multiple: true,
        options: [
          { label: '是', value: '是' },
          { label: '否', value: '否' }
        ]
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.RADIO, {
        widget: {
          type: WidgetType.RADIO,
          config: {
            options: ['是', '否'],
            options_colors: ['not-a-real-color', '12345G']
          }
        }
      }),
      searchType: SearchType.IN,
      modelValue: ['否']
    })

    const tag = wrapper.find('.multiselect-tag.el-tag-stub')
    expect(tag.attributes('data-type')).toBe('')
    expect(tag.attributes('data-style')).toBe('{}')
  })

  it('keeps widget renderer for callback-driven select search', () => {
    hasRequestComponent.mockReturnValue(true)
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        remote: true
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.SELECT, {
        callbacks: ['OnSelectFuzzy']
      }),
      searchType: SearchType.EQ
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="el-select"]').exists()).toBe(false)
  })

  it('keeps widget renderer for callback-driven multiselect search', () => {
    hasRequestComponent.mockReturnValue(true)
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        remote: true,
        multiple: true
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.MULTI_SELECT, {
        callbacks: ['OnSelectFuzzy']
      }),
      searchType: SearchType.CONTAINS
    })

    expect(wrapper.find('[data-testid="widget-search"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="el-select"]').exists()).toBe(false)
  })

  it('preserves widget display metadata when widget search renderer updates the value', async () => {
    hasRequestComponent.mockReturnValue(true)

    const wrapper = mountSearchInput({
      field: createField(WidgetType.SELECT, {
        callbacks: ['OnSelectFuzzy']
      }),
      searchType: SearchType.EQ
    })

    await wrapper.get('[data-testid="widget-search"]').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([[
      {
        raw: 'open',
        display: '开启',
        meta: { source: 'widget' }
      }
    ]])
  })

  it('initializes remote select values from onInitOptions and matches option value types', async () => {
    const onInitOptions = vi.fn().mockResolvedValue([
      { label: 'Alice', value: 1 },
      { label: 'Bob', value: 2 }
    ])

    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        remote: true,
        multiple: true,
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
      },
      onInitOptions
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.SELECT),
      searchType: SearchType.IN,
      modelValue: '1,2'
    })

    await nextTick()
    await flushPromises()
    await nextTick()

    expect(onInitOptions).toHaveBeenCalledWith(['1', '2'])
    expect(wrapper.get('[data-testid="el-select"]').attributes('data-model-value')).toBe('[1,2]')
    expect(wrapper.get('[data-testid="el-select"]').attributes('data-loading')).toBe('false')
    expect(wrapper.findAll('.el-option-stub').map(option => option.attributes('data-value'))).toEqual(['1', '2'])
  })

  it('keeps selected remote options when a new remote query does not include them', async () => {
    const onInitOptions = vi.fn().mockResolvedValue([
      { label: 'Seed', value: 'seed' }
    ])
    const onRemoteMethod = vi.fn().mockResolvedValue([
      { label: 'Alice', value: 'alice' }
    ])

    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_SELECT,
      props: {
        clearable: true,
        remote: true,
        style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
      },
      onInitOptions,
      onRemoteMethod
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.SELECT),
      searchType: SearchType.EQ,
      modelValue: 'seed'
    })

    await nextTick()
    await flushPromises()
    await nextTick()

    expect(wrapper.findAll('.el-option-stub').map(option => option.attributes('data-value'))).toEqual(['seed'])

    await wrapper.get('[data-testid="remote-query"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(onRemoteMethod).toHaveBeenCalledWith('alice')
    expect(wrapper.findAll('.el-option-stub').map(option => option.attributes('data-value'))).toEqual(['alice', 'seed'])
    expect(wrapper.get('[data-testid="el-select"]').attributes('data-model-value')).toBe('"seed"')
  })

  it('hydrates number range values from external state and emits partial/full range objects', async () => {
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.NUMBER_RANGE_INPUT,
      props: {
        minPlaceholder: '最小值',
        maxPlaceholder: '最大值',
        precision: 0,
        step: 1
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.NUMBER),
      searchType: `${SearchType.GTE},${SearchType.LTE}`,
      modelValue: { min: 1, max: 5 }
    })

    await nextTick()

    const rangeInputs = wrapper.findAll('[data-testid="input-number"]')
    expect(rangeInputs).toHaveLength(2)

    const minRangeInput = rangeInputs[0]!
    const maxRangeInput = rangeInputs[1]!
    expect(minRangeInput.attributes('data-model-value')).toBe('1')
    expect(maxRangeInput.attributes('data-model-value')).toBe('5')

    await minRangeInput.get('[data-testid="apply-input-number"]').trigger('click')
    await nextTick()

    const emittedUpdates = wrapper.emitted('update:modelValue') ?? []
    expect(emittedUpdates[0]).toEqual([{ min: 10, max: 5 }])

    await maxRangeInput.get('[data-testid="apply-input-number"]').trigger('click')
    await nextTick()

    expect(emittedUpdates[1]).toEqual([{ min: 10, max: 20 }])
  })

  it('hydrates date range arrays and emits range arrays plus immediate null on clear', async () => {
    createSearchComponentConfigMock.mockImplementation(() => ({
      component: SearchComponent.EL_DATE_PICKER,
      props: {
        type: 'datetimerange',
        clearable: true
      }
    }))

    const wrapper = mountSearchInput({
      field: createField(WidgetType.DATETIME),
      searchType: `${SearchType.GTE},${SearchType.LTE}`,
      modelValue: ['1710000000000', '1710086400000']
    })

    await nextTick()

    expect(wrapper.get('[data-testid="date-picker"]').attributes('data-model-value')).toBe('["1710000000000","1710086400000"]')

    await wrapper.get('[data-testid="apply-date-range"]').trigger('click')
    await nextTick()

    const emittedUpdates = wrapper.emitted('update:modelValue') ?? []
    expect(emittedUpdates[0]).toEqual([['1711000000000', '1711086400000']])

    await wrapper.get('[data-testid="clear-date-range"]').trigger('click')
    await nextTick()

    expect(emittedUpdates[1]).toEqual([null])
  })
})
