import axios from 'axios'
import { AxiosHeaders, type AxiosRequestConfig, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import { Logger } from '@/architecture/shared/logger'
import { getApiBaseURL } from '@/architecture/infrastructure/config/runtime'
import { getCurrentRouteFullPath, getCurrentRoutePath, navigateTo } from '@/architecture/shared/routing/navigation'
import type { ApiResponse } from '@/architecture/shared/apiTypes'
import { isWorkspaceForbiddenError } from '@/architecture/shared/apiError'
import { extractApiMessage, isAuthExpiredBusinessResponse, isRefreshRequestUrl } from './authSession'

const CLIENT_SOURCE_HEADER = 'X-Client-Source'
const CLIENT_SOURCE_BROWSER = 'browser'

type AuthRetryAxiosRequestConfig = InternalAxiosRequestConfig & {
  _retryAfterRefresh?: boolean
}

type BusinessResponseError = Error & {
  response: AxiosResponse<ApiResponse | Blob>
}

interface AuthFetchOptions {
  retryOnAuthExpired?: boolean
}

// 创建axios实例
// 注意：使用相对路径，通过 Vite 代理转发到网关，避免跨域问题
const service = axios.create({
  baseURL: getApiBaseURL(),
  timeout: 300000, // 300 秒（5分钟），与后端超时时间保持一致
  headers: {
    'Content-Type': 'application/json'
  }
})

function setHeader(config: InternalAxiosRequestConfig, key: string, value: string) {
  if (!config.headers) {
    config.headers = new AxiosHeaders()
  }

  config.headers.set(key, value)
}

function removeHeader(config: InternalAxiosRequestConfig, key: string) {
  if (!config.headers) {
    return
  }

  config.headers.delete(key)
}

function isLikelyHtmlResponse(response: AxiosResponse<ApiResponse | Blob>): boolean {
  const contentType = String(response.headers?.['content-type'] || '').toLowerCase()
  const data = response.data
  return typeof data === 'string' && (
    contentType.includes('text/html') ||
    /^\s*(<!doctype html|<html[\s>])/i.test(data)
  )
}

function rejectUnexpectedApiResponse(
  response: AxiosResponse<ApiResponse | Blob>,
  message: string
): Promise<never> {
  const error = new Error(message) as BusinessResponseError
  error.response = response
  Logger.error('Request', 'API 响应格式异常', {
    url: response.config.url,
    method: response.config.method,
    status: response.status,
    contentType: response.headers?.['content-type']
  })
  return Promise.reject(error)
}

function getRefreshTokenValue(): string {
  const authStore = useAuthStore()
  const storeRefreshToken = authStore.refreshToken
  return (typeof storeRefreshToken === 'string' ? storeRefreshToken : '') || localStorage.getItem('refresh_token') || ''
}

function getAccessTokenValue(): string {
  const authStore = useAuthStore()
  const storeToken = authStore.token
  return (typeof storeToken === 'string' ? storeToken : '') || localStorage.getItem('token') || ''
}

// 请求拦截器
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = getAccessTokenValue()
    const isRefreshRequest = isRefreshRequestUrl(config.url)

    // 添加token到请求头（后端使用X-Token头部）
    if (!isRefreshRequest && token && typeof token === 'string' && token.trim()) {
      setHeader(config, 'X-Token', token)
    } else if (isRefreshRequest) {
      removeHeader(config, 'X-Token')
    } else {
      Logger.warn('Request', 'No token found', {
        hasStoredToken: Boolean(token && typeof token === 'string' && token.trim()),
        url: config.url
      })
    }

    setHeader(config, CLIENT_SOURCE_HEADER, CLIENT_SOURCE_BROWSER)

    return config
  },
  (error) => {
    Logger.error('Request', '请求拦截器错误', error)
    return Promise.reject(error)
  }
)

// 无感刷新：请求共享同一个 refresh Promise，避免并发重复刷新
let refreshPromise: Promise<string> | null = null
let sessionExpiredPromise: Promise<void> | null = null

