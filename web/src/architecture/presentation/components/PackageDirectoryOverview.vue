<template>
  <section v-if="packageNode" class="directory-overview" v-loading="loading">
    <div class="overview-head">
      <div class="overview-head-main">
        <div class="overview-title">目录概览</div>
        <div class="overview-path">{{ packageNode.full_code_path || '-' }}</div>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadOverview">
        刷新
      </el-button>
    </div>

    <div class="overview-metrics">
      <div class="metric-item">
        <div class="metric-icon metric-icon--directory">
          <el-icon><Folder /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">子目录</div>
          <div class="metric-value">{{ resourceStats.directories }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--function">
          <el-icon><Operation /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">函数</div>
          <div class="metric-value">{{ resourceStats.functions }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--docs">
          <el-icon><Document /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">文档</div>
          <div class="metric-value">{{ resourceStats.docs }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--task">
          <el-icon><Timer /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">定时函数</div>
          <div class="metric-value">{{ scheduledFunctionTotal }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--session">
          <el-icon><ChatLineRound /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">定时会话</div>
          <div class="metric-value">{{ scheduledAgentTotal }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--running">
          <el-icon><VideoPlay /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">执行中</div>
          <div class="metric-value">{{ runningTaskCount }}</div>
        </div>
      </div>

      <div class="metric-item">
        <div class="metric-icon metric-icon--next">
          <el-icon><Clock /></el-icon>
        </div>
        <div class="metric-content">
          <div class="metric-label">最近触发</div>
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
              定时函数
            </div>
            <div class="scheduled-panel-subtitle">当前目录及子目录内的函数定时配置</div>
          </div>
          <el-tag size="small" type="primary">{{ scheduledFunctionTotal }}</el-tag>
        </div>

        <div v-if="displayFunctionTasks.length > 0" class="task-list">
          <button
            v-for="item in displayFunctionTasks"
            :key="item.key"
            type="button"
            class="task-row"
            @click="openTask(item)"
          >
            <span class="task-row-main">
              <span class="task-row-title">
                {{ item.task.title || '未命名定时函数' }}
                <el-tag
                  v-if="item.task.inflight_execution_id"
                  size="small"
                  type="primary"
                  effect="light"
                >
                  执行中
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

        <el-empty
          v-else
          description="暂无定时函数"
          :image-size="76"
          class="overview-empty"
        />
      </section>

      <section class="scheduled-panel">
        <div class="scheduled-panel-head">
          <div>
            <div class="scheduled-panel-title">
              <el-icon><ChatLineRound /></el-icon>
              定时会话
            </div>
            <div class="scheduled-panel-subtitle">当前目录及子目录内的工作台定时会话</div>
          </div>
          <el-tag size="small" type="success">{{ scheduledAgentTotal }}</el-tag>
        </div>

        <div v-if="displayAgentTasks.length > 0" class="task-list">
          <button
            v-for="item in displayAgentTasks"
            :key="item.key"
            type="button"
            class="task-row"
            @click="openTask(item)"
          >
            <span class="task-row-main">
              <span class="task-row-title">
                {{ item.task.title || '未命名定时会话' }}
                <el-tag
                  v-if="item.task.inflight_execution_id"
                  size="small"
                  type="primary"
                  effect="light"
                >
                  执行中
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

        <el-empty
          v-else
          description="暂无定时会话"
          :image-size="76"
          class="overview-empty"
        />
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ChatLineRound, Clock, Document, Folder, Operation, Refresh, Timer, VideoPlay } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/architecture/domain/types'
import { getDirectoryOverview, type DirectoryOverviewResp, type DirectoryOverviewScheduledTask, type DirectoryOverviewStats } from '@/architecture/presentation/context/api/service-tree'
import type { TimerTask } from '@/architecture/presentation/context/api/timer'
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

const TASK_DISPLAY_LIMIT = 8

const router = useRouter()
const loading = ref(false)
const loadSeq = ref(0)
const overview = ref<DirectoryOverviewResp | null>(null)
const errorMessage = ref('')

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

const displayFunctionTasks = computed(() => {
  return (overview.value?.scheduled_function_tasks || [])
    .map(normalizeOverviewTask)
    .slice(0, TASK_DISPLAY_LIMIT)
})

const displayAgentTasks = computed(() => {
  return (overview.value?.scheduled_agent_tasks || [])
    .map(normalizeOverviewTask)
    .slice(0, TASK_DISPLAY_LIMIT)
})

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
    errorMessage.value = error instanceof Error ? error.message : '目录概览加载失败'
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
  if (!payload || typeof payload !== 'object') return ''
  const message = (payload as Record<string, unknown>).message
  return typeof message === 'string' ? message.trim() : ''
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
    void loadOverview()
  },
  { immediate: true }
)
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

.task-row:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.34);
  background: color-mix(in srgb, var(--el-color-primary) 6%, var(--app-shell-panel-bg-strong, #fff));
  transform: translateY(-1px);
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

@media (max-width: 720px) {
  .overview-head,
  .scheduled-panel-head {
    flex-direction: column;
    align-items: stretch;
  }

  .task-row {
    grid-template-columns: 1fr;
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
