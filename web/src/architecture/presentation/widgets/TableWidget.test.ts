import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import TableWidget from './TableWidget.vue'
import type { FieldConfig } from '@/architecture/domain/types'

const tableField: FieldConfig = {
  code: 'options',
  name: '投票选项统计',
  data: { type: '[]struct' },
  widget: { type: 'table' },
  children: [
    {
      code: 'content',
      name: '选项内容',
      data: { type: 'string' },
      widget: { type: 'input', config: {} },
    },
    {
      code: 'percentage',
      name: '得票率%',
      data: { type: 'float' },
      widget: { type: 'progress', config: { max: 100, unit: '%' } },
    },
  ],
}

describe('TableWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders progress widgets for response table cells', () => {
    const wrapper = mount(TableWidget, {
      props: {
        field: tableField,
        fieldPath: 'options',
        mode: 'response',
        value: {
          raw: [
            { content: 'A', percentage: 62.5 },
            { content: 'B', percentage: 37.5 },
          ],
          display: '',
          meta: {},
        },
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          ElTable: {
            props: ['data'],
            template: '<table><tbody><tr v-for="(row, index) in data" :key="index"><slot :row="row" :$index="index" /></tr></tbody></table>',
          },
          ElTableColumn: {
            props: ['label'],
            template: '<td><slot :row="$parent.$attrs.row" :$index="$parent.$attrs.$index" /></td>',
          },
        },
      },
    })

    expect(wrapper.find('.progress-widget').exists()).toBe(true)
    expect(wrapper.find('.progress-fill').exists()).toBe(true)
  })
})
