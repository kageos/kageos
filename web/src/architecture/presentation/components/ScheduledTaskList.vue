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

    <div class="scheduled-list-filter">
      <el-select v-model="statusFilter" :placeholder="t('scheduledTask.allStatuses')" clearable style="width: 160px" @change="loadList">
        <el-option :label="t('scheduledTask.allStatuses')" value="" />
        <el-option :label="t('scheduledTask.taskStatusPending')" value="pending" />
        <el-option :label="t('scheduledTask.taskStatusPaused')" value="paused" />
        <el-option :label="t('scheduledTask.taskStatusDone')" value="done" />
        <el-option :label="t('scheduledTask.taskStatusFailed')" value="failed" />
        <el-option :label="t('scheduledTask.taskStatusCancelled')" value="cancelled" />
      </el-select>
    </div>

    <el-empty v-if="!loading && list.length === 0" :description="t('scheduledTask.emptyFunctions')" />

    <el-table
      v-else
      :data="list"
      row-key="id"
      stripe
      class="scheduled-table"
      :row-class-name="() => 'is-clickable'"
      @row-click="openTaskDrawer"
    >
      <el-table-column prop="title" :label="t('scheduledTask.name')" min-width="200" show-overflow-tooltip />
      <el-table-column :label="t('scheduledTask.schedule')" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ scheduleLabel(row.schedule) }}</template>
      </el-table-column>
      <el-table-column prop="next_run_at" :label="t('scheduledTask.nextRun')" width="180">
        <template #default="{ row }">{{ formatDateTime(row.next_run_at) }}</template>
      </el-table-column>
      <el-table-column prop="status" :label="t('scheduledTask.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="taskStatusTag(row.status)" size="small">{{ taskStatusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="run_count" :label="t('scheduledTask.runCount')" width="72" align="center" />
      <el-table-column :label="t('scheduledTask.actions')" width="168" fixed="right" align="center">
        <template #default="{ row }">
          <div class="table-row-actions">
            <el-tooltip :content="t('scheduledTask.runNow')" placement="top" effect="light">
              <el-button
                text
                type="primary"
                :icon="VideoPlay"
                :disabled="isTerminal(row.status)"
                @click.stop="handleRunNow(row)"
              />
            </el-tooltip>
            <el-tooltip v-if="row.status === 'paused'" :content="t('scheduledTask.resume')" placement="top" effect="light">
              <el-button text type="primary" :icon="CaretRight" @click.stop="handleResume(row)" />
            </el-tooltip>
            <el-tooltip v-else :content="t('scheduledTask.pause')" placement="top" effect="light">
              <el-button
                text
                type="warning"
                :icon="VideoPause"
                :disabled="isTerminal(row.status)"
                @click.stop="handlePause(row)"
              />
            </el-tooltip>
            <el-tooltip :content="t('scheduledTask.cancel')" placement="top" effect="light">
              <el-button
                text
                type="danger"
                :icon="Close"
                :disabled="isTerminal(row.status)"
                @click.stop="handleCancel(row)"
              />
            </el-tooltip>
            <el-tooltip :content="t('scheduledTask.delete')" placement="top" effect="light">
              <el-button
                text
                type="danger"
                :icon="Delete"
                :disabled="!!row.inflight_execution_id"
                @click.stop="handleDelete(row)"
              />
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="total > pageSize" class="scheduled-pagination">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="loadList"
      />
    </div>

    <ScheduledTaskDialog
      v-model="showCreateDialog"
      :full-code-path="resourcePath || ''"
      :function-detail="functionDetail"
      @success="handleCreated"
    />

    <el-drawer
      v-model="drawerVisible"
      :title="selectedTask?.title || t('scheduledTask.functionDetailTitle')"
      direction="rtl"
      size="520px"
      :destroy-on-close="false"
      class="scheduled-task-drawer"
    >
      <template v-if="selectedTask">
        <section class="drawer-section">
          <div class="drawer-section-title">{{ t('scheduledTask.taskInfo') }}</div>
          <dl class="drawer-info-grid">
            <dt>{{ t('scheduledTask.schedule') }}</dt>
            <dd>{{ scheduleLabel(selectedTask.schedule) }}</dd>
            <dt>{{ t('scheduledTask.nextRun') }}</dt>
            <dd>{{ formatDateTime(selectedTask.next_run_at) }}</dd>
            <dt>{{ t('scheduledTask.status') }}</dt>
            <dd>
              <el-tag :type="taskStatusTag(selectedTask.status)" size="small">
                {{ taskStatusLabel(selectedTask.status) }}
              </el-tag>
            </dd>
            <dt>{{ t('scheduledTask.runCount') }}</dt>
            <dd>{{ selectedTask.run_count || 0 }}</dd>
            <dt>{{ t('scheduledTask.functionPath') }}</dt>
            <dd class="is-mono">{{ selectedTask.resource_key || resourcePath || '-' }}</dd>
            <template v-if="selectedTask.last_error_message">
              <dt>{{ t('scheduledTask.recentError') }}</dt>
              <dd class="is-error-text">{{ selectedTask.last_error_message }}</dd>
            </template>
          </dl>
        </section>

        <section class="drawer-section is-executions">
          <div class="drawer-section-head">
            <div class="drawer-section-title">{{ t('scheduledTask.executionRecords') }}</div>
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
                  <el-tag :type="executionStatusTag(execution.status)" size="small" effect="light">
                    {{ executionStatusLabel(execution.status) }}
                  </el-tag>
                  <span class="execution-trigger">
                    {{ execution.trigger_type === 'manual' ? t('scheduledTask.manualTrigger') : t('scheduledTask.scheduledTrigger') }}
                  </span>
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
      </template>
    </el-drawer>
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
const drawerVisible = ref(false)
const selectedTask = ref<TimerTask | null>(null)
const focusedExecutionId = ref(0)
const appliedFocusKey = ref('')
const executionStates = reactive<Record<number, ExecutionState>>({})

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

