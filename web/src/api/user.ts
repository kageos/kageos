import { get, put, post } from '@/utils/request'
import type { UserInfo } from '@/types'

// 更新用户信息
export interface UpdateUserReq {
  nickname?: string
  signature?: string
  avatar?: string
  gender?: 'male' | 'female' | 'other' | ''
}

export interface UpdateUserResp {
  user: UserInfo
}

/**
 * 更新当前登录用户信息
 */
export function updateUser(data: UpdateUserReq) {
  return put<UpdateUserResp>('/hr/api/v1/user/update', data)
}

// 根据用户名精确查询
export interface QueryUserResp {
  user: UserInfo
}

/**
 * 根据用户名精确查询用户信息
 */
export function queryUser(username: string) {
  return get<QueryUserResp>('/hr/api/v1/user/query', { username })
}

// 模糊查询用户
export interface SearchUsersFuzzyResp {
  users: UserInfo[]
}

/**
 * 模糊查询用户（支持用户名和邮箱）
 * @param keyword 搜索关键词
 * @param limit 返回数量限制，默认10，最大100
 */
export function searchUsersFuzzy(keyword: string, limit: number = 10) {
  return get<SearchUsersFuzzyResp>('/hr/api/v1/user/search_fuzzy', { keyword, limit })
}

// 批量获取用户信息
export interface GetUsersByUsernamesReq {
  usernames: string[]
}

export interface GetUsersByUsernamesResp {
  users: UserInfo[]
}

/**
 * 根据用户名列表批量获取用户信息
 * @param usernames 用户名列表，最多100个
 */
export function getUsersByUsernames(usernames: string[]) {
  return post<GetUsersByUsernamesResp>('/hr/api/v1/users', { usernames })
}

// 分配用户组织架构
export interface AssignUserReq {
  username: string
  department_full_path?: string | null // 部门完整路径，null 表示清空
  leader_username?: string | null // Leader 用户名，null 表示清空
}

export interface AssignUserResp {
  user: UserInfo
}

/**
 * 分配用户组织架构（更新用户的部门和 Leader）
 */
export function assignUserOrganization(data: AssignUserReq) {
  return post<AssignUserResp>('/hr/api/v1/user/assign', data)
}


// 根据部门获取用户列表
export interface GetUsersByDepartmentReq {
  department_full_path: string
}

export interface GetUsersByDepartmentResp {
  users: UserInfo[]
}

/**
 * 根据部门完整路径获取用户列表
 */
export function getUsersByDepartment(departmentFullPath: string) {
  return get<GetUsersByDepartmentResp>('/hr/api/v1/user/department', { department_full_path: departmentFullPath })
}

