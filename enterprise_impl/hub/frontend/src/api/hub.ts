/**
 * Hub API 客户端
 * 
 * Hub 是 AI-Agent-OS 的应用市场，提供目录发布、浏览、克隆等功能。
 */

import { getHubBaseURL } from '@/config/hub'
import { get, post, del } from '@/utils/request'

/** Hub 前端配置（主站前端地址，用于「试用」跳转） */
export interface HubConfigResp {
  main_site_url: string  // 主站前端地址（如 http://localhost:5173 或 http://125.122.96.207:8999），不是后端 API
}

/**
 * 获取 Hub 配置（主站前端地址）
 * 用于点击「试用」时跳转：主站前端 + /workspace + 目录 full_code_path
 */
export function getHubConfig(): Promise<HubConfigResp> {
  const baseURL = getHubBaseURL()
  return get<HubConfigResp>(`${baseURL}/config`)
}

/**
 * Hub 目录信息
 */
export interface HubDirectoryInfo {
  id: number
  created_at: string
  updated_at: string
  status?: string                 // active=在架；deleted=已下架（列表不展示，通过链接仍可访问）
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
  copy_url?: string                 // 复制链接，格式 hub://host/path@version，用于粘贴到工作空间
  star_count?: number               // 星星数（类似 GitHub star）
  has_starred?: boolean              // 当前用户是否已加星（仅详情返回，未登录或未加星为 false）
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

/** 费用筛选：全部 / 免费 / 收费 */
export type FeeTypeFilter = '' | 'free' | 'paid'

/**
 * 获取 Hub 目录列表
 */
/** 列表排序：latest=最新，hot=热门，stars=按星数，downloads=按复制数 */
export type OrderByFilter = 'latest' | 'hot' | 'stars' | 'downloads'

export async function getHubDirectoryList(params?: {
  page?: number
  page_size?: number
  search?: string
  category?: string
  publisher_username?: string
  /** 只看自己：true 时后端按当前用户过滤（需带 token/网关带 X-Request-User），未登录返回 401 */
  mine_only?: boolean
  fee_type?: FeeTypeFilter
  order_by?: OrderByFilter
}): Promise<HubDirectoryListResp> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories`
  const p = { ...params }
  if (p.fee_type === '') delete (p as Record<string, unknown>).fee_type
  if (p.mine_only === false) delete (p as Record<string, unknown>).mine_only
  return get<HubDirectoryListResp>(url, p || {})
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
  version_description?: string        // 当前查看版本的更新说明（推送时填的「本版本更新说明」）
  // ⭐ files 字段已移除，不再返回和展示文件
}

/**
 * 目录版本项（历史版本列表）
 */
export interface HubDirectoryVersionItem {
  version: string
  version_num: number
  snapshot_at: string
  is_current: boolean
  description?: string               // 本版本更新说明（可选）
  publisher_username?: string       // 该版本的上传人
}

export interface DownloadHubDirectoryBundleParams {
  hubDirectoryId?: number
  fullCodePath?: string
  version?: string
}

export interface DownloadHubDirectoryBundleResult {
  blob: Blob
  filename: string
}

function parseDownloadFileName(contentDisposition: string | null): string {
  if (!contentDisposition) {
    return 'hub-directory-bundle.json'
  }

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match?.[1]) {
    return decodeURIComponent(utf8Match[1])
  }

  const plainMatch = contentDisposition.match(/filename="([^"]+)"/i) || contentDisposition.match(/filename=([^;]+)/i)
  if (plainMatch?.[1]) {
    return plainMatch[1].trim()
  }

  return 'hub-directory-bundle.json'
}

async function parseDownloadError(response: Response): Promise<string> {
  try {
    const data = await response.json() as { msg?: string; message?: string }
    return data.msg || data.message || '导出失败'
  } catch {
    return '导出失败'
  }
}

/**
 * 获取 Hub 目录详情
 * @param version 可选，不传则返回最新版本
 */
/** 用 full_code_path 获取详情（推荐，用于详情页 URL 展示） */
export async function getHubDirectoryDetailByPath(
  fullCodePath: string,
  includeTree?: boolean,
  version?: string
): Promise<HubDirectoryDetail> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/detail`
  const params: Record<string, string | number | boolean> = {
    full_code_path: fullCodePath,
    include_tree: includeTree ?? false
  }
  if (version) params.version = version
  return get<HubDirectoryDetail>(url, params)
}

