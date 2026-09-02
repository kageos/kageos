<template>
  <div class="trend-grid">
    <article v-for="chart in charts" :key="chart.key" class="trend-card" :class="{ 'trend-card-wide': chart.key === 'usage' }">
      <div class="trend-card-header">
        <div>
          <h5>{{ chart.title }}</h5>
          <p>{{ chart.description }}</p>
        </div>
        <span class="trend-range">{{ rangeLabel }}</span>
      </div>
      <div class="trend-chart-shell" @mouseleave="clearHover(chart)">
        <svg
          class="trend-chart"
          viewBox="0 0 880 240"
          role="img"
          tabindex="0"
          :aria-label="chart.title"
          @mousemove="handlePointer($event, chart)"
          @focus="showInitialHover(chart)"
          @keydown.left.prevent="moveHover(chart, -1)"
          @keydown.right.prevent="moveHover(chart, 1)"
          @keydown.esc="clearHover(chart)"
        >
          <g class="trend-grid-lines">
            <template v-for="tick in yTicks(chart)" :key="`${chart.key}-y-${tick.value}`">
              <line x1="66" :y1="tick.y" x2="864" :y2="tick.y" />
              <text x="56" :y="tick.y + 4" text-anchor="end">{{ chart.format(tick.value) }}</text>
            </template>
            <template v-for="tick in xTicks" :key="`${chart.key}-x-${tick.index}`">
              <line :x1="tick.x" y1="16" :x2="tick.x" y2="202" class="vertical-grid-line" />
              <text :x="tick.x" y="226" :text-anchor="tick.anchor">{{ tick.label }}</text>
            </template>
          </g>
          <polyline
            v-for="series in chart.series"
            :key="series.key"
            :points="seriesPoints(series.key, chart.max)"
            class="trend-line"
            :stroke="series.color"
          />
          <g v-if="hoveredIndex(chart) !== null" class="trend-hover-markers">
            <line :x1="pointX(hoveredIndex(chart)!)" y1="16" :x2="pointX(hoveredIndex(chart)!)" y2="202" />
            <circle
              v-for="series in chart.series"
              :key="`${chart.key}-${series.key}-hover`"
              :cx="pointX(hoveredIndex(chart)!)"
              :cy="pointY(series.key, chart.max, hoveredIndex(chart)!)"
              r="5"
              :fill="series.color"
            />
          </g>
        </svg>
        <div
          v-if="hoveredIndex(chart) !== null"
          class="trend-tooltip"
          :class="tooltipSideClass(hoveredIndex(chart)!)"
          :style="{ left: `${pointX(hoveredIndex(chart)!) / 8.8}%` }"
        >
          <strong>{{ formatTooltipTime(props.history[hoveredIndex(chart)!]!.collected_at) }}</strong>
          <span v-for="series in chart.series" :key="`${chart.key}-${series.key}-tooltip`">
            <i :style="{ backgroundColor: series.color }" />
            {{ series.label }}
            <b>{{ tooltipValue(chart, series.key, hoveredIndex(chart)!) }}</b>
          </span>
        </div>
      </div>
      <div class="trend-legend">
        <span v-for="series in chart.series" :key="series.key">
          <i :style="{ backgroundColor: series.color }" />
          {{ series.label }}
          <strong>{{ chart.format(latestValue(series.key)) }}</strong>
          <small>{{ t('systemSettings.resources.peakValue', { value: chart.format(peakValue(series.key)) }) }}</small>
        </span>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SystemResourceHistoryPoint } from '@/architecture/presentation/context/api/system-settings'

type MetricKey = keyof Pick<SystemResourceHistoryPoint,
  'disk_used_percent' | 'memory_used_percent' | 'cpu_used_percent' |
  'network_rx_bytes_per_second' | 'network_tx_bytes_per_second' |
  'disk_read_bytes_per_second' | 'disk_write_bytes_per_second'>

interface TrendSeries {
  key: MetricKey
  label: string
  color: string
}

interface TrendChart {
  key: string
  title: string
  description: string
  max: number
  format: (value: number) => string
  series: TrendSeries[]
}

const props = defineProps<{ history: SystemResourceHistoryPoint[] }>()
const { t } = useI18n()
const plot = { left: 66, right: 864, top: 16, bottom: 202 }
const hoverState = ref<{ chartKey: string; index: number } | null>(null)

const usageSeries = computed<TrendSeries[]>(() => [
  { key: 'cpu_used_percent', label: t('systemSettings.resources.cpuUsage'), color: 'var(--el-color-success)' },
  { key: 'memory_used_percent', label: t('systemSettings.resources.memoryUsage'), color: 'var(--el-color-warning)' },
  { key: 'disk_used_percent', label: t('systemSettings.resources.diskUsage'), color: 'var(--el-color-primary)' },
])

