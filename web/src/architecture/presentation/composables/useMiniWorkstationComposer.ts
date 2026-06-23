import { ElMessage } from 'element-plus'
import { computed, ref, watch, type Ref } from 'vue'
import { getLLMList, type LLMInfo } from '@/architecture/presentation/context/api/agent'
import { workspaceChatStream, type WorkspaceChatMessageFile, type WorkspaceChatReq, type WorkspaceChatStreamOnEvent } from '@/architecture/presentation/context/api/workspace'
import type { ChatMessageFile } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { translate } from '@/architecture/shared/i18n'

export interface UseMiniWorkstationComposerOptions {
  fullCodePath: Ref<string>
  sessionId: Ref<string | undefined>
  maximized: Ref<boolean>
  inputText: Ref<string>
  inputRef: Ref<HTMLTextAreaElement | undefined>
  attachedFiles: Ref<WorkspaceChatMessageFile[]>
  sending: Ref<boolean>
  sendMessage: (content: string, streamFn: (onEvent: WorkspaceChatStreamOnEvent) => Promise<void>, files?: ChatMessageFile[]) => Promise<void>
  beforeSend?: (payload: { text: string; files: WorkspaceChatMessageFile[] | null }) => BeforeSendDecision | Promise<BeforeSendDecision>
  onTaskStarted?: (sessionId: string) => void
  onToolCallOk?: (payload: { name: string }) => void
  onMaximizedSessionStarted?: (sessionId: string) => void
}

type BeforeSendDecision = boolean | {
  cancel?: boolean
  preserveDraft?: boolean
  interactionAction?: string
}

interface SendWorkspaceMessageOptions {
  newSession?: boolean
  displayText?: string
  sessionIdOverride?: string
  contextUsage?: string
  artifactKind?: string
  interactionAction?: string
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
  let activeStreamAbortController: AbortController | null = null

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
        ...(options.interactionAction ? { interaction_action: options.interactionAction } : {}),
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

    const streamFn = async (onEvent: WorkspaceChatStreamOnEvent) => {
      const controller = new AbortController()
      activeStreamAbortController = controller
      try {
        await workspaceChatStream(payload, (event, data) => {
          void onEvent(event, data)
          if (event === 'session') {
            const sessionData = data as { session_id?: unknown }
            if (typeof sessionData.session_id === 'string') {
              onTaskStarted?.(sessionData.session_id)
              if (maximized.value) {
                onMaximizedSessionStarted?.(sessionData.session_id)
              }
            }
          }
          if (event === 'tool_call') {
            const toolCallData = data as { status?: unknown; name?: unknown }
            if (toolCallData.status === 'ok' && typeof toolCallData.name === 'string') {
              onToolCallOk?.({ name: toolCallData.name })
            }
          }
        }, { signal: controller.signal })
      } finally {
        if (activeStreamAbortController === controller) {
          activeStreamAbortController = null
        }
      }
    }

    try {
      const messageFiles: ChatMessageFile[] | undefined = files?.length
        ? files.map(file => ({ ...file }))
        : undefined
      await sendMessage(options.displayText || text || (files?.length ? translate('miniWorkstation.uploadedFileFallback') : ''), streamFn, messageFiles)
      return true
    } catch {
      ElMessage.error(translate('miniWorkstation.sendFailed'))
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
      ElMessage.success(translate('miniWorkstation.queuedSuccess'))
      return
    }
    const beforeSendDecision = beforeSend ? await beforeSend({ text, files }) : false
    if (shouldCancelSend(beforeSendDecision)) {
      if (!shouldPreserveDraft(beforeSendDecision)) {
        inputText.value = ''
        attachedFiles.value = []
      }
      return
    }
    const interactionAction = getBeforeSendInteractionAction(beforeSendDecision)

    inputText.value = ''
    attachedFiles.value = []
    await sendWorkspaceMessage(text, files, { interactionAction })
  }

  function shouldCancelSend(decision: BeforeSendDecision): boolean {
    if (decision === true) return true
    if (!decision || typeof decision !== 'object') return false
    return !!decision.cancel
  }

  function shouldPreserveDraft(decision: BeforeSendDecision): boolean {
    return !!decision && typeof decision === 'object' && !!decision.preserveDraft
  }

  function getBeforeSendInteractionAction(decision: BeforeSendDecision): string | undefined {
    if (!decision || typeof decision !== 'object') return undefined
    return decision.interactionAction
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

  async function sendTextToSession(targetSessionId: string, content: string, displayText?: string, meta?: { contextUsage?: string; artifactKind?: string; interactionAction?: string; resume?: boolean }): Promise<boolean> {
    const text = content.trim()
    if (!fullCodePath.value || !targetSessionId || !text || sending.value) {
      return false
    }
    return sendWorkspaceMessage(text, null, {
      sessionIdOverride: targetSessionId,
      displayText,
      contextUsage: meta?.contextUsage,
      artifactKind: meta?.artifactKind,
      interactionAction: meta?.interactionAction,
      resume: meta?.resume
    })
  }

  function abortActiveStream() {
    if (!activeStreamAbortController) {
      return
    }
    activeStreamAbortController.abort()
    activeStreamAbortController = null
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
    sendTextToSession,
    abortActiveStream
  }
}