function getExecutionState(taskID: number): ExecutionState {
  if (!executionStates[taskID]) {
    executionStates[taskID] = {
      loading: false,
      loaded: false,
      error: '',
      status: '',
      page: 1,
      pageSize: 10,
      total: 0,
      list: [],
    }
  }
  return executionStates[taskID]
}

async function loadList() {
  if (!props.resourcePath) {
    list.value = []
    total.value = 0
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
    await openFocusedTaskIfNeeded()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.loadFunctionsFailed'))
  } finally {
    loading.value = false
  }
}

async function loadExecutions(task: TimerTask, reset = false) {
  const state = getExecutionState(task.id)
  if (reset) state.page = 1
  state.loading = true
  state.error = ''
  try {
    const resp = await listTimerExecutions(task.id, {
      status: state.status,
      page: state.page,
      page_size: state.pageSize,
    })
    state.list = resp.list || []
    state.total = Number(resp.total || 0)
    state.loaded = true
  } catch (error) {
    state.error = error instanceof Error ? error.message : t('scheduledTask.loadExecutionsFailed')
  } finally {
    state.loading = false
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

async function openTaskDrawer(task: TimerTask, syncRoute = true) {
  selectedTask.value = task
  selectedExecutionState.page = 1
  selectedExecutionState.status = ''
  selectedExecutionState.loaded = false
  selectedExecutionState.list = []
  selectedExecutionState.total = 0
  selectedExecutionState.error = ''
  drawerVisible.value = true
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
  await openTaskDrawer(task, false)
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
    const state = getExecutionState(task.id)
    if (state.loaded) await loadExecutions(task, true)
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
    delete executionStates[task.id]
    if (selectedTask.value?.id === task.id) {
      drawerVisible.value = false
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
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: hidden;
  padding: 16px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 14px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--app-shell-panel-muted-bg, #f1f5f9) 78%, #fff 22%), color-mix(in srgb, var(--app-shell-bg, #eef2f6) 92%, #fff 8%)),
    var(--app-shell-bg, var(--el-bg-color-page));
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.8));
}

.scheduled-list-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  flex-shrink: 0;
  padding: 14px 16px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 10px 24px rgba(15, 23, 42, 0.06));
}

.scheduled-list-title {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}

.scheduled-list-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
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
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 999px;
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.scheduled-list-filter {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-shell-panel-bg-strong, #fff) 72%, var(--app-shell-panel-muted-bg, #f1f5f9) 28%);
}

/* ─── 表格 ─── */

.scheduled-table {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 10px 24px rgba(15, 23, 42, 0.06));
}

