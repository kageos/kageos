import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import DepartmentWidget from './DepartmentWidget.vue'
import { WidgetType } from '@/core/constants/widget'

const DepartmentPickerDialogStub = defineComponent({
  name: 'DepartmentPickerDialog',
  template: '<div data-testid="department-picker-dialog" />'
})

describe('DepartmentWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows highlighted selected tag and clears value in single search mode', async () => {
    const wrapper = mount(DepartmentWidget, {
      props: {
        field: {
          code: 'department',
          name: '部门',
          search: 'eq',
          widget: {
            type: WidgetType.DEPARTMENT,
            config: {}
          }
        } as any,
        value: {
          raw: '/tech/frontend',
          display: '前端研发 (技术中心 / 前端研发)',
          meta: {
            departmentInfo: {
              full_code_path: '/tech/frontend',
              full_name_path: '技术中心 / 前端研发',
              name: '前端研发'
            }
          }
        },
        mode: 'search',
        fieldPath: 'department'
      },
      global: {
        stubs: {
          ElButton: true,
          ElIcon: true,
          DepartmentPickerDialog: DepartmentPickerDialogStub,
          DepartmentDisplay: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('.search-selected-value').exists()).toBe(true)
    expect(wrapper.text()).toContain('前端研发')

    await wrapper.get('.search-tag-remove').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.at(-1)?.[0]).toMatchObject({
      raw: null,
      display: ''
    })
  })
})
