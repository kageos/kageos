/**
 * FilesWidget - 文件上传组件
 * 支持多文件上传、拖拽上传、文件管理
 */

import { h, ref, computed } from 'vue'
import {
  ElUpload,
  ElButton,
  ElIcon,
  ElProgress,
  ElMessage,
  ElTag,
  ElPopconfirm,
  ElInput,
  ElImage,
  ElCard,
} from 'element-plus'
import {
  Upload,
  Document,
  Delete,
  View,
  Download,
  VideoPlay,
  Picture,
  Files,
  Folder,
} from '@element-plus/icons-vue'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'
import { uploadFile, notifyBatchUploadComplete, type FileInfo, type BatchUploadCompleteItem } from '@/utils/upload'
import type { UploadProgress, UploadResult } from '@/utils/upload/types'
import type { Uploader } from '@/utils/upload'
import type { UploadFileResult } from '@/utils/upload'
import { Logger } from '../utils/logger'
import { getElementPlusFormProps } from './utils/widgetHelpers'

/**
 * Files 配置
 */
export interface FilesConfig {
  accept?: string      // 文件类型限制：.pdf,.doc,.docx,image/*,video/*
  max_size?: string    // 单个文件最大大小：10MB, 1GB
  max_count?: number   // 最大文件数量，默认 5
  [key: string]: any
}

/**
 * File 数据结构（对应后端）
 */
export interface FileItem {
  name: string
  source_name?: string  // ✨ 源文件名称（上传时的原始文件名）
  storage?: string      // ✨ 存储引擎（minio/qiniu/tencentcos/aliyunoss/awss3）
  description: string
  hash: string
  size: number
  upload_ts: number
  local_path: string
  is_uploaded: boolean
  url: string           // ✨ 外部访问地址（前端下载使用）
  server_url?: string   // ✨ 内部访问地址（服务端下载使用）
  downloaded?: boolean
}

/**
 * Files 数据结构（对应后端）
 */
export interface FilesData {
  files: FileItem[]
  remark: string
  metadata: Record<string, any>
}

/**
 * 上传中的文件状态
 */
interface UploadingFile {
  uid: string
  name: string
  size: number
  percent: number
  status: 'uploading' | 'success' | 'error'
  error?: string
  speed?: string  // ✨ 上传速度
  rawFile?: File  // ✨ 原始文件，用于重试
  uploader?: Uploader  // ✨ 上传器实例，用于取消
  cancel?: () => void  // ✨ 取消上传方法
  retry?: () => void  // ✨ 重试上传方法
  fileInfo?: FileInfo  // ✨ 文件信息（用于批量complete）
  downloadURL?: string  // ✨ 下载URL（批量complete后填充）
  storage?: string  // ✨ 存储引擎类型（从uploadResult获取）
}

export class FilesWidget extends BaseWidget {
  // 常量定义
  private static readonly MAX_DISPLAY_FILES = 3  // 表格单元格最多显示的文件数量
  
  // 组件私有状态
  private uploadingFiles = ref<UploadingFile[]>([])
  private filesConfig: FilesConfig
  private router: string
  
  // ✨ 批量complete相关
  private pendingCompleteQueue: BatchUploadCompleteItem[] = []  // 待批量complete的队列
  private batchCompleteTimer: ReturnType<typeof setTimeout> | null = null  // 批量complete定时器
  private readonly BATCH_COMPLETE_DELAY = 500  // 批量complete延迟（ms），等待更多文件完成
  private readonly BATCH_COMPLETE_MAX_SIZE = 10  // 批量complete最大批次大小

  constructor(props: WidgetRenderProps) {
    super(props)

    // 解析配置（使用基类的辅助方法）
    this.filesConfig = this.getConfig<FilesConfig>()
    
    // ✅ 获取 router（如果是临时 Widget 则为空）
    this.router = this.getRouter()

    // ✅ 初始化或验证数据结构（只有在标准 Widget 时才处理）
    // 🔥 注意：响应参数场景下，initialValue 已经正确设置，不要覆盖
    if (!this.isTemporary) {
      const currentValue = this.value.value
      
      // 🔥 响应参数场景：如果已经有有效数据，不要初始化
      // 检查数据是否有效（有 files 数组）
      if (currentValue && 
          currentValue.raw && 
          typeof currentValue.raw === 'object' && 
          'files' in currentValue.raw &&
          Array.isArray((currentValue.raw as FilesData).files)) {
        // 数据有效，不需要初始化，直接返回
        return
      }
      
      // 检查是否需要初始化或修复数据结构
      // 🔥 只有当值完全为空时才初始化
      if (!currentValue || 
          currentValue.raw === null || 
          currentValue.raw === undefined) {
        // 只有在完全没有值时才初始化
        this.initializeEmptyValue()
      } else {
        // 数据无效，但不要覆盖（可能是响应参数中的有效数据）
        Logger.warn('FilesWidget', `Invalid FilesData structure for ${this.fieldPath}, but keeping original value`, currentValue)
      }
    }
  }

  /**
   * 验证是否为有效的 FilesData 结构
   */
  private isValidFilesData(data: any): boolean {
    if (!data || typeof data !== 'object') {
      return false
    }
    // 检查是否有 files 字段（数组）
    return Array.isArray(data.files)
  }

  /**
   * 获取 router
   * 临时 Widget（表格渲染）不需要 router，返回空字符串
   */
  private getRouter(): string {
    // ✅ 临时 Widget 不需要上传功能，返回空字符串
    if (!this.formRenderer) {
      return ''
    }
    
    // ✅ 从 formRenderer 获取 router（使用标准接口）
    return this.formRenderer.getFunctionRouter()
  }

