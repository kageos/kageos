<template>
  <div class="right-sidebar-session-panel" data-testid="workspace-sidebar-sessions">
    <div class="right-session-header" data-testid="workspace-sidebar-sessions-header">
      <el-icon :size="16" color="var(--el-color-primary)"><FolderOpened /></el-icon>
      <span class="right-session-dir">{{ dirName }}</span>
    </div>

    <div class="right-session-tabs" data-testid="workspace-sidebar-sessions-tabs">
      <div :class="['right-tab', { active: activeTab === 'all' }]" data-testid="workspace-sidebar-tab-all" @click="updateActiveTab('all')">
        全部
      </div>
      <div :class="['right-tab', { active: activeTab === 'running' }]" data-testid="workspace-sidebar-tab-running" @click="updateActiveTab('running')">
        执行中
        <span v-if="runningCount > 0" class="right-tab-badge">{{ runningCount }}</span>
      </div>
      <div :class="['right-tab', { active: activeTab === 'finished' }]" data-testid="workspace-sidebar-tab-finished" @click="updateActiveTab('finished')">
        已结束
      </div>
      <div :class="['right-tab', { active: activeTab === 'scheduled' }]" data-testid="workspace-sidebar-tab-scheduled" @click="updateActiveTab('scheduled')">
        定时
        <span v-if="scheduledCount > 0" class="right-tab-badge right-tab-badge-primary">{{ scheduledCount }}</span>
      </div>
    </div>

    <el-input
      :model-value="searchKeyword"
      class="right-session-search"
      :placeholder="activeTab === 'scheduled' ? '搜索定时会话…' : '搜索会话…'"
      clearable
      :prefix-icon="Search"
      data-testid="workspace-sidebar-search"
      @update:model-value="updateSearchKeyword"
    />

    <div
      v-if="activeTab === 'scheduled'"
      class="right-session-list"
      data-testid="workspace-sidebar-scheduled-list"
      v-loading="scheduledLoading"
    >
      <div class="scheduled-section">
        <div class="scheduled-section-title">
          <span>定时任务</span>
          <div class="scheduled-section-title-actions">
            <span>{{ scheduledTasks.length }}</span>
            <el-button link size="small" type="primary" @click="$emit('manage-scheduled-tasks')">
              管理
            </el-button>
          </div>
        </div>
        <div
          v-for="task in scheduledTasks"
          :key="task.id"
          class="right-session-card scheduled-task-card"
          @click="toggleScheduledTaskRecords(task.id)"
        >
          <div class="right-session-card-head">
            <el-icon :size="12" color="var(--el-color-primary)"><Timer /></el-icon>
            <span class="right-session-card-title">{{ task.name || '未命名定时会话' }}</span>
            <el-button
              class="scheduled-card-expand"
              text
              size="small"
              @click.stop="toggleScheduledTaskRecords(task.id)"
            >
              {{ isScheduledTaskExpanded(task.id) ? '收起' : '记录' }}
            </el-button>
          </div>

          <div class="scheduled-task-goal">{{ task.goal }}</div>

          <div class="right-session-card-meta">
            <el-tag :type="scheduledStatusTag(task.status)" size="small" effect="plain">
              {{ scheduledStatusLabel(task.status) }}
            </el-tag>
            <span class="right-session-time">{{ nextRunLabel(task.next_run_at) }}</span>
          </div>

          <div class="scheduled-task-foot">
            <span>{{ scheduleTypeLabel(task.schedule_type) }}</span>
            <span>{{ task.run_count }} 次</span>
          </div>

          <div class="scheduled-task-actions" @click.stop>
            <el-button
              v-if="task.last_session_id"
              link
              size="small"
              type="primary"
              @click="$emit('open-scheduled-session', task)"
            >
              最近会话
            </el-button>
            <el-button
              link
              size="small"
              type="primary"
              :loading="scheduledTaskActionId === task.id"
              @click="$emit('run-scheduled-task-now', task)"
            >
              立即执行
            </el-button>
          </div>

          <div v-if="isScheduledTaskExpanded(task.id)" class="scheduled-task-records" @click.stop>
            <div
              v-for="record in getScheduledTaskRecords(task.id)"
              :key="`${record.task.id}-${record.execution.id}`"
              :class="['scheduled-task-record', { disabled: !record.execution.session_id }]"
              @click="$emit('open-scheduled-execution', record)"
            >
              <div class="scheduled-task-record-head">
                <el-tag :type="executionStatusTag(record.execution.status)" size="small" effect="plain">
                  {{ executionStatusLabel(record.execution.status) }}
                </el-tag>
                <span>{{ formatRelativeTime(record.execution.started_at || record.execution.scheduled_at || record.execution.created_at) }}</span>
              </div>
              <div class="scheduled-task-record-timestamp">
                {{ formatTimestamp(record.execution.started_at || record.execution.scheduled_at || record.execution.created_at) }}
              </div>
              <div class="scheduled-task-record-summary">
                {{ record.execution.output_summary || record.execution.error_message || record.execution.input_goal || '暂无摘要' }}
              </div>
            </div>
            <div v-if="getScheduledTaskRecords(task.id).length === 0" class="scheduled-task-record-empty">
              暂无该任务的执行会话记录
            </div>
          </div>
        </div>
        <div v-if="scheduledTasks.length === 0 && !scheduledLoading" class="scheduled-section-empty">
          暂无定时会话
        </div>
      </div>

      <div class="scheduled-section">
        <div class="scheduled-section-title">
          <span>执行记录</span>
          <span>{{ scheduledExecutions.length }}</span>
        </div>
        <div
          v-for="record in scheduledExecutions"
          :key="`${record.task.id}-${record.execution.id}`"
          :class="['right-session-card', 'scheduled-execution-card', { disabled: !record.execution.session_id }]"
          @click="$emit('open-scheduled-execution', record)"
        >
          <div class="right-session-card-head">
            <el-icon
              v-if="record.execution.status === 'running'"
              class="is-loading"
              :size="12"
              color="var(--el-color-primary)"
            >
              <Loading />
            </el-icon>
            <el-icon v-else :size="12" color="var(--el-color-primary)"><Timer /></el-icon>
            <span class="right-session-card-title">{{ record.task.name || '未命名定时会话' }}</span>
          </div>

          <div class="scheduled-task-goal">
            {{ record.execution.output_summary || record.execution.error_message || record.task.goal }}
          </div>

          <div class="right-session-card-meta">
            <el-tag :type="executionStatusTag(record.execution.status)" size="small" effect="plain">
              {{ executionStatusLabel(record.execution.status) }}
            </el-tag>
            <span class="right-session-time">
              {{ formatRelativeTime(record.execution.started_at || record.execution.scheduled_at || record.execution.created_at) }}
            </span>
          </div>
          <div class="right-session-card-timestamp">
            {{ formatTimestamp(record.execution.started_at || record.execution.scheduled_at || record.execution.created_at) }}
          </div>

          <div class="scheduled-task-foot">
            <span>{{ record.execution.tool_call_count }} 次工具调用</span>
            <span v-if="record.execution.session_id">打开会话</span>
            <span v-else>无会话</span>
          </div>
        </div>
        <div v-if="scheduledExecutions.length === 0 && !scheduledLoading" class="scheduled-section-empty">
          暂无执行记录
        </div>
      </div>

      <div v-if="scheduledTasks.length === 0 && scheduledExecutions.length === 0 && !scheduledLoading" class="right-session-empty">
        <el-empty :description="emptyDescription" :image-size="48" />
      </div>
    </div>

    <div v-else class="right-session-list" data-testid="workspace-sidebar-list" v-loading="loading">
      <div
        v-for="session in sessions"
        :key="session.session_id"
        :class="['right-session-card', { generating: session.status === 'generating' }]"
        :data-testid="`workspace-session-card-${session.session_id}`"
        @click="$emit('open-session', session)"
      >
        <div class="right-session-card-head">
          <el-icon
            v-if="session.status === 'generating'"
            class="is-loading"
            :size="12"
            color="var(--el-color-primary)"
          >
            <Loading />
          </el-icon>
          <span class="right-session-card-title">{{ session.title || '未命名会话' }}</span>
        </div>

        <div v-if="session.user" class="right-session-card-user">
          <UserDisplay :username="session.user" mode="simple" size="small" />
        </div>

        <div class="right-session-card-meta">
          <el-tag v-if="session.status === 'generating'" type="primary" size="small" effect="light">执行中</el-tag>
          <el-tag v-else-if="session.status === 'done'" type="success" size="small" effect="plain">已完成</el-tag>
          <el-tag v-else-if="session.status === 'cancelled'" type="info" size="small" effect="plain">已取消</el-tag>
          <span class="right-session-time">{{ formatRelativeTime(session.updated_at) }}</span>
        </div>
        <div class="right-session-card-timestamp">
          {{ formatTimestamp(session.updated_at || session.created_at) }}
        </div>

        <div v-if="session.status === 'generating'" class="right-session-card-actions">
          <el-button
            size="small"
            link
            type="danger"
            :loading="cancellingTaskId === session.session_id"
            :data-testid="`workspace-session-stop-${session.session_id}`"
            @click.stop="$emit('cancel-task', session)"
          >
            停止
          </el-button>
        </div>
      </div>

      <div v-if="sessions.length === 0 && !loading" class="right-session-empty">
        <el-empty :description="emptyDescription" :image-size="48" />
      </div>
    </div>

    <div class="right-session-footer" data-testid="workspace-sidebar-footer">
      <el-button type="primary" :icon="ChatDotRound" class="right-new-session-btn" data-testid="workspace-sidebar-create-session" @click="$emit('create-session')">
        新增会话
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChatDotRound, FolderOpened, Loading, Search, Timer } from '@element-plus/icons-vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import type { WorkspaceSessionItem } from '@/api/workspace'
import type { ScheduledAgentTaskItem } from '@/api/scheduledAgentTask'
import type { ScheduledAgentExecutionRecord } from '@/architecture/presentation/composables/useWorkspaceSidebarSessions'

