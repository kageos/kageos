/**
 * 角色管理 API
 */

import { get, post, put, del } from '@/utils/request'

// ==================== 类型定义 ====================

/**
 * 角色信息
 */
export interface Role {
  id: number
  name: string
  code: string
  description: string
  is_system: boolean // 是否为系统角色（不可删除）
  created_at: string
  updated_at: string
  permissions?: RolePermission[] // 角色权限列表（可选）
}

/**
 * 角色权限（按资源类型分组）
 */
export interface RolePermission {
  id: number
  role_id: number
  resource_type: string // 资源类型（如 "directory", "table", "form", "chart"）
  action: string // 权限点（如 "directory:read", "function:write"）
}

/**
 * 创建角色请求
 */
export interface CreateRoleReq {
  name: string
  code: string
  description?: string
  permissions: Record<string, string[]> // 权限点列表，按资源类型分组，例如：{ "directory": ["directory:read", "directory:write"], "table": ["function:read", "function:write"] }
}

/**
 * 创建角色响应
 */
export interface CreateRoleResp {
  role: Role
}

/**
 * 更新角色请求
 */
export interface UpdateRoleReq {
  name?: string
  description?: string
  permissions?: Record<string, string[]> // 权限点列表，按资源类型分组
}

/**
 * 更新角色响应
 */
export interface UpdateRoleResp {
  role: Role
}

/**
 * 获取角色响应
 */
export interface GetRoleResp {
  role: Role
}

/**
 * 获取角色列表响应
 */
export interface GetRolesResp {
  roles: Role[]
}

/**
 * 删除角色响应
 */
export interface DeleteRoleResp {
  message: string
}

// ==================== 角色分配相关 ====================

/**
 * 给用户分配角色请求
 */
export interface AssignRoleToUserReq {
  user: string // 工作空间所属用户
  app: string // 工作空间应用代码
  username: string // 要分配角色的用户名
  role_code: string // 角色代码
  resource_path: string // 资源路径（支持通配符，如 "/user/app/*"）
  start_time?: string // 开始时间（ISO 8601 格式，可选）
  end_time?: string // 结束时间（ISO 8601 格式，可选）
}

/**
 * 给用户分配角色响应
 */
export interface AssignRoleToUserResp {
  assignment: RoleAssignment
}

/**
 * 给组织架构分配角色请求
 */
export interface AssignRoleToDepartmentReq {
  user: string // 工作空间所属用户
  app: string // 工作空间应用代码
  department_path: string // 组织架构路径（如 "/org/master/bizit"）
  role_code: string // 角色代码
  resource_path: string // 资源路径（支持通配符，如 "/user/app/*"）
  start_time?: string // 开始时间（ISO 8601 格式，可选）
  end_time?: string // 结束时间（ISO 8601 格式，可选）
}

/**
 * 给组织架构分配角色响应
 */
export interface AssignRoleToDepartmentResp {
  assignment: RoleAssignment
}

/**
 * 角色分配信息
 */
export interface RoleAssignment {
  id: number
  user: string
  app: string
  subject_type: 'user' | 'department'
  subject: string // 用户名或组织架构路径
  role_code: string
  resource_path: string
  start_time?: string
  end_time?: string
  created_at: string
  updated_at: string
}

/**
 * 移除用户角色请求
 */
export interface RemoveRoleFromUserReq {
  user: string
  app: string
  username: string
  role_code: string
  resource_path: string
}

/**
 * 移除用户角色响应
 */
export interface RemoveRoleFromUserResp {
  message: string
}

/**
 * 移除组织架构角色请求
 */
export interface RemoveRoleFromDepartmentReq {
  user: string
  app: string
  department_path: string
  role_code: string
  resource_path: string
}

/**
 * 移除组织架构角色响应
 */
export interface RemoveRoleFromDepartmentResp {
  message: string
}

/**
 * 获取用户角色请求
 */
