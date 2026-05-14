import { describe, expect, it } from 'vitest'
import { buildSearchParamsString } from './searchParams'

describe('searchParams', () => {
  it('builds direct request field params', () => {
    expect(
      buildSearchParamsString(
        {
          title: '春风',
          status: '处理中',
          empty: ''
        },
        [
          { code: 'title' } as any,
          { code: 'status' } as any,
          { code: 'empty' } as any
        ]
      )
    ).toEqual({
      title: '春风',
      status: '处理中'
    })
  })

  it('serializes array field values with comma separators', () => {
    expect(
      buildSearchParamsString(
        { departments: ['研发', '产品'] },
        [{ code: 'departments' } as any]
      )
    ).toEqual({
      departments: '研发,产品'
    })
  })
})
