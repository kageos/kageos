import { describe, expect, it } from 'vitest'
import { getFieldWidgetOptionColors, getWidgetOptionColors } from './widgetOptionColors'

describe('widgetOptionColors', () => {
  it('reads canonical options_colors from widget config', () => {
    expect(getWidgetOptionColors({
      options_colors: ['67C23A', 'F56C6C']
    })).toEqual(['67C23A', 'F56C6C'])
  })

  it('accepts legacy aliases and comma-separated values', () => {
    expect(getWidgetOptionColors({
      option_colors: '67C23A, F56C6C'
    })).toEqual(['67C23A', 'F56C6C'])

    expect(getWidgetOptionColors({
      options_color: '409EFF'
    })).toEqual(['409EFF'])
  })

  it('falls back to widget-level colors when config is missing them', () => {
    expect(getFieldWidgetOptionColors({
      code: 'status',
      name: '状态',
      widget: {
        type: 'select',
        config: {},
        options_colors: ['E6A23C']
      }
    } as any)).toEqual(['E6A23C'])
  })
})
