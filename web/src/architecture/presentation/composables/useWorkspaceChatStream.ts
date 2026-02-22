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
      const prev = m.tool_calls || []
      // 合并：已有 running/ok/error 的项保留 status/result/error，不覆盖；且不截断：若 prev 更长则保留多出的项，这样执行过程中已产生的文件历史不会丢
      const fromStream = streamList.map((item, i) => {
        const existing = prev[i]
        const args = (item.arguments && item.arguments.trim()) ? item.arguments : (existing?.arguments ?? item.arguments)
        if (existing && ['running', 'ok', 'error'].includes(existing.status)) {
          return {
            ...item,
            status: existing.status,
            arguments: args,
            result: existing.result,
            error: existing.error,
          }
        }
        return { ...item, arguments: args }
      })
      const list = prev.length > streamList.length
        ? fromStream.concat(prev.slice(streamList.length))
        : fromStream
      messages.value[lastIdx] = { ...m, tool_calls: list }
    }
    if (event === 'tool_call' && typeof data.name === 'string') {
      const status = String(data.status || 'ok')
      const argumentsStr = (typeof data.arguments === 'string' && data.arguments.trim()) ? data.arguments : undefined
      const resultStr = typeof data.result === 'string' ? data.result : undefined
      const errorStr = typeof data.error === 'string' ? data.error : undefined
      const prev = m.tool_calls || []
      // 按顺序更新：后端按执行顺序发 tool_call，用「第一个未完成」的槽位，避免同名/错位导致只保留最后一个
      const pendingIndex = prev.findIndex((t) => t.status === 'streaming' || t.status === 'running')
      const list: Array<{ name: string; status: string; arguments?: string; result?: string; error?: string }> =
        pendingIndex >= 0
          ? prev.map((t, i) =>
              i === pendingIndex
                ? {
                    name: data.name as string,
                    status,
                    arguments: argumentsStr ?? t.arguments,
                    result: resultStr ?? t.result,
                    error: errorStr ?? t.error,
                  }
                : t
            )
          : [...prev, { name: data.name as string, status, arguments: argumentsStr, result: resultStr, error: errorStr }]
      messages.value[lastIdx] = { ...m, tool_calls: list }
    }
    if (event === 'content' && typeof data.content === 'string') {
      messages.value[lastIdx] = { ...m, content: m.content + data.content }
    }
    if (event === 'done') {
      sending.value = false
      if (Array.isArray(data.tool_calls)) {
        const doneList = data.tool_calls as Array<{ name: string; status: string; arguments?: string; result?: string; error?: string }>
        const prev = m.tool_calls || []
        // 以当前已有 tool_calls 为基准合并，不因 doneList 更短而丢失已更新的 result（避免只展示最后一个）
        const merged = prev.map((t, i) => {
          const dc = doneList[i]
          if (!dc) return t
          return {
            ...t,
            ...dc,
            arguments: dc.arguments ?? t.arguments,
            result: dc.result ?? t.result,
            error: dc.error ?? t.error,
          }
        })
        // 若 done 列表更长则追加
        if (doneList.length > prev.length) {
          for (let i = prev.length; i < doneList.length; i++) {
            merged.push({ ...doneList[i] })
          }
        }
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
