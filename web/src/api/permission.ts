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
  reason?: string  // 申请理由（可选）
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
  user: string  // 工作空间所属用户（必填）
  app: string   // 工作空间应用代码（必填）
}

/**
 * 获取工作空间权限响应
 * ⭐ 直接返回原始权限记录，让前端处理
 */
export interface GetWorkspacePermissionsResp {
  records: Array<{  // 原始权限记录
    id: number
    user: string
    resource: string
    action: string
    app_id: number
  }>
}

/**
 * 获取工作空间的所有权限
 * 用于权限申请页面显示已有权限
 */
export function getWorkspacePermissions(params: GetWorkspacePermissionsReq): Promise<GetWorkspacePermissionsResp> {
  return get<GetWorkspacePermissionsResp>('/workspace/api/v1/permission/workspace', params)
}

/**
 * 添加权限请求（用于赋权）
 * ⭐ Subject 可以是用户名（如 "liubeiluo"）或组织架构路径（如 "/org/sub/qsearch"）
 */
export interface AddPermissionReq {
  subject: string  // 权限主体：用户名或组织架构路径
  resource_path: string  // 资源路径
  action: string  // 权限点
}

/**
 * 添加权限（赋权给用户）
 */
export function addPermission(data: AddPermissionReq): Promise<void> {
  return post<void>('/workspace/api/v1/permission/add', data)
}

