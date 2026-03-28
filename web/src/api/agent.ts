import { get, post } from '@/utils/request'

// ==================== LLM 相关 ====================

export interface LLMInfo {
  id: number
  name: string
  provider: string
  model: string
  api_key?: string
  has_api_key: boolean
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
  configs: LLMInfo[]
  total: number
  _metadata?: Record<string, any>
}

export interface LLMGetReq {
  id: number
}

export interface LLMGetResp {
  id: number
  name: string
  provider: string
  model: string
  api_key?: string
  has_api_key: boolean
  api_base: string
  timeout: number
  max_tokens: number
  extra_config: string
  use_thinking: boolean
  is_default: boolean
  visibility: number
  admin: string
  is_admin: boolean
  created_at: string
  updated_at: string
}

export interface LLMGetDefaultResp {
  id: number
  name: string
  provider: string
  model: string
  api_key?: string
  has_api_key: boolean
  api_base: string
  timeout: number
  max_tokens: number
  extra_config: string
  use_thinking: boolean
  is_default: boolean
  visibility: number
  admin: string
  is_admin: boolean
  created_at: string
  updated_at: string
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
  id: number
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
  id: number
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
