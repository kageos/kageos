<template>
  <span
    class="tree-node"
    :class="[
      {
        'tree-node-draggable': draggable,
        'is-active': active,
      },
      showScheduledAgentBadge ? `agent-state-${scheduledAgentState}` : '',
    ]"
    :draggable="draggable"
    :title="title"
  >
    <img
      v-if="node.type === 'package'"
      src="/service-tree/custom-folder.svg"
      :alt="isRootNode(node) ? t('serviceTree.workspaceAlt') : t('serviceTree.directoryAlt')"
      class="node-icon package-icon-img"
      :class="nodeIconClass"
    />
    <template v-else-if="node.type === 'function'">
      <img
        v-if="node.template_type === TEMPLATE_TYPE.FORM"
        src="/service-tree/编辑.svg"
        :alt="t('serviceTree.formAlt')"
        class="node-icon form-icon-img"
        :class="nodeIconClass"
      />
      <el-icon
        v-else
        class="node-icon"
        :class="nodeIconClass"
      >
        <component :is="functionIcon" />
      </el-icon>
    </template>
    <img
      v-else-if="node.type === 'docs'"
      src="/文档.svg"
      :alt="t('serviceTree.docsAlt')"
      class="node-icon docs-icon-img"
      :class="nodeIconClass"
    />
    <span v-else class="node-icon fx-icon" :class="nodeIconClass">fx</span>

    <span class="node-label">{{ displayLabel }}</span>

    <button
      v-if="showAccessLock"
      type="button"
      :class="['access-lock-badge', { 'is-pending': accessRequestPending }]"
      :title="accessLockTitle"
      :aria-label="accessLockTitle"
      data-testid="service-tree-access-lock"
      @click.stop="emit('access-request-click')"
    >
      <el-icon><Clock v-if="accessRequestPending" /><Lock v-else /></el-icon>
    </button>

    <el-badge
      v-if="showPermissionRequestBadge"
      :value="permissionRequestBadgeValue"
      :max="99"
      :class="['permission-request-count-badge', permissionRequestBadgeClass]"
      :title="permissionRequestBadgeTitle"
      data-testid="service-tree-permission-request-badge"
      @click.stop="emit('permission-request-click')"
    />

    <el-badge
      v-if="showRuntimeBadge"
      :value="runtimeBadgeValue"
      :max="99"
      :class="['runtime-state-badge', runtimeBadgeClass]"
      :title="runtimeBadgeTitle"
    />

    <el-popover
      v-if="showScheduledAgentBadge"
      trigger="hover"
      placement="right-start"
      :width="360"
      :show-after="180"
      :hide-after="120"
      :teleported="true"
      popper-class="service-tree-agent-popover"
      @show="loadScheduledAgentDetails"
    >
      <template #reference>
        <span
          :class="['scheduled-agent-badge', `is-${scheduledAgentState}`]"
          title=""
          :aria-label="scheduledAgentBadgeTitle"
        >
          <AgentEmployeeMascot
            variant="mark"
            :state="scheduledAgentVisualState"
            :label="scheduledAgentBadgeTitle"
          />
        </span>
      </template>

      <section
        class="scheduled-agent-hover-card"
        :class="`is-${scheduledAgentState}`"
        data-testid="scheduled-agent-hover-card"
        :aria-label="`${displayLabel}的数字员工详情`"
      >
        <header class="scheduled-agent-hover-head">
          <span class="scheduled-agent-hover-avatar">
            <AgentEmployeeMascot
              variant="employee"
              :state="scheduledAgentVisualState"
              :label="scheduledAgentBadgeTitle"
            />
          </span>
          <span class="scheduled-agent-hover-heading">
            <strong>数字员工 · {{ displayLabel }}</strong>
            <small>{{ scheduledAgentBadgeTitle }}</small>
          </span>
        </header>

        <div class="scheduled-agent-hover-stats">
          <span>
            <strong>{{ scheduledAgentTotal }}</strong>
            <small>全部</small>
          </span>
          <span>
            <strong>{{ scheduledAgentEnabled }}</strong>
            <small>已启动</small>
          </span>
          <span>
            <strong>{{ scheduledAgentRunning }}</strong>
            <small>处理中</small>
          </span>
          <span :class="{ 'is-attention': scheduledAgentFailed > 0 }">
            <strong>{{ scheduledAgentFailed }}</strong>
            <small>需关注</small>
          </span>
        </div>

        <div v-if="scheduledAgentOwner" class="scheduled-agent-hover-owner">
          <span>负责人</span>
          <strong>{{ scheduledAgentOwner }}</strong>
        </div>

        <div v-if="scheduledAgentDetailsLoading" class="scheduled-agent-hover-loading">
          正在读取员工详情…
        </div>
        <div v-else-if="scheduledAgentDetailsError" class="scheduled-agent-hover-error">
          {{ scheduledAgentDetailsError }}
        </div>
        <div v-else-if="visibleScheduledAgentDetails.length > 0" class="scheduled-agent-hover-list">
          <article
            v-for="item in visibleScheduledAgentDetails"
            :key="scheduledAgentTaskKey(item)"
            :class="['scheduled-agent-hover-task', `is-${scheduledAgentTaskState(item)}`]"
          >
            <div class="scheduled-agent-hover-task-head">
              <AgentEmployeeMascot
                variant="mark"
                :state="scheduledAgentTaskState(item)"
                :label="`${scheduledAgentTaskTitle(item)}，${scheduledAgentTaskStatus(item)}`"
              />
              <strong>{{ scheduledAgentTaskTitle(item) }}</strong>
              <span>{{ scheduledAgentTaskStatus(item) }}</span>
            </div>
            <p>{{ scheduledAgentTaskPurpose(item) }}</p>
            <div class="scheduled-agent-hover-task-meta">
              <span>{{ item.resource_name || item.resource?.name || item.resource_path || '当前目录' }}</span>
              <span>{{ scheduleLabel(item.task.schedule) }}</span>
              <span v-if="item.task.next_run_at">下次 {{ formatDateTime(item.task.next_run_at) }}</span>
            </div>
            <div v-if="item.task.last_error_message" class="scheduled-agent-hover-task-error">
              {{ item.task.last_error_message }}
            </div>
          </article>
          <div v-if="hiddenScheduledAgentDetailCount > 0" class="scheduled-agent-hover-more">
            还有 {{ hiddenScheduledAgentDetailCount }} 名员工，请进入目录查看
          </div>
        </div>
        <div v-else class="scheduled-agent-hover-empty">
          暂未读取到员工明细
        </div>

        <div v-if="scheduledAgentDetailsWarning" class="scheduled-agent-hover-warning">
          {{ scheduledAgentDetailsWarning }}
        </div>

        <button
          type="button"
          class="scheduled-agent-hover-open"
          @click.stop="emit('scheduled-agent-click')"
        >
          查看全部数字员工
        </button>
      </section>
    </el-popover>

    <button
      v-if="showNotificationRouteBadge"
      type="button"
      :class="['notification-route-badge', `notification-route-badge-${notificationRouteBadgeTone}`]"
      :title="notificationRouteBadgeTitle"
      @click.stop="$emit('notification-route-click')"
    >
      <el-icon><BellFilled /></el-icon>
    </button>

    <el-badge
      v-if="showNotificationBadge"
      :value="notificationBadgeValue"
      :max="99"
      :class="['notification-count-badge', notificationBadgeClass]"
      :title="notificationBadgeTitle"
      @click.stop="$emit('notification-click')"
    />

    <slot name="actions" />
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { BellFilled, Clock, Document, Lock } from '@element-plus/icons-vue'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'
import FormIcon from '@/architecture/presentation/shared/components/icons/FormIcon.vue'
import AgentEmployeeMascot from './AgentEmployeeMascot.vue'
import type { ServiceTree } from '@/architecture/domain/types'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import { isRootNode } from '@/architecture/domain/utils/tree-utils'
import {
  getDirectoryOverview,
  type DirectoryOverviewScheduledTask,
} from '@/architecture/presentation/context/api/service-tree'
import { formatDateTime, scheduleLabel } from './utils/timerSchedule'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  node: ServiceTree
  label?: string
  title?: string
  draggable?: boolean
  active?: boolean
  showRuntimeBadge?: boolean
  runtimeBadgeValue?: string | number
  runtimeBadgeClass?: string
  runtimeBadgeTitle?: string
  showScheduledAgentBadge?: boolean
  scheduledAgentBadgeTitle?: string
  scheduledAgentState?: 'running' | 'enabled' | 'paused' | 'failed'
  showNotificationBadge?: boolean
  notificationBadgeValue?: string | number
  notificationBadgeClass?: string
  notificationBadgeTitle?: string
  showNotificationRouteBadge?: boolean
  notificationRouteBadgeTitle?: string
  notificationRouteBadgeTone?: string
  showAccessLock?: boolean
  accessRequestPending?: boolean
  accessLockTitle?: string
  showPermissionRequestBadge?: boolean
  permissionRequestBadgeValue?: string | number
  permissionRequestBadgeClass?: string
  permissionRequestBadgeTitle?: string
}>(), {
  label: '',
  title: '',
  draggable: false,
  active: false,
  showRuntimeBadge: false,
  runtimeBadgeValue: '',
  runtimeBadgeClass: '',
  runtimeBadgeTitle: '',
  showScheduledAgentBadge: false,
  scheduledAgentBadgeTitle: '',
  scheduledAgentState: 'paused',
  showNotificationBadge: false,
  notificationBadgeValue: '',
  notificationBadgeClass: '',
  notificationBadgeTitle: '',
  showNotificationRouteBadge: false,
  notificationRouteBadgeTitle: '',
  notificationRouteBadgeTone: 'direct',
  showAccessLock: false,
  accessRequestPending: false,
  accessLockTitle: '',
  showPermissionRequestBadge: false,
  permissionRequestBadgeValue: '',
  permissionRequestBadgeClass: '',
  permissionRequestBadgeTitle: '',
})

