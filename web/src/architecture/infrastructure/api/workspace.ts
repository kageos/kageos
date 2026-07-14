import { getApiBaseURL } from '@/architecture/infrastructure/config/runtime'
import { authFetch, get, post } from '@/architecture/infrastructure/apiClient/request'

/** 工作台消息中上传文件：稳定引用 bucket/object_key */
export interface WorkspaceChatMessageFile {
  ref: string
  bucket?: string
  key?: string
  name: string
  source_name?: string
  storage?: string
  description?: string
  hash?: string
  size?: number
  upload_ts?: number
  is_uploaded?: boolean
  download_url?: string
  server_download_url?: string
  upload_user?: string
}

export interface WorkspaceChatMessageFiles {
  refs: string
}

/** 工作台对话请求（只认 LLM，单模式） */
export interface WorkspaceChatReq {
  full_code_path: string
  /** 会话归属的具体函数/资源；Agent 执行仍由服务端使用父目录 */
  resource_full_code_path?: string
  message: {
    content: string
    display_content?: string
    files?: string
    context_usage?: string
    artifact_kind?: string
    interaction_action?: string
  }
  session_id?: string
  mode_code?: string
  /** LLM 配置 ID，0 表示使用默认 LLM */
  llm_config_id?: number
  /** 不保存本次 message，只基于已有会话消息继续执行 */
  resume?: boolean
}

/** 工作台会话项 */
export interface WorkspaceSessionItem {
  session_id: string
  title: string
  source?: 'workspace' | 'automation_agent' | string
  automation_task_id?: number
  automation_task_code?: string
  automation_task_title?: string
  user?: string
  mode_code?: string
  status: string // active | generating | output | pending_confirmation | pending_build_repair | done | cancelled
  role_id?: string
  role_display_name?: string
  full_code_path?: string
  directory_name?: string
  resource_tree_id?: number
  resource_full_code_path?: string
  resource_name?: string
  parent_session_id?: string
  handoff_kind?: string
  handoff_target_role?: string
  context_policy?: string
  model_context_anchor_message_id?: number
  archived_for_model?: boolean
  archive_reason?: string
  pending_interaction?: WorkspaceInteraction
  created_at: string
  updated_at: string
}

export interface WorkspaceInteraction {
  id?: string
  card_type?: string
  artifact_kind?: string
  status: string
  blocking: boolean
  title?: string
  description?: string
  help_text?: string
  view_text?: string
  confirm_text?: string
  revise_text?: string
  cancel_text?: string
  target_role_on_confirm?: string
  allowed_actions?: string[]
  artifact?: unknown
}

export interface WorkspaceInteractionEventReq {
  session_id: string
  action: string
  interaction_id?: string
  card_type?: string
  status?: string
  artifact_kind?: string
  content?: string
  display_content?: string
}

export interface WorkspaceHandoffReq {
  source_session_id: string
  full_code_path: string
  target_role: string
  artifact_kind: string
  artifact: unknown
  remark?: string
  context_policy?: string
  title?: string
  display_content?: string
}

export interface WorkspaceHandoffResp {
  session_id: string
  source_session_id: string
  target_role: string
  artifact_kind: string
  context_policy: string
  handoff_packet_id?: number
  message_id?: number
  content: string
  display_content: string
  handoff_context?: string
}

/** 获取工作台会话列表请求 */
export interface ListWorkspaceSessionsReq {
  full_code_path: string
  page?: number
  page_size?: number
  session_scope?: 'human' | 'automation' | 'all'
  automation_task_id?: number
}

export interface WorkspaceAutomationAgentItem {
  task_id: number
  task_code?: string
  task_title: string
}

/** 获取工作台会话列表响应 */
export interface ListWorkspaceSessionsResp {
  sessions: WorkspaceSessionItem[]
  automation_agents: WorkspaceAutomationAgentItem[]
  total: number
  page: number
  page_size: number
}

export interface ToolResultMetadata {
  display_file_fields?: string[]
}

export interface WorkspaceToolField {
  name: string
  type?: string
  description?: string
  required?: boolean
}

export interface WorkspaceToolDetail {
  name: string
  description: string
  token: string
  type_label: string
  input_fields?: WorkspaceToolField[]
  input_schema?: Record<string, unknown>
  output_schema?: Record<string, unknown>
}

export interface BatchWorkspaceToolDetailsReq {
  names: string[]
  include_schema?: boolean
}

export interface BatchWorkspaceToolDetailsResp {
  tools: WorkspaceToolDetail[]
  missing?: string[]
}

