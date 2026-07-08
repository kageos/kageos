<template>
  <section class="detail-document-section is-executions">
    <div class="detail-section-head">
      <div>
        <div class="detail-section-title">{{ t('scheduledTask.executionRecords') }}</div>
        <div class="detail-section-subtitle">{{ t('scheduledTask.executionRecordsHint') }}</div>
      </div>
      <div class="drawer-section-controls">
        <el-select
          :model-value="state.status"
          :placeholder="t('scheduledTask.allStatuses')"
          clearable
          size="small"
          style="width: 120px"
          @change="handleStatusChange"
        >
          <el-option :label="t('scheduledTask.allStatuses')" value="" />
          <el-option :label="t('scheduledTask.executionStatusQueued')" value="queued" />
          <el-option :label="t('scheduledTask.executionStatusRunning')" value="running" />
          <el-option :label="t('scheduledTask.executionStatusSuccess')" value="success" />
          <el-option :label="t('scheduledTask.executionStatusFailed')" value="failed" />
          <el-option :label="t('scheduledTask.executionStatusTimeout')" value="timeout" />
        </el-select>
        <el-button size="small" :icon="Refresh" @click="emit('refresh')">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <div v-if="state.loading" v-loading="true" class="drawer-executions-loading" />

    <el-alert
      v-else-if="state.error"
      :title="state.error"
      type="error"
      show-icon
      :closable="false"
    />

    <el-empty
      v-else-if="state.loaded && state.list.length === 0"
      :description="t('scheduledTask.emptyExecutions')"
      :image-size="56"
    />

    <div v-else-if="state.loaded" class="execution-timeline">
      <article
        v-for="execution in state.list"
        :key="execution.id"
        :class="['execution-card', executionCardClass(execution), { 'is-focused': execution.id === focusedExecutionId }]"
      >
        <div class="execution-card-rail" />
        <div class="execution-card-main">
          <div class="execution-card-head">
            <div class="execution-card-title-line">
              <el-tag :type="executionStatusTag(execution.status)" size="small" effect="light">
                {{ executionStatusLabel(execution.status) }}
              </el-tag>
              <span class="execution-trigger">{{ executionTriggerLabel(execution) }}</span>
            </div>
            <el-button
              v-if="getExecutionOpenSessionID(execution)"
              link
              type="primary"
              size="small"
              class="execution-open-session"
              @click="openExecutionSession(execution)"
            >
              {{ t('scheduledTask.openSession') }}
            </el-button>
          </div>

          <div class="execution-time">{{ formatDateTime(execution.scheduled_at) }}</div>

          <div class="execution-facts">
            <span v-if="getExecutionOpenSessionID(execution)">
              {{ t('scheduledTask.session', { id: shortSessionID(getExecutionOpenSessionID(execution)) }) }}
            </span>
            <span>{{ executionToolStats(execution) }}</span>
            <span v-if="execution.duration_millis">{{ formatDuration(execution.duration_millis) }}</span>
          </div>

          <div v-if="executionHumanSummary(execution)" class="execution-summary">
            {{ executionHumanSummary(execution) }}
          </div>

          <div v-if="executionErrorMessage(execution)" class="execution-error-card">
            <div class="execution-error-title">{{ executionErrorTitle(execution) }}</div>
            <div class="execution-error-hint">{{ executionErrorHint(execution) }}</div>
            <div class="execution-error-detail">{{ executionErrorMessage(execution) }}</div>
          </div>
        </div>
      </article>
    </div>

    <div v-if="state.loaded && state.total > state.pageSize" class="execution-pagination">
      <el-pagination
        small
        :current-page="state.page"
        :page-size="state.pageSize"
        :total="state.total"
        layout="prev, pager, next"
        @current-change="emit('page-change', $event)"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { TimerExecution } from '@/architecture/presentation/context/api/timer'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import {
  executionStatusLabel,
  executionStatusTag,
  formatDateTime,
  formatDuration,
} from './utils/timerSchedule'

interface ScheduledAgentExecutionState {
  loading: boolean
  loaded: boolean
  error: string
  status: string
  page: number
  pageSize: number
  total: number
  list: TimerExecution[]
}

const props = withDefaults(defineProps<{
  state: ScheduledAgentExecutionState
  focusedExecutionId?: number
  workspacePath?: string
}>(), {
  focusedExecutionId: 0,
  workspacePath: '',
})

const emit = defineEmits<{
  (e: 'status-change', value: string): void
  (e: 'refresh'): void
  (e: 'page-change', value: number): void
}>()

const { t } = useI18n()

function handleStatusChange(value: string | number | boolean | null | undefined) {
  emit('status-change', typeof value === 'string' ? value : String(value || ''))
}

function getExecutionSessionID(execution: TimerExecution): string {
  const payload = execution.result_payload
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>
    const sessionID = record.session_id || record.sessionId
    return typeof sessionID === 'string' ? sessionID : ''
  }
  return ''
}

function executionTriggerLabel(execution: TimerExecution): string {
  return execution.trigger_type === 'manual' ? t('scheduledTask.manualTrigger') : t('scheduledTask.scheduledTrigger')
}

function shortSessionID(sessionID: string): string {
  const value = sessionID.trim()
  if (value.length <= 12) return value
  return `${value.slice(0, 8)}...${value.slice(-4)}`
}

function executionCardClass(execution: TimerExecution): string {
  return `is-${execution.status || 'unknown'}`
}

