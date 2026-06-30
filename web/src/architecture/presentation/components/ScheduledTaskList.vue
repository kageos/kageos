<template>
  <div class="scheduled-task-list" v-loading="loading">
    <div class="scheduled-list-header">
      <div>
        <div class="scheduled-list-title">{{ t('scheduledTask.functionTitle') }}</div>
        <div class="scheduled-list-subtitle">{{ resourcePath ? t('scheduledTask.currentFunction') : t('scheduledTask.noFunctionSelected') }}</div>
      </div>
      <div class="scheduled-list-actions">
        <span class="scheduled-total">{{ t('scheduledTask.totalCount', { count: total }) }}</span>
        <el-button :icon="Refresh" @click="loadList">{{ t('common.refresh') }}</el-button>
        <el-button type="primary" :icon="Plus" :disabled="!resourcePath" @click="showCreateDialog = true">
          {{ t('scheduledTask.newTask') }}
        </el-button>
      </div>
    </div>

    <div class="scheduled-agent-workspace">
      <aside class="scheduled-agent-sidebar">
        <div class="scheduled-list-filter">
          <div class="scheduled-sidebar-title">{{ t('scheduledTask.sessionList') }}</div>
          <el-select v-model="statusFilter" :placeholder="t('scheduledTask.allStatuses')" clearable size="small" @change="loadList">
            <el-option :label="t('scheduledTask.allStatuses')" value="" />
            <el-option :label="t('scheduledTask.taskStatusPending')" value="pending" />
            <el-option :label="t('scheduledTask.taskStatusPaused')" value="paused" />
            <el-option :label="t('scheduledTask.taskStatusDone')" value="done" />
            <el-option :label="t('scheduledTask.taskStatusFailed')" value="failed" />
            <el-option :label="t('scheduledTask.taskStatusCancelled')" value="cancelled" />
          </el-select>
        </div>

        <el-empty
          v-if="!loading && list.length === 0"
          :description="t('scheduledTask.emptyFunctions')"
          :image-size="72"
          class="scheduled-sidebar-empty"
        />

        <div v-else class="agent-session-list" role="list">
          <button
            v-for="task in list"
            :key="task.id"
            type="button"
            :class="[
              'agent-session-item',
              `is-${task.status || 'unknown'}`,
              { 'is-active': selectedTask?.id === task.id, 'is-running': !!task.inflight_execution_id }
            ]"
            @click="selectTask(task)"
          >
            <span class="agent-session-item-head">
              <span class="agent-session-item-title">
                {{ task.title || t('scheduledTask.unnamedFunctionTask') }}
              </span>
              <el-tag :type="taskStatusTag(task.status)" size="small" effect="light">
                {{ taskStatusLabel(task.status) }}
              </el-tag>
            </span>
            <span class="agent-session-item-summary is-mono">
              {{ task.resource_key || resourcePath || '-' }}
            </span>
            <span v-if="task.last_error_message" class="agent-session-item-error">
              {{ task.last_error_message }}
            </span>
            <span class="agent-session-item-meta">
              <span>{{ scheduleLabel(task.schedule) }}</span>
              <span>{{ t('scheduledTask.nextRun') }} {{ formatDateTime(task.next_run_at) }}</span>
              <span>{{ t('scheduledTask.runCount') }} {{ task.run_count || 0 }}</span>
            </span>
          </button>
        </div>

        <div v-if="total > pageSize" class="scheduled-pagination">
          <el-pagination
            v-model:current-page="page"
            small
            :page-size="pageSize"
            :total="total"
            layout="prev, pager, next"
            @current-change="loadList"
          />
        </div>
      </aside>

      <main class="scheduled-agent-detail">
        <el-empty
          v-if="!selectedTask"
          :description="t('scheduledTask.selectFunctionHint')"
          :image-size="96"
          class="scheduled-detail-empty"
        />

        <template v-else>
          <div class="scheduled-detail-shell">
            <section class="detail-document">
              <div class="detail-document-toolbar">
                <div>
                  <div class="detail-kicker">{{ t('scheduledTask.functionTitle') }}</div>
                  <div class="detail-toolbar-title">{{ selectedTask.title || t('scheduledTask.unnamedFunctionTask') }}</div>
                </div>
              </div>

              <section class="detail-document-section is-executions">
                <div class="detail-section-head">
                  <div>
                    <div class="detail-section-title">{{ t('scheduledTask.executionRecords') }}</div>
                    <div class="detail-section-subtitle">{{ t('scheduledTask.executionRecordsHint') }}</div>
                  </div>
                  <div class="drawer-section-controls">
                    <el-select
                      v-model="selectedExecutionState.status"
                      :placeholder="t('scheduledTask.allStatuses')"
                      clearable
                      size="small"
                      style="width: 120px"
                      @change="loadSelectedExecutions(true)"
                    >
                      <el-option :label="t('scheduledTask.allStatuses')" value="" />
                      <el-option :label="t('scheduledTask.executionStatusQueued')" value="queued" />
                      <el-option :label="t('scheduledTask.executionStatusRunning')" value="running" />
                      <el-option :label="t('scheduledTask.executionStatusSuccess')" value="success" />
                      <el-option :label="t('scheduledTask.executionStatusFailed')" value="failed" />
                      <el-option :label="t('scheduledTask.executionStatusTimeout')" value="timeout" />
                    </el-select>
                    <el-button size="small" :icon="Refresh" @click="loadSelectedExecutions(true)">{{ t('common.refresh') }}</el-button>
                  </div>
                </div>

                <div v-if="selectedExecutionState.loading" v-loading="true" class="drawer-executions-loading" />

                <el-alert
                  v-else-if="selectedExecutionState.error"
                  :title="selectedExecutionState.error"
                  type="error"
                  show-icon
                  :closable="false"
                />

                <el-empty
                  v-else-if="selectedExecutionState.loaded && selectedExecutionState.list.length === 0"
                  :description="t('scheduledTask.emptyExecutions')"
                  :image-size="56"
                />

                <div v-else-if="selectedExecutionState.loaded" class="execution-timeline">
                  <article
                    v-for="execution in selectedExecutionState.list"
                    :key="execution.id"
                    :class="['execution-card', `is-${execution.status || 'unknown'}`, { 'is-focused': execution.id === focusedExecutionId }]"
                  >
                    <div class="execution-card-rail" />
                    <div class="execution-card-main">
                      <div class="execution-card-head">
                        <div class="execution-card-title-line">
                          <el-tag :type="executionStatusTag(execution.status)" size="small" effect="light">
                            {{ executionStatusLabel(execution.status) }}
                          </el-tag>
                          <span class="execution-trigger">
                            {{ execution.trigger_type === 'manual' ? t('scheduledTask.manualTrigger') : t('scheduledTask.scheduledTrigger') }}
                          </span>
                        </div>
                      </div>

                      <div class="execution-time">{{ formatDateTime(execution.scheduled_at) }}</div>

                      <div class="execution-facts">
                        <span v-if="execution.duration_millis">{{ formatDuration(execution.duration_millis) }}</span>
                        <span v-if="execution.executor_run_id" class="is-mono">
                          {{ execution.executor_run_id }}
                        </span>
                      </div>

                      <div v-if="execution.error_message" class="execution-error-card">
                        <div class="execution-error-title">{{ t('scheduledTask.executionError') }}</div>
                        <div class="execution-error-detail">{{ execution.error_message }}</div>
                      </div>
                    </div>
                  </article>
                </div>

                <div
                  v-if="selectedExecutionState.loaded && selectedExecutionState.total > selectedExecutionState.pageSize"
                  class="execution-pagination"
                >
                  <el-pagination
                    small
                    :current-page="selectedExecutionState.page"
                    :page-size="selectedExecutionState.pageSize"
                    :total="selectedExecutionState.total"
                    layout="prev, pager, next"
                    @current-change="(nextPage: number) => handleSelectedExecutionPageChange(nextPage)"
                  />
                </div>
              </section>
            </section>

            <aside class="detail-aside">
              <section class="detail-aside-card">
                <div class="detail-aside-card-head">
                  <div class="detail-aside-title">{{ t('scheduledTask.functionDetailTitle') }}</div>
                  <el-tag :type="taskStatusTag(selectedTask.status)" effect="light">
                    {{ taskStatusLabel(selectedTask.status) }}
                  </el-tag>
                </div>
                <div class="detail-aside-name">{{ selectedTask.title || t('scheduledTask.unnamedFunctionTask') }}</div>
                <div class="detail-aside-path">{{ selectedTask.resource_key || resourcePath || '-' }}</div>
                <div class="detail-aside-actions">
                  <el-tooltip :content="t('scheduledTask.runNow')" placement="top" effect="light">
                    <el-button
                      type="primary"
                      :icon="VideoPlay"
                      :disabled="isTerminal(selectedTask.status)"
                      @click="handleRunNow(selectedTask)"
                    />
                  </el-tooltip>
                  <el-tooltip
                    :content="selectedTask.status === 'paused' ? t('scheduledTask.resume') : t('scheduledTask.pause')"
                    placement="top"
                    effect="light"
                  >
                    <el-button
                      :type="selectedTask.status === 'paused' ? 'primary' : 'warning'"
                      :icon="selectedTask.status === 'paused' ? CaretRight : VideoPause"
                      :disabled="isTerminal(selectedTask.status)"
                      @click="selectedTask.status === 'paused' ? handleResume(selectedTask) : handlePause(selectedTask)"
                    />
                  </el-tooltip>
                  <el-tooltip :content="t('scheduledTask.cancel')" placement="top" effect="light">
                    <el-button
                      type="danger"
                      :icon="Close"
                      :disabled="isTerminal(selectedTask.status)"
                      @click="handleCancel(selectedTask)"
                    />
                  </el-tooltip>
                  <el-tooltip :content="t('scheduledTask.delete')" placement="top" effect="light">
                    <el-button
                      type="danger"
                      plain
                      :icon="Delete"
                      :disabled="!!selectedTask.inflight_execution_id"
                      @click="handleDelete(selectedTask)"
                    />
                  </el-tooltip>
                </div>
              </section>

              <section class="detail-aside-card">
                <div class="detail-aside-title">{{ t('scheduledTask.schedule') }}</div>
                <div class="detail-property-list">
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.schedule') }}</span>
                    <strong>{{ scheduleLabel(selectedTask.schedule) }}</strong>
                  </div>
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.nextRun') }}</span>
                    <strong>{{ formatDateTime(selectedTask.next_run_at) }}</strong>
                  </div>
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.runCount') }}</span>
                    <strong>{{ selectedTask.run_count || 0 }}</strong>
                  </div>
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.functionPath') }}</span>
                    <strong class="is-mono">{{ selectedTask.resource_key || resourcePath || '-' }}</strong>
                  </div>
                </div>
              </section>

              <el-alert
                v-if="selectedTask.last_error_message"
                class="detail-alert"
                type="error"
                show-icon
                :closable="false"
                :title="selectedTask.last_error_message"
              />
            </aside>
          </div>
        </template>
      </main>
    </div>

    <ScheduledTaskDialog
      v-model="showCreateDialog"
      :full-code-path="resourcePath || ''"
      :function-detail="functionDetail"
      @success="handleCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CaretRight, Close, Delete, Plus, Refresh, VideoPause, VideoPlay } from '@element-plus/icons-vue'
