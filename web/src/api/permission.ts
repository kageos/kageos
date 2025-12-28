/**
 * 权限管理 API
 */

import { get, post } from '@/utils/request'

/**
 * 权限申请请求
 */
export interface PermissionApplyReq {
  resource_path: string  // 资源路径
  action?: string  // 权限点（如 table:search，可选，如果提供了 actions 则忽略）
  actions?: string[]  // 权限点列表（可选，如果提供则批量申请）
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

/**
 * 获取工作空间权限请求参数
 */
export interface GetWorkspacePermissionsReq {
  user: string  // 用户名
  app: string   // 应用名
}

/**
 * 获取工作空间权限响应
 */
export interface GetWorkspacePermissionsResp {
  user: string  // 用户名
  app: string   // 应用名
  permissions: Record<string, Record<string, boolean>>  // 权限结果（full_code_path -> action -> hasPermission）
}

/**
 * 获取工作空间的所有权限
 * 用于权限申请页面显示已有权限
 */
export function getWorkspacePermissions(params: GetWorkspacePermissionsReq): Promise<GetWorkspacePermissionsResp> {
  return get<GetWorkspacePermissionsResp>('/workspace/api/v1/permission/workspace', params)
}