  /**
   * 初始化空值
   * 🔥 注意：只在确实没有值时才调用，不要覆盖已有的值
   */
  private initializeEmptyValue(): void {
    // 🔥 检查是否已经有值（避免覆盖响应参数中的值）
    const existingValue = this.safeGetValue(this.fieldPath)
    if (existingValue && existingValue.raw && 
        typeof existingValue.raw === 'object' && 'files' in existingValue.raw &&
        Array.isArray((existingValue.raw as FilesData).files)) {
      // 已经有有效值，不初始化
      return
    }
    
    const emptyData: FilesData = {
      files: [],
      remark: '',
      metadata: {},
    }

    this.safeSetValue(this.fieldPath, {
      raw: emptyData,
      display: '0 个文件',
      meta: {},
    })
    
  }

  /**
   * 获取当前文件列表
   */
  private getCurrentFiles(): FileItem[] {
    const currentValue = this.safeGetValue(this.fieldPath)
    const data = currentValue?.raw as FilesData | null
    return data?.files || []
  }

  /**
   * 更新文件列表
   */
  private updateFiles(files: FileItem[]): void {
    const currentValue = this.safeGetValue(this.fieldPath)
    const data = (currentValue?.raw as FilesData) || {
      files: [],
      remark: '',
      metadata: {},
    }

    const newData: FilesData = {
      ...data,
      files,
    }

    this.safeSetValue(this.fieldPath, {
      raw: newData,
      display: `${files.length} 个文件`,
      meta: {},
    })
  }

  /**
   * 解析文件大小限制
   */
  private parseMaxSize(maxSizeStr?: string): number {
    if (!maxSizeStr) return Infinity

    const units: Record<string, number> = {
      B: 1,
      KB: 1024,
      MB: 1024 * 1024,
      GB: 1024 * 1024 * 1024,
    }

    const match = maxSizeStr.match(/^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB)$/i)
    if (!match || !match[1] || !match[2]) {
      Logger.error('FilesWidget', `Invalid max_size format: ${maxSizeStr}`)
      return Infinity
    }

