import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SelectWidgetDialogTrigger from './SelectWidgetDialogTrigger.vue'

describe('SelectWidgetDialogTrigger', () => {
  it('renders placeholder styling when there is no value', () => {
    const wrapper = mount(SelectWidgetDialogTrigger, {
      props: {
        displayValue: '',
        fallbackLabel: '请选择状态',
        hasValue: false,
        showClear: false
      },
      global: {
        stubs: {
          ElIcon: true
        }
      }
    })

    expect(wrapper.find('.select-placeholder').exists()).toBe(true)
    expect(wrapper.find('.select-label').exists()).toBe(false)
    expect(wrapper.text()).toContain('请选择状态')
  })

  it('renders selected label styling when there is a value', () => {
    const wrapper = mount(SelectWidgetDialogTrigger, {
      props: {
        displayValue: '已选择',
        fallbackLabel: '请选择状态',
        hasValue: true,
        showClear: false
      },
      global: {
        stubs: {
          ElIcon: true
        }
      }
    })

    expect(wrapper.find('.select-label').exists()).toBe(true)
    expect(wrapper.find('.select-placeholder').exists()).toBe(false)
    expect(wrapper.text()).toContain('已选择')
  })
})
