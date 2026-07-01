<template>
  <div class="prd-chart-preview" :data-chart-state="renderState">
    <div
      v-if="showFallback"
      class="prd-chart-fallback"
      :class="`is-${fallbackType}`"
      aria-hidden="true"
    >
      <div class="prd-chart-fallback-head">
        <strong>{{ fallbackTitle }}</strong>
        <span>{{ fallbackSubtitle }}</span>
      </div>

      <div v-if="fallbackType === 'pie'" class="prd-chart-fallback-pie-wrap">
        <div class="prd-chart-fallback-pie"></div>
        <div class="prd-chart-fallback-legend">
          <span v-for="item in fallbackData" :key="item.label">
            <i :style="{ backgroundColor: item.color }"></i>
            {{ item.label }}
          </span>
        </div>
      </div>

      <div v-else-if="fallbackType === 'gauge'" class="prd-chart-fallback-gauge-wrap">
        <div class="prd-chart-fallback-gauge">
          <div class="prd-chart-fallback-gauge-value">{{ fallbackGaugeValue }}%</div>
        </div>
        <span>{{ metricNames()[0] }}</span>
      </div>

      <svg
        v-else-if="fallbackType === 'line'"
        class="prd-chart-fallback-line"
        viewBox="0 0 260 150"
        role="img"
      >
        <path d="M18 126H246M18 94H246M18 62H246M18 30H246" class="grid" />
        <polyline :points="fallbackLinePoints" class="line" fill="none" />
        <circle
          v-for="item in fallbackLineDots"
          :key="`${item.x}-${item.y}`"
          :cx="item.x"
          :cy="item.y"
          r="4"
          class="dot"
        />
      </svg>

      <div v-else class="prd-chart-fallback-bars">
        <div
          v-for="item in fallbackData"
          :key="item.label"
          class="prd-chart-fallback-bar"
        >
          <div
            class="prd-chart-fallback-bar-fill"
            :style="{ height: fallbackBarHeight(item.value), backgroundColor: item.color }"
          ></div>
          <span>{{ item.label }}</span>
        </div>
      </div>

      <div v-if="renderError" class="prd-chart-fallback-error">
        {{ renderError }}
      </div>
    </div>

    <div
      ref="chartRef"
      :class="['prd-chart-canvas', { 'is-rendered': rendered && !renderError }]"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { EChartsType, EChartsCoreOption, ResizeOpts, SetOptionOpts } from 'echarts/core'
import { loadEChartsRuntime } from './utils/chartEChartsRuntime'

type RuntimeChartType = 'bar' | 'line' | 'pie' | 'gauge'

interface FallbackDatum {
  label: string
  value: number
  color: string
}

type ChartPreviewRow = Record<string, unknown>

const props = withDefaults(defineProps<{
  active: boolean
  title: string
  chartType: string
  dimension: string
  metrics: string[]
  previewData?: ChartPreviewRow[]
}>(), {
  previewData: () => [],
})

const chartRef = ref<HTMLElement | null>(null)
const rendered = ref(false)
const renderError = ref('')

let chart: EChartsType | null = null
let resizeObserver: ResizeObserver | null = null
let resizeFrame: number | null = null
let retryTimer: number | null = null
let renderVersion = 0
const resizeTimers: number[] = []

