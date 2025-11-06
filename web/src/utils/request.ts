import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import type { ApiResponse } from '@/types'

// 创建axios实例
// 注意：使用相对路径，通过 Vite 代理转发到网关，避免跨域问题
// 在生产环境可以通过 VITE_API_BASE_URL 环境变量指定绝对路径
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
    const authStore = useAuthStore()
    
    // 从store获取token - 直接访问store中的token（Pinia会自动解包ref）
    let token: string = ''
    
    // 尝试多种方式获取token
    if (authStore.token) {
      // 如果是ref对象，访问.value
      if (typeof authStore.token === 'object' && 'value' in authStore.token) {
        token = authStore.token.value as string
      } else {
        // 直接就是值
        token = authStore.token as string
      }
    }
    
    // 如果还是空，尝试从localStorage获取
    if (!token) {
      token = localStorage.getItem('token') || ''
    }

    // 添加token到请求头（后端使用X-Token头部）
    if (token && typeof token === 'string' && token.trim()) {
      // 确保headers对象存在
      if (!config.headers) {
        config.headers = {} as any
      }
      
      // 设置X-Token头部
      if (typeof config.headers.set === 'function') {
        // AxiosHeaders对象
        config.headers.set('X-Token', token)
      } else {
        // 普通对象，直接赋值
        (config.headers as any)['X-Token'] = token
      }
      
      console.log('[Request] URL:', config.url, 'X-Token:', token.substring(0, 20) + '...')
    } else {
      console.error('[Request] ❌ No token found!')
      console.error('[Request] Store token:', authStore.token)
      console.error('[Request] LocalStorage token:', localStorage.getItem('token'))
      console.error('[Request] URL:', config.url)
    }

    return config
  },
  (error) => {
    console.error('请求拦截器错误:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const { code, message, data } = response.data

    // 请求成功
    if (code === 0) {
      return data
    }

    // 业务错误
    ElMessage.error(message || '请求失败')
    return Promise.reject(new Error(message || '请求失败'))
  },
  async (error) => {
    const { response } = error

    if (response) {
      const { status, data } = response

      switch (status) {
        case 401:
          // 未授权，清除token并跳转到登录页
          const authStore = useAuthStore()
          await ElMessageBox.confirm(
            '登录状态已过期，请重新登录',
            '提示',
            {
              confirmButtonText: '重新登录',
              cancelButtonText: '取消',
              type: 'warning'
            }
          )
          authStore.logout()
          router.push('/login')
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
// 支持两种模式：
// 1. params 参数 - 作为查询参数（默认）
// 2. data 参数 - 作为 body（用于特殊场景，如回调接口）
export function get<T = any>(url: string, params?: any, useBody: boolean = false): Promise<T> {
  if (useBody) {
    // 特殊场景：GET 请求带 body（用于回调接口）
    console.log('[Request GET with Body] URL:', url)
    console.log('[Request GET with Body] Data:', params)
    
    return service.request({
      url,
      method: 'GET',
      data: params,
      headers: {
        'Content-Type': 'application/json'
      }
    })
  } else {
    // 标准场景：GET 请求使用查询参数
    // 确保 params 是对象，并且只包含有值的字段
    const cleanParams: Record<string, any> = {}
    if (params && typeof params === 'object') {
      Object.keys(params).forEach(key => {
        const value = params[key]
        // 只包含非空值（排除 null、undefined、空字符串）
        if (value !== null && value !== undefined && value !== '') {
          cleanParams[key] = value
        }
      })
    }
    console.log('[Request GET] URL:', url)
    console.log('[Request GET] Original Params:', params)
    console.log('[Request GET] Cleaned Params:', cleanParams)
    console.log('[Request GET] Sorts:', cleanParams.sorts)
    return service.get(url, { params: cleanParams })
  }
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
// 支持两种模式：
// 1. 无参数 - 标准 DELETE（默认）
// 2. data 参数 - 带 body 的 DELETE（用于特殊场景，如回调接口）
export function del<T = any>(url: string, data?: any): Promise<T> {
  if (data) {
    // 特殊场景：DELETE 请求带 body
    return service.request({
      url,
      method: 'DELETE',
      data
    })
  } else {
    // 标准场景：DELETE 请求无 body
    return service.delete(url)
  }
}

// 封装文件上传
export function upload<T = any>(url: string, formData: FormData): Promise<T> {
  return service.post(url, formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

// 封装文件下载
export function download(url: string, params?: any): Promise<void> {
  return service.get(url, {
    params,
    responseType: 'blob'
  }).then((response: any) => {
    const blob = new Blob([response])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = getFilenameFromResponse(response) || 'download'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  })
}

// 从响应头获取文件名
function getFilenameFromResponse(response: any): string | null {
  const contentDisposition = response.headers['content-disposition']
  if (contentDisposition) {
    const filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/
    const matches = filenameRegex.exec(contentDisposition)
    if (matches && matches[1]) {
      return matches[1].replace(/['"]/g, '')
    }
  }
  return null
}

export default service