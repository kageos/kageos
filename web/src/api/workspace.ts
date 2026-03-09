import { useAuthStore } from '@/stores/auth'
import axiosInstance from '@/utils/request'

/** 工作台消息中上传文件：与后端 sdk/agent-app/types.Files 对齐，供后端注入到 <files> 并供大模型拼到 run_form_submit 的 body */
export interface WorkspaceChatMessageFile {
  name: string
  source_name?: string
  storage?: string
  description?: string
  hash?: string
  size?: number
  upload_ts?: number
  local_path?: string
  is_uploaded?: boolean
  url: string
  server_url?: string
  upload_user?: string
}

export interface WorkspaceChatMessageFiles {
  files: WorkspaceChatMessageFile[]
  widget_type?: string
  data_type?: string
  remark?: string
  metadata?: Record<string, unknown>
}

/** 工作台对话请求（只认 LLM，单模式） */
export interface WorkspaceChatReq {
  full_code_path: string
  message: { content: string; files?: WorkspaceChatMessageFiles }
  session_id?: string
  /** LLM 配置 ID，0 表示使用默认 LLM */
  llm_config_id?: number
}

/** 工作台会话项 */
export interface WorkspaceSessionItem {
  session_id: string
  title: string
  agent_id?: number | null
  agent_name?: string
  status: string // active | generating | done | cancelled
  full_code_path?: string
  created_at: string
  updated_at: string
}

/** 获取工作台会话列表请求 */
export interface ListWorkspaceSessionsReq {
  full_code_path: string
  page?: number
  page_size?: number
}

/** 获取工作台会话列表响应 */
export interface ListWorkspaceSessionsResp {
  sessions: WorkspaceSessionItem[]
  total: number
  page: number
  page_size: number
}

/** 工作台模式项（列表/详情） */
export interface WorkspaceModeItem {
  id: number
  code: string
  name: string
  description?: string
  system_prompt_fragment?: string
  tool_names?: string[]
  agent_id?: number | null
  sort_order: number
  is_builtin: boolean
}

/** 工作台模式列表响应 */
export interface ListWorkspaceModesResp {
  list: WorkspaceModeItem[]
  total: number
  page: number
  page_size: number
}

/**
 * 获取工作台模式列表（供下拉选择或管理页）
 */
export async function getWorkspaceModes(params?: { page?: number; page_size?: number }): Promise<ListWorkspaceModesResp> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  const url = `${base}/agent/api/v1/workspace/modes`
  const res = await axiosInstance.get<ListWorkspaceModesResp>(url, { params: params || {} })
  return res as unknown as ListWorkspaceModesResp
}

/** 创建工作台模式请求 */
export interface CreateWorkspaceModeReq {
  code: string
  name: string
  description?: string
  system_prompt_fragment?: string
  tool_names?: string[]
  agent_id?: number | null
  sort_order?: number
}

/** 更新工作台模式请求 */
export interface UpdateWorkspaceModeReq {
  name?: string
  description?: string
  system_prompt_fragment?: string
  tool_names?: string[]
  agent_id?: number | null
  sort_order?: number
}

/** 获取工作台工具名列表响应（供模式配置多选） */
export interface ListWorkspaceToolNamesResp {
  names: string[]
}

/** 工作台工具定义（list_tools 返回） */
export interface WorkspaceToolDef {
  name: string
  description?: string
  input_schema?: Record<string, unknown>
  output_schema?: Record<string, unknown>
}

/** 工作台工具列表响应 */
export interface ListWorkspaceToolsResp {
  tools: WorkspaceToolDef[]
}

/** 获取工作台工具列表（完整定义，供管理页右侧展示） */
export async function getWorkspaceTools(): Promise<WorkspaceToolDef[]> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  const res = await axiosInstance.get<ListWorkspaceToolsResp>(`${base}/agent/api/v1/workspace/tools`)
  const o = res as unknown as ListWorkspaceToolsResp
  return o?.tools ?? []
}

/** 获取工作台模式详情（按 id） */
export async function getWorkspaceMode(id: number): Promise<WorkspaceModeItem> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  return axiosInstance.get<WorkspaceModeItem>(`${base}/agent/api/v1/workspace/modes/${id}`) as Promise<WorkspaceModeItem>
}

/** 按 code 获取工作台模式 */
export async function getWorkspaceModeByCode(code: string): Promise<WorkspaceModeItem> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  return axiosInstance.get<WorkspaceModeItem>(`${base}/agent/api/v1/workspace/modes/by-code`, { params: { code } }) as Promise<WorkspaceModeItem>
}

/** 创建工作台模式 */
export async function createWorkspaceMode(req: CreateWorkspaceModeReq): Promise<WorkspaceModeItem> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  return axiosInstance.post<WorkspaceModeItem>(`${base}/agent/api/v1/workspace/modes`, req) as Promise<WorkspaceModeItem>
}

/** 更新工作台模式 */
export async function updateWorkspaceMode(id: number, req: UpdateWorkspaceModeReq): Promise<WorkspaceModeItem> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  return axiosInstance.put<WorkspaceModeItem>(`${base}/agent/api/v1/workspace/modes/${id}`, req) as Promise<WorkspaceModeItem>
}

