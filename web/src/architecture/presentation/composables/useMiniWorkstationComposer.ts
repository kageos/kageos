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
  inputRef: Ref<{ focus: () => void } | undefined>
  attachedFiles: Ref<WorkspaceChatMessageFile[]>
  uploading: Ref<boolean>
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
  sessionId: string
  fullCodePath: string
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
    uploading,
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
  const belongsToCurrentSession = (message: QueuedWorkspaceMessage) => (
    message.sessionId === sessionId.value && message.fullCodePath === fullCodePath.value
  )
  const queuedCount = computed(() => queuedMessages.value.filter(belongsToCurrentSession).length)
  let checkingSend = false
  let drainingQueue = false
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
          const accepted = onEvent(event, data)
          if (accepted === false) return
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

  async function checkBeforeSend(text: string, files: WorkspaceChatMessageFile[] | null): Promise<BeforeSendDecision> {
    try {
      return beforeSend ? await beforeSend({ text, files }) : false
    } catch {
      ElMessage.error(translate('miniWorkstation.sendFailed'))
      return { cancel: true, preserveDraft: true }
    }
  }

  async function handleSend() {
    if (uploading.value) {
      ElMessage.warning(translate('miniWorkstation.waitForUpload'))
      return
    }
    const draft = inputText.value
    const text = draft.trim()
    const files = attachedFiles.value.length > 0 ? [...attachedFiles.value] : null
    if (!fullCodePath.value || (!text && !files?.length) || checkingSend || (drainingQueue && !sending.value)) return
    const targetSessionId = sessionId.value
    const targetPath = fullCodePath.value
    const clearSubmittedDraft = () => {
      if (inputText.value === draft) inputText.value = ''
      attachedFiles.value = attachedFiles.value.filter(file => !files?.includes(file))
    }
    if (sending.value) {
      // A first response may not have assigned a session yet. Keep the draft
      // instead of guessing which future session owns it.
      if (!targetSessionId) {
        ElMessage.warning(translate('miniWorkstation.waitForSession'))
        return
      }
      queuedMessages.value.push({ text, files, sessionId: targetSessionId, fullCodePath: targetPath })
      clearSubmittedDraft()
      ElMessage.success(translate('miniWorkstation.queuedSuccess'))
      return
    }
    checkingSend = true
    try {
      const decision = await checkBeforeSend(text, files)
      if (sessionId.value !== targetSessionId || fullCodePath.value !== targetPath || uploading.value || sending.value) return
      if (shouldCancelSend(decision)) {
        if (!shouldPreserveDraft(decision)) clearSubmittedDraft()
        return
      }
      clearSubmittedDraft()
      // sendWorkspaceMessage captures the current context synchronously.
      checkingSend = false
      await sendWorkspaceMessage(text, files, { interactionAction: getBeforeSendInteractionAction(decision) })
    } finally {
      checkingSend = false
    }
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

  async function retryQueuedMessages() {
    if (drainingQueue || checkingSend || sending.value || uploading.value) return
    drainingQueue = true
    try {
      while (!sending.value && !uploading.value) {
        const next = queuedMessages.value.find(belongsToCurrentSession)
        if (!next) break
        const decision = await checkBeforeSend(next.text, next.files)
        // Switching sessions or starting an upload during an async check must
        // not redirect or accidentally release a queued message.
        if (!belongsToCurrentSession(next) || sending.value || uploading.value || checkingSend) break
        if (shouldCancelSend(decision)) break
        const index = queuedMessages.value.indexOf(next)
        queuedMessages.value.splice(index, 1)
        const sent = await sendWorkspaceMessage(next.text, next.files, {
          sessionIdOverride: next.sessionId,
          interactionAction: getBeforeSendInteractionAction(decision)
        })
        if (!sent) {
          queuedMessages.value.splice(index, 0, next)
          break
        }
      }
    } finally {
      drainingQueue = false
    }
  }

  watch([sending, uploading, sessionId, fullCodePath], () => {
    void retryQueuedMessages()
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
    retryQueuedMessages,
    onLLMSelectVisibleChange,
    onInputEnter,
    handleSend,
    sendText,
    sendTextInNewSession,
    sendTextToSession,
    abortActiveStream
  }
}
