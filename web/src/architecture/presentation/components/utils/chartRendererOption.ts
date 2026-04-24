import type { EChartsCoreOption } from 'echarts/core'
import type { Chart, ChartSeries } from '@/core/types/chart'

export type RenderableChart = Chart & {
  __placeholder?: boolean
  __placeholderMessage?: string
}

export function formatChartMetadataValue(value: any): string {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

export function getChartMetadataSpan(_count: number): number {
  return 6
}

const formatAxisValueLabel = (value: number): string => {
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1)}M`
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}K`
  }
  return value.toString()
}

const formatSeriesTooltip = (params: any): string => {
  if (!params) {
    return '无数据'
  }

  if (Array.isArray(params)) {
    if (params.length === 0) {
      return '无数据'
    }

    const title = params[0]?.axisValue || params[0]?.name || ''
    const lines = params.map((param: any) => {
      const value = typeof param.value === 'number'
        ? (param.value % 1 === 0 ? param.value : param.value.toFixed(2))
        : param.value
      const name = param.seriesName || param.name || ''
      const color = param.color || '#5470c6'

      return `<div style="display: flex; align-items: center; margin-bottom: 4px;">
        <span style="display: inline-block; width: 10px; height: 10px; background-color: ${color}; border-radius: 50%; margin-right: 8px;"></span>
        <span style="flex: 1;">${name}:</span>
        <span style="font-weight: bold; margin-left: 10px;">${value}</span>
      </div>`
    }).join('')

    return `<div style="font-weight: bold; margin-bottom: 8px;">${title}</div>${lines}`
  }

  const value = typeof params.value === 'number'
    ? (params.value % 1 === 0 ? params.value : params.value.toFixed(2))
    : params.value
  const title = params.name || params.axisValue || params.seriesName || ''
  const color = params.color || '#5470c6'
  const name = params.seriesName || '数值'

  return `<div style="font-weight: bold; margin-bottom: 8px;">${title}</div>
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
])

const createCartesianGridOption = () => ({
  id: 'main-grid',
  left: '3%',
  right: '4%',
  bottom: '3%',
  top: '15%',
  outerBoundsContain: 'axisLabel'
})

const createSeriesId = (chartType: string, series: ChartSeries, index: number): string => {
  return `${chartType}-${index}-${series.name || 'series'}`
}

const getSafeSeriesConfig = (config?: Record<string, any>): Record<string, any> => {
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
      return series.data.some((item: any) => {
        if (typeof item === 'number') {
          return item !== 0
        }
        if (typeof item === 'object' && item !== null && 'value' in item) {
          return Number((item as any).value) !== 0
        }
        return Boolean(item)
      })
    default:
      return series.data.some((item: any) => Number(item) !== 0 || String(item).trim() !== '')
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
            ?.map((item: any) => (typeof item === 'object' && item !== null ? item.name : null))
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

  switch (chart.chart_type) {
    case 'bar':
      option.grid = {
        ...createCartesianGridOption(),
        top: chart.title ? '15%' : '10%',
      }
      option.tooltip = {
        show: true,
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
        formatter: formatSeriesTooltip
      }
      option.xAxis = {
        id: 'main-x-axis',
        gridIndex: 0,
        type: 'category',
        data: chart.x_axis || [],
        axisLabel: {
          fontSize: 12,
          color: '#374151'
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db'
          }
        }
      }
      option.yAxis = {
        id: 'main-y-axis',
        gridIndex: 0,
        type: 'value',
        axisLabel: {
          fontSize: 12,
          color: '#374151',
          formatter: (value: number) => formatAxisValueLabel(value)
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
      }
      option.series = chart.series.map((series, index) => ({
        name: series.name,
        id: createSeriesId('bar', series, index),
        type: 'bar',
        coordinateSystem: 'cartesian2d',
        xAxisIndex: 0,
        yAxisIndex: 0,
        data: series.data,
        ...getSafeSeriesConfig(series.config),
      }))
      break

    case 'line':
      option.grid = {
        ...createCartesianGridOption(),
        top: chart.title ? '15%' : '10%',
      }
      option.tooltip = {
        show: true,
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
        formatter: formatSeriesTooltip
      }
      if (!chart.x_axis || chart.x_axis.length === 0) {
        return {
          ...option,
          grid: {
            ...createCartesianGridOption(),
            top: chart.title ? '15%' : '10%',
          },
          xAxis: { id: 'main-x-axis', gridIndex: 0, type: 'category', data: [] },
          yAxis: { id: 'main-y-axis', gridIndex: 0, type: 'value' },
          series: []
        }
      }
      option.xAxis = {
        id: 'main-x-axis',
        gridIndex: 0,
        type: 'category',
        data: chart.x_axis,
        axisLabel: {
          fontSize: 12,
          color: '#374151'
        },
        axisLine: {
          lineStyle: {
            color: '#d1d5db'
          }
        }
      }
      option.yAxis = {
        id: 'main-y-axis',
        gridIndex: 0,
        type: 'value',
        axisLabel: {
          fontSize: 12,
          color: '#374151',
          formatter: (value: number) => formatAxisValueLabel(value)
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
      }
      option.series = chart.series.map((series, index) => ({
        name: series.name,
        id: createSeriesId('line', series, index),
        type: 'line',
        coordinateSystem: 'cartesian2d',
        xAxisIndex: 0,
        yAxisIndex: 0,
        data: series.data || [],
        ...getSafeSeriesConfig(series.config),
      }))
      break

    case 'pie':
      option.tooltip = {
        trigger: 'item',
        formatter: (params: any) => {
          const value = typeof params.value === 'number'
            ? (params.value % 1 === 0 ? params.value : params.value.toFixed(2))
            : (typeof params.value === 'object' && params.value?.value !== undefined)
              ? (typeof params.value.value === 'number'
                ? (params.value.value % 1 === 0 ? params.value.value : params.value.value.toFixed(2))
                : params.value.value)
              : params.value
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
        data: series.data.map((item: any) => {
          if (typeof item === 'object' && item !== null) {
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
        formatter: (params: any) => {
          const value = typeof params.value === 'number'
            ? (params.value % 1 === 0 ? params.value : params.value.toFixed(2))
            : (typeof params.value === 'object' && params.value?.value !== undefined)
              ? (typeof params.value.value === 'number'
                ? (params.value.value % 1 === 0 ? params.value.value : params.value.value.toFixed(2))
                : params.value.value)
              : params.value
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
        let gaugeData: any[] = []
        if (series.data.length > 0) {
          const firstItem = series.data[0]
          if (typeof firstItem === 'number') {
            gaugeData = [{ value: firstItem }]
          } else if (typeof firstItem === 'object' && firstItem !== null) {
            gaugeData = [firstItem]
          } else {
            gaugeData = [{ value: parseFloat(String(firstItem)) || 0 }]
          }
        }

        const defaultConfig: any = {
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