/** 删除工作台模式（内置不可删） */
export async function deleteWorkspaceMode(id: number): Promise<void> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  await axiosInstance.delete(`${base}/agent/api/v1/workspace/modes/${id}`)
}

/** 获取工作台工具名列表（供模式配置时多选） */
export async function listWorkspaceToolNames(): Promise<string[]> {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  const res = await axiosInstance.get<ListWorkspaceToolNamesResp>(`${base}/agent/api/v1/workspace/tools/names`)
  const o = res as unknown as ListWorkspaceToolNamesResp
  return o?.names ?? []
}

/** 流式事件回调：event 为 session|agent_id|tool_call|content|done|error，data 为对应负载 */
export type WorkspaceChatStreamOnEvent = (event: string, data: Record<string, unknown>) => void

/**
 * 工作台对话流式接口（SSE）：通过 onEvent 逐步接收 session、agent_id、tool_call、content、done、error
 */
export async function workspaceChatStream(
  data: WorkspaceChatReq,
  onEvent: WorkspaceChatStreamOnEvent
): Promise<void> {
  let token = localStorage.getItem('token') || ''
  try {
    const auth = useAuthStore()
    const t = auth.token as unknown
    token = (t && typeof t === 'object' && 'value' in (t as object))
      ? String((t as { value: string }).value)
      : String(t || token)
  } catch {
    /* use localStorage default */
  }
  const base = import.meta.env.VITE_API_BASE_URL || ''
  const url = `${base}/agent/api/v1/workspace/chat/stream`
  const res = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token': token,
    },
    body: JSON.stringify(data),
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
    onEvent('error', { message: msg })
    throw new Error(msg)
  }
  const reader = res.body?.getReader()
  if (!reader) {
    onEvent('error', { message: '响应体不可读' })
    throw new Error('响应体不可读')
  }
  const dec = new TextDecoder()
  let buf = ''
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buf += dec.decode(value, { stream: true })
    const parts = buf.split('\n\n')
    buf = parts.pop() || ''
    for (const block of parts) {
      let ev = ''
      let dataStr = ''
      for (const line of block.split('\n')) {
        if (line.startsWith('event:')) ev = line.slice(6).trim()
        else if (line.startsWith('data:')) dataStr = line.slice(5).trim()
      }
      if (!ev) continue
      try {
        const obj = dataStr ? JSON.parse(dataStr) : {}
        onEvent(ev, typeof obj === 'object' && obj != null ? obj : {})
      } catch {
        onEvent(ev, { raw: dataStr })
      }
    }
  }
  if (buf.trim()) {
    let ev = ''
    let dataStr = ''
    for (const line of buf.split('\n')) {
      if (line.startsWith('event:')) ev = line.slice(6).trim()
      else if (line.startsWith('data:')) dataStr = line.slice(5).trim()
    }
    if (ev) {
      try {
        const obj = dataStr ? JSON.parse(dataStr) : {}
        onEvent(ev, typeof obj === 'object' && obj != null ? obj : {})
      } catch {
        onEvent(ev, { raw: dataStr })
      }
    }
  }
}

/**
 * 获取工作台会话列表
 */
export async function getWorkspaceSessions(params: ListWorkspaceSessionsReq): Promise<ListWorkspaceSessionsResp> {
  return axiosInstance.get<ListWorkspaceSessionsResp>('/agent/api/v1/workspace/sessions', { params })
}

/** 工作台消息信息 */
export interface WorkspaceMessageInfo {
  id: number
  session_id: string
  agent_id: number
  role: 'user' | 'assistant' | 'tool'
  content: string
  /** 用户消息附带的文件列表 JSON，解析后为 { files: WorkspaceChatMessageFile[] } */
  files?: string | null
  tool_calls?: WorkspaceChatToolCallSummary[]
  created_at: string
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
  name: string       // 工具名称
  status: string     // ok / error
  arguments?: string // 参数（JSON 字符串，可选）
  result?: string    // 结果内容（从对应的 tool 消息中获取，可选）
  error?: string     // 错误信息（如果有，可选）
}

/**
 * 获取工作台会话消息列表
 */
export async function getWorkspaceMessages(params: ListWorkspaceMessagesReq): Promise<ListWorkspaceMessagesResp> {
  return axiosInstance.get<ListWorkspaceMessagesResp>('/agent/api/v1/workspace/messages', { params })
}

/** 查询当前用户所有正在执行的工作台任务 */
export async function getRunningSessions(): Promise<{ sessions: WorkspaceSessionItem[] }> {
  return axiosInstance.get('/agent/api/v1/workspace/sessions/running')
}

/** 查询当前用户最近已结束的工作台任务 */
export async function getFinishedSessions(limit = 20): Promise<{ sessions: WorkspaceSessionItem[] }> {
  return axiosInstance.get('/agent/api/v1/workspace/sessions/finished', { params: { limit } })
}

/** 取消执行中的工作台任务 */
export async function cancelWorkspaceChat(sessionId: string): Promise<void> {
  return axiosInstance.post('/agent/api/v1/workspace/chat/cancel', { session_id: sessionId })
}
