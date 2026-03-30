import { describe, expect, it } from 'vitest'
import { buildSearchControlStyle, buildSearchRangeFieldStyle } from './searchControlStyle'

describe('searchControlStyle', () => {
  it('forces fallback controls to fill the search grid while preserving non-width styles', () => {
    expect(buildSearchControlStyle({
      width: '200px',
      minWidth: '120px',
      maxWidth: '240px',
      height: '36px'
    })).toEqual({
      height: '36px',
      width: '100%',
      minWidth: 0,
      maxWidth: '100%'
    })
  })

  it('makes range fields stretch evenly inside range layouts', () => {
    expect(buildSearchRangeFieldStyle({
      width: '160px'
    })).toEqual({
      width: '100%',
      minWidth: 0,
      maxWidth: '100%',
      flex: '1 1 0'
    })
  })
})
