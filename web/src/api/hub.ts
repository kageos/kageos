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

