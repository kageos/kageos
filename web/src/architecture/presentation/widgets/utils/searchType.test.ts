import { describe, expect, it } from 'vitest'
import { resolveWidgetSearchType } from './searchType'

describe('resolveWidgetSearchType', () => {
  it('prefers explicit searchType from widget props', () => {
    expect(resolveWidgetSearchType('in')).toBe('in')
  })

  it('returns empty string when no searchType is available', () => {
    expect(resolveWidgetSearchType(undefined)).toBe('')
  })
})
