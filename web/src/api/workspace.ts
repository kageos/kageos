import { useAuthStore } from '@/stores/auth'
import axiosInstance from '@/utils/request'

/** 工作台对话请求 */
export interface WorkspaceChatReq {
  full_code_path: string
  message: { content: string; files?: unknown }
  session_id?: string
  agent_id?: number
}

/** 工作台会话项 */
export interface WorkspaceSessionItem {
  session_id: string
  title: string
  agent_id?: number | null
  agent_name?: string
  status: string
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
