import { describe, expect, it } from 'vitest'
import { buildSliderMarks } from './sliderMarks'

describe('buildSliderMarks', () => {
  it('keeps dense marks only when the total count stays small', () => {
    expect(buildSliderMarks({
      min: 0,
      max: 30,
      step: 10,
      unit: '%'
    })).toEqual({
      0: '0%',
      10: '10%',
      20: '20%',
      30: '30%'
    })
  })

  it('reduces dense percentage sliders to sparse key marks', () => {
    expect(buildSliderMarks({
      min: 0,
      max: 100,
      step: 5,
      unit: '%'
    })).toEqual({
      0: '0%',
      25: '25%',
      50: '50%',
      75: '75%',
      100: '100%'
    })
  })

  it('snaps sparse marks to the nearest valid step', () => {
    expect(buildSliderMarks({
      min: 0,
      max: 95,
      step: 10
    })).toEqual({
      0: '0',
      20: '20',
      50: '50',
      70: '70',
      95: '95'
    })
  })
})