import type { FunctionDetail } from '@/architecture/domain/types'
import {
  cancelTimerTask,
  deleteTimerTask,
  getTimerExecution,
  listTimerExecutions,
  listTimerTasks,
  pauseTimerTask,
  resumeTimerTask,
  runTimerTaskNow,
  type TimerExecution,
  type TimerTask,
} from '@/architecture/presentation/context/api/timer'
import {
  executionStatusLabel,
  executionStatusTag,
  formatDateTime,
  formatDuration,
  scheduleLabel,
  taskStatusLabel,
  taskStatusTag,
} from './utils/timerSchedule'
import {
  buildScheduledExecutionRoute,
} from '@/architecture/shared/routing/platformRouteParams'
import ScheduledTaskDialog from './ScheduledTaskDialog.vue'

interface ExecutionState {
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
  resourcePath?: string
  functionDetail?: FunctionDetail | null
  autoLoad?: boolean
  focusTaskId?: number | string
  focusExecutionId?: number | string
}>(), {
  resourcePath: '',
  functionDetail: null,
  autoLoad: false,
  focusTaskId: '',
  focusExecutionId: '',
})

const emit = defineEmits<{
  (e: 'total-change', value: number): void
}>()

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const loading = ref(false)
const list = ref<TimerTask[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const statusFilter = ref('')
const showCreateDialog = ref(false)
const selectedTask = ref<TimerTask | null>(null)
const focusedExecutionId = ref(0)
const appliedFocusKey = ref('')

const selectedExecutionState = reactive<ExecutionState>({
  loading: false,
  loaded: false,
  error: '',
  status: '',
  page: 1,
  pageSize: 10,
  total: 0,
  list: [],
})

async function loadList() {
  if (!props.resourcePath) {
    list.value = []
    total.value = 0
    selectedTask.value = null
    emit('total-change', 0)
    return
  }

  loading.value = true
  try {
    const resp = await listTimerTasks({
      executor_key: 'app.function',
      resource_scope: 'function',
      resource_key: props.resourcePath,
      status: statusFilter.value,
      page: page.value,
      page_size: pageSize,
    })
    list.value = resp.list || []
    total.value = Number(resp.total || 0)
    emit('total-change', total.value)
    if (selectedTask.value) {
      selectedTask.value = list.value.find(item => item.id === selectedTask.value?.id) || null
    }
    await openFocusedTaskIfNeeded()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.loadFunctionsFailed'))
  } finally {
    loading.value = false
  }
}

async function loadSelectedExecutions(reset = false) {
  if (!selectedTask.value) return
  if (reset) selectedExecutionState.page = 1
  selectedExecutionState.loading = true
  selectedExecutionState.error = ''
  try {
    const resp = await listTimerExecutions(selectedTask.value.id, {
      status: selectedExecutionState.status,
      page: selectedExecutionState.page,
      page_size: selectedExecutionState.pageSize,
    })
    selectedExecutionState.list = resp.list || []
    await ensureFocusedExecutionLoaded()
    selectedExecutionState.total = Number(resp.total || 0)
    selectedExecutionState.loaded = true
  } catch (error) {
    selectedExecutionState.error = error instanceof Error ? error.message : t('scheduledTask.loadExecutionsFailed')
  } finally {
    selectedExecutionState.loading = false
  }
}

function handleSelectedExecutionPageChange(nextPage: number) {
  selectedExecutionState.page = nextPage
  void loadSelectedExecutions()
}

async function selectTask(task: TimerTask, syncRoute = true) {
  selectedTask.value = task
  selectedExecutionState.page = 1
  selectedExecutionState.status = ''
  selectedExecutionState.loaded = false
  selectedExecutionState.list = []
  selectedExecutionState.total = 0
  selectedExecutionState.error = ''
  if (syncRoute && props.resourcePath) {
    await router.replace({
      path: route.path,
      query: buildScheduledExecutionRoute({
        fullCodePath: props.resourcePath,
        kind: 'function',
        taskId: task.id,
      }).query,
    })
  }
  await loadSelectedExecutions()
}

function normalizeFocusID(value?: number | string): number {
  const num = Number(value || 0)
  return Number.isFinite(num) && num > 0 ? num : 0
}

function currentFocusKey(): string {
  const taskID = normalizeFocusID(props.focusTaskId)
  if (!taskID) return ''
  return `${taskID}:${normalizeFocusID(props.focusExecutionId)}`
}

async function openFocusedTaskIfNeeded() {
  const key = currentFocusKey()
  if (!key || appliedFocusKey.value === key) return
  const taskID = normalizeFocusID(props.focusTaskId)
  const task = list.value.find(item => item.id === taskID)
  if (!task) return
  appliedFocusKey.value = key
  focusedExecutionId.value = normalizeFocusID(props.focusExecutionId)
  await selectTask(task, false)
}

async function ensureFocusedExecutionLoaded() {
  if (!selectedTask.value || !focusedExecutionId.value) return
  if (selectedExecutionState.list.some(item => item.id === focusedExecutionId.value)) return
  try {
    const execution = await getTimerExecution(selectedTask.value.id, focusedExecutionId.value)
    selectedExecutionState.list = [execution, ...selectedExecutionState.list]
  } catch {
    // 聚焦执行可能已被删除或不属于当前任务，忽略即可。
  }
}

function isTerminal(status: string): boolean {
  return ['done', 'failed', 'cancelled'].includes(status)
}

async function handleRunNow(task: TimerTask) {
  try {
    await runTimerTaskNow(task.id)
    ElMessage.success(t('scheduledTask.submittedRunNow'))
    await loadList()
    if (selectedTask.value?.id === task.id) await loadSelectedExecutions(true)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.runNowFailed'))
  }
}

async function handlePause(task: TimerTask) {
  try {
    await pauseTimerTask(task.id)
    ElMessage.success(t('scheduledTask.pausedSuccess'))
    await loadList()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.pauseFailed'))
  }
}

async function handleResume(task: TimerTask) {
  try {
    await resumeTimerTask(task.id)
    ElMessage.success(t('scheduledTask.resumedSuccess'))
    await loadList()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.resumeFailed'))
  }
}

async function handleCancel(task: TimerTask) {
  try {
    await ElMessageBox.confirm(t('scheduledTask.cancelFunctionConfirm'), t('scheduledTask.cancelFunctionTitle'), {
      type: 'warning',
      confirmButtonText: t('scheduledTask.cancelTaskButton'),
      cancelButtonText: t('common.back'),
    })
    await cancelTimerTask(task.id)
    ElMessage.success(t('scheduledTask.cancelledSuccess'))
    await loadList()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.cancelFailed'))
  }
}

async function handleDelete(task: TimerTask) {
  try {
    await ElMessageBox.confirm(
      t('scheduledTask.deleteFunctionConfirm'),
      t('scheduledTask.deleteFunctionTitle'),
      {
        type: 'warning',
        confirmButtonText: t('scheduledTask.delete'),
        cancelButtonText: t('common.back'),
        confirmButtonClass: 'el-button--danger',
      }
    )
    await deleteTimerTask(task.id)
    ElMessage.success(t('scheduledTask.deletedSuccess'))
    if (selectedTask.value?.id === task.id) {
      selectedTask.value = null
    }
    if (list.value.length <= 1 && page.value > 1) page.value -= 1
    await loadList()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.deleteFailed'))
  }
}

function handleCreated() {
  page.value = 1
  void loadList()
}

watch(
  () => [props.autoLoad, props.resourcePath],
  ([autoLoad]) => {
    if (autoLoad) {
      page.value = 1
      void loadList()
    }
  },
  { immediate: true }
)

watch(
  () => [props.focusTaskId, props.focusExecutionId],
  () => {
    appliedFocusKey.value = ''
    void openFocusedTaskIfNeeded()
  }
)

defineExpose({ load: loadList })
</script>

<style scoped lang="scss">
.scheduled-task-list {
  --scheduled-session-ink: var(--el-text-color-primary);
  --scheduled-session-muted: var(--el-text-color-secondary);
  --scheduled-session-soft: var(--app-shell-bg, var(--el-bg-color-page));
  --scheduled-session-paper: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  --scheduled-session-tint: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
  --scheduled-session-line: var(--app-shell-panel-border, var(--el-border-color-lighter));
  --scheduled-session-accent: var(--el-color-primary);
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: hidden;
  padding: 16px;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 14px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--app-shell-panel-muted-bg, var(--el-fill-color-light)) 78%, transparent), var(--scheduled-session-soft)),
    var(--scheduled-session-soft);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.8));
}