.scheduled-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.scheduled-table :deep(.el-table__header-wrapper th) {
  height: 44px;
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

.scheduled-table :deep(.el-table__row) {
  background: color-mix(in srgb, var(--app-shell-panel-bg-strong, #fff) 92%, var(--app-shell-panel-muted-bg, #f1f5f9) 8%);
}

.scheduled-table :deep(.el-table__row--striped td.el-table__cell) {
  background: color-mix(in srgb, var(--app-shell-panel-muted-bg, #f1f5f9) 46%, #fff 54%);
}

.scheduled-table :deep(.el-table__row.is-clickable) {
  cursor: pointer;
}

.scheduled-table :deep(.el-table__row:hover > td.el-table__cell) {
  background: color-mix(in srgb, var(--el-color-primary) 7%, var(--app-shell-panel-bg-strong, #fff));
}

.scheduled-table :deep(.el-table__cell) {
  border-bottom-color: color-mix(in srgb, var(--app-shell-panel-border, #cbd5e1) 58%, transparent);
}

.table-row-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.table-row-actions :deep(.el-button) {
  width: 30px;
  height: 30px;
  margin: 0;
  border-radius: 8px;
  background: transparent;
}

.table-row-actions :deep(.el-button:hover) {
  background: color-mix(in srgb, currentColor 10%, transparent);
}

.scheduled-pagination {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
  padding: 2px 4px 0;
}

.scheduled-task-list :deep(.el-button) {
  border-radius: 8px;
}

.scheduled-task-list :deep(.el-empty) {
  flex: 1;
  border: 1px dashed var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-shell-panel-bg-strong, #fff) 72%, var(--app-shell-panel-muted-bg, #f1f5f9) 28%);
}

/* ─── 抽屉 ─── */

.drawer-section {
  padding: 16px 20px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-shell-panel-border, #cbd5e1) 50%, transparent);
}

.drawer-section:last-child {
  border-bottom: none;
}

.drawer-section.is-executions {
  border-bottom: none;
}

.drawer-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.drawer-section-title {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--el-text-color-secondary);
  margin-bottom: 12px;
}

.drawer-section-head .drawer-section-title {
  margin-bottom: 0;
}

.drawer-section-controls {
  display: flex;
  gap: 8px;
  align-items: center;
}

.drawer-info-grid {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 10px 16px;
  margin: 0;
  font-size: 13px;
}

.drawer-info-grid dt {
  color: var(--el-text-color-secondary);
  font-weight: 500;
}

.drawer-info-grid dd {
  margin: 0;
  color: var(--el-text-color-primary);
  line-height: 1.5;
  word-break: break-word;
}

.drawer-info-grid dd.is-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.drawer-info-grid dd.is-error-text {
  color: var(--el-color-danger);
  font-size: 12px;
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
  border: 1px solid color-mix(in srgb, var(--app-shell-panel-border, #cbd5e1) 60%, transparent);
  border-radius: 10px;
  background: var(--el-bg-color, #fff);
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.execution-card.is-focused {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 18%, transparent);
}

.execution-card:hover {
  border-color: color-mix(in srgb, var(--el-color-primary) 24%, var(--app-shell-panel-border, #cbd5e1));
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.05);
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
  justify-content: flex-start;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.execution-trigger {
  font-size: 12px;
  color: var(--el-text-color-secondary);
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
  color: var(--el-text-color-secondary);
  background: color-mix(in srgb, var(--app-shell-panel-muted-bg, #f1f5f9) 70%, #fff 30%);
  border: 1px solid color-mix(in srgb, var(--app-shell-panel-border, #cbd5e1) 50%, transparent);
}

.execution-facts span.is-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
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
  padding: 0 20px 16px;
}

@media (max-width: 768px) {
  .scheduled-task-list {
    padding: 12px;
  }

  .scheduled-list-header {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>

<style lang="scss">
.scheduled-task-drawer {
  .el-drawer__header {
    margin-bottom: 0;
    padding: 18px 20px 14px;
    border-bottom: 1px solid color-mix(in srgb, #cbd5e1 50%, transparent);
  }

  .el-drawer__title {
    font-size: 15px;
    font-weight: 700;
    color: var(--el-text-color-primary, #0f172a);
  }

  .el-drawer__body {
    padding: 0;
    overflow-y: auto;
  }
}
</style>
