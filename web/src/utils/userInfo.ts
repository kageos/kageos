/**
 * 用户信息相关工具函数
 */
import type { UserInfo } from '@/types'

/**
 * 创建占位符用户对象
 * 当用户信息未找到时，至少显示用户名
 */
export function createPlaceholderUser(username: string): UserInfo {
  return {
    username,
    nickname: '',
    avatar: '',
    email: '',
  } as UserInfo
}

/**
 * 格式化用户显示名称
 * @param user 用户信息
 * @returns 格式化后的显示名称，格式：username(nickname) 或 username
 */
export function formatUserDisplayName(user: UserInfo | null | undefined): string {
  if (!user) return ''
  return user.nickname ? `${user.username}(${user.nickname})` : user.username
}

/**
 * 从 modelValue 解析用户名列表
 * @param modelValue 可以是字符串、字符串数组或 null
 * @param multiple 是否多选模式
 * @returns 用户名数组
 */
export function parseUsernamesFromModelValue(
  modelValue: string | string[] | null,
  multiple: boolean = false
): string[] {
  if (!modelValue) return []
  
  if (multiple) {
    // 多选模式：modelValue 应该是数组
    if (Array.isArray(modelValue)) {
      return modelValue.map(u => String(u).trim()).filter(u => u)
    } else if (modelValue) {
      // 如果不是数组但是有值，转换为数组
      return [String(modelValue).trim()].filter(u => u)
    }
  } else {
    // 单选模式：modelValue 应该是字符串
    if (modelValue) {
      const username = String(modelValue).trim()
      return username ? [username] : []
    }
  }
  
  return []
}

/**
 * 检查两个用户名列表是否相同（顺序和内容）
 */
export function isUsernameListEqual(list1: string[], list2: string[]): boolean {
  if (list1.length !== list2.length) return false
  return list1.every((username, index) => username === list2[index])
}

/**
 * 从用户列表中提取用户名
 */
export function extractUsernames(users: UserInfo[]): string[] {
  return users.map(u => u.username).filter(Boolean)
}

/**
 * 根据用户名列表构建用户映射
 */
export function buildUserMap(users: UserInfo[]): Map<string, UserInfo> {
  const map = new Map<string, UserInfo>()
  users.forEach(user => {
    if (user.username) {
      map.set(user.username, user)
    }
  })
  return map
}

/**
 * 按照指定的用户名顺序重新组织用户列表
 * @param usernames 指定的用户名顺序
 * @param userMap 用户映射
 * @param createPlaceholder 是否创建占位符（当用户不存在时）
 * @returns 按顺序组织的用户列表
 */
export function reorderUsersByUsernames(
  usernames: string[],
  userMap: Map<string, UserInfo>,
  createPlaceholder: boolean = true
): UserInfo[] {
  return usernames.map(username => {
    const user = userMap.get(username)
    if (user) {
      return user
    }
    if (createPlaceholder) {
      return createPlaceholderUser(username)
    }
    return null
  }).filter((user): user is UserInfo => user !== null)
}

