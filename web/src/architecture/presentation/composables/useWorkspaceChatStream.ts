/**
 * useWorkspaceChatStream - 流式工作台对话核心
 *
 * 维护 messages、sending、sessionId、agentId，暴露 handleEvent 与 send。
 * 调用方负责构造 payload 并调用 API（如 workspaceChatStream），在 SSE 回调里调用 handleEvent 即可。
 * 便于 Mini 工作台与后续其他流式工具对话复用同一套消息状态与事件处理。
 */

import { ref, watch, onUnmounted, type Ref } from 'vue'

export interface ChatMessageFile {
  name: string
  source_name?: string
  url: string
  [key: string]: unknown
}

/** 单条 assistant 消息内的工具调用项（与 ChatMessage.tool_calls 元素同构） */
export type ChatMessageToolCall = { name: string; status: string; arguments?: string; result?: string; result_data?: unknown; error?: string }

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

/** 流式时「已显示字符数」，用于打字机平滑（MiniMax 等大块吐出时不再一卡一卡）；调小更丝滑、调大更快追上 */
export const SMOOTH_CHARS_PER_TICK = 5
export const SMOOTH_TICK_MS = 22

export interface UseWorkspaceChatStreamReturn {
  messages: Ref<ChatMessage[]>
  sending: Ref<boolean>
  sessionId: Ref<string | undefined>
  agentId: Ref<number | null>
  /** 当前流式消息的「已显示长度」，仅对最后一条 assistant 的最后一个 content 块生效，用于平滑展示 */
  streamingDisplayLength: Ref<number>
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
  /** 流式时最后一条 assistant 的最后一个 content 块「已显示字符数」，定时器追赶真实长度，实现打字机平滑 */
  const streamingDisplayLength = ref(0)
  let smoothTimer: ReturnType<typeof setInterval> | null = null

  function stopSmoothTimer() {
    if (smoothTimer != null) {
      clearInterval(smoothTimer)
      smoothTimer = null
    }
  }

  watch(sending, (val) => {
    if (!val) {
      stopSmoothTimer()
      return
    }
    streamingDisplayLength.value = 0
    smoothTimer = setInterval(() => {
      const list = messages.value
      const last = list[list.length - 1]
      if (!last || last.role !== 'assistant') return
      const blocks = last.blocks ?? []
      const lastBlock = blocks[blocks.length - 1]
      if (!lastBlock || lastBlock.type !== 'content') return
      const fullLen = lastBlock.text.length
      if (streamingDisplayLength.value < fullLen) {
        streamingDisplayLength.value = Math.min(
          streamingDisplayLength.value + SMOOTH_CHARS_PER_TICK,
          fullLen
        )
      }
    }, SMOOTH_TICK_MS)
  })

  onUnmounted(stopSmoothTimer)

  /** 统计 blocks 中已经分配的 tool_calls 数量（排除指定 blockIndex） */
  function countCallsInBlocks(blocks: AssistantBlock[], excludeIdx?: number): number {
    let count = 0
    for (const [i, block] of blocks.entries()) {
      if (i === excludeIdx) continue
      if (block.type === 'tool_calls') count += block.calls.length
    }
    return count
  }

  /**
   * tool_calls_stream 用：更新 blocks 中的 tool_calls 块
   * - last block 有未完成的 → 更新它
   * - last block 全部完成且有新的流式 → 创建新 block（每个工具一块）
   * - last block 不是 tool_calls → 创建新 block
   */
  function updateToolCallsBlocks(blocks: AssistantBlock[], list: ChatMessageToolCall[]): AssistantBlock[] {
    const lastBlock = blocks[blocks.length - 1]
    if (lastBlock && lastBlock.type === 'tool_calls') {
      const lastBlockAllDone = lastBlock.calls.every((c) => c.status === 'ok' || c.status === 'error')
      const prevCount = countCallsInBlocks(blocks, blocks.length - 1)
      const callsForBlock = list.slice(prevCount)

      if (lastBlockAllDone && callsForBlock.length > lastBlock.calls.length) {
        // 上一块全部完成，有新的流式工具 → 保留上一块，为新工具创建新 block
        const newCalls = callsForBlock.slice(lastBlock.calls.length)
        return [...blocks, { type: 'tool_calls' as const, calls: newCalls }]
      }
      // 更新当前块（正在流式的工具参数在变长、或状态在更新）
      return [...blocks.slice(0, -1), { type: 'tool_calls' as const, calls: callsForBlock }]
    }
    const prevCount = countCallsInBlocks(blocks)
    const newCalls = list.slice(prevCount)
    return newCalls.length > 0 ? [...blocks, { type: 'tool_calls' as const, calls: newCalls }] : blocks
  }