.scheduled-list-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  flex-shrink: 0;
  padding: 14px 16px;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 12px;
  background: var(--scheduled-session-paper);
  box-shadow: var(--app-shell-panel-shadow-soft, 0 10px 24px rgba(15, 23, 42, 0.06));
}

.scheduled-list-title {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--scheduled-session-ink);
}

.scheduled-list-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--scheduled-session-muted);
  word-break: break-all;
}

.scheduled-list-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.scheduled-total {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 9px;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 999px;
  background: var(--scheduled-session-tint);
  font-size: 12px;
  color: var(--scheduled-session-accent);
}

.scheduled-list-filter {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--scheduled-session-line);
  background: var(--scheduled-session-tint);
}

.scheduled-list-filter :deep(.el-select) {
  width: 132px;
  flex: 0 0 auto;
}

.scheduled-sidebar-title {
  min-width: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--scheduled-session-ink);
}

.scheduled-agent-workspace {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(270px, 340px) minmax(0, 1fr);
  gap: 14px;
}

.scheduled-agent-sidebar,
.scheduled-agent-detail {
  min-width: 0;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 12px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 10px 24px rgba(15, 23, 42, 0.06));
}

.scheduled-agent-sidebar {
  display: flex;
  min-height: 0;
  overflow: hidden;
  flex-direction: column;
}

