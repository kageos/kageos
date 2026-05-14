import { computed, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import {
  listScheduledTasks,
  getScheduledTask,
  cancelScheduledTask,
  deleteScheduledTask,
  listScheduledTaskExecutions,
  getScheduledTaskExecution,
  type ScheduledTaskItem,
  type ScheduledTaskExecutionItem
} from '@/architecture/infrastructure/api/scheduledTask'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import {
  formatExecutionDateTime,
  parseExecutionObject,
  readExecutionNumber
} from '@/architecture/presentation/utils/executionLog'
import { getErrorMessage } from '@/architecture/runtime/utils/apiError'

export interface ScheduledTaskListProps {
  resourcePath?: string
  autoLoad?: boolean
}

export interface ScheduledTaskOperateLogPayload {
  source: 'scheduled_task'
  traceId?: string
}

export interface ScheduledTaskReplayContext {
  source: 'scheduled_task'
  title?: string
  taskId?: number
  executionId?: number
  traceId?: string
  executedAt?: string
}

export interface ScheduledTaskReplayPayload {
  requestBody?: Record<string, any> | null
  responseBody?: Record<string, any> | null
  responseMetadata?: Record<string, any> | null
  replayContext?: ScheduledTaskReplayContext | null
}

export interface ScheduledTaskListEmit {
  (e: 'total-change', total: number): void
  (e: 'open-function-operate-log', payload: ScheduledTaskOperateLogPayload): void
  (e: 'apply-execution', payload: ScheduledTaskReplayPayload): void
}

interface InlineExecutionState {
  loading: boolean
  loaded: boolean
  list: ScheduledTaskExecutionItem[]
  total: number
  page: number
  pageSize: number
  status: string
  error: string
}

export function useScheduledTaskList(
  props: ScheduledTaskListProps,
  emit: ScheduledTaskListEmit
) {
  const authStore = useAuthStore()
  const route = useRoute()

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
  const inlineExecutionStates = ref<Record<number, InlineExecutionState>>({})
  const executionFilterForm = ref({
    status: ''
  })

  const executionDetailVisible = ref(false)
  const currentExecution = ref<ScheduledTaskExecutionItem | null>(null)
  const appliedDeepLinkKey = ref('')
  const applyingDeepLink = ref(false)

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

  function notifyOnLabel(notifyOn?: string) {
    const mapping: Record<string, string> = {
      none: '不通知',
      all: '每次完成',
      success: '仅成功',
      failed: '仅失败'
    }
    return mapping[notifyOn || 'none'] ?? notifyOn
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
      } else {
        const res = await listScheduledTasks(filteredParams)
        list.value = res.list ?? []
        total.value = res.total ?? 0
        emitResourceTotal(res.total ?? 0)
      }
    } catch {
      list.value = []
      total.value = 0
      emitResourceTotal(0)
    } finally {
      loading.value = false
    }
    await applyExecutionDeepLink()
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

  function canCancelTask(task?: ScheduledTaskItem | null): boolean {
    if (!task || task.status !== 'pending') {
      return false
    }
    const currentUser = authStore.userName?.trim()
    const createdBy = task.created_by?.trim()
    return !!currentUser && currentUser === createdBy
  }

  function canDeleteTask(task?: ScheduledTaskItem | null): boolean {
    if (!task) {
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

  function handleDelete(row: ScheduledTaskItem) {
    ElMessageBox.confirm(`确定删除定时任务「${row.name || '未命名任务'}」？删除后任务将从列表移除。`, '删除任务', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      confirmButtonClass: 'el-button--danger'
    })
      .then(async () => {
        try {
          await deleteScheduledTask(row.id)
          ElMessage.success('已删除')
          if (currentTask.value?.id === row.id) {
            taskDetailVisible.value = false
          }
          await loadList()
        } catch (error: unknown) {
          ElMessage.error(getErrorMessage(error, '删除失败'))
        }
      })
      .catch(() => {})
  }

  function openTaskDetail(row: ScheduledTaskItem) {
    currentTask.value = row
    taskDetailTitle.value = row.name ? `任务详情：${row.name}` : '任务详情'
    taskDetailVisible.value = true
  }

  function createInlineExecutionState(): InlineExecutionState {
    return {
      loading: false,
      loaded: false,
      list: [],
      total: 0,
      page: 1,
      pageSize: 5,
      status: '',
      error: ''
    }
  }

  function ensureInlineExecutionState(taskID: number): InlineExecutionState {
    if (!inlineExecutionStates.value[taskID]) {
      inlineExecutionStates.value[taskID] = createInlineExecutionState()
    }
    return inlineExecutionStates.value[taskID]
  }

  function getInlineExecutionState(taskID: number): InlineExecutionState {
    return ensureInlineExecutionState(taskID)
  }

  async function loadInlineExecutions(row: ScheduledTaskItem, force = false) {
    const state = ensureInlineExecutionState(row.id)
    if (state.loading || (state.loaded && !force)) {
      return
    }

    state.loading = true
    state.error = ''
    try {
      const res = await listScheduledTaskExecutions(row.id, {
        status: state.status || undefined,
        page: state.page,
        page_size: state.pageSize
      })
      state.list = res.list ?? []
      state.total = res.total ?? 0
      state.loaded = true
    } catch (error: unknown) {
      state.list = []
      state.total = 0
      state.error = getErrorMessage(error, '执行记录加载失败')
    } finally {
      state.loading = false
    }
  }

  function handleTaskExpandChange(row: ScheduledTaskItem, expandedRows: ScheduledTaskItem[] | boolean) {
    if (!Array.isArray(expandedRows)) {
      return
    }
    const expanded = expandedRows.some((item) => item.id === row.id)
    if (expanded) {
      loadInlineExecutions(row)
    }
  }

  function refreshInlineExecutions(row: ScheduledTaskItem) {
    loadInlineExecutions(row, true)
  }

  function handleInlineExecutionStatusChange(row: ScheduledTaskItem) {
    const state = ensureInlineExecutionState(row.id)
    state.page = 1
    loadInlineExecutions(row, true)
  }

  function handleInlineExecutionPageChange(row: ScheduledTaskItem, nextPage: number) {
    const state = ensureInlineExecutionState(row.id)
    if (!Number.isFinite(nextPage) || nextPage < 1 || state.page === nextPage) {
      return
    }
    state.page = nextPage
    loadInlineExecutions(row, true)
  }

  function handleCancelFromDetail() {
    if (!currentTask.value) {
      return
    }
    handleCancel(currentTask.value)
  }

  function handleDeleteFromDetail() {
    if (!currentTask.value) {
      return
    }
    handleDelete(currentTask.value)
  }

  async function openExecutionsFromDetail() {
    if (!currentTask.value) {
      return
    }
    taskDetailVisible.value = false
    await openExecutions(currentTask.value)
  }

  async function openExecutions(row: ScheduledTaskItem) {
    currentTaskId.value = row.id
    currentTaskName.value = row.name || '未命名任务'
    currentExecutionTask.value = row
    executionsPage.value = 1
    executionFilterForm.value.status = ''
    executionsVisible.value = true
    await loadExecutions()
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

  function openExecutionDetail(row: ScheduledTaskExecutionItem, task?: ScheduledTaskItem) {
    if (task) {
      currentTaskId.value = task.id
      currentTaskName.value = task.name || '未命名任务'
      currentExecutionTask.value = task
    }
    currentExecution.value = row
    executionDetailVisible.value = true
  }

  function openInlineExecutionDetail(task: ScheduledTaskItem, execution: ScheduledTaskExecutionItem) {
    openExecutionDetail(execution, task)
  }

  function normalizeQueryValue(value: unknown): string {
    if (Array.isArray(value)) {
      return value[0] ? String(value[0]) : ''
    }
    return value ? String(value) : ''
  }

  function readQueryID(...keys: string[]): number {
    for (const key of keys) {
      const raw = normalizeQueryValue(route.query[key])
      if (!raw) {
        continue
      }
      const parsed = Number(raw)
      if (Number.isFinite(parsed) && parsed > 0) {
        return parsed
      }
    }
    return 0
  }

  async function applyExecutionDeepLink() {
    if (!props.autoLoad || !props.resourcePath) {
      return
    }
    if (normalizeQueryValue(route.query._panel) !== 'scheduledTask') {
      return
    }
    const taskID = readQueryID('_scheduled_task_id', '_task_id')
    if (!taskID || applyingDeepLink.value) {
      return
    }
    const executionID = readQueryID('_scheduled_execution_id', '_execution_id')
    const linkKey = `${props.resourcePath}:${taskID}:${executionID}`
    if (appliedDeepLinkKey.value === linkKey) {
      return
    }

    applyingDeepLink.value = true
    appliedDeepLinkKey.value = linkKey
    try {
      const task = list.value.find((item) => item.id === taskID) || await getScheduledTask(taskID)
      await openExecutions(task)
      if (!executionID) {
        return
      }
      const execution = executions.value.find((item) => item.id === executionID) ||
        await getScheduledTaskExecution(taskID, executionID)
      openExecutionDetail(execution)
    } catch (error: unknown) {
      ElMessage.error(getErrorMessage(error, '打开执行结果失败'))
    } finally {
      applyingDeepLink.value = false
    }
  }

  const currentExecutionRequestPayload = computed(() => parseObjectPayload(currentExecution.value?.request_payload))

  const currentExecutionResponseEnvelope = computed(() => parseObjectPayload(currentExecution.value?.response_payload))

  const currentExecutionResponsePayload = computed(() => {
    const envelope = currentExecutionResponseEnvelope.value
    if (!envelope) {
      return null
    }
    const result = envelope.result
    if (result !== undefined && result !== null) {
      return result
    }
    return envelope
  })

  const currentExecutionResponseMetadata = computed(() => {
    const envelope = currentExecutionResponseEnvelope.value || {}
    return {
      version: envelope.version,
      total_cost_mill: envelope.total_cost_mill ?? currentExecution.value?.duration_millis,
      err_code: envelope.err_code,
      error: envelope.error || currentExecution.value?.error_message,
      trace_id: envelope.trace_id || currentExecution.value?.trace_id
    }
  })

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

  function buildScheduledTaskReplayPayload(
    task: ScheduledTaskItem | null | undefined,
    execution: ScheduledTaskExecutionItem
  ): ScheduledTaskReplayPayload {
    const requestBody = parseObjectPayload(execution.request_payload)
    const responseEnvelope = parseObjectPayload(execution.response_payload)
    const result = responseEnvelope?.result
    const responseBody = result !== undefined && result !== null
      ? (result && typeof result === 'object' && !Array.isArray(result) ? result as Record<string, any> : { result })
      : responseEnvelope

    const responseMetadata: Record<string, any> = {
      scheduled_task_id: task?.id || execution.task_id,
      scheduled_task_execution_id: execution.id,
      replay_source: 'scheduled_task'
    }
    if (responseEnvelope?.version) {
      responseMetadata.version = responseEnvelope.version
    }
    if (responseEnvelope?.total_cost_mill !== undefined || execution.duration_millis !== undefined) {
      responseMetadata.total_cost_mill = responseEnvelope?.total_cost_mill ?? execution.duration_millis
    }
    if (responseEnvelope?.err_code !== undefined) {
      responseMetadata.err_code = responseEnvelope.err_code
    }
    if (responseEnvelope?.error || execution.error_message) {
      responseMetadata.error = responseEnvelope?.error || execution.error_message
    }
    if (responseEnvelope?.trace_id || execution.trace_id) {
      responseMetadata.trace_id = responseEnvelope?.trace_id || execution.trace_id
    }

    return {
      requestBody,
      responseBody,
      responseMetadata,
      replayContext: {
        source: 'scheduled_task',
        title: '定时任务记录回填',
        taskId: task?.id || execution.task_id,
        executionId: execution.id,
        traceId: execution.trace_id || responseMetadata.trace_id,
        executedAt: execution.executed_at
      }
    }
  }

  function applyExecutionToForm(execution: ScheduledTaskExecutionItem, task?: ScheduledTaskItem | null) {
    const sourceTask = task || currentExecutionTask.value
    emit('apply-execution', buildScheduledTaskReplayPayload(sourceTask, execution))
    executionDetailVisible.value = false
    executionsVisible.value = false
  }

  watch(
    () => [
      props.resourcePath,
      props.autoLoad,
      route.query._panel,
      route.query._scheduled_task_id,
      route.query._scheduled_execution_id,
      route.query._task_id,
      route.query._execution_id
    ] as const,
    () => {
      applyExecutionDeepLink()
    },
    { immediate: true }
  )

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
    getInlineExecutionState,
    executionFilterForm,
    executionDetailVisible,
    currentExecution,
    currentExecutionRequestPayload,
    currentExecutionResponsePayload,
    currentExecutionResponseMetadata,
    scheduleTypeLabel,
    actionLabel,
    statusTagType,
    statusLabel,
    notifyOnLabel,
    formatDateTime,
    formatPayload,
    getScheduleSummary,
    runAtLabel,
    getExecutionDuration,
    loadList,
    handleFilterChange,
    handlePageSizeChange,
    resetFilters,
    canCancelTask,
    canDeleteTask,
    handleCancel,
    handleDelete,
    openTaskDetail,
    handleCancelFromDetail,
    handleDeleteFromDetail,
    openExecutionsFromDetail,
    handleTaskExpandChange,
    refreshInlineExecutions,
    handleInlineExecutionStatusChange,
    handleInlineExecutionPageChange,
    openExecutions,
    loadExecutions,
    handleExecutionFilterChange,
    handleExecutionPageSizeChange,
    handleExecutionRowClick,
    openExecutionDetail,
    openInlineExecutionDetail,
    canOpenFunctionOperateLog,
    openFunctionOperateLog,
    applyExecutionToForm
  }
}
