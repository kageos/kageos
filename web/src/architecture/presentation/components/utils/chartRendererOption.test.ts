import { describe, expect, it } from 'vitest'
import { buildChartEChartsOption, type RenderableChart } from './chartRendererOption'

type CartesianOption = {
  tooltip?: {
    trigger?: string
    triggerOn?: string
    confine?: boolean
    enterable?: boolean
    axisPointer?: { snap?: boolean; type?: string }
    formatter?: (params: unknown) => string
  }
  dataZoom?: Array<{ type?: string; start?: number }>
  grid?: { bottom?: string | number }
  xAxis?: { axisLabel?: { formatter?: (value: string | number) => string } }
  yAxis?: { scale?: boolean }
  series?: Array<{ smooth?: boolean; showSymbol?: boolean; sampling?: string }>
}

const labels = (count: number): string[] => {
  return Array.from({ length: count }, (_, index) => `2026-06-26 00:${String(index).padStart(2, '0')}:00`)
}

describe('chartRendererOption', () => {
  it('adds zoom controls and smooth line defaults for dense line charts', () => {
    const chart: RenderableChart = {
      chart_type: 'line',
      x_axis: labels(200),
      series: [
        { name: '价格', data: Array.from({ length: 200 }, (_, index) => index) },
      ],
    }

    const option = buildChartEChartsOption(chart) as CartesianOption

    expect(option.tooltip?.trigger).toBe('axis')
    expect(option.tooltip?.triggerOn).toBe('mousemove|click')
    expect(option.tooltip?.confine).toBe(true)
    expect(option.tooltip?.enterable).toBe(true)
    expect(option.tooltip?.axisPointer?.type).toBe('cross')
    expect(option.tooltip?.axisPointer?.snap).toBe(true)
    expect(option.dataZoom).toHaveLength(2)
    expect(option.dataZoom?.[0]?.type).toBe('inside')
    expect(option.dataZoom?.[1]?.type).toBe('slider')
    expect(option.dataZoom?.[0]?.start).toBeCloseTo(20)
    expect(option.grid?.bottom).toBe(72)
    expect(option.xAxis?.axisLabel?.formatter?.('2026-06-26 01:23:45')).toBe('06-26\n01:23')
    expect(option.yAxis?.scale).toBe(true)
    expect(option.series?.[0]?.smooth).toBe(true)
    expect(option.series?.[0]?.showSymbol).toBe(false)
  })

  it('formats hover tooltip with full axis label and series values', () => {
    const chart: RenderableChart = {
      chart_type: 'line',
      x_axis: labels(20),
      series: [
        { name: '趋势', data: Array.from({ length: 20 }, (_, index) => index + 0.123) },
      ],
    }

    const option = buildChartEChartsOption(chart) as CartesianOption
    const tooltipHtml = option.tooltip?.formatter?.([
      {
        axisValue: '2026-06-26 01:23:45',
        seriesName: '趋势',
        value: 123.456,
        color: '#5470c6',
      },
    ])

    expect(tooltipHtml).toContain('2026-06-26 01:23:45')
    expect(tooltipHtml).toContain('趋势')
    expect(tooltipHtml).toContain('123.46')
    expect(option.series?.[0]?.showSymbol).toBe(true)
  })

  it('enables line sampling only for very large category series', () => {
    const chart: RenderableChart = {
      chart_type: 'line',
      x_axis: labels(900),
      series: [
        { name: '趋势', data: Array.from({ length: 900 }, (_, index) => index) },
      ],
    }

    const option = buildChartEChartsOption(chart) as CartesianOption

    expect(option.series?.[0]?.sampling).toBe('lttb')
    expect(option.xAxis?.axisLabel?.formatter?.('2026-06-26 01:23:45')).toBe('06-26\n01:23')
  })

  it('keeps small cartesian charts uncluttered', () => {
    const chart: RenderableChart = {
      chart_type: 'bar',
      x_axis: labels(12),
      series: [
        { name: '数量', data: Array.from({ length: 12 }, (_, index) => index) },
      ],
    }

    const option = buildChartEChartsOption(chart) as CartesianOption

    expect(option.tooltip?.trigger).toBe('axis')
    expect(option.dataZoom).toBeUndefined()
    expect(option.grid?.bottom).toBe('3%')
  })
})