const fallbackType = computed<RuntimeChartType>(() => runtimeChartType())
const fallbackTitle = computed(() => props.title || '图表预览')
const fallbackSubtitle = computed(() => {
  return [
    props.chartType || chartTypeLabel(fallbackType.value),
    props.dimension || '维度',
    metricNames().join('、'),
  ].filter(Boolean).join(' / ')
})
const fallbackData = computed<FallbackDatum[]>(() => {
  const dimension = props.dimension || '维度'
  const previewRows = previewDataRows()
  if (previewRows.length > 0) {
    return previewRows.slice(0, 8).map((row, index) => ({
      label: previewRowLabel(row, index),
      value: previewRowNumber(row, metricNames()[0], index),
      color: fallbackPalette[index % fallbackPalette.length] || '#2563eb',
    }))
  }
  return [
    { label: `${dimension} A`, value: 42, color: '#2563eb' },
    { label: `${dimension} B`, value: 28, color: '#16a34a' },
    { label: `${dimension} C`, value: 18, color: '#f59e0b' },
    { label: `${dimension} D`, value: 12, color: '#dc2626' },
  ]
})
const fallbackMax = computed(() => Math.max(...fallbackData.value.map(item => item.value), 1))
const fallbackGaugeValue = computed(() => Math.round(gaugePreviewValue()))
const fallbackLineDots = computed(() => {
  const values = fallbackData.value
  const max = fallbackMax.value
  const width = 224
  const height = 96
  const left = 18
  const top = 30
  const step = values.length > 1 ? width / (values.length - 1) : width
  return values.map((item, index) => ({
    x: Math.round(left + step * index),
    y: Math.round(top + height - (item.value / max) * height),
  }))
})
const fallbackLinePoints = computed(() => fallbackLineDots.value.map(item => `${item.x},${item.y}`).join(' '))
const showFallback = computed(() => !rendered.value || Boolean(renderError.value))
const renderState = computed(() => {
  if (renderError.value) return 'fallback'
  return rendered.value ? 'rendered' : 'pending'
})

function runtimeChartType(): RuntimeChartType {
  const normalized = String(props.chartType || '').toLowerCase()
  if (normalized.includes('line')) return 'line'
  if (normalized.includes('pie')) return 'pie'
  if (normalized.includes('gauge')) return 'gauge'
  return 'bar'
}

function chartTypeLabel(type: RuntimeChartType): string {
  if (type === 'line') return 'LineChart'
  if (type === 'pie') return 'PieChart'
  if (type === 'gauge') return 'GaugeChart'
  return 'BarChart'
}

function metricNames() {
  return props.metrics.length > 0 ? props.metrics : ['数量']
}

const fallbackPalette = ['#2563eb', '#16a34a', '#f59e0b', '#dc2626', '#7c3aed', '#0891b2', '#ea580c', '#4b5563']

function previewDataRows(): ChartPreviewRow[] {
  return Array.isArray(props.previewData)
    ? props.previewData.filter((row): row is ChartPreviewRow => row != null && typeof row === 'object' && !Array.isArray(row))
    : []
}

function previewRowLabel(row: ChartPreviewRow, index: number): string {
  const dimension = props.dimension || '维度'
  for (const key of [dimension, 'name', 'label', '日期', '时间', '分类', '状态']) {
    const value = row[key]
    if (value != null && value !== '') return String(value)
  }
  for (const value of Object.values(row)) {
    if (typeof value === 'string' && value.trim()) return value
  }
  return `${dimension} ${index + 1}`
}

function previewRowNumber(row: ChartPreviewRow, metric: string | undefined, index: number): number {
  const direct = metric ? toFiniteNumber(row[metric]) : null
  if (direct != null) return direct
  for (const key of ['value', '数量', '金额', '销售额', '订单数', '占比', '比例', '分数']) {
    const value = toFiniteNumber(row[key])
    if (value != null) return value
  }
  for (const value of Object.values(row)) {
    const numeric = toFiniteNumber(value)
    if (numeric != null) return numeric
  }
  return Math.max(8, 42 - index * 7)
}

function toFiniteNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const normalized = value.replace(/,/g, '').replace(/%$/, '').trim()
    if (!normalized) return null
    const numeric = Number(normalized)
    return Number.isFinite(numeric) ? numeric : null
  }
  return null
}

function gaugePreviewValue(): number {
  const rows = previewDataRows()
  const first = rows[0]
  const metric = metricNames()[0]
  const raw = first ? previewRowNumber(first, metric, 0) : 72
  return Math.max(0, Math.min(100, raw))
}

