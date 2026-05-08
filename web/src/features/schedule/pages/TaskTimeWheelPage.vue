<template>
  <div class="task-wheel-page">
    <section class="task-wheel-topbar">
      <div class="topbar-main">
        <el-button circle class="back-button" title="返回" @click="goBack">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <div class="title-block">
          <span class="page-kicker">平台调度</span>
          <h1>任务时间轮</h1>
        </div>
      </div>

      <div class="topbar-actions">
        <el-input
          v-model="keyword"
          clearable
          class="task-search"
          placeholder="搜索任务、路径、创建人"
          :prefix-icon="Search"
        />
        <el-select v-model="sourceFilter" class="filter-select" placeholder="类型">
          <el-option label="全部类型" value="all" />
          <el-option label="函数任务" value="function" />
          <el-option label="定时会话" value="agent" />
        </el-select>
        <el-select v-model="statusFilter" class="filter-select" placeholder="状态">
          <el-option label="全部状态" value="all" />
          <el-option label="活跃任务" value="active" />
          <el-option label="待执行" value="pending" />
          <el-option label="已暂停" value="paused" />
          <el-option label="已完成" value="done" />
          <el-option label="失败" value="failed" />
          <el-option label="已取消" value="cancelled" />
        </el-select>
        <el-button :icon="Refresh" :loading="loading" @click="loadTasks">刷新</el-button>
      </div>
    </section>

    <section class="metric-strip">
      <div class="metric-item">
        <span class="metric-label">全部</span>
        <strong>{{ allTasks.length }}</strong>
      </div>
      <div class="metric-item metric-item--active">
        <span class="metric-label">活跃</span>
        <strong>{{ activeCount }}</strong>
      </div>
      <div class="metric-item metric-item--agent">
        <span class="metric-label">定时会话</span>
        <strong>{{ agentCount }}</strong>
      </div>
      <div class="metric-item metric-item--danger">
        <span class="metric-label">异常</span>
        <strong>{{ errorCount }}</strong>
      </div>
    </section>

    <el-alert
      v-if="loadError"
      class="load-alert"
      type="warning"
      :closable="false"
      show-icon
      :title="loadError"
    />

    <section class="wheel-grid">
      <div class="wheel-panel" v-loading="loading">
        <div class="panel-head">
          <div>
            <span class="section-label">调度视图</span>
            <h2>调度序列</h2>
          </div>
          <div class="panel-tools">
            <el-button-group class="range-presets">
              <el-button
                v-for="preset in rangePresetOptions"
                :key="preset.value"
                size="small"
                :type="rangePreset === preset.value ? 'primary' : 'default'"
                @click="applyRangePreset(preset.value)"
              >
                {{ preset.label }}
              </el-button>
            </el-button-group>
            <el-date-picker
              v-model="timeRangeValue"
              class="range-picker"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              format="MM-DD HH:mm"
              :clearable="false"
              @change="handleTimeRangeChange"
            />
            <el-tag effect="plain">{{ wheelTasks.length }} / {{ filteredTasks.length }}</el-tag>
          </div>
        </div>

        <div class="spring-stage">
          <div class="time-axis">
            <span class="time-axis-label time-axis-label--start">{{ axisStartLabel }}</span>
            <span class="time-axis-label time-axis-label--end">{{ axisEndLabel }}</span>
          </div>
          <div class="timewire-layer" :style="timewireStyle">
            <svg class="spring-wire" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
              <defs>
                <linearGradient id="springWireGradient" x1="0" y1="0" x2="1" y2="0">
                  <stop offset="0%" stop-color="#38bdf8" stop-opacity="0.48" />
                  <stop offset="42%" stop-color="#22d3ee" stop-opacity="0.96" />
                  <stop offset="72%" stop-color="#34d399" stop-opacity="0.92" />
                  <stop offset="100%" stop-color="#fbbf24" stop-opacity="0.82" />
                </linearGradient>
                <linearGradient id="springTraceGradient" x1="0" y1="0" x2="1" y2="0">
                  <stop offset="0%" stop-color="#818cf8" stop-opacity="0" />
                  <stop offset="48%" stop-color="#e0f2fe" stop-opacity="0.92" />
                  <stop offset="100%" stop-color="#f0abfc" stop-opacity="0" />
                </linearGradient>
              </defs>
              <path class="spring-wire-shadow" :d="springWirePath" />
              <path class="spring-wire-halo" :d="springWirePath" />
              <path class="spring-wire-glow" :d="springWirePath" />
              <path class="spring-wire-body" :d="springWirePath" stroke="url(#springWireGradient)" />
              <path class="spring-wire-trace" :d="springWirePath" stroke="url(#springTraceGradient)" />
              <path class="spring-wire-highlight" :d="springWirePath" />
            </svg>
            <span class="timewire-pulse"></span>
            <span
              v-for="mark in wireMarks"
              :key="`wire-mark-${mark}`"
              class="wire-mark"
              :style="getWireMarkStyle(mark)"
            ></span>
            <span
              v-for="hour in hourMarks"
              :key="`hour-mark-${hour}`"
              class="hour-marker"
              :class="{ 'is-major': shouldShowHourMarkerLabel(hour) }"
              :style="getHourMarkStyle(hour)"
            >
              <span v-if="shouldShowHourMarkerLabel(hour)" class="hour-marker-label">
                {{ formatHourMarker(hour) }}
              </span>
            </span>
            <span
              v-if="currentPointerVisible"
              class="current-pointer"
              :style="currentPointerStyle"
            >
              <span class="current-pointer-beam"></span>
              <span class="current-pointer-head"></span>
              <span class="current-pointer-label">{{ currentPointerLabel }}</span>
            </span>
          </div>

          <button
            v-for="(task, index) in wheelTasks"
            :key="task.id"
            type="button"
            class="task-hanger"
            :class="[
              `task-hanger--${task.source}`,
              `task-hanger--${taskPhase(task)}`,
              { 'is-selected': selectedTask?.id === task.id }
            ]"
            :style="getNodeStyle(task, index, wheelTasks.length)"
            @click="selectTask(task)"
          >
            <span class="wire-pin"><span class="pin-core"></span></span>
            <span class="hanger-line"></span>
            <span class="task-card">
              <span class="node-main">{{ task.name }}</span>
              <span class="node-time">{{ formatShortTaskTime(task) }}</span>
              <span class="node-status">{{ taskPhaseLabel(task) }}</span>
            </span>
          </button>

          <div v-if="!loading && wheelTasks.length === 0" class="wheel-empty">
            暂无匹配的定时任务
          </div>
        </div>
      </div>
    </section>

    <section class="table-panel">
      <div class="panel-head table-panel-head">
        <div>
          <span class="section-label">任务注册表</span>
          <h2>任务清单</h2>
        </div>
        <span class="table-total">共 {{ filteredTasks.length }} 条</span>
      </div>

      <el-table
        :data="filteredTasks"
        stripe
        class="task-table"
        empty-text="暂无定时任务"
        @row-click="selectTask"
      >
        <el-table-column label="任务" min-width="260">
          <template #default="{ row }">
            <div class="task-name-cell">
              <strong>{{ row.name }}</strong>
              <span v-if="row.description">{{ row.description }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="110">
          <template #default="{ row }">
            <el-tag effect="plain" :type="row.source === 'agent' ? 'warning' : 'success'">
              {{ sourceLabel(row.source) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="tagTypeForStatus(row.status)" effect="light">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="调度" min-width="180">
          <template #default="{ row }">{{ scheduleSummary(row) }}</template>
        </el-table-column>
        <el-table-column label="下次执行" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.nextRunAt || row.runAt) }}</template>
        </el-table-column>
        <el-table-column prop="fullCodePath" label="路径" min-width="260" show-overflow-tooltip />
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click.stop="openTask(row)">打开</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import type { CSSProperties } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Refresh, Search } from '@element-plus/icons-vue'
import {
  listScheduledTasks,
  type ScheduledTaskItem
} from '@/api/scheduledTask'
import {
  listScheduledAgentTasks,
  type ScheduledAgentTaskItem
} from '@/api/scheduledAgentTask'
import { resolveWorkspaceUrl } from '@/utils/route'

