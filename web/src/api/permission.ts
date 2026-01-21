/**
 * 权限管理 API
 */

import { get, post } from '@/utils/request'

/**
 * 权限申请请求
 */
export interface PermissionApplyReq {
  resource_path: string  // 资源路径
  role_id: number  // 角色ID（必填）
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
  role_id: number // ⭐ 角色ID
  role_name: string // ⭐ 角色名称
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

/**
 * 查询资源的所有权限分配请求参数
 */
export interface GetResourcePermissionsReq {
  user: string  // 租户用户（必填）
  app: string   // 应用代码（必填）
  resource_path: string  // 资源路径（full-code-path，必填）
}

/**
 * 资源权限分配信息
 */
export interface ResourcePermissionAssignment {
  id: number  // 分配ID
  subject_type: 'user' | 'department'  // 权限主体类型
  subject: string  // 权限主体：用户名或组织架构路径
  subject_name: string  // 权限主体名称（用户昵称或部门名称）
  role_id: number  // 角色ID
  role_code: string  // 角色编码
  role_name: string  // 角色名称
  resource_path: string  // 资源路径
  resource_name: string  // 资源名称（节点名称）
  start_time: string  // 生效开始时间（ISO 8601 格式字符串）
  end_time?: string  // 生效结束时间（可选，ISO 8601 格式字符串，nil 表示永久）
  created_by?: string  // 创建者
  created_at: string  // 创建时间（ISO 8601 格式字符串）
}

/**
 * 查询资源的所有权限分配响应
 */
export interface GetResourcePermissionsResp {
  assignments: ResourcePermissionAssignment[]  // 权限分配列表
  total: number  // 总数
}

/**
 * 查询资源的所有权限分配
 * 用于权限管理 Tab 显示资源的所有权限列表
 */
export function getResourcePermissions(params: GetResourcePermissionsReq): Promise<GetResourcePermissionsResp> {
  return get<GetResourcePermissionsResp>('/workspace/api/v1/permission/resource', params)
}

