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
  subject_type?: 'user' | 'department'  // 权限主体类型：user（用户）或 department（部门），可选，默认为 user
  subject?: string  // 权限主体：用户名或组织架构路径，可选，默认为当前用户
  reason?: string  // 申请理由（可选）
  end_time?: string  // 权限结束时间（ISO 8601 格式，可选，空字符串或 null 表示永久）
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
 * 获取权限申请列表（新接口）
 */
export interface GetPermissionRequestsReq {
  resource_path?: string  // 资源路径（可选）
  status?: string  // 状态：pending、approved、rejected
  page?: number
  page_size?: number
}

export interface PermissionRequestInfo {
  id: number
  app_id: number
  applicant_username: string
  subject_type: string
  subject: string
  resource_path: string
  resource_name: string // ⭐ 资源名称（中文）
  action: string
  start_time: string
  end_time?: string
  reason: string
  status: string
  approved_at?: string
  approved_by?: string
  rejected_at?: string
  rejected_by?: string
  reject_reason?: string
  created_at: string
  approvers: string[] // ⭐ 审批人列表
}

export interface GetPermissionRequestsResp {
  total: number
  page: number
  page_size: number
  records: PermissionRequestInfo[]
}

export function getPermissionRequests(params?: GetPermissionRequestsReq): Promise<GetPermissionRequestsResp> {
  return get<GetPermissionRequestsResp>('/workspace/api/v1/permission/requests', params)
}

/**
 * 审批通过权限申请
 */
export interface ApprovePermissionRequestReq {
  request_id: number
}

export function approvePermissionRequest(data: ApprovePermissionRequestReq): Promise<void> {
  return post<void>('/workspace/api/v1/permission/request/approve', data)
}

/**
 * 审批拒绝权限申请
 */
export interface RejectPermissionRequestReq {
  request_id: number
  reason: string
}

export function rejectPermissionRequest(data: RejectPermissionRequestReq): Promise<void> {
  return post<void>('/workspace/api/v1/permission/request/reject', data)
}

/**
 * 获取权限申请列表（旧接口，保留兼容）
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
  end_time?: string  // 权限结束时间（ISO 8601 格式，可选，空字符串或 null 表示永久）
}

/**
 * 添加权限（赋权给用户）
 */
export function addPermission(data: AddPermissionReq): Promise<void> {
  return post<void>('/workspace/api/v1/permission/add', data)
}

