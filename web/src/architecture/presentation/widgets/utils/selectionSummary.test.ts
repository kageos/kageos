import { describe, expect, it } from 'vitest'
import { buildSelectionSummary } from './selectionSummary'

describe('selectionSummary', () => {
  it('keeps only the configured number of visible values', () => {
    expect(buildSelectionSummary(['A', 'B', 'C'], 1)).toEqual({
      visibleValues: ['A'],
      hiddenCount: 2
    })
  })

  it('returns an empty summary for empty input', () => {
    expect(buildSelectionSummary(null, 1)).toEqual({
      visibleValues: [],
      hiddenCount: 0
    })
  })
})
