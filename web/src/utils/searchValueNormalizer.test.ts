import { describe, expect, it } from 'vitest'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import { denormalizeSearchValue, normalizeSearchValue } from './searchValueNormalizer'

describe('searchValueNormalizer', () => {
  it('normalizes switch values into string flags', () => {
    expect(normalizeSearchValue(true, { widgetType: WidgetType.SWITCH })).toBe('true')
    expect(normalizeSearchValue(false, { widgetType: WidgetType.SWITCH })).toBe('false')
  })

  it('normalizes multiselect contains search into comma string', () => {
    expect(normalizeSearchValue(['a', 'b'], {
      widgetType: WidgetType.MULTI_SELECT,
      searchType: 'contains'
    })).toBe('a,b')
  })

  it('denormalizes multiselect contains search into arrays', () => {
    expect(denormalizeSearchValue('a,b', {
      widgetType: WidgetType.MULTI_SELECT,
      searchType: 'contains'
    })).toEqual(['a', 'b'])
  })

  it('keeps select in search values as arrays', () => {
    const value = ['1', '2']

    expect(normalizeSearchValue(value, {
      widgetType: WidgetType.SELECT,
      searchType: 'in'
    })).toEqual(value)

    expect(denormalizeSearchValue(value, {
      widgetType: WidgetType.SELECT,
      searchType: 'in'
    })).toEqual(value)
  })
})