function buildOption(): EChartsCoreOption {
  const chartType = runtimeChartType()
  const title = props.title || '图表预览'
  const metrics = metricNames()
  const previewRows = previewDataRows()
  const theme = chartTheme()

  if (chartType === 'pie') {
    return {
      backgroundColor: 'transparent',
      title: { text: title, left: 12, top: 8, textStyle: { color: theme.text, fontSize: 13, fontWeight: 600 } },
      tooltip: { trigger: 'item' },
      legend: { bottom: 0, left: 'center', textStyle: { color: theme.secondary } },
      series: [{
        name: metrics[0],
        type: 'pie',
        radius: ['36%', '66%'],
        center: ['50%', '48%'],
        data: fallbackData.value.map(item => ({
          name: item.label,
          value: item.value,
        })),
      }],
    }
  }

  if (chartType === 'gauge') {
    return {
      backgroundColor: 'transparent',
      title: { text: title, left: 12, top: 8, textStyle: { color: theme.text, fontSize: 13, fontWeight: 600 } },
      series: [{
        name: metrics[0],
        type: 'gauge',
        progress: { show: true, width: 12 },
        axisLine: { lineStyle: { width: 12 } },
        detail: { valueAnimation: true, formatter: `{value}%`, fontSize: 22 },
        data: [{ value: gaugePreviewValue(), name: metrics[0] }],
      }],
    }
  }

  const categories = fallbackData.value.map(item => item.label)
  return {
    backgroundColor: 'transparent',
    title: { text: title, left: 12, top: 8, textStyle: { color: theme.text, fontSize: 13, fontWeight: 600 } },
    tooltip: { trigger: 'axis' },
    legend: { top: 34, right: 16, textStyle: { color: theme.secondary } },
    grid: { left: 40, right: 22, top: 72, bottom: 34, containLabel: true },
    xAxis: {
      type: 'category',
      data: categories,
      axisLine: { lineStyle: { color: theme.border } },
      axisTick: { lineStyle: { color: theme.border } },
      axisLabel: { color: theme.secondary },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: theme.border } },
      splitLine: { lineStyle: { color: theme.grid } },
      axisLabel: { color: theme.secondary },
    },
    series: metrics.map((metric, index) => ({
      name: metric,
      type: chartType,
      smooth: chartType === 'line',
      data: fallbackData.value.map((item, itemIndex) => {
        const row = previewRows[itemIndex]
        if (row) return previewRowNumber(row, metric, itemIndex)
        return item.value + index * 9 + itemIndex * (index + 4)
      }),
    })),
  }
}

function chartTheme() {
  const style = chartRef.value ? getComputedStyle(chartRef.value) : null
  const cssVar = (name: string, fallback: string) => {
    const value = style?.getPropertyValue(name).trim()
    return value || fallback
  }
  return {
    text: cssVar('--el-text-color-primary', '#e8eef7'),
    secondary: cssVar('--el-text-color-secondary', '#94a3b8'),
    border: cssVar('--prd-border', 'rgba(148, 163, 184, 0.28)'),
    grid: cssVar('--prd-border-soft', 'rgba(148, 163, 184, 0.18)'),
  }
}

function fallbackBarHeight(value: number): string {
  const height = Math.max(28, Math.round((value / fallbackMax.value) * 142))
  return `${height}px`
}

function clearRetryTimer() {
  if (retryTimer !== null) {
    window.clearTimeout(retryTimer)
    retryTimer = null
  }
}

function cancelFrame(frame: number) {
  if (typeof window !== 'undefined' && typeof window.cancelAnimationFrame === 'function') {
    window.cancelAnimationFrame(frame)
  } else {
    window.clearTimeout(frame)
  }
}

function requestFrame(callback: FrameRequestCallback): number {
  if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
    return window.requestAnimationFrame(callback)
  }
  return window.setTimeout(() => callback(Date.now()), 16)
}

function clearPendingResize() {
  if (resizeFrame !== null) {
    cancelFrame(resizeFrame)
    resizeFrame = null
  }
  while (resizeTimers.length > 0) {
    const timer = resizeTimers.pop()
    if (timer !== undefined) window.clearTimeout(timer)
  }
}

