/**
 * Hub API 客户端
 *
 * Hub 是 AI-Agent-OS 的应用市场，提供目录发布、浏览、克隆等功能。
 */

import { getHubBaseURL, isHubEnabled } from '@/config/hub'
import { post, get } from '@/utils/request'

/**
 * 发布目录到 Hub 请求
 */
export interface PublishDirectoryToHubReq {
  api_key?: string                    // API Key（私有化部署需要）
  source_user: string                 // 源用户
  source_app: string                  // 源应用
  source_directory_path: string       // 源目录完整路径，如：/user/app/plugins/pdf
  name: string                        // 目录名称
  description?: string                 // 目录描述
  category?: string                   // 分类
  tags?: string[]                     // 标签
  service_fee_personal?: number        // 个人用户服务费
  service_fee_enterprise?: number     // 企业用户服务费
}

/**
 * 发布目录到 Hub 响应
 */
export interface PublishDirectoryToHubResp {
  hub_directory_id: number            // Hub 目录 ID
  hub_directory_url: string           // Hub 目录 URL
  directory_count: number             // 包含的子目录数量
  file_count: number                  // 包含的文件数量
}

/**
 * 发布目录到 Hub
 */
export async function publishDirectoryToHub(
  data: PublishDirectoryToHubReq
): Promise<PublishDirectoryToHubResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  return post<PublishDirectoryToHubResp>(
    '/workspace/api/v1/service_tree/publish_to_hub',
    data
  )
}

/**
 * 推送目录到 Hub 请求（更新已发布的目录）
 */
export interface PushDirectoryToHubReq {
  source_user: string                 // 源用户
  source_app: string                  // 源应用
  source_directory_path: string       // 源目录完整路径
  name?: string                       // 目录名称（可选）
  description?: string                // 目录描述（可选）
  category?: string                   // 分类（可选）
  tags?: string[]                     // 标签（可选）
  service_fee_personal?: number       // 个人用户服务费（可选）
  service_fee_enterprise?: number     // 企业用户服务费（可选）
  version: string                     // 新版本号（必需）
  api_key?: string                    // API Key（私有化部署需要）
}

/**
 * 推送目录到 Hub 响应
 */
export interface PushDirectoryToHubResp {
  hub_directory_id: number            // Hub 目录 ID
  hub_directory_url: string           // Hub 目录 URL
  directory_count: number             // 包含的子目录数量
  file_count: number                  // 包含的文件数量
  old_version: string                 // 旧版本号
  new_version: string                 // 新版本号
}

/**
 * 推送目录到 Hub（更新已发布的目录）
 */
export async function pushDirectoryToHub(
  data: PushDirectoryToHubReq
): Promise<PushDirectoryToHubResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  return post<PushDirectoryToHubResp>(
    '/workspace/api/v1/service_tree/push_to_hub',
    data
  )
}

/**
 * 从 Hub 拉取目录请求
 */
export interface PullDirectoryFromHubReq {
  hub_link: string                     // Hub 链接，格式：hub://host/full_code_path@version
  target_user: string                  // 目标用户
  target_app: string                   // 目标应用
  target_directory_path?: string      // 目标目录路径（可选，默认为应用根目录）
}

/**
 * 从 Hub 拉取目录响应
 */
export interface PullDirectoryFromHubResp {
  message: string                      // 成功消息
  directory_count: number              // 安装的目录数量
  file_count: number                   // 安装的文件数量
  target_directory_path: string        // 目标目录路径
  service_tree_id: number               // 根目录的 ServiceTree ID
  hub_directory_id: number              // Hub 目录 ID
  hub_directory_name: string           // Hub 目录名称
  hub_version: string                  // Hub 目录版本
  hub_version_num: number              // Hub 目录版本号（数字部分）
}

/**
 * 从 Hub 拉取目录
 */
export async function pullDirectoryFromHub(
  data: PullDirectoryFromHubReq
): Promise<PullDirectoryFromHubResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  return post<PullDirectoryFromHubResp>(
    '/workspace/api/v1/service_tree/pull_from_hub',
    data
  )
}