type SidebarTab = 'all' | 'running' | 'finished' | 'scheduled'

const props = defineProps<{
  dirName: string
  loading: boolean
  scheduledLoading: boolean
  activeTab: SidebarTab
  searchKeyword: string
  runningCount: number
  scheduledCount: number
  sessions: WorkspaceSessionItem[]
  scheduledTasks: ScheduledAgentTaskItem[]
  scheduledExecutions: ScheduledAgentExecutionRecord[]
  cancellingTaskId: string | null
  scheduledTaskActionId: number | null
  formatRelativeTime: (timeStr: string) => string
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', value: SidebarTab): void
  (e: 'update:searchKeyword', value: string): void
  (e: 'open-session', session: WorkspaceSessionItem): void
  (e: 'open-scheduled-session', task: ScheduledAgentTaskItem): void
  (e: 'open-scheduled-execution', record: ScheduledAgentExecutionRecord): void
  (e: 'run-scheduled-task-now', task: ScheduledAgentTaskItem): void
  (e: 'manage-scheduled-tasks'): void
  (e: 'cancel-task', session: WorkspaceSessionItem): void
  (e: 'create-session'): void
}>()

const expandedScheduledTaskIds = ref<Set<number>>(new Set())

const emptyDescription = computed(() => {
  if (props.searchKeyword) return '无匹配会话'
  if (props.activeTab === 'running') return '暂无执行中的会话'
  if (props.activeTab === 'finished') return '暂无已结束的会话'
  if (props.activeTab === 'scheduled') return '暂无定时会话记录'
  return '暂无会话记录'
})

