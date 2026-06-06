import { describe, expect, it, vi } from 'vitest'
import { SelectFuzzyQueryType } from '@/architecture/domain/constants/select'
import { hydrateTableSearchFuzzyDisplay } from './tableSearchFuzzyHydration'

function createFunctionDetail(request: any[]) {
  return {
    method: 'GET',
    router: '/jobs',
    schema: {
      version: 1,
      type: 'table',
      table: {
        request,
        fields: []
      }
    }
  } as any
}

describe('tableSearchFuzzyHydration', () => {
  it('hydrates OnSelectFuzzy select search display from raw URL value', async () => {
    const selectFuzzyRunner = vi.fn().mockResolvedValue({
      error_msg: '',
      items: [
        {
          label: '前端开发工程师',
          value: 1,
          display_info: {
            部门: '技术'
          }
        }
      ]
    })

    const searchForm = {
      job_id: '1'
    }

    const result = await hydrateTableSearchFuzzyDisplay({
      functionDetail: createFunctionDetail([
        {
          code: 'job_id',
          name: '职位',
          callbacks: ['OnSelectFuzzy'],
          widget: { type: 'select' },
          data: { type: 'int' }
        }
      ]),
      searchForm,
      selectFuzzyRunner
    })

    expect(selectFuzzyRunner).toHaveBeenCalledWith('GET', '/jobs', {
      code: 'job_id',
      type: SelectFuzzyQueryType.BY_VALUE,
      value: 1,
      request: {},
      value_type: 'int'
    })
    expect(result.job_id).toMatchObject({
      raw: '1',
      display: '前端开发工程师',
      meta: {
        displayInfo: {
          部门: '技术'
        }
      }
    })
  })

  it('hydrates OnSelectFuzzy multiselect search display from comma URL value', async () => {
    const selectFuzzyRunner = vi.fn().mockResolvedValue({
      error_msg: '',
      items: [
        { label: '前端', value: 1 },
        { label: '后端', value: 2 }
      ]
    })

    const result = await hydrateTableSearchFuzzyDisplay({
      functionDetail: createFunctionDetail([
        {
          code: 'job_ids',
          name: '职位',
          callbacks: ['OnSelectFuzzy'],
          widget: { type: 'multiselect' },
          data: { type: '[]int' }
        }
      ]),
      searchForm: {
        job_ids: '1,2'
      },
      selectFuzzyRunner
    })

    expect(selectFuzzyRunner).toHaveBeenCalledWith('GET', '/jobs', {
      code: 'job_ids',
      type: SelectFuzzyQueryType.BY_VALUES,
      value: [1, 2],
      request: {},
      value_type: '[]int'
    })
    expect(result.job_ids).toMatchObject({
      raw: '1,2',
      display: '前端, 后端'
    })
  })

  it('does not query when display is already hydrated', async () => {
    const selectFuzzyRunner = vi.fn()
    const searchForm = {
      job_id: {
        raw: '1',
        display: '前端开发工程师',
        meta: {}
      }
    }

    const result = await hydrateTableSearchFuzzyDisplay({
      functionDetail: createFunctionDetail([
        {
          code: 'job_id',
          name: '职位',
          callbacks: ['OnSelectFuzzy'],
          widget: { type: 'select' },
          data: { type: 'int' }
        }
      ]),
      searchForm,
      selectFuzzyRunner
    })

    expect(selectFuzzyRunner).not.toHaveBeenCalled()
    expect(result).toBe(searchForm)
  })
})