type TimeWheelTaskSource = 'function' | 'agent'

interface TimeWheelTask {
  id: string
  rawId: number
  source: TimeWheelTaskSource
  name: string
  description: string
  fullCodePath: string
  status: string
  scheduleType: string
  runAt?: string
  nextRunAt?: string
  cronExpr?: string
  intervalSeconds?: number
  maxRuns?: number
  runCount: number
  createdBy: string
  createdAt: string
}

type TagType = '' | 'success' | 'warning' | 'info' | 'danger'
type TimeRangePreset = 'today24' | 'yesterday24' | 'tomorrow24' | 'custom'
type TaskPhase = 'pending' | 'finished' | 'running' | 'failed' | 'paused'

interface NormalizedTimeRange {
  start: Date
  end: Date
  startMs: number
  endMs: number
}

const ROOT_FULL_CODE_PATH = '/'
const PAGE_SIZE = 100
const MAX_PAGES = 10
const WHEEL_TASK_LIMIT = 24
const HOURS_IN_WHEEL = 24
const HOUR_MARK_COUNT = HOURS_IN_WHEEL + 1
const WIRE_MARK_COUNT = HOURS_IN_WHEEL * 3
const WHEEL_TICK_MS = 1000
const WHEEL_TICK_RADIANS = (Math.PI * 2) / HOURS_IN_WHEEL
const MS_PER_HOUR = 60 * 60 * 1000
const SPRING_LAYER_TOP_PERCENT = 16
const SPRING_LAYER_HEIGHT_PERCENT = 44
const SPRING_COIL_COUNT = HOURS_IN_WHEEL
const SPRING_AMPLITUDE = 30

const rangePresetOptions: Array<{
  label: string
  value: Exclude<TimeRangePreset, 'custom'>
  createRange: () => [Date, Date]
}> = [
  { label: '今日 0-24', value: 'today24', createRange: () => createDayRange(0) },
  { label: '昨日 0-24', value: 'yesterday24', createRange: () => createDayRange(-1) },
  { label: '明日 0-24', value: 'tomorrow24', createRange: () => createDayRange(1) }
]

const router = useRouter()
const allTasks = ref<TimeWheelTask[]>([])
const selectedTaskId = ref('')
const loading = ref(false)
const loadError = ref('')
const keyword = ref('')
const sourceFilter = ref<'all' | TimeWheelTaskSource>('all')
const statusFilter = ref('all')
const rotation = ref(0)
const currentTimeMs = ref(Date.now())
const wireMarks = Array.from({ length: WIRE_MARK_COUNT }, (_, index) => index)
const hourMarks = Array.from({ length: HOUR_MARK_COUNT }, (_, index) => index)
const rangePreset = ref<TimeRangePreset>('today24')
const timeRangeValue = ref<[Date, Date]>(createDayRange(0))

let wheelTimer: number | null = null

const searchFilteredTasks = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase()
  return allTasks.value.filter((task) => {
    if (sourceFilter.value !== 'all' && task.source !== sourceFilter.value) return false
    if (!matchesStatusFilter(task.status, statusFilter.value)) return false
    if (!normalizedKeyword) return true

    const haystack = [
      task.name,
      task.description,
      task.fullCodePath,
      task.createdBy,
      task.status,
      task.scheduleType
    ].join(' ').toLowerCase()
    return haystack.includes(normalizedKeyword)
  })
})

