import { describe, expect, it } from 'vitest'
import { getOptionLightPalette, getOptionSolidColor, normalizeOptionColor } from './select'

describe('select option colors', () => {
  it('accepts only RRGGBB in options_colors values', () => {
    expect(normalizeOptionColor('409eff')).toBe('#409EFF')
    expect(normalizeOptionColor('409EFF')).toBe('#409EFF')
    expect(normalizeOptionColor('#409EFF')).toBeNull()
    expect(normalizeOptionColor('success')).toBeNull()
    expect(normalizeOptionColor('rgb(64, 158, 255)')).toBeNull()
  })

  it('builds display colors from normalized hex colors only', () => {
    expect(getOptionSolidColor('409EFF')).toBe('#409EFF')
    expect(getOptionLightPalette('409EFF')).toMatchObject({
      backgroundColor: 'rgba(64, 158, 255, 0.12)',
      borderColor: 'rgba(64, 158, 255, 0.28)'
    })
    expect(getOptionLightPalette('success')).toBeNull()
  })
})
