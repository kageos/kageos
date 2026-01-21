import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import router from '@/router'

// 创建axios实例
const service = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',  // 开发环境使用相对路径（走 Vite 代理），生产环境可配置绝对路径
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 从 localStorage 获取 token
    const token = localStorage.getItem('token') || ''

    // 添加token到请求头（后端使用X-Token头部）
    if (token && typeof token === 'string' && token.trim()) {
      if (!config.headers) {
        config.headers = {} as any
      }
      
      if (typeof config.headers.set === 'function') {
        config.headers.set('X-Token', token)
      } else {
        (config.headers as any)['X-Token'] = token
      }
    }

    return config
  },
  (error) => {
    console.error('[Request] 请求拦截器错误', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    const { code, data } = response.data as any
    const message = (response.data as any).msg || (response.data as any).message

    // 请求成功
    if (code === 0) {
      return data
    }

    // 业务错误
    const error = new Error(message || '请求失败') as any
    error.response = response
    return Promise.reject(error)
  },
  async (error) => {
    const { response } = error

    if (response) {
      const { status, data } = response

      switch (status) {
        case 401:
          // 未授权，清除token并跳转到登录页
          await ElMessageBox.confirm(
            '登录状态已过期，请重新登录',
            '提示',
            {
              confirmButtonText: '重新登录',
              cancelButtonText: '取消',
              type: 'warning'
            }
          )
          localStorage.removeItem('token')
          // 跳转到 OS 登录页（Hub 和 OS 共享用户系统）
          window.location.href = import.meta.env.VITE_OS_LOGIN_URL || 'http://localhost:5173/login'
          break

        case 403:
          ElMessage.error('没有权限访问')
          break

        case 404:
          ElMessage.error('请求的资源不存在')
          break

        case 500:
          ElMessage.error('服务器内部错误')
          break

        default:
          ElMessage.error(data?.message || '网络错误')
      }
    } else if (error.code === 'ECONNABORTED') {
      ElMessage.error('请求超时，请检查网络连接')
    } else {
      ElMessage.error('网络错误，请检查网络连接')
    }

    return Promise.reject(error)
  }
)

// 封装GET请求
export function get<T = any>(url: string, params?: any): Promise<T> {
  const cleanParams: Record<string, any> = {}
  if (params && typeof params === 'object') {
    Object.keys(params).forEach(key => {
      const value = params[key]
      if (value !== null && value !== undefined && value !== '') {
        cleanParams[key] = value
      }
    })
  }
  return service.get(url, { params: cleanParams })
}

// 封装POST请求
export function post<T = any>(url: string, data?: any): Promise<T> {
  return service.post(url, data)
}

// 封装PUT请求
export function put<T = any>(url: string, data?: any): Promise<T> {
  return service.put(url, data)
}

// 封装DELETE请求
export function del<T = any>(url: string, data?: any): Promise<T> {
  if (data) {
    return service.request({
      url,
      method: 'DELETE',
      data
    })
  } else {
    return service.delete(url)
  }
}

export default service

