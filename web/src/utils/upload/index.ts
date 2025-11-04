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
 * 统一上传入口
 * 
 * 流程：
 * 1. 用户拖文件/选择文件
 * 2. 调用此函数 → 先请求后端获取上传凭证（包含域名）
 * 3. 后端返回：{ method, url, upload_host, upload_domain, storage, ... }
 * 4. 根据 method 创建对应的上传器
 * 5. 使用上传器执行上传（此时已知道上传域名）
 */
export async function uploadFile(
  router: string,
  file: File,
  onProgress: (progress: UploadProgress & { uploadDomain?: string }) => void
): Promise<UploadResult> {
  
  // ✨ Step 1: 获取上传凭证（包含域名信息）
  // 这一步会请求后端 API，后端会根据配置的存储类型返回对应的上传凭证
  const credentials = await getUploadCredentials(router, file)
  
  // ✨ Step 2: 根据上传方式创建对应的上传器
  // 此时已知道上传方式（presigned_url / form_upload / sdk_upload）
  if (!credentials.method) {
    console.error('[Upload] 凭证数据:', credentials)
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
    
    // Step 4: 通知后端上传成功，并获取下载 URL
    const downloadURL = await notifyUploadComplete(credentials.key, true)
    
    // ✅ 返回上传结果（包含下载 URL、key 和存储引擎类型）
    return {
      downloadURL: downloadURL || credentials.key,
      key: credentials.key,
      storage: credentials.storage, // ✨ 存储引擎类型
    }
    
  } catch (error: any) {
    // 通知后端上传失败
    await notifyUploadComplete(credentials.key, false, error.message)
    throw error
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
  
  // ✅ 调试：打印完整响应
  console.log('[Upload] 后端响应:', JSON.stringify(response, null, 2))
  
  // ✅ 先检查业务错误（code !== 0 表示后端返回了错误）
  if (response.code !== undefined && response.code !== 0) {
    const errorMsg = response.msg || response.message || '上传失败'
    console.error('[Upload] 后端返回错误:', {
      code: response.code,
      msg: errorMsg,
      data: response.data,
    })
    throw new Error(errorMsg)
  }
  
  // ✅ 验证响应结构（只有在成功时才检查）
  if (!response.data) {
    console.error('[Upload] 响应结构错误:', response)
    throw new Error('后端返回数据格式错误: 缺少 data 字段')
  }
  
  const data = response.data
  
  // ✅ 验证必需字段（检查多种可能的字段名）
  const method = data.method || data.Method || (data as any).method
  
  if (!method) {
    console.error('[Upload] 后端返回数据（缺少 method）:', data)
    console.error('[Upload] 数据类型检查:', {
      'data.method': data.method,
      'data.Method': data.Method,
      'typeof data.method': typeof data.method,
      'data keys': Object.keys(data),
    })
    throw new Error(`后端未返回上传方式 (method 字段)，当前值: ${data.method}`)
  }
  
  // ✅ 确保 method 字段存在
  data.method = method
  
  // ✨ 此时已获取到上传凭证，包含域名信息
  console.log('[Upload] 获取上传凭证成功:', {
    method: data.method,
    upload_host: data.upload_host,
    upload_domain: data.upload_domain,
    key: data.key,
  })
  
  return data
}

/**
 * 通知后端上传完成
 * ✅ 返回下载 URL（如果上传成功）
 */
async function notifyUploadComplete(key: string, success: boolean, error?: string): Promise<string | null> {
  const token = localStorage.getItem('token') || ''
  
  try {
    const res = await fetch('/api/v1/storage/upload_complete', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Token': token,
      },
      body: JSON.stringify({ key, success, error }),
    })
    
    if (!res.ok) {
      console.error('通知上传完成失败:', res.statusText)
      return null
    }
    
    const response = await res.json()
    
    // ✅ 返回下载 URL（如果上传成功）
    if (success && response.data?.download_url) {
      return response.data.download_url
    }
    
    return null
  } catch (err) {
    console.error('通知上传完成失败:', err)
    return null
  }
}
