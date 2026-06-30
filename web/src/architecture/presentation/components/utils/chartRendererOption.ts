import type { EChartsCoreOption } from 'echarts/core'
import type { Chart, ChartSeries } from '@/architecture/domain/types/chart'

type ChartMetadataValue = string | number | boolean | null | undefined | Record<string, unknown> | unknown[]
type ChartDataObject = Record<string, unknown> & {
  name?: string
  value?: unknown
}
type ChartDataItem = string | number | boolean | null | ChartDataObject
type SeriesConfig = Record<string, unknown>
type TooltipParam = {
  axisValue?: string | number
  axisValueLabel?: string | number
  name?: string
  value?: unknown
  seriesName?: string
  color?: string
  percent?: string | number
}
type GaugeSeriesOption = SeriesConfig & {
  name: string
  type: 'gauge'
  data: ChartDataObject[]
  detail: SeriesConfig
  axisLabel: SeriesConfig
}

export const CARTESIAN_DATA_ZOOM_THRESHOLD = 80
export const CARTESIAN_DEFAULT_VISIBLE_POINTS = 160
const CHART_METADATA_PREVIEW_MAX_LENGTH = 48
const CHART_METADATA_PREVIEW_SUFFIX = '...'
const DATE_TIME_AXIS_LABEL_RE = /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::\d{2})?)?$/
const VALUE_FORMAT_COMPACT = 'compact'
const VALUE_FORMAT_PLAIN = 'plain'
const VALUE_FORMAT_DURATION_MS = 'duration_ms'
const VALUE_FORMAT_PERCENT = 'percent'
const SUPPORTED_VALUE_FORMATS = new Set([
  VALUE_FORMAT_COMPACT,
  VALUE_FORMAT_PLAIN,
  VALUE_FORMAT_DURATION_MS,
  VALUE_FORMAT_PERCENT,
])

export type RenderableChart = Chart & {
  __placeholder?: boolean
  __placeholderMessage?: string
}

export function formatChartMetadataValue(value: ChartMetadataValue): string {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

export function buildChartMetadataPreview(value: string, maxLength = CHART_METADATA_PREVIEW_MAX_LENGTH): { text: string; truncated: boolean } {
  const normalizedValue = value.replace(/\s+/g, ' ').trim()
  if (normalizedValue.length <= maxLength) {
    return { text: normalizedValue, truncated: false }
  }

  const visibleLength = Math.max(0, maxLength - CHART_METADATA_PREVIEW_SUFFIX.length)
  return {
    text: `${normalizedValue.slice(0, visibleLength).trimEnd()}${CHART_METADATA_PREVIEW_SUFFIX}`,
    truncated: true,
  }
}

export function getChartMetadataSpan(count: number): number {
  void count
  return 6
}

const normalizeValueFormat = (value: unknown): string => {
  if (typeof value !== 'string') return ''
  return value.trim().toLowerCase()
}

const resolveYAxisValueFormat = (chart: RenderableChart): string => {
  const valueFormat = normalizeValueFormat(chart.y_axis?.value_format)
  return SUPPORTED_VALUE_FORMATS.has(valueFormat) ? valueFormat : ''
}

const formatDecimalValue = (value: number, maxFractionDigits = 2): string => {
  if (!Number.isFinite(value)) return String(value)
  if (Number.isInteger(value)) return value.toString()
  return value.toFixed(maxFractionDigits).replace(/\.?0+$/, '')
}

const formatCompactAxisValueLabel = (value: number): string => {
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1)}M`
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}K`
  }
  return formatDecimalValue(value)
}

const formatDurationMSValueLabel = (value: number): string => {
  const absValue = Math.abs(value)
  if (absValue < 1000) {
    return `${formatDecimalValue(value)}ms`
  }
  if (absValue < 60000) {
    return `${formatDecimalValue(value / 1000)}s`
  }
  return `${formatDecimalValue(value / 60000)}min`
}