.agent-session-list {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: grid;
  align-content: start;
  gap: 9px;
  padding: 12px;
}

.agent-session-item {
  appearance: none;
  width: 100%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 7px;
  padding: 12px 13px;
  border: 1px solid var(--scheduled-session-line);
  border-left: 3px solid transparent;
  border-radius: 10px;
  background: var(--scheduled-session-paper);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.agent-session-item:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.34);
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--scheduled-session-paper));
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.06);
}

.agent-session-item.is-active {
  border-color: rgba(var(--el-color-primary-rgb), 0.5);
  border-left-color: var(--scheduled-session-accent);
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--scheduled-session-paper));
  box-shadow: 0 8px 20px rgba(var(--el-color-primary-rgb), 0.12);
}

.agent-session-item.is-failed {
  border-left-color: var(--el-color-danger);
}

.agent-session-item.is-paused {
  border-left-color: var(--el-color-warning);
}

.agent-session-item.is-running {
  border-left-color: var(--el-color-primary);
}

.agent-session-item-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.agent-session-item-title {
  min-width: 0;
  color: var(--scheduled-session-ink);
  font-size: 13px;
  font-weight: 700;
  line-height: 1.4;
  word-break: break-word;
}

.agent-session-item-summary {
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.55;
  word-break: break-all;
}

