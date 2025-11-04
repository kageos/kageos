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
} from 'element-plus'
import {
  Upload,
  Document,
  Delete,
  View,
  Download,
} from '@element-plus/icons-vue'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'
import { uploadFile } from '@/utils/upload'
import type { UploadProgress, UploadResult } from '@/utils/upload/types'
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
  url: string
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
  cancel?: () => void
}

export class FilesWidget extends BaseWidget {
  // 常量定义
  private static readonly MAX_DISPLAY_FILES = 3  // 表格单元格最多显示的文件数量
  
  // 组件私有状态
  private uploadingFiles = ref<UploadingFile[]>([])
  private filesConfig: FilesConfig
  private router: string

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
    if (!match) {
      Logger.error(`[FilesWidget] Invalid max_size format: ${maxSizeStr}`)
      return Infinity
    }

    const [, size, unit] = match
    return parseFloat(size) * units[unit.toUpperCase()]
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

      const isAccepted = accept.some(pattern => {
        // 扩展名匹配：.pdf
        if (pattern.startsWith('.')) {
          return fileName.endsWith(pattern)
        }
        // MIME 通配符：image/*
        if (pattern.includes('/*')) {
          const prefix = pattern.split('/')[0]
          return fileType.startsWith(prefix)
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
    }
    this.uploadingFiles.value.push(uploadingFile)

    try {
      // ✨ 调用统一上传工具（后端会根据配置返回对应的上传方式）
      // ✅ uploadFile 现在返回 UploadResult，包含 downloadURL、key 和 storage
      const uploadResult = await uploadFile(
        this.router,
        rawFile,
        (progress: UploadProgress) => {
          // 更新进度
          const file = this.uploadingFiles.value.find(f => f.uid === uid)
          if (file) {
            file.percent = progress.percent
          }
        }
      )

      // 上传成功，更新状态
      const file = this.uploadingFiles.value.find(f => f.uid === uid)
      if (file) {
        file.status = 'success'
      }

      // 添加到文件列表
      // ✅ downloadURL 已经是完整的下载 URL（从上传完成接口返回）
      const newFile: FileItem = {
        name: rawFile.name,
        source_name: rawFile.name, // ✨ 源文件名称（上传时的原始文件名）
        storage: uploadResult.storage || 'minio', // ✨ 存储引擎类型（从上传凭证获取）
        description: '',
        hash: '', // 后端会计算
        size: rawFile.size,
        upload_ts: Date.now(),
        local_path: '',
        is_uploaded: true,
        url: uploadResult.downloadURL, // ✅ 直接使用返回的下载 URL
        downloaded: false,
      }

      const currentFiles = this.getCurrentFiles()
      this.updateFiles([...currentFiles, newFile])

      ElMessage.success('上传成功')

      // 2 秒后移除上传记录
      setTimeout(() => {
        const index = this.uploadingFiles.value.findIndex(f => f.uid === uid)
        if (index !== -1) {
          this.uploadingFiles.value.splice(index, 1)
        }
      }, 2000)

    } catch (error: any) {
      Logger.error('[FilesWidget] Upload failed:', error)

      // 更新状态
      const file = this.uploadingFiles.value.find(f => f.uid === uid)
      if (file) {
        file.status = 'error'
        file.error = error.message
      }

      ElMessage.error(`上传失败: ${error.message}`)
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
      Logger.error('[FilesWidget] Download failed:', error)
      ElMessage.error(`下载失败: ${error.message}`)
    }
  }

  /**
   * 更新文件描述
   */
  private handleUpdateDescription(index: number, description: string): void {
    const currentFiles = this.getCurrentFiles()
    const newFiles = [...currentFiles]
    newFiles[index] = { ...newFiles[index], description }
    this.updateFiles(newFiles)
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
   * 显示完整的文件列表，支持下载
   */
  renderForDetail(value?: FieldValue): any {
    const currentValue = value || this.safeGetValue(this.fieldPath)
    const data = (currentValue?.raw as FilesData) || { files: [], remark: '', metadata: {} }
    const currentFiles = data.files || []
    
    // 🔥 详情展示模式：禁用所有编辑功能
    const isDisabled = true  // 详情始终禁用
    const isMaxReached = false  // 不限制显示数量
    
    // 🔥 构建子元素数组（过滤掉 false 值）
    const children: any[] = []
    
    // 已上传的文件列表
    if (currentFiles.length > 0) {
      children.push(
        h('div', { 
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
          }, `已上传文件 (${currentFiles.length})`),
          ...currentFiles.map((file, index) =>
            h('div', { 
              class: 'uploaded-file', 
              key: file.url || file.name || index,
              style: {
                backgroundColor: 'var(--el-bg-color)',
                border: '1px solid var(--el-border-color-light)',
                borderRadius: '6px',
                padding: '12px',
                marginBottom: '10px',
                transition: 'all 0.2s ease',
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

              // 文件描述（只读显示）
              file.description && h('div', { 
                class: 'file-description',
                style: {
                  marginBottom: '8px',
                  fontSize: '13px',
                  color: 'var(--el-text-color-secondary)',
                }
              }, file.description),

              // 操作按钮（只显示下载，不显示删除）
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
              ]),
            ])
          ),
        ])
      )
    } else {
      // 如果没有文件，显示提示
      children.push(
        h('div', {
          style: {
            padding: '20px',
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
            }
          }, data.remark),
        ])
      )
    }
    
    return h('div', { 
      class: 'files-widget',
      style: {
        padding: '20px',
        backgroundColor: 'var(--el-fill-color-lighter)',
        borderRadius: '8px',
        border: '1px solid var(--el-border-color-light)',
      }
    }, children)
  }

  /**
   * 🔥 获取复制文本
   * 复制文件名称列表（换行分隔），如果有 URL 则复制 URL
   */
  onCopy(): string {
    const currentValue = this.safeGetValue(this.fieldPath)
    const data = (currentValue?.raw as FilesData) || { files: [], remark: '', metadata: {} }
    const currentFiles = data.files || []
    
    if (currentFiles.length === 0) {
      return ''
    }
    
    // 如果有多个文件，复制文件名称列表（换行分隔）
    // 如果只有一个文件且有 URL，复制 URL
    if (currentFiles.length === 1 && currentFiles[0].url) {
      return currentFiles[0].url
    }
    
    // 否则复制文件名称列表
    return currentFiles.map(file => file.name || file.source_name || '未知文件').join('\n')
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
        ...this.uploadingFiles.value.map(file =>
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
            file.error && h('div', { 
              class: 'error-message',
              style: {
                marginTop: '8px',
                fontSize: '12px',
                color: 'var(--el-color-danger)',
              }
            }, file.error),
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
   * ✅ 支持点击文件下载
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

    return h('div', { 
      class: 'files-table-cell',
      style: { 
        display: 'flex', 
        flexDirection: 'column', 
        gap: '4px',
        padding: '4px 0'
      }
    }, [
      h(ElTag, { 
        size: 'small', 
        type: 'info',
        style: { marginBottom: '4px', width: 'fit-content' }
      }, () => `${files.length} 个文件`),
      ...files.slice(0, FilesWidget.MAX_DISPLAY_FILES).map((file, index) =>
        h('div', {
          key: file.url || file.name || index,
          class: 'file-item',
          title: file.name || file.description || '文件',
          style: { 
            display: 'flex', 
            alignItems: 'center', 
            gap: '6px',
            padding: '4px 8px',
            backgroundColor: '#f5f7fa',
            borderRadius: '4px',
            cursor: 'pointer',
            transition: 'all 0.2s',
          },
          onClick: () => {
            // ✅ 点击文件时下载
            if (file.is_uploaded !== false && (file.url || file.name)) {
              this.handleDownloadFile(file)
            }
          },
          onMouseenter: (e: MouseEvent) => {
            const target = e.currentTarget as HTMLElement
            if (target) {
              target.style.backgroundColor = '#e4e7ed'
            }
          },
          onMouseleave: (e: MouseEvent) => {
            const target = e.currentTarget as HTMLElement
            if (target) {
              target.style.backgroundColor = '#f5f7fa'
            }
          },
        }, [
          h(ElIcon, { size: 14, style: { color: '#409EFF' } }, () => h(Document)),
          h('span', { 
            class: 'file-name', 
            style: { 
              fontSize: '12px',
              color: '#606266',
              flex: 1,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            } 
          }, file.name || '未知文件'),
        ])
      ),
      files.length > FilesWidget.MAX_DISPLAY_FILES && h('span', { 
        class: 'more-files', 
        style: { 
          marginTop: '4px',
          color: '#909399', 
          fontSize: '12px',
          fontStyle: 'italic'
        } 
      }, `+${files.length - FilesWidget.MAX_DISPLAY_FILES} 个文件`),
    ])
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

