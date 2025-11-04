/**
 * SDK 上传器
 * 适用于：需要使用特定 SDK 的云存储
 */

import type { Uploader } from './index'
import type { UploadCredentials, UploadProgress } from './types'

export class SDKUploader implements Uploader {
  async upload(
    credentials: UploadCredentials,
    file: File,
    onProgress: (progress: UploadProgress) => void
  ): Promise<void> {
    // TODO: 根据 sdk_config 动态加载对应的 SDK
    // 例如：七牛云 SDK、又拍云 SDK 等
    throw new Error('SDK 上传暂未实现，请使用其他上传方式')
  }

  cancel(): void {
    // TODO: 取消 SDK 上传
  }
}