.agent-session-item-summary.is-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.agent-session-item-error {
  padding: 7px 8px;
  border-radius: 7px;
  color: var(--el-color-danger);
  background: color-mix(in srgb, var(--el-color-danger) 8%, var(--scheduled-session-paper));
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.agent-session-item-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px 8px;
  margin-top: 1px;
}

.agent-session-item-meta span {
  position: relative;
  max-width: 100%;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: none;
  color: var(--scheduled-session-muted);
  font-size: 11.5px;
  line-height: 1.5;
}

.agent-session-item-meta span:not(:first-child)::before {
  content: '·';
  margin-right: 8px;
  color: color-mix(in srgb, var(--scheduled-session-muted) 55%, transparent);
}

.scheduled-pagination {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
  padding: 8px 10px 10px;
  border-top: 1px solid var(--scheduled-session-line);
}

.scheduled-task-list :deep(.el-button) {
  border-radius: 8px;
}

.scheduled-sidebar-empty,
.scheduled-detail-empty {
  flex: 1;
  border: 1px dashed var(--scheduled-session-line);
  background: color-mix(in srgb, var(--scheduled-session-paper) 72%, var(--scheduled-session-tint) 28%);
}

.drawer-section-controls {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.scheduled-agent-detail {
  min-height: 0;
  overflow: hidden;
  padding: 0;
  display: flex;
}

.scheduled-detail-empty {
  flex: 1;
  min-height: 360px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.scheduled-detail-shell {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(260px, 310px);
  overflow: hidden;
}

.detail-document {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 24px 28px 30px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
}

.detail-document-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--scheduled-session-line);
}

