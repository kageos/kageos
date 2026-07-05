/**
 * useWorkspaceChatStream - 流式工作台对话核心
 *
 * 维护 messages、sending、sessionId，暴露 handleEvent 与 send。
 * 调用方负责构造 payload 并调用 API（如 workspaceChatStream），在 SSE 回调里调用 handleEvent 即可。
 * 便于 Mini 工作台与后续其他流式工具对话复用同一套消息状态与事件处理。
 */

import { ref, watch, onUnmounted, type Ref } from 'vue'
import type {
  LLMUsageInfo,
  ToolResultMetadata,
  WorkspaceModelContextPlan,
  WorkspaceChatStreamOnEvent,
  WorkspaceStreamContentPayload,
  WorkspaceStreamDonePayload,
  WorkspaceStreamErrorPayload,
  WorkspaceStreamEventName,
  WorkspaceStreamPayload,
  WorkspaceStreamSessionPayload,
  WorkspaceStreamThinkingPayload,
  WorkspaceStreamToolCallPayload,
  WorkspaceStreamToolCallsDeltaPayload
} from '@/architecture/presentation/context/api/workspace'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'

export interface ChatMessageFile {
  ref?: string
  name: string
  source_name?: string
  download_url?: string
  [key: string]: unknown
}

/** 单条 assistant 消息内的工具调用项（与 ChatMessage.tool_calls 元素同构） */
export type ChatMessageToolCall = {
  id?: string
  index?: number
  round?: number
  name: string
  status: string
  arguments?: string
  result?: string
  result_data?: unknown
  metadata?: ToolResultMetadata
  error?: string
}

/** assistant 消息内的块：按事件顺序排列，用于「文本 → 工具调用 → 文本 → …」的层次展示 */
export type AssistantBlock =
  | { type: 'content'; text: string }
  | { type: 'thinking'; text: string }
  | { type: 'tool_calls'; calls: ChatMessageToolCall[] }

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  raw_content?: string
  user?: string
  /** 仅 user 消息：附带文件列表（发送时展示、加载会话时由接口解析） */
  files?: ChatMessageFile[]
  context_usage?: string
  artifact_kind?: string
  tool_calls?: ChatMessageToolCall[]
  llm_config_id?: number
  llm_config_name?: string
  llm_provider?: string
  llm_model?: string
  llm_usage?: LLMUsageInfo
  model_context_plan?: WorkspaceModelContextPlan
  model_context_plans?: WorkspaceModelContextPlan[]
  /** assistant 专用：按顺序的 content / tool_calls 块，有则按块渲染，否则退化为上面整段 content + 下面整段 tool_calls */
  blocks?: AssistantBlock[]
  created_at?: string
}

export type StreamEventHandler = WorkspaceChatStreamOnEvent

interface ActiveStreamRun {
  epoch: number
  assistantIndex: number
}

/** 流式时「已显示字符数」，用于打字机平滑（兼容端大块吐出时不再一卡一卡）；调小更丝滑、调大更快追上 */
export const SMOOTH_CHARS_PER_TICK = 5
export const SMOOTH_TICK_MS = 22

export interface UseWorkspaceChatStreamReturn {
  messages: Ref<ChatMessage[]>
  sending: Ref<boolean>
  sessionId: Ref<string | undefined>
  /** 当前流式消息的「已显示长度」，仅对最后一条 assistant 的最后一个 content 块生效，用于平滑展示 */
  streamingDisplayLength: Ref<number>
  /** 由调用方在 SSE 回调里调用，用于更新最后一条 assistant 消息及 sessionId */
  handleEvent: StreamEventHandler
  /** 发送一条用户消息并跑流：追加 user + assistant，调用 streamFn(handleEvent)。可选传 files 以便发送后立即展示附件 */
  send: (content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>, files?: ChatMessageFile[]) => Promise<void>
  /** 加载/覆盖消息列表（如切换会话时） */
  setMessages: (msgs: ChatMessage[]) => void
}