export const workspaceStreamEvents = {
  session: 'session',
  modelContextPlan: 'model_context_plan',
  toolCall: 'tool_call',
  toolCallsStreamDelta: 'tool_calls_stream_delta',
  thinking: 'thinking',
  content: 'content',
  done: 'done',
  error: 'error'
} as const

export type WorkspaceStreamEventName = typeof workspaceStreamEvents[keyof typeof workspaceStreamEvents]
export type WorkspaceToolCallStatus = 'ok' | 'error' | 'running' | 'streaming'

export interface WorkspaceStreamSessionPayload {
  session_id: string
}

export interface WorkspaceStreamToolCallPayload {
  id?: string
  index: number
  round: number
  name: string
  status: WorkspaceToolCallStatus
  arguments?: string
  result?: string
  result_data?: unknown
  metadata?: ToolResultMetadata
  error?: string
}

export interface WorkspaceStreamToolCallDeltaUpdate {
  index: number
  round: number
  id?: string
  name?: string
  delta: string
}

export interface WorkspaceStreamToolCallsDeltaPayload {
  updates: WorkspaceStreamToolCallDeltaUpdate[]
}

export interface WorkspaceStreamContentPayload {
  content: string
}

export interface WorkspaceStreamThinkingPayload {
  content: string
}

export interface WorkspaceModelContextPlan {
  protocol_version: string
  session_id: string
  round: number
  mode_code?: string
  role: WorkspaceModelContextRole
  execution: WorkspaceModelContextExecution
  messages: WorkspaceModelContextMessages
  handoff?: WorkspaceModelContextHandoff | null
  docs: WorkspaceModelContextDocs
  tools: WorkspaceModelContextTools
  cache_plan: WorkspaceModelContextCachePlan
  llm?: WorkspaceModelContextLLM | null
  budget?: WorkspaceModelContextBudget | null
}

export interface WorkspaceModelContextRole {
  id: string
  display_name?: string
  source?: string
  responsibility?: string
  handoff_required?: string[]
  allowed_tools?: string[]
  forbidden_tools?: string[]
  allowed_transitions?: string[]
}

export interface WorkspaceModelContextExecution {
  full_code_path: string
  directory_name?: string
  directory_code?: string
  directory_type?: string
  children_count: number
  files_count: number
  scope_policy: string
}

export interface WorkspaceModelContextMessages {
  context_policy: string
  model_context_anchor_message_id?: number
  parent_session_id?: string
  source_history_policy: string
  system_messages: number
  llm_messages: number
  total_stored_messages: number
  included_stored_messages: number
  excluded_stored_messages: number
  excluded_by_anchor: number
  excluded_display_only: number
  excluded_by_reduction?: number
  included?: WorkspaceModelContextMessageRef[]
  excluded?: WorkspaceModelContextMessageRef[]
  truncated?: boolean
}

export interface WorkspaceModelContextMessageRef {
  id: number
  role: string
  context_usage?: string
  artifact_kind?: string
  reason?: string
}

export interface WorkspaceModelContextHandoff {
  packet_version?: string
  source_session_id?: string
  source_role?: string
  target_role?: string
  artifact_kind?: string
  execute_directory?: string
  workspace_directory?: string
  target_app_directory?: string
  task_context?: string[]
  key_information?: string[]
  references?: string[]
  validation_status?: string
}

export interface WorkspaceModelContextDocs {
  document_package?: string[]
  required_docs?: string[]
  optional_docs?: string[]
  loaded_docs?: string[]
  missing_docs?: string[]
}

export interface WorkspaceModelContextTools {
  requested_names?: string[]
  llm_tools?: string[]
  llm_tool_count: number
  role_allowed_tools?: string[]
  role_allowed_tool_count?: number
  policy: string
}

export interface WorkspaceModelContextCachePlan {
  stable_prefix_strategy: string
  stable_prefix_items?: string[]
  actual_usage_field: string
  prompt_cache_key?: string
  prompt_cache_retention?: string
  result?: WorkspaceModelContextCacheResult
}

export interface WorkspaceModelContextCacheResult {
  status: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
  cache_hit_rate_percent: number
  cached_tokens_reported: boolean
  source: string
}

export interface WorkspaceModelContextLLM {
  config_id?: number
  config_name?: string
  provider?: string
  model?: string
  request_model?: string
  max_tokens?: number
  message_count: number
  tool_count: number
}

export interface WorkspaceModelContextBudget {
  reducer_level: number
  reducer_reason?: string
  estimated_input_tokens: number
  estimated_tool_tokens: number
  output_reserve_tokens: number
  estimated_total_tokens: number
  soft_limit_tokens: number
  tokens_until_soft_limit: number
  status: string
}