.detail-kicker {
  color: var(--scheduled-session-accent);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.4;
}

.detail-toolbar-title {
  margin-top: 4px;
  color: var(--scheduled-session-ink);
  font-size: 16px;
  font-weight: 760;
  line-height: 1.4;
  word-break: break-word;
}

.detail-document-section {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--scheduled-session-line);
}

.detail-document-toolbar + .detail-document-section {
  border-top: 0;
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

.detail-property strong.is-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.detail-alert {
  flex-shrink: 0;
}

.drawer-executions-loading {
  min-height: 120px;
}

/* ─── 执行时间线 ─── */

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

.execution-facts span.is-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
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

.execution-error-detail {
  margin-top: 6px;
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

@media (max-width: 1100px) {
  .scheduled-agent-workspace {
    grid-template-columns: minmax(240px, 300px) minmax(0, 1fr);
  }

  .scheduled-detail-shell {
    grid-template-columns: minmax(0, 1fr) minmax(240px, 280px);
  }
}

@media (max-width: 860px) {
  .scheduled-agent-workspace {
    grid-template-columns: 1fr;
  }

  .scheduled-agent-sidebar {
    max-height: 360px;
  }

  .scheduled-detail-shell {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .detail-document,
  .detail-aside {
    overflow: visible;
  }

  .detail-aside {
    border-top: 1px solid var(--scheduled-session-line);
    border-left: 0;
  }

  .detail-document-toolbar,
  .detail-section-head {
    flex-direction: column;
  }
}

@media (max-width: 768px) {
  .scheduled-task-list {
    padding: 12px;
  }

  .scheduled-list-header {
    align-items: stretch;
    flex-direction: column;
  }

  .detail-document {
    padding: 18px 16px 22px;
  }

  .detail-aside {
    padding: 12px;
  }
}
</style>
