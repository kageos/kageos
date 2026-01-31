/**
 * useWorkspaceChatStream - 流式工作台对话核心
 *
 * 维护 messages、sending、sessionId、agentId，暴露 handleEvent 与 send。
 * 调用方负责构造 payload 并调用 API（如 workspaceChatStream），在 SSE 回调里调用 handleEvent 即可。
 * 便于 WorkstationChat 与后续其他流式工具对话复用同一套消息状态与事件处理。
 */

import { ref, type Ref } from 'vue'

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  tool_calls?: Array<{ name: string; status: string; arguments?: string }>
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
  /** 发送一条用户消息并跑流：追加 user + assistant，调用 streamFn(handleEvent)，streamFn 内调 API 并传 handleEvent */
  send: (content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>) => Promise<void>
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
      const list = (data.tool_calls as Array<{ name?: string; arguments?: string }>).map((t) => ({
        name: typeof t.name === 'string' ? t.name : '',
        status: 'streaming' as const,
        arguments: typeof t.arguments === 'string' ? t.arguments : undefined,
      }))
      messages.value[lastIdx] = { ...m, tool_calls: list }
    }
    if (event === 'tool_call' && typeof data.name === 'string') {
      const status = String(data.status || 'ok')
      const argumentsStr = typeof data.arguments === 'string' ? data.arguments : undefined
      const prev = m.tool_calls || []
      const lastSameNameIndex = prev.map((t, i) => (t.name === data.name ? i : -1)).filter((i) => i >= 0).pop()
      let list: Array<{ name: string; status: string; arguments?: string }>
      if (lastSameNameIndex !== undefined) {
        list = prev.map((t, i) =>
          i === lastSameNameIndex ? { name: data.name as string, status, arguments: argumentsStr ?? t.arguments } : t
        )
      } else {
        list = [...prev, { name: data.name as string, status, arguments: argumentsStr }]
      }
      messages.value[lastIdx] = { ...m, tool_calls: list }
    }
    if (event === 'content' && typeof data.content === 'string') {
      messages.value[lastIdx] = { ...m, content: m.content + data.content }
    }
    if (event === 'done') {
      sending.value = false
      if (Array.isArray(data.tool_calls)) {
        messages.value[lastIdx] = { ...m, tool_calls: data.tool_calls as Array<{ name: string; status: string }> }
      }
    }
    if (event === 'error') {
      messages.value[lastIdx] = { ...m, content: m.content || String(data.message || '请求失败') }
      sending.value = false
    }
  }

  async function send(content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>) {
    if (sending.value) return
    const now = new Date().toISOString()
    messages.value.push({ role: 'user', content, created_at: now })
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