const emit = defineEmits<{
  (e: 'notification-click'): void
  (e: 'notification-route-click'): void
  (e: 'scheduled-agent-click'): void
  (e: 'access-request-click'): void
  (e: 'permission-request-click'): void
}>()

const scheduledAgentDetails = ref<DirectoryOverviewScheduledTask[]>([])
const scheduledAgentDetailsLoading = ref(false)
const scheduledAgentDetailsError = ref('')
const scheduledAgentDetailsWarning = ref('')
let scheduledAgentDetailsPath = ''
let scheduledAgentDetailsLoadedAt = 0
let scheduledAgentDetailsLoadSeq = 0

const SCHEDULED_AGENT_DETAIL_LIMIT = 4
const SCHEDULED_AGENT_DETAIL_CACHE_MS = 15_000

const displayLabel = computed(() => {
  return props.label || props.node.name || props.node.code || props.node.full_code_path || '-'
})

const scheduledAgentTotal = computed(() => Number(props.node.scheduled_agent_tasks || 0))
const scheduledAgentEnabled = computed(() => Number(props.node.enabled_agent_tasks || 0))
const scheduledAgentRunning = computed(() => Number(props.node.running_agent_tasks || 0))
const scheduledAgentFailed = computed(() => Number(props.node.failed_agent_tasks || 0))
const scheduledAgentOwner = computed(() => {
  return String(props.node.admins || props.node.owner || '')
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
    .join('、')
})

