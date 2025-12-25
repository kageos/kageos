/**
 * 权限管理 API
 */

import { get, post } from '@/utils/request'

/**
 * 权限申请请求
 */
export interface PermissionApplyReq {
  resource_path: string  // 资源路径
  action: string  // 权限点（如 table:search）
  reason: string  // 申请理由
}

/**
 * 权限申请响应
 */
export interface PermissionApplyResp {
  id: string  // 申请ID
  status: string  // 申请状态（pending、approved、rejected）
  message: string  // 响应消息
}

/**
 * 提交权限申请
 */
export function applyPermission(data: PermissionApplyReq): Promise<PermissionApplyResp> {
  return post<PermissionApplyResp>('/workspace/api/v1/permission/apply', data)
}

/**
 * 获取权限申请列表
 */
export function getPermissionApplications(params?: {
  username?: string
  status?: string
  page?: number
  page_size?: number
}): Promise<any> {
  return get('/workspace/api/v1/permission/apply/list', { params })
}

/**
 * 获取权限申请详情
 */
export function getPermissionApplication(id: string): Promise<any> {
  return get(`/workspace/api/v1/permission/apply/${id}`)
}

