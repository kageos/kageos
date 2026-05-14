import { describe, expect, it } from 'vitest'
import { resolveSearchFieldLayoutClass } from './searchFieldLayout'
import { WidgetType } from '@/architecture/domain/constants/widget'

describe('searchFieldLayout', () => {
  it('keeps datetime fields on default layout', () => {
    expect(
      resolveSearchFieldLayoutClass({
        code: 'created_at',
        name: '创建时间',
        widget: { type: WidgetType.DATETIME }
      } as any)
    ).toBe('')
  })

  it('keeps normal text search on default layout', () => {
    expect(
      resolveSearchFieldLayoutClass({
        code: 'title',
        name: '标题',
        widget: { type: WidgetType.INPUT }
      } as any)
    ).toBe('')
  })
})
