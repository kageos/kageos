/**
 * 统一文件上传工具
 * 当前官方路径：MinIO presigned_url
 */

import type { UploadCredentials, UploadProgress, UploadResult } from './types'
import { PresignedURLUploader } from './presigned-url'
import { authFetch } from '@/architecture/infrastructure/apiClient/request'
import {
  ensurePublicAnonymousToken,
  getCurrentPublicShareId,
  publicShareAnonymousHeaders,
} from '@/architecture/infrastructure/api/publicShare'

export type { UploadCredentials, UploadProgress, UploadResult } from './types'

type UploadCredentialsResponseData = UploadCredentials & {
  Method?: UploadCredentials['method']
}

/**
 * 文件上传器接口
 */
export interface Uploader {
  upload(
    credentials: UploadCredentials,
    file: File,
    onProgress: (progress: UploadProgress) => void
  ): Promise<void>
  
  cancel(): void
}

function createUploader(method: string | undefined): Uploader {
  if (!method) {
    throw new Error(`上传凭证缺少 method 字段，无法创建上传器`)
  }

  const normalized = String(method).toLowerCase().trim()
  if (normalized !== 'presigned_url') {
    console.error('[uploadFile] 未知的上传方式:', {
      original: method,
      normalized,
      type: typeof method,
    })
    throw new Error(`不支持的上传方式: ${method}（当前仅支持 presigned_url）`)
  }

  return new PresignedURLUploader()
}

/**
 * 文件信息（用于批量complete）
 */
export interface FileInfo {
  key: string
  bucket: string
  ref: string
  router: string
  file_name: string
  description?: string
  file_size: number
  content_type: string
  hash?: string
  thumbnail_ref?: string
  thumbnail_url?: string
  preview_kind?: string
  error?: string
}

/**
 * 上传结果（包含上传器实例）
 */
export interface UploadFileResult extends UploadResult {
  uploader: Uploader  // ✨ 上传器实例，用于取消上传
  fileInfo?: FileInfo // ✨ 文件信息（用于批量complete）
}

export interface UploadFileOptions {
  /** 原文件 object key；传入后后端会生成与原文件同路径的缩略图/封面 key */
  previewForKey?: string
}

/**
 * 统一上传入口
 * 
 * 流程：
 * 1. 用户拖文件/选择文件
 * 2. 调用此函数 → 先请求后端获取上传凭证（包含域名）
 * 3. 后端返回：{ method, upload_url, upload_host, upload_domain, storage, ... }
 * 4. 校验 method 并创建预签名 URL 上传器
 * 5. 使用上传器执行上传（此时已知道上传域名）
 * 
 * ✨ 返回上传器实例，支持取消上传
 */