function updateActiveTab(value: SidebarTab) {
  emit('update:activeTab', value)
}

function updateSearchKeyword(value: string | number) {
  emit('update:searchKeyword', String(value ?? ''))
}

function isScheduledTaskExpanded(taskId: number): boolean {
  return expandedScheduledTaskIds.value.has(taskId)
}

function toggleScheduledTaskRecords(taskId: number) {
  const next = new Set(expandedScheduledTaskIds.value)
  if (next.has(taskId)) {
    next.delete(taskId)
  } else {
    next.add(taskId)
  }
  expandedScheduledTaskIds.value = next
}

function getScheduledTaskRecords(taskId: number): ScheduledAgentExecutionRecord[] {
  return props.scheduledExecutions.filter((record) => record.task.id === taskId)
}

function scheduledStatusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: '待执行',
    paused: '已暂停',
    done: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return map[status] || status || '-'
}

function scheduledStatusTag(status: string): 'success' | 'warning' | 'info' | 'primary' | 'danger' {
  const map: Record<string, 'success' | 'warning' | 'info' | 'primary' | 'danger'> = {
    pending: 'primary',
    paused: 'warning',
    done: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return map[status] || 'info'
}

function executionStatusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: '待执行',
    running: '运行中',
    success: '成功',
    failed: '失败',
    timeout: '超时',
    cancelled: '已取消'
  }
  return map[status] || status || '-'
}

function executionStatusTag(status: string): 'success' | 'warning' | 'info' | 'primary' | 'danger' {
  const map: Record<string, 'success' | 'warning' | 'info' | 'primary' | 'danger'> = {
    pending: 'info',
    running: 'primary',
    success: 'success',
    failed: 'danger',
    timeout: 'warning',
    cancelled: 'info'
  }
  return map[status] || 'info'
}

function scheduleTypeLabel(scheduleType: string): string {
  const map: Record<string, string> = {
    atime: '一次',
    cron: 'Cron',
    every: '循环'
  }
  return map[scheduleType] || scheduleType || '-'
}

function nextRunLabel(value?: string): string {
  if (!value) return '暂无下次执行'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '暂无下次执行'
  return `下次 ${date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })}`
}

