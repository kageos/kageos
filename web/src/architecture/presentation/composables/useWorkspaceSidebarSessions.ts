import { computed, onUnmounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { cancelWorkspaceChat, getWorkspaceSessions, type WorkspaceSessionItem } from '@/api/workspace'

type SidebarTab = 'all' | 'running' | 'finished'

export interface WorkspaceSidebarContext {
  fullCodePath: string
  dirName: string
}

export interface UseWorkspaceSidebarSessionsOptions {
  workstationContext: ComputedRef<WorkspaceSidebarContext | null>
  sidebarVisible: Ref<boolean>
  onOpenSession: (session: WorkspaceSessionItem) => void
}

export function useWorkspaceSidebarSessions(options: UseWorkspaceSidebarSessionsOptions) {
  const { workstationContext, sidebarVisible, onOpenSession } = options

  const sessions = ref<WorkspaceSessionItem[]>([])
  const sessionsLoading = ref(false)
  const activeTab = ref<SidebarTab>('all')
  const sessionSearchKeyword = ref('')
  const cancellingTaskId = ref<string | null>(null)

  let pollTimer: ReturnType<typeof setInterval> | null = null

  const runningCount = computed(() =>
    sessions.value.filter((s: WorkspaceSessionItem) => s.status === 'generating').length
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

  function startPoll() {
    stopPoll()
    pollTimer = setInterval(() => {
      if (sessions.value.some((s: WorkspaceSessionItem) => s.status === 'generating')) {
        loadSessions()
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

  watch(
    [() => workstationContext.value?.fullCodePath, sidebarVisible],
    ([path, visible]) => {
      stopPoll()
      if (path && visible) {
        loadSessions()
        startPoll()
      }
    },
    { immediate: true }
  )

  onUnmounted(() => {
    stopPoll()
  })

  return {
    sessions,
    sessionsLoading,
    activeTab,
    sessionSearchKeyword,
    cancellingTaskId,
    runningCount,
    filteredSessions,
    openSession,
    formatRelativeTime,
    handleCancelTask,
    loadSessions
  }
}
