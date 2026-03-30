import { describe, expect, it } from 'vitest'
import { resolveSearchFieldLayoutClass, SearchFieldLayoutClass } from './searchFieldLayout'
import { WidgetType } from '@/core/constants/widget'

describe('searchFieldLayout', () => {
  it('marks timestamp gte/lte fields as wide', () => {
    expect(
      resolveSearchFieldLayoutClass({
        code: 'created_at',
        name: '创建时间',
        search: 'gte,lte',
        widget: { type: WidgetType.TIMESTAMP }
      } as any)
    ).toBe(SearchFieldLayoutClass.WIDE)
  })

  it('keeps normal text search on default layout', () => {
    expect(
      resolveSearchFieldLayoutClass({
        code: 'title',
        name: '标题',
        search: 'like',
        widget: { type: WidgetType.INPUT }
      } as any)
    ).toBe('')
  })
})
