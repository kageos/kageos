import { ElMessage } from 'element-plus'
import { onUnmounted, ref, watch, type Ref } from 'vue'
import {
  cancelWorkspaceChat,
  getFinishedSessions,
  getRunningSessions,
  getWorkspaceMessages,
  getWorkspaceSessionSSEStatus,
  getWorkspaceSessions,
  type WorkspaceAutomationAgentItem,
  type WorkspaceMessageInfo,
  type WorkspaceSessionItem
} from '@/architecture/presentation/context/api/workspace'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import type { AssistantBlock, ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { Logger } from '@/architecture/shared/logger'
import { fileNameFromRef, parseFileRefs } from '@/architecture/presentation/widgets/filesWidgetTypes'
import { splitWorkspaceThinkBlocks, stripWorkspaceThinkBlocks } from './useWorkspaceThinkFilter'

export interface UseMiniWorkstationSessionsOptions {
  fullCodePath: Ref<string>
  initialSessionId: Ref<string | undefined>
  maximized: Ref<boolean>
  sending: Ref<boolean>
  sessionId: Ref<string | undefined>
  setMessages: (messages: ChatMessage[]) => void
  abortActiveStream?: () => void
  onSelectMaximizedSession?: (sessionId: string) => void
  sessionSourceFilter: Ref<string>
}

export function resolveWorkspaceSessionSourceFilter(value: string): {
  session_scope: 'human' | 'automation'
  automation_task_id?: number
} {
  const normalized = String(value || '').trim()
  if (!normalized.startsWith('agent:')) return { session_scope: 'human' }
  const taskID = Number(normalized.slice('agent:'.length))
  if (!Number.isFinite(taskID) || taskID <= 0) return { session_scope: 'human' }
  return { session_scope: 'automation', automation_task_id: taskID }
}

export function normalizeWorkspaceSessionMessages(rawMessages: WorkspaceMessageInfo[]): ChatMessage[] {
  return rawMessages
    .filter(message => message.role === 'user' || message.role === 'assistant')
    .map(message => {
      const rawDisplayContent = message.display_content || message.content || ''
      const thinkingContent = typeof message.thinking_content === 'string' ? message.thinking_content : ''
      const displaySegments = message.role === 'assistant'
        ? (thinkingContent
            ? [
                { type: 'thinking' as const, text: thinkingContent },
                ...splitWorkspaceThinkBlocks(rawDisplayContent).filter(segment => segment.type === 'content')
              ]
            : splitWorkspaceThinkBlocks(rawDisplayContent))
        : []
      const displayContent = message.role === 'assistant'
        ? (displaySegments.length ? displaySegments.filter(segment => segment.type === 'content').map(segment => segment.text).join('') : stripWorkspaceThinkBlocks(rawDisplayContent))
        : rawDisplayContent
      const toolCalls = message.tool_calls || []
      const assistantBlocks: AssistantBlock[] | undefined = (() => {
        if (message.role !== 'assistant') return undefined
        const blocks = displaySegments
          .filter(segment => segment.text)
          .map(segment => ({ type: segment.type, text: segment.text }) as AssistantBlock)
        if (!blocks.length && displayContent) {
          blocks.push({ type: 'content', text: displayContent })
        }
        if (toolCalls.length) {
          blocks.push({ type: 'tool_calls', calls: toolCalls })
        }
        return blocks.length ? blocks : undefined
      })()
      return {
        role: message.role as 'user' | 'assistant',
        user: message.user || '',
        content: displayContent,
        raw_content: message.content || '',
        files: message.files
          ? parseFileRefs(message.files).map(ref => ({ ref, name: fileNameFromRef(ref), source_name: fileNameFromRef(ref) }))
          : [],
        context_usage: message.context_usage || '',
        artifact_kind: message.artifact_kind || '',
        tool_calls: message.tool_calls || [],
        llm_config_id: message.llm_config_id,
        llm_config_name: message.llm_config_name || '',
        llm_provider: message.llm_provider || '',
        llm_model: message.llm_model || '',
        llm_usage: message.llm_usage,
        model_context_plan: message.model_context_plan,
        model_context_plans: message.model_context_plan ? [message.model_context_plan] : undefined,
        created_at: message.created_at,
        blocks: assistantBlocks
      }
    })
}

export function useMiniWorkstationSessions(options: UseMiniWorkstationSessionsOptions) {
  const { fullCodePath, initialSessionId, maximized, sending, sessionId, setMessages, abortActiveStream, onSelectMaximizedSession, sessionSourceFilter } = options

  const miniSessionList = ref<WorkspaceSessionItem[]>([])
  const globalSessionList = ref<WorkspaceSessionItem[]>([])
  const automationAgents = ref<WorkspaceAutomationAgentItem[]>([])
  const loadingSessions = ref(false)
  const loadingGlobalSessions = ref(false)
  const sessionLoadFailed = ref(false)
  const globalSessionLoadFailed = ref(false)
  const stopping = ref(false)

  let miniStreamCleanup: (() => void) | null = null
  let miniPollTimer: ReturnType<typeof setInterval> | null = null

  async function loadMiniSessions() {
    if (!fullCodePath.value) {
      miniSessionList.value = []
      return
    }

    loadingSessions.value = true
    sessionLoadFailed.value = false
    try {
      const source = resolveWorkspaceSessionSourceFilter(sessionSourceFilter.value)
      const response = await getWorkspaceSessions({ full_code_path: fullCodePath.value, page: 1, page_size: 100, ...source })
      miniSessionList.value = response.sessions || []
      automationAgents.value = response.automation_agents || []
    } catch {
      miniSessionList.value = []
      sessionLoadFailed.value = true
    } finally {
      loadingSessions.value = false
    }
  }

  async function loadGlobalSessions() {
    loadingGlobalSessions.value = true
    globalSessionLoadFailed.value = false
    try {
      const [running, finished] = await Promise.allSettled([
        getRunningSessions(),
        getFinishedSessions(60)
      ])
      const merged = [
        ...(running.status === 'fulfilled' ? running.value.sessions || [] : []),
        ...(finished.status === 'fulfilled' ? finished.value.sessions || [] : [])
      ]
      if (running.status === 'rejected' && finished.status === 'rejected') {
        globalSessionLoadFailed.value = true
      }
      const byId = new Map<string, WorkspaceSessionItem>()
      for (const session of merged) {
        if (!session.session_id) continue
        byId.set(session.session_id, session)
      }
      globalSessionList.value = Array.from(byId.values())
        .sort((left, right) => new Date(right.updated_at || right.created_at).getTime() - new Date(left.updated_at || left.created_at).getTime())
    } catch {
      globalSessionList.value = []
      globalSessionLoadFailed.value = true
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
    const targetSessionId = sessionId.value
    if (!targetSessionId || stopping.value) {
      return
    }

    stopping.value = true
    try {
      await cancelWorkspaceChat(targetSessionId)
      abortActiveStream?.()
      sending.value = false
      stopMiniPoll()
      stopMiniStreamListening()
      ElMessage.success('已停止')
      await Promise.allSettled([
        loadMiniSessions(),
        loadGlobalSessions(),
        loadMiniSessionMessages(targetSessionId)
      ])
    } catch (error: any) {
      ElMessage.error(error?.message || '停止失败')
    } finally {
      stopping.value = false
    }
  }

  async function loadMiniSessionMessages(targetSessionId: string) {
    try {
      const response = await getWorkspaceMessages({ session_id: targetSessionId })
      if (sessionId.value !== targetSessionId) {
        return
      }
      setMessages(normalizeWorkspaceSessionMessages(response?.messages || []))
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
    let pollInFlight = false
    miniPollTimer = setInterval(async () => {
      if (sessionId.value !== targetSessionId) {
        stopMiniPoll()
        return
      }
      if (sending.value || pollInFlight) {
        return
      }
      pollInFlight = true
      try {
        try {
          const { connected } = await getWorkspaceSessionSSEStatus(targetSessionId)
          if (connected) {
            return
          }
        } catch {
          // 存活检测失败时仍按原逻辑拉取，避免漏更新
        }
        await loadMiniSessionMessages(targetSessionId)
      } finally {
        pollInFlight = false
      }
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
    automationAgents,
    loadingSessions,
    loadingGlobalSessions,
    sessionLoadFailed,
    globalSessionLoadFailed,
    stopping,
    loadMiniSessions,
    loadGlobalSessions,
    handleNewSession,
    handleStopSession,
    handleSelectSession,
    loadMiniSessionMessages,
    formatRelativeTime,
    formatMessageTime,
    startMiniStreamListening,
    startMiniPoll,
    stopMiniStreamListening,
    stopMiniPoll
  }
}
