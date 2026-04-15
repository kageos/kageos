import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent, h, nextTick } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import TimestampWidget from './TimestampWidget.vue'
import { WidgetType } from '@/core/constants/widget'
import { useFormDataStore } from '@/core/stores-v2/formData'

const ElDatePickerStub = defineComponent({
  name: 'ElDatePicker',
  props: {
    modelValue: {
      type: [String, Number, Array, Date, null],
      default: null
    },
    placeholder: {
      type: String,
      default: ''
    },
    type: {
      type: String,
      default: 'datetime'
    }
  },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit }) {
    return () =>
      h('div', {
        'data-testid': 'date-picker',
        'data-type': props.type,
        'data-placeholder': props.placeholder
      }, [
        h(
          'button',
          {
            type: 'button',
            'data-testid': 'emit-date',
            onClick: () => {
              const value = props.placeholder.includes('结束')
                ? 1712016000000
                : 1711929600000
              emit('update:modelValue', value)
              emit('change', value)
            }
          },
          'emit-date'
        ),
        h(
          'button',
          {
            type: 'button',
            'data-testid': 'clear-date',
            onClick: () => {
              emit('update:modelValue', '')
              emit('change', null)
            }
          },
          'clear-date'
        )
      ])
  }
})

const ElTimePickerStub = defineComponent({
  name: 'ElTimePicker',
  props: {
    modelValue: {
      type: [String, Number, Array, Date, null],
      default: null
    }
  },
  emits: ['update:modelValue', 'change'],
  setup(_props, { emit }) {
    return () =>
      h(
        'button',
        {
          type: 'button',
          'data-testid': 'time-picker',
          onClick: () => {
            emit('update:modelValue', 1711929600000)
            emit('change', 1711929600000)
          }
        },
        'time-picker'
      )
  }
})

function buildField() {
  return {
    code: 'created_at',
    name: '创建时间',
    search: 'gte,lte',
    data: { type: 'timestamp' },
    widget: {
      type: WidgetType.TIMESTAMP,
      config: {
        format: 'YYYY-MM-DD HH:mm:ss'
      }
    }
  }
}

describe('TimestampWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('emits selected timestamp in search mode', async () => {
    const wrapper = mount(TimestampWidget, {
      props: {
        field: buildField(),
        value: { raw: null, display: '', meta: {} },
        mode: 'search',
        searchType: 'eq',
        fieldPath: 'created_at'
      },
      global: {
        stubs: {
          ElDatePicker: ElDatePickerStub,
          ElTimePicker: ElTimePickerStub
        }
      }
    })

    await wrapper.get('[data-testid="emit-date"]').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.[0]?.[0]).toMatchObject({
      raw: 1711929600000,
      widgetType: WidgetType.TIMESTAMP
    })
  })

  it('uses independent start/end pickers for gte/lte search', async () => {
    const wrapper = mount(TimestampWidget, {
      props: {
        field: buildField(),
        value: { raw: null, display: '', meta: {} },
        mode: 'search',
        searchType: 'gte,lte',
        fieldPath: 'updated_at'
      },
      global: {
        stubs: {
          ElDatePicker: ElDatePickerStub,
          ElTimePicker: ElTimePickerStub
        }
      }
    })

    const pickers = wrapper.findAll('[data-testid="date-picker"]')
    expect(pickers).toHaveLength(2)
    expect(pickers[0]?.attributes('data-type')).toBe('datetime')
    expect(pickers[1]?.attributes('data-type')).toBe('datetime')

    await pickers[0]!.get('[data-testid="emit-date"]').trigger('click')

    const firstEmitted = wrapper.emitted('update:modelValue')
    expect(firstEmitted).toBeTruthy()
    expect(firstEmitted?.[0]?.[0]).toMatchObject({
      raw: [1711929600000, null],
      widgetType: WidgetType.TIMESTAMP
    })

    await wrapper.setProps({
      value: firstEmitted?.[0]?.[0] as any
    })

    await pickers[1]!.get('[data-testid="emit-date"]').trigger('click')

    const secondEmitted = wrapper.emitted('update:modelValue')
    expect(secondEmitted?.[1]?.[0]).toMatchObject({
      raw: [1711929600000, 1712016000000],
      widgetType: WidgetType.TIMESTAMP
    })
  })

  it('normalizes zero timestamp to null in edit mode', async () => {
    const wrapper = mount(TimestampWidget, {
      props: {
        field: buildField(),
        value: { raw: 0, display: '0', meta: { fromInitialData: true } },
        mode: 'edit',
        fieldPath: 'created_at'
      },
      global: {
        stubs: {
          ElDatePicker: ElDatePickerStub,
          ElTimePicker: ElTimePickerStub
        }
      }
    })

    await nextTick()

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.[0]?.[0]).toMatchObject({
      raw: null,
      display: '',
      widgetType: WidgetType.TIMESTAMP,
      meta: {
        fromInitialData: true
      }
    })

    expect(useFormDataStore().getValue('created_at')).toMatchObject({
      raw: null,
      display: ''
    })
  })

  it('emits null when edit mode timestamp is cleared', async () => {
    const wrapper = mount(TimestampWidget, {
      props: {
        field: buildField(),
        value: { raw: 1711929600000, display: '2024-04-01 00:00:00', meta: {} },
        mode: 'edit',
        fieldPath: 'created_at'
      },
      global: {
        stubs: {
          ElDatePicker: ElDatePickerStub,
          ElTimePicker: ElTimePickerStub
        }
      }
    })

    await wrapper.get('[data-testid="clear-date"]').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.[0]?.[0]).toMatchObject({
      raw: null,
      display: '',
      widgetType: WidgetType.TIMESTAMP
    })
  })
})
