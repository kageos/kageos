import { onUnmounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import {
  listScheduledTasks,
  cancelScheduledTask,
  listScheduledTaskExecutions,
  type ScheduledTaskItem,
  type ScheduledTaskExecutionItem
} from '@/api/scheduledTask'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import {
  formatExecutionDateTime,
  parseExecutionObject,
  readExecutionNumber
} from '@/architecture/presentation/utils/executionLog'
import { getErrorMessage } from '@/utils/apiError'

export interface ScheduledTaskListProps {
  resourcePath?: string
  autoLoad?: boolean
}

export interface ScheduledTaskOperateLogPayload {
  source: 'scheduled_task'
  traceId?: string
}

export interface ScheduledTaskListEmit {
  (e: 'total-change', total: number): void
  (e: 'open-function-operate-log', payload: ScheduledTaskOperateLogPayload): void
}

export function useScheduledTaskList(
  props: ScheduledTaskListProps,
  emit: ScheduledTaskListEmit
) {
  const authStore = useAuthStore()

  const loading = ref(false)
  const list = ref<ScheduledTaskItem[]>([])
  const total = ref(0)
  const resourceTotal = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const filterForm = ref({
    status: ''
  })

  const taskDetailVisible = ref(false)
  const currentTask = ref<ScheduledTaskItem | null>(null)
  const taskDetailTitle = ref('任务详情')

  const executionsVisible = ref(false)
  const executions = ref<ScheduledTaskExecutionItem[]>([])
  const executionsTotal = ref(0)
  const executionsPage = ref(1)
  const executionsPageSize = ref(20)
  const executionsLoading = ref(false)
  const currentTaskId = ref(0)
  const currentTaskName = ref('')
  const currentExecutionTask = ref<ScheduledTaskItem | null>(null)
  const executionFilterForm = ref({
    status: ''
  })

  const executionDetailVisible = ref(false)
  const currentExecution = ref<ScheduledTaskExecutionItem | null>(null)

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
    return formatExecutionDateTime(value)
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

  function parseObjectPayload(raw?: string | null): Record<string, unknown> | null {
    return parseExecutionObject(raw)
  }

  function getScheduleSummary(task: ScheduledTaskItem) {
    switch (task.schedule_type) {
      case 'cron':
        return task.cron_expr ? `Cron：${task.cron_expr}（从下一次命中开始）` : '按 Cron 表达式重复执行'
      case 'every': {
        const seconds = task.interval_seconds || 0
        const runLimit = task.max_runs ? `，最多执行 ${task.max_runs} 次` : '，不限制次数'
        return `创建后立即执行一次，每 ${seconds} 秒执行一次${runLimit}`
      }
      case 'atime':
      default:
        return '到点执行一次'
    }
  }

  function runAtLabel(task: ScheduledTaskItem) {
    return task.schedule_type === 'atime' ? '执行时间' : '生效时间'
  }

  function getExecutionDuration(execution: ScheduledTaskExecutionItem): number | null {
    const direct = readExecutionNumber(execution.duration_millis)
    if (direct !== null && direct >= 0) {
      return direct
    }
    const payload = parseObjectPayload(execution.response_payload)
    const topLevel = readExecutionNumber(payload?.total_cost_mill)
    if (topLevel !== null && topLevel >= 0) {
      return topLevel
    }
    const result = payload?.result
    if (result && typeof result === 'object' && !Array.isArray(result)) {
      const nested = readExecutionNumber((result as Record<string, unknown>).total_cost_mill)
      if (nested !== null && nested >= 0) {
        return nested
      }
    }
    return null
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

  function canCancelTask(task?: ScheduledTaskItem | null): boolean {
    if (!task || task.status !== 'pending') {
      return false
    }
    const currentUser = authStore.userName?.trim()
    const createdBy = task.created_by?.trim()
    return !!currentUser && currentUser === createdBy
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
        } catch (error: unknown) {
          ElMessage.error(getErrorMessage(error, '取消失败'))
        }
      })
      .catch(() => {})
  }

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

  function openExecutions(row: ScheduledTaskItem) {
    currentTaskId.value = row.id
    currentTaskName.value = row.name || '未命名任务'
    currentExecutionTask.value = row
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

  function handleExecutionRowClick(row: ScheduledTaskExecutionItem) {
    openExecutionDetail(row)
  }

  function openExecutionDetail(row: ScheduledTaskExecutionItem) {
    currentExecution.value = row
    executionDetailVisible.value = true
  }

  function normalizeTaskAction(action?: string) {
    const normalized = (action || 'execute').trim().toLowerCase()
    return normalized === 'form' ? 'execute' : normalized
  }

  function canOpenFunctionOperateLog(task?: ScheduledTaskItem | null): boolean {
    return normalizeTaskAction(task?.action) === 'execute'
  }

  function openFunctionOperateLog(execution: ScheduledTaskExecutionItem) {
    emit('open-function-operate-log', {
      source: 'scheduled_task',
      traceId: execution.trace_id || undefined
    })
    executionDetailVisible.value = false
    executionsVisible.value = false
  }

  return {
    loading,
    list,
    total,
    resourceTotal,
    page,
    pageSize,
    filterForm,
    taskDetailVisible,
    currentTask,
    taskDetailTitle,
    executionsVisible,
    executions,
    executionsTotal,
    executionsPage,
    executionsPageSize,
    executionsLoading,
    currentTaskName,
    currentExecutionTask,
    executionFilterForm,
    executionDetailVisible,
    currentExecution,
    scheduleTypeLabel,
    actionLabel,
    statusTagType,
    statusLabel,
    formatDateTime,
    formatPayload,
    getScheduleSummary,
    runAtLabel,
    getExecutionDuration,
    loadList,
    handleFilterChange,
    handlePageSizeChange,
    resetFilters,
    handleTaskRowClick,
    canCancelTask,
    handleCancel,
    openTaskDetail,
    handleCancelFromDetail,
    openExecutionsFromDetail,
    openExecutions,
    loadExecutions,
    handleExecutionFilterChange,
    handleExecutionPageSizeChange,
    handleExecutionRowClick,
    openExecutionDetail,
    canOpenFunctionOperateLog,
    openFunctionOperateLog
  }
}