const filteredTasks = computed(() => {
  const { startMs, endMs } = selectedTimeRange.value
  return searchFilteredTasks.value.filter((task) => {
    const time = taskTime(task)
    return Number.isFinite(time) && time >= startMs && time <= endMs
  })
})

const wheelTasks = computed(() => {
  return [...filteredTasks.value]
    .sort(compareTasksByTime)
    .slice(0, WHEEL_TASK_LIMIT)
})

const selectedTask = computed(() => {
  return filteredTasks.value.find(task => task.id === selectedTaskId.value) || filteredTasks.value[0] || null
})

const selectedTimeRange = computed<NormalizedTimeRange>(() => normalizeTimeRange(timeRangeValue.value))

const wheelTimeRange = computed(() => {
  const { startMs, endMs } = selectedTimeRange.value
  return { min: startMs, max: endMs }
})

const activeCount = computed(() => allTasks.value.filter(task => matchesStatusFilter(task.status, 'active')).length)
const agentCount = computed(() => allTasks.value.filter(task => task.source === 'agent').length)
const errorCount = computed(() => allTasks.value.filter(task => ['failed', 'timeout'].includes(task.status)).length)
const axisStartLabel = computed(() => formatAxisDate(selectedTimeRange.value.start))
const axisEndLabel = computed(() => formatAxisDate(selectedTimeRange.value.end))
const currentPointerProgress = computed(() => {
  const { startMs, endMs } = selectedTimeRange.value
  if (endMs <= startMs) return -1
  return (currentTimeMs.value - startMs) / (endMs - startMs)
})
const currentPointerVisible = computed(() => currentPointerProgress.value >= 0 && currentPointerProgress.value <= 1)
const currentPointerStyle = computed<CSSProperties>(() => {
  const point = getSpringPoint(currentPointerProgress.value)
  return {
    left: `${point.x}%`,
    top: `${point.y}%`
  }
})
const currentPointerLabel = computed(() => {
  const date = new Date(currentTimeMs.value)
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
})

const timewireStyle = computed<CSSProperties>(() => {
  const progress = (rotation.value % (Math.PI * 2)) / (Math.PI * 2)
  const point = getSpringPoint(progress)
  return {
    '--pulse-x': `${point.x}%`,
    '--pulse-y': `${point.y}%`,
    '--trace-offset': `${Math.round(progress * -96)}`
  } as CSSProperties
})

const springWirePath = computed(() => {
  const sampleCount = 220
  const commands: string[] = []

  for (let index = 0; index <= sampleCount; index += 1) {
    const progress = index / sampleCount
    const point = getSpringPoint(progress)
    commands.push(`${index === 0 ? 'M' : 'L'} ${point.x.toFixed(2)} ${point.y.toFixed(2)}`)
  }

  return commands.join(' ')
})

watch(filteredTasks, (tasks) => {
  if (!tasks.some(task => task.id === selectedTaskId.value)) {
    selectedTaskId.value = tasks[0]?.id || ''
  }
})

onMounted(() => {
  void loadTasks()
  wheelTimer = window.setInterval(tickWheel, WHEEL_TICK_MS)
})

onUnmounted(() => {
  if (wheelTimer) {
    window.clearInterval(wheelTimer)
    wheelTimer = null
  }
})

async function loadTasks() {
  loading.value = true
  loadError.value = ''

  try {
    const [functionResult, agentResult] = await Promise.allSettled([
      loadFunctionTasks(),
      loadAgentTasks()
    ])
    const nextTasks: TimeWheelTask[] = []
    const errors: string[] = []

    if (functionResult.status === 'fulfilled') {
      nextTasks.push(...functionResult.value.map(mapFunctionTask))
    } else {
      errors.push('函数任务加载失败')
    }

    if (agentResult.status === 'fulfilled') {
      nextTasks.push(...agentResult.value.map(mapAgentTask))
    } else {
      errors.push('定时会话加载失败')
    }

    allTasks.value = nextTasks.sort(compareTasksByTime)
    selectedTaskId.value = filteredTasks.value[0]?.id || ''

    if (errors.length > 0) {
      loadError.value = errors.join('，')
      ElMessage.warning(loadError.value)
    }
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '定时任务加载失败'
    ElMessage.error(loadError.value)
  } finally {
    loading.value = false
  }
}

async function loadFunctionTasks(): Promise<ScheduledTaskItem[]> {
  const result: ScheduledTaskItem[] = []
  let total = 0

  for (let page = 1; page <= MAX_PAGES; page += 1) {
    const resp = await listScheduledTasks({
      full_code_path: ROOT_FULL_CODE_PATH,
      page,
      page_size: PAGE_SIZE
    })
    const list = resp.list || []
    result.push(...list)
    total = resp.total || result.length
    if (result.length >= total || list.length === 0) break
  }

  return result
}

async function loadAgentTasks(): Promise<ScheduledAgentTaskItem[]> {
  const result: ScheduledAgentTaskItem[] = []
  let total = 0

  for (let page = 1; page <= MAX_PAGES; page += 1) {
    const resp = await listScheduledAgentTasks({
      full_code_path: ROOT_FULL_CODE_PATH,
      page,
      page_size: PAGE_SIZE
    })
    const list = resp.list || []
    result.push(...list)
    total = resp.total || result.length
    if (result.length >= total || list.length === 0) break
  }

  return result
}

