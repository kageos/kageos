import { get, post } from '@/utils/request'
import axiosInstance from '@/utils/request'

// ==================== 智能体相关 ====================

export interface KnowledgeBaseInfo {
  id: number
  name: string
  description: string
  status: string
  document_count: number
}

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
  msg_subject?: string // 消息主题，仅插件类型，自动生成
  nats_host?: string // NATS 服务器地址
  knowledge_base_id: number
  knowledge_base?: KnowledgeBaseInfo // 预加载的知识库信息
  llm_config_id: number // LLM配置ID，如果为0则使用默认LLM
  llm_config?: LLMConfigInfo // 预加载的LLM配置信息
  metadata: string
  created_at: string
  updated_at: string
}

export interface AgentListReq {
  agent_type?: 'knowledge_only' | 'plugins'
  enabled?: boolean
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
  knowledge_base_id: number
  llm_config_id?: number // LLM配置ID，如果为0或不提供则使用默认LLM
  metadata?: string
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
  knowledge_base_id: number
  llm_config_id?: number // LLM配置ID，如果为0或不提供则使用默认LLM
  metadata?: string
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
  return get<AgentListResp>('/api/v1/agent/agents/list', params)
}

/**
 * 获取智能体详情
 */
export function getAgent(params: AgentGetReq) {
  return get<AgentGetResp>('/api/v1/agent/agents/get', params)
}

/**
 * 创建智能体
 */
export function createAgent(data: AgentCreateReq) {
  return post<AgentCreateResp>('/api/v1/agent/agents/create', data)
}

/**
 * 更新智能体
 */
export function updateAgent(data: AgentUpdateReq) {
  return post<AgentUpdateResp>('/api/v1/agent/agents/update', data)
}

/**
 * 删除智能体
 */
export function deleteAgent(params: AgentDeleteReq) {
  return post('/api/v1/agent/agents/delete', params)
}

/**
 * 启用智能体
 */
export function enableAgent(params: AgentEnableReq) {
  return post('/api/v1/agent/agents/enable', params)
}

/**
 * 禁用智能体
 */
export function disableAgent(params: AgentDisableReq) {
  return post('/api/v1/agent/agents/disable', params)
}

// ==================== 知识库相关 ====================

export interface KnowledgeInfo {
  id: number
  name: string
  description: string
  status: string
  document_count: number
  content_hash: string
  user: string
  created_at: string
  updated_at: string
}

export interface KnowledgeListReq {
  page: number
  page_size: number
}

export interface KnowledgeListResp {
  code: number
  data: {
    knowledge_bases: KnowledgeInfo[]
    total: number
  }
  msg: string
}

export interface KnowledgeGetReq {
  id: number
}

export interface KnowledgeGetResp {
  code: number
  data: KnowledgeInfo
  msg: string
}

export interface KnowledgeCreateReq {
  name: string
  description?: string
  status?: string
}

export interface KnowledgeCreateResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface KnowledgeUpdateReq {
  id: number
  name: string
  description?: string
  status?: string
}

export interface KnowledgeUpdateResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface KnowledgeDeleteReq {
  id: number
}

export interface KnowledgeAddDocumentReq {
  knowledge_base_id: number
  parent_id?: number
  title: string
  content: string
  file_type?: string
  sort_order?: number
}

export interface KnowledgeAddDocumentResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface DocumentInfo {
  id: number
  knowledge_base_id: number
  parent_id: number
  doc_id: string
  title: string
  content: string
  file_type: string
  file_size: number
  status: string
  sort_order: number
  path: string
  user: string
  created_at: string
  updated_at: string
  children?: DocumentInfo[]
}

export interface KnowledgeListDocumentsReq {
  knowledge_base_id: number
  page: number
  page_size: number
}

export interface KnowledgeListDocumentsResp {
  code: number
  data: {
    documents: DocumentInfo[]
    total: number
  }
  msg: string
}

/**
 * 获取知识库列表
 */
export function getKnowledgeList(params: KnowledgeListReq) {
  return get<KnowledgeListResp>('/api/v1/agent/knowledge/list', params)
}

/**
 * 获取知识库详情
 */
export function getKnowledge(params: KnowledgeGetReq) {
  return get<KnowledgeGetResp>('/api/v1/agent/knowledge/get', params)
}

/**
 * 创建知识库
 */
export function createKnowledge(data: KnowledgeCreateReq) {
  return post<KnowledgeCreateResp>('/api/v1/agent/knowledge/create', data)
}

/**
 * 更新知识库
 */
export function updateKnowledge(data: KnowledgeUpdateReq) {
  return post<KnowledgeUpdateResp>('/api/v1/agent/knowledge/update', data)
}

/**
 * 删除知识库
 */
export function deleteKnowledge(params: KnowledgeDeleteReq) {
  return post('/api/v1/agent/knowledge/delete', { id: params.id })
}

