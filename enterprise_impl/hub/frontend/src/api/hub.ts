/**
 * Hub API 客户端
 * 
 * Hub 是 AI-Agent-OS 的应用市场，提供目录发布、浏览、克隆等功能。
 */

import { getHubBaseURL } from '@/config/hub'
import { get } from '@/utils/request'

/**
 * Hub 目录信息
 */
export interface HubDirectoryInfo {
  id: number
  created_at: string
  updated_at: string
  name: string
  description: string
  category: string
  tags: string[]
  package_path: string              // 目录路径
  full_code_path: string            // 完整代码路径
  source_user: string
  source_app: string
  source_directory_path: string
  publisher_username: string
  published_at: string
  service_fee_personal: number
  service_fee_enterprise: number
  download_count: number
  trial_count: number
  rating: number
  version: string
  directory_count: number           // 子目录数量
  file_count: number                // 文件数量
  function_count: number            // 函数数量
}

/**
 * 获取 Hub 目录列表响应
 */
export interface HubDirectoryListResp {
  items: HubDirectoryInfo[]
  page: number
  page_size: number
  total: number
}

/**
 * 获取 Hub 目录列表
 */
export async function getHubDirectoryList(params?: {
  page?: number
  page_size?: number
  search?: string
  category?: string
  publisher_username?: string
}): Promise<HubDirectoryListResp> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories`

  return get<HubDirectoryListResp>(url, params || {})
}

/**
 * 目录文件信息
 */
export interface DirectoryFileInfo {
  file_name: string
  relative_path: string
  file_type: string
  file_size: number
}

/**
 * 函数信息
 */
export interface HubFunctionInfo {
  id: number
  name: string
  code: string
  full_code_path: string
  description: string
  template_type: string
  tags: string[]
  ref_id: number
  version: string
  version_num: number
}

/**
 * 目录树节点
 */
export interface DirectoryTreeNode {
  type: 'package' | 'function'  // 节点类型：package（目录）或 function（函数）
  name: string
  path: string
  // ⭐ files 字段已移除，不再返回和展示文件
  functions?: HubFunctionInfo[]  // 函数列表
  subdirectories: DirectoryTreeNode[]
}

/**
 * Hub 目录详情
 */
export interface HubDirectoryDetail extends HubDirectoryInfo {
  directory_tree?: DirectoryTreeNode  // 目录树结构（可选）
  // ⭐ files 字段已移除，不再返回和展示文件
}

/**
 * 获取 Hub 目录详情
 */
export async function getHubDirectoryDetail(
  hubDirectoryId: number,
  includeTree?: boolean
): Promise<HubDirectoryDetail> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/detail`

  return get<HubDirectoryDetail>(url, {
    hub_directory_id: hubDirectoryId,
    include_tree: includeTree || false
  })
}
