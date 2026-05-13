import { computed, onUnmounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { cancelWorkspaceChat, getWorkspaceSessions, type WorkspaceSessionItem } from '@/api/workspace'
import {
  listScheduledAgentExecutions,
  listScheduledAgentTasks,
  runScheduledAgentTaskNow,
  type ScheduledAgentExecutionItem,
  type ScheduledAgentTaskItem
} from '@/api/scheduledAgentTask'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import { featureFlags } from '@/config/features'

type SidebarTab = 'all' | 'running' | 'finished' | 'scheduled'

const SCHEDULED_AGENT_TASK_PAGE_SIZE = 100
const SCHEDULED_AGENT_EXECUTION_PAGE_SIZE = 100
const SCHEDULED_AGENT_EXECUTION_BATCH_SIZE = 8

export interface WorkspaceSidebarContext {
  fullCodePath: string
  dirName: string
}

export interface UseWorkspaceSidebarSessionsOptions {
  workstationContext: ComputedRef<WorkspaceSidebarContext | null>
  sidebarVisible: Ref<boolean>
  onOpenSession: (session: WorkspaceSessionItem) => void
}

export interface ScheduledAgentExecutionRecord {
  task: ScheduledAgentTaskItem
  execution: ScheduledAgentExecutionItem
}

export function useWorkspaceSidebarSessions(options: UseWorkspaceSidebarSessionsOptions) {
  const { workstationContext, sidebarVisible, onOpenSession } = options

  const sessions = ref<WorkspaceSessionItem[]>([])
  const scheduledAgentTasks = ref<ScheduledAgentTaskItem[]>([])
  const scheduledAgentExecutions = ref<ScheduledAgentExecutionRecord[]>([])
  const sessionsLoading = ref(false)
  const scheduledAgentTasksLoading = ref(false)
  const activeTab = ref<SidebarTab>('all')
  const sessionSearchKeyword = ref('')
  const cancellingTaskId = ref<string | null>(null)
  const scheduledAgentTaskActionId = ref<number | null>(null)

  let pollTimer: ReturnType<typeof setInterval> | null = null

  const runningCount = computed(() =>
    sessions.value.filter((s: WorkspaceSessionItem) => s.status === 'generating').length
  )

  const scheduledAgentTaskCount = computed(() =>
    featureFlags.scheduledTasks
      ? (
    scheduledAgentTasks.value.filter((task) => task.status === 'pending' || task.status === 'paused').length +
    scheduledAgentExecutions.value.length
      )
      : 0
  )

  async function loadSessions() {
    const ctx = workstationContext.value
    if (!ctx) {
      sessions.value = []
      return
    }

    sessionsLoading.value = true
    try {
      const res = await getWorkspaceSessions({ full_code_path: ctx.fullCodePath })
      sessions.value = res.sessions || []
    } catch {
      sessions.value = []
    } finally {
      sessionsLoading.value = false
    }
  }

  async function loadScheduledAgentTasks() {
    if (!featureFlags.scheduledTasks) {
      scheduledAgentTasks.value = []
      scheduledAgentExecutions.value = []
      return
    }
    const ctx = workstationContext.value
    if (!ctx) {
      scheduledAgentTasks.value = []
      scheduledAgentExecutions.value = []
      return
    }

    scheduledAgentTasksLoading.value = true
    try {
      const tasks = await loadAllScheduledAgentTasks(ctx.fullCodePath)
      scheduledAgentTasks.value = tasks
      await loadScheduledAgentExecutions(tasks)
    } catch {
      scheduledAgentTasks.value = []
      scheduledAgentExecutions.value = []
    } finally {
      scheduledAgentTasksLoading.value = false
    }
  }

  async function loadAllScheduledAgentTasks(fullCodePath: string) {
    const tasks: ScheduledAgentTaskItem[] = []
    let page = 1

    while (true) {
      const res = await listScheduledAgentTasks({
        full_code_path: fullCodePath,
        page,
        page_size: SCHEDULED_AGENT_TASK_PAGE_SIZE
      })
      const pageItems = res.list || []
      tasks.push(...pageItems)

      if (pageItems.length === 0 || tasks.length >= (res.total ?? tasks.length)) {
        break
      }
      page += 1
    }

    return tasks
  }

  async function loadScheduledAgentExecutions(tasks: ScheduledAgentTaskItem[]) {
    const records: ScheduledAgentExecutionRecord[] = []

    for (let index = 0; index < tasks.length; index += SCHEDULED_AGENT_EXECUTION_BATCH_SIZE) {
      const batch = tasks.slice(index, index + SCHEDULED_AGENT_EXECUTION_BATCH_SIZE)
      const settled = await Promise.allSettled(batch.map(loadAllScheduledAgentExecutions))
      records.push(...settled.flatMap((result) => result.status === 'fulfilled' ? result.value : []))
    }

    scheduledAgentExecutions.value = records
      .sort((a, b) => {
        const left = new Date(a.execution.started_at || a.execution.scheduled_at || a.execution.created_at).getTime()
        const right = new Date(b.execution.started_at || b.execution.scheduled_at || b.execution.created_at).getTime()
        return right - left
      })
  }

  async function loadAllScheduledAgentExecutions(task: ScheduledAgentTaskItem) {
    const executions: ScheduledAgentExecutionRecord[] = []
    let page = 1

    while (true) {
      const resp = await listScheduledAgentExecutions(task.id, {
        page,
        page_size: SCHEDULED_AGENT_EXECUTION_PAGE_SIZE
      })
      const pageItems = resp.list || []
      executions.push(...pageItems.map((execution) => ({ task, execution })))

      if (pageItems.length === 0 || executions.length >= (resp.total ?? executions.length)) {
        break
      }
      page += 1
    }

    return executions
  }

  function startPoll() {
    stopPoll()
    pollTimer = setInterval(() => {
      if (sessions.value.some((s: WorkspaceSessionItem) => s.status === 'generating')) {
        loadSessions()
      }
      if (featureFlags.scheduledTasks && activeTab.value === 'scheduled') {
        loadScheduledAgentTasks()
      }
    }, 5000)
  }

  function stopPoll() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function openSession(session: WorkspaceSessionItem) {
    onOpenSession(session)
  }

  function formatRelativeTime(timeStr: string) {
    const time = new Date(timeStr)
    const now = new Date()
    const diff = now.getTime() - time.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)
    if (minutes < 1) return '刚刚'
    if (minutes < 60) return `${minutes}分钟前`
    if (hours < 24) return `${hours}小时前`
    if (days < 7) return `${days}天前`
    return time.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }

  const filteredSessions = computed(() => {
    let list = sessions.value
    if (activeTab.value === 'running') {
      list = list.filter((s: WorkspaceSessionItem) => s.status === 'generating')
    } else if (activeTab.value === 'finished') {
      list = list.filter((s: WorkspaceSessionItem) => s.status === 'done' || s.status === 'cancelled')
    }

    const keyword = sessionSearchKeyword.value.trim().toLowerCase()
    if (!keyword) {
      return list
    }

    return list.filter((s: WorkspaceSessionItem) => {
      const title = (s.title || '').toLowerCase()
      const user = (s.user || '').toLowerCase()
      return title.includes(keyword) || user.includes(keyword)
    })
  })

  const filteredScheduledAgentTasks = computed(() => {
    const orderedTasks = scheduledAgentTasks.value.slice().sort((a, b) => {
      const statusOrder: Record<string, number> = {
        pending: 0,
        paused: 1,
        failed: 2,
        done: 3,
        cancelled: 4
      }
      const leftOrder = statusOrder[a.status] ?? 9
      const rightOrder = statusOrder[b.status] ?? 9
      if (leftOrder !== rightOrder) {
        return leftOrder - rightOrder
      }

      if (a.status === 'pending' || a.status === 'paused') {
        const leftNext = a.next_run_at ? new Date(a.next_run_at).getTime() : Number.MAX_SAFE_INTEGER
        const rightNext = b.next_run_at ? new Date(b.next_run_at).getTime() : Number.MAX_SAFE_INTEGER
        return leftNext - rightNext || b.id - a.id
      }

      const leftUpdated = new Date(a.updated_at || a.created_at).getTime()
      const rightUpdated = new Date(b.updated_at || b.created_at).getTime()
      return rightUpdated - leftUpdated || b.id - a.id
    })
    const keyword = sessionSearchKeyword.value.trim().toLowerCase()
    if (!keyword) {
      return orderedTasks
    }

    return orderedTasks.filter((task: ScheduledAgentTaskItem) => {
      return [
        task.name,
        task.goal,
        task.full_code_path,
        task.last_session_id,
        task.last_error_message,
        task.source_ref
      ].some((field) => (field || '').toLowerCase().includes(keyword))
    })
  })

  const filteredScheduledAgentExecutions = computed(() => {
    const keyword = sessionSearchKeyword.value.trim().toLowerCase()
    if (!keyword) {
      return scheduledAgentExecutions.value
    }

    return scheduledAgentExecutions.value.filter(({ task, execution }) => {
      return [
        task.name,
        task.goal,
        task.full_code_path,
        task.source_ref,
        execution.session_id,
        execution.status,
        execution.output_summary,
        execution.error_message,
        execution.trace_id
      ].some((field) => (field || '').toLowerCase().includes(keyword))
    })
  })

  function openScheduledAgentTask(task: ScheduledAgentTaskItem) {
    if (!task.last_session_id) {
      ElMessage.info('该定时会话还没有执行记录')
      return
    }
    onOpenSession({
      session_id: task.last_session_id,
      title: task.name,
      user: task.request_user || task.created_by,
      status: task.status === 'cancelled' ? 'cancelled' : 'done',
      full_code_path: task.full_code_path,
      created_at: task.created_at,
      updated_at: task.updated_at
    })
  }

  function openScheduledAgentExecution(record: ScheduledAgentExecutionRecord) {
    const sessionID = record.execution.session_id
    if (!sessionID) {
      ElMessage.info('该执行记录还没有会话 ID')
      return
    }
    onOpenSession({
      session_id: sessionID,
      title: record.task.name,
      user: record.task.request_user || record.task.created_by,
      status: record.execution.status === 'running' ? 'generating' : record.execution.status === 'cancelled' ? 'cancelled' : 'done',
      full_code_path: record.task.full_code_path,
      created_at: record.execution.created_at,
      updated_at: record.execution.updated_at
    })
  }

  async function handleCancelTask(task: WorkspaceSessionItem) {
    cancellingTaskId.value = task.session_id
    try {
      await cancelWorkspaceChat(task.session_id)
      ElMessage.success('已停止该任务')
      loadSessions()
    } catch (e: any) {
      ElMessage.error(e?.message || '停止失败')
    } finally {
      cancellingTaskId.value = null
    }
  }

  async function handleRunScheduledAgentTaskNow(task: ScheduledAgentTaskItem) {
    scheduledAgentTaskActionId.value = task.id
    try {
      await runScheduledAgentTaskNow(task.id)
      ElMessage.success('已触发执行')
      await loadScheduledAgentTasks()
    } catch (e: any) {
      ElMessage.error(e?.message || '触发失败')
    } finally {
      scheduledAgentTaskActionId.value = null
    }
  }

  watch(
    [() => workstationContext.value?.fullCodePath, sidebarVisible],
    ([path, visible]) => {
      stopPoll()
      if (path && visible) {
        loadSessions()
        if (featureFlags.scheduledTasks) {
          loadScheduledAgentTasks()
        }
        startPoll()
      }
    },
    { immediate: true }
  )

  watch(activeTab, (tab) => {
    if (tab === 'scheduled' && featureFlags.scheduledTasks) {
      loadScheduledAgentTasks()
    } else if (tab === 'scheduled' && !featureFlags.scheduledTasks) {
      activeTab.value = 'all'
    }
  })

  const unsubscribeScheduledTaskCreated = eventBus.on(WorkspaceEvent.scheduledAgentTaskCreated, () => {
    if (featureFlags.scheduledTasks && sidebarVisible.value) {
      loadScheduledAgentTasks()
    }
  })

  onUnmounted(() => {
    stopPoll()
    unsubscribeScheduledTaskCreated()
  })

  return {
    sessions,
    scheduledAgentTasks,
    scheduledAgentExecutions,
    sessionsLoading,
    scheduledAgentTasksLoading,
    activeTab,
    sessionSearchKeyword,
    cancellingTaskId,
    scheduledAgentTaskActionId,
    runningCount,
    scheduledAgentTaskCount,
    filteredSessions,
    filteredScheduledAgentTasks,
    filteredScheduledAgentExecutions,
    openSession,
    openScheduledAgentTask,
    openScheduledAgentExecution,
    formatRelativeTime,
    handleCancelTask,
    handleRunScheduledAgentTaskNow,
    loadSessions,
    loadScheduledAgentTasks
  }
}
