import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SelectWidget from './SelectWidget.vue'
import { WidgetType } from '@/core/constants/widget'

const { selectFuzzyMock } = vi.hoisted(() => ({
  selectFuzzyMock: vi.fn()
}))

vi.mock('@/api/function', () => ({
  selectFuzzy: (...args: any[]) => selectFuzzyMock(...args)
}))

describe('SelectWidget', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    selectFuzzyMock.mockReset()
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
          display: '',
          meta: {}
        },
        mode: 'search',
        fieldPath: 'job_id',
        functionMethod: 'GET',
        functionRouter: '/jobs'
      },
      global: {
        stubs: {
          FuzzySearchDialog: true,
          FieldStatistics: true,
          ElIcon: true,
          ElTag: true
        }
      }
    })

    await flushPromises()

    expect(selectFuzzyMock).toHaveBeenCalled()
    expect(wrapper.text()).toContain('前端开发工程师 - 技术 (北京, 20000-35000元)')
    expect(wrapper.text()).not.toContain('>1<')
  })
})
