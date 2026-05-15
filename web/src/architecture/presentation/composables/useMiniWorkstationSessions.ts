import { ElMessage } from 'element-plus'
import { onUnmounted, ref, watch, type Ref } from 'vue'
import {
  cancelWorkspaceChat,
  getFinishedSessions,
  getRunningSessions,
  getWorkspaceMessages,
  getWorkspaceSessionSSEStatus,
  getWorkspaceSessions,
  type WorkspaceSessionItem
} from '@/architecture/presentation/context/api/workspace'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { Logger } from '@/architecture/shared/logger'
import { fileNameFromRef, parseFileRefs } from '@/architecture/presentation/widgets/filesWidgetTypes'

export interface UseMiniWorkstationSessionsOptions {
  fullCodePath: Ref<string>
  initialSessionId: Ref<string | undefined>
  maximized: Ref<boolean>
  sending: Ref<boolean>
  sessionId: Ref<string | undefined>
  setMessages: (messages: ChatMessage[]) => void
  onSelectMaximizedSession?: (sessionId: string) => void
}

function normalizeSessionMessages(rawMessages: any[]): ChatMessage[] {
  return rawMessages
    .filter(message => message.role === 'user' || message.role === 'assistant')
    .map((message: any) => {
      const displayContent = message.display_content || message.content || ''
      return {
        role: message.role as 'user' | 'assistant',
        user: message.user || message.created_by || '',
        content: displayContent,
        files: message.files
          ? parseFileRefs(message.files).map(ref => ({ ref, name: fileNameFromRef(ref), source_name: fileNameFromRef(ref) }))
          : [],
        tool_calls: message.tool_calls || [],
        llm_config_id: message.llm_config_id,
        llm_config_name: message.llm_config_name || '',
        llm_provider: message.llm_provider || '',
        llm_model: message.llm_model || '',
        created_at: message.created_at,
        blocks: (() => {
          const content = displayContent
          const toolCalls = message.tool_calls || []
          if (message.role !== 'assistant') {
            return undefined
          }
          if (content && toolCalls.length) {
            return [{ type: 'content' as const, text: content }, { type: 'tool_calls' as const, calls: toolCalls }]
          }
          if (content) {
            return [{ type: 'content' as const, text: content }]
          }
          if (toolCalls.length) {
            return [{ type: 'tool_calls' as const, calls: toolCalls }]
          }
          return undefined
        })()
      }
    })
}