async function refreshAccessTokenForRetry(): Promise<string> {
  if (!getRefreshTokenValue()) {
    throw new Error('No refresh token')
  }

  const authStore = useAuthStore()
  if (!refreshPromise) {
    refreshPromise = authStore.refreshUserToken().finally(() => {
      refreshPromise = null
    })
  }

  return refreshPromise
}

async function handleSessionExpired(): Promise<void> {
  if (!sessionExpiredPromise) {
    sessionExpiredPromise = (async () => {
      const authStore = useAuthStore()
      await authStore.logout({
        callApi: false,
        notify: false,
        redirectToLogin: false,
      })

      const currentPath = getCurrentRoutePath()
      const currentFullPath = getCurrentRouteFullPath()
      if (currentPath !== '/login') {
        await navigateTo({
          path: '/login',
          query: currentFullPath ? { redirect: currentFullPath } : undefined,
        })
      }

      ElMessage.warning('登录已过期，请重新登录')
    })().finally(() => {
      sessionExpiredPromise = null
    })
  }

  return sessionExpiredPromise
}

function getFetchUrl(input: RequestInfo | URL): string | undefined {
  if (typeof input === 'string') {
    return input
  }
  if (input instanceof URL) {
    return input.toString()
  }
  if (input instanceof Request) {
    return input.url
  }
  return undefined
}

function createAuthFetchInit(input: RequestInfo | URL, init: RequestInit = {}): RequestInit {
  const headers = new Headers(input instanceof Request ? input.headers : undefined)
  new Headers(init.headers).forEach((value, key) => {
    headers.set(key, value)
  })

  const url = getFetchUrl(input)
  if (isRefreshRequestUrl(url)) {
    headers.delete('X-Token')
  } else {
    const token = getAccessTokenValue()
    if (token.trim()) {
      headers.set('X-Token', token)
    } else {
      headers.delete('X-Token')
    }
  }
  headers.set(CLIENT_SOURCE_HEADER, CLIENT_SOURCE_BROWSER)

  return {
    ...init,
    headers,
  }
}

async function isAuthExpiredFetchResponse(response: Response): Promise<boolean> {
  if (response.status === 401) {
    return true
  }

  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) {
    return false
  }

  const payload = await response.clone().json().catch(() => null)
  return isAuthExpiredBusinessResponse(payload)
}

export async function authFetch(
  input: RequestInfo | URL,
  init: RequestInit = {},
  options: AuthFetchOptions = {}
): Promise<Response> {
  const retryOnAuthExpired = options.retryOnAuthExpired !== false
  const isRefreshRequest = isRefreshRequestUrl(getFetchUrl(input))

  const request = () => fetch(input, createAuthFetchInit(input, init))
  let response = await request()

  if (!retryOnAuthExpired || isRefreshRequest || !(await isAuthExpiredFetchResponse(response))) {
    return response
  }

  try {
    await refreshAccessTokenForRetry()
    response = await request()
    return response
  } catch (refreshError) {
    await handleSessionExpired()
    throw refreshError
  }
}