export async function uploadFile(
  router: string,
  file: File,
  onProgress: (progress: UploadProgress & { uploadDomain?: string }) => void,
  options: UploadFileOptions = {}
): Promise<UploadFileResult> {
  
  // ✨ Step 0: 计算文件 SHA256 hash（用于文件标识和 SDK 下载缓存）
  let hash = ''
  try {
    hash = await calculateSHA256(file)
    if (!hash) {
      console.warn('[uploadFile] Hash calculation returned empty string for file:', file.name)
    }
  } catch (error) {
    console.error('[uploadFile] Hash calculation failed for file:', file.name, error)
    // 继续上传，但hash为空（不影响上传流程）
  }
  
  // ✨ Step 1: 获取上传凭证（包含域名信息）
  // 这一步会请求后端 API，后端会根据配置的存储类型返回对应的上传凭证
  const credentials = await getUploadCredentials(router, file, options)
  
  // ✨ Step 2: 校验上传方式并创建上传器
  const uploader = createUploader(credentials.method)
  
  // ✨ Step 3: 执行上传
  // 此时已知道上传域名（credentials.upload_domain）
  try {
    // 包装进度回调，添加上传域名信息
    const progressWrapper = (progress: UploadProgress) => {
      onProgress({
        ...progress,
        uploadDomain: credentials.upload_domain, // ✨ 传递上传域名
      })
    }
    
    await uploader.upload(credentials, file, progressWrapper)
    
    // ✅ 返回上传结果（包含文件信息，但不立即调用complete）
    // complete 由调用方统一批量处理
    return {
      downloadURL: '', // 暂时为空，批量complete后会返回
      key: credentials.key,
      bucket: credentials.bucket,
      ref: credentials.ref || `${credentials.bucket}/${credentials.key}`,
      storage: credentials.storage, // ✨ 存储引擎类型
      uploader, // ✨ 上传器实例，用于取消上传
      // ✨ 新增：文件信息（用于批量complete）
      fileInfo: {
        key: credentials.key,
        bucket: credentials.bucket,
        ref: credentials.ref || `${credentials.bucket}/${credentials.key}`,
        router,
        file_name: file.name,
        file_size: file.size,
        content_type: file.type || '',
        hash, // ✨ 添加文件hash
      },
    }
    
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    // 上传失败，返回错误信息（不立即调用complete，由调用方统一处理）
    throw {
      ...(isRecord(error) ? error : {}),
      fileInfo: {
        key: credentials.key,
        bucket: credentials.bucket,
        ref: credentials.ref || `${credentials.bucket}/${credentials.key}`,
        router,
        file_name: file.name,
        file_size: file.size,
        content_type: file.type || '',
        hash, // ✨ 添加文件hash（即使上传失败也记录）
        error: message,
      },
    }
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

/**
 * 获取上传凭证
 * 
 * 请求后端 API，后端会返回：
 * {
 *   method: "presigned_url",
 *   upload_url: "http://localhost:9000/...",
 *   upload_host: "localhost:9000",      // ✨ 上传目标 host
 *   upload_domain: "http://localhost:9000", // ✨ 上传完整域名
 *   headers: {...},
 *   ...
 * }
 */
async function getUploadCredentials(router: string, file: File, options: UploadFileOptions = {}): Promise<UploadCredentials> {
  // ✅ 处理文件类型（某些文件如 .dmg 可能没有 MIME 类型）
  const contentType = file.type || 'application/octet-stream'
  const publicShareId = getCurrentPublicShareId()
  if (publicShareId) {
    await ensurePublicAnonymousToken()
  }
  const endpoint = publicShareId
    ? `/storage/api/v1/public/share/${encodeURIComponent(publicShareId)}/upload_token`
    : '/storage/api/v1/upload_token'
  
  const res = await authFetch(endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(publicShareId ? publicShareAnonymousHeaders() : {}),
    },
    body: JSON.stringify({
      router,
      file_name: file.name,
      content_type: contentType,
      file_size: file.size,
      preview_for_key: options.previewForKey || undefined,
    }),
  })
  
  if (!res.ok) {
    const errorText = await res.text()
    throw new Error(`获取上传凭证失败: ${res.statusText} - ${errorText}`)
  }
  
  const response = await res.json()
  
  // ✅ 先检查业务错误（code !== 0 表示后端返回了错误）
  // 🔥 统一使用 msg 字段
  if (response.code !== undefined && response.code !== 0) {
    const errorMsg = response.msg || '上传失败'
    throw new Error(errorMsg)
  }
  
  // ✅ 验证响应结构（只有在成功时才检查）
  if (!response.data) {
    throw new Error('后端返回数据格式错误: 缺少 data 字段')
  }
  
  const data = response.data as UploadCredentialsResponseData
  
  // ✅ 验证必需字段（检查多种可能的字段名）
  const method = data.method || data.Method
  
  if (!method) {
    throw new Error(`后端未返回上传方式 (method 字段)，当前值: ${data.method}`)
  }
  
  // ✅ 确保 method 字段存在
  data.method = method
  
  return data
}

/**
 * 通知后端上传完成（单个文件）
 * ✅ 返回下载 URL（如果上传成功）
 */
interface UploadCompleteParams {
  key: string
  bucket?: string
  success: boolean
  error?: string
  router: string
  file_name: string
  description?: string
  file_size: number
  content_type: string
  hash?: string
  storage?: string      // ⭐ 存储引擎类型
  upload_user?: string  // 🔥 上传用户
  thumbnail_ref?: string
  preview_kind?: string
}

// ⭐ 上传完成响应（完整文件信息）
export interface UploadCompleteResult {
  download_url: string              // 外部访问的下载地址（前端使用）
  server_download_url?: string      // 内部访问的下载地址（服务端使用）
  key: string                       // 文件 Key
  bucket?: string                   // 存储桶
  ref?: string                      // 稳定文件引用：bucket/object_key
  file_name: string                  // 文件名
  description?: string               // 文件描述
  file_size: number                  // 文件大小
  content_type: string               // 文件类型
  hash?: string                      // 文件hash
  storage?: string                   // 存储引擎类型
  thumbnail_ref?: string
  thumbnail_url?: string
  preview_kind?: string
}

