/**
 * 预签名 URL 上传器
 * 适用于：MinIO、腾讯云 COS、阿里云 OSS、AWS S3
 */

import type { Uploader } from './index'
import type { UploadCredentials, UploadProgress } from './types'

export class PresignedURLUploader implements Uploader {
  private xhr: XMLHttpRequest | null = null
  private startTime = 0

  async upload(
    credentials: UploadCredentials,
    file: File,
    onProgress: (progress: UploadProgress) => void
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      this.xhr = new XMLHttpRequest()
      this.startTime = Date.now()

      // 监听上传进度
      this.xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          const percent = Math.round((e.loaded / e.total) * 100)
          const speed = this.calculateSpeed(e.loaded)
          
          onProgress({
            percent,
            loaded: e.loaded,
            total: e.total,
            speed,
          })
        }
      })

      // 上传完成
      this.xhr.addEventListener('load', () => {
        if (this.xhr!.status === 200) {
          resolve()
        } else {
          reject(new Error(`上传失败: ${this.xhr!.statusText}`))
        }
      })

      // 上传失败
      this.xhr.addEventListener('error', () => {
        reject(new Error('网络错误'))
      })

      // 上传中断
      this.xhr.addEventListener('abort', () => {
        reject(new Error('上传已取消'))
      })

      // 发起上传（HTTP PUT）
      // presigned URL 可能指向内部地址（如 localhost:9000），需要替换为当前浏览器 origin
      // 通过 Nginx 反向代理 /ai-agent-os/ 转发到 MinIO
      if (!credentials.upload_url) {
        reject(new Error('上传凭证缺少 upload_url'))
        return
      }

      const uploadUrl = this.rewritePresignedUrl(credentials.upload_url)
      this.xhr.open('PUT', uploadUrl)
      
      // 设置请求头
      if (credentials.headers) {
        Object.entries(credentials.headers).forEach(([key, value]) => {
          this.xhr!.setRequestHeader(key, value)
        })
      }
      
      // ✨ 如果跨域，可以设置 CORS 相关头部（如果需要）
      // 注意：预签名 URL 已经包含了必要的认证信息，通常不需要额外设置
      
      this.xhr.send(file)
    })
  }

  cancel(): void {
    if (this.xhr) {
      this.xhr.abort()
      this.xhr = null
    }
  }

  private rewritePresignedUrl(url: string): string {
    try {
      const parsed = new URL(url)
      const currentOrigin = window.location.origin
      const urlOrigin = parsed.origin
      // 如果 presigned URL 的 origin 与浏览器不同（如内部地址 localhost:9000），
      // 替换为当前 origin，走 Nginx 代理
      if (urlOrigin !== currentOrigin) {
        return `${currentOrigin}${parsed.pathname}${parsed.search}`
      }
    } catch {
      // URL 解析失败，原样返回
    }
    return url
  }

  private calculateSpeed(loaded: number): string {
    if (!this.startTime || loaded === 0) return '0 KB/s'
    
    const elapsed = (Date.now() - this.startTime) / 1000  // 秒
    const speed = loaded / elapsed  // 字节/秒
    
    if (speed < 1024) return `${speed.toFixed(0)} B/s`
    if (speed < 1024 * 1024) return `${(speed / 1024).toFixed(2)} KB/s`
    return `${(speed / (1024 * 1024)).toFixed(2)} MB/s`
  }
}