export function useWorkspaceChatStream(): UseWorkspaceChatStreamReturn {
  const authStore = useAuthStore()
  const messages = ref<ChatMessage[]>([])
  const sending = ref(false)
  const sessionId = ref<string | undefined>(undefined)
  /** 流式时最后一条 assistant 的最后一个 content 块「已显示字符数」，定时器追赶真实长度，实现打字机平滑 */
  const streamingDisplayLength = ref(0)
  let smoothTimer: ReturnType<typeof setInterval> | null = null
  let streamEpoch = 0
  let activeStreamRun: ActiveStreamRun | null = null

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
   * tool_calls_stream_delta 用：更新 blocks 中的 tool_calls 块
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
    const nextBlocks = blocks.map((block) => {
      if (block.type === 'tool_calls') {
        const count = block.calls.length
        const updated = list.slice(offset, offset + count)
        offset += count
        return { type: 'tool_calls' as const, calls: updated }
      }
      return block
    })
    if (offset < list.length) {
      nextBlocks.push({ type: 'tool_calls' as const, calls: list.slice(offset) })
    }
    return nextBlocks
  }

  function findToolCallIndex(list: ChatMessageToolCall[], id?: string, round?: number, index?: number): number {
    const normalizedID = String(id || '').trim()
    if (normalizedID) {
      const byID = list.findIndex((call) => String(call.id || '').trim() === normalizedID)
      if (byID >= 0) return byID
    }
    if (typeof round === 'number' && typeof index === 'number') {
      return list.findIndex((call) => call.round === round && call.index === index)
    }
    return -1
  }

  function mergeToolCallUpdate(
    list: ChatMessageToolCall[],
    update: Partial<ChatMessageToolCall> & { delta?: string }
  ): ChatMessageToolCall[] {
    const next = list.map((call) => ({ ...call }))
    const foundIndex = findToolCallIndex(next, update.id, update.round, update.index)
    const slotIndex = foundIndex >= 0 ? foundIndex : next.length
    const existing = next[slotIndex]
    const merged: ChatMessageToolCall = {
      ...(existing || { name: '', status: 'streaming', arguments: '' }),
      name: update.name ?? existing?.name ?? '',
      status: update.status ?? existing?.status ?? 'streaming',
      arguments: update.delta != null
        ? `${existing?.arguments || ''}${update.delta}`
        : update.arguments ?? existing?.arguments,
    }
    if (update.id !== undefined) merged.id = update.id
    if (update.index !== undefined) merged.index = update.index
    if (update.round !== undefined) merged.round = update.round
    if (update.result !== undefined) merged.result = update.result
    if (update.result_data !== undefined) merged.result_data = update.result_data
    if (update.metadata !== undefined) merged.metadata = update.metadata
    if (update.error !== undefined) merged.error = update.error
    next[slotIndex] = merged
    return next
  }

  function invalidateActiveStream() {
    streamEpoch += 1
    activeStreamRun = null
  }

  function isActiveStreamRun(run: ActiveStreamRun): boolean {
    return activeStreamRun === run && run.epoch === streamEpoch
  }

  function handleEvent(event: WorkspaceStreamEventName, data: WorkspaceStreamPayload) {
    return applyEventToAssistant(messages.value.length - 1, event, data)
  }

  function handleStreamEvent(run: ActiveStreamRun, event: WorkspaceStreamEventName, data: WorkspaceStreamPayload) {
    if (!isActiveStreamRun(run)) return false
    const accepted = applyEventToAssistant(run.assistantIndex, event, data)
    if (accepted && (event === 'done' || event === 'error') && activeStreamRun === run) {
      activeStreamRun = null
    }
    return accepted
  }

  function applyEventToAssistant(targetIdx: number, event: WorkspaceStreamEventName, data: WorkspaceStreamPayload): boolean {
    if (event === 'session') {
      const payload = data as WorkspaceStreamSessionPayload
      if (typeof payload.session_id === 'string') {
        sessionId.value = payload.session_id
      }
    }

    const m = messages.value[targetIdx]
    if (!m || m.role !== 'assistant') return false

    if (event === 'model_context_plan') {
      const payload = data as WorkspaceModelContextPlan
      if (isWorkspaceModelContextPlan(payload)) {
        messages.value[targetIdx] = {
          ...m,
          model_context_plan: payload,
          model_context_plans: upsertWorkspaceModelContextPlan(m.model_context_plans, payload)
        }
      }
    }
    // 增量协议：tool_calls_stream_delta（index + 可选 name + delta）
    if (event === 'tool_calls_stream_delta') {
      const payload = data as WorkspaceStreamToolCallsDeltaPayload
      const updates = Array.isArray(payload.updates) ? payload.updates : []
      if (updates.length === 0) return true
      const blocks = m.blocks ?? []
      let list = m.tool_calls || []

      for (const u of updates) {
        const idx = typeof u.index === 'number' ? u.index : 0
        const round = typeof u.round === 'number' ? u.round : undefined
        const id = typeof u.id === 'string' ? u.id : undefined
        const delta = typeof u.delta === 'string' ? u.delta : ''
        const name = typeof u.name === 'string' && u.name ? u.name : undefined
        list = mergeToolCallUpdate(list, { id, index: idx, round, name, delta, status: 'streaming' })
      }

      const nextBlocks = updateToolCallsBlocks(blocks, list)
      messages.value[targetIdx] = { ...m, tool_calls: list, blocks: nextBlocks }
    }
    if (event === 'tool_call') {
      const payload = data as WorkspaceStreamToolCallPayload
      if (typeof payload.name !== 'string') return true
      const status = String(payload.status || 'ok')
      const argumentsStr = (typeof payload.arguments === 'string' && payload.arguments.trim()) ? payload.arguments : undefined
      const resultStr = typeof payload.result === 'string' ? payload.result : undefined
      const resultData = Object.prototype.hasOwnProperty.call(payload, 'result_data') ? payload.result_data : undefined
      const metadata = Object.prototype.hasOwnProperty.call(payload, 'metadata') ? payload.metadata as ToolResultMetadata : undefined
      const errorStr = typeof payload.error === 'string' ? payload.error : undefined
      const blocks = m.blocks ?? []
      const prev = m.tool_calls || []
      const id = typeof payload.id === 'string' ? payload.id : undefined
      const index = typeof payload.index === 'number' ? payload.index : undefined
      const round = typeof payload.round === 'number' ? payload.round : undefined
      const list = mergeToolCallUpdate(prev, {
        id,
        index,
        round,
        name: payload.name,
        status,
        arguments: argumentsStr,
        result: resultStr,
        result_data: resultData,
        metadata,
        error: errorStr
      })
      const nextBlocks = updateLastToolCallsBlock(blocks, list)
      messages.value[targetIdx] = { ...m, tool_calls: list, blocks: nextBlocks }
    }
    if (event === 'thinking') {
      const payload = data as WorkspaceStreamThinkingPayload
      if (typeof payload.content !== 'string' || payload.content === '') return true
      const blocks = m.blocks ?? []
      const last = blocks[blocks.length - 1]
      if (last && last.type === 'thinking') {
        const next = [...blocks.slice(0, -1), { type: 'thinking' as const, text: last.text + payload.content }]
        messages.value[targetIdx] = { ...m, blocks: next }
      } else {
        messages.value[targetIdx] = { ...m, blocks: [...blocks, { type: 'thinking', text: payload.content }] }
      }
    }
    if (event === 'content') {
      const payload = data as WorkspaceStreamContentPayload
      if (typeof payload.content !== 'string') return true
      const blocks = m.blocks ?? []
      const last = blocks[blocks.length - 1]
      const newContent = m.content + payload.content
      if (last && last.type === 'content') {
        const next = [...blocks.slice(0, -1), { type: 'content' as const, text: last.text + payload.content }]
        messages.value[targetIdx] = { ...m, content: newContent, blocks: next }
      } else {
        messages.value[targetIdx] = { ...m, content: newContent, blocks: [...blocks, { type: 'content', text: payload.content }] }
      }
    }
    if (event === 'done') {
      const payload = data as WorkspaceStreamDonePayload
      sending.value = false
      const llmMeta = extractLLMMetadata(payload, m)
      if (Array.isArray(payload.tool_calls)) {
        const doneList = payload.tool_calls as ChatMessageToolCall[]
        const prev = m.tool_calls || []
        const merged = prev.map((call) => ({ ...call }))
        for (const doneCall of doneList) {
          const foundIndex = findToolCallIndex(merged, doneCall.id, doneCall.round, doneCall.index)
          if (foundIndex >= 0) {
            const current = merged[foundIndex] || { name: doneCall.name || '', status: doneCall.status || 'ok' }
            merged[foundIndex] = {
              ...current,
              ...doneCall,
              arguments: doneCall.arguments ?? current.arguments,
              result: doneCall.result ?? current.result,
              result_data: doneCall.result_data ?? current.result_data,
              error: doneCall.error ?? current.error
            }
          } else {
            merged.push({ ...doneCall })
          }
        }
        const blocks = m.blocks ?? []
        // 保留现有 block 结构，只更新每个 tool_calls block 的数据
        const nextBlocks = rebuildAllToolCallsBlocks(blocks, merged)
        messages.value[targetIdx] = { ...m, ...llmMeta, tool_calls: merged, blocks: nextBlocks }
      } else {
        messages.value[targetIdx] = { ...m, ...llmMeta }
      }
    }
    if (event === 'error') {
      const payload = data as WorkspaceStreamErrorPayload
      const rawErr = String(payload.message || '请求失败')
      const isCancelled = /context canceled|cancelled|abort/i.test(rawErr)
      if (isCancelled) {
        const hint = '⏹ 任务已停止'
        const blocks = m.blocks ?? []
        messages.value[targetIdx] = { ...m, blocks: [...blocks, { type: 'content' as const, text: hint }] }
      } else {
        const blocks = m.blocks ?? []
        const last = blocks[blocks.length - 1]
        const nextBlocks =
          last && last.type === 'content'
            ? [...blocks.slice(0, -1), { type: 'content' as const, text: last.text + (m.content ? '\n\n' : '') + rawErr }]
            : [...blocks, { type: 'content' as const, text: rawErr }]
        messages.value[targetIdx] = { ...m, content: m.content || rawErr, blocks: nextBlocks }
      }
      sending.value = false
    }
    return true
  }

  function extractLLMMetadata(data: WorkspaceStreamDonePayload, message?: ChatMessage): Partial<ChatMessage> {
    const meta: Partial<ChatMessage> = {}
    if (typeof data.llm_config_id === 'number') meta.llm_config_id = data.llm_config_id
    if (typeof data.llm_config_name === 'string') meta.llm_config_name = data.llm_config_name
    if (typeof data.llm_provider === 'string') meta.llm_provider = data.llm_provider
    if (typeof data.llm_model === 'string') meta.llm_model = data.llm_model
    if (isLLMUsageInfo(data.llm_usage)) meta.llm_usage = data.llm_usage
    if (isWorkspaceModelContextPlan(data.model_context_plan)) {
      meta.model_context_plan = data.model_context_plan
      meta.model_context_plans = upsertWorkspaceModelContextPlan(message?.model_context_plans, data.model_context_plan)
    }
    return meta
  }

  function upsertWorkspaceModelContextPlan(list: WorkspaceModelContextPlan[] | undefined, plan: WorkspaceModelContextPlan): WorkspaceModelContextPlan[] {
    const existing = Array.isArray(list) ? [...list] : []
    const idx = existing.findIndex(item => item.round === plan.round && item.session_id === plan.session_id)
    if (idx >= 0) {
      existing[idx] = plan
      return existing
    }
    return [...existing, plan].sort((left, right) => left.round - right.round)
  }

  function isWorkspaceModelContextPlan(value: unknown): value is WorkspaceModelContextPlan {
    if (!value || typeof value !== 'object') return false
    const plan = value as Partial<WorkspaceModelContextPlan>
    return typeof plan.protocol_version === 'string' &&
      typeof plan.session_id === 'string' &&
      typeof plan.round === 'number' &&
      !!plan.role &&
      typeof plan.role === 'object' &&
      !!plan.execution &&
      typeof plan.execution === 'object' &&
      !!plan.messages &&
      typeof plan.messages === 'object'
  }

  function isLLMUsageInfo(value: unknown): value is LLMUsageInfo {
    if (!value || typeof value !== 'object') return false
    const usage = value as Partial<Record<keyof LLMUsageInfo, unknown>>
    return typeof usage.prompt_tokens === 'number' &&
      typeof usage.completion_tokens === 'number' &&
      typeof usage.total_tokens === 'number' &&
      typeof usage.cached_tokens === 'number' &&
      (usage.cached_tokens_reported === undefined || typeof usage.cached_tokens_reported === 'boolean')
  }

  async function send(content: string, streamFn: (onEvent: StreamEventHandler) => Promise<void>, files?: ChatMessageFile[]) {
    if (sending.value) return
    streamingDisplayLength.value = 0
    const now = new Date().toISOString()
    const currentUser = authStore.user?.username || authStore.userName || ''
    messages.value.push({ role: 'user', user: currentUser, content, files: files?.length ? files : undefined, created_at: now })
    messages.value.push({ role: 'assistant', user: currentUser, content: '', tool_calls: [], blocks: [], created_at: now })
    sending.value = true
    const idx = messages.value.length - 1
    const run: ActiveStreamRun = {
      epoch: streamEpoch,
      assistantIndex: idx
    }
    activeStreamRun = run
    try {
      await streamFn((event, data) => handleStreamEvent(run, event, data))
    } catch (e: unknown) {
      if (!isActiveStreamRun(run)) return
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
      if (isActiveStreamRun(run)) {
        activeStreamRun = null
        sending.value = false
      }
    }
  }

  function setMessages(msgs: ChatMessage[]) {
    invalidateActiveStream()
    messages.value = msgs
  }

  return {
    messages,
    sending,
    sessionId,
    streamingDisplayLength,
    handleEvent,
    send,
    setMessages,
  }
}
