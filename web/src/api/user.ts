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
  return put<UpdateUserResp>('/api/v1/user/update', data)
}

// 根据用户名精确查询
export interface QueryUserResp {
  user: UserInfo
}

/**
 * 根据用户名精确查询用户信息
 */
export function queryUser(username: string) {
  return get<QueryUserResp>('/api/v1/user/query', { username })
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
  return get<SearchUsersFuzzyResp>('/api/v1/user/search_fuzzy', { keyword, limit })
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
  return post<GetUsersByUsernamesResp>('/api/v1/users', { usernames })
}