function mapFunctionTask(task: ScheduledTaskItem): TimeWheelTask {
  return {
    id: `function-${task.id}`,
    rawId: task.id,
    source: 'function',
    name: task.name || pathTail(task.full_code_path),
    description: task.action ? actionLabel(task.action) : '',
    fullCodePath: task.full_code_path,
    status: task.status,
    scheduleType: task.schedule_type,
    runAt: task.run_at,
    nextRunAt: task.next_run_at,
    cronExpr: task.cron_expr,
    intervalSeconds: task.interval_seconds,
    maxRuns: task.max_runs,
    runCount: task.run_count,
    createdBy: task.created_by,
    createdAt: task.created_at
  }
}

function mapAgentTask(task: ScheduledAgentTaskItem): TimeWheelTask {
  return {
    id: `agent-${task.id}`,
    rawId: task.id,
    source: 'agent',
    name: task.name || pathTail(task.full_code_path),
    description: task.goal || '',
    fullCodePath: task.full_code_path,
    status: task.status,
    scheduleType: task.schedule_type,
    runAt: task.run_at,
    nextRunAt: task.next_run_at,
    cronExpr: task.cron_expr,
    intervalSeconds: task.interval_seconds,
    maxRuns: task.max_runs,
    runCount: task.run_count,
    createdBy: task.created_by,
    createdAt: task.created_at
  }
}

function tickWheel() {
  currentTimeMs.value = Date.now()
  rotation.value = (rotation.value + WHEEL_TICK_RADIANS) % (Math.PI * 2)
}

function getNodeStyle(task: TimeWheelTask, index: number, total: number): CSSProperties {
  const x = getTaskTimePosition(task, index, total)
  const progress = Math.min(Math.max((x - 8) / 84, 0), 1)
  const railY = getSpringStageY(progress)
  const dropLanes = [34, 70, 106, 142, 178]
  const drop = dropLanes[index % dropLanes.length]
  const translateX = x < 12 ? '-15%' : x > 88 ? '-85%' : '-50%'

  return {
    left: `${x}%`,
    top: `${railY}%`,
    opacity: 1,
    zIndex: `${80 + (index % dropLanes.length) * 8}`,
    transform: `translate(${translateX}, -8px)`,
    '--hanger-drop': `${drop}px`
  } as CSSProperties
}

function getWireMarkStyle(index: number): CSSProperties {
  const progress = index / Math.max(WIRE_MARK_COUNT - 1, 1)
  const glow = Math.cos(rotation.value + progress * Math.PI * 2)
  const size = 5 + (glow + 1) * 1.5
  const point = getSpringPoint(progress)

  return {
    left: `${point.x}%`,
    top: `${point.y}%`,
    width: `${size}px`,
    height: `${size}px`,
    opacity: 0.38 + (glow + 1) * 0.18,
    transform: 'translate(-50%, -50%)'
  }
}

function getHourMarkStyle(index: number): CSSProperties {
  const progress = index / Math.max(HOURS_IN_WHEEL, 1)
  const point = getSpringPoint(progress)
  return {
    left: `${point.x}%`,
    top: `${point.y}%`
  }
}

function getTaskTimePosition(task: TimeWheelTask, index: number, total: number): number {
  const count = Math.max(total, 1)
  const fallback = count === 1 ? 50 : 8 + (index / (count - 1)) * 84
  const time = taskTime(task)
  const { min, max } = wheelTimeRange.value
  if (!Number.isFinite(time) || time >= Number.MAX_SAFE_INTEGER || max <= min) {
    return fallback
  }
  return Math.min(92, Math.max(8, 8 + ((time - min) / (max - min)) * 84))
}

function getSpringPoint(progress: number) {
  const safeProgress = Math.min(Math.max(progress, 0), 1)
  return {
    x: safeProgress * 100,
    y: getSpringCurveY(safeProgress)
  }
}

function getSpringCurveY(progress: number): number {
  return 50 + Math.sin(rotation.value + progress * Math.PI * 2 * SPRING_COIL_COUNT) * SPRING_AMPLITUDE
}

function getSpringStageY(progress: number): number {
  return SPRING_LAYER_TOP_PERCENT + (getSpringCurveY(progress) / 100) * SPRING_LAYER_HEIGHT_PERCENT
}

function createDayRange(dayOffset: number): [Date, Date] {
  const start = new Date()
  start.setHours(0, 0, 0, 0)
  start.setDate(start.getDate() + dayOffset)
  return [start, new Date(start.getTime() + HOURS_IN_WHEEL * MS_PER_HOUR)]
}

function normalizeTimeRange(value: [Date, Date] | null | undefined): NormalizedTimeRange {
  const fallback = createDayRange(0)
  const start = toValidDate(value?.[0]) || fallback[0]
  const end = toValidDate(value?.[1]) || fallback[1]
  const normalizedStart = start.getTime() <= end.getTime() ? start : end
  const normalizedEnd = new Date(normalizedStart.getTime() + HOURS_IN_WHEEL * MS_PER_HOUR)

  return {
    start: normalizedStart,
    end: normalizedEnd,
    startMs: normalizedStart.getTime(),
    endMs: normalizedEnd.getTime()
  }
}