const scheduledAgentVisualState = computed<'working' | 'ready' | 'paused' | 'failed'>(() => {
  if (props.scheduledAgentState === 'running') return 'working'
  if (props.scheduledAgentState === 'failed') return 'failed'
  if (props.scheduledAgentState === 'enabled') return 'ready'
  return 'paused'
})

const orderedScheduledAgentDetails = computed(() => {
  const priority = {
    working: 0,
    failed: 1,
    ready: 2,
    paused: 3,
  } as const
  return [...scheduledAgentDetails.value].sort((left, right) => {
    return priority[scheduledAgentTaskState(left)] - priority[scheduledAgentTaskState(right)]
  })
})

const visibleScheduledAgentDetails = computed(() => {
  return orderedScheduledAgentDetails.value.slice(0, SCHEDULED_AGENT_DETAIL_LIMIT)
})

const hiddenScheduledAgentDetailCount = computed(() => {
  return Math.max(0, scheduledAgentDetails.value.length - SCHEDULED_AGENT_DETAIL_LIMIT)
})

async function loadScheduledAgentDetails() {
  const path = props.node.full_code_path || ''
  if (!path || scheduledAgentDetailsLoading.value) return
  if (
    scheduledAgentDetailsPath === path
    && Date.now() - scheduledAgentDetailsLoadedAt < SCHEDULED_AGENT_DETAIL_CACHE_MS
  ) {
    return
  }

  const loadSeq = scheduledAgentDetailsLoadSeq + 1
  scheduledAgentDetailsLoadSeq = loadSeq
  scheduledAgentDetailsLoading.value = true
  scheduledAgentDetailsError.value = ''
  scheduledAgentDetailsWarning.value = ''
  try {
    const overview = await getDirectoryOverview(path)
    if (loadSeq !== scheduledAgentDetailsLoadSeq || path !== props.node.full_code_path) return
    scheduledAgentDetails.value = (overview.scheduled_agent_tasks || [])
      .filter(item => item.kind === 'agent')
    scheduledAgentDetailsPath = path
    scheduledAgentDetailsLoadedAt = Date.now()
    if (overview.partial || (overview.warnings || []).length > 0) {
      scheduledAgentDetailsWarning.value = '部分员工详情暂不可用，进入目录可继续查看'
    }
  } catch {
    if (loadSeq !== scheduledAgentDetailsLoadSeq || path !== props.node.full_code_path) return
    scheduledAgentDetailsError.value = '员工详情加载失败，请进入目录重试'
  } finally {
    if (loadSeq === scheduledAgentDetailsLoadSeq) {
      scheduledAgentDetailsLoading.value = false
    }
  }
}