export interface WorkspaceStreamDonePayload {
  session_id: string
  tool_calls?: WorkspaceChatToolCallSummary[]
  llm_config_id?: number
  llm_config_name?: string
  llm_provider?: string
  llm_model?: string
  llm_usage?: LLMUsageInfo
  model_context_plan?: WorkspaceModelContextPlan
}

export interface WorkspaceStreamErrorPayload {
  message: string
}

export interface WorkspaceStreamPayloadMap {
  session: WorkspaceStreamSessionPayload
  model_context_plan: WorkspaceModelContextPlan
  tool_call: WorkspaceStreamToolCallPayload
  tool_calls_stream_delta: WorkspaceStreamToolCallsDeltaPayload
  thinking: WorkspaceStreamThinkingPayload
  content: WorkspaceStreamContentPayload
  done: WorkspaceStreamDonePayload
  error: WorkspaceStreamErrorPayload
}

export type WorkspaceStreamPayload = WorkspaceStreamPayloadMap[WorkspaceStreamEventName]

/** 创建阶段交接会话：旧会话保留展示，新会话只携带结构化产物进入目标阶段 */
export async function createWorkspaceHandoff(req: WorkspaceHandoffReq): Promise<WorkspaceHandoffResp> {
  return post<WorkspaceHandoffResp>('/agent/api/v1/workspace/sessions/handoff', req)
}

/** 流式事件回调：event 为 session|tool_call|tool_calls_stream_delta|content|done|error；返回 false 表示调用方已丢弃过期流事件。 */
export type WorkspaceChatStreamOnEvent = (event: WorkspaceStreamEventName, data: WorkspaceStreamPayload) => boolean | void

export interface WorkspaceChatStreamOptions {
  signal?: AbortSignal
}

function isWorkspaceStreamEventName(event: string): event is WorkspaceStreamEventName {
  return Object.values(workspaceStreamEvents).includes(event as WorkspaceStreamEventName)
}

function parseWorkspaceSSEBlock(block: string): { event: WorkspaceStreamEventName; data: WorkspaceStreamPayload } | null {
  let event = ''
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trim())
    }
  }
  if (!event || !isWorkspaceStreamEventName(event)) return null

  const dataStr = dataLines.join('\n').trim()
  if (!dataStr) return { event, data: {} as WorkspaceStreamPayload }
  try {
    const obj = JSON.parse(dataStr)
    return { event, data: (typeof obj === 'object' && obj != null ? obj : {}) as WorkspaceStreamPayload }
  } catch {
    return { event, data: { message: dataStr } as WorkspaceStreamPayload }
  }
}

/**
 * 工作台对话流式接口（SSE）：通过 onEvent 逐步接收 session、tool_call、content、done、error
 */
export async function workspaceChatStream(
  data: WorkspaceChatReq,
  onEvent: WorkspaceChatStreamOnEvent,
  options: WorkspaceChatStreamOptions = {}
): Promise<void> {
  const base = getApiBaseURL()
  const url = `${base}/agent/api/v1/workspace/chat/stream`
  const res = await authFetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
    signal: options.signal,
  })
  if (!res.ok) {
    const t = await res.text()
    let msg = `HTTP ${res.status}`
    try {
      const j = JSON.parse(t)
      if (j?.msg) msg = j.msg
    } catch {
      if (t) msg = t.slice(0, 200)
    }
    onEvent(workspaceStreamEvents.error, { message: msg })
    throw new Error(msg)
  }

  const contentType = res.headers.get('content-type') || ''
  if (contentType.includes('application/json')) {
    const j = await res.json().catch(() => null)
    const msg = j?.msg || j?.message || '工作台流式请求失败'
    onEvent(workspaceStreamEvents.error, { message: msg })
    throw new Error(msg)
  }

  const reader = res.body?.getReader()
  if (!reader) {
    onEvent(workspaceStreamEvents.error, { message: '响应体不可读' })
    throw new Error('响应体不可读')
  }
  const dec = new TextDecoder()
  let buf = ''
  for (;;) {
    if (options.signal?.aborted) {
      await reader.cancel().catch(() => undefined)
      throw new DOMException('The operation was aborted.', 'AbortError')
    }
    const { done, value } = await reader.read()
    if (done) break
    buf += dec.decode(value, { stream: true })
    buf = buf.replace(/\r\n/g, '\n')
    const parts = buf.split('\n\n')
    buf = parts.pop() || ''
    for (const block of parts) {
      const parsed = parseWorkspaceSSEBlock(block)
      if (parsed) onEvent(parsed.event, parsed.data)
    }
  }
  if (buf.trim()) {
    const parsed = parseWorkspaceSSEBlock(buf.replace(/\r\n/g, '\n'))
    if (parsed) onEvent(parsed.event, parsed.data)
  }
}

