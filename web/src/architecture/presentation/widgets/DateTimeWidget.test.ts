import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent, h, nextTick } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import DateTimeWidget from './DateTimeWidget.vue'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { useFormDataStore } from '@/architecture/presentation/context/formRuntimeContext'

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
                ? '2026-04-22 00:00:00'
                : '2026-04-21 00:00:00'
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
            emit('update:modelValue', '10:30:00')
            emit('change', '10:30:00')
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
    data: { type: 'string' },
    widget: {
      type: WidgetType.DATETIME,
      config: {
        format: 'YYYY-MM-DD HH:mm:ss'
      }
    }
  }
}

describe('DateTimeWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('emits readable datetime string in search mode', async () => {
    const wrapper = mount(DateTimeWidget, {
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
      raw: '2026-04-21 00:00:00',
      display: '2026-04-21 00:00:00',
      widgetType: WidgetType.DATETIME
    })
  })

  it('uses readable datetime strings for gte/lte range search', async () => {
    const wrapper = mount(DateTimeWidget, {
      props: {
        field: buildField(),
        value: { raw: null, display: '', meta: {} },
        mode: 'search',
        searchType: 'gte,lte',
        fieldPath: 'created_at'
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

    await pickers[0]!.get('[data-testid="emit-date"]').trigger('click')

    const firstEmitted = wrapper.emitted('update:modelValue')
    expect(firstEmitted?.[0]?.[0]).toMatchObject({
      raw: ['2026-04-21 00:00:00', null],
      widgetType: WidgetType.DATETIME
    })

    await wrapper.setProps({
      value: firstEmitted?.[0]?.[0] as any
    })

    await pickers[1]!.get('[data-testid="emit-date"]').trigger('click')

    const secondEmitted = wrapper.emitted('update:modelValue')
    expect(secondEmitted?.[1]?.[0]).toMatchObject({
      raw: ['2026-04-21 00:00:00', '2026-04-22 00:00:00'],
      widgetType: WidgetType.DATETIME
    })
  })

  it('normalizes legacy numeric value to datetime string in edit mode', async () => {
    const localMillis = new Date(2026, 3, 21, 0, 0, 0).getTime()
    const wrapper = mount(DateTimeWidget, {
      props: {
        field: buildField(),
        value: { raw: localMillis, display: String(localMillis), meta: { fromInitialData: true } },
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
      raw: '2026-04-21 00:00:00',
      display: '2026-04-21 00:00:00',
      widgetType: WidgetType.DATETIME,
      meta: {
        fromInitialData: true
      }
    })

    expect(useFormDataStore().getValue('created_at')).toMatchObject({
      raw: '2026-04-21 00:00:00'
    })
  })
})
