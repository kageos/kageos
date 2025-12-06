/**
 * Hub API 客户端
 * 
 * Hub 是 AI-Agent-OS 的应用市场，提供应用发布、浏览、克隆等功能。
 */

import { getHubBaseURL, isHubEnabled } from '@/config/hub'
import { post, get } from '@/utils/request'

/**
 * 函数组源代码
 */
export interface PackageSourceCode {
  package: string           // 函数组路径，如：tools/cashier
  full_group_code: string   // 完整函数组代码，如：/luobei/demo/tools/cashier
  source_code: string       // Go 源代码内容
  functions?: FunctionInfo[] // 函数列表（功能列表）
}

/**
 * 发布应用到 Hub 请求
 */
export interface PublishHubAppReq {
  api_key?: string                    // API Key（私有化部署需要）
  source_user: string                 // 源用户
  source_app: string                  // 源应用
  packages: PackageSourceCode[]       // 函数组源代码列表
  name: string                        // 应用名称
  description: string                 // 应用描述
  category?: string                   // 分类
  tags?: string[]                     // 标签
  service_fee_personal?: number        // 个人用户服务费
  service_fee_enterprise?: number     // 企业用户服务费
  version: string                     // 版本号
}

/**
 * 发布应用到 Hub 响应
 */
export interface PublishHubAppResp {
  hub_app_id: number                  // Hub 应用 ID
  hub_app_url: string                 // Hub 应用 URL
}

/**
 * 发布应用到 Hub
 */
export async function publishHubApp(data: PublishHubAppReq): Promise<PublishHubAppResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  const baseURL = getHubBaseURL()
  const url = `${baseURL}/hub_apps/publish`

  return post<PublishHubAppResp>(url, data)
}

/**
 * Hub 函数组信息
 */
export interface HubFunctionGroupInfo {
  id: number
  full_group_code: string      // 完整函数组代码
  group_code: string           // 函数组代码
  group_name: string           // 函数组名称
  package: string              // 函数组路径
  full_path: string            // 完整路径
  version: string              // 版本号
  function_count: number       // 函数数量
}

/**
 * Hub 应用信息
 */
export interface HubAppInfo {
  id: number
  created_at: string
  updated_at: string
  name: string
  description: string
  category: string
  tags: string[]
  source_type: string          // 'saas' | 'private'
  source_url: string
  source_user: string
  source_app: string
  publisher_username: string
  published_at: string
  service_fee_personal: number
  service_fee_enterprise: number
  download_count: number
  trial_count: number
  rating: number
  version: string
  function_groups?: HubFunctionGroupInfo[]  // 函数组列表（仅在详情中返回）
}

/**
 * 获取 Hub 应用列表响应
 */
export interface HubAppListResp {
  items: HubAppInfo[]
  page: number
  page_size: number
  total: number
}

/**
 * 获取 Hub 应用列表
 */
export async function getHubAppList(params?: {
  page?: number
  page_size?: number
  search?: string
  category?: string
  publisher_username?: string
}): Promise<HubAppListResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  const baseURL = getHubBaseURL()
  const url = `${baseURL}/hub_apps`

  return get<HubAppListResp>(url, params || {})
}

/**
 * 获取 Hub 应用详情
 */
export async function getHubAppDetail(hubAppId: number): Promise<HubAppInfo> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  const baseURL = getHubBaseURL()
  const url = `${baseURL}/hub_apps/detail`

  return get<HubAppInfo>(url, { hub_app_id: hubAppId })
}

