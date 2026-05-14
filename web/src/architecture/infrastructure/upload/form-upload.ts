/**
 * 表单上传器
 * 适用于：七牛云、又拍云等使用表单上传的云存储
 */

import type { Uploader } from './index'
import type { UploadCredentials, UploadProgress } from './types'

export class FormUploader implements Uploader {
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

      // 构建表单数据
      const formData = new FormData()
      
      // 添加云存储要求的表单字段
      if (credentials.form_data) {
        Object.entries(credentials.form_data).forEach(([key, value]) => {
          formData.append(key, value)
        })
      }
      
      // 添加文件（通常字段名为 'file'）
      formData.append('file', file)

      // 发起上传（HTTP POST）
      this.xhr.open('POST', credentials.post_url!)
      this.xhr.send(formData)
    })
  }

  cancel(): void {
    if (this.xhr) {
      this.xhr.abort()
      this.xhr = null
    }
  }

  private calculateSpeed(loaded: number): string {
    if (!this.startTime || loaded === 0) return '0 KB/s'
    
    const elapsed = (Date.now() - this.startTime) / 1000
    const speed = loaded / elapsed
    
    if (speed < 1024) return `${speed.toFixed(0)} B/s`
    if (speed < 1024 * 1024) return `${(speed / 1024).toFixed(2)} KB/s`
    return `${(speed / (1024 * 1024)).toFixed(2)} MB/s`
  }
}