export interface GetUserRolesReq {
  user: string
  app: string
  username: string
}

/**
 * 获取用户角色响应
 */
export interface GetUserRolesResp {
  assignments: RoleAssignment[]
}

/**
 * 获取组织架构角色请求
 */
export interface GetDepartmentRolesReq {
  user: string
  app: string
  department_path: string
}

/**
 * 获取组织架构角色响应
 */
export interface GetDepartmentRolesResp {
  assignments: RoleAssignment[]
}

// ==================== API 调用 ====================

/**
 * 获取所有角色
 * @param resourceType 可选的资源类型过滤（directory、table、form、chart、app）
 */
export function getRoles(resourceType?: string): Promise<GetRolesResp> {
  const params = resourceType ? { resource_type: resourceType } : undefined
  return get<GetRolesResp>('/workspace/api/v1/role', params)
}

/**
 * 获取角色详情
 */
export function getRole(id: number): Promise<GetRoleResp> {
  return get<GetRoleResp>(`/workspace/api/v1/role/${id}`)
}

/**
 * 创建角色
 */
export function createRole(data: CreateRoleReq): Promise<CreateRoleResp> {
  return post<CreateRoleResp>('/workspace/api/v1/role', data)
}

/**
 * 更新角色
 */
export function updateRole(id: number, data: UpdateRoleReq): Promise<UpdateRoleResp> {
  return put<UpdateRoleResp>(`/workspace/api/v1/role/${id}`, data)
}

/**
 * 删除角色
 */
export function deleteRole(id: number): Promise<DeleteRoleResp> {
  return del<DeleteRoleResp>(`/workspace/api/v1/role/${id}`)
}

/**
 * 给用户分配角色
 */
export function assignRoleToUser(data: AssignRoleToUserReq): Promise<AssignRoleToUserResp> {
  return post<AssignRoleToUserResp>('/workspace/api/v1/role/assign/user', data)
}

/**
 * 给组织架构分配角色
 */
export function assignRoleToDepartment(data: AssignRoleToDepartmentReq): Promise<AssignRoleToDepartmentResp> {
  return post<AssignRoleToDepartmentResp>('/workspace/api/v1/role/assign/department', data)
}

/**
 * 移除用户角色
 */
export function removeRoleFromUser(data: RemoveRoleFromUserReq): Promise<RemoveRoleFromUserResp> {
  return post<RemoveRoleFromUserResp>('/workspace/api/v1/role/remove/user', data)
}

/**
 * 移除组织架构角色
 */
export function removeRoleFromDepartment(data: RemoveRoleFromDepartmentReq): Promise<RemoveRoleFromDepartmentResp> {
  return post<RemoveRoleFromDepartmentResp>('/workspace/api/v1/role/remove/department', data)
}

/**
 * 获取用户角色
 */
export function getUserRoles(data: GetUserRolesReq): Promise<GetUserRolesResp> {
  return post<GetUserRolesResp>('/workspace/api/v1/role/user', data)
}

/**
 * 获取组织架构角色
 */
export function getDepartmentRoles(data: GetDepartmentRolesReq): Promise<GetDepartmentRolesResp> {
  return post<GetDepartmentRolesResp>('/workspace/api/v1/role/department', data)
}

/**
 * 获取可用于权限申请的角色列表请求
 */
export interface GetRolesForPermissionRequestReq {
  node_type: string // 节点类型 (package, function)
  template_type?: string // 模板类型 (table, form, chart)
}

/**
 * 获取可用于权限申请的角色列表响应
 */
export interface GetRolesForPermissionRequestResp {
  roles: Role[]
}

/**
 * 获取可用于权限申请的角色列表（根据节点类型过滤）
 */
export function getRolesForPermissionRequest(params: GetRolesForPermissionRequestReq): Promise<GetRolesForPermissionRequestResp> {
  return get<GetRolesForPermissionRequestResp>('/workspace/api/v1/role/for_request', params)
}
