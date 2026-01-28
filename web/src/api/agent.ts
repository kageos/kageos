import { get, post } from '@/utils/request'
import axiosInstance from '@/utils/request'

// ==================== 智能体相关 ====================

export interface LLMConfigInfo {
  id: number
  name: string
  provider: string
  model: string
  is_default: boolean
}

export interface AgentInfo {
  id: number
  name: string
  agent_type: 'knowledge_only' | 'plugin'
  chat_type: string
  enabled: boolean
  description: string
  system_prompt_template?: string // System Prompt模板，支持{knowledge}变量
  timeout: number
  plugin_function_path?: string // 插件函数路径（full-code-path，仅 plugin 类型需要）
  docs_paths?: string // 文档路径（逗号分隔）
  llm_config_id: number // LLM配置ID，如果为0则使用默认LLM
  llm_config?: LLMConfigInfo // 预加载的LLM配置信息
  metadata: string
  logo?: string // 智能体 Logo URL（可选）
  greeting?: string // 开场白内容（可选）
  greeting_type?: 'text' | 'md' | 'html' // 开场白格式类型：text, md, html
  generation_count: number // 生成次数统计
  visibility: number // 0: 公开, 1: 私有
  admin: string // 管理员列表（逗号分隔）
  is_admin: boolean // 当前用户是否是管理员
  created_at: string
  updated_at: string
}

export interface AgentListReq {
  agent_type?: 'knowledge_only' | 'plugins'
  enabled?: boolean
  docs_paths?: string // 按文档路径过滤（可选）
  llm_config_id?: number // 按LLM配置ID过滤（可选，0表示默认LLM）
  plugin_function_path?: string // 按插件函数路径过滤（可选）
  scope?: 'mine' | 'market' // mine: 我的, market: 市场
  page: number
  page_size: number
}

export interface AgentListResp {
  code: number
  data: {
    agents: AgentInfo[]
    total: number
  }
  msg: string
}

export interface AgentGetReq {
  id: number
}

export interface AgentGetResp {
  code: number
  data: AgentInfo
  msg: string
}

export interface AgentCreateReq {
  name: string
  agent_type: 'knowledge_only' | 'plugin'
  chat_type: string
  description?: string
  system_prompt_template?: string // System Prompt模板，支持{knowledge}变量
  timeout?: number
  plugin_function_path?: string // 插件函数路径（full-code-path，仅 plugin 类型需要）
  docs_paths?: string // 文档路径（逗号分隔）
  llm_config_id?: number // LLM配置ID，如果为0或不提供则使用默认LLM
  metadata?: string
  greeting?: string // 开场白内容（可选）
  greeting_type?: 'text' | 'md' | 'html' // 开场白格式类型：text, md, html（默认text）
  visibility?: number // 0: 公开, 1: 私有（默认0）
  admin?: string // 管理员列表（逗号分隔，默认创建用户）
}

export interface AgentCreateResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface AgentUpdateReq {
  id: number
  name: string
  agent_type: 'knowledge_only' | 'plugin'
  chat_type: string
  description?: string
  timeout?: number
  plugin_function_path?: string // 插件函数路径（full-code-path，仅 plugin 类型需要）
  docs_paths?: string // 文档路径（逗号分隔）
  llm_config_id?: number // LLM配置ID，如果为0或不提供则使用默认LLM
  metadata?: string
  greeting?: string // 开场白内容（可选）
  greeting_type?: 'text' | 'md' | 'html' // 开场白格式类型：text, md, html（默认text）
  visibility?: number // 0: 公开, 1: 私有
  admin?: string // 管理员列表（逗号分隔）
}

export interface AgentUpdateResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface AgentDeleteReq {
  id: number
}

export interface AgentEnableReq {
  id: number
}

export interface AgentDisableReq {
  id: number
}

/**
 * 获取智能体列表
 */
export function getAgentList(params: AgentListReq) {
  return get<AgentListResp>('/agent/api/v1/agents/list', params)
}

