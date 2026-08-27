<template>
  <section v-if="packageNode" class="directory-overview" v-loading="loading">
    <div class="overview-head">
      <div class="overview-head-main">
        <div class="overview-title">{{ t('scheduledTask.overviewTitle') }}</div>
        <div class="overview-path">{{ packageNode.full_code_path || '-' }}</div>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadOverview">
        {{ t('common.refresh') }}
      </el-button>
    </div>

    <aside
      v-if="floatingAgentTasks.length > 0"
      :class="[
        'agent-presence-float',
        `is-${agentPresenceState}`,
        { 'is-expanded': agentRosterExpanded },
      ]"
      data-testid="agent-presence-float"
      aria-label="本服务目录数字员工"
    >
      <div class="agent-presence-head">
        <span class="agent-presence-title">
          <span class="agent-presence-live-dot" aria-hidden="true" />
          数字员工
        </span>
        <span class="agent-presence-summary">{{ agentPresenceSummary }}</span>
      </div>
      <div id="agent-presence-team" class="agent-presence-team">
        <div
          v-for="item in floatingAgentTasks"
          :key="`presence:${item.key}`"
          :class="[
            'agent-presence-employee',
            `is-${agentEmployeeState(item.task)}`,
            { 'is-started': isAgentTaskStarted(item.task) },
          ]"
        >
          <button
            type="button"
            class="agent-presence-open"
            :title="`${item.task.title || t('scheduledTask.unnamedAgentTask')} · ${agentPresenceStatus(item.task)}`"
            @click="openTask(item)"
          >
            <span class="agent-presence-avatar">
              <AgentEmployeeMascot
                variant="employee"
                :state="agentEmployeeState(item.task)"
                :label="`${item.task.title || t('scheduledTask.unnamedAgentTask')}，${agentPresenceStatus(item.task)}`"
              />
            </span>
            <span class="agent-presence-employee-copy">
              <strong>{{ item.task.title || t('scheduledTask.unnamedAgentTask') }}</strong>
              <span class="agent-presence-status-line">
                <small>{{ agentPresenceStatus(item.task) }}</small>
                <span
                  :class="['agent-presence-start-state', { 'is-started': isAgentTaskStarted(item.task) }]"
                >
                  {{ agentTaskActivationLabel(item.task) }}
                </span>
              </span>
            </span>
          </button>
          <button
            v-if="canStartAgentTask(item.task)"
            type="button"
            class="agent-presence-start"
            :disabled="isStartingAgentTask(item.task)"
            :aria-label="`启动 ${item.task.title || t('scheduledTask.unnamedAgentTask')}`"
            @click="startAgentTask(item)"
          >
            <el-icon v-if="!isStartingAgentTask(item.task)"><CaretRight /></el-icon>
            {{ isStartingAgentTask(item.task) ? '启动中…' : '启动' }}
          </button>
        </div>
      </div>
      <button
        v-if="canExpandAgentRoster"
        type="button"
        class="agent-presence-more"
        :aria-expanded="agentRosterExpanded"
        aria-controls="agent-presence-team"
        @click="agentRosterExpanded = !agentRosterExpanded"
      >
        <el-icon>
          <ArrowUp v-if="agentRosterExpanded" />
          <ArrowDown v-else />
        </el-icon>
        {{ agentRosterExpanded ? '收起员工' : `展开其余 ${hiddenAgentTaskCount} 名员工` }}
      </button>
    </aside>

    <div class="overview-metrics">
      <div class="metric-item">
        <div class="metric-icon metric-icon--directory">
          <el-icon><Folder /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">{{ t('scheduledTask.subdirectories') }}</div>
          <div class="metric-value">{{ resourceStats.directories }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--function">
          <el-icon><Operation /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">{{ t('scheduledTask.functions') }}</div>
          <div class="metric-value">{{ resourceStats.functions }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--docs">
          <el-icon><Document /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">{{ t('scheduledTask.docs') }}</div>
          <div class="metric-value">{{ resourceStats.docs }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--task">
          <el-icon><Timer /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">{{ t('scheduledTask.scheduledFunctions') }}</div>
          <div class="metric-value">{{ scheduledFunctionTotal }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--session">
          <el-icon><ChatLineRound /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">{{ t('scheduledTask.scheduledAgents') }}</div>
          <div class="metric-value">{{ scheduledAgentTotal }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--running">
          <el-icon><VideoPlay /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">{{ t('scheduledTask.runningTasks') }}</div>
          <div class="metric-value">{{ runningTaskCount }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--next">
          <el-icon><Clock /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">{{ t('scheduledTask.nextTrigger') }}</div>
          <div class="metric-value metric-value--time">{{ nextRunLabel }}</div>
        </div>
      </div>
    </div>

    <el-alert
      v-if="errorMessage"
      class="overview-alert"
      type="warning"
      show-icon
      :closable="false"
      :title="errorMessage"
    />

    <el-alert
      v-if="partialHint"
      class="overview-alert"
      type="info"
      show-icon
      :closable="false"
      :title="partialHint"
    />

    <div class="scheduled-panels">
      <section class="scheduled-panel">
        <div class="scheduled-panel-head">
          <div>
            <div class="scheduled-panel-title">
              <el-icon><Timer /></el-icon>
              {{ t('scheduledTask.scheduledFunctions') }}
            </div>
            <div class="scheduled-panel-subtitle">{{ t('scheduledTask.functionOverviewSubtitle') }}</div>
          </div>
          <el-tag size="small" type="primary">{{ scheduledFunctionTotal }}</el-tag>
        </div>

        <template v-if="displayFunctionTasks.length > 0">
          <div class="task-list">
            <button
              v-for="item in displayFunctionTasks"
              :key="item.key"
              type="button"
              class="task-row"
              @click="openTask(item)"
            >
              <span class="task-row-main">
                <span class="task-row-title">
                  {{ item.task.title || t('scheduledTask.unnamedFunctionTask') }}
                  <el-tag
                    v-if="item.task.inflight_execution_id"
                    size="small"
                    type="primary"
                    effect="light"
                  >
                    <el-icon class="is-loading" style="margin-right: 4px"><Loading /></el-icon>{{ t('scheduledTask.running') }}
                  </el-tag>
                </span>
                <span class="task-row-path">{{ item.resourceName }} · {{ item.resourcePath }}</span>
                <span v-if="item.task.last_error_message" class="task-row-error">
                  {{ item.task.last_error_message }}
                </span>
              </span>
              <span class="task-row-side">
                <el-tag :type="taskStatusTag(item.task.status)" size="small">
                  {{ taskStatusLabel(item.task.status) }}
                </el-tag>
                <span class="task-row-schedule">{{ scheduleLabel(item.task.schedule) }}</span>
                <span class="task-row-next">{{ formatDateTime(item.task.next_run_at) }}</span>
              </span>
            </button>
          </div>
          <div v-if="functionTaskItemCount > TASK_PAGE_SIZE" class="overview-pagination is-function">
            <el-pagination
              v-model:current-page="functionTaskPage"
              size="small"
              :page-size="TASK_PAGE_SIZE"
              :total="functionTaskItemCount"
              layout="prev, pager, next"
            />
          </div>
        </template>

        <el-empty
          v-else
          :description="t('scheduledTask.emptyFunctions')"
          :image-size="76"
          class="overview-empty"
        />
      </section>

      <section class="scheduled-panel">
        <div class="scheduled-panel-head">
          <div>
            <div class="scheduled-panel-title">
              <AgentEmployeeMascot
                class="scheduled-panel-agent-mark"
                variant="mark"
                :state="agentPresenceMascotState"
                label="数字员工"
              />
              数字员工
            </div>
            <div class="scheduled-panel-subtitle">{{ t('scheduledTask.agentOverviewSubtitle') }}</div>
          </div>
          <el-tag size="small" type="success">{{ scheduledAgentTotal }}</el-tag>
        </div>

        <template v-if="displayAgentTasks.length > 0">
          <div class="task-list">
            <button
              v-for="item in displayAgentTasks"
              :key="item.key"
              type="button"
              :class="['task-row', 'is-agent-row', { 'is-agent-working': !!item.task.inflight_execution_id }]"
              @click="openTask(item)"
            >
              <span class="task-row-agent-avatar">
                <AgentEmployeeMascot
                  variant="mark"
                  :state="agentEmployeeState(item.task)"
                  :label="`${item.task.title || t('scheduledTask.unnamedAgentTask')}，${agentPresenceStatus(item.task)}`"
                />
              </span>
              <span class="task-row-main">
                <span class="task-row-title">
                  {{ item.task.title || t('scheduledTask.unnamedAgentTask') }}
                  <el-tag v-if="item.builtin" size="small" type="info" effect="plain">服务目录内置</el-tag>
                  <el-tag v-else size="small" type="success" effect="plain">自定义</el-tag>
                  <el-tag
                    v-if="item.task.inflight_execution_id"
                    size="small"
                    type="primary"
                    effect="light"
                  >
                    <el-icon class="is-loading" style="margin-right: 4px"><Loading /></el-icon>{{ t('scheduledTask.running') }}
                  </el-tag>
                </span>
                <span class="task-row-path">{{ item.resourceName }} · {{ item.resourcePath }}</span>
                <span v-if="getAgentMessage(item.task)" class="task-row-message">
                  {{ getAgentMessage(item.task) }}
                </span>
                <span v-if="item.task.last_error_message" class="task-row-error">
                  {{ item.task.last_error_message }}
                </span>
              </span>
              <span class="task-row-side">
                <el-tag :type="taskStatusTag(item.task.status)" size="small">
                  {{ taskStatusLabel(item.task.status) }}
                </el-tag>
                <span class="task-row-schedule">{{ scheduleLabel(item.task.schedule) }}</span>
                <span class="task-row-next">{{ formatDateTime(item.task.next_run_at) }}</span>
              </span>
            </button>
          </div>
          <div v-if="agentTaskItemCount > TASK_PAGE_SIZE" class="overview-pagination is-agent">
            <el-pagination
              v-model:current-page="agentTaskPage"
              size="small"
              :page-size="TASK_PAGE_SIZE"
              :total="agentTaskItemCount"
              layout="prev, pager, next"
            />
          </div>
        </template>

        <el-empty
          v-else
          :description="t('scheduledTask.emptyAgents')"
          :image-size="76"
          class="overview-empty"
        />
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ArrowDown, ArrowUp, CaretRight, ChatLineRound, Clock, Document, Folder, Operation, Refresh, Timer, VideoPlay, Loading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { ServiceTree } from '@/architecture/domain/types'
import { getDirectoryOverview, type DirectoryOverviewResp, type DirectoryOverviewScheduledTask, type DirectoryOverviewStats } from '@/architecture/presentation/context/api/service-tree'
import { resumeTimerTask, type TimerTask } from '@/architecture/presentation/context/api/timer'
import AgentEmployeeMascot from './AgentEmployeeMascot.vue'
import {
  formatDateTime,
  scheduleLabel,
  taskStatusLabel,
  taskStatusTag,
} from './utils/timerSchedule'
import { buildScheduledExecutionRoute } from '@/architecture/shared/routing/platformRouteParams'

interface ScheduledOverviewItem extends DirectoryOverviewScheduledTask {
  key: string
  resourcePath: string
  resourceName: string
}

const props = defineProps<{
  packageNode: ServiceTree | null
}>()

const TASK_PAGE_SIZE = 8

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const loadSeq = ref(0)
const overview = ref<DirectoryOverviewResp | null>(null)
const errorMessage = ref('')
const functionTaskPage = ref(1)
const agentTaskPage = ref(1)
const startingAgentTaskIDs = ref<Set<number>>(new Set())
const agentRosterExpanded = ref(false)

const emptyStats: DirectoryOverviewStats = {
  directories: 0,
  functions: 0,
  docs: 0,
  total_run_count: 0,
  scheduled_function_tasks: 0,
  scheduled_agent_tasks: 0,
  running_tasks: 0,
  failed_tasks: 0,
  paused_tasks: 0,
}

const overviewStats = computed(() => overview.value?.stats || emptyStats)

const resourceStats = computed(() => ({
  directories: overviewStats.value.directories,
  functions: overviewStats.value.functions,
  docs: overviewStats.value.docs,
  totalRunCount: overviewStats.value.total_run_count,
}))

const scheduledFunctionTotal = computed(() => overviewStats.value.scheduled_function_tasks)
const scheduledAgentTotal = computed(() => overviewStats.value.scheduled_agent_tasks)
const runningTaskCount = computed(() => overviewStats.value.running_tasks)
const nextRunLabel = computed(() => formatDateTime(overviewStats.value.next_run_at))
const partialHint = computed(() => (overview.value?.warnings || []).join('；'))

const normalizedFunctionTasks = computed(() => {
  return (overview.value?.scheduled_function_tasks || [])
    .map(normalizeOverviewTask)
})

const normalizedAgentTasks = computed(() => {
  return (overview.value?.scheduled_agent_tasks || [])
    .map(normalizeOverviewTask)
})

const agentPresencePriority = {
  working: 0,
  failed: 1,
  ready: 2,
  paused: 3,
} as const

const orderedAgentTasks = computed(() => {
  return [...normalizedAgentTasks.value]
    .sort((left, right) => {
      return agentPresencePriority[agentEmployeeState(left.task)] - agentPresencePriority[agentEmployeeState(right.task)]
    })
})
const floatingAgentTasks = computed(() => {
  if (agentRosterExpanded.value) return orderedAgentTasks.value
  return orderedAgentTasks.value.slice(0, 3)
})
const hiddenAgentTaskCount = computed(() => Math.max(0, normalizedAgentTasks.value.length - 3))
const canExpandAgentRoster = computed(() => normalizedAgentTasks.value.length > 3)
const agentRunningCount = computed(() => {
  return normalizedAgentTasks.value.filter(item => Boolean(item.task.inflight_execution_id)).length
})
const agentFailedCount = computed(() => {
  return normalizedAgentTasks.value.filter(item => agentEmployeeState(item.task) === 'failed').length
})
const agentEnabledCount = computed(() => {
  return normalizedAgentTasks.value.filter(item => agentEmployeeState(item.task) === 'ready').length
})
const agentStartedCount = computed(() => {
  return normalizedAgentTasks.value.filter(item => isAgentTaskStarted(item.task)).length
})
const agentPresenceState = computed<'running' | 'enabled' | 'paused' | 'failed'>(() => {
  if (agentRunningCount.value > 0) return 'running'
  if (agentFailedCount.value > 0) return 'failed'
  if (agentEnabledCount.value > 0) return 'enabled'
  return 'paused'
})
const agentPresenceMascotState = computed<'working' | 'ready' | 'paused' | 'failed'>(() => {
  if (agentPresenceState.value === 'running') return 'working'
  if (agentPresenceState.value === 'enabled') return 'ready'
  return agentPresenceState.value
})
const agentPresenceSummary = computed(() => {
  const summary = [`已启动 ${agentStartedCount.value}/${normalizedAgentTasks.value.length}`]
  if (agentRunningCount.value > 0) summary.push(`${agentRunningCount.value} 名正在处理`)
  if (agentFailedCount.value > 0) summary.push(`${agentFailedCount.value} 名需关注`)
  return summary.join(' · ')
})

const functionTaskItemCount = computed(() => normalizedFunctionTasks.value.length)
const agentTaskItemCount = computed(() => normalizedAgentTasks.value.length)

const displayFunctionTasks = computed(() => {
  const start = (functionTaskPage.value - 1) * TASK_PAGE_SIZE
  return normalizedFunctionTasks.value.slice(start, start + TASK_PAGE_SIZE)
})

const displayAgentTasks = computed(() => {
  const start = (agentTaskPage.value - 1) * TASK_PAGE_SIZE
  return normalizedAgentTasks.value.slice(start, start + TASK_PAGE_SIZE)
})

function clampTaskPage(page: number, itemCount: number): number {
  const lastPage = Math.max(1, Math.ceil(itemCount / TASK_PAGE_SIZE))
  return Math.min(Math.max(1, page), lastPage)
}

function fallbackStatsFromTree(): DirectoryOverviewStats {
  const stats: DirectoryOverviewStats = {
    directories: 0,
    functions: 0,
    docs: 0,
    total_run_count: 0,
    scheduled_function_tasks: 0,
    scheduled_agent_tasks: 0,
    running_tasks: 0,
    failed_tasks: 0,
    paused_tasks: 0,
  }

  function walk(node?: ServiceTree | null) {
    if (!node?.children?.length) return
    for (const child of node.children) {
      if (child.type === 'package') {
        stats.directories += 1
        walk(child)
      } else if (child.type === 'function') {
        stats.functions += 1
        stats.total_run_count += Number(child.run_count || 0)
      } else if (child.type === 'docs') {
        stats.docs += 1
      }
    }
  }

  walk(props.packageNode)
  return stats
}

async function loadOverview() {
  const currentSeq = loadSeq.value + 1
  loadSeq.value = currentSeq
  errorMessage.value = ''
  if (!props.packageNode?.full_code_path) {
    overview.value = null
    return
  }

  loading.value = true
  try {
    const resp = await getDirectoryOverview(props.packageNode.full_code_path)
    if (loadSeq.value !== currentSeq) return
    overview.value = resp
  } catch (error) {
    if (loadSeq.value !== currentSeq) return
    errorMessage.value = error instanceof Error ? error.message : t('scheduledTask.overviewLoadFailed')
    overview.value = {
      stats: fallbackStatsFromTree(),
      scheduled_function_tasks: [],
      scheduled_agent_tasks: [],
      warnings: [],
      partial: true,
    }
  } finally {
    if (loadSeq.value === currentSeq) {
      loading.value = false
    }
  }
}

function normalizeOverviewTask(item: DirectoryOverviewScheduledTask): ScheduledOverviewItem {
  const resourcePath = item.resource_path || item.resource?.full_code_path || item.task.resource_key || ''
  return {
    ...item,
    key: `${item.kind}:${resourcePath}:${item.task.id}`,
    resourcePath,
    resourceName: item.resource_name || item.resource?.name || item.resource?.code || resourcePath || '-',
  }
}

function getAgentMessage(task: TimerTask): string {
	const payload = task.executor_payload
	if (payload && typeof payload === 'object') {
		const message = (payload as Record<string, unknown>).message
		if (typeof message === 'string' && message.trim()) return message.trim()
	}
	return typeof task.description === 'string' ? task.description.trim() : ''
}

function agentPresenceStatus(task: TimerTask): string {
  const state = agentEmployeeState(task)
  if (state === 'working') return '正在处理'
  if (state === 'failed') return '需要关注'
  if (state === 'ready') return '待命'
  return taskStatusLabel(task.status)
}

function agentEmployeeState(task: TimerTask): 'working' | 'ready' | 'paused' | 'failed' {
  if (task.inflight_execution_id) return 'working'
  if (task.status === 'failed' || Boolean(task.last_error_message?.trim())) return 'failed'
  if (task.status === 'pending') return 'ready'
  return 'paused'
}

function isAgentTaskStarted(task: TimerTask): boolean {
  return task.status === 'pending' || Boolean(task.inflight_execution_id)
}

function canStartAgentTask(task: TimerTask): boolean {
  return task.status === 'paused' && task.schedule?.type !== 'manual'
}

function agentTaskActivationLabel(task: TimerTask): string {
  if (isAgentTaskStarted(task)) return '已启动'
  if (task.schedule?.type === 'manual') return '待设置计划'
  if (canStartAgentTask(task)) return '未启动'
  return '已停止'
}

function isStartingAgentTask(task: TimerTask): boolean {
  return startingAgentTaskIDs.value.has(task.id)
}

async function startAgentTask(item: ScheduledOverviewItem) {
  if (!canStartAgentTask(item.task) || isStartingAgentTask(item.task)) return

  startingAgentTaskIDs.value = new Set(startingAgentTaskIDs.value).add(item.task.id)
  try {
    await resumeTimerTask(item.task.id)
    ElMessage.success(`已启动 ${item.task.title || t('scheduledTask.unnamedAgentTask')}`)
    await loadOverview()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '启动数字员工失败')
  } finally {
    const next = new Set(startingAgentTaskIDs.value)
    next.delete(item.task.id)
    startingAgentTaskIDs.value = next
  }
}

function openTask(item: ScheduledOverviewItem) {
  const fullCodePath = item.resourcePath || item.task.resource_key || ''
  if (!fullCodePath) return
  void router.replace(buildScheduledExecutionRoute({
    fullCodePath,
    kind: item.kind,
    taskId: item.task.id,
  }))
}

watch(
  () => props.packageNode?.full_code_path,
  () => {
    overview.value = null
    functionTaskPage.value = 1
    agentTaskPage.value = 1
    agentRosterExpanded.value = false
    void loadOverview()
  },
  { immediate: true }
)

watch(functionTaskItemCount, (itemCount) => {
  functionTaskPage.value = clampTaskPage(functionTaskPage.value, itemCount)
})

watch(agentTaskItemCount, (itemCount) => {
  agentTaskPage.value = clampTaskPage(agentTaskPage.value, itemCount)
})
</script>

<style scoped lang="scss">
.directory-overview {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-bottom: 22px;
}

.overview-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.overview-head-main {
  min-width: 0;
}

.agent-presence-float {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 18;
  display: grid;
  overflow: hidden;
  width: min(330px, calc(100vw - 32px));
  gap: 10px;
  padding: 12px;
  border: 1px solid color-mix(in srgb, #64748b 22%, transparent);
  border-radius: 16px;
  background: color-mix(in srgb, var(--el-bg-color) 90%, transparent);
  box-shadow:
    0 18px 48px rgba(15, 23, 42, 0.18),
    0 2px 8px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(16px);
}

.agent-presence-float.is-expanded {
  width: min(360px, calc(100vw - 32px));
  max-height: min(76vh, 680px);
}

.agent-presence-float.is-running {
  border-color: color-mix(in srgb, var(--el-color-primary) 42%, transparent);
  box-shadow:
    0 18px 48px rgba(37, 99, 235, 0.2),
    0 0 0 1px rgba(var(--el-color-primary-rgb), 0.08);
}

.agent-presence-float.is-failed {
  border-color: color-mix(in srgb, #f97316 45%, transparent);
  box-shadow:
    0 18px 48px rgba(194, 65, 12, 0.16),
    0 0 0 1px rgba(249, 115, 22, 0.08);
}

.agent-presence-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.agent-presence-title {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 700;
}

.agent-presence-live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #94a3b8;
  box-shadow: 0 0 0 4px rgba(148, 163, 184, 0.12);
}

.agent-presence-float.is-enabled .agent-presence-live-dot {
  background: #10b981;
  box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.12);
}

.agent-presence-float.is-running .agent-presence-live-dot {
  background: var(--el-color-primary);
  box-shadow: 0 0 0 4px rgba(var(--el-color-primary-rgb), 0.12);
  animation: agent-presence-live 1.1s ease-in-out infinite;
}

.agent-presence-float.is-failed .agent-presence-live-dot {
  background: #f97316;
  box-shadow: 0 0 0 4px rgba(249, 115, 22, 0.12);
  animation: agent-presence-alert 1.35s ease-in-out infinite;
}

.agent-presence-summary,
.agent-presence-more {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.agent-presence-team {
  display: grid;
  min-height: 0;
  gap: 6px;
}

.agent-presence-float.is-expanded .agent-presence-team {
  overflow-y: auto;
  max-height: min(60vh, 548px);
  padding-right: 4px;
  overscroll-behavior: contain;
  scrollbar-width: thin;
}

.agent-presence-employee {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  width: 100%;
  min-height: 60px;
  padding: 5px 6px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: inherit;
  text-align: left;
  transition: background-color 0.18s ease, transform 0.18s ease;
}

.agent-presence-employee:hover {
  background: var(--el-fill-color-light);
  transform: translateX(-2px);
}

.agent-presence-open {
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.agent-presence-avatar {
  display: inline-flex;
  width: 58px;
  height: 52px;
  align-items: center;
  justify-content: center;
}

.agent-presence-employee-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.agent-presence-employee-copy strong,
.agent-presence-employee-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-presence-employee-copy strong {
  color: var(--el-text-color-primary);
  font-size: 12px;
  font-weight: 650;
}

.agent-presence-employee-copy small {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.agent-presence-status-line {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.agent-presence-status-line small {
  min-width: 0;
}

.agent-presence-start-state {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
  color: #64748b;
  font-size: 10px;
  font-weight: 700;
}

.agent-presence-start-state::before {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #94a3b8;
  content: '';
}

.agent-presence-start-state.is-started {
  color: #059669;
}

.agent-presence-start-state.is-started::before {
  background: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.11);
}

.agent-presence-start {
  display: inline-flex;
  height: 28px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 0 9px 0 7px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 38%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
  transition: background-color 0.18s ease, border-color 0.18s ease, transform 0.18s ease;
}

.agent-presence-start:hover:not(:disabled) {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 16%, transparent);
  transform: translateY(-1px);
}

.agent-presence-start:disabled {
  cursor: wait;
  opacity: 0.65;
}

.agent-presence-start :deep(.el-icon) {
  font-size: 13px;
}

.agent-presence-more {
  display: inline-flex;
  min-height: 30px;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 4px 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 11px;
  cursor: pointer;
  transition: color 0.18s ease, background-color 0.18s ease, border-color 0.18s ease;
}

.agent-presence-more:hover {
  border-color: var(--el-border-color-light);
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}

.agent-presence-more :deep(.el-icon) {
  font-size: 12px;
}

@keyframes agent-presence-live {
  0%,
  100% {
    transform: scale(0.88);
    opacity: 0.72;
  }
  50% {
    transform: scale(1);
    opacity: 1;
  }
}

@keyframes agent-presence-alert {
  0%,
  100% {
    transform: scale(0.88);
    opacity: 0.68;
  }
  50% {
    transform: scale(1);
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .agent-presence-live-dot {
    animation: none !important;
  }
}

.overview-title {
  font-size: 20px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}

.overview-path {
  margin-top: 4px;
  font-family: Monaco, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.overview-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}

.metric-item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  min-height: 84px;
  padding: 14px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 8px;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 10px 24px rgba(15, 23, 42, 0.06));
}

.metric-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 38px;
  height: 38px;
  border-radius: 8px;
  font-size: 19px;
}

.metric-icon--directory {
  color: var(--el-color-primary);
  background: rgba(var(--el-color-primary-rgb), 0.12);
}

.metric-icon--function {
  color: var(--el-color-success);
  background: rgba(16, 185, 129, 0.12);
}

.metric-icon--docs {
  color: #0891b2;
  background: rgba(8, 145, 178, 0.12);
}

.metric-icon--task {
  color: var(--el-color-warning);
  background: rgba(245, 158, 11, 0.14);
}

.metric-icon--session {
  color: #2563eb;
  background: rgba(37, 99, 235, 0.12);
}

.metric-icon--running {
  color: #7c3aed;
  background: rgba(124, 58, 237, 0.12);
}

.metric-icon--next {
  color: var(--el-color-info);
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
}

.metric-content {
  min-width: 0;
}

.metric-label {
  font-size: 12px;
  line-height: 1.35;
  color: var(--el-text-color-secondary);
}

.metric-value {
  margin-top: 3px;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.25;
  color: var(--el-text-color-primary);
}

.metric-value--time {
  font-size: 13px;
  font-weight: 600;
  word-break: break-word;
}

.overview-alert {
  border-radius: 8px;
}

.scheduled-panels {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.scheduled-panel {
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 8px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--app-shell-panel-bg-strong, #fff) 84%, var(--app-shell-panel-muted-bg, #f1f5f9) 16%), var(--app-shell-panel-bg-strong, #fff));
}

.scheduled-panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.scheduled-panel-title {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}

.scheduled-panel-agent-mark {
  width: 23px;
  height: 23px;
  margin-right: 1px;
}

.scheduled-panel-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.overview-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.task-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  width: 100%;
  min-height: 86px;
  padding: 12px;
  border: 1px solid color-mix(in srgb, var(--app-shell-panel-border, #cbd5e1) 72%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-shell-panel-bg-strong, #fff) 90%, var(--app-shell-panel-muted-bg, #f1f5f9) 10%);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.task-row.is-agent-row {
  grid-template-columns: 32px minmax(0, 1fr) auto;
}

.task-row-agent-avatar {
  display: inline-grid;
  width: 32px;
  height: 32px;
  place-items: center;
  align-self: flex-start;
}

.task-row-agent-avatar :deep(.agent-employee-mascot) {
  width: 30px;
  height: 30px;
}

.task-row:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.34);
  background: color-mix(in srgb, var(--el-color-primary) 6%, var(--app-shell-panel-bg-strong, #fff));
  transform: translateY(-1px);
}

.task-row.is-agent-working {
  border-color: color-mix(in srgb, var(--el-color-primary) 34%, var(--directory-overview-line));
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--directory-overview-paper));
  animation: agent-task-working 1.2s ease-in-out infinite alternate;
}

@keyframes agent-task-working {
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(3px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .task-row.is-agent-working,
  .agent-presence-float.is-running .agent-presence-live-dot,
  .agent-presence-employee.is-working .agent-presence-avatar {
    animation: none;
  }
}

.task-row-main,
.task-row-side {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.task-row-main {
  gap: 5px;
}

.task-row-side {
  align-items: flex-end;
  gap: 5px;
}

.task-row-title {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.task-row-path,
.task-row-message,
.task-row-schedule,
.task-row-next {
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}

.task-row-path {
  font-family: Monaco, Menlo, Consolas, monospace;
  word-break: break-all;
}

.task-row-message {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.task-row-error {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-color-danger);
}

.task-row-schedule,
.task-row-next {
  text-align: right;
  white-space: nowrap;
}

.overview-empty {
  min-height: 180px;
  border: 1px dashed var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-shell-panel-bg-strong, #fff) 72%, var(--app-shell-panel-muted-bg, #f1f5f9) 28%);
}

@media (max-width: 1120px) {
  .scheduled-panels {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 960px) {
  .agent-presence-float {
    position: static;
    width: 100%;
    max-height: none;
  }

  .agent-presence-team {
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  }

  .agent-presence-float.is-expanded .agent-presence-team {
    max-height: min(58vh, 548px);
  }
}

@media (max-width: 720px) {
  .overview-head,
  .scheduled-panel-head {
    flex-direction: column;
    align-items: stretch;
  }

  .task-row {
    grid-template-columns: 1fr;
  }

  .task-row.is-agent-row {
    grid-template-columns: 32px minmax(0, 1fr);
  }

  .task-row.is-agent-row .task-row-side {
    grid-column: 2;
  }

  .task-row-side {
    align-items: flex-start;
  }

  .task-row-schedule,
  .task-row-next {
    text-align: left;
    white-space: normal;
  }
}
</style>