export async function notifyUploadComplete(params: UploadCompleteParams): Promise<UploadCompleteResult | null> {
  try {
    const publicShareId = getCurrentPublicShareId()
    if (publicShareId) {
      await ensurePublicAnonymousToken()
    }
    const endpoint = publicShareId
      ? `/storage/api/v1/public/share/${encodeURIComponent(publicShareId)}/upload_complete`
      : '/storage/api/v1/upload_complete'
    const res = await authFetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(publicShareId ? publicShareAnonymousHeaders() : {}),
      },
      body: JSON.stringify({
        key: params.key,
        bucket: params.bucket,
        success: params.success,
        error: params.error,
        router: params.router,
        file_name: params.file_name,
        description: params.description || '',
        file_size: params.file_size,
        content_type: params.content_type,
        hash: params.hash,
        upload_user: params.upload_user,  // 🔥 传递上传用户
        thumbnail_ref: params.thumbnail_ref || '',
        preview_kind: params.preview_kind || '',
      }),
    })
    
    if (!res.ok) {
      return null
    }
    
    const response = await res.json()
    
    // ✅ 返回完整文件信息（如果上传成功）
    if (params.success && response.data?.download_url) {
      return {
        download_url: response.data.download_url,
        server_download_url: response.data.server_download_url || '',
        key: params.key,
        bucket: response.data.bucket || params.bucket || '',
        ref: response.data.ref || '',
        file_name: params.file_name,
        description: response.data.description || '',
        file_size: params.file_size,
        content_type: params.content_type,
        hash: params.hash || '',
        storage: params.storage, // ⭐ 从参数中获取存储引擎类型
        thumbnail_ref: response.data.thumbnail_ref || params.thumbnail_ref || '',
        thumbnail_url: response.data.thumbnail_url || '',
        preview_kind: response.data.preview_kind || params.preview_kind || '',
      }
    }
    
    return null
  } catch (_err) {
    return null
  }
}

/**
 * 批量通知后端上传完成
 * ✅ 返回所有文件的下载 URL（如果上传成功）
 */
export interface BatchUploadCompleteItem {
  key: string
  bucket?: string
  success: boolean
  error?: string
  router: string
  file_name: string
  description?: string
  file_size: number
  content_type: string
  hash?: string
  upload_user?: string  // 🔥 上传用户
  thumbnail_ref?: string
  preview_kind?: string
}

export interface BatchUploadCompleteResult {
  key: string
  bucket?: string
  ref?: string
  status: string
  download_url?: string      // ✨ 外部访问的下载地址（前端使用）
  description?: string       // 文件描述
  server_download_url?: string // ✨ 内部访问的下载地址（服务端使用）
  hash?: string              // ✨ 文件hash（用于 SDK 下载缓存）
  thumbnail_ref?: string
  thumbnail_url?: string
  preview_kind?: string
  error?: string
}

export async function notifyBatchUploadComplete(
  items: BatchUploadCompleteItem[]
): Promise<Map<string, BatchUploadCompleteResult>> {
  const results = new Map<string, BatchUploadCompleteResult>()
  
  if (items.length === 0) {
    return results
  }
  
  try {
    const publicShareId = getCurrentPublicShareId()
    if (publicShareId) {
      await ensurePublicAnonymousToken()
    }
    const endpoint = publicShareId
      ? `/storage/api/v1/public/share/${encodeURIComponent(publicShareId)}/batch_upload_complete`
      : '/storage/api/v1/batch_upload_complete'
    const res = await authFetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(publicShareId ? publicShareAnonymousHeaders() : {}),
      },
      body: JSON.stringify({ items }),
    })
    
    if (!res.ok) {
      // 如果批量接口失败，返回所有失败的结果
      items.forEach(item => {
        results.set(item.key, {
          key: item.key,
          status: 'failed',
          error: '批量通知失败',
        })
      })
      return results
    }
    
    const response = await res.json()
    
    // ✅ 处理批量响应结果
    if (response.data?.results && Array.isArray(response.data.results)) {
      response.data.results.forEach((result: BatchUploadCompleteResult) => {
        results.set(result.key, result)
      })
    }
    
    return results
  } catch (_err) {
    // 如果批量接口出错，返回所有失败的结果
    items.forEach(item => {
      results.set(item.key, {
        key: item.key,
        status: 'failed',
        error: '批量通知失败',
      })
    })
    return results
  }
}

/**
 * 计算文件的 SHA256 hash（用于文件标识与下载缓存）
 * 优先使用 Web Crypto API；非 HTTPS 等无 crypto.subtle 时用 js-sha256 兜底，尽量保证有 hash
 * @param file 文件对象
 * @returns SHA256 hash 字符串（十六进制）
 */
async function calculateSHA256(file: File): Promise<string> {
  const arrayBuffer = await file.arrayBuffer()
  const bytes = new Uint8Array(arrayBuffer)

  // 优先使用原生 Web Crypto（安全上下文：HTTPS / localhost）
  if (crypto?.subtle?.digest) {
    try {
      const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
      return Array.from(new Uint8Array(hashBuffer))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('')
    } catch (e) {
      console.warn('[calculateSHA256] crypto.subtle.digest failed, fallback to js-sha256:', e)
    }
  } else {
    console.warn('[calculateSHA256] crypto.subtle not available (e.g. non-HTTPS), using js-sha256 fallback')
  }

  // 兜底：纯 JS SHA256（非 HTTPS 或 digest 失败时）
  const { sha256 } = await import('js-sha256')
  return sha256(bytes)
}