function toValidDate(value?: Date | string | number): Date | null {
  if (!value) return null
  const date = value instanceof Date ? value : new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function applyRangePreset(value: Exclude<TimeRangePreset, 'custom'>) {
  const preset = rangePresetOptions.find(item => item.value === value)
  if (!preset) return
  rangePreset.value = value
  timeRangeValue.value = preset.createRange()
}

function handleTimeRangeChange() {
  const normalized = normalizeTimeRange(timeRangeValue.value)
  timeRangeValue.value = [normalized.start, normalized.end]
  rangePreset.value = 'custom'
}

function shouldShowHourMarkerLabel(index: number): boolean {
  return index === 0 || index === HOURS_IN_WHEEL || index % 4 === 0
}

function formatHourMarker(index: number): string {
  const date = new Date(selectedTimeRange.value.startMs + index * MS_PER_HOUR)
  return `${pad(date.getHours())}:00`
}

function selectTask(task: TimeWheelTask) {
  selectedTaskId.value = task.id
}

function goBack() {
  if (window.history.length > 1) {
    router.back()
    return
  }
  void router.push('/workspace')
}

function openTask(task: TimeWheelTask) {
  if (!task.fullCodePath) {
    ElMessage.warning('任务路径为空')
    return
  }
  const path = resolveWorkspaceUrl(task.fullCodePath)
  const panel = task.source === 'agent' ? 'scheduledAgentTask' : 'scheduledTask'
  void router.push({ path, query: { _panel: panel } })
}

function matchesStatusFilter(status: string, filter: string): boolean {
  if (filter === 'all') return true
  if (filter === 'active') return ['pending', 'paused', 'running'].includes(status)
  if (filter === 'done') return ['done', 'success'].includes(status)
  if (filter === 'failed') return ['failed', 'timeout'].includes(status)
  return status === filter
}

function compareTasksByTime(left: TimeWheelTask, right: TimeWheelTask): number {
  return taskTime(left) - taskTime(right)
}

function taskTime(task: TimeWheelTask): number {
  const value = task.nextRunAt || task.runAt || task.createdAt
  const time = Date.parse(value || '')
  return Number.isFinite(time) ? time : Number.MAX_SAFE_INTEGER
}

function sourceLabel(source: TimeWheelTaskSource): string {
  return source === 'agent' ? '定时会话' : '函数任务'
}

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: '待执行',
    paused: '已暂停',
    running: '运行中',
    done: '已完成',
    success: '成功',
    failed: '失败',
    timeout: '超时',
    cancelled: '已取消'
  }
  return map[status] || status || '-'
}

function taskPhase(task: TimeWheelTask): TaskPhase {
  if (['failed', 'timeout'].includes(task.status)) return 'failed'
  if (['running'].includes(task.status)) return 'running'
  if (['paused', 'cancelled'].includes(task.status)) return 'paused'
  if (['done', 'success'].includes(task.status)) return 'finished'
  return 'pending'
}

function taskPhaseLabel(task: TimeWheelTask): string {
  const phase = taskPhase(task)
  if (phase === 'finished') return '已执行'
  if (phase === 'running') return '执行中'
  if (phase === 'failed') return '异常'
  if (phase === 'paused') return task.status === 'cancelled' ? '已取消' : '已暂停'
  if (task.runCount > 0) return `已执行 ${task.runCount} 次 / 待执行`
  return '待执行'
}

function tagTypeForStatus(status: string): TagType {
  if (['done', 'success'].includes(status)) return 'success'
  if (['failed', 'timeout'].includes(status)) return 'danger'
  if (status === 'pending') return 'warning'
  if (status === 'paused' || status === 'cancelled') return 'info'
  return ''
}

function scheduleSummary(task: TimeWheelTask): string {
  if (task.scheduleType === 'atime') {
    return `单次 ${formatDateTime(task.runAt)}`
  }
  if (task.scheduleType === 'cron') {
    return `Cron ${task.cronExpr || '-'}`
  }
  if (task.scheduleType === 'every') {
    return `每 ${formatInterval(task.intervalSeconds)}`
  }
  return task.scheduleType || '-'
}

function formatShortTaskTime(task: TimeWheelTask): string {
  const value = task.nextRunAt || task.runAt
  if (!value) return statusLabel(task.status)
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return statusLabel(task.status)
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function formatAxisDate(value?: Date | string): string {
  if (!value) return '-'
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function formatInterval(seconds?: number): string {
  if (!seconds || seconds < 1) return '-'
  if (seconds % 3600 === 0) return `${seconds / 3600} 小时`
  if (seconds % 60 === 0) return `${seconds / 60} 分钟`
  return `${seconds} 秒`
}

function actionLabel(action: string): string {
  const map: Record<string, string> = {
    execute: '执行函数',
    table_create: '新增数据',
    table_update: '更新数据',
    table_delete: '删除数据'
  }
  return map[action] || action
}

function pathTail(path: string): string {
  const parts = path.split('/').filter(Boolean)
  return parts[parts.length - 1] || '未命名任务'
}

function pad(value: number): string {
  return String(value).padStart(2, '0')
}
</script>

<style scoped lang="scss">
.task-wheel-page {
  min-height: 100vh;
  padding: 24px;
  color: var(--el-text-color-primary);
  background:
    linear-gradient(180deg, rgba(16, 185, 129, 0.08), transparent 300px),
    var(--el-bg-color-page);
}

.task-wheel-topbar,
.metric-strip,
.wheel-panel,
.table-panel {
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.08);
}

.task-wheel-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
  border-radius: 14px;
}

.topbar-main,
.topbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.topbar-actions {
  flex-wrap: wrap;
  justify-content: flex-end;
}

.back-button {
  flex-shrink: 0;
}

.title-block {
  min-width: 0;

  h1 {
    margin: 3px 0 0;
    font-size: 24px;
    line-height: 1.2;
    letter-spacing: 0;
  }
}

.page-kicker,
.section-label,
.metric-label {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
}

.task-search {
  width: min(320px, 38vw);
}

.filter-select {
  width: 128px;
}

.metric-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  margin-top: 16px;
  overflow: hidden;
  border-radius: 14px;
}