export function useMiniWorkstationSessions(options: UseMiniWorkstationSessionsOptions) {
  const { fullCodePath, initialSessionId, maximized, sending, sessionId, setMessages, onSelectMaximizedSession } = options

  const miniSessionList = ref<WorkspaceSessionItem[]>([])
  const globalSessionList = ref<WorkspaceSessionItem[]>([])
  const loadingSessions = ref(false)
  const loadingGlobalSessions = ref(false)
  const stopping = ref(false)

  let miniStreamCleanup: (() => void) | null = null
  let miniPollTimer: ReturnType<typeof setInterval> | null = null

  async function loadMiniSessions() {
    if (!fullCodePath.value) {
      miniSessionList.value = []
      return
    }

    loadingSessions.value = true
    try {
      const response = await getWorkspaceSessions({ full_code_path: fullCodePath.value })
      miniSessionList.value = response.sessions || []
    } catch {
      miniSessionList.value = []
    } finally {
      loadingSessions.value = false
    }
  }

  async function loadGlobalSessions() {
    loadingGlobalSessions.value = true
    try {
      const [running, finished] = await Promise.allSettled([
        getRunningSessions(),
        getFinishedSessions(60)
      ])
      const merged = [
        ...(running.status === 'fulfilled' ? running.value.sessions || [] : []),
        ...(finished.status === 'fulfilled' ? finished.value.sessions || [] : [])
      ]
      const byId = new Map<string, WorkspaceSessionItem>()
      for (const session of merged) {
        if (!session.session_id) continue
        byId.set(session.session_id, session)
      }
      globalSessionList.value = Array.from(byId.values())
        .sort((left, right) => new Date(right.updated_at || right.created_at).getTime() - new Date(left.updated_at || left.created_at).getTime())
    } catch {
      globalSessionList.value = []
    } finally {
      loadingGlobalSessions.value = false
    }
  }

  function handleNewSession() {
    stopMiniPoll()
    stopMiniStreamListening()
    sending.value = false
    sessionId.value = undefined
    setMessages([])
  }

  async function handleStopSession() {
    if (!sessionId.value || stopping.value) {
      return
    }

    stopping.value = true
    try {
      await cancelWorkspaceChat(sessionId.value)
      sending.value = false
      stopMiniPoll()
      stopMiniStreamListening()
      ElMessage.success('已停止')
      if (maximized.value) {
        void loadMiniSessions()
      }
    } catch (error: any) {
      ElMessage.error(error?.message || '停止失败')
    } finally {
      stopping.value = false
    }
  }

  async function loadMiniSessionMessages(targetSessionId: string) {
    try {
      const response = await getWorkspaceMessages({ session_id: targetSessionId })
      setMessages(normalizeSessionMessages(response?.messages || []))
    } catch (error) {
      Logger.error('[MiniWorkstationSessions]', '加载会话消息失败', { error })
    }
  }

  function stopMiniStreamListening() {
    if (miniStreamCleanup) {
      miniStreamCleanup()
      miniStreamCleanup = null
    }
  }

  function stopMiniPoll() {
    if (miniPollTimer) {
      clearInterval(miniPollTimer)
      miniPollTimer = null
    }
  }

  function startMiniStreamListening(targetSessionId: string) {
    stopMiniStreamListening()
    const handleUpdate = (payload: { session_id: string; messages: ChatMessage[] }) => {
      if (payload.session_id === targetSessionId && sessionId.value === targetSessionId) {
        stopMiniPoll()
        setMessages(payload.messages)
      }
    }
    const handleDone = (payload: { session_id: string }) => {
      if (payload.session_id === targetSessionId) {
        stopMiniStreamListening()
        void loadMiniSessionMessages(targetSessionId)
      }
    }
    const offUpdate = eventBus.on('workspace:stream-update', handleUpdate)
    const offDone = eventBus.on('workspace:stream-done', handleDone)
    miniStreamCleanup = () => {
      offUpdate()
      offDone()
    }
  }

  function startMiniPoll(targetSessionId: string) {
    if (sending.value) {
      return
    }

    stopMiniPoll()
    miniPollTimer = setInterval(async () => {
      if (sessionId.value !== targetSessionId) {
        stopMiniPoll()
        return
      }
      if (sending.value) {
        return
      }
      try {
        const { connected } = await getWorkspaceSessionSSEStatus(targetSessionId)
        if (connected) {
          return
        }
      } catch {
        // 存活检测失败时仍按原逻辑拉取，避免漏更新
      }
      await loadMiniSessionMessages(targetSessionId)
    }, 3000)
  }

  async function handleSelectSession(targetSessionId: string) {
    if (targetSessionId === sessionId.value) {
      return
    }

    stopMiniPoll()
    stopMiniStreamListening()
    sending.value = false
    sessionId.value = targetSessionId
    setMessages([])

    if (maximized.value) {
      onSelectMaximizedSession?.(targetSessionId)
    }

    await loadMiniSessionMessages(targetSessionId)
    if (sessionId.value !== targetSessionId) {
      return
    }

    const found = miniSessionList.value.find(session => session.session_id === targetSessionId)
      || globalSessionList.value.find(session => session.session_id === targetSessionId)
    if (found?.status === 'generating') {
      startMiniStreamListening(targetSessionId)
      if (!maximized.value) {
        startMiniPoll(targetSessionId)
      }
    }
  }

  function formatRelativeTime(timeStr: string): string {
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

  function formatMessageTime(isoString: string): string {
    if (!isoString) return '—'
    const date = new Date(isoString)
    if (Number.isNaN(date.getTime())) return '—'
    const y = date.getFullYear()
    const M = String(date.getMonth() + 1).padStart(2, '0')
    const d = String(date.getDate()).padStart(2, '0')
    const h = String(date.getHours()).padStart(2, '0')
    const m = String(date.getMinutes()).padStart(2, '0')
    const s = String(date.getSeconds()).padStart(2, '0')
    return `${y}-${M}-${d} ${h}:${m}:${s}`
  }

  watch(maximized, (value) => {
    if (value && fullCodePath.value) {
      void loadMiniSessions()
    }
  })

  watch(initialSessionId, async (newSessionId) => {
    if (!newSessionId || !fullCodePath.value) {
      return
    }
    if (newSessionId === sessionId.value) {
      return
    }
    stopMiniPoll()
    stopMiniStreamListening()
    sending.value = false
    sessionId.value = newSessionId
    setMessages([])
    await loadMiniSessionMessages(newSessionId)
    if (sessionId.value !== newSessionId) {
      return
    }
    startMiniStreamListening(newSessionId)
    if (!maximized.value) {
      startMiniPoll(newSessionId)
    }
  }, { immediate: true })

  onUnmounted(() => {
    stopMiniPoll()
    stopMiniStreamListening()
  })

  return {
    miniSessionList,
    globalSessionList,
    loadingSessions,
    loadingGlobalSessions,
    stopping,
    loadMiniSessions,
    loadGlobalSessions,
    handleNewSession,
    handleStopSession,
    handleSelectSession,
    formatRelativeTime,
    formatMessageTime,
    startMiniStreamListening,
    startMiniPoll,
    stopMiniStreamListening,
    stopMiniPoll
  }
}