function scheduledAgentTaskState(item: DirectoryOverviewScheduledTask): 'working' | 'ready' | 'paused' | 'failed' {
  if (item.task.inflight_execution_id) return 'working'
  if (item.task.status === 'failed' || Boolean(item.task.last_error_message?.trim())) return 'failed'
  if (item.task.status === 'pending') return 'ready'
  return 'paused'
}

function scheduledAgentTaskStatus(item: DirectoryOverviewScheduledTask): string {
  const state = scheduledAgentTaskState(item)
  if (state === 'working') return '正在处理'
  if (state === 'failed') return '需要关注'
  if (state === 'ready') return '待命'
  return '未启动'
}

function scheduledAgentTaskTitle(item: DirectoryOverviewScheduledTask): string {
  return item.task.title?.trim() || '未命名数字员工'
}

function scheduledAgentTaskPurpose(item: DirectoryOverviewScheduledTask): string {
  const payload = item.task.executor_payload
  if (payload && typeof payload === 'object') {
    const message = (payload as Record<string, unknown>).message
    if (typeof message === 'string' && message.trim()) return message.trim()
  }
  return item.task.description?.trim() || '尚未填写工作说明'
}

function scheduledAgentTaskKey(item: DirectoryOverviewScheduledTask): string {
  return `${item.resource_path || item.task.resource_key || ''}:${item.task.id}`
}