  /**
   * tool_call 事件用：更新「最后一个 tool_calls block」
   */
  function updateLastToolCallsBlock(blocks: AssistantBlock[], list: ChatMessageToolCall[]): AssistantBlock[] {
    const tcBlockIdx = blocks.map((b, i) => (b.type === 'tool_calls' ? i : -1)).filter((i) => i >= 0).pop()
    if (tcBlockIdx !== undefined) {
      const prevCount = countCallsInBlocks(blocks, tcBlockIdx)
      return [...blocks.slice(0, tcBlockIdx), { type: 'tool_calls' as const, calls: list.slice(prevCount) }, ...blocks.slice(tcBlockIdx + 1)]
    }
    const prevCount = countCallsInBlocks(blocks)
    const newCalls = list.slice(prevCount)
    return newCalls.length > 0 ? [...blocks, { type: 'tool_calls' as const, calls: newCalls }] : blocks
  }

  /**
   * done 事件用：保留现有 block 结构，只用 merged 数据更新每个 tool_calls block 的内容
   */
  function rebuildAllToolCallsBlocks(blocks: AssistantBlock[], list: ChatMessageToolCall[]): AssistantBlock[] {
    let offset = 0
    return blocks.map((block) => {
      if (block.type === 'tool_calls') {
        const count = block.calls.length
        const updated = list.slice(offset, offset + count)
        offset += count
        return { type: 'tool_calls' as const, calls: updated }
      }
      return block
    })
  }

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