.metric-item {
  display: flex;
  min-height: 82px;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  padding: 16px 18px;
  background: linear-gradient(135deg, rgba(148, 163, 184, 0.08), transparent);

  strong {
    font-size: 28px;
    line-height: 1;
  }
}

.metric-item--active {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.12), transparent);
}

.metric-item--agent {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.12), transparent);
}

.metric-item--danger {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.1), transparent);
}

.load-alert {
  margin-top: 16px;
}

.wheel-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 16px;
  margin-top: 16px;
}

.wheel-panel,
.table-panel {
  border-radius: 14px;
}

.wheel-panel {
  min-width: 0;
  padding: 20px;
  border-color: rgba(34, 211, 238, 0.34);
  color: #dbeafe;
  background:
    linear-gradient(135deg, rgba(12, 18, 38, 0.98), rgba(2, 8, 23, 0.98) 58%, rgba(10, 29, 35, 0.96));
  box-shadow:
    0 28px 70px rgba(2, 6, 23, 0.36),
    inset 0 0 0 1px rgba(125, 211, 252, 0.08);
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;

  h2 {
    margin: 3px 0 0;
    font-size: 18px;
    line-height: 1.25;
    letter-spacing: 0;
  }
}

.panel-tools {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.range-presets {
  flex-shrink: 0;
}

.range-picker {
  width: 292px;
}

.spring-stage {
  position: relative;
  height: min(72vh, 760px);
  min-height: 640px;
  overflow: hidden;
  border: 1px solid rgba(34, 211, 238, 0.34);
  border-radius: 12px;
  background:
    linear-gradient(180deg, rgba(3, 7, 18, 0.86), rgba(8, 17, 34, 0.98)),
    linear-gradient(115deg, transparent 0 36%, rgba(34, 211, 238, 0.1) 47%, transparent 60%),
    repeating-linear-gradient(90deg, rgba(125, 211, 252, 0.1) 0 1px, transparent 1px 64px),
    repeating-linear-gradient(0deg, rgba(52, 211, 153, 0.075) 0 1px, transparent 1px 46px),
    #020617;
  box-shadow:
    inset 0 0 0 1px rgba(148, 163, 184, 0.1),
    inset 0 0 48px rgba(34, 211, 238, 0.1),
    0 20px 60px rgba(2, 6, 23, 0.34);
}

.time-axis {
  position: absolute;
  top: 16px;
  right: 28px;
  left: 28px;
  height: 24px;
  border-bottom: 1px solid rgba(125, 211, 252, 0.24);
  color: rgba(191, 219, 254, 0.82);
  font-size: 11px;
  font-weight: 700;
  pointer-events: none;
}

.time-axis::before,
.time-axis::after {
  content: '';
  position: absolute;
  bottom: -4px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #22d3ee;
  box-shadow: 0 0 14px rgba(34, 211, 238, 0.85);
}

.time-axis::before { left: 0; }
.time-axis::after { right: 0; }

.time-axis-label {
  position: absolute;
  top: 0;
  white-space: nowrap;
}

.time-axis-label--start { left: 0; }
.time-axis-label--end { right: 0; }

.timewire-layer {
  position: absolute;
  left: 5%;
  right: 5%;
  top: 16%;
  height: 44%;
  pointer-events: none;
}

.spring-stage::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, transparent 0, rgba(34, 211, 238, 0.12) 50%, transparent 100%),
    linear-gradient(180deg, transparent 0 24%, rgba(125, 211, 252, 0.12) 25%, transparent 26% 67%, rgba(52, 211, 153, 0.08) 68%, transparent 69%),
    repeating-linear-gradient(180deg, transparent 0 9px, rgba(148, 163, 184, 0.052) 10px),
    repeating-linear-gradient(90deg, transparent 0 138px, rgba(224, 242, 254, 0.075) 139px, transparent 141px);
  pointer-events: none;
}

.spring-stage::after {
  content: '';
  position: absolute;
  inset: 0;
  border: 1px solid rgba(125, 211, 252, 0.16);
  border-radius: 12px;
  box-shadow:
    inset 0 0 38px rgba(34, 211, 238, 0.13),
    inset 0 -80px 120px rgba(8, 145, 178, 0.08);
  pointer-events: none;
}

.spring-wire,
.timewire-pulse,
.wire-mark,
.hour-marker,
.current-pointer {
  position: absolute;
}