function chartSize(element = chartRef.value) {
  if (!element) return { width: 0, height: 0 }
  const rect = element.getBoundingClientRect()
  return {
    width: Math.round(rect.width || element.clientWidth || element.offsetWidth || 0),
    height: Math.round(rect.height || element.clientHeight || element.offsetHeight || 0),
  }
}

function resizeChart() {
  if (!chart || !chartRef.value) return
  const size = chartSize()
  if (size.width > 0 && size.height > 0) {
    const resizeOptions: ResizeOpts = { width: size.width, height: size.height }
    chart.resize(resizeOptions)
  } else {
    chart.resize()
  }
}

function scheduleResizePasses() {
  clearPendingResize()
  resizeFrame = requestFrame(() => {
    resizeFrame = null
    resizeChart()
  })
  ;[60, 180, 360].forEach((delay) => {
    resizeTimers.push(window.setTimeout(resizeChart, delay))
  })
}

function waitForFrame(): Promise<void> {
  return new Promise((resolve) => {
    requestFrame(() => resolve())
  })
}

function queueRenderRetry(version: number, attempt: number) {
  clearRetryTimer()
  retryTimer = window.setTimeout(() => {
    retryTimer = null
    void renderChart(attempt + 1, version)
  }, Math.min(320, 60 + attempt * 50))
}

async function renderChart(attempt = 0, version = ++renderVersion) {
  if (!props.active) {
    rendered.value = false
    return
  }

  if (attempt === 0) {
    clearRetryTimer()
    rendered.value = false
    renderError.value = ''
  }

  await nextTick()
  await waitForFrame()
  const element = chartRef.value
  if (!element || version !== renderVersion || !props.active) return

  const size = chartSize(element)
  if ((size.width < 40 || size.height < 160) && attempt < 8) {
    queueRenderRetry(version, attempt)
    return
  }

  if (size.width < 40 || size.height < 160) {
    renderError.value = '图表容器尺寸未就绪，已显示静态预览'
    return
  }

  const chartType = runtimeChartType()
  try {
    const runtime = await loadEChartsRuntime(chartType)
    if (!chartRef.value || version !== renderVersion || !props.active) return

    const nextSize = chartSize(chartRef.value)
    if (!chart) {
      chart = runtime.init(chartRef.value, null, {
        renderer: 'canvas',
        useDirtyRect: false,
        width: nextSize.width,
        height: nextSize.height,
      })
    }

    const setOptionOptions: SetOptionOpts = {
      notMerge: false,
      replaceMerge: ['grid', 'xAxis', 'yAxis', 'series'],
      lazyUpdate: false,
    }
    chart.setOption(buildOption(), setOptionOptions)
    resizeChart()
    rendered.value = true
    renderError.value = ''
    scheduleResizePasses()

    if (!resizeObserver && typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => scheduleResizePasses())
      resizeObserver.observe(chartRef.value)
    }
  } catch (error) {
    rendered.value = false
    renderError.value = 'ECharts 图表加载失败，已显示静态预览'
    console.warn('[PrdChartPreview] render failed', error)
  }
}

watch(
  () => [props.active, props.title, props.chartType, props.dimension, props.metrics.join('|'), JSON.stringify(props.previewData || [])] as const,
  () => { void renderChart() },
  { immediate: true, flush: 'post' }
)

onMounted(() => {
  void renderChart()
})

onBeforeUnmount(() => {
  clearRetryTimer()
  clearPendingResize()
  resizeObserver?.disconnect()
  resizeObserver = null
  chart?.dispose()
  chart = null
})
</script>

<style scoped lang="scss">
.prd-chart-preview {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 300px;
  border: 1px solid var(--prd-border-soft, var(--el-border-color-lighter));
  border-radius: var(--el-border-radius-base);
  background: var(--prd-surface, var(--el-fill-color-blank));
  overflow: hidden;
}

.prd-chart-canvas {
  position: absolute;
  inset: 0;
  z-index: 2;
  width: 100%;
  height: 100%;
  min-height: 300px;
  opacity: 0;
  transition: opacity 0.16s ease;
}