    // 增量协议：tool_calls_stream_delta（index + 可选 name + delta）
    if (event === 'tool_calls_stream_delta' && Array.isArray(data.updates)) {
      const updates = data.updates as Array<{ index?: number; name?: string; delta?: string }>
      const blocks = m.blocks ?? []
      const prev = m.tool_calls || []

      let completedCount = 0
      for (const t of prev) {
        if (t.status === 'ok' || t.status === 'error') completedCount++
        else break
      }
      const completedCalls = prev.slice(0, completedCount)
      const currentRoundPrev = prev.slice(completedCount)

      for (const u of updates) {
        const idx = typeof u.index === 'number' ? u.index : 0
        const delta = typeof u.delta === 'string' ? u.delta : ''
        const name = typeof u.name === 'string' ? u.name : ''

        while (currentRoundPrev.length <= idx) {
          currentRoundPrev.push({ name: '', status: 'streaming', arguments: '' })
        }
        let slot = currentRoundPrev[idx]
        if (!slot) {
          slot = { name: '', status: 'streaming', arguments: '' }
          currentRoundPrev[idx] = slot
        }
        if (name) slot.name = name
        slot.arguments = (slot.arguments || '') + delta
        slot.status = 'streaming'
      }

      const list = [...completedCalls, ...currentRoundPrev]
      const nextBlocks = updateToolCallsBlocks(blocks, list)
      messages.value[lastIdx] = { ...m, tool_calls: list, blocks: nextBlocks }
    }
    // 兼容旧协议：tool_calls_stream（全量，后端已改为发 delta，此处保留以防回滚）
    if (event === 'tool_calls_stream' && Array.isArray(data.tool_calls)) {
      const streamList = (data.tool_calls as Array<{ name?: string; arguments?: string }>).map((t) => ({
        name: typeof t.name === 'string' ? t.name : '',
        status: 'streaming' as const,
        arguments: typeof t.arguments === 'string' ? t.arguments : undefined,
      }))
      const blocks = m.blocks ?? []
      const prev = m.tool_calls || []

      let completedCount = 0
      for (const t of prev) {
        if (t.status === 'ok' || t.status === 'error') completedCount++
        else break
      }
      const completedCalls = prev.slice(0, completedCount)
      const currentRoundPrev = prev.slice(completedCount)

      const fromStream: ChatMessageToolCall[] = streamList.map((item, i) => {
        const existing = currentRoundPrev[i]
        const args = (item.arguments && item.arguments.trim()) ? item.arguments : (existing?.arguments ?? item.arguments)
        if (existing && ['running', 'ok', 'error'].includes(existing.status)) {
          return { ...item, status: existing.status, arguments: args, result: existing.result, result_data: existing.result_data, error: existing.error }
        }
        return { ...item, arguments: args }
      })
      const currentRound: ChatMessageToolCall[] = currentRoundPrev.length > streamList.length
        ? [...fromStream, ...currentRoundPrev.slice(streamList.length)]
        : fromStream

      const list = [...completedCalls, ...currentRound]
      const nextBlocks = updateToolCallsBlocks(blocks, list)
      messages.value[lastIdx] = { ...m, tool_calls: list, blocks: nextBlocks }
    }
    if (event === 'tool_call' && typeof data.name === 'string') {
      const status = String(data.status || 'ok')
      const argumentsStr = (typeof data.arguments === 'string' && data.arguments.trim()) ? data.arguments : undefined
      const resultStr = typeof data.result === 'string' ? data.result : undefined
      const resultData = Object.prototype.hasOwnProperty.call(data, 'result_data') ? data.result_data : undefined
      const errorStr = typeof data.error === 'string' ? data.error : undefined
      const blocks = m.blocks ?? []
      const prev = m.tool_calls || []
      const pendingIndex = prev.findIndex((t) => t.status === 'streaming' || t.status === 'running')
      const list: ChatMessageToolCall[] =
        pendingIndex >= 0
          ? prev.map((t, i) =>
              i === pendingIndex
                ? { name: data.name as string, status, arguments: argumentsStr ?? t.arguments, result: resultStr ?? t.result, result_data: resultData ?? t.result_data, error: errorStr ?? t.error }
                : t
            )
          : [...prev, { name: data.name as string, status, arguments: argumentsStr, result: resultStr, result_data: resultData, error: errorStr }]
      const nextBlocks = updateLastToolCallsBlock(blocks, list)
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
          return {
            ...t,
            ...dc,
            arguments: dc.arguments ?? t.arguments,
            result: dc.result ?? t.result,
            result_data: dc.result_data ?? t.result_data,
            error: dc.error ?? t.error
          }
        })
        if (doneList.length > prev.length) {
          for (let i = prev.length; i < doneList.length; i++) {
            const nextCall = doneList[i]
            if (nextCall) merged.push({ ...nextCall })
          }
        }
        const blocks = m.blocks ?? []
        // 保留现有 block 结构，只更新每个 tool_calls block 的数据
        const nextBlocks = rebuildAllToolCallsBlocks(blocks, merged)
        messages.value[lastIdx] = { ...m, tool_calls: merged, blocks: nextBlocks }
      }
    }
    if (event === 'error') {
      const rawErr = String(data.message || '请求失败')
      const isCancelled = /context canceled|cancelled|abort/i.test(rawErr)
      if (isCancelled) {
        const hint = '⏹ 任务已停止'
        const blocks = m.blocks ?? []
        messages.value[lastIdx] = { ...m, blocks: [...blocks, { type: 'content' as const, text: hint }] }
      } else {
        const blocks = m.blocks ?? []
        const last = blocks[blocks.length - 1]
        const nextBlocks =
          last && last.type === 'content'
            ? [...blocks.slice(0, -1), { type: 'content' as const, text: last.text + (m.content ? '\n\n' : '') + rawErr }]
            : [...blocks, { type: 'content' as const, text: rawErr }]
        messages.value[lastIdx] = { ...m, content: m.content || rawErr, blocks: nextBlocks }
      }
      sending.value = false
    }
  }

  async function send(content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>, files?: ChatMessageFile[]) {
    if (sending.value) return
    streamingDisplayLength.value = 0
    const now = new Date().toISOString()
    messages.value.push({ role: 'user', content, files: files?.length ? files : undefined, created_at: now })
    messages.value.push({ role: 'assistant', content: '', tool_calls: [], blocks: [], created_at: now })
    sending.value = true
    const idx = messages.value.length - 1
    try {
      await streamFn(handleEvent)
    } catch (e: unknown) {
      const errMsg = e instanceof Error ? e.message : String(e)
      const isCancelled = /context canceled|cancelled|abort/i.test(errMsg)
      if (!isCancelled) {
        const msg = messages.value[idx]
        if (msg && msg.role === 'assistant') {
          const text = msg.content || `请求失败：${errMsg}`
          messages.value[idx] = { ...msg, content: text, blocks: [{ type: 'content', text }] }
        }
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
    streamingDisplayLength,
    handleEvent,
    send,
    setMessages,
  }
}