.spring-wire {
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.spring-wire path {
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}

.spring-wire-shadow {
  stroke: rgba(34, 211, 238, 0.18);
  stroke-width: 44;
  filter: blur(14px);
}

.spring-wire-halo {
  stroke: rgba(129, 140, 248, 0.16);
  stroke-width: 28;
  filter: blur(6px);
}

.spring-wire-glow {
  stroke: rgba(45, 212, 191, 0.34);
  stroke-width: 17;
  filter:
    drop-shadow(0 0 22px rgba(34, 211, 238, 0.78))
    drop-shadow(0 0 38px rgba(52, 211, 153, 0.24));
}

.spring-wire-body {
  stroke-width: 9;
  filter: drop-shadow(0 0 10px rgba(224, 242, 254, 0.38));
}

.spring-wire-trace {
  stroke-width: 3.2;
  stroke-dasharray: 9 18;
  stroke-dashoffset: var(--trace-offset, 0);
  opacity: 0.88;
  transition: stroke-dashoffset 0.78s cubic-bezier(0.16, 1, 0.3, 1);
}

.spring-wire-highlight {
  stroke: rgba(236, 254, 255, 0.86);
  stroke-width: 2.2;
  opacity: 0.9;
}

.timewire-pulse {
  top: var(--pulse-y, 50%);
  left: var(--pulse-x, 0%);
  width: 26px;
  height: 26px;
  border: 1px solid rgba(224, 242, 254, 0.92);
  border-radius: 50%;
  background:
    radial-gradient(circle, #ecfeff 0 16%, #22d3ee 20% 48%, rgba(129, 140, 248, 0.3) 52%, rgba(34, 211, 238, 0.08) 70%);
  box-shadow:
    0 0 22px rgba(34, 211, 238, 0.98),
    0 0 44px rgba(16, 185, 129, 0.54),
    0 0 72px rgba(129, 140, 248, 0.24);
  transform: translate(-50%, -50%);
  transition:
    left 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    top 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 0.28s ease;
}

.wire-mark {
  border: 1px solid rgba(191, 219, 254, 0.68);
  border-radius: 50%;
  background: #22d3ee;
  box-shadow:
    0 0 12px rgba(34, 211, 238, 0.85),
    0 0 26px rgba(16, 185, 129, 0.36),
    0 0 42px rgba(129, 140, 248, 0.16);
  transition:
    width 0.78s ease,
    height 0.78s ease,
    opacity 0.78s ease,
    top 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    left 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.78s cubic-bezier(0.16, 1, 0.3, 1);
  will-change: transform, left, top, opacity;
}

.hour-marker {
  width: 9px;
  height: 9px;
  border: 1px solid rgba(224, 242, 254, 0.62);
  border-radius: 50%;
  background: rgba(2, 6, 23, 0.88);
  box-shadow:
    0 0 12px rgba(34, 211, 238, 0.64),
    0 0 24px rgba(125, 211, 252, 0.22);
  transform: translate(-50%, -50%);
  transition:
    top 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    left 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 0.24s ease;
}

.hour-marker::after {
  content: '';
  position: absolute;
  top: 11px;
  left: 50%;
  width: 1px;
  height: 34px;
  background: linear-gradient(180deg, rgba(125, 211, 252, 0.56), transparent);
  transform: translateX(-50%);
}

.hour-marker.is-major {
  width: 13px;
  height: 13px;
  border-color: rgba(224, 242, 254, 0.92);
  background: rgba(14, 165, 233, 0.22);
}

.hour-marker-label {
  position: absolute;
  top: 48px;
  left: 50%;
  color: rgba(191, 219, 254, 0.78);
  font-size: 10px;
  font-weight: 700;
  white-space: nowrap;
  transform: translateX(-50%);
}

.current-pointer {
  width: 1px;
  height: 1px;
  z-index: 60;
  transform: translate(-50%, -50%);
  transition:
    top 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    left 0.78s cubic-bezier(0.16, 1, 0.3, 1);
  will-change: left, top;
}

.current-pointer-beam {
  position: absolute;
  bottom: 10px;
  left: 50%;
  width: 2px;
  height: 116px;
  background: linear-gradient(180deg, transparent, rgba(236, 254, 255, 0.88), rgba(34, 211, 238, 0.08));
  box-shadow:
    0 0 18px rgba(34, 211, 238, 0.86),
    0 0 46px rgba(129, 140, 248, 0.38);
  transform: translateX(-50%);
}

.current-pointer-beam::before {
  content: '';
  position: absolute;
  top: 0;
  left: 50%;
  width: 76px;
  height: 76px;
  border: 1px solid rgba(125, 211, 252, 0.16);
  border-radius: 50%;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.16), transparent 65%);
  transform: translate(-50%, -28%);
}

.current-pointer-head {
  position: absolute;
  top: -17px;
  left: 50%;
  width: 0;
  height: 0;
  border-right: 9px solid transparent;
  border-left: 9px solid transparent;
  border-top: 18px solid #ecfeff;
  filter:
    drop-shadow(0 0 10px rgba(34, 211, 238, 0.95))
    drop-shadow(0 0 22px rgba(129, 140, 248, 0.45));
  transform: translateX(-50%);
}

.current-pointer-label {
  position: absolute;
  bottom: 130px;
  left: 50%;
  min-width: 70px;
  padding: 4px 8px;
  border: 1px solid rgba(125, 211, 252, 0.5);
  border-radius: 999px;
  color: #ecfeff;
  font-size: 11px;
  font-weight: 800;
  line-height: 1;
  text-align: center;
  background: rgba(2, 8, 23, 0.84);
  box-shadow:
    0 0 18px rgba(34, 211, 238, 0.32),
    inset 0 0 0 1px rgba(224, 242, 254, 0.08);
  transform: translateX(-50%);
}

.task-hanger {
  position: absolute;
  display: flex;
  width: clamp(178px, 17vw, 252px);
  cursor: pointer;
  border: 0;
  padding: 0;
  flex-direction: column;
  align-items: center;
  color: #e0f2fe;
  background: transparent;
  transition:
    filter 0.16s ease,
    opacity 0.78s ease,
    top 0.78s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.78s cubic-bezier(0.16, 1, 0.3, 1);
  will-change: transform, left, top, opacity;

  &:hover,
  &.is-selected {
    filter: drop-shadow(0 18px 28px rgba(34, 211, 238, 0.35));

    .task-card {
      border-color: rgba(125, 211, 252, 0.9);
      background: linear-gradient(135deg, rgba(8, 47, 73, 0.98), rgba(2, 44, 34, 0.94));
      box-shadow:
        0 20px 42px rgba(8, 145, 178, 0.32),
        inset 0 0 0 1px rgba(224, 242, 254, 0.14);
    }
  }
}

.wire-pin {
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(125, 211, 252, 0.88);
  border-radius: 50%;
  background: rgba(2, 6, 23, 0.92);
  box-shadow:
    0 0 0 4px rgba(34, 211, 238, 0.12),
    0 0 22px rgba(34, 211, 238, 0.82),
    inset 0 0 12px rgba(125, 211, 252, 0.2);
}