watch(
  () => props.node.full_code_path,
  () => {
    scheduledAgentDetailsLoadSeq += 1
    scheduledAgentDetails.value = []
    scheduledAgentDetailsLoading.value = false
    scheduledAgentDetailsError.value = ''
    scheduledAgentDetailsWarning.value = ''
    scheduledAgentDetailsPath = ''
    scheduledAgentDetailsLoadedAt = 0
  }
)

const functionIcon = computed(() => {
  if (props.node.template_type === TEMPLATE_TYPE.TABLE) return TableIcon
  if (props.node.template_type === TEMPLATE_TYPE.FORM) return FormIcon
  if (props.node.template_type === TEMPLATE_TYPE.CHART) return ChartIcon
  return Document
})

const nodeIconClass = computed(() => {
  if (props.node.type === 'package') return 'package-icon'
  if (props.node.type === 'function') {
    if (props.node.template_type === TEMPLATE_TYPE.TABLE) return 'table-icon'
    if (props.node.template_type === TEMPLATE_TYPE.FORM) return 'form-icon'
    if (props.node.template_type === TEMPLATE_TYPE.CHART) return 'chart-icon'
    return 'function-icon'
  }
  if (props.node.type === 'docs') return 'docs-icon'
  return 'function-icon'
})
</script>

<style scoped>
.tree-node {
  --agent-accent: #64748b;
  --agent-badge-bg: rgba(100, 116, 139, 0.12);
  --agent-badge-border: rgba(100, 116, 139, 0.28);

  display: flex;
  width: 100%;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 8px;

  &.agent-state-running {
    --agent-accent: #2563eb;
    --agent-badge-bg: rgba(37, 99, 235, 0.16);
    --agent-badge-border: rgba(37, 99, 235, 0.38);
  }

  &.agent-state-enabled {
    --agent-accent: #059669;
    --agent-badge-bg: rgba(5, 150, 105, 0.15);
    --agent-badge-border: rgba(5, 150, 105, 0.34);
  }

  &.agent-state-failed {
    --agent-accent: #ea580c;
    --agent-badge-bg: rgba(234, 88, 12, 0.17);
    --agent-badge-border: rgba(234, 88, 12, 0.4);
  }

  &.agent-state-paused {
    --agent-accent: #64748b;
    --agent-badge-bg: rgba(100, 116, 139, 0.11);
    --agent-badge-border: rgba(100, 116, 139, 0.26);
  }

  &.tree-node-draggable {
    cursor: grab;

    &:active {
      cursor: grabbing;
    }
  }

  &.is-active {
    .node-label {
      color: var(--el-text-color-primary);
      font-weight: 500;
    }

    .node-icon {
      color: #6366f1;
      opacity: 0.9;
    }
  }
}

.node-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  margin-right: 8px;
  color: #6366f1;
  opacity: 0.8;
  transition: color 0.2s ease;

  &.package-icon {
    color: #6366f1;
    opacity: 0.8;
  }

  &.package-icon-img,
  &.form-icon-img,
  &.group-icon-img {
    width: 16px;
    height: 16px;
    object-fit: contain;
    opacity: 0.9;
  }

  &.table-icon {
    color: #10b981;
    opacity: 0.9;
  }

  &.form-icon {
    color: #3b82f6;
    opacity: 0.9;
  }

  &.function-icon {
    color: #6366f1;
    opacity: 0.8;
  }

  &.fx-icon {
    color: #6366f1;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
    font-size: 12px;
    font-style: italic;
    font-weight: 600;
    opacity: 0.8;
  }

  &.group-icon {
    color: #909399;
    opacity: 0.9;
  }
}