async function retryWithFreshToken(
  config?: AuthRetryAxiosRequestConfig,
  originalError?: unknown
): Promise<AxiosResponse<ApiResponse | Blob>> {
  if (!config || config._retryAfterRefresh || isRefreshRequestUrl(config.url) || !getRefreshTokenValue()) {
    await handleSessionExpired()
    return Promise.reject(originalError)
  }

  try {
    await refreshAccessTokenForRetry()
    config._retryAfterRefresh = true
    return await service.request(config)
  } catch (refreshError) {
    await handleSessionExpired()
    return Promise.reject(refreshError)
  }
}

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse | Blob>) => {
    // 🔥 如果是 blob 响应（文件下载），直接返回，不进行 JSON 解析
    if (response.data instanceof Blob) {
      return response
    }

    if (isLikelyHtmlResponse(response)) {
      return rejectUnexpectedApiResponse(response, '接口返回了页面内容，请检查前端代理或网关配置')
    }
    
    // 普通 JSON 响应处理
    const responsePayload = response.data as ApiResponse<AxiosResponse<ApiResponse | Blob>>
    if (!responsePayload || typeof responsePayload !== 'object' || !('code' in responsePayload)) {
      return rejectUnexpectedApiResponse(response, '接口返回格式异常，请检查后端服务或网关配置')
    }
    const { code, data, metadata } = responsePayload
    // 🔥 统一使用 msg 字段
    const msg = extractApiMessage(responsePayload) || '请求失败'

    // 请求成功
    if (code === 0) {
      // 🔥 如果存在 metadata 且 data 是对象，将 metadata 附加到 data 上
      // 这样调用方可以通过 data._metadata 访问元数据
      if (metadata && typeof data === 'object' && data !== null && !Array.isArray(data)) {
        ;(data as unknown as Record<string, unknown>)._metadata = metadata
      }
      return data
    }

    // 🔥 不在这里显示错误消息，让调用方自己处理（避免重复提示）
    // ElMessage.error(msg || '请求失败')
    // 🔥 保留完整的错误信息，包括 response 对象
    const error = new Error(msg) as BusinessResponseError
    error.response = response

    const logPayload = {
      code,
      msg,
      url: response.config.url,
      method: response.config.method
    }

    // workspace 无权限是页面可处理状态，不当作未处理异常噪音输出
    if (isWorkspaceForbiddenError(error)) {
      Logger.warn('Request', '业务拒绝', logPayload)
    } else {
      Logger.error('Request', '业务错误', logPayload)
    }

    if (isAuthExpiredBusinessResponse(responsePayload)) {
      return retryWithFreshToken(response.config as AuthRetryAxiosRequestConfig, error)
    }

    return Promise.reject(error)
  },
  async (error) => {
    const { response, config } = error

    if (response) {
      const { status, data } = response

      if (status === 401) {
        return retryWithFreshToken(config as AuthRetryAxiosRequestConfig, error)
      }

      switch (status) {
        case 403:
          ElMessage.error(extractApiMessage(data) || '请求被拒绝')
          break

        case 404:
          ElMessage.error('请求的资源不存在')
          break

        case 500:
          ElMessage.error('服务器内部错误')
          break

        default:
          ElMessage.error(extractApiMessage(data) || '网络错误')
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
export function get<T = unknown>(url: string, params?: unknown, useBody: boolean = false, config?: AxiosRequestConfig): Promise<T> {
  if (useBody) {
    // 特殊场景：GET 请求带 body（用于回调接口）
    return service.request({
      ...config,
      url,
      method: 'GET',
      data: params,
      headers: {
        ...(config?.headers || {}),
        'Content-Type': 'application/json'
      }
    })
  } else {
    // 标准场景：GET 请求使用查询参数
    // 确保 params 是对象，并且只包含有值的字段
    const cleanParams: Record<string, unknown> = {}
    if (params && typeof params === 'object') {
      const paramRecord = params as Record<string, unknown>
      Object.keys(paramRecord).forEach(key => {
        const value = paramRecord[key]
        // 只包含非空值（排除 null、undefined、空字符串）
        if (value !== null && value !== undefined && value !== '') {
          cleanParams[key] = value
        }
      })
    }
    return service.get(url, { ...config, params: cleanParams })
  }
}

// 封装POST请求
export function post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return service.post(url, data, config)
}

// 封装PUT请求
export function put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return service.put(url, data, config)
}

// 封装PATCH请求
export function patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return service.patch(url, data, config)
}

// 封装DELETE请求
// 支持两种模式：
// 1. 无参数 - 标准 DELETE（默认）
// 2. data 参数 - 带 body 的 DELETE（用于特殊场景，如回调接口）
export function del<T = unknown>(url: string, data?: unknown): Promise<T> {
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
export function upload<T = unknown>(url: string, formData: FormData): Promise<T> {
  return service.post(url, formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

// 封装文件下载
export function download(url: string, params?: unknown): Promise<void> {
  return service.get(url, {
    params,
    responseType: 'blob'
  }).then((response: Blob | AxiosResponse<Blob>) => {
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
function getFilenameFromResponse(response: Blob | AxiosResponse<Blob>): string | null {
  if (response instanceof Blob) {
    return null
  }
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
    } catch {
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

export default service
