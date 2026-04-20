import { ElMessage } from 'element-plus'
import { ref, type Ref } from 'vue'
import { getLLMList, type LLMInfo } from '@/api/agent'
import { workspaceChatStream, type WorkspaceChatMessageFile, type WorkspaceChatReq } from '@/api/workspace'

export interface UseMiniWorkstationComposerOptions {
  fullCodePath: Ref<string>
  sessionId: Ref<string | undefined>
  maximized: Ref<boolean>
  inputText: Ref<string>
  inputRef: Ref<HTMLTextAreaElement | undefined>
  attachedFiles: Ref<WorkspaceChatMessageFile[]>
  sending: Ref<boolean>
  sendMessage: (content: string, streamFn: (onEvent: (event: string, data: Record<string, unknown>) => Promise<void> | void) => Promise<void>, files?: any[]) => Promise<void>
  onTaskStarted?: (sessionId: string) => void
  onToolCallOk?: (payload: { name: string }) => void
  onMaximizedSessionStarted?: (sessionId: string) => void
}

export function useMiniWorkstationComposer(options: UseMiniWorkstationComposerOptions) {
  const {
    fullCodePath,
    sessionId,
    maximized,
    inputText,
    inputRef,
    attachedFiles,
    sendMessage,
    onTaskStarted,
    onToolCallOk,
    onMaximizedSessionStarted
  } = options

  const llmList = ref<LLMInfo[]>([])
  const llmLoading = ref(false)
  const selectedLLMConfigId = ref<number>(0)

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

  async function handleSend() {
    const text = inputText.value.trim()
    const files = attachedFiles.value.length > 0 ? [...attachedFiles.value] : null
    if (!fullCodePath.value || (!text && !files?.length)) {
      return
    }

    inputText.value = ''
    attachedFiles.value = []

    const payload: WorkspaceChatReq = {
      full_code_path: fullCodePath.value,
      message: {
        content: text || '',
        ...(files?.length ? { files: files.map(file => file.ref).filter(Boolean).join(',') } : {})
      },
      session_id: sessionId.value
    }

    if (selectedLLMConfigId.value > 0) {
      payload.llm_config_id = selectedLLMConfigId.value
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
      await sendMessage(text || (files?.length ? '已上传文件' : ''), streamFn, files?.length ? (files as any) : undefined)
    } catch {
      ElMessage.error('发送失败')
    }
  }

  return {
    inputText,
    inputRef,
    llmList,
    llmLoading,
    selectedLLMConfigId,
    onLLMSelectVisibleChange,
    onInputEnter,
    handleSend
  }
}
