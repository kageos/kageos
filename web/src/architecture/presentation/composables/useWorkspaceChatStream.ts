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

/** 单条 assistant 消息内的工具调用项（与 ChatMessage.tool_calls 元素同构） */
export type ChatMessageToolCall = { name: string; status: string; arguments?: string; result?: string; error?: string }

/** assistant 消息内的块：按事件顺序排列，用于「文本 → 工具调用 → 文本 → …」的层次展示 */
export type AssistantBlock =
  | { type: 'content'; text: string }
  | { type: 'tool_calls'; calls: ChatMessageToolCall[] }

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  /** 仅 user 消息：附带文件列表（发送时展示、加载会话时由接口解析） */
  files?: ChatMessageFile[]
  tool_calls?: ChatMessageToolCall[]
  /** assistant 专用：按顺序的 content / tool_calls 块，有则按块渲染，否则退化为上面整段 content + 下面整段 tool_calls */
  blocks?: AssistantBlock[]
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
      const blocks = m.blocks ?? []
      const prev = m.tool_calls || []
      const fromStream = streamList.map((item, i) => {
        const existing = prev[i]
        const args = (item.arguments && item.arguments.trim()) ? item.arguments : (existing?.arguments ?? item.arguments)
        if (existing && ['running', 'ok', 'error'].includes(existing.status)) {
          return { ...item, status: existing.status, arguments: args, result: existing.result, error: existing.error }
        }
        return { ...item, arguments: args }
      })
      const list = prev.length > streamList.length ? fromStream.concat(prev.slice(streamList.length)) : fromStream
      const last = blocks[blocks.length - 1]
      let nextBlocks: AssistantBlock[]
      if (last && last.type === 'tool_calls') {
        nextBlocks = [...blocks.slice(0, -1), { type: 'tool_calls', calls: list }]
      } else {
        nextBlocks = [...blocks, { type: 'tool_calls', calls: list }]
      }
      messages.value[lastIdx] = { ...m, tool_calls: list, blocks: nextBlocks }
    }
    if (event === 'tool_call' && typeof data.name === 'string') {
      const status = String(data.status || 'ok')
      const argumentsStr = (typeof data.arguments === 'string' && data.arguments.trim()) ? data.arguments : undefined
      const resultStr = typeof data.result === 'string' ? data.result : undefined
      const errorStr = typeof data.error === 'string' ? data.error : undefined
      const blocks = m.blocks ?? []
      const prev = m.tool_calls || []
      const pendingIndex = prev.findIndex((t) => t.status === 'streaming' || t.status === 'running')
      const list: ChatMessageToolCall[] =
        pendingIndex >= 0
          ? prev.map((t, i) =>
              i === pendingIndex
                ? { name: data.name as string, status, arguments: argumentsStr ?? t.arguments, result: resultStr ?? t.result, error: errorStr ?? t.error }
                : t
            )
          : [...prev, { name: data.name as string, status, arguments: argumentsStr, result: resultStr, error: errorStr }]
      // 更新最后一个 tool_calls 块
      const tcBlockIdx = blocks.map((b, i) => (b.type === 'tool_calls' ? i : -1)).filter((i) => i >= 0).pop()
      const nextBlocks =
        tcBlockIdx !== undefined
          ? [...blocks.slice(0, tcBlockIdx), { type: 'tool_calls' as const, calls: list }, ...blocks.slice(tcBlockIdx + 1)]
          : [...blocks, { type: 'tool_calls' as const, calls: list }]
      messages.value[lastIdx] = { ...m, tool_calls: list, blocks: nextBlocks }
    }
    if (event === 'content' && typeof data.content === 'string') {
      const blocks = m.blocks ?? []
      const last = blocks[blocks.length - 1]
      const newContent = m.content + data.content
      if (last && last.type === 'content') {
        const next = [...blocks.slice(0, -1), { type: 'content' as const, text: last.text + data.content }]
        messages.value[lastIdx] = { ...m, content: newContent, blocks: next }
      } else {
        messages.value[lastIdx] = { ...m, content: newContent, blocks: [...blocks, { type: 'content', text: data.content }] }
      }
    }
    if (event === 'done') {
      sending.value = false
      if (Array.isArray(data.tool_calls)) {
        const doneList = data.tool_calls as ChatMessageToolCall[]
        const prev = m.tool_calls || []
        const merged = prev.map((t, i) => {
          const dc = doneList[i]
          if (!dc) return t
          return { ...t, ...dc, arguments: dc.arguments ?? t.arguments, result: dc.result ?? t.result, error: dc.error ?? t.error }
        })
        if (doneList.length > prev.length) {
          for (let i = prev.length; i < doneList.length; i++) merged.push({ ...doneList[i] })
        }
        const blocks = m.blocks ?? []
        const tcBlockIdx = blocks.map((b, i) => (b.type === 'tool_calls' ? i : -1)).filter((i) => i >= 0).pop()
        const nextBlocks =
          tcBlockIdx !== undefined
            ? [...blocks.slice(0, tcBlockIdx), { type: 'tool_calls' as const, calls: merged }, ...blocks.slice(tcBlockIdx + 1)]
            : [...blocks, { type: 'tool_calls' as const, calls: merged }]
        messages.value[lastIdx] = { ...m, tool_calls: merged, blocks: nextBlocks }
      }
    }
    if (event === 'error') {
      const errText = String(data.message || '请求失败')
      const blocks = m.blocks ?? []
      const last = blocks[blocks.length - 1]
      const nextBlocks =
        last && last.type === 'content'
          ? [...blocks.slice(0, -1), { type: 'content' as const, text: last.text + (m.content ? '\n\n' : '') + errText }]
          : [...blocks, { type: 'content' as const, text: errText }]
      messages.value[lastIdx] = { ...m, content: m.content || errText, blocks: nextBlocks }
      sending.value = false
    }
  }

  async function send(content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>, files?: ChatMessageFile[]) {
    if (sending.value) return
    const now = new Date().toISOString()
    messages.value.push({ role: 'user', content, files: files?.length ? files : undefined, created_at: now })
    messages.value.push({ role: 'assistant', content: '', tool_calls: [], blocks: [], created_at: now })
    sending.value = true
    const idx = messages.value.length - 1
    try {
      await streamFn(handleEvent)
    } catch (e: unknown) {
      const errMsg = e instanceof Error ? e.message : String(e)
      const msg = messages.value[idx]
      if (msg && msg.role === 'assistant') {
        const text = msg.content || `请求失败：${errMsg}`
        messages.value[idx] = { ...msg, content: text, blocks: [{ type: 'content', text }] }
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
