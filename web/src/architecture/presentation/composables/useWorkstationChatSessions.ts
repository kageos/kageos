import { computed, nextTick, onMounted, onUnmounted, ref, toRaw, watch, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getWorkspaceMessages,
  getWorkspaceSessionSSEStatus,
  getWorkspaceSessions,
  type WorkspaceChatMessageFile,
  type WorkspaceSessionItem
} from '@/api/workspace'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { eventBus } from '@/architecture/infrastructure/eventBus'
import { extractFileGroupsFromResult, type OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import { Logger } from '@/core/utils/logger'

interface UseWorkstationChatSessionsOptions {
  fullCodePath: Ref<string>
  initialSessionId: Ref<string>
  initialInputText: Ref<string>
  initialAttachedFiles: Ref<WorkspaceChatMessageFile[]>
  visible: Ref<boolean>
  messages: Ref<ChatMessage[]>
  sending: Ref<boolean>
  sessionId: Ref<string | undefined>
  messagesRef: Ref<HTMLElement | null>
  setMessages: (messages: ChatMessage[]) => void
  setInputText: (value: string) => void
  setAttachedFiles: (files: WorkspaceChatMessageFile[]) => void
  clearInitialInput: () => void
  updateSessionId: (value: string | undefined) => void
}

function parseMessageFiles(filesStr: string | null | undefined): WorkspaceChatMessageFile[] {
  if (!filesStr) return []
  try {
    const parsed = JSON.parse(filesStr) as { files?: WorkspaceChatMessageFile[] }
    return Array.isArray(parsed?.files) ? parsed.files : []
  } catch {
    return []
  }
}

function stripFilesBlockForDisplay(content: string): string {
  if (!content) return ''
  const stripped = content
    .replace(/<files>[\s\S]*?<\/files>/i, '')
    .replace(/\s*以上\s*<files>\s*标签中的 JSON[^。]*。\s*/g, '')
    .trim()
  return stripped || content
}

export function useWorkstationChatSessions({
  fullCodePath,
  initialSessionId,
  initialInputText,
  initialAttachedFiles,
  visible,
  messages,
  sending,
  sessionId,
  messagesRef,
  setMessages,
  setInputText,
  setAttachedFiles,
  clearInitialInput,
  updateSessionId
}: UseWorkstationChatSessionsOptions) {
  const sessionList = ref<WorkspaceSessionItem[]>([])
  const loadingSessions = ref(false)
  const sessionSidebarExpanded = ref(true)
  const sessionSearchKeyword = ref('')

  let generatingPollTimer: ReturnType<typeof setInterval> | null = null
  let streamCleanup: (() => void) | null = null

  const filteredSessionList = computed(() => {
    const keyword = sessionSearchKeyword.value.trim().toLowerCase()
    if (!keyword) return sessionList.value
    return sessionList.value.filter((session) => {
      const title = (session.title || '').toLowerCase()
      const user = (session.user || '').toLowerCase()
      return title.includes(keyword) || user.includes(keyword)
    })
  })

  async function loadSessions() {
    if (!fullCodePath.value) {
      sessionList.value = []
      return
    }
    loadingSessions.value = true
    try {
      const result = await getWorkspaceSessions({ full_code_path: fullCodePath.value })
      sessionList.value = result.sessions || []
    } catch (error) {
      Logger.error('[WorkstationChatSessions]', '加载会话列表失败', { error })
      ElMessage.error('加载会话列表失败')
      sessionList.value = []
    } finally {
      loadingSessions.value = false
    }
  }

  async function loadSessionMessages(targetSessionId: string) {
    try {
      const result = await getWorkspaceMessages({ session_id: targetSessionId })
      const nextMessages = result.messages
        .filter((message) => message.role === 'user' || message.role === 'assistant')
        .map((message) => {
          const role = message.role as 'user' | 'assistant'
          const content = message.content || ''
          const tool_calls = message.tool_calls || []
          const created_at = message.created_at ?? (message as { createdAt?: string }).createdAt ?? ''
          let blocks: ChatMessage['blocks'] | undefined
          if (role === 'assistant' && (content || tool_calls.length)) {
            if (content && tool_calls.length) {
              blocks = [{ type: 'content', text: content }, { type: 'tool_calls', calls: tool_calls }]
            } else if (content) {
              blocks = [{ type: 'content', text: content }]
            } else {
              blocks = [{ type: 'tool_calls', calls: tool_calls }]
            }
          }
          return {
            role,
            content,
            files: parseMessageFiles(message.files),
            tool_calls,
            blocks,
            created_at
          }
        })
      setMessages(nextMessages as ChatMessage[])
      setTimeout(() => {
        if (messagesRef.value) {
          messagesRef.value.scrollTop = messagesRef.value.scrollHeight
        }
      }, 100)
    } catch (error) {
      Logger.error('[WorkstationChatSessions]', '加载会话消息失败', { error })
      ElMessage.error('加载会话消息失败')
      setMessages([])
    }
  }

  function startGeneratingPoll(sid: string) {
    if (sending.value) return
    stopGeneratingPoll()
    generatingPollTimer = setInterval(async () => {
      if (sessionId.value !== sid) {
        stopGeneratingPoll()
        return
      }
      if (sending.value) return
      try {
        const { connected } = await getWorkspaceSessionSSEStatus(sid)
        if (connected) return
      } catch {
        // ignore and fallback to pulling
      }
      await loadSessionMessages(sid)
      await loadSessions()
      const session = sessionList.value.find((item) => item.session_id === sid)
      if (!session || session.status !== 'generating') {
        stopGeneratingPoll()
      }
    }, 3000)
  }

  function stopGeneratingPoll() {
    if (generatingPollTimer) {
      clearInterval(generatingPollTimer)
      generatingPollTimer = null
    }
  }

  function startStreamListening(sid: string) {
    stopStreamListening()

    const handleUpdate = (payload: { session_id: string; messages: ChatMessage[] }) => {
      if (payload.session_id === sid && sessionId.value === sid) {
        stopGeneratingPoll()
        setMessages(payload.messages)
        nextTick(() => {
          if (messagesRef.value) {
            messagesRef.value.scrollTop = messagesRef.value.scrollHeight
          }
        })
      }
    }

    const handleDone = (payload: { session_id: string }) => {
      if (payload.session_id === sid) {
        stopStreamListening()
        void loadSessionMessages(sid)
        void loadSessions()
      }
    }

    const offUpdate = eventBus.on('workspace:stream-update', handleUpdate)
    const offDone = eventBus.on('workspace:stream-done', handleDone)
    streamCleanup = () => {
      offUpdate()
      offDone()
    }
  }

  function stopStreamListening() {
    if (streamCleanup) {
      streamCleanup()
      streamCleanup = null
    }
  }

  function handleNewSession() {
    stopGeneratingPoll()
    stopStreamListening()
    sessionId.value = undefined
    setMessages([])
    updateSessionId(undefined)
    ElMessage.success('已创建新会话，发送第一条消息后将自动保存')
  }

  async function handleSelectSession(targetSessionId: string) {
    if (targetSessionId === sessionId.value) return
    stopGeneratingPoll()
    stopStreamListening()
    sessionId.value = targetSessionId
    updateSessionId(targetSessionId)
    await loadSessionMessages(targetSessionId)
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
    if (!isoString) return ''
    const date = new Date(isoString)
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hour = String(date.getHours()).padStart(2, '0')
    const minute = String(date.getMinutes()).padStart(2, '0')
    const second = String(date.getSeconds()).padStart(2, '0')
    return `${year}-${month}-${day} ${hour}:${minute}:${second}`
  }

  function getMessageDisplayContent(message: { role: string; content: string }): string {
    return message.role === 'user' ? stripFilesBlockForDisplay(message.content) : message.content
  }

  function getFileGroupsFromCalls(calls: Array<{ result?: string }>): OutputFileGroup[] {
    const groups: OutputFileGroup[] = []
    for (const call of calls) {
      groups.push(...extractFileGroupsFromResult(call.result))
    }
    return groups
  }

  function getMessageFileGroups(message: ChatMessage): OutputFileGroup[] {
    return getFileGroupsFromCalls(message.tool_calls ?? [])
  }

  watch(sending, (value) => {
    if (value) {
      stopGeneratingPoll()
    }
  })

  watch(messages, (newMessages) => {
    if (!visible.value && sending.value && sessionId.value) {
      eventBus.emit('workspace:stream-update', {
        session_id: sessionId.value,
        messages: JSON.parse(JSON.stringify(toRaw(newMessages)))
      })
    }
  })

  watch(sending, (value, oldValue) => {
    if (!visible.value && oldValue === true && value === false && sessionId.value) {
      eventBus.emit('workspace:stream-done', { session_id: sessionId.value })
    }
  })

  watch(fullCodePath, (newPath) => {
    if (newPath) {
      void loadSessions()
      if (!initialSessionId.value) {
        sessionId.value = undefined
        setMessages([])
      }
    } else {
      sessionList.value = []
      sessionId.value = undefined
      setMessages([])
    }
  })

  watch(initialSessionId, async (newSid) => {
    if (!newSid || !fullCodePath.value) return
    stopGeneratingPoll()
    stopStreamListening()
    await loadSessions()
    sessionId.value = newSid
    await loadSessionMessages(newSid)
    updateSessionId(newSid)
    const found = sessionList.value.find((item) => item.session_id === newSid)
    if (found?.status === 'generating') {
      startStreamListening(newSid)
      startGeneratingPoll(newSid)
    }
    if (initialInputText.value || initialAttachedFiles.value.length > 0) {
      if (initialInputText.value) {
        setInputText(initialInputText.value)
      }
      if (initialAttachedFiles.value.length) {
        setAttachedFiles(initialAttachedFiles.value)
      }
      nextTick(() => clearInitialInput())
    }
  })

  watch(
    () => [visible.value, initialInputText.value] as const,
    ([nextVisible, text]) => {
      if (!nextVisible || !text || !fullCodePath.value) return
      if (sessionId.value) return
      setInputText(text)
      if (initialAttachedFiles.value.length) {
        setAttachedFiles(initialAttachedFiles.value)
      }
      nextTick(() => clearInitialInput())
    }
  )

  onMounted(async () => {
    if (fullCodePath.value) {
      await loadSessions()
      if (initialSessionId.value) {
        const found = sessionList.value.find((session) => session.session_id === initialSessionId.value)
        if (found) {
          sessionId.value = initialSessionId.value
          await loadSessionMessages(initialSessionId.value)
          if (found.status === 'generating') {
            startStreamListening(initialSessionId.value)
            startGeneratingPoll(initialSessionId.value)
          }
        }
      }
    }
  })

  onUnmounted(() => {
    stopGeneratingPoll()
    stopStreamListening()
  })

  return {
    sessionList,
    loadingSessions,
    sessionSidebarExpanded,
    sessionSearchKeyword,
    filteredSessionList,
    loadSessions,
    handleNewSession,
    handleSelectSession,
    formatRelativeTime,
    formatMessageTime,
    getMessageDisplayContent,
    getFileGroupsFromCalls,
    getMessageFileGroups
  }
}
