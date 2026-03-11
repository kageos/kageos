import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { Logger } from '@/core/utils/logger'
import router from '@/router'
import type { ApiResponse } from '@/types'
import type { PermissionInfo } from './permission'
import { getPermissionDisplayName } from './permission'

// 创建axios实例
// 注意：使用相对路径，通过 Vite 代理转发到网关，避免跨域问题
// 在生产环境可以通过 VITE_API_BASE_URL 环境变量指定绝对路径
const service = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',  // 开发环境使用相对路径（走 Vite 代理），生产环境可配置绝对路径
  timeout: 300000, // 300 秒（5分钟），与后端超时时间保持一致
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
    } else {
      Logger.warn('Request', 'No token found', {
        storeToken: authStore.token,
        localStorageToken: localStorage.getItem('token'),
        url: config.url
      })
    }

    return config
  },
  (error) => {
    Logger.error('Request', '请求拦截器错误', error)
    return Promise.reject(error)
  }
)

// 无感刷新：401 时正在刷新的 Promise，避免多个请求并发刷新
let refreshPromise: Promise<string> | null = null

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse | Blob>) => {
    // 🔥 如果是 blob 响应（文件下载），直接返回，不进行 JSON 解析
    if (response.data instanceof Blob) {
      return response
    }
    
    // 普通 JSON 响应处理
    const { code, data } = response.data as ApiResponse
    // 🔥 统一使用 msg 字段
    const msg = (response.data as any).msg || '请求失败'
    // 🔥 获取 metadata（如 total_cost_mill、trace_id 等）
    const metadata = (response.data as any).metadata

    // 请求成功
    if (code === 0) {
      // 🔥 如果存在 metadata 且 data 是对象，将 metadata 附加到 data 上
      // 这样调用方可以通过 data._metadata 访问元数据
      if (metadata && typeof data === 'object' && data !== null && !Array.isArray(data)) {
        (data as any)._metadata = metadata
      }
      return data
    }

    // 业务错误 - 记录错误信息
    Logger.error('Request', '业务错误', {
      code,
      msg,
      url: response.config.url,
      method: response.config.method
    })
    
    // 🔥 不在这里显示错误消息，让调用方自己处理（避免重复提示）
    // ElMessage.error(msg || '请求失败')
    // 🔥 保留完整的错误信息，包括 response 对象
    const error = new Error(msg) as any
    error.response = response
    return Promise.reject(error)
  },
  async (error) => {
    const { response, config } = error

    if (response) {
      const { status, data } = response

      if (status === 401) {
        const authStore = useAuthStore()
        const isRefreshRequest = config?.url && String(config.url).includes('/auth/refresh')
        const refreshTokenValue =
          (typeof authStore.refreshToken === 'object' && authStore.refreshToken && 'value' in authStore.refreshToken
            ? (authStore.refreshToken as { value: string }).value
            : (authStore.refreshToken as string)) ||
          localStorage.getItem('refresh_token') ||
          ''

        // refresh 接口本身 401：说明 refresh_token 也过期了，直接登出
        if (isRefreshRequest || !refreshTokenValue) {
          authStore.logout()
          router.push('/login')
          ElMessage.warning('登录已过期，请重新登录')
          return Promise.reject(error)
        }

        // 无感刷新：只允许一个 refresh 进行，其他 401 等待同一个结果后重试
        if (!refreshPromise) {
          refreshPromise = authStore.refreshUserToken()
        }
        try {
          await refreshPromise
          // 刷新成功，用新 token 重试原请求（请求拦截器会从 store 取新 token）
          return service.request(config)
        } catch (e) {
          refreshPromise = null
          authStore.logout()
          router.push('/login')
          ElMessage.warning('登录已过期，请重新登录')
          return Promise.reject(error)
        } finally {
          refreshPromise = null
        }
      }

      switch (status) {
        case 403:
          // ⭐ 权限不足：显示详细的权限信息和申请链接
          handlePermissionDenied(data)
          break

        case 404:
          ElMessage.error('请求的资源不存在')
          break

        case 500:
          ElMessage.error('服务器内部错误')
          break

        default:
          ElMessage.error(data?.msg || '网络错误')
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
    // response 已经是 Blob 对象（因为响应拦截器检测到 Blob 后直接返回了 response）
    // 如果 response 是完整的 AxiosResponse，取 response.data；否则直接使用 response
    const blob = response instanceof Blob ? response : (response.data instanceof Blob ? response.data : new Blob([response.data || response]))
    const blobUrl = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = getFilenameFromResponse(response) || 'download'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(blobUrl)
  })
}

// 从响应头获取文件名（支持 RFC 5987 格式）
function getFilenameFromResponse(response: any): string | null {
  const contentDisposition = response.headers['content-disposition']
  if (!contentDisposition) {
    return null
  }
  
  // 优先尝试 RFC 5987 格式：filename*=UTF-8''encoded-filename
  const rfc5987Regex = /filename\*=UTF-8''([^;]+)/
  const rfc5987Match = rfc5987Regex.exec(contentDisposition)
  if (rfc5987Match && rfc5987Match[1]) {
    try {
      // URL 解码
      return decodeURIComponent(rfc5987Match[1])
    } catch (e) {
      // 解码失败，继续尝试其他格式
    }
  }
  
  // 尝试标准格式：filename="filename" 或 filename=filename
  const filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/
  const matches = filenameRegex.exec(contentDisposition)
  if (matches && matches[1]) {
    const filename = matches[1].replace(/['"]/g, '')
    // 如果文件名包含路径分隔符，只取最后一部分
    if (filename.includes('/')) {
      return filename.split('/').pop() || filename
    }
    return filename
  }
  
  return null
}

/**
 * 处理权限不足错误（403）
 * @param data 响应数据（包含权限信息）
 * ⭐ 不弹窗，而是将权限信息存储到 store 中，供详情页面显示
 */
function handlePermissionDenied(data: any) {
  // 尝试从响应数据中提取权限信息
  const permissionInfo: PermissionInfo | undefined = data?.data

  // ⭐ 直接使用导入的 store，避免异步问题
  // 注意：usePermissionErrorStore 必须在函数内部调用，不能在模块级别调用
  const permissionErrorStore = usePermissionErrorStore()

  if (permissionInfo) {
    // ⭐ 将权限信息存储到 store 中，供详情页面显示
    permissionErrorStore.setError(permissionInfo)
  } else {
    // 没有详细的权限信息，显示通用错误提示（但不弹窗）
    const errorMessage = data?.msg || '没有权限访问该资源'
    // ⭐ 也存储到 store 中，使用默认的权限信息结构
    permissionErrorStore.setError({
      resource_path: '',
      action: '',
      action_display: '',
      apply_url: '',
      error_message: errorMessage
    })
  }
}

export default service