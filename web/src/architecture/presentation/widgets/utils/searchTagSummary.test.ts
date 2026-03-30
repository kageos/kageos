import { describe, expect, it } from 'vitest'
import { buildSearchTagSummary } from './searchTagSummary'

describe('searchTagSummary', () => {
  it('keeps only the configured number of visible values', () => {
    expect(buildSearchTagSummary(['A', 'B', 'C'], 1)).toEqual({
      visibleValues: ['A'],
      hiddenCount: 2
    })
  })

  it('returns an empty summary for empty input', () => {
    expect(buildSearchTagSummary(null, 1)).toEqual({
      visibleValues: [],
      hiddenCount: 0
    })
  })
})