.node-label {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-state-badge {
  flex-shrink: 0;
  margin-left: 6px;
}

.runtime-state-badge :deep(.el-badge__content) {
  background: rgba(14, 165, 233, 0.12) !important;
  color: #0ea5e9 !important;
  border: 1px solid rgba(14, 165, 233, 0.25) !important;
  box-shadow: none !important;
  font-weight: 600 !important;
  padding: 0 6px !important;
  border-radius: 12px !important;
}

.runtime-state-badge-thinking :deep(.el-badge__content) {
  background: rgba(14, 165, 233, 0.12) !important;
  color: #0ea5e9 !important;
  border-color: rgba(14, 165, 233, 0.25) !important;
}

.runtime-state-badge-tool :deep(.el-badge__content) {
  background: rgba(245, 158, 11, 0.12) !important;
  color: #d97706 !important;
  border-color: rgba(245, 158, 11, 0.25) !important;
}

.runtime-state-badge-approval :deep(.el-badge__content) {
  background: rgba(168, 85, 247, 0.12) !important;
  color: #a855f7 !important;
  border-color: rgba(168, 85, 247, 0.25) !important;
}

.runtime-state-badge-failed :deep(.el-badge__content) {
  background: rgba(239, 68, 68, 0.12) !important;
  color: #ef4444 !important;
  border-color: rgba(239, 68, 68, 0.25) !important;
}

.scheduled-agent-badge {
  display: inline-grid;
  width: 25px;
  height: 25px;
  flex-shrink: 0;
  place-items: center;
  margin-left: 6px;
  border: 1px solid var(--agent-badge-border);
  border-radius: 8px;
  background: var(--agent-badge-bg);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--agent-accent) 18%, transparent);
  line-height: 1;
  transition:
    transform 0.2s ease,
    background-color 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.scheduled-agent-badge:hover {
  transform: scale(1.08);
}

.scheduled-agent-hover-card {
  --agent-hover-accent: #64748b;
  --agent-hover-soft: rgba(100, 116, 139, 0.1);
  display: grid;
  gap: 10px;
  color: var(--el-text-color-primary);
}

.scheduled-agent-hover-card.is-running {
  --agent-hover-accent: #2563eb;
  --agent-hover-soft: rgba(37, 99, 235, 0.09);
}

.scheduled-agent-hover-card.is-enabled {
  --agent-hover-accent: #059669;
  --agent-hover-soft: rgba(5, 150, 105, 0.09);
}

.scheduled-agent-hover-card.is-failed {
  --agent-hover-accent: #ea580c;
  --agent-hover-soft: rgba(234, 88, 12, 0.1);
}

.scheduled-agent-hover-head {
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
}

.scheduled-agent-hover-avatar {
  display: inline-grid;
  width: 58px;
  height: 52px;
  place-items: center;
}

