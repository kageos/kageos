import { del, get, post, put } from '@/architecture/infrastructure/apiClient/request'

// ==================== LLM 相关 ====================

export interface LLMInfo {
  id: number
  code?: string
  name: string
  provider: string
  protocol: string
  model: string
  api_key?: string
  has_api_key: boolean
  api_base: string
  endpoint_path: string
  api_version: string
  auth_scheme: string
  headers: string
  timeout: number
  max_tokens: number
  extra_config: string
  capabilities: string
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
  code?: string
  name: string
  provider: string
  protocol: string
  model: string
  api_key?: string
  has_api_key: boolean
  api_base: string
  endpoint_path: string
  api_version: string
  auth_scheme: string
  headers: string
  timeout: number
  max_tokens: number
  extra_config: string
  capabilities: string
  is_default: boolean
  visibility: number
  admin: string
  is_admin: boolean
  created_at: string
  updated_at: string
}

export interface LLMGetDefaultResp {
  id: number
  code?: string
  name: string
  provider: string
  protocol: string
  model: string
  api_key?: string
  has_api_key: boolean
  api_base: string
  endpoint_path: string
  api_version: string
  auth_scheme: string
  headers: string
  timeout: number
  max_tokens: number
  extra_config: string
  capabilities: string
  is_default: boolean
  visibility: number
  admin: string
  is_admin: boolean
  created_at: string
  updated_at: string
}

export interface LLMCreateReq {
  name: string
  provider?: string
  protocol?: string
  model: string
  api_key?: string
  api_base?: string
  endpoint_path?: string
  api_version?: string
  auth_scheme?: string
  headers?: string
  timeout?: number
  max_tokens?: number
  extra_config?: string
  capabilities?: string
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
  provider?: string
  protocol?: string
  model: string
  api_key?: string
  api_base?: string
  endpoint_path?: string
  api_version?: string
  auth_scheme?: string
  headers?: string
  timeout?: number
  max_tokens?: number
  extra_config?: string
  capabilities?: string
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

export interface LLMProbeReq {
  id?: number
  provider?: string
  protocol?: string
  model?: string
  api_key?: string
  api_base?: string
  endpoint_path?: string
  api_version?: string
  auth_scheme?: string
  headers?: string
  extra_config?: string
  max_tokens?: number
  timeout?: number
}

export interface LLMProbeAttempt {
  provider: string
  protocol: string
  api_base: string
  ok: boolean
  error?: string
}

export interface LLMProbeResp {
  ok: boolean
  provider: string
  protocol: string
  api_base: string
  endpoint_path?: string
  api_version?: string
  auth_scheme?: string
  model?: string
  message?: string
  error?: string
  capabilities?: Record<string, boolean>
  attempts?: LLMProbeAttempt[]
}

/**
 * 获取LLM配置列表
 */
export function getLLMList(params: LLMListReq) {
	return get<LLMListResp>('/agent/api/v1/llm-configs', params)
}

/**
 * 获取LLM配置详情
 */
export function getLLM(params: LLMGetReq) {
	return get<LLMGetResp>(`/agent/api/v1/llm-configs/${params.id}`)
}

/**
 * 获取默认LLM配置
 */
export function getDefaultLLM() {
	return get<LLMGetDefaultResp>('/agent/api/v1/llm-configs/default')
}

/**
 * 创建LLM配置
 */
export function createLLM(data: LLMCreateReq) {
	return post<LLMCreateResp>('/agent/api/v1/llm-configs', data)
}

/**
 * 更新LLM配置
 */
export function updateLLM(data: LLMUpdateReq) {
	return put<LLMUpdateResp>(`/agent/api/v1/llm-configs/${data.id}`, data)
}

/**
 * 检测LLM协议和密钥可用性
 */
export function probeLLM(data: LLMProbeReq) {
	return post<LLMProbeResp>('/agent/api/v1/llm-configs/probe', data)
}

/**
 * 删除LLM配置
 */
export function deleteLLM(params: LLMDeleteReq) {
	return del(`/agent/api/v1/llm-configs/${params.id}`)
}

/**
 * 设置默认LLM配置
 */
export function setDefaultLLM(params: LLMSetDefaultReq) {
	return put(`/agent/api/v1/llm-configs/${params.id}/default`)
}