/**
 * 获取智能体详情
 */
export function getAgent(params: AgentGetReq) {
  return get<AgentGetResp>('/agent/api/v1/agents/get', params)
}

/**
 * 创建智能体
 */
export function createAgent(data: AgentCreateReq) {
  return post<AgentCreateResp>('/agent/api/v1/agents/create', data)
}

/**
 * 更新智能体
 */
export function updateAgent(data: AgentUpdateReq) {
  return post<AgentUpdateResp>('/agent/api/v1/agents/update', data)
}

/**
 * 删除智能体
 */
export function deleteAgent(params: AgentDeleteReq) {
  return post('/agent/api/v1/agents/delete', params)
}

/**
 * 启用智能体
 */
export function enableAgent(params: AgentEnableReq) {
  return post('/agent/api/v1/agents/enable', params)
}

/**
 * 禁用智能体
 */
export function disableAgent(params: AgentDisableReq) {
  return post('/agent/api/v1/agents/disable', params)
}

// ==================== Agent Chat 相关 ====================

export interface AgentChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export interface AgentChatReq {
  agent_id: number
  messages: AgentChatMessage[]
}

export interface AgentChatUsage {
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

export interface AgentChatResp {
  content: string
  usage?: AgentChatUsage
}

// FunctionGenAgentChatReq 相关接口
export interface FunctionGenChatMessage {
  content: string
  files?: Array<{
    url: string
    remark: string
  }>
}

export interface FunctionGenAgentChatReq {
  agent_id: number
  full_code_path: string
  package?: string // Package 名称
  session_id?: string
  existing_files?: string[] // 当前 package 下已存在的文件名（不含 .go 后缀）
  message: FunctionGenChatMessage
}

export interface FunctionGenAgentChatResp {
  session_id: string
  content: string
  record_id?: number
  status: string // generating/completed/failed
  can_continue: boolean // 是否可以继续输入（true: 可以继续输入, false: 不能再输入）
  usage?: AgentChatUsage
}

/**
 * 智能体聊天 - 函数生成类型（设置 600 秒超时时间）
 */
export function functionGenChat(data: FunctionGenAgentChatReq) {
  return axiosInstance.post<FunctionGenAgentChatResp>('/agent/api/v1/chat/function_gen', data, {
    timeout: 600000 // 600 秒
  })
}

/**
 * @deprecated 使用 functionGenChat 代替
 */
export function agentChat(data: AgentChatReq) {
  return axiosInstance.post<AgentChatResp>('/agent/api/v1/chat/function_gen', data, {
    timeout: 600000
  })
}

// ==================== 会话和消息相关 ====================

export interface ChatSessionInfo {
  id: number
  tree_id: number
  session_id: string
  agent_id: number // 关联的智能体ID
  agent?: AgentInfo // 关联的智能体信息（可选）
  title: string
  user: string
  created_at: string
  updated_at: string
}

export interface ChatSessionListReq {
  full_code_path: string
  page: number
  page_size: number
}

export interface ChatSessionListResp {
  sessions: ChatSessionInfo[]
  total: number
}

/**
 * 获取会话列表
 */
export function getChatSessionList(params: ChatSessionListReq) {
  return axiosInstance.get<ChatSessionListResp>('/agent/api/v1/chat/sessions', {
    params
  })
}

export interface ChatMessageInfo {
  id: number
  session_id: string
  agent_id: number // 处理该消息的智能体ID
  role: 'user' | 'assistant' | 'system'
  content: string
  files?: string // JSON 字符串，格式：[{"url":"...","remark":"..."}]
  user: string
  created_at: string
}

export interface ChatMessageListReq {
  session_id: string
}

export interface ChatMessageListResp {
  messages: ChatMessageInfo[]
}

/**
 * 获取消息列表
 */
export function getChatMessageList(params: ChatMessageListReq) {
  return axiosInstance.get<ChatMessageListResp>('/agent/api/v1/chat/messages', {
    params
  })
}

// ==================== 代码生成状态查询相关 ====================

export interface FunctionGenStatusReq {
  record_id: number
}

export interface FunctionGenStatusResp {
  record_id: number
  status: 'generating' | 'completed' | 'failed'
  code?: string
  error_msg?: string
  full_code_paths?: string[]
  duration: number
  created_at: string
  updated_at: string
}

/**
 * 查询代码生成状态
 */
export function getFunctionGenStatus(params: FunctionGenStatusReq) {
  return axiosInstance.get<FunctionGenStatusResp>('/agent/api/v1/chat/function_gen/status', {
    params
  })
}

// ==================== LLM 相关 ====================

export interface LLMInfo {
  id: number
  name: string
  provider: string
  model: string
  api_base: string
  timeout: number
  max_tokens: number
  extra_config: string
  use_thinking: boolean // 是否使用思考模式（GLM特有功能）
  is_default: boolean
  visibility: number // 0: 公开, 1: 私有
  admin: string // 管理员列表（逗号分隔）
  is_admin: boolean // 当前用户是否是管理员
  created_at: string
  updated_at: string
}

export interface LLMListReq {
  scope?: 'mine' | 'market' // mine: 我的, market: 市场
  page: number
  page_size: number
}

export interface LLMListResp {
  code: number
  data: {
    configs: LLMInfo[]
    total: number
  }
  msg: string
}

export interface LLMGetReq {
  id: number
}

export interface LLMGetResp {
  code: number
  data: LLMInfo
  msg: string
}

export interface LLMGetDefaultResp {
  code: number
  data: LLMInfo
  msg: string
}

export interface LLMCreateReq {
  name: string
  provider: string
  model: string
  api_key?: string
  api_base?: string
  timeout?: number
  max_tokens?: number
  extra_config?: string
  use_thinking?: boolean // 是否使用思考模式（GLM特有功能）
  is_default?: boolean
  visibility?: number // 0: 公开, 1: 私有（默认0）
  admin?: string // 管理员列表（逗号分隔，默认创建用户）
}

export interface LLMCreateResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface LLMUpdateReq {
  id: number
  name: string
  provider: string
  model: string
  api_key?: string
  api_base?: string
  timeout?: number
  max_tokens?: number
  extra_config?: string
  use_thinking?: boolean // 是否使用思考模式（GLM特有功能）
  is_default?: boolean
  visibility?: number // 0: 公开, 1: 私有
  admin?: string // 管理员列表（逗号分隔）
}

export interface LLMUpdateResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface LLMDeleteReq {
  id: number
}

export interface LLMSetDefaultReq {
  id: number
}

/**
 * 获取LLM配置列表
 */
export function getLLMList(params: LLMListReq) {
  return get<LLMListResp>('/agent/api/v1/llm/list', params)
}

/**
 * 获取LLM配置详情
 */
export function getLLM(params: LLMGetReq) {
  return get<LLMGetResp>('/agent/api/v1/llm/get', params)
}

/**
 * 获取默认LLM配置
 */
export function getDefaultLLM() {
  return get<LLMGetDefaultResp>('/agent/api/v1/llm/get_default')
}

/**
 * 创建LLM配置
 */
export function createLLM(data: LLMCreateReq) {
  return post<LLMCreateResp>('/agent/api/v1/llm/create', data)
}

/**
 * 更新LLM配置
 */
export function updateLLM(data: LLMUpdateReq) {
  return post<LLMUpdateResp>('/agent/api/v1/llm/update', data)
}

/**
 * 删除LLM配置
 */
export function deleteLLM(params: LLMDeleteReq) {
  return post('/agent/api/v1/llm/delete', params)
}

/**
 * 设置默认LLM配置
 */
export function setDefaultLLM(params: LLMSetDefaultReq) {
  return post('/agent/api/v1/llm/set_default', params)
}