const networkSeries = computed<TrendSeries[]>(() => [
  { key: 'network_rx_bytes_per_second', label: t('systemSettings.resources.networkDownload'), color: 'var(--el-color-primary)' },
  { key: 'network_tx_bytes_per_second', label: t('systemSettings.resources.networkUpload'), color: 'var(--el-color-danger)' },
])

const diskIOSeries = computed<TrendSeries[]>(() => [
  { key: 'disk_read_bytes_per_second', label: t('systemSettings.resources.diskRead'), color: 'var(--el-color-primary)' },
  { key: 'disk_write_bytes_per_second', label: t('systemSettings.resources.diskWrite'), color: 'var(--el-color-warning)' },
])

const charts = computed<TrendChart[]>(() => [
  {
    key: 'usage', title: t('systemSettings.resources.usageTrend'),
    description: t('systemSettings.resources.usageTrendDesc'), max: 100,
    format: (value) => `${value.toFixed(0)}%`, series: usageSeries.value,
  },
  {
    key: 'network', title: t('systemSettings.resources.networkTrend'),
    description: t('systemSettings.resources.networkTrendDesc'), max: niceMax(networkSeries.value),
    format: formatRate, series: networkSeries.value,
  },
  {
    key: 'disk-io', title: t('systemSettings.resources.diskIOTrend'),
    description: t('systemSettings.resources.diskIOTrendDesc'), max: niceMax(diskIOSeries.value),
    format: formatRate, series: diskIOSeries.value,
  },
])

const rangeLabel = computed(() => {
  if (!props.history.length) return '-'
  return `${formatAxisTime(props.history[0]!.collected_at)} — ${formatAxisTime(props.history[props.history.length - 1]!.collected_at)}`
})

const xTicks = computed(() => {
  const last = props.history.length - 1
  const indexes = [...new Set([0, Math.round(last / 4), Math.round(last / 2), Math.round(last * 3 / 4), last])]
  return indexes.map((index, position) => ({
    index,
    x: plot.left + index * (plot.right - plot.left) / Math.max(1, last),
    label: formatAxisTime(props.history[index]!.collected_at),
    anchor: position === 0 ? 'start' : position === indexes.length - 1 ? 'end' : 'middle',
  }))
})

function metricValues(series: TrendSeries[]) {
  return props.history.flatMap(point => series.map(item => Number(point[item.key]) || 0))
}

function niceMax(series: TrendSeries[]) {
  const max = Math.max(1, ...metricValues(series))
  const exponent = 10 ** Math.floor(Math.log10(max))
  const normalized = max / exponent
  const nice = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10
  return nice * exponent
}

function yTicks(chart: TrendChart) {
  return [1, 0.75, 0.5, 0.25, 0].map(ratio => ({
    value: chart.max * ratio,
    y: plot.bottom - ratio * (plot.bottom - plot.top),
  }))
}

function seriesPoints(key: MetricKey, max: number) {
  return props.history.map((point, index) => {
    return `${pointX(index).toFixed(1)},${pointY(key, max, index).toFixed(1)}`
  }).join(' ')
}

function pointX(index: number) {
  return plot.left + index * (plot.right - plot.left) / Math.max(1, props.history.length - 1)
}

function pointY(key: MetricKey, max: number, index: number) {
  const ratio = Math.min(1, Math.max(0, Number(props.history[index]?.[key]) / Math.max(1, max)))
  return plot.bottom - ratio * (plot.bottom - plot.top)
}

function hoveredIndex(chart: TrendChart) {
  return hoverState.value?.chartKey === chart.key ? hoverState.value.index : null
}

function handlePointer(event: MouseEvent, chart: TrendChart) {
  const bounds = (event.currentTarget as SVGElement).getBoundingClientRect()
  const viewX = (event.clientX - bounds.left) * 880 / Math.max(1, bounds.width)
  const ratio = (Math.min(plot.right, Math.max(plot.left, viewX)) - plot.left) / (plot.right - plot.left)
  hoverState.value = { chartKey: chart.key, index: Math.round(ratio * Math.max(0, props.history.length - 1)) }
}

function showInitialHover(chart: TrendChart) {
  if (hoverState.value?.chartKey !== chart.key) {
    hoverState.value = { chartKey: chart.key, index: Math.max(0, props.history.length - 1) }
  }
}

function moveHover(chart: TrendChart, step: number) {
  const current = hoveredIndex(chart) ?? Math.max(0, props.history.length - 1)
  hoverState.value = {
    chartKey: chart.key,
    index: Math.min(Math.max(0, props.history.length - 1), Math.max(0, current + step)),
  }
}

