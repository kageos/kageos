import { get, post, put, del } from '@/utils/request'
import type { FieldValue } from '@/core/types/field'

/**
 * 快链相关 API
 */

// 快链数据接口
export interface QuickLink {
  id: number
  name: string
  function_router: string
  function_method: string
  template_type: 'form' | 'table' | 'chart'
  request_params: Record<string, FieldValue>
  field_metadata?: Record<string, {
    editable?: boolean
    readonly?: boolean
    hint?: string
    highlight?: boolean
  }>
  metadata?: {
    table_state?: {
      page?: number
      page_size?: number
      sorts?: any[]
      search_params?: Record<string, any>
    }
    chart_filters?: Record<string, any>
    response_params?: Record<string, FieldValue>
  }
  created_at: string
  updated_at: string
  created_by: string
}

// 创建快链请求
export interface CreateQuickLinkReq {
  name: string
  function_router: string
  function_method: string
  template_type: 'form' | 'table' | 'chart'
  request_params: Record<string, FieldValue>
  field_metadata?: Record<string, {
    editable?: boolean
    readonly?: boolean
    hint?: string
    highlight?: boolean
  }>
  metadata?: {
    table_state?: {
      page?: number
      page_size?: number
      sorts?: any[]
      search_params?: Record<string, any>
    }
    chart_filters?: Record<string, any>
    response_params?: Record<string, FieldValue>
  }
}

// 创建快链响应
export interface CreateQuickLinkResp {
  id: number
  name: string
  url: string
}

// 获取快链响应
export interface GetQuickLinkResp {
  id: number
  name: string
  function_router: string
  function_method: string
  template_type: 'form' | 'table' | 'chart'
  request_params: Record<string, FieldValue>
  field_metadata?: Record<string, {
    editable?: boolean
    readonly?: boolean
    hint?: string
    highlight?: boolean
  }>
  metadata?: {
    table_state?: {
      page?: number
      page_size?: number
      sorts?: any[]
      search_params?: Record<string, any>
    }
    chart_filters?: Record<string, any>
    response_params?: Record<string, FieldValue>
  }
  created_at: string
  updated_at: string
  created_by: string
}

// 快链列表请求
export interface ListQuickLinksReq {
  function_router?: string
  page?: number
  page_size?: number
}

// 快链列表项
export interface QuickLinkItem {
  id: number
  name: string
  function_router: string
  function_method: string
  template_type: 'form' | 'table' | 'chart'
  created_at: string
  updated_at: string
}

// 快链列表响应
export interface ListQuickLinksResp {
  list: QuickLinkItem[]
  total: number
}

// 更新快链请求
export interface UpdateQuickLinkReq {
  name?: string
  request_params?: Record<string, FieldValue>
  field_metadata?: Record<string, {
    editable?: boolean
    readonly?: boolean
    hint?: string
    highlight?: boolean
  }>
  metadata?: {
    table_state?: {
      page?: number
      page_size?: number
      sorts?: any[]
      search_params?: Record<string, any>
    }
    chart_filters?: Record<string, any>
    response_params?: Record<string, FieldValue>
  }
}

// 更新快链响应
export interface UpdateQuickLinkResp {
  id: number
  name: string
}

/**
 * 创建快链
 */
export function createQuickLink(data: CreateQuickLinkReq) {
  return post<CreateQuickLinkResp>('/workspace/api/v1/quicklink/create', data)
}

/**
 * 获取快链（公开访问，不验证用户）
 */
export function getQuickLink(id: number) {
  return get<GetQuickLinkResp>('/workspace/api/v1/quicklink/get', { id })
}

/**
 * 获取快链列表
 */
export function listQuickLinks(params?: ListQuickLinksReq) {
  return get<ListQuickLinksResp>('/workspace/api/v1/quicklink/list', params)
}

/**
 * 更新快链
 */
export function updateQuickLink(id: number, data: UpdateQuickLinkReq) {
  return put<UpdateQuickLinkResp>(`/workspace/api/v1/quicklink/${id}`, data)
}

/**
 * 删除快链
 */
export function deleteQuickLink(id: number) {
  return del(`/workspace/api/v1/quicklink/${id}`)
}