/** 用 hub_directory_id 获取详情（兼容旧链接） */
export async function getHubDirectoryDetail(
  hubDirectoryId: number,
  includeTree?: boolean,
  version?: string
): Promise<HubDirectoryDetail> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/detail`
  const params: Record<string, string | number | boolean> = {
    hub_directory_id: hubDirectoryId,
    include_tree: includeTree ?? false
  }
  if (version) params.version = version
  return get<HubDirectoryDetail>(url, params)
}

/**
 * 获取 Hub 目录版本列表（用于详情页右侧历史版本）
 */
/** 用 full_code_path 获取版本列表（推荐） */
export async function getHubDirectoryVersionsByPath(
  fullCodePath: string
): Promise<{ items: HubDirectoryVersionItem[] }> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/versions`
  return get<{ items: HubDirectoryVersionItem[] }>(url, {
    full_code_path: fullCodePath
  })
}

export async function getHubDirectoryVersions(
  hubDirectoryId: number
): Promise<{ items: HubDirectoryVersionItem[] }> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/versions`
  return get<{ items: HubDirectoryVersionItem[] }>(url, {
    hub_directory_id: hubDirectoryId
  })
}

export async function downloadHubDirectoryBundle(
  params: DownloadHubDirectoryBundleParams
): Promise<DownloadHubDirectoryBundleResult> {
  const baseURL = getHubBaseURL()
  const query = new URLSearchParams()
  if (params.hubDirectoryId) {
    query.set('hub_directory_id', String(params.hubDirectoryId))
  }
  if (params.fullCodePath) {
    query.set('full_code_path', params.fullCodePath)
  }
  if (params.version) {
    query.set('version', params.version)
  }

  const token = localStorage.getItem('token') || ''
  const response = await fetch(`${baseURL}/directories/export_bundle?${query.toString()}`, {
    headers: token.trim()
      ? { 'X-Token': token }
      : undefined
  })

  const contentDisposition = response.headers.get('content-disposition')
  if (!contentDisposition?.toLowerCase().includes('attachment')) {
    throw new Error(await parseDownloadError(response))
  }

  return {
    blob: await response.blob(),
    filename: parseDownloadFileName(contentDisposition)
  }
}

/** 为目录加星（需要登录） */
export async function starHubDirectory(hubDirectoryId: number): Promise<void> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/${hubDirectoryId}/star`
  await post(url, {})
}

/** 取消星星（需要登录） */
export async function unstarHubDirectory(hubDirectoryId: number): Promise<void> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/${hubDirectoryId}/star`
  await del(url)
}

/** 删除应用（软删除：只改状态，数据保留，通过链接仍可访问；仅发布者可操作，需要登录） */
export async function deleteHubDirectory(hubDirectoryId: number): Promise<void> {
  const baseURL = getHubBaseURL()
  const url = `${baseURL}/directories/${hubDirectoryId}`
  await del(url)
}

// ==================== Pub Key 管理 ====================

export interface PubKeyItem {
  id: number
  name: string
  key_prefix: string
  created_at: string
  last_used_at: string | null
}

export interface GeneratePubKeyResp {
  id: number
  name: string
  key: string
  key_prefix: string
}

export async function generatePubKey(name: string): Promise<GeneratePubKeyResp> {
  const baseURL = getHubBaseURL()
  return post<GeneratePubKeyResp>(`${baseURL}/pub_key/generate`, { name })
}

export async function listPubKeys(): Promise<PubKeyItem[]> {
  const baseURL = getHubBaseURL()
  return get<PubKeyItem[]>(`${baseURL}/pub_key/list`)
}

export async function deletePubKey(id: number): Promise<void> {
  const baseURL = getHubBaseURL()
  await del(`${baseURL}/pub_key/${id}`)
}