function formatTimestamp(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  const y = date.getFullYear()
  const M = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const m = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${M}-${d} ${h}:${m}:${s}`
}
</script>

<style scoped lang="scss">
.right-sidebar-session-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  padding: 10px 0 0;
}

.right-session-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 14px 10px;
  padding: 14px 14px 12px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 18px;
  background: var(--app-shell-panel-bg-strong);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
  flex-shrink: 0;
}

.right-session-dir {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.right-session-tabs {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px 2px;
  flex-shrink: 0;
}

.right-tab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 13px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  border: 1px solid transparent;
  border-radius: 999px;
  transition: all 0.2s ease;
  white-space: nowrap;

  &:hover {
    color: var(--el-color-primary);
    background: rgba(var(--el-color-primary-rgb), 0.06);
  }

  &.active {
    color: var(--el-color-primary);
    font-weight: 600;
    border-color: rgba(var(--el-color-primary-rgb), 0.16);
    background: rgba(var(--el-color-primary-rgb), 0.1);
    box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
  }
}

.right-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  margin-left: 4px;
  font-size: 10px;
  line-height: 1;
  color: #fff;
  background: var(--el-color-danger);
  border-radius: 8px;
}

.right-tab-badge-primary {
  background: var(--el-color-primary);
}

.right-session-search {
  flex-shrink: 0;
  padding: 6px 14px 4px;
}

.right-session-search :deep(.el-input__wrapper) {
  border-radius: 14px;
  padding: 4px 12px;
  background: var(--app-shell-panel-bg-strong);
  border: 1px solid var(--app-shell-panel-border);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
}

.right-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 10px 14px 18px;
}

.right-session-card {
  padding: 14px 14px 12px;
  margin-bottom: 10px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 20px;
  background: var(--app-shell-panel-bg-strong);
  box-shadow: 0 16px 32px rgba(15, 23, 42, 0.07);
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--el-color-primary);
    background: var(--app-shell-panel-bg-strong);
    box-shadow: 0 20px 36px rgba(15, 23, 42, 0.11);
    transform: translateY(-2px);
  }

  &.generating {
    border-left: 3px solid var(--el-color-primary);
  }
}

.right-session-card.disabled {
  cursor: default;
}

.scheduled-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.scheduled-section + .scheduled-section {
  margin-top: 14px;
}

.scheduled-section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 3px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
}

.scheduled-section-title-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.scheduled-section-title-actions :deep(.el-button) {
  margin-left: 0;
  padding: 0;
  font-size: 12px;
}

.scheduled-section-empty {
  padding: 10px 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-align: center;
}

.right-session-card-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.right-session-card-user {
  margin-bottom: 8px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.right-session-card-user :deep(.user-display-wrapper) {
  display: inline-flex;
}

.right-session-card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.scheduled-card-expand {
  flex-shrink: 0;
  padding: 0 2px;
  font-size: 11px;
}

.scheduled-task-goal {
  display: -webkit-box;
  margin-top: 6px;
  overflow: hidden;
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.45;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.scheduled-task-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.scheduled-task-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px 8px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed var(--app-shell-panel-border);
}

.scheduled-task-actions :deep(.el-button) {
  margin-left: 0;
  font-size: 11px;
}

.scheduled-task-records {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 10px;
  padding: 8px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 14px;
  background: rgba(var(--el-color-primary-rgb), 0.035);
}

.scheduled-task-record {
  padding: 8px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 12px;
  background: var(--app-shell-panel-bg);
  cursor: pointer;
  transition: all 0.18s ease;

  &:hover {
    border-color: var(--el-color-primary);
    transform: translateY(-1px);
  }

  &.disabled {
    cursor: default;
    opacity: 0.72;
  }
}

.scheduled-task-record-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--el-text-color-placeholder);
  font-size: 11px;
}

.scheduled-task-record-timestamp,
.right-session-card-timestamp {
  margin-top: 4px;
  color: var(--el-text-color-placeholder);
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 10px;
  letter-spacing: 0.02em;
}

.scheduled-task-record-summary {
  display: -webkit-box;
  margin-top: 6px;
  overflow: hidden;
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.45;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.scheduled-task-record-empty {
  padding: 8px 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-align: center;
}

.right-session-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.right-session-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.right-session-card-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.right-session-empty {
  padding: 32px 8px;
  text-align: center;
}

.right-session-footer {
  flex-shrink: 0;
  padding: 12px 14px 16px;
  border-top: 1px solid var(--app-shell-panel-border);
  background: var(--app-shell-panel-bg);
}

.right-new-session-btn {
  width: 100%;
  height: 44px;
  border-radius: 16px;
  box-shadow: 0 14px 30px rgba(var(--el-color-primary-rgb), 0.2);
}
</style>