const formatAxisValueLabel = (value: number, valueFormat?: unknown): string => {
  switch (normalizeValueFormat(valueFormat)) {
    case VALUE_FORMAT_PLAIN:
      return formatDecimalValue(value)
    case VALUE_FORMAT_DURATION_MS:
      return formatDurationMSValueLabel(value)
    case VALUE_FORMAT_PERCENT:
      return `${formatDecimalValue(value)}%`
    case VALUE_FORMAT_COMPACT:
    default:
      return formatCompactAxisValueLabel(value)
  }
}

const escapeTooltipHtml = (value: unknown): string => {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

const isChartDataObject = (value: unknown): value is ChartDataObject => {
  return typeof value === 'object' && value !== null
}

const formatTooltipValue = (value: unknown, valueFormat?: unknown): string => {
  if (typeof value === 'number') {
    switch (normalizeValueFormat(valueFormat)) {
      case VALUE_FORMAT_COMPACT:
        return formatCompactAxisValueLabel(value)
      case VALUE_FORMAT_PLAIN:
        return formatDecimalValue(value)
      case VALUE_FORMAT_DURATION_MS:
        return formatDurationMSValueLabel(value)
      case VALUE_FORMAT_PERCENT:
        return `${formatDecimalValue(value)}%`
      default:
        return value % 1 === 0 ? value.toString() : value.toFixed(2)
    }
  }

  if (Array.isArray(value)) {
    return value.map((item) => formatTooltipValue(item, valueFormat)).join(', ')
  }

  if (isChartDataObject(value) && value.value !== undefined) {
    return formatTooltipValue(value.value, valueFormat)
  }

  return escapeTooltipHtml(value ?? '-')
}

const formatSeriesTooltip = (params: TooltipParam | TooltipParam[] | null | undefined, valueFormat?: unknown): string => {
  if (!params) {
    return '无数据'
  }

  if (Array.isArray(params)) {
    if (params.length === 0) {
      return '无数据'
    }

    const title = params[0]?.axisValueLabel || params[0]?.axisValue || params[0]?.name || ''
    const lines = params.map((param) => {
      const value = formatTooltipValue(param.value, valueFormat)
      const name = escapeTooltipHtml(param.seriesName || param.name || '数值')
      const color = param.color || '#5470c6'

      return `<div style="display: flex; align-items: center; margin-bottom: 4px;">
        <span style="display: inline-block; width: 10px; height: 10px; background-color: ${color}; border-radius: 50%; margin-right: 8px;"></span>
        <span style="flex: 1;">${name}:</span>
        <span style="font-weight: bold; margin-left: 10px;">${value}</span>
      </div>`
    }).join('')

    return `<div style="font-weight: bold; margin-bottom: 8px;">${escapeTooltipHtml(title)}</div>${lines}`
  }

  const value = formatTooltipValue(params.value, valueFormat)
  const title = params.name || params.axisValueLabel || params.axisValue || params.seriesName || ''
  const color = params.color || '#5470c6'
  const name = escapeTooltipHtml(params.seriesName || '数值')

  return `<div style="font-weight: bold; margin-bottom: 8px;">${escapeTooltipHtml(title)}</div>
    <div style="display: flex; align-items: center;">
      <span style="display: inline-block; width: 10px; height: 10px; background-color: ${color}; border-radius: 50%; margin-right: 8px;"></span>
      <span style="flex: 1;">${name}:</span>
      <span style="font-weight: bold; margin-left: 10px;">${value}</span>
    </div>`
}

const STRUCTURAL_SERIES_CONFIG_KEYS = new Set([
  'id',
  'type',
  'name',
  'data',
  'coordinateSystem',
  'xAxisIndex',
  'xAxisId',
  'yAxisIndex',
  'yAxisId',
  'gridIndex',
  'gridId',
  'datasetIndex',
  'datasetId',
  'encode',
  'dimensions',
  'seriesLayoutBy',
  'coordinateSystemUsage',
])

const createCartesianGridOption = (hasDataZoom = false) => ({
  left: '3%',
  right: '4%',
  bottom: hasDataZoom ? 72 : '3%',
  top: '15%',
  outerBoundsContain: 'axisLabel'
})

const shouldUseCartesianDataZoom = (xAxis?: string[]): boolean => {
  return Array.isArray(xAxis) && xAxis.length > CARTESIAN_DATA_ZOOM_THRESHOLD
}

const truncateAxisLabel = (label: string, maxLength: number): string => {
  if (label.length <= maxLength) {
    return label
  }
  return `${label.slice(0, Math.max(0, maxLength - 1))}…`
}

const formatCategoryAxisLabel = (value: string | number, pointCount: number): string => {
  const label = String(value)
  const match = label.match(DATE_TIME_AXIS_LABEL_RE)
  if (match) {
    const [, year, month, day, hour, minute] = match
    if (hour && minute) {
      if (pointCount <= 48) {
        return `${hour}:${minute}`
      }
      if (pointCount <= 240) {
        return `${month}-${day}\n${hour}:${minute}`
      }
      return `${month}-${day}`
    }
    if (pointCount > 90) {
      return `${month}-${day}`
    }
    return year ? `${year}-${month}-${day}` : label
  }

  if (pointCount > 120) {
    return truncateAxisLabel(label, 8)
  }
  if (pointCount > 40) {
    return truncateAxisLabel(label, 12)
  }
  return truncateAxisLabel(label, 18)
}

const createCategoryAxisLabelOption = (xAxis: string[]) => ({
  fontSize: 12,
  color: '#374151',
  hideOverlap: true,
  margin: 10,
  formatter: (value: string | number) => {
    const pointCount = shouldUseCartesianDataZoom(xAxis)
      ? Math.min(xAxis.length, CARTESIAN_DEFAULT_VISIBLE_POINTS)
      : xAxis.length
    return formatCategoryAxisLabel(value, pointCount)
  },
})

const createSeriesId = (chartType: string, series: ChartSeries, index: number): string => {
  return `${chartType}-${index}-${series.name || 'series'}`
}

const getSafeSeriesConfig = (config?: SeriesConfig): SeriesConfig => {
  if (!config) {
    return {}
  }

  return Object.fromEntries(
    Object.entries(config).filter(([key]) => !STRUCTURAL_SERIES_CONFIG_KEYS.has(key))
  )
}

const hasRenderableSeriesData = (series: ChartSeries, chartType: string): boolean => {
  if (!series || !Array.isArray(series.data) || series.data.length === 0) {
    return false
  }

  switch (chartType) {
    case 'pie':
    case 'gauge':
      return series.data.some((item: ChartDataItem) => {
        if (typeof item === 'number') {
          return item !== 0
        }
        if (isChartDataObject(item) && 'value' in item) {
          return Number(item.value) !== 0
        }
        return Boolean(item)
      })
    default:
      return series.data.some((item: ChartDataItem) => Number(item) !== 0 || String(item).trim() !== '')
  }
}

const hasRenderableChartData = (chart: Chart | null | undefined): boolean => {
  if (!chart || !Array.isArray(chart.series) || chart.series.length === 0) {
    return false
  }

  return chart.series.some((series) => hasRenderableSeriesData(series, chart.chart_type))
}

const createZeroValueChart = (base?: Partial<RenderableChart> | null): RenderableChart => {
  const chartType = base?.chart_type || 'bar'
  const placeholderMessage = base?.__placeholderMessage || '当前暂无图表数据，已按 0 值占位显示。'

  if (chartType === 'pie') {
    const pieSeries = Array.isArray(base?.series) && base.series.length > 0
      ? base.series.map((series, index) => {
          const candidateNames = series.data
            ?.map((item: ChartDataItem) => (isChartDataObject(item) ? item.name : null))
            .filter(Boolean) as string[] | undefined

          const pieData = (candidateNames && candidateNames.length > 0 ? candidateNames : ['暂无数据'])
            .map((name) => ({ name, value: 0 }))

          return {
            ...series,
            name: series.name || `系列${index + 1}`,
            data: pieData,
          }
        })
      : [{
          name: '数值',
          data: [{ name: '暂无数据', value: 0 }],
        }]

    return {
      chart_type: 'pie',
      title: base?.title,
      metadata: base?.metadata,
      series: pieSeries,
      widget_type: base?.widget_type,
      data_type: base?.data_type,
      __placeholder: true,
      __placeholderMessage: placeholderMessage,
    }
  }

  if (chartType === 'gauge') {
    const gaugeSeries = Array.isArray(base?.series) && base.series.length > 0
      ? base.series.map((series, index) => ({
          ...series,
          name: series.name || `系列${index + 1}`,
          data: [{ value: 0 }],
        }))
      : [{
          name: '当前值',
          data: [{ value: 0 }],
        }]

    return {
      chart_type: 'gauge',
      title: base?.title,
      metadata: base?.metadata,
      series: gaugeSeries,
      widget_type: base?.widget_type,
      data_type: base?.data_type,
      __placeholder: true,
      __placeholderMessage: placeholderMessage,
    }
  }

  const xAxis = Array.isArray(base?.x_axis) && base.x_axis.length > 0 ? base.x_axis : ['暂无数据']
  const commonSeries = Array.isArray(base?.series) && base.series.length > 0
    ? base.series.map((series, index) => ({
        ...series,
        name: series.name || `系列${index + 1}`,
        data: xAxis.map(() => 0),
      }))
    : [{
        name: '数值',
        data: xAxis.map(() => 0),
      }]

  return {
    chart_type: chartType === 'line' ? 'line' : 'bar',
    title: base?.title,
    x_axis: xAxis,
    metadata: base?.metadata,
    series: commonSeries,
    widget_type: base?.widget_type,
    data_type: base?.data_type,
    __placeholder: true,
    __placeholderMessage: placeholderMessage,
  }
}

export function normalizeRenderableChart(chart: Chart, hasFilters: boolean): RenderableChart {
  if (hasRenderableChartData(chart)) {
    return chart as RenderableChart
  }

  return createZeroValueChart({
    ...chart,
    __placeholderMessage: hasFilters
      ? '当前暂无图表数据，已按 0 值占位显示，可继续调整筛选条件。'
      : '当前暂无图表数据，已按 0 值占位显示。'
  })
}

export function createPendingQueryChart(title: string): RenderableChart {
  return createZeroValueChart({
    chart_type: 'bar',
    title: title || '图表',
    __placeholderMessage: '请先设置筛选条件后查询，当前以 0 值占位显示。'
  })
}

export function buildChartEChartsOption(chart: RenderableChart): EChartsCoreOption {
  if (!chart || !chart.series || chart.series.length === 0) {
    return {}
  }

  const option: EChartsCoreOption = {
    backgroundColor: '#ffffff',
    animation: true,
    animationDuration: 700,
    animationDurationUpdate: 500,
    animationEasing: 'cubicOut',
    animationEasingUpdate: 'cubicOut',
    title: chart.title ? {
      text: chart.title,
      left: 'center',
      top: 10,
      textStyle: {
        fontSize: 18,
        fontWeight: 'bold',
        color: '#1f2937'
      }
    } : undefined,
    legend: {
      data: chart.series.map((series) => series.name),
      top: chart.title ? 40 : 10,
      textStyle: {
        fontSize: 13,
        color: '#374151'
      },
      itemWidth: 20,
      itemHeight: 14
    }
  }

  const xAxis = chart.x_axis || []
  const yAxisValueFormat = resolveYAxisValueFormat(chart)
  const isDenseCartesianData = shouldUseCartesianDataZoom(xAxis)

  switch (chart.chart_type) {
    case 'bar':
      option.grid = [{
        ...createCartesianGridOption(false),
        top: chart.title ? '15%' : '10%',
      }]
      option.tooltip = {
        show: true,
        trigger: 'axis',
        triggerOn: 'mousemove|click',
        confine: true,
        enterable: true,
        axisPointer: {
          type: 'shadow'
        },
        backgroundColor: 'rgba(50, 50, 50, 0.9)',
        borderColor: '#333',
        borderWidth: 1,
        padding: [10, 15],
        textStyle: {
          color: '#fff',
          fontSize: 13,
          lineHeight: 20
        },
        formatter: (params: TooltipParam | TooltipParam[]) => formatSeriesTooltip(params, yAxisValueFormat)
      }
      option.xAxis = [{
        type: 'category',
        data: xAxis,
        axisLine: {
          lineStyle: {
            color: '#d1d5db'
          }
        },
        axisLabel: createCategoryAxisLabelOption(xAxis)
      }]
      option.yAxis = [{
        type: 'value',
        axisLabel: {
          fontSize: 12,
          color: '#374151',
          formatter: (value: number) => formatAxisValueLabel(value, yAxisValueFormat)
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db'
          }
        },
        splitLine: {
          lineStyle: {
            color: '#e5e7eb',
            type: 'dashed'
          }
        }
      }]
      option.series = chart.series.map((series, index) => {
        const safeConfig = getSafeSeriesConfig(series.config)
        return {
          name: series.name,
          id: createSeriesId('bar', series, index),
          type: 'bar',
          ...safeConfig,
          silent: false,
          tooltip: {
            ...((safeConfig.tooltip || {}) as SeriesConfig),
            show: true,
          },
          emphasis: {
            focus: 'series',
            ...((safeConfig.emphasis || {}) as SeriesConfig),
          },
          data: series.data,
        }
      })
      break

    case 'line':
      option.grid = [{
        ...createCartesianGridOption(false),
        top: chart.title ? '15%' : '10%',
      }]
      option.tooltip = {
        show: true,
        trigger: 'axis',
        triggerOn: 'mousemove|click',
        confine: true,
        enterable: true,
        axisPointer: {
          type: 'cross',
          snap: true,
          label: {
            show: true,
            backgroundColor: '#111827'
          },
          lineStyle: {
            color: '#6b7280',
            width: 1,
            type: 'dashed'
          },
          crossStyle: {
            color: '#6b7280',
            width: 1,
            type: 'dashed'
          }
        },
        backgroundColor: 'rgba(50, 50, 50, 0.9)',
        borderColor: '#333',
        borderWidth: 1,
        padding: [10, 15],
        textStyle: {
          color: '#fff',
          fontSize: 13,
          lineHeight: 20
        },
        formatter: (params: TooltipParam | TooltipParam[]) => formatSeriesTooltip(params, yAxisValueFormat)
      }
      if (xAxis.length === 0) {
        return {
          ...option,
          grid: [{
            ...createCartesianGridOption(),
            top: chart.title ? '15%' : '10%',
          }],
          xAxis: [{ type: 'category', data: [], axisLabel: createCategoryAxisLabelOption([]) }],
          yAxis: [{ type: 'value' }],
          series: []
        }
      }
      option.xAxis = [{
        type: 'category',
        data: xAxis,
        axisLabel: createCategoryAxisLabelOption(xAxis),
        axisLine: {
          lineStyle: {
            color: '#d1d5db'
          }
        }
      }]
      option.yAxis = [{
        type: 'value',
        scale: true,
        axisLabel: {
          fontSize: 12,
          color: '#374151',
          formatter: (value: number) => formatAxisValueLabel(value, yAxisValueFormat)
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db'
          }
        },
        splitLine: {
          lineStyle: {
            color: '#e5e7eb',
            type: 'dashed'
          }
        }
      }]
      option.series = chart.series.map((series, index) => ({
        name: series.name,
        id: createSeriesId('line', series, index),
        type: 'line',
        ...getSafeSeriesConfig(series.config),
        smooth: true,
        showSymbol: !isDenseCartesianData,
        symbolSize: isDenseCartesianData ? 5 : 6,
        emphasis: {
          focus: 'series',
          scale: true
        },
        ...(xAxis.length > 800 ? { sampling: 'lttb' } : {}),
        data: series.data || [],
      }))
      break

    case 'pie':
      option.tooltip = {
        trigger: 'item',
        formatter: (params: TooltipParam) => {
          const value = formatTooltipValue(params.value)
          const name = params.name || ''
          const percent = params.percent ? ` (${params.percent}%)` : ''
          return `<div style="font-weight: bold; margin-bottom: 8px;">${name}</div>
            <div style="display: flex; align-items: center;">
              <span style="display: inline-block; width: 10px; height: 10px; background-color: ${params.color || '#5470c6'}; border-radius: 50%; margin-right: 8px;"></span>
              <span style="flex: 1;">数值:</span>
              <span style="font-weight: bold; margin-left: 10px;">${value}${percent}</span>
            </div>`
        }
      }
      option.series = chart.series.map((series) => ({
        name: series.name,
        type: 'pie',
        radius: '50%',
        stillShowZeroSum: true,
        data: series.data.map((item: ChartDataItem) => {
          if (isChartDataObject(item)) {
            return item
          }
          return { value: item }
        }),
        label: {
          fontSize: 13,
          color: '#374151',
          fontWeight: 'normal',
          formatter: '{b}: {c} ({d}%)'
        },
        labelLine: {
          lineStyle: {
            color: '#6b7280'
          }
        },
        emphasis: {
          label: {
            fontSize: 14,
            fontWeight: 'bold'
          },
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        },
        ...getSafeSeriesConfig(series.config),
      }))
      break

    case 'gauge':
      option.tooltip = {
        trigger: 'item',
        backgroundColor: 'rgba(50, 50, 50, 0.9)',
        borderColor: '#333',
        borderWidth: 1,
        padding: [10, 15],
        textStyle: {
          color: '#fff',
          fontSize: 13,
          lineHeight: 20
        },
        formatter: (params: TooltipParam) => {
          const value = formatTooltipValue(params.value)
          const name = params.seriesName || params.name || ''
          return `<div style="font-weight: bold; margin-bottom: 8px;">${name}</div>
            <div style="display: flex; align-items: center;">
              <span style="flex: 1;">当前值:</span>
              <span style="font-weight: bold; margin-left: 10px;">${value}</span>
            </div>`
        }
      }
      option.series = chart.series.map((series) => {
        const safeConfig = getSafeSeriesConfig(series.config)
        let gaugeData: ChartDataObject[] = []
        if (series.data.length > 0) {
          const firstItem = series.data[0]
          if (typeof firstItem === 'number') {
            gaugeData = [{ value: firstItem }]
          } else if (isChartDataObject(firstItem)) {
            gaugeData = [firstItem]
          } else {
            gaugeData = [{ value: parseFloat(String(firstItem)) || 0 }]
          }
        }

        const defaultConfig: GaugeSeriesOption = {
          name: series.name,
          type: 'gauge',
          data: gaugeData,
          detail: {
            fontSize: 16,
            color: '#1f2937',
            fontWeight: 'bold',
            formatter: '{value}%'
          },
          axisLabel: {
            fontSize: 12,
            color: '#374151'
          }
        }

        if (Object.keys(safeConfig).length > 0) {
          Object.assign(defaultConfig, safeConfig)
          if (safeConfig.detail) {
            defaultConfig.detail = {
              ...defaultConfig.detail,
              ...safeConfig.detail
            }
          }
          if (safeConfig.axisLabel) {
            defaultConfig.axisLabel = {
              ...defaultConfig.axisLabel,
              ...safeConfig.axisLabel
            }
          }
        }

        return defaultConfig
      })
      break

    default:
      return {}
  }

  return option
}
