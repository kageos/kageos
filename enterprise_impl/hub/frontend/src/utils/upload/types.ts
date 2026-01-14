/**
 * 上传相关类型定义
 */

/**
 * 上传方式
 */
export type UploadMethod = 'presigned_url' | 'form_upload' | 'sdk_upload'

/**
 * 上传凭证
 */
export interface UploadCredentials {
  // 通用字段
  key: string           // 文件 Key
  bucket: string        // 存储桶
  expire: string        // 过期时间
  method: UploadMethod  // 上传方式
  storage?: string      // ✨ 存储引擎（minio/qiniu/tencentcos/aliyunoss/awss3）
  
  // 预签名 URL 上传（MinIO、COS、OSS、S3）
  url?: string                      // 预签名 URL
  headers?: Record<string, string>  // 请求头
  
  // 上传域名信息 ✨ 新增
  upload_host?: string   // 上传目标 host（例如：localhost:9000，用于 CORS、进度监听）
  upload_domain?: string // 上传完整域名（例如：http://localhost:9000，用于日志、调试）
  
  // 表单上传（七牛云、又拍云等）
  form_data?: Record<string, string>  // 表单字段
  post_url?: string                   // POST 地址
  
  // SDK 上传（特殊云存储）
  sdk_config?: Record<string, any>  // SDK 配置
  
  // CDN 域名（用于下载访问）
  cdn_domain?: string
}

/**
 * 上传进度
 */
export interface UploadProgress {
  percent: number     // 进度百分比（0-100）
  loaded: number      // 已上传字节数
  total: number       // 文件总大小
  speed?: string      // 上传速度（可选）
}

/**
 * 上传结果
 */
export interface UploadResult {
  downloadURL: string  // 下载 URL
  key: string          // 文件 Key
  storage?: string     // ✨ 存储引擎类型（minio/qiniu/tencentcos/aliyunoss/awss3）
}