.pin-core {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #67e8f9;
}

.hanger-line {
  width: 1px;
  height: var(--hanger-drop, 34px);
  background: linear-gradient(180deg, rgba(224, 242, 254, 0.92), rgba(125, 211, 252, 0.74), rgba(52, 211, 153, 0.08));
  box-shadow:
    0 0 14px rgba(34, 211, 238, 0.72),
    0 0 28px rgba(52, 211, 153, 0.24);
}

.task-card {
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr);
  grid-template-rows: auto auto auto;
  column-gap: 10px;
  width: 100%;
  min-height: 66px;
  padding: 12px 14px;
  border: 1px solid rgba(34, 211, 238, 0.58);
  border-radius: 8px;
  clip-path: polygon(0 0, calc(100% - 16px) 0, 100% 16px, 100% 100%, 0 100%);
  text-align: left;
  background:
    linear-gradient(135deg, rgba(4, 12, 28, 0.98), rgba(8, 31, 45, 0.94) 58%, rgba(14, 47, 60, 0.88)),
    repeating-linear-gradient(90deg, rgba(125, 211, 252, 0.08) 0 1px, transparent 1px 18px);
  box-shadow:
    0 22px 52px rgba(2, 6, 23, 0.46),
    inset 0 0 0 1px rgba(224, 242, 254, 0.11),
    inset 0 0 28px rgba(34, 211, 238, 0.05);
  backdrop-filter: blur(12px);
  transition: border-color 0.16s ease, box-shadow 0.16s ease, background 0.16s ease;
}

.task-card::before {
  content: '';
  grid-row: 1 / 4;
  align-self: center;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #34d399;
  box-shadow:
    0 0 0 4px rgba(16, 185, 129, 0.16),
    0 0 18px rgba(16, 185, 129, 0.72);
}

.task-hanger--agent {
  .wire-pin {
    border-color: rgba(251, 191, 36, 0.88);
    box-shadow:
      0 0 0 3px rgba(245, 158, 11, 0.14),
      0 0 18px rgba(245, 158, 11, 0.74);
  }

  .pin-core,
  .task-card::before {
    background: #fbbf24;
    box-shadow:
      0 0 0 4px rgba(245, 158, 11, 0.16),
      0 0 18px rgba(245, 158, 11, 0.72);
  }

  .task-card {
    border-color: rgba(251, 191, 36, 0.5);
  }
}

.task-hanger--finished {
  .wire-pin {
    border-color: rgba(52, 211, 153, 0.92);
  }

  .pin-core,
  .task-card::before {
    background: #34d399;
  }

  .node-status {
    color: #86efac;
  }
}

.task-hanger--pending {
  .wire-pin {
    border-color: rgba(56, 189, 248, 0.92);
  }

  .pin-core,
  .task-card::before {
    background: #38bdf8;
  }

  .node-status {
    color: #93c5fd;
  }
}

.task-hanger--running {
  .wire-pin {
    border-color: rgba(168, 85, 247, 0.92);
  }

  .pin-core,
  .task-card::before {
    background: #a855f7;
  }

  .task-card {
    border-color: rgba(168, 85, 247, 0.64);
  }

  .node-status {
    color: #d8b4fe;
  }
}

.task-hanger--failed {
  .wire-pin {
    border-color: rgba(248, 113, 113, 0.92);
  }

  .pin-core,
  .task-card::before {
    background: #f87171;
  }

  .task-card {
    border-color: rgba(248, 113, 113, 0.62);
  }

  .node-status {
    color: #fecaca;
  }
}

.task-hanger--paused {
  .wire-pin {
    border-color: rgba(148, 163, 184, 0.86);
  }

  .pin-core,
  .task-card::before {
    background: #94a3b8;
  }

  .task-card {
    border-color: rgba(148, 163, 184, 0.5);
  }

  .node-status {
    color: #cbd5e1;
  }
}

.node-main,
.node-time,
.node-status {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-main {
  color: #ecfeff;
  font-size: 14px;
  font-weight: 700;
}

.node-time {
  color: #7dd3fc;
  font-size: 11px;
  font-weight: 600;
}

.node-status {
  margin-top: 2px;
  color: rgba(191, 219, 254, 0.74);
  font-size: 11px;
  font-weight: 700;
}

.wheel-empty {
  display: flex;
  height: 100%;
  align-items: center;
  justify-content: center;
  color: rgba(191, 219, 254, 0.74);
}

.table-panel {
  margin-top: 16px;
  padding: 18px;
}

.table-panel-head {
  margin-bottom: 12px;
}

.table-total {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  font-weight: 600;
}

.task-table {
  width: 100%;
}

.task-name-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;

  strong,
  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

@media (max-width: 1080px) {
  .task-wheel-topbar {
    align-items: stretch;
    flex-direction: column;
  }

  .topbar-actions {
    justify-content: flex-start;
  }

  .task-search {
    width: min(100%, 420px);
  }

  .wheel-grid {
    grid-template-columns: 1fr;
  }

  .panel-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .panel-tools {
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 720px) {
  .task-wheel-page {
    padding: 14px;
  }

  .topbar-main {
    align-items: flex-start;
  }

  .topbar-actions,
  .task-search,
  .filter-select,
  .topbar-actions :deep(.el-button),
  .range-picker {
    width: 100%;
  }

  .range-presets {
    display: grid;
    width: 100%;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .range-presets :deep(.el-button) {
    width: 100%;
  }

  .metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .spring-stage {
    height: 560px;
    min-height: 520px;
  }

  .task-hanger {
    width: min(68vw, 220px);
  }
}
</style>
