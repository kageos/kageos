import { describe, expect, it } from 'vitest'
import { resolveWidgetSearchType } from './searchType'

describe('resolveWidgetSearchType', () => {
  it('prefers explicit searchType from widget props', () => {
    expect(resolveWidgetSearchType('in', 'eq')).toBe('in')
  })

  it('falls back to field.search when widget props are empty', () => {
    expect(resolveWidgetSearchType('', 'contains')).toBe('contains')
  })

  it('returns empty string when no searchType is available', () => {
    expect(resolveWidgetSearchType(undefined, null)).toBe('')
  })
})
