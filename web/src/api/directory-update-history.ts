import { get } from '@/utils/request'

// API 摘要信息
export interface ApiSummary {
  code: string
  name: string
  desc: string
  router: string
  method: string
  full_code_path: string
  template_type?: string // 模板类型（如 form、table、chart）
}

// 目录变更信息
export interface DirectoryChangeInfo {
  full_code_path: string
  dir_version: string
  dir_version_num: number
  app_version?: string
  app_version_num?: number
  added_apis: ApiSummary[]
  updated_apis: ApiSummary[]
  deleted_apis: ApiSummary[]
  added_count: number
  updated_count: number
  deleted_count: number
  summary: string
  requirement?: string // 变更需求（用户输入）
  change_description?: string // 变更描述（大模型输出）
  duration?: number // 变更耗时（毫秒）
  updated_by: string
  created_at: string
  directory_name?: string // 目录名称
  directory_desc?: string // 目录描述
}

// 应用版本更新信息
export interface AppVersionUpdateInfo {
  app_version: string
  directory_changes: DirectoryChangeInfo[]
}

// 获取应用版本更新历史响应（App视角）
export interface GetAppVersionUpdateHistoryResp {
  app_id: number
  app_version: string
  versions: AppVersionUpdateInfo[]
}

// 分页信息
export interface PaginatedInfo {
  current_page: number
  total_count: number
  total_pages: number
  page_size: number
}

// 获取目录更新历史响应（目录视角）
export interface GetDirectoryUpdateHistoryResp {
  app_id: number
  full_code_path: string
  directory_changes: DirectoryChangeInfo[]
  paginated: PaginatedInfo
}

// 获取应用版本更新历史（App视角）
// 如果 app_version 为空，返回所有版本
export function getAppVersionUpdateHistory(appId: number, appVersion?: string) {
  const params: Record<string, string | number> = { app_id: appId }
  if (appVersion) {
    params.app_version = appVersion
  }
  return get<GetAppVersionUpdateHistoryResp>('/workspace/api/v1/directory_update_history/app_version', params)
}

// 获取目录更新历史（目录视角）
export function getDirectoryUpdateHistory(
  appId: number,
  fullCodePath: string,
  page: number = 1,
  pageSize: number = 10
) {
  return get<GetDirectoryUpdateHistoryResp>('/workspace/api/v1/directory_update_history/directory', {
    app_id: appId,
    full_code_path: fullCodePath,
    page,
    page_size: pageSize
  })
}