/**
 * 获取工作台会话列表
 */
export async function getWorkspaceSessions(params: ListWorkspaceSessionsReq): Promise<ListWorkspaceSessionsResp> {
  return get<ListWorkspaceSessionsResp>('/agent/api/v1/workspace/sessions', params)
}

/** 工作台消息信息 */
export interface WorkspaceMessageInfo {
  id: number
  session_id: string
  role: 'user' | 'assistant' | 'tool'
  user?: string
  created_by?: string
  content: string
  display_content?: string
  thinking_content?: string
  /** 用户消息附带的文件列表 JSON，解析后为 { files: WorkspaceChatMessageFile[] } */
  files?: string | null
  tool_calls?: WorkspaceChatToolCallSummary[]
  llm_config_id?: number
  llm_config_name?: string
  llm_provider?: string
  llm_model?: string
  llm_usage?: LLMUsageInfo
  model_context_plan?: WorkspaceModelContextPlan
  context_usage?: string
  artifact_kind?: string
  created_at: string
}

export interface LLMUsageInfo {
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
  cached_tokens_reported?: boolean
}

/** 获取工作台会话消息列表请求 */
export interface ListWorkspaceMessagesReq {
  session_id: string
}

/** 获取工作台会话消息列表响应 */
export interface ListWorkspaceMessagesResp {
  messages: WorkspaceMessageInfo[]
}

/** 工作台工具调用摘要 */
export interface WorkspaceChatToolCallSummary {
  id?: string        // tool_call_id（用于关联 tool 消息）
  index?: number     // 本轮内工具下标
  round?: number     // 模型工具调用轮次
  name: string       // 工具名称
  status: string     // ok / error
  arguments?: string // 参数（JSON 字符串，可选）
  result?: string    // 结果内容（从对应的 tool 消息中获取，可选）
  result_data?: unknown // 结构化结果（优先供前端提取文件/展示字段）
  metadata?: ToolResultMetadata // 工具结果元数据
  error?: string     // 错误信息（如果有，可选）
}

/**
 * 获取工作台会话消息列表
 */
export async function getWorkspaceMessages(params: ListWorkspaceMessagesReq): Promise<ListWorkspaceMessagesResp> {
  return get<ListWorkspaceMessagesResp>('/agent/api/v1/workspace/messages', params)
}

/** 查询当前用户所有正在执行的工作台任务 */
export async function getRunningSessions(): Promise<{ sessions: WorkspaceSessionItem[] }> {
  return get<{ sessions: WorkspaceSessionItem[] }>('/agent/api/v1/workspace/sessions/running')
}

/** 查询当前用户最近已结束的工作台任务 */
export async function getFinishedSessions(limit = 20): Promise<{ sessions: WorkspaceSessionItem[] }> {
  return get<{ sessions: WorkspaceSessionItem[] }>('/agent/api/v1/workspace/sessions/finished', { limit })
}

/** 清除会话的待确认/待测试状态 */
export async function resolveWorkspaceSessionInteraction(sessionId: string): Promise<void> {
  await post('/agent/api/v1/workspace/sessions/interaction/resolve', { session_id: sessionId })
}

/** 记录工作台交互卡片事件，仅用于审计展示，不进入模型上下文 */
export async function recordWorkspaceInteractionEvent(req: WorkspaceInteractionEventReq): Promise<void> {
  await post('/agent/api/v1/workspace/sessions/interaction/event', req)
}

/** 取消执行中的工作台任务 */
export async function cancelWorkspaceChat(sessionId: string): Promise<void> {
  await post('/agent/api/v1/workspace/chat/cancel', { session_id: sessionId })
}

/** 检查 session 的 SSE 连接是否存活（SSE 存活则不轮询大消息列表，节省带宽） */
export async function getWorkspaceSessionSSEStatus(sessionId: string): Promise<{ connected: boolean }> {
  const res = await get<{ connected?: boolean }>(
    `/agent/api/v1/workspace/sessions/${encodeURIComponent(sessionId)}/sse-status`
  )
  return { connected: !!res?.connected }
}

/** 批量获取内置工作台工具展示详情 */
export async function batchGetWorkspaceToolDetails(
  req: BatchWorkspaceToolDetailsReq
): Promise<BatchWorkspaceToolDetailsResp> {
  return post<BatchWorkspaceToolDetailsResp>('/agent/api/v1/workspace/tools/batch_detail', {
    names: req.names || [],
    include_schema: !!req.include_schema
  })
}