/**
 * 添加文档
 */
export function addKnowledgeDocument(data: KnowledgeAddDocumentReq) {
  return post<KnowledgeAddDocumentResp>('/api/v1/agent/knowledge/add_document', data)
}

/**
 * 获取文档列表
 */
export function getKnowledgeDocuments(params: KnowledgeListDocumentsReq) {
  return get<KnowledgeListDocumentsResp>('/api/v1/agent/knowledge/list_documents', {
    knowledge_base_id: params.knowledge_base_id,
    page: params.page,
    page_size: params.page_size
  })
}

export interface KnowledgeGetDocumentReq {
  id: number
}

export interface KnowledgeGetDocumentResp {
  code: number
  data: DocumentInfo
  msg: string
}

export interface KnowledgeUpdateDocumentReq {
  id: number
  parent_id?: number
  title: string
  content: string
  file_type?: string
  status?: string
  sort_order?: number
}

export interface KnowledgeUpdateDocumentResp {
  code: number
  data: {
    id: number
  }
  msg: string
}

export interface KnowledgeDeleteDocumentReq {
  id: number
}

/**
 * 获取文档详情
 */
export function getKnowledgeDocument(params: KnowledgeGetDocumentReq) {
  return get<KnowledgeGetDocumentResp>('/api/v1/agent/knowledge/get_document', params)
}

/**
 * 更新文档
 */
export function updateKnowledgeDocument(data: KnowledgeUpdateDocumentReq) {
  return post<KnowledgeUpdateDocumentResp>('/api/v1/agent/knowledge/update_document', data)
}

/**
 * 删除文档
 */
export function deleteKnowledgeDocument(params: KnowledgeDeleteDocumentReq) {
  return post('/api/v1/agent/knowledge/delete_document', { id: params.id })
}

export interface KnowledgeGetDocumentsTreeReq {
  knowledge_base_id: number
}

export interface KnowledgeGetDocumentsTreeResp {
  code: number
  data: {
    documents: DocumentInfo[]
  }
  msg: string
}

/**
 * 获取知识库文档树（目录结构）
 */
export function getKnowledgeDocumentsTree(params: KnowledgeGetDocumentsTreeReq) {
  return get<KnowledgeGetDocumentsTreeResp>('/api/v1/agent/knowledge/get_documents_tree', {
    knowledge_base_id: params.knowledge_base_id
  })
}

export interface KnowledgeDocumentSortUpdate {
  id: number
  parent_id: number
  sort_order: number
  path: string
}

export interface KnowledgeUpdateDocumentsSortReq {
  knowledge_base_id: number
  updates: KnowledgeDocumentSortUpdate[]
}

/**
 * 批量更新文档排序
 */
export function updateKnowledgeDocumentsSort(data: KnowledgeUpdateDocumentsSortReq) {
  return post('/api/v1/agent/knowledge/update_documents_sort', data)
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
  tree_id: number
  session_id?: string
  message: FunctionGenChatMessage
}

export interface FunctionGenAgentChatResp {
  session_id: string
  content: string
  record_id?: number
  status: string
  usage?: AgentChatUsage
}

/**
 * 智能体聊天 - 函数生成类型（设置 600 秒超时时间）
 */
export function functionGenChat(data: FunctionGenAgentChatReq) {
  return axiosInstance.post<FunctionGenAgentChatResp>('/api/v1/agent/chat/function_gen', data, {
    timeout: 600000 // 600 秒
  })
}

/**
 * @deprecated 使用 functionGenChat 代替
 */
export function agentChat(data: AgentChatReq) {
  return axiosInstance.post<AgentChatResp>('/api/v1/agent/chat/function_gen', data, {
    timeout: 600000
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
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface LLMListReq {
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
  is_default?: boolean
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
  is_default?: boolean
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
  return get<LLMListResp>('/api/v1/agent/llm/list', params)
}

/**
 * 获取LLM配置详情
 */
export function getLLM(params: LLMGetReq) {
  return get<LLMGetResp>('/api/v1/agent/llm/get', params)
}

/**
 * 获取默认LLM配置
 */
export function getDefaultLLM() {
  return get<LLMGetDefaultResp>('/api/v1/agent/llm/get_default')
}

/**
 * 创建LLM配置
 */
export function createLLM(data: LLMCreateReq) {
  return post<LLMCreateResp>('/api/v1/agent/llm/create', data)
}

/**
 * 更新LLM配置
 */
export function updateLLM(data: LLMUpdateReq) {
  return post<LLMUpdateResp>('/api/v1/agent/llm/update', data)
}

/**
 * 删除LLM配置
 */
export function deleteLLM(params: LLMDeleteReq) {
  return post('/api/v1/agent/llm/delete', params)
}

/**
 * 设置默认LLM配置
 */
export function setDefaultLLM(params: LLMSetDefaultReq) {
  return post('/api/v1/agent/llm/set_default', params)
}