    const size = match[1]
    const unit = match[2].toUpperCase() as keyof typeof units
    const unitValue = units[unit]
    if (!unitValue) {
      Logger.error('FilesWidget', `Unknown unit: ${unit}`)
      return Infinity
    }
    return parseFloat(size) * unitValue
  }

  /**
   * 格式化文件大小
   */
  private formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
  }

  /**
   * 验证文件
   */
  private validateFile(file: File): boolean {
    const maxSize = this.parseMaxSize(this.filesConfig.max_size)
    const maxCount = this.filesConfig.max_count || 5
    const currentFiles = this.getCurrentFiles()

    // 检查数量限制
    if (currentFiles.length >= maxCount) {
      ElMessage.error(`最多只能上传 ${maxCount} 个文件`)
      return false
    }

    // 检查大小限制
    if (file.size > maxSize) {
      ElMessage.error(`文件大小不能超过 ${this.filesConfig.max_size}`)
      return false
    }

    // 检查文件类型
    if (this.filesConfig.accept && this.filesConfig.accept !== '*') {
      const accept = this.filesConfig.accept.split(',').map(a => a.trim())
      const fileName = file.name.toLowerCase()
      const fileType = file.type.toLowerCase()

      const isAccepted = accept.some((pattern: string) => {
        // 扩展名匹配：.pdf
        if (pattern.startsWith('.')) {
          return fileName.endsWith(pattern)
        }
        // MIME 通配符：image/*
        if (pattern.includes('/*')) {
          const prefix = pattern.split('/')[0]
          return prefix && fileType && fileType.startsWith(prefix)
        }
        // MIME 类型：application/pdf
        return fileType === pattern
      })

      if (!isAccepted) {
        ElMessage.error(`不支持的文件类型，仅支持：${this.filesConfig.accept}`)
        return false
      }
    }

    return true
  }

  /**
   * 处理文件选择
   */
  private async handleFileSelect(rawFile: File): Promise<void> {
    // ✅ 临时 Widget 不支持上传
    if (this.isTemporary) {
      ElMessage.error('临时组件不支持文件上传')
      return
    }

    // ✅ 检查 router 是否存在
    if (!this.router) {
      ElMessage.error('缺少函数路径，无法上传文件')
      return
    }

    if (!this.validateFile(rawFile)) {
      return
    }

    const uid = `${Date.now()}_${Math.random().toString(36).slice(2)}`

    // 添加到上传列表
    const uploadingFile: UploadingFile = {
      uid,
      name: rawFile.name,
      size: rawFile.size,
      percent: 0,
      status: 'uploading',
      speed: '0 KB/s',
      rawFile, // ✨ 保存原始文件，用于重试
    }
    
    // ✨ 定义取消方法
    uploadingFile.cancel = () => {
      if (uploadingFile.uploader) {
        uploadingFile.uploader.cancel()
        uploadingFile.status = 'error'
        uploadingFile.error = '上传已取消'
        ElMessage.warning('上传已取消')
        // 2 秒后移除
        setTimeout(() => {
          const index = this.uploadingFiles.value.findIndex((f: UploadingFile) => f.uid === uid)
          if (index !== -1) {
            this.uploadingFiles.value.splice(index, 1)
          }
        }, 2000)
      }
    }
    
    // ✨ 定义重试方法
    uploadingFile.retry = () => {
      if (uploadingFile.rawFile) {
        // 重置状态
        uploadingFile.status = 'uploading'
        uploadingFile.percent = 0
        uploadingFile.error = undefined
        uploadingFile.speed = '0 KB/s'
        // 重新上传
        this.handleFileSelect(uploadingFile.rawFile)
      }
    }
    
    this.uploadingFiles.value.push(uploadingFile)

    try {
      // ✨ 调用统一上传工具（后端会根据配置返回对应的上传方式）
      // ✅ uploadFile 现在返回 UploadFileResult，包含 uploader 实例
      const uploadResult: UploadFileResult = await uploadFile(
        this.router,
        rawFile,
        (progress: UploadProgress) => {
          // 更新进度和速度
          const file = this.uploadingFiles.value.find((f: UploadingFile) => f.uid === uid)
          if (file) {
            file.percent = progress.percent
            file.speed = progress.speed || '0 KB/s'  // ✨ 保存上传速度
          }
        }
      )
      
      // ✨ 保存上传器实例、文件信息和存储类型
      uploadingFile.uploader = uploadResult.uploader
      uploadingFile.fileInfo = uploadResult.fileInfo
      uploadingFile.storage = uploadResult.storage

      // 上传成功，更新状态（但downloadURL暂时为空，等待批量complete）
      const file = this.uploadingFiles.value.find((f: UploadingFile) => f.uid === uid)
      if (file) {
        file.status = 'success'
      }

      // ✨ 添加到批量complete队列
      if (uploadResult.fileInfo) {
        // 调试：检查hash是否存在
        if (!uploadResult.fileInfo.hash) {
          Logger.warn('FilesWidget', `File ${uploadResult.fileInfo.file_name} has no hash`, {
            key: uploadResult.fileInfo.key,
            fileInfo: uploadResult.fileInfo,
          })
        }
        this.addToCompleteQueue({
          key: uploadResult.fileInfo.key,
          success: true,
          router: uploadResult.fileInfo.router,
          file_name: uploadResult.fileInfo.file_name,
          file_size: uploadResult.fileInfo.file_size,
          content_type: uploadResult.fileInfo.content_type,
          hash: uploadResult.fileInfo.hash || '', // 确保hash字段存在（即使为空）
        })
      }

    } catch (error: any) {
      Logger.error('FilesWidget', 'Upload failed', error)

      // 更新状态
      const file = this.uploadingFiles.value.find((f: UploadingFile) => f.uid === uid)
      if (file) {
        file.status = 'error'
        file.error = error.message || '上传失败'
      }

      // ✨ 失败的文件也添加到批量complete队列（用于记录失败）
      if (error.fileInfo) {
        this.addToCompleteQueue({
          key: error.fileInfo.key,
          success: false,
          error: error.fileInfo.error || error.message || '上传失败',
          router: error.fileInfo.router,
          file_name: error.fileInfo.file_name,
          file_size: error.fileInfo.file_size,
          content_type: error.fileInfo.content_type,
          hash: error.fileInfo.hash, // ✨ 即使失败也传递hash（如果已计算）
        })
      }

      ElMessage.error(`上传失败: ${error.message || '未知错误'}`)
    }
  }

  /**
   * ✨ 添加到批量complete队列
   */
  private addToCompleteQueue(item: BatchUploadCompleteItem): void {
    this.pendingCompleteQueue.push(item)
    
    // 如果队列达到最大批次大小，立即触发批量complete
    if (this.pendingCompleteQueue.length >= this.BATCH_COMPLETE_MAX_SIZE) {
      this.flushCompleteQueue()
      return
    }
    
    // 否则，设置延迟批量complete（等待更多文件完成）
    if (this.batchCompleteTimer) {
      clearTimeout(this.batchCompleteTimer)
    }
    this.batchCompleteTimer = setTimeout(() => {
      this.flushCompleteQueue()
    }, this.BATCH_COMPLETE_DELAY)
  }

  /**
   * ✨ 批量complete处理
   */
  private async flushCompleteQueue(): Promise<void> {
    if (this.pendingCompleteQueue.length === 0) {
      return
    }
    
    // 取出队列中的所有项目
    const items = [...this.pendingCompleteQueue]
    this.pendingCompleteQueue = []
    
    if (this.batchCompleteTimer) {
      clearTimeout(this.batchCompleteTimer)
      this.batchCompleteTimer = null
    }
    
    try {
      // ✨ 批量调用complete接口
      const results = await notifyBatchUploadComplete(items)
      
      // 更新每个文件的状态和下载URL
      items.forEach(item => {
        const result = results.get(item.key)
        const uploadingFile = this.uploadingFiles.value.find((f: UploadingFile) => f.fileInfo?.key === item.key)
        
        if (result && item.success && result.status === 'completed') {
          // 上传成功，更新下载URL并添加到文件列表
          if (uploadingFile && uploadingFile.fileInfo) {
            uploadingFile.downloadURL = result.download_url || ''
            
            const newFile: FileItem = {
              name: uploadingFile.name,
              source_name: uploadingFile.name,
              storage: uploadingFile.storage || '', // ✨ 从uploadingFile获取存储类型（从uploadResult.storage获取，后端返回）
              description: '',
              hash: result.hash || uploadingFile.fileInfo?.hash || '', // ✨ 从响应中获取hash，如果没有则从fileInfo获取
              size: uploadingFile.size,
              upload_ts: Date.now(),
              local_path: '',
              is_uploaded: true,
              url: result.download_url || '',           // ✨ 外部访问地址（前端下载使用）
              server_url: result.server_download_url || '', // ✨ 内部访问地址（服务端下载使用）
              downloaded: false,
            }
            
            const currentFiles = this.getCurrentFiles()
            this.updateFiles([...currentFiles, newFile])
            
            // 2秒后移除上传记录
            setTimeout(() => {
              const index = this.uploadingFiles.value.findIndex((f: UploadingFile) => f.uid === uploadingFile.uid)
              if (index !== -1) {
                this.uploadingFiles.value.splice(index, 1)
              }
            }, 2000)
          }
          
          // 单个文件成功时不显示消息（批量成功时统一显示）
        } else if (!item.success || (result && result.status === 'failed')) {
          // 上传失败
          if (uploadingFile) {
            uploadingFile.status = 'error'
            uploadingFile.error = result?.error || item.error || '上传失败'
          }
        }
      })
      
      // 如果所有文件都成功，显示批量成功提示
      const successCount = items.filter(item => item.success && results.get(item.key)?.status === 'completed').length
      if (successCount > 1) {
        ElMessage.success(`批量上传完成：${successCount} 个文件`)
      } else if (successCount === 1) {
        // 单个文件成功时也显示
        ElMessage.success('上传成功')
      }
      
    } catch (error: any) {
      Logger.error('FilesWidget', 'Batch complete failed', error)
      // 如果批量complete失败，标记所有文件为错误
      items.forEach(item => {
        const uploadingFile = this.uploadingFiles.value.find((f: UploadingFile) => f.fileInfo?.key === item.key)
        if (uploadingFile) {
          uploadingFile.status = 'error'
          uploadingFile.error = '批量通知失败'
        }
      })
    }
  }

  /**
   * 删除文件
   */
  private handleDeleteFile(index: number): void {
    const currentFiles = this.getCurrentFiles()
    const newFiles = [...currentFiles]
    newFiles.splice(index, 1)
    this.updateFiles(newFiles)
    ElMessage.success('删除成功')
  }

  /**
   * 下载文件
   * ✅ 使用 fetch 下载，确保带上 JWT token
   */
  private async handleDownloadFile(file: FileItem): Promise<void> {
    try {
      // ✅ 确定下载 URL
      let downloadURL = file.url
      
      // 如果 url 不是完整的 URL，说明是 key，需要构建完整 URL
      if (!downloadURL || (!downloadURL.startsWith('http://') && !downloadURL.startsWith('https://'))) {
        // 使用 key 构建下载 URL
        downloadURL = `/api/v1/storage/download/${encodeURIComponent(file.url)}`
      }
      
      // ✅ 使用 fetch 下载，带上 JWT token
      const token = localStorage.getItem('token') || ''
      const res = await fetch(downloadURL, {
        headers: { 
          'X-Token': token,
        },
      })

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({ msg: res.statusText }))
        throw new Error(errorData.msg || `下载失败: ${res.statusText}`)
      }

      // ✅ 获取文件 Blob
      const blob = await res.blob()
      
      // ✅ 创建下载链接并触发下载
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = file.name || 'download'
      document.body.appendChild(link)
      link.click()
      
      // ✅ 清理
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      
      ElMessage.success('下载成功')
    } catch (error: any) {
      Logger.error('FilesWidget', 'Download failed', error)
      ElMessage.error(`下载失败: ${error.message}`)
    }
  }

  /**
   * 更新文件描述
   */
  private handleUpdateDescription(index: number, description: string): void {
    const currentFiles = this.getCurrentFiles()
    if (index < 0 || index >= currentFiles.length) {
      return
    }
    const newFiles = [...currentFiles]
    const fileToUpdate = newFiles[index]
    if (fileToUpdate) {
      newFiles[index] = { ...fileToUpdate, description }
      this.updateFiles(newFiles)
    }
  }

  /**
   * 更新备注
   */
  private handleUpdateRemark(remark: string): void {
    const currentValue = this.safeGetValue(this.fieldPath)
    const data = (currentValue?.raw as FilesData) || {
      files: [],
      remark: '',
      metadata: {},
    }

    const newData: FilesData = {
      ...data,
      remark,
    }

    this.safeSetValue(this.fieldPath, {
      raw: newData,
      display: `${data.files.length} 个文件`,
      meta: {},
    })
  }

  /**
   * 渲染表单项
   */
  /**
   * 🔥 渲染响应参数（只读模式）
   * 重写 BaseWidget 的方法，在响应参数中禁用上传和编辑功能
   */
  renderForResponse(): any {
    // 响应参数模式：使用 renderForDetail() 展示（详情展示更丰富）
    // 🔥 使用 props.value（WidgetBuilder.create 时传递的 initialValue）
    // 这是最可靠的值，因为它是创建时直接传递的
    const currentValue = this.value.value
    
    // 🔥 直接使用 props.value，不尝试从 formManager 获取（因为可能已经被构造函数覆盖）
    return this.renderForDetail(currentValue)
  }

  /**
   * 🔥 渲染详情展示（用于 TableRenderer 详情抽屉）
   * 使用九宫格布局展示文件，支持点击预览
   */
  renderForDetail(value?: FieldValue): any {
    const currentValue = value || this.safeGetValue(this.fieldPath)
    const data = (currentValue?.raw as FilesData) || { files: [], remark: '', metadata: {} }
    const currentFiles = data.files || []
    
    // 🔥 构建子元素数组
    const children: any[] = []
    
    // 已上传的文件列表 - 九宫格布局
    if (currentFiles.length > 0) {
      children.push(
        h('div', { 
          class: 'files-grid-container',
          style: {
            marginBottom: '20px',
          }
        }, [
          h('div', { 
            class: 'section-title',
            style: {
              fontSize: '14px',
              fontWeight: '500',
              color: 'var(--el-text-color-primary)',
              marginBottom: '16px',
              paddingBottom: '8px',
              borderBottom: '1px solid var(--el-border-color-lighter)',
            }
          }, `已上传文件 (${currentFiles.length})`),
          h('div', {
            class: 'files-grid',
            style: {
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
              gap: '16px',
            }
          }, currentFiles.map((file, index) => {
            const isImage = this.isImageFile(file)
            const canPreview = file.is_uploaded && file.url
            
            return h(ElCard, {
              key: file.url || file.name || index,
              class: 'file-grid-item',
              style: {
                cursor: canPreview ? 'pointer' : 'default',
                transition: 'all 0.2s ease',
              },
              shadow: 'hover',
              onClick: canPreview ? () => this.handlePreviewInNewWindow(file) : undefined,
            }, {
              // 头部：文件名
              header: () => h('div', {
                style: {
                  fontSize: '13px',
                  fontWeight: '500',
                  color: 'var(--el-text-color-primary)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  display: '-webkit-box',
                  WebkitLineClamp: 2,
                  WebkitBoxOrient: 'vertical',
                  lineHeight: '1.5',
                  wordBreak: 'break-word',
                  padding: '0 4px',
                },
                title: file.name,
              }, file.name),
              // 内容：图片预览或文件封面
              default: () => {
                const coverUrl = this.getFileCoverUrl(file)
                
                // 如果是图片且有URL，显示图片预览
                if (isImage && file.is_uploaded && coverUrl) {
                  return h('div', {
                    style: {
                      width: '100%',
                      height: '150px',
                      backgroundColor: 'var(--el-fill-color-light)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      overflow: 'hidden',
                      borderRadius: '4px',
                    }
                  }, [
                    h(ElImage, {
                      src: coverUrl,
                      fit: 'cover',
                      style: {
                        width: '100%',
                        height: '100%',
                        cursor: 'pointer',
                      },
                      previewSrcList: currentFiles
                        .filter(f => this.isImageFile(f) && f.is_uploaded && f.url)
                        .map(f => f.url || ''),
                      previewTeleported: true,
                      hideOnClickModal: false,
                      initialIndex: currentFiles
                        .filter(f => this.isImageFile(f) && f.is_uploaded && f.url)
                        .findIndex(f => f.url === file.url),
                      onClick: (e: Event) => {
                        // 图片点击时，使用 ElImage 的预览功能，不触发卡片点击
                        e.stopPropagation()
                      }
                    })
                  ])
                }
                
                // 其他文件类型，显示带颜色的封面图标
                return h('div', {
                  style: {
                    width: '100%',
                    height: '150px',
                    borderRadius: '4px',
                    overflow: 'hidden',
                  }
                }, [
                  this.getFileTypeIcon(file)
                ])
              },
              // 底部：文件大小和下载按钮
              footer: () => h('div', {
                style: {
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '8px',
                }
              }, [
                h('div', {
                  style: {
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    fontSize: '11px',
                    color: 'var(--el-text-color-secondary)',
                  }
                }, [
                  h('span', this.formatSize(file.size)),
                  canPreview && h(ElIcon, {
                    size: 12,
                    style: { color: 'var(--el-color-primary)' }
                  }, {
                    default: () => h(View)
                  }),
                ]),
                file.is_uploaded && h(ElButton, {
                  size: 'small',
                  type: 'primary',
                  icon: Download,
                  onClick: (e: MouseEvent) => {
                    e.stopPropagation() // 阻止触发卡片点击事件
                    this.handleDownloadFile(file)
                  },
                  style: {
                    width: '100%',
                    fontSize: '11px',
                  }
                }, {
                  default: () => '下载'
                }),
              ]),
            })
          }))
        ])
      )
    } else {
      // 如果没有文件，显示提示
      children.push(
        h('div', {
          style: {
            padding: '40px',
            textAlign: 'center',
            color: 'var(--el-text-color-secondary)',
          }
        }, '暂无文件')
      )
    }

    // 备注（只读显示）
    if (data.remark) {
      children.push(
        h('div', { 
          class: 'files-remark',
          style: {
            marginTop: '20px',
            paddingTop: '20px',
            borderTop: '1px solid var(--el-border-color-lighter)',
          }
        }, [
          h('div', { 
            class: 'section-title',
            style: {
              fontSize: '14px',
              fontWeight: '500',
              color: 'var(--el-text-color-primary)',
              marginBottom: '12px',
            }
          }, '备注'),
          h('div', {
            style: {
              fontSize: '14px',
              color: 'var(--el-text-color-primary)',
              whiteSpace: 'pre-wrap',
              lineHeight: '1.6',
            }
          }, data.remark),
        ])
      )
    }
    
    return h('div', { 
      class: 'files-widget-detail',
      style: {
        padding: '20px',
      }
    }, children)
  }
  
  /**
   * 判断是否为图片文件
   */
  private isImageFile(file: FileItem): boolean {
    const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg']
    const fileName = (file.name || '').toLowerCase()
    return imageExtensions.some(ext => fileName.endsWith(ext))
  }

  /**
   * 获取文件类型图标（带渐变背景的封面）
   */
  private getFileTypeIcon(file: FileItem): any {
    const fileName = (file.name || '').toLowerCase()
    
    // PDF
    if (fileName.endsWith('.pdf')) {
      return h('div', {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          color: 'white',
        }
      }, [
        h(ElIcon, { size: 48 }, { default: () => h(Document) }),
        h('span', { style: { marginTop: '8px', fontSize: '12px', fontWeight: '500' } }, 'PDF')
      ])
    }
    
    // 视频
    if (['.mp4', '.avi', '.mov', '.wmv', '.flv', '.mkv', '.webm'].some(ext => fileName.endsWith(ext))) {
      return h('div', {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
          color: 'white',
        }
      }, [
        h(ElIcon, { size: 48 }, { default: () => h(VideoPlay) }),
        h('span', { style: { marginTop: '8px', fontSize: '12px', fontWeight: '500' } }, '视频')
      ])
    }
    
    // Word
    if (['.doc', '.docx'].some(ext => fileName.endsWith(ext))) {
      return h('div', {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
          color: 'white',
        }
      }, [
        h(ElIcon, { size: 48 }, { default: () => h(Files) }),
        h('span', { style: { marginTop: '8px', fontSize: '12px', fontWeight: '500' } }, 'Word')
      ])
    }
    
    // Excel
    if (['.xls', '.xlsx', '.csv'].some(ext => fileName.endsWith(ext))) {
      return h('div', {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
          color: 'white',
        }
      }, [
        h(ElIcon, { size: 48 }, { default: () => h(Files) }),
        h('span', { style: { marginTop: '8px', fontSize: '12px', fontWeight: '500' } }, 'Excel')
      ])
    }
    
    // PowerPoint
    if (['.ppt', '.pptx'].some(ext => fileName.endsWith(ext))) {
      return h('div', {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
          color: 'white',
        }
      }, [
        h(ElIcon, { size: 48 }, { default: () => h(Files) }),
        h('span', { style: { marginTop: '8px', fontSize: '12px', fontWeight: '500' } }, 'PPT')
      ])
    }
    
    // 压缩文件
    if (['.zip', '.rar', '.7z', '.tar', '.gz'].some(ext => fileName.endsWith(ext))) {
      return h('div', {
        style: {
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%)',
          color: '#8b4513',
        }
      }, [
        h(ElIcon, { size: 48 }, { default: () => h(Folder) }),
        h('span', { style: { marginTop: '8px', fontSize: '12px', fontWeight: '500' } }, '压缩包')
      ])
    }
    
    // 默认文件图标
    return h('div', {
      style: {
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
        color: '#333',
      }
    }, [
      h(ElIcon, { size: 48 }, { default: () => h(Document) }),
      h('span', { style: { marginTop: '8px', fontSize: '12px', fontWeight: '500' } }, '文件')
    ])
  }

  /**
   * 获取文件封面URL（尝试获取PDF第一页等）
   */
  private getFileCoverUrl(file: FileItem): string | null {
    // 如果是图片，直接返回URL
    if (this.isImageFile(file) && file.is_uploaded && file.url) {
      return file.url
    }
    
    // 对于PDF，可以尝试获取封面（需要后端支持或使用PDF.js）
    // 这里暂时返回null，后续可以扩展
    return null
  }

  /**
   * 在新窗口预览文件（支持PDF、图片等）
   * 对于需要认证的文件，通过添加 token 参数或使用下载接口
   */
  private async handlePreviewInNewWindow(file: FileItem): Promise<void> {
    if (!file.is_uploaded || !file.url) {
      ElMessage.warning('文件未上传，无法预览')
      return
    }

    try {
      let previewURL = file.url

      // 如果 url 不是完整的 URL，需要构建完整 URL
      if (!previewURL.startsWith('http://') && !previewURL.startsWith('https://')) {
        previewURL = `/api/v1/storage/download/${encodeURIComponent(file.url)}`
      }

      // 对于需要认证的文件，添加 token 参数
      if (previewURL.startsWith('/api/')) {
        const token = localStorage.getItem('token') || ''
        // 如果 URL 已经有参数，使用 &，否则使用 ?
        const separator = previewURL.includes('?') ? '&' : '?'
        previewURL = `${previewURL}${separator}token=${encodeURIComponent(token)}`
      }

      // 在新窗口打开文件
      // 浏览器会根据文件类型自动处理（PDF、图片等）
      window.open(previewURL, '_blank')
    } catch (error: any) {
      Logger.error('FilesWidget', 'Preview failed', error)
      ElMessage.error(`预览失败: ${error.message}`)
    }
  }

  /**
   * 🔥 获取复制文本
   * 复制文件名称列表（换行分隔），如果有 URL 则复制 URL
   */
  getCopyText(): string {
    const currentValue = this.safeGetValue(this.fieldPath)
    const data = (currentValue?.raw as FilesData) || { files: [], remark: '', metadata: {} }
    const currentFiles = data.files || []
    
    if (currentFiles.length === 0) {
      return ''
    }
    
    // 如果有多个文件，复制文件名称列表（换行分隔）
    // 如果只有一个文件且有 URL，复制 URL
    const firstFile = currentFiles[0]
    if (currentFiles.length === 1 && firstFile && firstFile.url) {
      return firstFile.url
    }
    
    // 否则复制文件名称列表
    return currentFiles.map((file: FileItem) => file.name || file.source_name || '未知文件').join('\n')
  }

  render() {
    // ✅ 临时 Widget（表格渲染）只显示简单的文件列表
    if (this.isTemporary) {
      return this.renderTableCell()
    }

    const currentFiles = this.getCurrentFiles()
    const currentValue = this.safeGetValue(this.fieldPath)
    const data = (currentValue?.raw as FilesData) || { files: [], remark: '', metadata: {} }
    const maxCount = this.filesConfig.max_count || 5
    // ✅ 从配置或 field 获取，如果 router 为空则表示只读模式
    const isDisabled = this.filesConfig.disabled || !this.router || this.router === ''
    const isMaxReached = currentFiles.length >= maxCount

    return h('div', { 
      class: 'files-widget',
      style: {
        padding: '20px',
        backgroundColor: 'var(--el-fill-color-lighter)',  // ✅ 使用 Element Plus 主题变量
        borderRadius: '8px',
        border: '1px solid var(--el-border-color-light)',  // ✅ 使用 Element Plus 主题变量
      }
    }, [
      // 上传区域
      !isDisabled && !isMaxReached && h('div', { 
        class: 'upload-area',
        style: {
          marginBottom: '20px',
          backgroundColor: 'var(--el-bg-color)',  // ✅ 使用 Element Plus 主题变量
          border: '2px dashed var(--el-border-color)',  // ✅ 使用 Element Plus 主题变量
          borderRadius: '8px',
          padding: '24px',
          transition: 'all 0.3s ease',
          cursor: 'pointer',
        },
        onMouseenter: (e: MouseEvent) => {
          const target = e.currentTarget as HTMLElement
          if (target) {
            target.style.borderColor = 'var(--el-color-primary)'
            target.style.backgroundColor = 'var(--el-color-primary-light-9)'
          }
        },
        onMouseleave: (e: MouseEvent) => {
          const target = e.currentTarget as HTMLElement
          if (target) {
            target.style.borderColor = 'var(--el-border-color)'
            target.style.backgroundColor = 'var(--el-bg-color)'
          }
        },
      }, [
        h(ElUpload, {
          autoUpload: false,
          showFileList: false,
          drag: true,
          accept: this.filesConfig.accept || '*',
          onChange: (file: any) => {
            if (file.raw) {
              this.handleFileSelect(file.raw)
            }
          },
        }, {
          default: () => [
            h('div', { 
              class: 'upload-dragger-content',
              style: {
                textAlign: 'center',
              }
            }, [
              h(ElIcon, { 
                size: 48, 
                style: { color: 'var(--el-text-color-secondary)' } 
              }, () => h(Upload)),
              h('div', { 
                class: 'el-upload__text',
                style: {
                  marginTop: '12px',
                  fontSize: '16px',
                  color: 'var(--el-text-color-primary)',
                  fontWeight: '500',
                }
              }, [
                '将文件拖到此处，或',
                h('em', { 
                  style: { 
                    color: 'var(--el-color-primary)', 
                    fontStyle: 'normal', 
                    fontWeight: '500',
                    marginLeft: '4px' 
                  } 
                }, '点击上传'),
              ]),
              h('div', { 
                class: 'el-upload__tip',
                style: {
                  marginTop: '8px',
                  fontSize: '14px',
                  color: 'var(--el-text-color-secondary)',
                }
              }, [
                `支持 ${this.filesConfig.accept || '所有类型'}，`,
                this.filesConfig.max_size && `单个文件不超过 ${this.filesConfig.max_size}，`,
                `最多 ${maxCount} 个文件`,
              ].filter(Boolean).join('')),
            ]),
          ],
        }),
      ]),

      // 上传中的文件
      this.uploadingFiles.value.length > 0 && h('div', { 
        class: 'uploading-files',
        style: {
          marginBottom: '20px',
        }
      }, [
        h('div', { 
          class: 'section-title',
          style: {
            fontSize: '14px',
            fontWeight: '500',
            color: 'var(--el-text-color-primary)',
            marginBottom: '12px',
            paddingBottom: '8px',
            borderBottom: '1px solid var(--el-border-color-lighter)',
          }
        }, '上传中'),
        ...this.uploadingFiles.value.map((file: UploadingFile) =>
          h('div', { 
            class: 'uploading-file', 
            key: file.uid,
            style: {
              backgroundColor: 'var(--el-bg-color)',
              border: '1px solid var(--el-border-color-light)',
              borderRadius: '6px',
              padding: '12px',
              marginBottom: '10px',
            }
          }, [
            h('div', { 
              class: 'file-info',
              style: {
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                marginBottom: '8px',
              }
            }, [
              h(ElIcon, { 
                size: 16, 
                style: { color: 'var(--el-color-primary)' } 
              }, () => h(Document)),
              h('span', { 
                class: 'file-name',
                style: {
                  fontSize: '14px',
                  color: 'var(--el-text-color-primary)',
                  fontWeight: '500',
                  flex: 1,
                }
              }, file.name),
              h('span', { 
                class: 'file-size',
                style: {
                  fontSize: '12px',
                  color: 'var(--el-text-color-secondary)',
                }
              }, this.formatSize(file.size)),
            ]),
            h(ElProgress, {
              percentage: file.percent,
              status: file.status === 'error' ? 'exception' : undefined,
            }),
            // ✨ 显示上传速度和操作按钮
            h('div', {
              style: {
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginTop: '8px',
              }
            }, [
              // 上传速度或错误信息
              file.status === 'uploading' && file.speed && h('span', {
                style: {
                  fontSize: '12px',
                  color: 'var(--el-text-color-secondary)',
                }
              }, `速度: ${file.speed}`),
              file.error && h('span', { 
                style: {
                  fontSize: '12px',
                  color: 'var(--el-color-danger)',
                  flex: 1,
                }
              }, file.error),
              // 操作按钮
              h('div', {
                style: {
                  display: 'flex',
                  gap: '8px',
                }
              }, [
                // 取消按钮（上传中时显示）
                file.status === 'uploading' && file.cancel && h(ElButton, {
                  size: 'small',
                  type: 'danger',
                  onClick: file.cancel,
                }, () => '取消'),
                // 重试按钮（失败时显示）
                file.status === 'error' && file.retry && h(ElButton, {
                  size: 'small',
                  type: 'primary',
                  onClick: file.retry,
                }, () => '重试'),
              ]),
            ]),
          ])
        ),
      ]),

      // 已上传的文件列表
      currentFiles.length > 0 && h('div', { 
        class: 'uploaded-files',
        style: {
          marginBottom: '20px',
        }
      }, [
        h('div', { 
          class: 'section-title',
          style: {
            fontSize: '14px',
            fontWeight: '500',
            color: 'var(--el-text-color-primary)',
            marginBottom: '12px',
            paddingBottom: '8px',
            borderBottom: '1px solid var(--el-border-color-lighter)',
          }
        }, `已上传文件 (${currentFiles.length}/${maxCount})`),
        ...currentFiles.map((file, index) =>
          h('div', { 
            class: 'uploaded-file', 
            key: file.url,
            style: {
              backgroundColor: 'var(--el-bg-color)',
              border: '1px solid var(--el-border-color-light)',
              borderRadius: '6px',
              padding: '12px',
              marginBottom: '10px',
              transition: 'all 0.2s ease',
            },
            onMouseenter: (e: MouseEvent) => {
              const target = e.currentTarget as HTMLElement
              if (target) {
                target.style.borderColor = 'var(--el-color-primary)'
                target.style.backgroundColor = 'var(--el-color-primary-light-9)'
              }
            },
            onMouseleave: (e: MouseEvent) => {
              const target = e.currentTarget as HTMLElement
              if (target) {
                target.style.borderColor = 'var(--el-border-color-light)'
                target.style.backgroundColor = 'var(--el-bg-color)'
              }
            },
          }, [
            // 文件信息
            h('div', { 
              class: 'file-header',
              style: {
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                marginBottom: '8px',
              }
            }, [
              h(ElIcon, { 
                size: 16, 
                style: { color: 'var(--el-color-primary)' } 
              }, () => h(Document)),
              h('span', { 
                class: 'file-name', 
                title: file.name,
                style: {
                  fontSize: '14px',
                  color: 'var(--el-text-color-primary)',
                  fontWeight: '500',
                  flex: 1,
                }
              }, file.name),
              h('span', { 
                class: 'file-size',
                style: {
                  fontSize: '12px',
                  color: 'var(--el-text-color-secondary)',
                }
              }, this.formatSize(file.size)),
              h(ElTag, {
                size: 'small',
                type: file.is_uploaded ? 'success' : 'info',
              }, () => file.is_uploaded ? '已上传' : '本地'),
            ]),

            // 文件描述
            !isDisabled && h('div', { 
              class: 'file-description',
              style: {
                marginBottom: '8px',
              }
            }, [
              h(ElInput, {
                modelValue: file.description,
                'onUpdate:modelValue': (val: string) => this.handleUpdateDescription(index, val),
                placeholder: '添加文件描述（可选）',
                size: 'small',
                clearable: true,
              }),
            ]),

            // 操作按钮
            h('div', { 
              class: 'file-actions',
              style: {
                display: 'flex',
                gap: '8px',
              }
            }, [
              // 下载按钮
              file.is_uploaded && h(ElButton, {
                size: 'small',
                icon: Download,
                onClick: () => this.handleDownloadFile(file),
              }, () => '下载'),

              // 删除按钮
              !isDisabled && h(ElPopconfirm, {
                title: '确定删除此文件？',
                onConfirm: () => this.handleDeleteFile(index),
              }, {
                reference: () => h(ElButton, {
                  size: 'small',
                  type: 'danger',
                  icon: Delete,
                }, () => '删除'),
              }),
            ]),
          ])
        ),
      ]),

      // 备注（与上传组件融为一体，不单独成块）
      !isDisabled && h('div', { 
        class: 'files-remark',
        style: {
          marginTop: '20px',
          paddingTop: '20px',
          borderTop: '1px solid var(--el-border-color-lighter)',
        }
      }, [
        h('div', { 
          class: 'section-title',
          style: {
            fontSize: '14px',
            fontWeight: '500',
            color: 'var(--el-text-color-primary)',
            marginBottom: '12px',
          }
        }, '备注（可选）'),
        h(ElInput, {
          modelValue: data.remark,
          'onUpdate:modelValue': (val: string) => this.handleUpdateRemark(val),
          type: 'textarea',
          rows: 2,
          placeholder: '添加备注信息',
          maxlength: 500,
          showWordLimit: true,
        }),
      ]),
    ])
  }

  /**
   * 渲染表格单元格
   * ✅ 简化显示：只显示文件数量，详情在详情抽屉中查看
   */
  renderTableCell(value?: FieldValue) {
    // ✅ 如果传入了 value，使用它；否则从当前值获取
    const fieldValue = value || this.safeGetValue(this.fieldPath)
    
    // ✅ 解析 FilesData 结构
    let files: FileItem[] = []
    if (fieldValue?.raw) {
      const data = fieldValue.raw as FilesData | FileItem[]
      if (data && typeof data === 'object' && 'files' in data && Array.isArray(data.files)) {
        // FilesData 结构
        files = data.files
      } else if (Array.isArray(data)) {
        // 兼容：如果 raw 直接是数组，当作文件列表
        files = data
      }
    }
    
    if (files.length === 0) {
      return h('span', { style: { color: '#909399' } }, '-')
    }

    // ✅ 简化显示：只显示文件数量标签
    return h(ElTag, { 
      size: 'small', 
      type: 'info',
      style: { 
        fontSize: '12px'
      }
    }, {
      default: () => `${files.length} 个文件`
    })
  }

  /**
   * 渲染搜索输入（不支持搜索）
   */
  renderSearchInput() {
    return h('div', '文件组件不支持搜索')
  }

  /**
   * 🔥 重写：获取提交时的原始值
   * 确保返回完整的 FilesData 结构
   */
  getRawValueForSubmit(): FilesData {
    const currentValue = this.safeGetValue(this.fieldPath)
    const data = currentValue?.raw as FilesData | null
    
    // ✅ 确保返回完整的 FilesData 结构
    if (data && this.isValidFilesData(data)) {
      return {
        files: data.files || [],
        remark: data.remark || '',
        metadata: data.metadata || {},
      }
    }
    
    // ✅ 如果数据无效，返回空结构（而不是空对象）
    return {
      files: [],
      remark: '',
      metadata: {},
    }
  }

  /**
   * 🔥 静态方法：从原始数据加载为 FieldValue 格式
   * 处理后端返回的 FilesData 结构
   */
  static loadFromRawData(rawValue: any, field: FieldConfig): FieldValue {
    // 🔥 如果已经是 FieldValue 格式，直接返回
    if (rawValue && typeof rawValue === 'object' && 'raw' in rawValue && 'display' in rawValue) {
      return rawValue
    }
    
    // 🔥 空值处理：返回默认空结构
    if (rawValue === null || rawValue === undefined || rawValue === '') {
      return {
        raw: {
          files: [],
          remark: '',
          metadata: {},
        } as FilesData,
        display: '0 个文件',
        meta: {}
      }
    }

    // ✅ 解析 FilesData 结构
    let filesData: FilesData
    
    if (typeof rawValue === 'object') {
      // 检查是否是 FilesData 结构
      if (Array.isArray(rawValue.files)) {
        filesData = {
          files: rawValue.files || [],
          remark: rawValue.remark || '',
          metadata: rawValue.metadata || {},
        }
      } else if (Array.isArray(rawValue)) {
        // 兼容：如果直接是数组，包装成 FilesData
        // 类型断言：确保是 FileItem 数组
        filesData = {
          files: rawValue as FileItem[],
          remark: '',
          metadata: {},
        }
      } else {
        // 无效数据，返回空结构
        filesData = {
          files: [],
          remark: '',
          metadata: {},
        }
      }
    } else {
      // 非对象类型，返回空结构
      filesData = {
        files: [],
        remark: '',
        metadata: {},
      }
    }

    return {
      raw: filesData,
      display: `${filesData.files.length} 个文件`,
      meta: {}
    }
  }
}

