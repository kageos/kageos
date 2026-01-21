/**
 * Hub 前端用户 API
 * 调用 hr-server 的用户 API 获取用户信息
 * 注意：需要调用主项目（5173端口）的 hr-server API
 */

import axios from 'axios'

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
  // ⭐ 新增：组织架构相关字段
  department_full_path?: string
  department_name?: string
  department_full_name_path?: string
  leader_username?: string
  leader_display_name?: string
}

// 查询用户响应
export interface QueryUserResp {
  user: UserInfo
}

/**
 * 获取主项目（OS）的 API 基础地址
 * Hub 运行在 5174 端口，需要调用主项目（5173端口）的 API
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
    // 开发环境：主项目运行在 5173 端口
    return 'http://localhost:5173'
  } else {
    // 生产环境：使用当前域名（假设主项目和 Hub 在同一域名下）
    return window.location.origin
  }
}

/**
 * 创建用于调用主项目 API 的 axios 实例
 */
function createOSAPIInstance() {
  const instance = axios.create({
    baseURL: getOSAPIBaseURL(),
    timeout: 30000,
    headers: {
      'Content-Type': 'application/json'
    }
  })

  // 请求拦截器：添加 token
  instance.interceptors.request.use(
    (config) => {
      const token = localStorage.getItem('token') || ''
      if (token && typeof token === 'string' && token.trim()) {
        if (!config.headers) {
          config.headers = {} as any
        }
        (config.headers as any)['X-Token'] = token
      }
      return config
    },
    (error) => {
      return Promise.reject(error)
    }
  )

  // 响应拦截器：处理响应格式
  instance.interceptors.response.use(
    (response) => {
      const { code, data } = response.data as any
      if (code === 0) {
        return data
      }
      const error = new Error((response.data as any).msg || (response.data as any).message || '请求失败') as any
      error.response = response
      return Promise.reject(error)
    },
    (error) => {
      return Promise.reject(error)
    }
  )

  return instance
}

const osAPI = createOSAPIInstance()

/**
 * 根据用户名查询用户信息
 * @param username 用户名
 */
export async function queryUser(username: string): Promise<UserInfo | null> {
  if (!username) {
    return null
  }
  
  try {
    // ⭐ 调用主项目（5173端口）的 hr-server 接口
    const response = await osAPI.get<QueryUserResp>('/hr/api/v1/user/query', {
      params: { username }
    })
    return response.user || null
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
    // ⭐ 调用主项目（5173端口）的 hr-server 接口
    const response = await osAPI.post<GetUsersByUsernamesResp>('/hr/api/v1/users', { usernames })
    return response.users || []
  } catch (error) {
    console.error('[getUsersByUsernames] 批量获取用户信息失败:', error)
    return []
  }
}

