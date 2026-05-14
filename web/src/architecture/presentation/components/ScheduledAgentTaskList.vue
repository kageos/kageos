<template>
  <div class="scheduled-agent-task-list" v-loading="loading">
    <div class="section-header">
      <div class="section-copy">
        <div class="section-title">定时会话</div>
        <div class="section-desc">管理当前节点及子路径下的 Agent 自动运行任务，并追踪每次执行会话。</div>
      </div>
      <div class="section-actions">
        <span class="section-total">共 {{ total }} 个任务</span>
        <el-button type="primary" @click="loadList">刷新</el-button>
      </div>
    </div>

    <div class="filter-section">
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="状态">
          <el-select
            v-model="filterForm.status"
            placeholder="全部状态"
            clearable
            style="width: 160px"
            @change="handleFilterChange"
          >
            <el-option label="全部状态" value="" />
            <el-option label="待执行" value="pending" />
            <el-option label="已暂停" value="paused" />
            <el-option label="已完成" value="done" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="filterForm.keyword"
            clearable
            placeholder="搜索名称/目标/路径"
            style="width: 220px"
            @change="handleFilterChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>
      <div class="filter-summary">
        <span>当前展示当前节点及子路径下的定时会话</span>
      </div>
    </div>

    <div class="table-section">
      <el-empty
        v-if="!loading && filteredList.length === 0"
        :description="filterForm.status || filterForm.keyword ? '当前筛选条件下暂无定时会话' : '暂无定时会话'"
      />
      <el-table
        v-else
        :data="filteredList"
        row-key="id"
        stripe
        class="task-table"
        :preserve-expanded-content="true"
        @expand-change="handleExpandChange"
      >
        <el-table-column type="expand" width="48">
          <template #default="{ row }">
            <div class="execution-expand" v-loading="executionState(row.id).loading">
              <div class="execution-panel">
                <div class="execution-panel-header">
                  <div>
                    <div class="execution-title">执行会话记录</div>
                    <div class="execution-desc">每次执行会生成一条记录，成功拿到 session_id 后可直接打开会话。</div>
                  </div>
                  <el-button size="small" :icon="Refresh" @click.stop="loadExecutions(row, true)">刷新</el-button>
                </div>

                <el-alert
                  v-if="executionState(row.id).error"
                  :title="executionState(row.id).error"
                  type="error"
                  show-icon
                  :closable="false"
                  class="execution-alert"
                />
                <el-empty
                  v-else-if="executionState(row.id).loaded && executionState(row.id).list.length === 0"
                  description="暂无执行记录"
                  :image-size="56"
                />
                <el-table
                  v-else-if="executionState(row.id).loaded"
                  :data="executionState(row.id).list"
                  size="small"
                  stripe
                  class="execution-table"
                  @row-click="openExecutionSession(row, $event)"
                >
                  <el-table-column label="计划时间" width="170">
                    <template #default="{ row: execution }">{{ formatDateTime(execution.scheduled_at) }}</template>
                  </el-table-column>
                  <el-table-column label="状态" width="100">
                    <template #default="{ row: execution }">
                      <el-tag :type="executionStatusTag(execution.status)" size="small">
                        {{ executionStatusLabel(execution.status) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="会话" min-width="180" show-overflow-tooltip>
                    <template #default="{ row: execution }">{{ execution.session_id || '-' }}</template>
                  </el-table-column>
                  <el-table-column label="摘要" min-width="260" show-overflow-tooltip>
                    <template #default="{ row: execution }">
                      {{ execution.output_summary || execution.error_message || execution.input_goal || '-' }}
                    </template>
                  </el-table-column>
                  <el-table-column label="工具" width="80" align="center">
                    <template #default="{ row: execution }">{{ execution.tool_call_count || 0 }}</template>
                  </el-table-column>
                  <el-table-column label="耗时" width="100" align="center">
                    <template #default="{ row: execution }">{{ formatDuration(execution.duration_millis) }}</template>
                  </el-table-column>
                  <el-table-column label="操作" width="96" align="center" fixed="right">
                    <template #default="{ row: execution }">
                      <el-button
                        link
                        type="primary"
                        size="small"
                        :disabled="!execution.session_id"
                        @click.stop="openExecutionSession(row, execution)"
                      >
                        打开会话
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="任务名称" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="task-name-cell">
              <span class="task-name">{{ row.name || '未命名定时会话' }}</span>
              <span class="task-goal">{{ row.goal }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="full_code_path" label="路径" min-width="260" show-overflow-tooltip />

        <el-table-column label="调度" width="150">
          <template #default="{ row }">
            <div class="schedule-cell">
              <el-tag size="small">{{ scheduleTypeLabel(row.schedule_type) }}</el-tag>
              <el-tooltip :content="scheduleSummary(row)" placement="top" effect="light">
                <span class="schedule-summary-trigger">说明</span>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="执行身份" width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <UserDisplay
              :username="row.request_user || row.created_by || null"
              mode="card"
              layout="horizontal"
              size="small"
            />
          </template>
        </el-table-column>

        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="taskStatusTag(row.status)" size="small">{{ taskStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="下次执行" width="170">
          <template #default="{ row }">{{ row.next_run_at ? formatDateTime(row.next_run_at) : '-' }}</template>
        </el-table-column>

        <el-table-column prop="run_count" label="已执行" width="80" align="center" />

        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.updated_at) }}</template>
        </el-table-column>

        <el-table-column label="操作" width="250" align="center" fixed="right">
          <template #default="{ row }">
            <div class="table-row-actions">
              <el-button link type="primary" size="small" @click.stop="openEdit(row)">编辑</el-button>
              <el-button
                link
                type="primary"
                size="small"
                :loading="actionTaskId === row.id"
                @click.stop="runNow(row)"
              >
                立即执行
              </el-button>
              <el-button
                v-if="row.status === 'pending'"
                link
                type="warning"
                size="small"
                :loading="actionTaskId === row.id"
                @click.stop="pauseTask(row)"
              >
                暂停
              </el-button>
              <el-button
                v-if="row.status === 'paused'"
                link
                type="success"
                size="small"
                :loading="actionTaskId === row.id"
                @click.stop="resumeTask(row)"
              >
                恢复
              </el-button>
              <el-button link type="danger" size="small" :loading="actionTaskId === row.id" @click.stop="deleteTask(row)">
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-pagination
      v-if="total > 0"
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="[10, 20, 50]"
      layout="total, sizes, prev, pager, next"
      class="list-pagination"
      @current-change="loadList"
      @size-change="handlePageSizeChange"
    />

    <ScheduledAgentTaskDialog
      v-model="editDialogVisible"
      :full-code-path="editingTask?.full_code_path || props.resourcePath || ''"
      :task="editingTask"
      @success="handleEditSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import {
  deleteScheduledAgentTask,
  listScheduledAgentExecutions,
  listScheduledAgentTasks,
  pauseScheduledAgentTask,
  resumeScheduledAgentTask,
  runScheduledAgentTaskNow,
  type ScheduledAgentExecutionItem,
  type ScheduledAgentTaskItem,
  type ScheduledAgentTaskStatus
} from '@/architecture/infrastructure/api/scheduledAgentTask'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'
import ScheduledAgentTaskDialog from './ScheduledAgentTaskDialog.vue'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import type { WorkspaceSessionItem } from '@/architecture/infrastructure/api/workspace'

const props = withDefaults(defineProps<{
  resourcePath?: string
  autoLoad?: boolean
}>(), {
  autoLoad: false
})

const emit = defineEmits<{
  (e: 'total-change', total: number): void
  (e: 'open-session', session: WorkspaceSessionItem): void
}>()

interface ExecutionState {
  loading: boolean
  loaded: boolean
  error: string
  list: ScheduledAgentExecutionItem[]
}

const loading = ref(false)
const list = ref<ScheduledAgentTaskItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const actionTaskId = ref<number | null>(null)
const editDialogVisible = ref(false)
const editingTask = ref<ScheduledAgentTaskItem | null>(null)
const executionStates = reactive<Record<number, ExecutionState>>({})
const filterForm = reactive({
  status: '',
  keyword: ''
})

const filteredList = computed(() => {
  const keyword = filterForm.keyword.trim().toLowerCase()
  if (!keyword) {
    return list.value
  }
  return list.value.filter((task) => [
    task.name,
    task.goal,
    task.full_code_path,
    task.last_error_message,
    task.last_session_id
  ].some((field) => (field || '').toLowerCase().includes(keyword)))
})

function executionState(taskId: number): ExecutionState {
  if (!executionStates[taskId]) {
    executionStates[taskId] = {
      loading: false,
      loaded: false,
      error: '',
      list: []
    }
  }
  return executionStates[taskId]
}

async function loadList() {
  const path = props.resourcePath || ''
  if (!path) {
    list.value = []
    total.value = 0
    emit('total-change', 0)
    return
  }

  loading.value = true
  try {
    const res = await listScheduledAgentTasks({
      full_code_path: path,
      status: filterForm.status || undefined,
      page: page.value,
      page_size: pageSize.value
    })
    list.value = res.list || []
    total.value = res.total || 0
    emit('total-change', total.value)
  } catch (error: any) {
    list.value = []
    total.value = 0
    emit('total-change', 0)
    ElMessage.error(error?.message || '加载定时会话失败')
  } finally {
    loading.value = false
  }
}

async function loadExecutions(task: ScheduledAgentTaskItem, force = false) {
  const state = executionState(task.id)
  if (state.loading || (state.loaded && !force)) {
    return
  }
  state.loading = true
  state.error = ''
  try {
    const res = await listScheduledAgentExecutions(task.id, { page: 1, page_size: 50 })
    state.list = res.list || []
    state.loaded = true
  } catch (error: any) {
    state.error = error?.message || '加载执行记录失败'
  } finally {
    state.loading = false
  }
}

function handleExpandChange(row: ScheduledAgentTaskItem, expandedRows: ScheduledAgentTaskItem[]) {
  if (expandedRows.some((item) => item.id === row.id)) {
    void loadExecutions(row)
  }
}

function handleFilterChange() {
  page.value = 1
  void loadList()
}

function handlePageSizeChange() {
  page.value = 1
  void loadList()
}

function resetFilters() {
  filterForm.status = ''
  filterForm.keyword = ''
  page.value = 1
  void loadList()
}

function openEdit(task: ScheduledAgentTaskItem) {
  editingTask.value = task
  editDialogVisible.value = true
}

function handleEditSuccess() {
  editingTask.value = null
  void loadList()
}

async function runNow(task: ScheduledAgentTaskItem) {
  actionTaskId.value = task.id
  try {
    await runScheduledAgentTaskNow(task.id)
    ElMessage.success('已触发执行')
    await loadList()
    await loadExecutions(task, true)
  } catch (error: any) {
    ElMessage.error(error?.message || '触发失败')
  } finally {
    actionTaskId.value = null
  }
}

async function pauseTask(task: ScheduledAgentTaskItem) {
  actionTaskId.value = task.id
  try {
    await pauseScheduledAgentTask(task.id)
    ElMessage.success('已暂停')
    await loadList()
  } catch (error: any) {
    ElMessage.error(error?.message || '暂停失败')
  } finally {
    actionTaskId.value = null
  }
}

async function resumeTask(task: ScheduledAgentTaskItem) {
  actionTaskId.value = task.id
  try {
    await resumeScheduledAgentTask(task.id)
    ElMessage.success('已恢复')
    await loadList()
  } catch (error: any) {
    ElMessage.error(error?.message || '恢复失败')
  } finally {
    actionTaskId.value = null
  }
}

async function deleteTask(task: ScheduledAgentTaskItem) {
  try {
    await ElMessageBox.confirm(
      `确定删除定时会话「${task.name || '未命名定时会话'}」吗？删除后不会再自动执行。`,
      '删除定时会话',
      {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        confirmButtonClass: 'el-button--danger'
      }
    )
  } catch {
    return
  }

  actionTaskId.value = task.id
  try {
    await deleteScheduledAgentTask(task.id)
    ElMessage.success('已删除')
    eventBus.emit(WorkspaceEvent.scheduledAgentTaskCreated, { full_code_path: task.full_code_path })
    await loadList()
  } catch (error: any) {
    ElMessage.error(error?.message || '删除失败')
  } finally {
    actionTaskId.value = null
  }
}

function openExecutionSession(task: ScheduledAgentTaskItem, execution: ScheduledAgentExecutionItem) {
  if (!execution.session_id) {
    ElMessage.info('该执行记录还没有会话 ID')
    return
  }
  emit('open-session', {
    session_id: execution.session_id,
    title: task.name,
    user: task.request_user || task.created_by,
    status: execution.status === 'running' ? 'generating' : execution.status === 'cancelled' ? 'cancelled' : 'done',
    full_code_path: task.full_code_path,
    created_at: execution.created_at,
    updated_at: execution.updated_at
  })
}

function taskStatusLabel(status: ScheduledAgentTaskStatus | string): string {
  const map: Record<string, string> = {
    pending: '待执行',
    paused: '已暂停',
    done: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return map[status] || status || '-'
}

function taskStatusTag(status: ScheduledAgentTaskStatus | string): 'success' | 'warning' | 'info' | 'primary' | 'danger' {
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

function scheduleSummary(task: ScheduledAgentTaskItem): string {
  if (task.schedule_type === 'atime') {
    return `指定时间执行：${formatDateTime(task.run_at)}`
  }
  if (task.schedule_type === 'cron') {
    return `Cron：${task.cron_expr || '-'}`
  }
  return `每 ${task.interval_seconds || 0} 秒执行一次${task.max_runs ? `，最多 ${task.max_runs} 次` : ''}`
}

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatDuration(ms?: number): string {
  const value = Number(ms || 0)
  if (value <= 0) return '-'
  if (value < 1000) return `${value}ms`
  const seconds = value / 1000
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  return `${Math.floor(seconds / 60)}m ${Math.floor(seconds % 60)}s`
}

watch(
  () => [props.resourcePath, props.autoLoad] as const,
  ([path, autoLoad]) => {
    if (path && autoLoad) {
      void loadList()
    }
  },
  { immediate: true }
)

const unsubscribeScheduledTaskCreated = eventBus.on(WorkspaceEvent.scheduledAgentTaskCreated, () => {
  if (props.autoLoad) {
    void loadList()
  }
})

onUnmounted(() => {
  unsubscribeScheduledTaskCreated()
})

defineExpose({
  loadList
})
</script>

<style scoped lang="scss">
.scheduled-agent-task-list {
  padding: 20px;
}

.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.section-desc {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.section-actions,
.filter-section,
.schedule-cell,
.table-row-actions {
  display: flex;
  align-items: center;
}

.section-actions {
  gap: 12px;
  flex-shrink: 0;
}

.section-total,
.filter-summary {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.filter-section {
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px 8px;
  margin-bottom: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
}

.filter-form {
  flex: 1;
}

.table-section {
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
}

.task-table :deep(.el-table__expanded-cell) {
  padding: 0;
  background: var(--el-fill-color-light);
}

.task-name-cell {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.task-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.task-goal {
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.schedule-cell {
  gap: 8px;
}

.schedule-summary-trigger {
  color: var(--el-color-primary);
  font-size: 12px;
  cursor: help;
}

.table-row-actions {
  justify-content: center;
  flex-wrap: wrap;
  gap: 0 6px;
}

.table-row-actions :deep(.el-button) {
  margin-left: 0;
}

.execution-expand {
  min-height: 80px;
  padding: 12px 16px 12px 58px;
  background: var(--el-fill-color-light);
}

.execution-panel {
  overflow: hidden;
  border: 1px solid var(--app-auth-card-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: var(--app-auth-card-bg, var(--el-bg-color));
  box-shadow: var(--app-auth-card-shadow-soft, 0 8px 24px rgba(15, 23, 42, 0.06));
}

.execution-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.execution-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.execution-desc {
  margin-top: 2px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.execution-alert {
  margin: 12px;
}

.list-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}
</style>
