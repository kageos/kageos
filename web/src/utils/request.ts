import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
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

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const { code, data } = response.data
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

/**
 * 处理权限不足错误（403）
 * @param data 响应数据（包含权限信息）
 */
async function handlePermissionDenied(data: any) {
  // 尝试从响应数据中提取权限信息
  const permissionInfo: PermissionInfo | undefined = data?.data

  if (permissionInfo && permissionInfo.action_display && permissionInfo.apply_url) {
    // 有详细的权限信息，显示权限申请提示
    try {
      await ElMessageBox.confirm(
        `您没有 ${permissionInfo.action_display} 权限，是否立即申请？`,
        '权限不足',
        {
          confirmButtonText: '立即申请',
          cancelButtonText: '取消',
          type: 'warning',
          dangerouslyUseHTMLString: false
        }
      )
      // 用户点击"立即申请"，跳转到权限申请页面
      // 注意：apply_url 是后端返回的相对路径，如 /permissions/apply?resource=...&action=...
      // 这里需要根据实际的路由配置来处理
      if (permissionInfo.apply_url.startsWith('/')) {
        // 如果是相对路径，直接使用 router.push
        router.push(permissionInfo.apply_url)
      } else {
        // 如果是完整 URL，使用 window.open
        window.open(permissionInfo.apply_url, '_blank')
      }
    } catch {
      // 用户点击"取消"，不执行任何操作
    }
  } else {
    // 没有详细的权限信息，显示通用错误提示
    const errorMessage = data?.msg || permissionInfo?.error_message || '没有权限访问该资源'
    ElMessage.error(errorMessage)
  }
}

export default service