/**
 * Hub API 客户端
 * 
 * Hub 是 AI-Agent-OS 的应用市场，提供应用发布、浏览、克隆等功能。
 */

import { getHubBaseURL, isHubEnabled } from '@/config/hub'
import { get } from '@/utils/request'

/**
 * 发布应用到 Hub
 */
export interface PublishAppReq {
  api_key: string              // Hub API Key
  source_user: string          // 源应用用户
  source_app: string           // 源应用代码
  name: string                 // 应用名称
  description?: string         // 应用描述
  is_free?: boolean           // 是否免费
  is_open_source?: boolean    // 是否开源
  service_fee?: number        // 服务费
  support_clone?: boolean     // 是否支持克隆
}

export interface PublishAppResp {
  hub_app_id: number          // Hub 应用 ID
  hub_url: string             // Hub 应用 URL
  message: string             // 响应消息
}

/**
 * 发布应用到 Hub
 */
export async function publishApp(data: PublishAppReq): Promise<PublishAppResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  const baseURL = getHubBaseURL()
  const url = `${baseURL}/api/v1/apps/publish`

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to publish app' }))
    throw new Error(error.message || 'Failed to publish app')
  }

  return response.json()
}

/**
 * 获取应用列表
 */
export interface AppInfo {
  id: number
  name: string
  description: string
  source_user: string
  source_app: string
  is_free: boolean
  is_open_source: boolean
  service_fee?: number
  support_clone: boolean
  download_count: number
  publisher_username: string
  published_at: string
}

export interface AppListResp {
  apps: AppInfo[]
  total: number
  page: number
  page_size: number
}

/**
 * 获取应用列表
 */
export async function getAppList(params?: {
  api_key?: string
  page?: number
  page_size?: number
  category?: string
  keyword?: string
}): Promise<AppListResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  const baseURL = getHubBaseURL()
  const url = `${baseURL}/api/v1/apps`

  const queryParams = new URLSearchParams()
  if (params?.api_key) {
    queryParams.append('api_key', params.api_key)
  }
  if (params?.page) {
    queryParams.append('page', params.page.toString())
  }
  if (params?.page_size) {
    queryParams.append('page_size', params.page_size.toString())
  }
  if (params?.category) {
    queryParams.append('category', params.category)
  }
  if (params?.keyword) {
    queryParams.append('keyword', params.keyword)
  }

  const fullURL = queryParams.toString() ? `${url}?${queryParams.toString()}` : url

  const response = await fetch(fullURL, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to get app list' }))
    throw new Error(error.message || 'Failed to get app list')
  }

  return response.json()
}

/**
 * 获取应用详情
 */
export interface AppDetailResp extends AppInfo {
  // 可以扩展更多字段
}

/**
 * 获取应用详情
 */
export async function getAppDetail(appId: number, apiKey?: string): Promise<AppDetailResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  const baseURL = getHubBaseURL()
  const url = `${baseURL}/api/v1/apps/${appId}`

  const queryParams = new URLSearchParams()
  if (apiKey) {
    queryParams.append('api_key', apiKey)
  }

  const fullURL = queryParams.toString() ? `${url}?${queryParams.toString()}` : url

  const response = await fetch(fullURL, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to get app detail' }))
    throw new Error(error.message || 'Failed to get app detail')
  }

  return response.json()
}

