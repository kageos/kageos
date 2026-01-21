/**
 * 统一文件上传工具
 * 支持多种存储后端（MinIO、腾讯云 COS、阿里云 OSS、AWS S3、七牛云等）
 * 
 * 注意：Hub 通过网关访问 OS 的存储服务，API 路径为 /api/v1/storage/...
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
    if (!method) {
      console.error('[UploaderFactory] method 为 undefined 或空字符串')
      throw new Error(`不支持的上传方式: ${method}（method 字段为空）`)
    }
    
    const methodLower = String(method).toLowerCase().trim()
    
    switch (methodLower) {
      case 'presigned_url':
        return new PresignedURLUploader()
      
      case 'form_upload':
        return new FormUploader()
      
      case 'sdk_upload':
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
  uploader: Uploader
  fileInfo?: FileInfo
}

/**
 * 统一上传入口
 */
export async function uploadFile(
  router: string,
  file: File,
  onProgress: (progress: UploadProgress & { uploadDomain?: string }) => void
): Promise<UploadFileResult> {
  
  // 计算文件 SHA256 hash
  let hash = ''
  try {
    hash = await calculateSHA256(file)
    if (!hash) {
      console.warn('[uploadFile] Hash calculation returned empty string for file:', file.name)
    }
  } catch (error) {
    console.error('[uploadFile] Hash calculation failed for file:', file.name, error)
  }
  
  // 获取上传凭证（通过网关访问 OS 的存储服务）
  const credentials = await getUploadCredentials(router, file)
  
  if (!credentials.method) {
    throw new Error(`上传凭证缺少 method 字段，无法创建上传器`)
  }
  
  const uploader = UploaderFactory.create(credentials.method)
  
  try {
    const progressWrapper = (progress: UploadProgress) => {
      onProgress({
        ...progress,
        uploadDomain: credentials.upload_domain,
      })
    }
    
    await uploader.upload(credentials, file, progressWrapper)
    
    return {
      downloadURL: '',
      key: credentials.key,
      storage: credentials.storage,
      uploader,
      fileInfo: {
        key: credentials.key,
        router,
        file_name: file.name,
        file_size: file.size,
        content_type: file.type || '',
        hash,
      },
    }
    
  } catch (error: any) {
    throw {
      ...error,
      fileInfo: {
        key: credentials.key,
        router,
        file_name: file.name,
        file_size: file.size,
        content_type: file.type || '',
        hash,
        error: error.message,
      },
    }
  }
}

/**
 * 获取上传凭证（通过网关访问 OS 的存储服务）
 */
async function getUploadCredentials(router: string, file: File): Promise<UploadCredentials> {
  const token = localStorage.getItem('token') || ''
  
  const contentType = file.type || 'application/octet-stream'
  
  // 🔥 Hub 通过网关访问 OS 的存储服务
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
  
  if (response.code !== undefined && response.code !== 0) {
    const errorMsg = response.msg || response.message || '上传失败'
    throw new Error(errorMsg)
  }
  
  if (!response.data) {
    throw new Error('后端返回数据格式错误: 缺少 data 字段')
  }
  
  const data = response.data
  
  const method = data.method || data.Method || (data as any).method
  
  if (!method) {
    throw new Error(`后端未返回上传方式 (method 字段)，当前值: ${data.method}`)
  }
  
  data.method = method
  
  return data
}

/**
 * 通知后端上传完成（单个文件）
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
  upload_user?: string
}

export async function notifyUploadComplete(params: UploadCompleteParams): Promise<string | null> {
  const token = localStorage.getItem('token') || ''
  
  try {
    // 🔥 Hub 通过网关访问 OS 的存储服务
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
        upload_user: params.upload_user,
      }),
    })
    
    if (!res.ok) {
      return null
    }
    
    const response = await res.json()
    
    if (params.success && response.data?.download_url) {
      return response.data.download_url
    }
    
    return null
  } catch (err) {
    return null
  }
}

/**
 * 计算文件的 SHA256 hash
 */
async function calculateSHA256(file: File): Promise<string> {
  const arrayBuffer = await file.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
  return hashHex
}

