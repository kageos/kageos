<template>
  <div class="scheduled-task-list" v-loading="loading">
    <div class="section-header">
      <div class="section-copy">
        <div class="section-title">定时任务</div>
        <div class="section-desc">查看当前函数及子路径下的调度任务，并追踪每次执行结果。</div>
      </div>
      <div class="section-actions">
        <span class="section-total">共 {{ resourceTotal }} 个任务</span>
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
            <el-option label="已完成" value="done" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>
      <div class="filter-summary">
        <span>当前仅展示当前节点及其子路径下的任务</span>
        <span v-if="filterForm.status">筛选后 {{ total }} 条</span>
      </div>
    </div>

    <div class="table-section">
      <el-empty
        v-if="!loading && list.length === 0"
        :description="filterForm.status ? '当前筛选条件下暂无定时任务' : '暂无定时任务'"
      />
      <el-table
        v-else
        :data="list"
        stripe
        style="width: 100%"
        class="task-table"
        @row-click="handleTaskRowClick"
      >
        <el-table-column prop="name" label="任务名称" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="task-name">{{ row.name || '未命名任务' }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="full_code_path" label="函数路径" min-width="260" show-overflow-tooltip />

        <el-table-column label="动作" width="110">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ actionLabel(row.action) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="调度方式" width="140">
          <template #default="{ row }">
            <div class="schedule-cell">
              <el-tag size="small">{{ scheduleTypeLabel(row.schedule_type) }}</el-tag>
              <el-tooltip :content="getScheduleSummary(row)" placement="top" effect="light">
                <span class="schedule-summary-trigger">说明</span>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="request_user" label="执行身份" width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ row.request_user || row.created_by || '-' }}</template>
        </el-table-column>

        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <div class="status-cell">
              <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
              <el-tooltip
                v-if="row.error_message"
                :content="row.error_message"
                placement="top"
                effect="light"
              >
                <span class="error-dot" />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="下次执行" width="180">
          <template #default="{ row }">
            {{ row.next_run_at ? formatDateTime(row.next_run_at) : '-' }}
          </template>
        </el-table-column>

        <el-table-column prop="run_count" label="已执行" width="90" />

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>

        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click.stop="openTaskDetail(row)">
              详情
            </el-button>
            <el-button type="primary" link size="small" @click.stop="openExecutions(row)">
              执行记录
            </el-button>
            <el-button
              v-if="row.status === 'pending'"
              type="danger"
              link
              size="small"
              @click.stop="handleCancel(row)"
            >
              取消
            </el-button>
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

    <el-dialog
      v-model="taskDetailVisible"
      :title="taskDetailTitle"
      width="960px"
      destroy-on-close
    >
      <template v-if="currentTask">
        <div class="detail-overview">
          <div class="overview-item">
            <span class="overview-label">任务名称</span>
            <span class="overview-value">{{ currentTask.name || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">函数路径</span>
            <span class="overview-value">{{ currentTask.full_code_path }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">执行动作</span>
            <span class="overview-value">{{ actionLabel(currentTask.action) }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">请求方法</span>
            <span class="overview-value">{{ currentTask.method || 'POST' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">调度方式</span>
            <span class="overview-value">
              {{ scheduleTypeLabel(currentTask.schedule_type) }} / {{ getScheduleSummary(currentTask) }}
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">执行身份</span>
            <span class="overview-value">{{ currentTask.request_user || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">创建者</span>
            <span class="overview-value">{{ currentTask.created_by || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">当前状态</span>
            <span class="overview-value">
              <el-tag :type="statusTagType(currentTask.status)" size="small">
                {{ statusLabel(currentTask.status) }}
              </el-tag>
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">{{ runAtLabel(currentTask) }}</span>
            <span class="overview-value">{{ formatDateTime(currentTask.run_at) }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">下次执行</span>
            <span class="overview-value">
              {{ currentTask.next_run_at ? formatDateTime(currentTask.next_run_at) : '-' }}
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">时区</span>
            <span class="overview-value">{{ currentTask.timezone || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">已执行次数</span>
            <span class="overview-value">{{ currentTask.run_count || 0 }}</span>
          </div>
        </div>

        <el-alert
          v-if="currentTask.error_message"
          title="最近一次失败原因"
          :description="currentTask.error_message"
          type="warning"
          show-icon
          :closable="false"
          class="detail-alert"
        />

        <div class="payload-section">
          <div class="payload-header">
            <span class="payload-title">任务请求参数</span>
            <span class="payload-tip">创建任务时保存的请求体</span>
          </div>
          <pre class="payload-pre">{{ formatPayload(currentTask.payload) }}</pre>
        </div>
      </template>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="taskDetailVisible = false">关闭</el-button>
          <el-button type="primary" @click="openExecutionsFromDetail">查看执行记录</el-button>
          <el-button
            v-if="currentTask?.status === 'pending'"
            type="danger"
            @click="handleCancelFromDetail"
          >
            取消任务
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="executionsVisible"
      :title="`执行记录：${currentTaskName}`"
      width="88%"
      destroy-on-close
    >
      <div class="execution-toolbar">
        <el-form :inline="true" :model="executionFilterForm" class="filter-form">
          <el-form-item label="状态">
            <el-select
              v-model="executionFilterForm.status"
              placeholder="全部状态"
              clearable
              style="width: 160px"
              @change="handleExecutionFilterChange"
            >
              <el-option label="全部状态" value="" />
              <el-option label="成功" value="success" />
              <el-option label="失败" value="failed" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button @click="loadExecutions">刷新</el-button>
          </el-form-item>
        </el-form>
        <span class="section-total">共 {{ executionsTotal }} 条记录</span>
      </div>

      <el-empty
        v-if="!executionsLoading && executions.length === 0"
        :description="executionFilterForm.status ? '当前筛选条件下暂无执行记录' : '暂无执行记录'"
      />
      <el-table
        v-else
        :data="executions"
        stripe
        class="execution-table"
        v-loading="executionsLoading"
        @row-click="handleExecutionRowClick"
      >
        <el-table-column prop="executed_at" label="执行时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.executed_at) }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="120" align="center">
          <template #default="{ row }">
            <span v-if="!hasExecutionDuration(row)" class="duration-empty">-</span>
            <el-tooltip v-else :content="getExecutionDurationTip(row)" placement="top" effect="light">
              <el-tag :type="getExecutionDurationTagType(row)" size="small" effect="light">
                {{ formatExecutionDuration(row) }}
              </el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="trace_id" label="Trace ID" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ row.trace_id || '-' }}</template>
        </el-table-column>
        <el-table-column prop="error_message" label="错误信息" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">{{ row.error_message || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click.stop="openExecutionDetail(row)">
              详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="executionsTotal > 0"
        v-model:current-page="executionsPage"
        v-model:page-size="executionsPageSize"
        :total="executionsTotal"
        layout="total, prev, pager, next"
        class="executions-pagination"
        @current-change="loadExecutions"
        @size-change="handleExecutionPageSizeChange"
      />
    </el-dialog>

    <el-dialog
      v-model="executionDetailVisible"
      title="执行详情"
      width="720px"
      destroy-on-close
    >
      <template v-if="currentExecution">
        <div class="detail-overview execution-overview">
          <div class="overview-item">
            <span class="overview-label">执行时间</span>
            <span class="overview-value">{{ formatDateTime(currentExecution.executed_at) }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">执行状态</span>
            <span class="overview-value">
              <el-tag :type="currentExecution.status === 'success' ? 'success' : 'danger'" size="small">
                {{ statusLabel(currentExecution.status) }}
              </el-tag>
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">Trace ID</span>
            <span class="overview-value">{{ currentExecution.trace_id || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">耗时</span>
            <span class="overview-value">
              <span v-if="!hasExecutionDuration(currentExecution)">-</span>
              <el-tooltip v-else :content="getExecutionDurationTip(currentExecution)" placement="top" effect="light">
                <el-tag :type="getExecutionDurationTagType(currentExecution)" size="small" effect="light">
                  {{ formatExecutionDuration(currentExecution) }}
                </el-tag>
              </el-tooltip>
            </span>
          </div>
        </div>

        <el-alert
          v-if="currentExecution.error_message"
          title="执行失败"
          :description="currentExecution.error_message"
          type="error"
          show-icon
          :closable="false"
          class="detail-alert"
        />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listScheduledTasks,
  cancelScheduledTask,
  listScheduledTaskExecutions,
  type ScheduledTaskItem,
  type ScheduledTaskExecutionItem
} from '@/api/scheduledTask'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'

const props = withDefaults(
  defineProps<{
    resourcePath?: string
    autoLoad?: boolean
  }>(),
  { autoLoad: false }
)
const emit = defineEmits<{ (e: 'total-change', total: number): void }>()

const loading = ref(false)
const list = ref<ScheduledTaskItem[]>([])
const total = ref(0)
const resourceTotal = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filterForm = ref({
  status: ''
})

function scheduleTypeLabel(type: string) {
  const mapping: Record<string, string> = {
    atime: '指定时间',
    cron: 'Cron',
    every: '每 N 秒'
  }
  return mapping[type] ?? type
}

function actionLabel(action?: string) {
  const mapping: Record<string, string> = {
    execute: '普通执行',
    table_create: '表格新增',
    table_update: '表格更新',
    table_delete: '表格删除'
  }
  return action ? (mapping[action] ?? action) : '普通执行'
}

function statusTagType(status: string) {
  const mapping: Record<string, string> = {
    pending: 'warning',
    done: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return mapping[status] ?? 'info'
}

function statusLabel(status: string) {
  const mapping: Record<string, string> = {
    pending: '待执行',
    done: '已完成',
    failed: '失败',
    cancelled: '已取消',
    success: '成功'
  }
  return mapping[status] ?? status
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return '-'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString('zh-CN')
}

function formatPayload(raw?: string | null) {
  if (!raw) {
    return '{}'
  }
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

function parseObjectPayload(raw?: string | null): Record<string, any> | null {
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, any>
    }
  } catch {
    return null
  }
  return null
}

function readNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '' && !Number.isNaN(Number(value))) {
    return Number(value)
  }
  return null
}

function getScheduleSummary(task: ScheduledTaskItem) {
  switch (task.schedule_type) {
    case 'cron':
      return task.cron_expr ? `Cron：${task.cron_expr}` : '按 Cron 表达式重复执行'
    case 'every': {
      const seconds = task.interval_seconds || 0
      const runLimit = task.max_runs ? `，最多执行 ${task.max_runs} 次` : '，不限制次数'
      return `每 ${seconds} 秒执行一次${runLimit}`
    }
    case 'atime':
    default:
      return '到点执行一次'
  }
}

function runAtLabel(task: ScheduledTaskItem) {
  return task.schedule_type === 'atime' ? '执行时间' : '开始时间'
}

function getExecutionDuration(execution: ScheduledTaskExecutionItem): number | null {
  const direct = readNumber(execution.duration_millis)
  if (direct !== null && direct >= 0) {
    return direct
  }
  const payload = parseObjectPayload(execution.response_payload)
  const topLevel = readNumber(payload?.total_cost_mill)
  if (topLevel !== null && topLevel >= 0) {
    return topLevel
  }
  const result = payload?.result
  if (result && typeof result === 'object' && !Array.isArray(result)) {
    const nested = readNumber((result as Record<string, unknown>).total_cost_mill)
    if (nested !== null && nested >= 0) {
      return nested
    }
  }
  return null
}

function hasExecutionDuration(execution: ScheduledTaskExecutionItem): boolean {
  return getExecutionDuration(execution) !== null
}

function formatDuration(value: number | null): string {
  if (value === null || value < 0) {
    return '-'
  }
  if (value < 1000) {
    return `${value}ms`
  }
  if (value < 60000) {
    return `${(value / 1000).toFixed(value < 10000 ? 2 : 1)}s`
  }
  const minutes = Math.floor(value / 60000)
  const seconds = ((value % 60000) / 1000).toFixed(1)
  return `${minutes}分${seconds}秒`
}

function formatExecutionDuration(execution: ScheduledTaskExecutionItem): string {
  return formatDuration(getExecutionDuration(execution))
}

function getExecutionDurationTone(execution: ScheduledTaskExecutionItem): 'fast' | 'medium' | 'slow' | 'unknown' {
  const duration = getExecutionDuration(execution)
  if (duration === null) {
    return 'unknown'
  }
  if (duration < 1000) {
    return 'fast'
  }
  if (duration < 3000) {
    return 'medium'
  }
  return 'slow'
}

function getExecutionDurationTagType(execution: ScheduledTaskExecutionItem) {
  switch (getExecutionDurationTone(execution)) {
    case 'fast':
      return 'success'
    case 'medium':
      return 'warning'
    case 'slow':
      return 'danger'
    default:
      return 'info'
  }
}

function getExecutionDurationTip(execution: ScheduledTaskExecutionItem): string {
  const duration = getExecutionDuration(execution)
  if (duration === null) {
    return '未记录耗时'
  }
  switch (getExecutionDurationTone(execution)) {
    case 'fast':
      return `执行较快：${formatDuration(duration)}`
    case 'medium':
      return `执行中等：${formatDuration(duration)}`
    case 'slow':
      return `执行较慢：${formatDuration(duration)}`
    default:
      return formatDuration(duration)
  }
}

function emitResourceTotal(totalValue: number) {
  resourceTotal.value = totalValue
  emit('total-change', totalValue)
}

async function loadList() {
  if (!props.resourcePath) {
    list.value = []
    total.value = 0
    emitResourceTotal(0)
    return
  }

  loading.value = true
  try {
    const filteredParams = {
      full_code_path: props.resourcePath,
      status: filterForm.value.status || undefined,
      page: page.value,
      page_size: pageSize.value
    }

    if (filterForm.value.status) {
      const [filteredRes, baseRes] = await Promise.all([
        listScheduledTasks(filteredParams),
        listScheduledTasks({
          full_code_path: props.resourcePath,
          page: 1,
          page_size: 1
        })
      ])
      list.value = filteredRes.list ?? []
      total.value = filteredRes.total ?? 0
      emitResourceTotal(baseRes.total ?? 0)
      return
    }

    const res = await listScheduledTasks(filteredParams)
    list.value = res.list ?? []
    total.value = res.total ?? 0
    emitResourceTotal(res.total ?? 0)
  } catch {
    list.value = []
    total.value = 0
    emitResourceTotal(0)
  } finally {
    loading.value = false
  }
}

function handleFilterChange() {
  page.value = 1
  loadList()
}

function handlePageSizeChange() {
  page.value = 1
  loadList()
}

function resetFilters() {
  filterForm.value.status = ''
  page.value = 1
  loadList()
}

watch(
  () => [props.resourcePath, props.autoLoad] as const,
  ([path, auto]) => {
    if (path && auto) {
      page.value = 1
      loadList()
    } else if (!path) {
      list.value = []
      total.value = 0
      emitResourceTotal(0)
    }
  },
  { immediate: true }
)

const unsubscribeScheduledTaskCreated = eventBus.on(WorkspaceEvent.scheduledTaskCreated, () => {
  if (props.resourcePath) {
    loadList()
  }
})

onUnmounted(() => {
  unsubscribeScheduledTaskCreated()
})

function handleTaskRowClick(row: ScheduledTaskItem) {
  openTaskDetail(row)
}

function handleCancel(row: ScheduledTaskItem) {
  ElMessageBox.confirm(`确定取消定时任务「${row.name}」？`, '取消任务', {
    type: 'warning'
  })
    .then(async () => {
      try {
        await cancelScheduledTask(row.id)
        ElMessage.success('已取消')
        if (currentTask.value?.id === row.id) {
          taskDetailVisible.value = false
        }
        await loadList()
      } catch (error: any) {
        ElMessage.error(error?.message || '取消失败')
      }
    })
    .catch(() => {})
}

const taskDetailVisible = ref(false)
const currentTask = ref<ScheduledTaskItem | null>(null)

const taskDetailTitle = ref('任务详情')

function openTaskDetail(row: ScheduledTaskItem) {
  currentTask.value = row
  taskDetailTitle.value = row.name ? `任务详情：${row.name}` : '任务详情'
  taskDetailVisible.value = true
}

function handleCancelFromDetail() {
  if (!currentTask.value) {
    return
  }
  handleCancel(currentTask.value)
}

function openExecutionsFromDetail() {
  if (!currentTask.value) {
    return
  }
  taskDetailVisible.value = false
  openExecutions(currentTask.value)
}

const executionsVisible = ref(false)
const executions = ref<ScheduledTaskExecutionItem[]>([])
const executionsTotal = ref(0)
const executionsPage = ref(1)
const executionsPageSize = ref(20)
const executionsLoading = ref(false)
const currentTaskId = ref(0)
const currentTaskName = ref('')
const executionFilterForm = ref({
  status: ''
})

function openExecutions(row: ScheduledTaskItem) {
  currentTaskId.value = row.id
  currentTaskName.value = row.name || '未命名任务'
  executionsPage.value = 1
  executionFilterForm.value.status = ''
  executionsVisible.value = true
  loadExecutions()
}

async function loadExecutions() {
  if (!currentTaskId.value) {
    return
  }
  executionsLoading.value = true
  try {
    const res = await listScheduledTaskExecutions(currentTaskId.value, {
      status: executionFilterForm.value.status || undefined,
      page: executionsPage.value,
      page_size: executionsPageSize.value
    })
    executions.value = res.list ?? []
    executionsTotal.value = res.total ?? 0
  } catch {
    executions.value = []
    executionsTotal.value = 0
  } finally {
    executionsLoading.value = false
  }
}

function handleExecutionFilterChange() {
  executionsPage.value = 1
  loadExecutions()
}

function handleExecutionPageSizeChange() {
  executionsPage.value = 1
  loadExecutions()
}

const executionDetailVisible = ref(false)
const currentExecution = ref<ScheduledTaskExecutionItem | null>(null)

function handleExecutionRowClick(row: ScheduledTaskExecutionItem) {
  openExecutionDetail(row)
}

function openExecutionDetail(row: ScheduledTaskExecutionItem) {
  currentExecution.value = row
  executionDetailVisible.value = true
}
</script>

<style scoped>
.scheduled-task-list {
  padding: 20px;
}

.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.section-copy {
  min-width: 0;
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

.section-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.section-total {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.filter-section {
  display: flex;
  align-items: flex-start;
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

.filter-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  white-space: nowrap;
}

.table-section {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.task-table :deep(.el-table__row) {
  cursor: pointer;
}

.task-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.5;
}

.schedule-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.schedule-summary-trigger {
  font-size: 12px;
  color: var(--el-color-primary);
  cursor: help;
  white-space: nowrap;
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.error-dot {
  display: inline-flex;
  align-items: center;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-color-danger);
}

.duration-empty {
  color: var(--el-text-color-placeholder);
}

.list-pagination,
.executions-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}

.detail-overview {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 20px;
}

.overview-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--el-fill-color-light);
}

.overview-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.overview-value {
  font-size: 13px;
  color: var(--el-text-color-primary);
  line-height: 1.6;
  word-break: break-all;
}

.detail-alert {
  margin-top: 16px;
}

.payload-section,
.payload-panel {
  margin-top: 18px;
}

.payload-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 18px;
}

.payload-panel {
  min-width: 0;
}

.payload-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.payload-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.payload-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.payload-pre {
  margin: 0;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 52vh;
  overflow: auto;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.execution-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 8px;
}

.execution-table :deep(.el-table__row) {
  cursor: pointer;
}

.execution-overview {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

@media (max-width: 960px) {
  .scheduled-task-list {
    padding: 16px;
  }

  .section-header,
  .filter-section,
  .execution-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .section-actions {
    justify-content: space-between;
  }

  .filter-summary {
    white-space: normal;
  }

  .detail-overview,
  .execution-overview,
  .payload-grid {
    grid-template-columns: 1fr;
  }
}
</style>