.prd-chart-canvas.is-rendered {
  opacity: 1;
}

.prd-chart-fallback {
  position: absolute;
  inset: 0;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  background: var(--prd-surface, var(--el-fill-color-blank));
  color: var(--el-text-color-primary);
  pointer-events: none;
}

.prd-chart-fallback-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;

  strong,
  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    font-size: 13px;
    font-weight: 700;
    line-height: 1.3;
  }

  span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.3;
  }
}

.prd-chart-fallback-bars {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: 14px;
  padding: 12px 8px 22px;
  border-bottom: 1px solid var(--prd-border-soft, var(--el-border-color-extra-light));
}

.prd-chart-fallback-bar {
  display: flex;
  flex: 1 1 0;
  max-width: 54px;
  min-width: 28px;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  gap: 7px;
  min-height: 0;

  span {
    width: 100%;
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: 11px;
    line-height: 1.2;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.prd-chart-fallback-bar-fill {
  width: 100%;
  min-height: 28px;
  border-radius: 5px 5px 0 0;
}

.prd-chart-fallback-pie-wrap {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(120px, 168px) minmax(0, 1fr);
  align-items: center;
  gap: 20px;
  padding: 6px 10px 12px;
}

.prd-chart-fallback-pie {
  width: 100%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: conic-gradient(#2563eb 0 42%, #16a34a 42% 70%, #f59e0b 70% 88%, #dc2626 88% 100%);
  box-shadow: inset 0 0 0 28px var(--prd-surface, var(--el-fill-color-blank));
}

.prd-chart-fallback-legend {
  display: flex;
  flex-direction: column;
  gap: 9px;
  min-width: 0;

  span {
    display: flex;
    align-items: center;
    gap: 7px;
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-regular);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  i {
    display: inline-block;
    flex: 0 0 auto;
    width: 9px;
    height: 9px;
    border-radius: 50%;
  }
}

.prd-chart-fallback-gauge-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.prd-chart-fallback-gauge {
  position: relative;
  width: min(220px, 76%);
  aspect-ratio: 2 / 1;
  overflow: hidden;
  border-radius: 220px 220px 0 0;
  background: conic-gradient(from 180deg at 50% 100%, #2563eb 0 130deg, var(--el-border-color-lighter) 130deg 180deg);
}

.prd-chart-fallback-gauge::after {
  position: absolute;
  right: 26px;
  bottom: 0;
  left: 26px;
  height: calc(100% - 26px);
  content: '';
  border-radius: 180px 180px 0 0;
  background: var(--prd-surface, var(--el-fill-color-blank));
}

.prd-chart-fallback-gauge-value {
  position: absolute;
  right: 0;
  bottom: 8px;
  left: 0;
  z-index: 1;
  color: var(--el-text-color-primary);
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
  text-align: center;
}

.prd-chart-fallback-line {
  flex: 1;
  min-height: 0;
  width: 100%;

  .grid {
    stroke: var(--prd-border-soft, var(--el-border-color-extra-light));
    stroke-width: 1;
  }

  .line {
    stroke: #2563eb;
    stroke-linecap: round;
    stroke-linejoin: round;
    stroke-width: 4;
  }

  .dot {
    fill: #2563eb;
    stroke: var(--prd-surface, var(--el-fill-color-blank));
    stroke-width: 2;
  }
}

.prd-chart-fallback-error {
  flex: 0 0 auto;
  padding: 6px 8px;
  border: 1px solid var(--el-color-warning-light-5);
  border-radius: var(--el-border-radius-small);
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
  line-height: 1.35;
}

@media (max-width: 640px) {
  .prd-chart-fallback-pie-wrap {
    grid-template-columns: 1fr;
    justify-items: center;
    gap: 12px;
  }

  .prd-chart-fallback-pie {
    max-width: 150px;
  }

  .prd-chart-fallback-legend {
    width: 100%;
  }
}
</style>
