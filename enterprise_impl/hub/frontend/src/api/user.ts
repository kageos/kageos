/**
 * Hub 前端用户 API
 * 调用 OS 的用户 API 获取用户信息
 */

import { get } from '@/utils/request'

// 用户信息接口
export interface UserInfo {
  id: number
  username: string
  email: string
  register_type: string
  avatar: string
  nickname?: string
  signature?: string
  gender?: string
  email_verified: boolean
  status: string
  created_at: string
}

// 查询用户响应
export interface QueryUserResp {
  user: UserInfo
}

/**
 * 获取 OS API 基础地址
 */
function getOSAPIBaseURL(): string {
  // 从环境变量获取配置
  const osAPIBaseURL = import.meta.env.VITE_OS_API_BASE_URL
  
  if (osAPIBaseURL) {
    return osAPIBaseURL
  }
  
  // 如果没有配置，从当前域名推断
  const isDev = import.meta.env.DEV || import.meta.env.MODE === 'development'
  if (isDev) {
    // 开发环境：假设 OS 运行在 5173 端口
    return 'http://localhost:5173/workspace/api/v1'
  } else {
    // 生产环境：使用当前域名
    return `${window.location.origin}/workspace/api/v1`
  }
}

/**
 * 根据用户名查询用户信息
 * @param username 用户名
 */
export async function queryUser(username: string): Promise<UserInfo | null> {
  if (!username) {
    return null
  }
  
  try {
    const baseURL = getOSAPIBaseURL()
    const url = `${baseURL}/user/query?username=${encodeURIComponent(username)}`
    
    // 使用 fetch 直接调用 OS API（因为可能跨域）
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        // 传递 token（如果有）
        ...(localStorage.getItem('token') ? {
          'X-Token': localStorage.getItem('token') || ''
        } : {})
      }
    })
    
    if (!response.ok) {
      console.warn(`[queryUser] 获取用户信息失败: ${response.status}`)
      return null
    }
    
    const responseData = await response.json()
    // 处理 OS API 的响应格式：{ code: 0, data: { user: {...} }, msg: "成功" }
    if (responseData.code === 0 && responseData.data) {
      return responseData.data.user || null
    }
    // 降级处理：直接返回 user 字段（如果存在）
    if (responseData.user) {
      return responseData.user
    }
    return null
  } catch (error) {
    console.error('[queryUser] 获取用户信息失败:', error)
    return null
  }
}

// 批量获取用户信息请求
export interface GetUsersByUsernamesReq {
  usernames: string[]
}

// 批量获取用户信息响应
export interface GetUsersByUsernamesResp {
  users: UserInfo[]
}

/**
 * 根据用户名列表批量获取用户信息
 * @param usernames 用户名列表，最多100个
 */
export async function getUsersByUsernames(usernames: string[]): Promise<UserInfo[]> {
  if (!usernames || usernames.length === 0) {
    return []
  }
  
  try {
    const baseURL = getOSAPIBaseURL()
    const url = `${baseURL}/users`
    
    // 使用 fetch 直接调用 OS API（因为可能跨域）
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        // 传递 token（如果有）
        ...(localStorage.getItem('token') ? {
          'X-Token': localStorage.getItem('token') || ''
        } : {})
      },
      body: JSON.stringify({ usernames })
    })
    
    if (!response.ok) {
      console.warn(`[getUsersByUsernames] 批量获取用户信息失败: ${response.status}`)
      return []
    }
    
    const responseData = await response.json()
    // 处理 OS API 的响应格式：{ code: 0, data: { users: [...] }, msg: "成功" }
    if (responseData.code === 0 && responseData.data) {
      return responseData.data.users || []
    }
    // 降级处理：直接返回 users 字段（如果存在）
    if (responseData.users) {
      return responseData.users
    }
    return []
  } catch (error) {
    console.error('[getUsersByUsernames] 批量获取用户信息失败:', error)
    return []
  }
}

