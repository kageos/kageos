/**
 * useWorkspaceChatStream - 流式工作台对话核心
 *
 * 维护 messages、sending、sessionId、agentId，暴露 handleEvent 与 send。
 * 调用方负责构造 payload 并调用 API（如 workspaceChatStream），在 SSE 回调里调用 handleEvent 即可。
 * 便于 WorkstationChat 与后续其他流式工具对话复用同一套消息状态与事件处理。
 */

import { ref, type Ref } from 'vue'

export interface ChatMessageFile {
  name: string
  source_name?: string
  url?: string
  [key: string]: unknown
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  /** 仅 user 消息：附带文件列表（发送时展示、加载会话时由接口解析） */
  files?: ChatMessageFile[]
  tool_calls?: Array<{ name: string; status: string; arguments?: string; result?: string; error?: string }>
  created_at?: string
}

export type StreamEventHandler = (event: string, data: Record<string, unknown>) => void

export interface UseWorkspaceChatStreamReturn {
  messages: Ref<ChatMessage[]>
  sending: Ref<boolean>
  sessionId: Ref<string | undefined>
  agentId: Ref<number | null>
  /** 由调用方在 SSE 回调里调用，用于更新最后一条 assistant 消息及 sessionId/agentId */
  handleEvent: StreamEventHandler
  /** 发送一条用户消息并跑流：追加 user + assistant，调用 streamFn(handleEvent)。可选传 files 以便发送后立即展示附件 */
  send: (content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>, files?: ChatMessageFile[]) => Promise<void>
  /** 加载/覆盖消息列表（如切换会话时） */
  setMessages: (msgs: ChatMessage[]) => void
}

export function useWorkspaceChatStream(): UseWorkspaceChatStreamReturn {
  const messages = ref<ChatMessage[]>([])
  const sending = ref(false)
  const sessionId = ref<string | undefined>(undefined)
  const agentId = ref<number | null>(null)

  function handleEvent(event: string, data: Record<string, unknown>) {
    if (event === 'session' && typeof data.session_id === 'string') {
      sessionId.value = data.session_id
    }
    if (event === 'agent_id' && data.agent_id != null && Number(data.agent_id) > 0) {
      agentId.value = Number(data.agent_id)
    }

    const lastIdx = messages.value.length - 1
    const m = messages.value[lastIdx]
    if (!m || m.role !== 'assistant') return

    if (event === 'tool_calls_stream' && Array.isArray(data.tool_calls)) {
      const streamList = (data.tool_calls as Array<{ name?: string; arguments?: string }>).map((t) => ({
        name: typeof t.name === 'string' ? t.name : '',
        status: 'streaming' as const,
        arguments: typeof t.arguments === 'string' ? t.arguments : undefined,
      }))
      // 合并：已有 running/ok/error 的项不覆盖为 streaming，但参数以新流为准（流式会逐渐补全）
      const prev = m.tool_calls || []
      const list = streamList.map((item, i) => {
        const existing = prev[i]
        const args = (item.arguments && item.arguments.trim()) ? item.arguments : (existing?.arguments ?? item.arguments)
        if (existing && ['running', 'ok', 'error'].includes(existing.status))
          return { ...item, status: existing.status, arguments: args }
        return { ...item, arguments: args }
      })
      messages.value[lastIdx] = { ...m, tool_calls: list }
    }
    if (event === 'tool_call' && typeof data.name === 'string') {
      const status = String(data.status || 'ok')
      const argumentsStr = typeof data.arguments === 'string' ? data.arguments : undefined
      const prev = m.tool_calls || []
      // 先找「第一个同名且未完成」的槽位，避免连续两次同名工具时只更新最后一个
      const pendingSameNameIndex = prev.findIndex(
        (t) => t.name === data.name && (t.status === 'streaming' || t.status === 'running')
      )
      const lastSameNameIndex =
        pendingSameNameIndex >= 0
          ? pendingSameNameIndex
          : prev.map((t, i) => (t.name === data.name ? i : -1)).filter((i) => i >= 0).pop()
      const keepArgs = (argumentsStr && argumentsStr.trim()) ? argumentsStr : undefined
      const resultStr = typeof data.result === 'string' ? data.result : undefined
      const errorStr = typeof data.error === 'string' ? data.error : undefined
      let list: Array<{ name: string; status: string; arguments?: string; result?: string; error?: string }>
      if (lastSameNameIndex !== undefined) {
        list = prev.map((t, i) =>
          i === lastSameNameIndex
            ? {
                name: data.name as string,
                status,
                arguments: keepArgs ?? t.arguments,
                result: resultStr ?? t.result,
                error: errorStr ?? t.error,
              }
            : t
        )
      } else {
        list = [...prev, { name: data.name as string, status, arguments: keepArgs, result: resultStr, error: errorStr }]
      }
      messages.value[lastIdx] = { ...m, tool_calls: list }
    }
    if (event === 'content' && typeof data.content === 'string') {
      messages.value[lastIdx] = { ...m, content: m.content + data.content }
    }
    if (event === 'done') {
      sending.value = false
      if (Array.isArray(data.tool_calls)) {
        const doneList = data.tool_calls as Array<{ name: string; status: string; arguments?: string; result?: string; error?: string }>
        // 与会话结束后通过 message 列表加载时一致：保留已有 arguments/result/error，补全 name/status 等
        const merged = doneList.map((tc, i) => ({
          ...tc,
          arguments: tc.arguments ?? m.tool_calls?.[i]?.arguments,
          result: tc.result ?? m.tool_calls?.[i]?.result,
          error: tc.error ?? m.tool_calls?.[i]?.error,
        }))
        messages.value[lastIdx] = { ...m, tool_calls: merged }
      }
    }
    if (event === 'error') {
      messages.value[lastIdx] = { ...m, content: m.content || String(data.message || '请求失败') }
      sending.value = false
    }
  }

  async function send(content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>, files?: ChatMessageFile[]) {
    if (sending.value) return
    const now = new Date().toISOString()
    messages.value.push({ role: 'user', content, files: files?.length ? files : undefined, created_at: now })
    messages.value.push({ role: 'assistant', content: '', tool_calls: [], created_at: now })
    sending.value = true
    const idx = messages.value.length - 1
    try {
      await streamFn(handleEvent)
    } catch (e: unknown) {
      const errMsg = e instanceof Error ? e.message : String(e)
      const msg = messages.value[idx]
      if (msg && msg.role === 'assistant') {
        messages.value[idx] = { ...msg, content: msg.content || `请求失败：${errMsg}` }
      }
    } finally {
      sending.value = false
    }
  }

  function setMessages(msgs: ChatMessage[]) {
    messages.value = msgs
  }

  return {
    messages,
    sending,
    sessionId,
    agentId,
    handleEvent,
    send,
    setMessages,
  }
}
