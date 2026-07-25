<template>
  <aside class="detail-aside">
    <section class="detail-aside-card">
      <div class="detail-aside-card-head">
        <div class="detail-aside-title">{{ t('scheduledTask.agentDetailTitle') }}</div>
        <el-tag :type="taskStatusTag(task.status)" effect="light">
          {{ taskStatusLabel(task.status) }}
        </el-tag>
      </div>
      <div class="detail-aside-name">{{ task.title || t('scheduledTask.unnamedAgentTask') }}</div>
      <div class="detail-aside-path">{{ workspacePath || '-' }}</div>
      <div class="detail-aside-actions">
        <el-tooltip :content="t('scheduledTask.runNow')" placement="top" effect="light">
          <el-button
            type="primary"
            :icon="VideoPlay"
            :disabled="inlineEditing || isTerminal(task.status)"
            @click="emit('run-now', task)"
          />
        </el-tooltip>
        <el-tooltip
          :content="task.status === 'paused' ? t('scheduledTask.resume') : t('scheduledTask.pause')"
          placement="top"
          effect="light"
        >
          <el-button
            :type="task.status === 'paused' ? 'primary' : 'warning'"
            :icon="task.status === 'paused' ? CaretRight : VideoPause"
            :disabled="inlineEditing || isTerminal(task.status)"
            @click="task.status === 'paused' ? emit('resume', task) : emit('pause', task)"
          />
        </el-tooltip>
        <el-tooltip :content="t('scheduledTask.cancel')" placement="top" effect="light">
          <el-button
            type="danger"
            :icon="Close"
            :disabled="inlineEditing || isTerminal(task.status)"
            @click="emit('cancel', task)"
          />
        </el-tooltip>
        <el-tooltip :content="t('scheduledTask.delete')" placement="top" effect="light">
          <el-button
            type="danger"
            plain
            :icon="Delete"
            :disabled="inlineEditing || !!task.inflight_execution_id"
            @click="emit('delete', task)"
          />
        </el-tooltip>
      </div>
      <el-alert
        v-if="isTaskPaused(task)"
        class="detail-enable-hint"
        type="info"
        show-icon
        :closable="false"
        :title="t('scheduledTask.enableForUnattendedHint')"
      />
    </section>

    <section class="detail-aside-card">
      <div class="detail-aside-title">{{ t('scheduledTask.schedule') }}</div>
      <div v-if="!inlineEditing" class="detail-property-list">
        <div class="detail-property">
          <span>{{ t('scheduledTask.schedule') }}</span>
          <strong>{{ scheduleLabel(task.schedule) }}</strong>
        </div>
        <div class="detail-property">
          <span>{{ t('scheduledTask.nextRun') }}</span>
          <strong>{{ formatDateTime(task.next_run_at) }}</strong>
        </div>
        <div class="detail-property">
          <span>{{ t('scheduledTask.runCount') }}</span>
          <strong>{{ task.run_count || 0 }}</strong>
        </div>
        <div class="detail-property">
          <span>{{ t('scheduledTask.overlapPolicy') }}</span>
          <strong>{{ overlapPolicyLabel(task.overlap_policy) }}</strong>
        </div>
        <div v-if="task.overlap_policy === 'allow'" class="detail-property">
          <span>{{ t('scheduledTask.maxParallelism') }}</span>
          <strong>{{ task.max_parallelism || 1 }}</strong>
        </div>
        <div class="detail-property">
          <span>{{ t('scheduledTask.agentModel') }}</span>
          <strong>{{ llmConfigLabel }}</strong>
        </div>
        <div class="detail-property">
          <span>{{ t('scheduledTask.createdBy') }}</span>
          <strong>{{ task.created_by || task.request_user || '-' }}</strong>
        </div>
      </div>

      <el-form v-else class="detail-schedule-form" label-position="top">
        <el-form-item :label="t('scheduledTask.scheduleType')" required>
          <el-radio-group
            :model-value="scheduleType"
            class="detail-schedule-type"
            @update:model-value="updateScheduleType"
          >
            <el-radio-button value="atime">{{ t('scheduledTask.scheduleAtime') }}</el-radio-button>
            <el-radio-button value="cron">{{ t('scheduledTask.scheduleCron') }}</el-radio-button>
            <el-radio-button value="every">{{ t('scheduledTask.scheduleEvery') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="scheduleType === 'atime'" :label="t('scheduledTask.runAt')" required>
          <el-date-picker
            :model-value="runAt"
            type="datetime"
            :placeholder="t('scheduledTask.runAtPlaceholder')"
            format="YYYY-MM-DD HH:mm"
            value-format="YYYY-MM-DD HH:mm:ss"
            :shortcuts="dateTimeShortcuts"
            style="width: 100%"
            @update:model-value="updateRunAt"
          />
        </el-form-item>

        <el-form-item v-if="scheduleType === 'cron'" :label="t('scheduledTask.cron')" required>
          <el-input :model-value="cronExpr" placeholder="0 9 * * *" @update:model-value="updateCronExpr" />
        </el-form-item>

        <el-form-item v-if="scheduleType === 'every'" :label="t('scheduledTask.intervalSeconds')" required>
          <el-input-number
            :model-value="intervalSeconds"
            :min="1"
            :max="86400"
            style="width: 100%"
            @update:model-value="updateIntervalSeconds"
          />
        </el-form-item>

        <el-form-item v-if="scheduleType === 'every'" :label="t('scheduledTask.maxRuns')">
          <el-input-number
            :model-value="maxRuns"
            :min="0"
            :max="1000000"
            style="width: 100%"
            @update:model-value="updateMaxRuns"
          />
        </el-form-item>

        <el-form-item :label="t('scheduledTask.overlapPolicy')">
          <el-select :model-value="overlapPolicy" style="width: 100%" @update:model-value="updateOverlapPolicy">
            <el-option :label="t('scheduledTask.overlapForbid')" value="forbid" />
            <el-option :label="t('scheduledTask.overlapQueueLatest')" value="queue_latest" />
            <el-option :label="t('scheduledTask.overlapAllow')" value="allow" />
          </el-select>
          <div class="detail-inline-hint">{{ t(`scheduledTask.overlapHint_${overlapPolicy}`) }}</div>
        </el-form-item>

        <el-form-item v-if="overlapPolicy === 'allow'" :label="t('scheduledTask.maxParallelism')">
          <el-input-number
            :model-value="maxParallelism"
            :min="1"
            :max="16"
            style="width: 100%"
            @update:model-value="updateMaxParallelism"
          />
        </el-form-item>
      </el-form>
    </section>

    <el-alert
      v-if="task.last_error_message"
      class="detail-alert"
      type="error"
      show-icon
      :closable="false"
      :title="task.last_error_message"
    />
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CaretRight, Close, Delete, VideoPause, VideoPlay } from '@element-plus/icons-vue'
import type { TimerOverlapPolicy, TimerScheduleType, TimerTask } from '@/architecture/presentation/context/api/timer'
import { createRelativeDateTimeShortcuts } from '@/architecture/shared/date'
import {
  formatDateTime,
  scheduleLabel,
  taskStatusLabel,
  taskStatusTag,
} from './utils/timerSchedule'

defineProps<{
  task: TimerTask
  inlineEditing: boolean
  workspacePath: string
  llmConfigLabel: string
  scheduleType: TimerScheduleType
  runAt: string
  cronExpr: string
  intervalSeconds: number
  maxRuns: number
  overlapPolicy: TimerOverlapPolicy
  maxParallelism: number
}>()

const emit = defineEmits<{
  (e: 'run-now', task: TimerTask): void
  (e: 'pause', task: TimerTask): void
  (e: 'resume', task: TimerTask): void
  (e: 'cancel', task: TimerTask): void
  (e: 'delete', task: TimerTask): void
  (e: 'update:scheduleType', value: TimerScheduleType): void
  (e: 'update:runAt', value: string): void
  (e: 'update:cronExpr', value: string): void
  (e: 'update:intervalSeconds', value: number): void
  (e: 'update:maxRuns', value: number): void
  (e: 'update:overlapPolicy', value: TimerOverlapPolicy): void
  (e: 'update:maxParallelism', value: number): void
}>()

const { t } = useI18n()
const dateTimeShortcuts = computed(() => createRelativeDateTimeShortcuts())

function isTaskPaused(task?: TimerTask | null): boolean {
  return task?.status === 'paused'
}

function isTerminal(status: string): boolean {
  return ['done', 'failed', 'cancelled'].includes(status)
}

function updateScheduleType(value: string | number | boolean | undefined) {
  if (value === 'atime' || value === 'cron' || value === 'every') {
    emit('update:scheduleType', value)
  }
}

function updateRunAt(value: string | number | Date | null | undefined) {
  emit('update:runAt', value instanceof Date ? formatDateInput(value) : String(value || ''))
}

function updateCronExpr(value: string | number | null | undefined) {
  emit('update:cronExpr', String(value || ''))
}

function updateIntervalSeconds(value: number | null | undefined) {
  emit('update:intervalSeconds', Number(value || 0))
}

function updateMaxRuns(value: number | null | undefined) {
  emit('update:maxRuns', Number(value || 0))
}

function updateOverlapPolicy(value: string | number | boolean | undefined) {
  if (value === 'forbid' || value === 'queue_latest' || value === 'allow') {
    emit('update:overlapPolicy', value)
  }
}

function updateMaxParallelism(value: number | null | undefined) {
  emit('update:maxParallelism', Number(value || 1))
}

function overlapPolicyLabel(policy?: TimerOverlapPolicy): string {
  if (policy === 'queue_latest') return t('scheduledTask.overlapQueueLatest')
  if (policy === 'allow') return t('scheduledTask.overlapAllow')
  return t('scheduledTask.overlapForbid')
}

function formatDateInput(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0')
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
  ].join('-') + ' ' + [
    pad(date.getHours()),
    pad(date.getMinutes()),
    pad(date.getSeconds()),
  ].join(':')
}
</script>

