import { describe, expect, it } from 'vitest'
import {
  buildChartEChartsOption,
  buildChartMetadataPreview,
  formatChartMetadataValue,
  type RenderableChart,
} from './chartRendererOption'

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
  yAxis?: { scale?: boolean; axisLabel?: { formatter?: (value: number) => string } }
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

  it('formats duration values from y_axis.value_format', () => {
    const chart: RenderableChart = {
      chart_type: 'line',
      x_axis: labels(2),
      y_axis: { value_format: 'duration_ms' },
      series: [
        { name: '平均耗时', data: [1200, 3500] },
      ],
    }

    const option = buildChartEChartsOption(chart) as CartesianOption
    const tooltipHtml = option.tooltip?.formatter?.([
      {
        axisValue: '2026-06-26 01:23:45',
        seriesName: '平均耗时',
        value: 1200,
        color: '#5470c6',
      },
    ])

    expect(option.yAxis?.axisLabel?.formatter?.(950)).toBe('950ms')
    expect(option.yAxis?.axisLabel?.formatter?.(1200)).toBe('1.2s')
    expect(option.yAxis?.axisLabel?.formatter?.(65000)).toBe('1.08min')
    expect(tooltipHtml).toContain('平均耗时')
    expect(tooltipHtml).toContain('1.2s')
  })

  it('falls back when y_axis.value_format is missing or unsupported', () => {
    const chart: RenderableChart = {
      chart_type: 'line',
      x_axis: labels(2),
      y_axis: { value_format: 'duration_seconds' },
      series: [
        { name: '平均耗时', data: [1200, 3500] },
      ],
    }

    const option = buildChartEChartsOption(chart) as CartesianOption
    const tooltipHtml = option.tooltip?.formatter?.([
      {
        axisValue: '2026-06-26 01:23:45',
        seriesName: '平均耗时',
        value: 1200,
        color: '#5470c6',
      },
    ])

    expect(option.yAxis?.axisLabel?.formatter?.(1200)).toBe('1.2K')
    expect(tooltipHtml).toContain('1200')
    expect(tooltipHtml).not.toContain('1.2s')

    const malformedChart = {
      ...chart,
      y_axis: 'duration_ms',
    } as unknown as RenderableChart
    const malformedOption = buildChartEChartsOption(malformedChart) as CartesianOption
    expect(malformedOption.yAxis?.axisLabel?.formatter?.(1200)).toBe('1.2K')
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

  it('formats metadata values and compacts long previews', () => {
    const longValue = 'GitHub登录页 (#2)、API 404场景 (#6)、API 延迟场景 (#7)、API 500场景 (#10)、天翼云服务器 (#1)'
    const preview = buildChartMetadataPreview(longValue, 32)

    expect(formatChartMetadataValue(['展示目标', 10])).toBe('["展示目标",10]')
    expect(preview.truncated).toBe(true)
    expect(preview.text.endsWith('...')).toBe(true)
    expect(preview.text).not.toContain('API 500')
    expect(buildChartMetadataPreview('最近24小时', 32)).toEqual({
      text: '最近24小时',
      truncated: false,
    })
  })
})
