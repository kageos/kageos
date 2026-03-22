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
  remote_hub_url?: string             // 远程 Hub 地址（跨站发布）
  pub_key?: string                    // Pub Key（跨站发布认证）
}

/**
 * 发布目录到 Hub 响应
 */
export interface PublishDirectoryToHubResp {
  hub_full_code_path: string          // Hub 目录完整路径，前端用此拼详情 URL
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
 * version 不传时后端自动递增为 v{N+1}
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
  service_fee_enterprise?: number      // 企业用户服务费（可选）
  version?: string                    // 新版本号（可选，不传则自动递增）
  update_description?: string         // 本版本更新说明（可选，如：新增 xxx 功能）
  api_key?: string                    // API Key（私有化部署需要）
  remote_hub_url?: string             // 远程 Hub 地址（跨站发布）
  pub_key?: string                    // Pub Key（跨站发布认证）
}

/**
 * 获取推送表单信息响应（用于推送对话框预填 + 显示下一版本号）
 */
export interface GetHubPushFormInfoResp {
  name: string
  description: string
  category: string
  tags: string[]
  service_fee_personal: number
  service_fee_enterprise: number
  current_version: string             // 当前已发布版本（如 v2）
  next_version: string                // 下一版本号（自动递增，如 v3）
}

/**
 * 获取推送表单信息（当前已发布信息 + 下一版本号，用于推送对话框预填）
 */
export async function getHubPushFormInfo(params: {
  source_user: string
  source_app: string
  source_directory_path: string
}): Promise<GetHubPushFormInfoResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }
  return get<GetHubPushFormInfoResp>(
    '/workspace/api/v1/service_tree/hub_push_form_info',
    params
  )
}

/**
 * 推送目录到 Hub 响应
 */
export interface PushDirectoryToHubResp {
  hub_full_code_path: string          // Hub 目录完整路径，前端用此拼详情 URL
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
  target_directory_path: string       // 目标目录路径
  service_tree_id: number              // 根目录的 ServiceTree ID
  hub_directory_name: string           // Hub 目录名称
  hub_version_num: number              // Hub 目录版本号（数字部分），展示时格式化为 v{N}
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

/** 离线安装包：directory_tree 与 Hub 详情/发布接口结构一致 */
export interface ImportHubDirectoryBundleReq {
  target_user: string
  target_app: string
  target_directory_path?: string
  directory_tree: Record<string, unknown>
  hub_full_code_path?: string
  hub_version_num?: number
  hub_directory_name?: string
}

/**
 * 从离线 JSON 安装目录（与 Hub 详情页「导出 JSON 安装包」格式兼容）
 */
export async function importHubDirectoryBundle(
  data: ImportHubDirectoryBundleReq
): Promise<PullDirectoryFromHubResp> {
  if (!isHubEnabled()) {
    throw new Error('Hub is disabled. Please set VITE_HUB_ENABLED=true')
  }

  return post<PullDirectoryFromHubResp>(
    '/workspace/api/v1/service_tree/import_hub_bundle',
    data
  )
}