<style scoped lang="scss">
.detail-aside {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-left: 1px solid var(--scheduled-session-line);
  background: color-mix(in srgb, var(--scheduled-session-tint) 52%, var(--app-shell-panel-bg, var(--el-bg-color)));
}

.detail-aside-card {
  min-width: 0;
  padding: 14px;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-line) 74%, transparent);
  border-radius: 8px;
  background: var(--scheduled-session-paper);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.7));
}

.detail-aside-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.detail-aside-title {
  color: var(--scheduled-session-muted);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.45;
}

.detail-aside-name {
  margin-top: 10px;
  color: var(--scheduled-session-ink);
  font-size: 16px;
  font-weight: 760;
  line-height: 1.35;
  word-break: break-word;
}

.detail-aside-path {
  margin-top: 8px;
  color: var(--scheduled-session-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
  word-break: break-all;
}

.detail-aside-actions {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 14px;
}

.detail-aside-actions :deep(.el-button) {
  width: 100%;
  height: 34px;
  padding: 0;
  margin: 0;
}

.detail-enable-hint {
  margin-top: 12px;
}

.detail-property-list {
  display: grid;
  gap: 8px;
  margin-top: 12px;
}

.detail-property {
  min-width: 0;
  padding: 10px 0;
  border-top: 1px solid color-mix(in srgb, var(--scheduled-session-line) 62%, transparent);
}

.detail-property:first-child {
  border-top: 0;
  padding-top: 0;
}

.detail-property span {
  display: block;
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.45;
}

.detail-property strong {
  display: block;
  margin-top: 5px;
  color: var(--scheduled-session-ink);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
  word-break: break-word;
}

.detail-schedule-form {
  margin-top: 12px;
}

.detail-schedule-form :deep(.el-form-item__label) {
  color: var(--scheduled-session-ink);
  font-weight: 650;
}

.detail-schedule-form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.detail-schedule-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}

.detail-inline-hint {
  margin-top: 6px;
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.5;
}

.detail-schedule-type {
  display: grid;
  width: 100%;
  gap: 6px;
}

.detail-schedule-type :deep(.el-radio-button__inner) {
  width: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.detail-schedule-type :deep(.el-radio-button:first-child .el-radio-button__inner),
.detail-schedule-type :deep(.el-radio-button:last-child .el-radio-button__inner) {
  border-radius: 8px;
}

.detail-alert {
  flex-shrink: 0;
}

@media (max-width: 860px) {
  .detail-aside {
    overflow: visible;
    border-top: 1px solid var(--scheduled-session-line);
    border-left: 0;
  }
}

@media (max-width: 768px) {
  .detail-aside {
    padding: 12px;
  }
}
</style>
