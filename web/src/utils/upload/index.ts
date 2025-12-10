/**
 * 统一文件上传工具
 * 支持多种存储后端（MinIO、腾讯云 COS、阿里云 OSS、AWS S3、七牛云等）
 */

import type { UploadCredentials, UploadProgress, UploadResult } from './types'
import { PresignedURLUploader } from './presigned-url'
import { FormUploader } from './form-upload'
import { SDKUploader } from './sdk-upload'

/**
 * 文件上传器接口（策略模式）
 */
export interface Uploader {
  upload(
    credentials: UploadCredentials,
    file: File,
    onProgress: (progress: UploadProgress) => void
  ): Promise<void>
  
  cancel(): void
}

/**
 * 上传器工厂
 */
export class UploaderFactory {
  static create(method: string | undefined): Uploader {
    // ✅ 验证 method 参数
    if (!method) {
      console.error('[UploaderFactory] method 为 undefined 或空字符串')
      throw new Error(`不支持的上传方式: ${method}（method 字段为空）`)
    }
    
    // ✅ 转换为字符串并转为小写（防止大小写不一致）
    const methodLower = String(method).toLowerCase().trim()
    
    switch (methodLower) {
      case 'presigned_url':
        // 预签名 URL 上传（MinIO、COS、OSS、S3）
        return new PresignedURLUploader()
      
      case 'form_upload':
        // 表单上传（七牛云、又拍云等）
        return new FormUploader()
      
      case 'sdk_upload':
        // SDK 上传（特殊云存储）
        return new SDKUploader()
      
      default:
        console.error('[UploaderFactory] 未知的上传方式:', {
          original: method,
          normalized: methodLower,
          type: typeof method,
        })
        throw new Error(`不支持的上传方式: ${method}（支持的方式: presigned_url, form_upload, sdk_upload）`)
    }
  }
}

/**
 * 文件信息（用于批量complete）
 */
export interface FileInfo {
  key: string
  router: string
  file_name: string
  file_size: number
  content_type: string
  hash?: string
  error?: string
}

/**
 * 上传结果（包含上传器实例）
 */
export interface UploadFileResult extends UploadResult {
  uploader: Uploader  // ✨ 上传器实例，用于取消上传
  fileInfo?: FileInfo // ✨ 文件信息（用于批量complete）
}

/**
 * 统一上传入口
 * 
 * 流程：
 * 1. 用户拖文件/选择文件
 * 2. 调用此函数 → 先请求后端获取上传凭证（包含域名）
 * 3. 后端返回：{ method, url, upload_host, upload_domain, storage, ... }
 * 4. 根据 method 创建对应的上传器
 * 5. 使用上传器执行上传（此时已知道上传域名）
 * 
 * ✨ 返回上传器实例，支持取消上传
 */
export async function uploadFile(
  router: string,
  file: File,
  onProgress: (progress: UploadProgress & { uploadDomain?: string }) => void
): Promise<UploadFileResult> {
  
  // ✨ Step 0: 计算文件 SHA256 hash（用于秒传和去重）
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
  const credentials = await getUploadCredentials(router, file)
  
  // ✨ Step 2: 根据上传方式创建对应的上传器
  // 此时已知道上传方式（presigned_url / form_upload / sdk_upload）
  if (!credentials.method) {
    throw new Error(`上传凭证缺少 method 字段，无法创建上传器`)
  }
  
  const uploader = UploaderFactory.create(credentials.method)
  
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
      storage: credentials.storage, // ✨ 存储引擎类型
      uploader, // ✨ 上传器实例，用于取消上传
      // ✨ 新增：文件信息（用于批量complete）
      fileInfo: {
        key: credentials.key,
        router,
        file_name: file.name,
        file_size: file.size,
        content_type: file.type || '',
        hash, // ✨ 添加文件hash
      },
    }
    
  } catch (error: any) {
    // 上传失败，返回错误信息（不立即调用complete，由调用方统一处理）
    throw {
      ...error,
      fileInfo: {
        key: credentials.key,
        router,
        file_name: file.name,
        file_size: file.size,
        content_type: file.type || '',
        hash, // ✨ 添加文件hash（即使上传失败也记录）
        error: error.message,
      },
    }
  }
}

/**
 * 获取上传凭证
 * 
 * 请求后端 API，后端会返回：
 * {
 *   method: "presigned_url",
 *   url: "http://localhost:9000/...",
 *   upload_host: "localhost:9000",      // ✨ 上传目标 host
 *   upload_domain: "http://localhost:9000", // ✨ 上传完整域名
 *   headers: {...},
 *   ...
 * }
 */
async function getUploadCredentials(router: string, file: File): Promise<UploadCredentials> {
  const token = localStorage.getItem('token') || ''
  
  // ✅ 处理文件类型（某些文件如 .dmg 可能没有 MIME 类型）
  const contentType = file.type || 'application/octet-stream'
  
  const res = await fetch('/api/v1/storage/upload_token', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token': token,
    },
    body: JSON.stringify({
      router,
      file_name: file.name,
      content_type: contentType,
      file_size: file.size,
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
  
  const data = response.data
  
  // ✅ 验证必需字段（检查多种可能的字段名）
  const method = data.method || data.Method || (data as any).method
  
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
  success: boolean
  error?: string
  router: string
  file_name: string
  file_size: number
  content_type: string
  hash?: string
  upload_user?: string  // 🔥 上传用户
}

export async function notifyUploadComplete(params: UploadCompleteParams): Promise<string | null> {
  const token = localStorage.getItem('token') || ''
  
  try {
    const res = await fetch('/api/v1/storage/upload_complete', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Token': token,
      },
      body: JSON.stringify({
        key: params.key,
        success: params.success,
        error: params.error,
        router: params.router,
        file_name: params.file_name,
        file_size: params.file_size,
        content_type: params.content_type,
        hash: params.hash,
        upload_user: params.upload_user,  // 🔥 传递上传用户
      }),
    })
    
    if (!res.ok) {
      return null
    }
    
    const response = await res.json()
    
    // ✅ 返回下载 URL（如果上传成功）
    if (params.success && response.data?.download_url) {
      return response.data.download_url
    }
    
    return null
  } catch (err) {
    return null
  }
}

/**
 * 批量通知后端上传完成
 * ✅ 返回所有文件的下载 URL（如果上传成功）
 */
export interface BatchUploadCompleteItem {
  key: string
  success: boolean
  error?: string
  router: string
  file_name: string
  file_size: number
  content_type: string
  hash?: string
  upload_user?: string  // 🔥 上传用户
}

export interface BatchUploadCompleteResult {
  key: string
  status: string
  download_url?: string      // ✨ 外部访问的下载地址（前端使用）
  server_download_url?: string // ✨ 内部访问的下载地址（服务端使用）
  hash?: string              // ✨ 文件hash（用于文件缓存去重）
  error?: string
}

export async function notifyBatchUploadComplete(
  items: BatchUploadCompleteItem[]
): Promise<Map<string, BatchUploadCompleteResult>> {
  const token = localStorage.getItem('token') || ''
  const results = new Map<string, BatchUploadCompleteResult>()
  
  if (items.length === 0) {
    return results
  }
  
  try {
    const res = await fetch('/api/v1/storage/batch_upload_complete', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Token': token,
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
  } catch (err) {
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
 * 计算文件的 SHA256 hash
 * @param file 文件对象
 * @returns SHA256 hash 字符串（十六进制）
 */
async function calculateSHA256(file: File): Promise<string> {
  const arrayBuffer = await file.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
  return hashHex
}