function clearHover(chart: TrendChart) {
  if (hoverState.value?.chartKey === chart.key) hoverState.value = null
}

function tooltipSideClass(index: number) {
  const ratio = index / Math.max(1, props.history.length - 1)
  if (ratio < 0.18) return 'is-left'
  if (ratio > 0.82) return 'is-right'
  return ''
}

function tooltipValue(chart: TrendChart, key: MetricKey, index: number) {
  const value = Number(props.history[index]?.[key]) || 0
  return chart.key === 'usage' ? `${value.toFixed(1)}%` : chart.format(value)
}

function latestValue(key: MetricKey) {
  return Number(props.history[props.history.length - 1]?.[key]) || 0
}

function peakValue(key: MetricKey) {
  return Math.max(0, ...props.history.map(point => Number(point[key]) || 0))
}

function formatAxisTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString([], { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function formatTooltipTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function formatRate(value: number) {
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s']
  let amount = Math.max(0, value)
  let unit = 0
  while (amount >= 1024 && unit < units.length - 1) {
    amount /= 1024
    unit += 1
  }
  return `${amount.toFixed(unit === 0 ? 0 : amount >= 10 ? 0 : 1)} ${units[unit]}`
}
</script>

<style scoped>
.trend-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.trend-card {
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--border-light);
  border-radius: var(--border-radius-lg);
  background: var(--bg-secondary);
}

.trend-card-wide { grid-column: 1 / -1; }
.trend-card-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.trend-card-header h5 { margin: 0; color: var(--text-primary); font-size: 14px; }
.trend-card-header p { margin: 5px 0 0; color: var(--text-secondary); font-size: 12px; }
.trend-range { flex: none; color: var(--text-secondary); font-size: 11px; }
.trend-chart-shell { position: relative; }
.trend-chart { display: block; width: 100%; height: auto; min-height: 210px; margin-top: 10px; }
.trend-chart:focus-visible { border-radius: var(--border-radius-base); outline: 2px solid var(--el-color-primary); outline-offset: 2px; }
.trend-grid-lines line { stroke: var(--border-light); stroke-width: 1; vector-effect: non-scaling-stroke; }
.trend-grid-lines .vertical-grid-line { stroke-dasharray: 3 5; opacity: 0.6; }
.trend-grid-lines text { fill: var(--text-secondary); font-size: 10px; }
.trend-line { fill: none; stroke-width: 2.5; stroke-linecap: round; stroke-linejoin: round; vector-effect: non-scaling-stroke; }
.trend-hover-markers { pointer-events: none; }
.trend-hover-markers line { stroke: var(--text-secondary); stroke-dasharray: 4 4; stroke-width: 1; vector-effect: non-scaling-stroke; }
.trend-hover-markers circle { stroke: var(--bg-primary); stroke-width: 3; vector-effect: non-scaling-stroke; }
.trend-tooltip {
  position: absolute;
  z-index: 2;
  top: 22px;
  display: grid;
  min-width: 190px;
  gap: 7px;
  padding: 10px 12px;
  border: 1px solid var(--border-light);
  border-radius: var(--border-radius-base);
  background: var(--bg-primary);
  box-shadow: var(--el-box-shadow-light);
  color: var(--text-primary);
  font-size: 12px;
  pointer-events: none;
  transform: translateX(-50%);
}
.trend-tooltip.is-left { transform: translateX(0); }
.trend-tooltip.is-right { transform: translateX(-100%); }
.trend-tooltip strong { padding-bottom: 4px; border-bottom: 1px solid var(--border-light); font-size: 12px; }
.trend-tooltip span { display: grid; grid-template-columns: 8px minmax(0, 1fr) auto; align-items: center; gap: 7px; color: var(--text-secondary); }
.trend-tooltip i { width: 8px; height: 8px; border-radius: 50%; }
.trend-tooltip b { color: var(--text-primary); font-variant-numeric: tabular-nums; }
.trend-legend { display: flex; flex-wrap: wrap; gap: 10px 18px; padding-top: 10px; border-top: 1px solid var(--border-light); }
.trend-legend span { display: inline-flex; align-items: center; gap: 6px; color: var(--text-secondary); font-size: 12px; }
.trend-legend i { width: 8px; height: 8px; border-radius: 50%; }
.trend-legend strong { color: var(--text-primary); font-weight: 600; }
.trend-legend small { color: var(--text-secondary); }

@media (max-width: 980px) {
  .trend-grid { grid-template-columns: 1fr; }
  .trend-card-wide { grid-column: auto; }
}

@media (max-width: 640px) {
  .trend-card-header { flex-direction: column; }
  .trend-chart { min-height: 180px; }
}
</style>
