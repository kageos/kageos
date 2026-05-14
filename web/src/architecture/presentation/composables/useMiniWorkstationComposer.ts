import { ElMessage } from 'element-plus'
import { computed, ref, watch, type Ref } from 'vue'
import { getLLMList, type LLMInfo } from '@/architecture/infrastructure/api/agent'
import { workspaceChatStream, type WorkspaceChatMessageFile, type WorkspaceChatReq } from '@/architecture/infrastructure/api/workspace'

export interface UseMiniWorkstationComposerOptions {
  fullCodePath: Ref<string>
  sessionId: Ref<string | undefined>
  maximized: Ref<boolean>
  inputText: Ref<string>
  inputRef: Ref<HTMLTextAreaElement | undefined>
  attachedFiles: Ref<WorkspaceChatMessageFile[]>
  sending: Ref<boolean>
  sendMessage: (content: string, streamFn: (onEvent: (event: string, data: Record<string, unknown>) => Promise<void> | void) => Promise<void>, files?: any[]) => Promise<void>
  beforeSend?: (payload: { text: string; files: WorkspaceChatMessageFile[] | null }) => boolean | Promise<boolean>
  onTaskStarted?: (sessionId: string) => void
  onToolCallOk?: (payload: { name: string }) => void
  onMaximizedSessionStarted?: (sessionId: string) => void
}

interface SendWorkspaceMessageOptions {
  newSession?: boolean
  displayText?: string
  sessionIdOverride?: string
  contextUsage?: string
  artifactKind?: string
  resume?: boolean
}

interface QueuedWorkspaceMessage {
  text: string
  files: WorkspaceChatMessageFile[] | null
}

export function useMiniWorkstationComposer(options: UseMiniWorkstationComposerOptions) {
  const {
    fullCodePath,
    sessionId,
    maximized,
    inputText,
    inputRef,
    attachedFiles,
    sending,
    sendMessage,
    beforeSend,
    onTaskStarted,
    onToolCallOk,
    onMaximizedSessionStarted
  } = options

  const llmList = ref<LLMInfo[]>([])
  const llmLoading = ref(false)
  const selectedLLMConfigId = ref<number>(0)
  const queuedMessages = ref<QueuedWorkspaceMessage[]>([])
  const queuedCount = computed(() => queuedMessages.value.length)

  async function loadLLMs() {
    llmLoading.value = true
    try {
      const response = await getLLMList({ scope: 'market', page: 1, page_size: 200 }) as { configs?: LLMInfo[] }
      llmList.value = response?.configs ?? []
    } catch {
      llmList.value = []
    } finally {
      llmLoading.value = false
    }
  }

  function onLLMSelectVisibleChange(visible: boolean) {
    if (visible && llmList.value.length === 0) {
      void loadLLMs()
    }
  }

  function onInputEnter(event: KeyboardEvent) {
    if (event.shiftKey) {
      return
    }
    event.preventDefault()
    void handleSend()
  }

  async function sendWorkspaceMessage(text: string, files: WorkspaceChatMessageFile[] | null, options: SendWorkspaceMessageOptions = {}): Promise<boolean> {
    const payload: WorkspaceChatReq = {
      full_code_path: fullCodePath.value,
      message: {
        content: text || '',
        ...(options.displayText ? { display_content: options.displayText } : {}),
        ...(options.contextUsage ? { context_usage: options.contextUsage } : {}),
        ...(options.artifactKind ? { artifact_kind: options.artifactKind } : {}),
        ...(files?.length ? { files: files.map(file => file.ref).filter(Boolean).join(',') } : {})
      }
    }
    if (options.sessionIdOverride) {
      payload.session_id = options.sessionIdOverride
    } else if (!options.newSession && sessionId.value) {
      payload.session_id = sessionId.value
    }

    if (selectedLLMConfigId.value > 0) {
      payload.llm_config_id = selectedLLMConfigId.value
    }
    if (options.resume) {
      payload.resume = true
    }

    const streamFn = async (onEvent: (event: string, data: Record<string, unknown>) => Promise<void> | void) => {
      await workspaceChatStream(payload, (event, data) => {
        void onEvent(event, data as Record<string, unknown>)
        if (event === 'session' && typeof data.session_id === 'string') {
          onTaskStarted?.(data.session_id as string)
          if (maximized.value) {
            onMaximizedSessionStarted?.(data.session_id as string)
          }
        }
        if (
          event === 'tool_call'
          && (data as { status?: string })?.status === 'ok'
          && typeof (data as { name?: string })?.name === 'string'
        ) {
          onToolCallOk?.({ name: (data as { name: string }).name })
        }
      })
    }

    try {
      await sendMessage(options.displayText || text || (files?.length ? '已上传文件' : ''), streamFn, files?.length ? (files as any) : undefined)
      return true
    } catch {
      ElMessage.error('发送失败')
      return false
    }
  }

  async function handleSend() {
    const text = inputText.value.trim()
    const files = attachedFiles.value.length > 0 ? [...attachedFiles.value] : null
    if (!fullCodePath.value || (!text && !files?.length)) {
      return
    }
    if (sending.value) {
      queuedMessages.value.push({ text, files })
      inputText.value = ''
      attachedFiles.value = []
      ElMessage.success('已加入发送队列')
      return
    }
    if (beforeSend && await beforeSend({ text, files })) {
      inputText.value = ''
      attachedFiles.value = []
      return
    }

    inputText.value = ''
    attachedFiles.value = []
    await sendWorkspaceMessage(text, files)
  }

  watch(sending, (isSending) => {
    if (isSending || queuedMessages.value.length === 0) {
      return
    }
    const next = queuedMessages.value.shift()
    if (!next) {
      return
    }
    void sendWorkspaceMessage(next.text, next.files)
  })

  async function sendText(content: string): Promise<boolean> {
    const text = content.trim()
    if (!fullCodePath.value || !text || sending.value) {
      return false
    }
    return sendWorkspaceMessage(text, null)
  }

  async function sendTextInNewSession(content: string, displayText?: string): Promise<boolean> {
    const text = content.trim()
    if (!fullCodePath.value || !text || sending.value) {
      return false
    }
    return sendWorkspaceMessage(text, null, { newSession: true, displayText })
  }

  async function sendTextToSession(targetSessionId: string, content: string, displayText?: string, meta?: { contextUsage?: string; artifactKind?: string; resume?: boolean }): Promise<boolean> {
    const text = content.trim()
    if (!fullCodePath.value || !targetSessionId || !text || sending.value) {
      return false
    }
    return sendWorkspaceMessage(text, null, {
      sessionIdOverride: targetSessionId,
      displayText,
      contextUsage: meta?.contextUsage,
      artifactKind: meta?.artifactKind,
      resume: meta?.resume
    })
  }

  return {
    inputText,
    inputRef,
    llmList,
    llmLoading,
    selectedLLMConfigId,
    queuedCount,
    onLLMSelectVisibleChange,
    onInputEnter,
    handleSend,
    sendText,
    sendTextInNewSession,
    sendTextToSession
  }
}