function executionToolStats(execution: TimerExecution): string {
  const summary = execution.output_summary || ''
  const match = summary.match(/工具调用\s*(\d+)\s*次，失败\s*(\d+)\s*次/)
  if (match) {
    return t('scheduledTask.toolsSummary', { toolCalls: match[1], failures: match[2] })
  }

  const payload = execution.result_payload
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>
    const toolCalls = record.tool_calls || record.toolCalls
    if (typeof toolCalls === 'number') return t('scheduledTask.toolsCount', { toolCalls })
  }
  return t('scheduledTask.toolsZero')
}

function executionHumanSummary(execution: TimerExecution): string {
  return (execution.output_summary || '')
    .split('；')
    .map((item) => item.trim())
    .filter((item) => item && !item.startsWith('session_id=') && !/^工具调用\s*\d+\s*次/.test(item))
    .join('；')
}

function executionErrorMessage(execution: TimerExecution): string {
  return (execution.error_message || '')
    .replace(/^业务错误\s*\[\d+\]:\s*/, '')
    .trim()
}

function executionErrorTitle(execution: TimerExecution): string {
  const message = executionErrorMessage(execution)
  if (message.includes('服务目录不存在')) return t('scheduledTask.directoryMissingTitle')
  if (message.includes('timeout') || execution.status === 'timeout') return t('scheduledTask.timeoutTitle')
  if (message.includes('权限')) return t('scheduledTask.permissionFailedTitle')
  return t('scheduledTask.executionError')
}

function executionErrorHint(execution: TimerExecution): string {
  const message = executionErrorMessage(execution)
  if (message.includes('服务目录不存在')) {
    return props.workspacePath
      ? t('scheduledTask.directoryMissingHint', { path: props.workspacePath })
      : t('scheduledTask.workspaceMissingHint')
  }
  if (message.includes('权限')) return t('scheduledTask.permissionHint')
  return t('scheduledTask.genericErrorHint')
}

function getExecutionOpenSessionID(execution: TimerExecution): string {
  return getExecutionSessionID(execution) || (execution.executor_run_id || '').trim()
}

function getWorkspaceName(fullCodePath: string): string {
  const parts = fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || fullCodePath || t('scheduledTask.workspaceFallback')
}

function openExecutionSession(execution: TimerExecution) {
  const sessionID = getExecutionOpenSessionID(execution)
  if (!sessionID) {
    ElMessage.warning(t('scheduledTask.noOpenSession'))
    return
  }
  const fullCodePath = props.workspacePath.trim()
  if (!fullCodePath) {
    ElMessage.warning(t('scheduledTask.missingSessionPath'))
    return
  }

  eventBus.emit('workspace:open-workstation', {
    full_code_path: fullCodePath,
    session_id: sessionID,
    directory_name: getWorkspaceName(fullCodePath),
    initial_maximized: true,
    open_as_mini: true,
  })
}
</script>

<style scoped>
.detail-document-section {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--scheduled-session-line);
}

.detail-document-section.is-executions {
  padding-bottom: 4px;
}

.detail-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.detail-section-title {
  color: var(--scheduled-session-ink);
  font-size: 15px;
  font-weight: 740;
  line-height: 1.35;
}

.detail-section-subtitle {
  margin-top: 4px;
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.45;
}

.drawer-section-controls {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.drawer-executions-loading {
  min-height: 120px;
}

.execution-timeline {
  display: grid;
  gap: 10px;
}

.execution-card {
  position: relative;
  display: grid;
  grid-template-columns: 3px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-line) 60%, transparent);
  border-radius: 10px;
  background: var(--scheduled-session-paper);
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.execution-card.is-focused {
  border-color: var(--scheduled-session-accent);
  box-shadow: 0 0 0 2px rgba(var(--el-color-primary-rgb), 0.16);
}

.execution-card:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.28);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.06);
}

.execution-card-rail {
  background: var(--el-color-info);
  border-radius: 3px 0 0 3px;
}

.execution-card.is-success .execution-card-rail {
  background: var(--el-color-success);
}

.execution-card.is-failed .execution-card-rail,
.execution-card.is-timeout .execution-card-rail {
  background: var(--el-color-danger);
}

.execution-card.is-running .execution-card-rail,
.execution-card.is-queued .execution-card-rail {
  background: var(--el-color-primary);
}

.execution-card-main {
  min-width: 0;
  padding: 12px 14px 14px;
}

.execution-card-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.execution-card-title-line {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.execution-trigger {
  font-size: 12px;
  color: var(--scheduled-session-muted);
}

.execution-time {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.execution-open-session {
  flex-shrink: 0;
  padding: 0;
}

.execution-facts {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 10px;
}

.execution-facts span {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  color: var(--scheduled-session-muted);
  background: var(--scheduled-session-tint);
  border: 1px solid color-mix(in srgb, var(--scheduled-session-line) 54%, transparent);
}

.execution-summary {
  margin-top: 10px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}

.execution-error-card {
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--el-color-danger) 24%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-color-danger) 6%, #fff);
}

.execution-error-title {
  font-size: 13px;
  font-weight: 650;
  color: var(--el-color-danger);
}

.execution-error-hint {
  margin-top: 4px;
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.5;
}

.execution-error-detail {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed rgba(245, 108, 108, 0.28);
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.execution-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
  padding: 0;
}

@media (max-width: 860px) {
  .detail-section-head {
    flex-direction: column;
  }
}
</style>
