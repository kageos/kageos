import { ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  generateFilePreview,
  isFileAccepted,
  notifyBatchUploadComplete,
  uploadFile,
  type BatchUploadCompleteItem,
  type FileInfo,
  type UploadFileResult,
  type UploadProgress,
  type Uploader,
} from '@/architecture/presentation/context/uploadContext'
import { Logger } from '@/architecture/shared/logger'
import type { FileItem } from '../filesWidgetTypes'

export interface UploadingFile {
  uid: string
  name: string
  size: number
  percent: number
  status: 'uploading' | 'success' | 'error'
  error?: string
  speed?: string
  rawFile?: File
  uploader?: Uploader
  cancel?: () => void
  retry?: () => void
  fileInfo?: FileInfo
  downloadURL?: string
  storage?: string
}

interface UseFilesUploadManagerOptions {
  mode: () => string
  router: () => string
  accept: Ref<string>
  maxSize: Ref<string | undefined>
  maxCount: Ref<number>
  thumbnail: () => boolean
  currentFiles: Ref<FileItem[]>
  updateFiles: (files: FileItem[]) => Promise<void>
  resolveUploadUser: () => string
}

const BATCH_COMPLETE_DELAY = 500
const BATCH_COMPLETE_MAX_SIZE = 10

export function useFilesUploadManager(options: UseFilesUploadManagerOptions) {
  const isDragging = ref(false)
  const isHandlingDrop = ref(false)
  const uploadingFiles = ref<UploadingFile[]>([])
  const pendingCompleteQueue = ref<BatchUploadCompleteItem[]>([])
  const batchCompleteTimer = ref<ReturnType<typeof setTimeout> | null>(null)

  function parseMaxSize(maxSizeStr?: string): number {
    if (!maxSizeStr) return Infinity

    const units: Record<string, number> = {
      B: 1,
      KB: 1024,
      MB: 1024 * 1024,
      GB: 1024 * 1024 * 1024,
    }

    const match = maxSizeStr.match(/^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB)$/i)
    if (!match || !match[1] || !match[2]) {
      Logger.error('[FilesWidget]', `Invalid max_size format: ${maxSizeStr}`)
      return Infinity
    }

    const unit = match[2].toUpperCase() as keyof typeof units
    const unitValue = units[unit]
    if (!unitValue) {
      Logger.error('[FilesWidget]', `Unknown unit: ${unit}`)
      return Infinity
    }

    return parseFloat(match[1]) * unitValue
  }

  function validateFile(file: File): boolean {
    const maxSizeBytes = parseMaxSize(options.maxSize.value)
    const currentFilesCount = options.currentFiles.value.length
    const uploadingCount = uploadingFiles.value.length
    const totalCount = currentFilesCount + uploadingCount

    if (totalCount >= options.maxCount.value) {
      ElMessage.error(`最多只能上传 ${options.maxCount.value} 个文件，当前已有 ${currentFilesCount} 个文件，正在上传 ${uploadingCount} 个文件`)
      return false
    }

    if (file.size > maxSizeBytes) {
      ElMessage.error(`文件大小不能超过 ${options.maxSize.value}`)
      return false
    }

    if (!isFileAccepted(file, options.accept.value)) {
      ElMessage.error(`不支持的文件类型，仅支持：${options.accept.value}`)
      return false
    }

    return true
  }

  async function flushCompleteQueue(): Promise<void> {
    if (pendingCompleteQueue.value.length === 0) {
      return
    }

    const items = [...pendingCompleteQueue.value]
    pendingCompleteQueue.value = []

    if (batchCompleteTimer.value) {
      clearTimeout(batchCompleteTimer.value)
      batchCompleteTimer.value = null
    }

    try {
      const results = await notifyBatchUploadComplete(items)
      const newFiles: FileItem[] = []
      const completedUploadingFiles: UploadingFile[] = []

      for (const item of items) {
        const result = results.get(item.key)
        const uploadingFile = uploadingFiles.value.find((file: UploadingFile) => file.fileInfo?.key === item.key)

        if (result && item.success && result.status === 'completed') {
          if (uploadingFile && uploadingFile.fileInfo) {
            uploadingFile.downloadURL = result.download_url || ''
            const ref = result.ref || uploadingFile.fileInfo.ref || `${result.bucket || uploadingFile.fileInfo.bucket}/${item.key}`

            newFiles.push({
              ref,
              bucket: result.bucket || uploadingFile.fileInfo.bucket,
              key: item.key,
              name: uploadingFile.name,
              source_name: uploadingFile.name,
              storage: uploadingFile.storage || '',
              description: result.description || '',
              hash: result.hash || uploadingFile.fileInfo.hash || '',
              size: uploadingFile.size,
              content_type: uploadingFile.fileInfo.content_type,
              upload_ts: Date.now(),
              is_uploaded: true,
              download_url: result.download_url || '',
              server_download_url: result.server_download_url || '',
              thumbnail_ref: result.thumbnail_ref || '',
              thumbnail_url: result.thumbnail_url || '',
              preview_kind: result.preview_kind || '',
              downloaded: false,
              upload_user: item.upload_user || '',
            })
            completedUploadingFiles.push(uploadingFile)
          }
        } else if (!item.success || (result && result.status === 'failed')) {
          if (uploadingFile) {
            uploadingFile.status = 'error'
            uploadingFile.error = result?.error || item.error || '上传失败'
          }
        }
      }

      if (newFiles.length > 0) {
        await options.updateFiles([...options.currentFiles.value, ...newFiles])
        completedUploadingFiles.forEach((uploadingFile) => {
          const index = uploadingFiles.value.findIndex((file: UploadingFile) => file.uid === uploadingFile.uid)
          if (index !== -1) {
            uploadingFiles.value.splice(index, 1)
          }
        })
      }

      const successCount = items.filter(item => item.success && results.get(item.key)?.status === 'completed').length
      if (successCount > 1) {
        ElMessage.success(`批量上传完成：${successCount} 个文件`)
      } else if (successCount === 1) {
        ElMessage.success('上传成功')
      }
    } catch (error: any) {
      Logger.error('[FilesWidget]', 'Batch complete failed', error)
      items.forEach((item) => {
        const uploadingFile = uploadingFiles.value.find((file: UploadingFile) => file.fileInfo?.key === item.key)
        if (uploadingFile) {
          uploadingFile.status = 'error'
          uploadingFile.error = '批量通知失败'
        }
      })
    }
  }

  function addToCompleteQueue(item: BatchUploadCompleteItem): void {
    pendingCompleteQueue.value.push(item)

    if (pendingCompleteQueue.value.length >= BATCH_COMPLETE_MAX_SIZE) {
      void flushCompleteQueue()
      return
    }

    if (batchCompleteTimer.value) {
      clearTimeout(batchCompleteTimer.value)
    }
    batchCompleteTimer.value = setTimeout(() => {
      void flushCompleteQueue()
    }, BATCH_COMPLETE_DELAY)
  }

  async function uploadGeneratedPreview(router: string, rawFile: File, sourceKey: string): Promise<{ thumbnailRef: string; previewKind: string } | null> {
    if (!options.thumbnail()) {
      return null
    }
    if (!sourceKey) {
      return null
    }

    try {
      const preview = await generateFilePreview(rawFile)
      if (!preview) {
        return null
      }

      const previewResult = await uploadFile(router, preview.file, () => {}, { previewForKey: sourceKey })
      const thumbnailRef = previewResult.fileInfo?.ref || previewResult.ref || ''
      if (!thumbnailRef) {
        return null
      }

      return {
        thumbnailRef,
        previewKind: preview.kind,
      }
    } catch (error) {
      Logger.warn('[FilesWidget]', '生成或上传文件预览失败，已跳过预览关联', error)
      return null
    }
  }

  async function handleFileSelect(rawFile: File): Promise<void> {
    if (options.mode() !== 'edit') {
      ElMessage.error('当前模式不支持文件上传')
      return
    }

    const router = options.router()
    if (!router) {
      ElMessage.error('缺少函数路径，无法上传文件')
      return
    }

    if (!validateFile(rawFile)) {
      return
    }

    const uid = `${Date.now()}_${Math.random().toString(36).slice(2)}`

    const uploadingFile: UploadingFile = {
      uid,
      name: rawFile.name,
      size: rawFile.size,
      percent: 0,
      status: 'uploading',
      speed: '0 KB/s',
      rawFile,
    }

    uploadingFile.cancel = () => {
      if (uploadingFile.uploader) {
        uploadingFile.uploader.cancel()
        uploadingFile.status = 'error'
        uploadingFile.error = '上传已取消'
        ElMessage.warning('上传已取消')
        const index = uploadingFiles.value.findIndex((file: UploadingFile) => file.uid === uid)
        if (index !== -1) {
          uploadingFiles.value.splice(index, 1)
        }
      }
    }

    uploadingFile.retry = () => {
      if (uploadingFile.rawFile) {
        uploadingFile.status = 'uploading'
        uploadingFile.percent = 0
        uploadingFile.error = undefined
        uploadingFile.speed = '0 KB/s'
        void handleFileSelect(uploadingFile.rawFile)
      }
    }

    uploadingFiles.value.push(uploadingFile)

    try {
      const uploadResult: UploadFileResult = await uploadFile(
        router,
        rawFile,
        (progress: UploadProgress) => {
          const file = uploadingFiles.value.find((item: UploadingFile) => item.uid === uid)
          if (file) {
            file.percent = progress.percent
            file.speed = progress.speed || '0 KB/s'
          }
        }
      )

      uploadingFile.uploader = uploadResult.uploader
      uploadingFile.fileInfo = uploadResult.fileInfo
      uploadingFile.storage = uploadResult.storage

      const file = uploadingFiles.value.find((item: UploadingFile) => item.uid === uid)
      if (file) {
        file.status = 'success'
      }

      if (uploadResult.fileInfo) {
        const previewMeta = await uploadGeneratedPreview(router, rawFile, uploadResult.fileInfo.key)

        if (!uploadResult.fileInfo.hash) {
          Logger.warn('[FilesWidget]', `File ${uploadResult.fileInfo.file_name} has no hash`, {
            key: uploadResult.fileInfo.key,
            fileInfo: uploadResult.fileInfo,
          })
        }
        addToCompleteQueue({
          key: uploadResult.fileInfo.key,
          bucket: uploadResult.fileInfo.bucket,
          success: true,
          router: uploadResult.fileInfo.router,
          file_name: uploadResult.fileInfo.file_name,
          file_size: uploadResult.fileInfo.file_size,
          content_type: uploadResult.fileInfo.content_type,
          hash: uploadResult.fileInfo.hash || '',
          thumbnail_ref: previewMeta?.thumbnailRef || '',
          preview_kind: previewMeta?.previewKind || '',
          upload_user: options.resolveUploadUser(),
        })
      }
    } catch (error: any) {
      Logger.error('[FilesWidget]', 'Upload failed', error)

      const file = uploadingFiles.value.find((item: UploadingFile) => item.uid === uid)
      if (file) {
        file.status = 'error'
        file.error = error.message || '上传失败'
      }

      if (error.fileInfo) {
        addToCompleteQueue({
          key: error.fileInfo.key,
          bucket: error.fileInfo.bucket,
          success: false,
          error: error.fileInfo.error || error.message || '上传失败',
          router: error.fileInfo.router,
          file_name: error.fileInfo.file_name,
          file_size: error.fileInfo.file_size,
          content_type: error.fileInfo.content_type,
          hash: error.fileInfo.hash,
          upload_user: options.resolveUploadUser(),
        })
      }

      ElMessage.error(`上传失败: ${error.message || '未知错误'}`)
    }
  }

  function handleDragOver(): void {
    isDragging.value = true
  }

  function handleDragLeave(): void {
    isDragging.value = false
  }

  function handleDrop(e: DragEvent): void {
    e.preventDefault()
    e.stopPropagation()
    isDragging.value = false
    isHandlingDrop.value = true

    const files = e.dataTransfer?.files
    if (files && files.length > 0) {
      Array.from(files).forEach((file) => {
        void handleFileSelect(file)
      })
    }

    setTimeout(() => {
      isHandlingDrop.value = false
    }, 100)
  }

  function handleElUploadDrop(e: DragEvent): void {
    e.preventDefault()
    e.stopPropagation()
  }

  function handleFileChange(file: any): void {
    if (isHandlingDrop.value) {
      return
    }

    if (file.raw) {
      void handleFileSelect(file.raw)
    }
  }

  return {
    isDragging,
    isHandlingDrop,
    uploadingFiles,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleElUploadDrop,
    handleFileChange,
  }
}
