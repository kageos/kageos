import { describe, expect, it } from 'vitest'
import { buildMultiSelectRawValue } from './multiSelectValue'

describe('buildMultiSelectRawValue', () => {
  it('uses comma-separated string for contains search mode', () => {
    expect(buildMultiSelectRawValue({
      values: ['a', 'b'],
      mode: 'search',
      dataType: '[]string',
      searchType: 'contains'
    })).toBe('a,b')
  })

  it('uses array for in search mode', () => {
    expect(buildMultiSelectRawValue({
      values: ['a', 'b'],
      mode: 'search',
      dataType: '[]string',
      searchType: 'in'
    })).toEqual(['a', 'b'])
  })

  it('uses comma-separated string for edit mode string fields', () => {
    expect(buildMultiSelectRawValue({
      values: ['a', 'b'],
      mode: 'edit',
      dataType: 'string'
    })).toBe('a,b')
  })

  it('uses typed arrays for edit mode array fields', () => {
    expect(buildMultiSelectRawValue({
      values: ['1', '2'],
      mode: 'edit',
      dataType: '[]int'
    })).toEqual([1, 2])
  })
})