.scheduled-agent-hover-heading {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.scheduled-agent-hover-heading strong {
  overflow: hidden;
  font-size: 14px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-agent-hover-heading small {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 1.45;
}

.scheduled-agent-hover-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.scheduled-agent-hover-stats > span {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 7px 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.scheduled-agent-hover-stats strong {
  color: var(--agent-hover-accent);
  font-size: 15px;
  line-height: 1;
}

.scheduled-agent-hover-stats small {
  color: var(--el-text-color-secondary);
  font-size: 10px;
}

.scheduled-agent-hover-stats .is-attention {
  border-color: rgba(234, 88, 12, 0.26);
  background: rgba(234, 88, 12, 0.08);
}

.scheduled-agent-hover-stats .is-attention strong {
  color: #ea580c;
}

.scheduled-agent-hover-owner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 2px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.scheduled-agent-hover-owner strong {
  overflow: hidden;
  color: var(--el-text-color-regular);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-agent-hover-loading,
.scheduled-agent-hover-empty,
.scheduled-agent-hover-error,
.scheduled-agent-hover-warning {
  padding: 9px 10px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 1.45;
}

.scheduled-agent-hover-error,
.scheduled-agent-hover-warning {
  background: rgba(234, 88, 12, 0.08);
  color: #c2410c;
}

.scheduled-agent-hover-list {
  display: grid;
  overflow-y: auto;
  max-height: 350px;
  gap: 7px;
  padding-right: 2px;
  scrollbar-width: thin;
}

.scheduled-agent-hover-task {
  display: grid;
  gap: 5px;
  padding: 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-left: 3px solid #94a3b8;
  border-radius: 8px;
  background: var(--el-bg-color);
}

.scheduled-agent-hover-task.is-working {
  border-left-color: #2563eb;
}

.scheduled-agent-hover-task.is-ready {
  border-left-color: #10b981;
}

.scheduled-agent-hover-task.is-failed {
  border-left-color: #ea580c;
}

.scheduled-agent-hover-task-head {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
}

.scheduled-agent-hover-task-head :deep(.agent-employee-mascot) {
  width: 24px;
  height: 24px;
}

.scheduled-agent-hover-task-head strong {
  overflow: hidden;
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-agent-hover-task-head > span:last-child {
  color: var(--el-text-color-secondary);
  font-size: 10px;
  font-weight: 600;
}

.scheduled-agent-hover-task.is-working .scheduled-agent-hover-task-head > span:last-child {
  color: #2563eb;
}

.scheduled-agent-hover-task.is-ready .scheduled-agent-hover-task-head > span:last-child {
  color: #059669;
}

.scheduled-agent-hover-task.is-failed .scheduled-agent-hover-task-head > span:last-child {
  color: #c2410c;
}

.scheduled-agent-hover-task p {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--el-text-color-regular);
  font-size: 11px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.scheduled-agent-hover-task-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  color: var(--el-text-color-secondary);
  font-size: 10px;
  line-height: 1.4;
}

.scheduled-agent-hover-task-error {
  display: -webkit-box;
  overflow: hidden;
  color: #c2410c;
  font-size: 10px;
  line-height: 1.4;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.scheduled-agent-hover-more {
  padding: 3px 4px 0;
  color: var(--el-text-color-secondary);
  font-size: 10px;
  text-align: center;
}

.scheduled-agent-hover-open {
  width: 100%;
  min-height: 30px;
  border: 1px solid color-mix(in srgb, var(--agent-hover-accent) 32%, transparent);
  border-radius: 8px;
  background: var(--agent-hover-soft);
  color: var(--agent-hover-accent);
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
  transition: border-color 0.18s ease, background-color 0.18s ease;
}

.scheduled-agent-hover-open:hover {
  border-color: var(--agent-hover-accent);
  background: color-mix(in srgb, var(--agent-hover-accent) 14%, transparent);
}

:global(.service-tree-agent-popover.el-popper) {
  max-width: calc(100vw - 24px);
  padding: 12px;
  border-color: var(--el-border-color-light);
  border-radius: 12px;
  box-shadow: 0 18px 44px rgba(15, 23, 42, 0.16);
}

.tree-node.agent-state-running .scheduled-agent-badge {
  animation: scheduled-agent-working 1.45s ease-in-out infinite;
}

.tree-node.agent-state-failed .scheduled-agent-badge {
  animation: scheduled-agent-attention 1.15s ease-in-out infinite;
}

@keyframes scheduled-agent-working {
  0%,
  100% {
    box-shadow: 0 2px 8px rgba(37, 99, 235, 0.16);
  }
  50% {
    box-shadow:
      0 2px 10px rgba(37, 99, 235, 0.26),
      0 0 0 3px rgba(37, 99, 235, 0.1);
  }
}

@keyframes scheduled-agent-attention {
  0%,
  100% {
    box-shadow: 0 2px 8px rgba(234, 88, 12, 0.17);
  }
  50% {
    box-shadow:
      0 2px 10px rgba(234, 88, 12, 0.3),
      0 0 0 3px rgba(234, 88, 12, 0.11);
  }
}

@media (prefers-reduced-motion: reduce) {
  .tree-node.agent-state-running .scheduled-agent-badge,
  .tree-node.agent-state-failed .scheduled-agent-badge {
    animation: none;
  }
}

.notification-route-badge {
  display: inline-flex;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  margin-left: 6px;
  padding: 0;
  border: 1px solid rgba(37, 99, 235, 0.22);
  border-radius: 6px;
  background: rgba(37, 99, 235, 0.08);
  color: #2563eb;
  cursor: pointer;
  transition: background-color 0.2s, border-color 0.2s, color 0.2s, transform 0.2s;
}

.access-lock-badge {
  display: inline-flex;
  width: 20px;
  height: 20px;
  flex: 0 0 20px;
  align-items: center;
  justify-content: center;
  margin-left: 6px;
  padding: 0;
  border: 1px solid rgba(239, 68, 68, 0.28);
  border-radius: 6px;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  cursor: pointer;
  transition: background-color 0.18s ease, border-color 0.18s ease, transform 0.18s ease;
}

.access-lock-badge:hover {
  border-color: rgba(239, 68, 68, 0.46);
  background: rgba(239, 68, 68, 0.17);
  transform: translateY(-1px);
}

.access-lock-badge.is-pending {
  border-color: rgba(245, 158, 11, 0.3);
  background: rgba(245, 158, 11, 0.12);
  color: #d97706;
}

.access-lock-badge :deep(.el-icon) {
  font-size: 13px;
}

.permission-request-count-badge {
  flex-shrink: 0;
  margin-left: 6px;
  cursor: pointer;
}

.permission-request-count-badge :deep(.el-badge__content) {
  border: none !important;
  background: #f59e0b !important;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1) !important;
  color: #fff !important;
  font-size: 11px !important;
  font-weight: 700 !important;
}

.permission-request-count-badge.needs-review :deep(.el-badge__content) {
  background: #ef4444 !important;
}

.notification-route-badge:hover {
  border-color: rgba(37, 99, 235, 0.36);
  background: rgba(37, 99, 235, 0.14);
  transform: translateY(-1px);
}

.notification-route-badge :deep(.el-icon) {
  width: 13px;
  height: 13px;
  font-size: 13px;
}

.notification-route-badge-inherited {
  border-color: rgba(100, 116, 139, 0.22);
  background: rgba(100, 116, 139, 0.08);
  color: #64748b;
}

.notification-route-badge-inherited:hover {
  border-color: rgba(100, 116, 139, 0.34);
  background: rgba(100, 116, 139, 0.14);
}

.notification-route-badge-disabled {
  border-color: rgba(148, 163, 184, 0.22);
  background: rgba(148, 163, 184, 0.08);
  color: #94a3b8;
}

.notification-route-badge-failed {
  border-color: rgba(239, 68, 68, 0.28);
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.notification-route-badge-failed:hover {
  border-color: rgba(239, 68, 68, 0.42);
  background: rgba(239, 68, 68, 0.16);
}

.notification-count-badge {
  flex-shrink: 0;
  margin-left: 6px;
  cursor: pointer;
}

.notification-count-badge :deep(.el-badge__content) {
  background: #ef4444 !important;
  color: #ffffff !important;
  border: none !important;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1) !important;
  font-weight: 600 !important;
  font-size: 11px !important;
  padding: 0 6px !important;
  height: 16px !important;
  line-height: 16px !important;
  border-radius: 8px !important;
  transition: background-color 0.2s, transform 0.2s !important;
  animation: none !important;
}

.notification-count-badge:hover :deep(.el-badge__content) {
  background: #f87171 !important;
  transform: scale(1.05) !important;
}

.notification-count-badge.is-history :deep(.el-badge__content) {
  background: rgba(148, 163, 184, 0.15) !important;
  color: #94a3b8 !important;
  border: 1px solid rgba(148, 163, 184, 0.2) !important;
  box-shadow: none !important;
}

.notification-count-badge.is-history:hover :deep(.el-badge__content) {
  background: rgba(148, 163, 184, 0.25) !important;
  transform: scale(1.05) !important;
}

:slotted(.node-more-actions) {
  flex-shrink: 0;
  margin-left: auto;
  opacity: 0;
  transition: opacity 0.2s;
}

.tree-node:hover :slotted(.node-more-actions),
.tree-node.is-active :slotted(.node-more-actions) {
  opacity: 1;
}
</style>
