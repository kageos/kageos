<!--
  FilesWidget - 文件上传组件
  支持多文件上传、拖拽上传、文件管理
-->
<template>
  <div class="files-widget">
    <!-- 编辑模式 -->
    <template v-if="mode === 'edit'">
      <!-- 上传区域 -->
      <div
        v-if="!isDisabled && !isMaxReached"
        class="upload-area"
        @drop.prevent="handleDrop"
        @dragover.prevent="handleDragOver"
        @dragleave.prevent="handleDragLeave"
        :class="{ 'is-dragging': isDragging }"
      >
        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :show-file-list="false"
          :drag="true"
          :accept="accept"
          :on-change="handleFileChange"
        >
          <div class="upload-dragger-content">
            <el-icon :size="48" class="upload-icon">
              <Upload />
            </el-icon>
            <div class="el-upload__text">
              将文件拖到此处，或<em>点击上传</em>
            </div>
            <div class="el-upload__tip">
              {{ uploadTip }}
            </div>
          </div>
        </el-upload>
      </div>

      <!-- 上传中的文件 -->
      <div v-if="uploadingFiles.length > 0" class="uploading-files">
        <div class="section-title">上传中</div>
        <div
          v-for="file in uploadingFiles"
          :key="file.uid"
          class="uploading-file"
        >
          <div class="file-info">
            <el-icon :size="16" class="file-icon">
              <Document />
            </el-icon>
            <span class="file-name">{{ file.name }}</span>
            <span class="file-size">{{ formatSize(file.size) }}</span>
          </div>
          <el-progress
            :percentage="file.percent"
            :status="file.status === 'error' ? 'exception' : undefined"
          />
          <div class="file-actions">
            <span v-if="file.status === 'uploading' && file.speed" class="upload-speed">
              速度: {{ file.speed }}
            </span>
            <span v-if="file.error" class="upload-error">
              {{ file.error }}
            </span>
            <div class="action-buttons">
              <el-button
                v-if="file.status === 'uploading' && file.cancel"
                size="small"
                type="danger"
                @click="file.cancel()"
              >
                取消
              </el-button>
              <el-button
                v-if="file.status === 'error' && file.retry"
                size="small"
                type="primary"
                @click="file.retry()"
              >
                重试
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- 已上传的文件列表 -->
      <div v-if="currentFiles.length > 0" class="uploaded-files">
        <div class="section-title">
          已上传文件 ({{ currentFiles.length }}/{{ maxCount }})
        </div>
        <div
          v-for="(file, index) in currentFiles"
          :key="file.url || file.name || index"
          class="uploaded-file"
        >
          <div class="file-header">
            <el-icon :size="16" class="file-icon">
              <Document />
            </el-icon>
            <span 
              class="file-name" 
              :title="file.name"
              :class="{ 'file-name-clickable': isImageFile(file) && file.is_uploaded }"
              @click="isImageFile(file) && file.is_uploaded ? handlePreviewImage(file) : null"
            >
              {{ file.name }}
            </span>
            <span class="file-size">{{ formatSize(file.size) }}</span>
            <el-tag size="small" :type="file.is_uploaded ? 'success' : 'info'">
              {{ file.is_uploaded ? '已上传' : '本地' }}
            </el-tag>
          </div>

          <!-- 文件描述 -->
          <div v-if="!isDisabled" class="file-description">
            <el-input
              v-model="file.description"
              size="small"
              placeholder="添加文件描述（可选）"
              clearable
              @blur="handleUpdateDescription(index, file.description)"
            />
          </div>

          <!-- 操作按钮 -->
          <div class="file-actions">
            <el-button
              v-if="isImageFile(file) && file.is_uploaded"
              size="small"
              :icon="View"
              @click="handlePreviewImage(file)"
            >
              预览
            </el-button>
            <el-button
              v-if="file.is_uploaded"
              size="small"
              :icon="Download"
              @click="handleDownloadFile(file)"
            >
              下载
            </el-button>
            <el-popconfirm
              v-if="!isDisabled"
              title="确定删除此文件？"
              @confirm="handleDeleteFile(index)"
            >
              <template #reference>
                <el-button size="small" type="danger" :icon="Delete">
                  删除
                </el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>

      <!-- 备注 -->
      <div v-if="!isDisabled" class="files-remark">
        <div class="section-title">备注（可选）</div>
        <el-input
          v-model="remark"
          type="textarea"
          :rows="2"
          placeholder="添加备注信息"
          :maxlength="500"
          show-word-limit
          @blur="handleUpdateRemark"
        />
      </div>
    </template>

    <!-- 响应模式（只读） -->
    <template v-else-if="mode === 'response'">
      <div class="response-files">
        <!-- 已上传的文件列表 -->
        <div v-if="currentFiles.length > 0" class="uploaded-files">
          <div class="section-title">
            已上传文件 ({{ currentFiles.length }})
          </div>
          <div
            v-for="(file, index) in currentFiles"
            :key="file.url || file.name || index"
            class="uploaded-file"
          >
            <div class="file-header">
              <el-icon :size="16" class="file-icon">
                <Document />
              </el-icon>
              <span 
                class="file-name" 
                :title="file.name"
                :class="{ 'file-name-clickable': isImageFile(file) && file.is_uploaded }"
                @click="isImageFile(file) && file.is_uploaded ? handlePreviewImage(file) : null"
              >
                {{ file.name }}
              </span>
              <span class="file-size">{{ formatSize(file.size) }}</span>
              <el-tag size="small" :type="file.is_uploaded ? 'success' : 'info'">
                {{ file.is_uploaded ? '已上传' : '本地' }}
              </el-tag>
            </div>

            <!-- 文件描述（只读显示） -->
            <div v-if="file.description" class="file-description">
              {{ file.description }}
            </div>

            <!-- 操作按钮（只显示下载和预览） -->
            <div class="file-actions">
              <el-button
                v-if="isImageFile(file) && file.is_uploaded"
                size="small"
                :icon="View"
                @click="handlePreviewImage(file)"
              >
                预览
              </el-button>
              <el-button
                v-if="file.is_uploaded"
                size="small"
                :icon="Download"
                @click="handleDownloadFile(file)"
              >
                下载
              </el-button>
            </div>
          </div>
        </div>
        <div v-else class="empty-files">暂无文件</div>

        <!-- 备注（只读显示） -->
        <div v-if="remark" class="files-remark">
          <div class="section-title">备注</div>
          <div class="remark-content">{{ remark }}</div>
        </div>
      </div>
    </template>

    <!-- 表格单元格模式 -->
    <template v-else-if="mode === 'table-cell'">
      <div v-if="currentFiles.length > 0" class="files-table-cell">
        <!-- 🔥 简化展示：只显示文件数量和前几个文件名 -->
        <el-tag size="small" type="info" style="margin-right: 4px">
          {{ currentFiles.length }} 个文件
        </el-tag>
        <span v-if="displayFiles.length > 0" class="file-names">
          <span
            v-for="(file, index) in displayFiles"
            :key="file.url || file.name || index"
            class="file-name-inline"
            :title="file.name || '文件'"
          >
            {{ file.name || '未知文件' }}
            <span v-if="index < displayFiles.length - 1">, </span>
          </span>
          <span v-if="currentFiles.length > MAX_DISPLAY_FILES" class="more-files">
            ...
          </span>
        </span>
      </div>
      <span v-else class="empty-text">-</span>
    </template>

    <!-- 详情模式 -->
    <template v-else-if="mode === 'detail'">
      <div class="detail-files">
        <div v-if="currentFiles.length > 0" class="uploaded-files">
          <!-- 🔥 参考旧版本的紧凑列表布局 -->
          <div
            v-for="(file, index) in currentFiles"
            :key="file.url || file.name || index"
            class="detail-file-item"
            :class="{ 'file-clickable': isImageFile(file) && file.is_uploaded }"
            @click="isImageFile(file) && file.is_uploaded ? handlePreviewImage(file) : null"
          >
            <el-icon :size="16" class="file-icon">
              <Document />
            </el-icon>
            <span class="file-name" :title="file.name">
              {{ file.name }}
            </span>
            <span class="file-size">{{ formatSize(file.size) }}</span>
            <el-tag size="small" :type="file.is_uploaded ? 'success' : 'info'">
              {{ file.is_uploaded ? '已上传' : '本地' }}
            </el-tag>
            <div class="file-actions-inline">
              <el-button
                v-if="isImageFile(file) && file.is_uploaded"
                size="small"
                text
                :icon="View"
                @click.stop="handlePreviewImage(file)"
                title="预览"
              />
              <el-button
                v-if="file.is_uploaded"
                size="small"
                text
                :icon="Download"
                @click.stop="handleDownloadFile(file)"
                title="下载"
              />
            </div>
          </div>
        </div>
        <div v-else class="empty-files">暂无文件</div>

        <div v-if="remark" class="files-remark">
          <div class="section-title">备注</div>
          <div class="remark-content">{{ remark }}</div>
        </div>
      </div>
    </template>

    <!-- 图片预览对话框 -->
    <el-dialog
      v-model="previewVisible"
      :title="previewImageName"
      width="80%"
      :close-on-click-modal="true"
      @close="handleClosePreview"
    >
      <div class="image-preview-container">
        <el-image
          :src="previewImageUrl"
          :preview-src-list="[previewImageUrl]"
          fit="contain"
          style="max-width: 100%; max-height: 70vh;"
          :preview-teleported="true"
        />
      </div>
    </el-dialog>

    <!-- 文件详情对话框 -->
    <el-dialog
      v-model="fileDetailVisible"
      :title="currentDetailFile ? `文件详情 - ${currentDetailFile.name}` : '文件详情'"
      width="600px"
      :close-on-click-modal="true"
      @close="handleCloseFileDetail"
    >
      <div v-if="currentDetailFile" class="file-detail-content">
        <!-- 文件基本信息 -->
        <el-descriptions :column="1" border>
          <el-descriptions-item label="文件名">
            {{ currentDetailFile.name }}
          </el-descriptions-item>
          <el-descriptions-item label="文件大小">
            {{ formatSize(currentDetailFile.size) }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag size="small" :type="currentDetailFile.is_uploaded ? 'success' : 'info'">
              {{ currentDetailFile.is_uploaded ? '已上传' : '本地' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="currentDetailFile.description" label="描述">
            {{ currentDetailFile.description }}
          </el-descriptions-item>
          <el-descriptions-item v-if="currentDetailFile.upload_ts" label="上传时间">
            {{ new Date(currentDetailFile.upload_ts).toLocaleString() }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 图片预览区域 -->
        <div v-if="isImageFile(currentDetailFile) && currentDetailFile.is_uploaded" class="file-preview-section">
          <div class="section-title">预览</div>
          <div class="image-preview-container">
            <el-image
              v-if="previewImageUrl"
              :src="previewImageUrl"
              :preview-src-list="[previewImageUrl]"
              fit="contain"
              style="max-width: 100%; max-height: 400px;"
              :preview-teleported="true"
              :loading="'lazy'"
            />
            <div v-else class="loading-preview">加载中...</div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="file-detail-actions">
          <el-button
            v-if="isImageFile(currentDetailFile) && currentDetailFile.is_uploaded"
            type="primary"
            :icon="View"
            @click="handlePreviewImage(currentDetailFile)"
          >
            预览
          </el-button>
          <el-button
            v-if="currentDetailFile.is_uploaded"
            :icon="Download"
            @click="handleDownloadFile(currentDetailFile)"
          >
            下载
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  ElUpload,
  ElButton,
  ElIcon,
  ElProgress,
  ElMessage,
  ElTag,
  ElPopconfirm,
  ElInput,
  ElDialog,
  ElImage,
  ElDescriptions,
  ElDescriptionsItem,
} from 'element-plus'
import {
  Upload,
  Document,
  Delete,
  Download,
  View,
} from '@element-plus/icons-vue'
import type { WidgetComponentProps } from '../types'
import { uploadFile, notifyBatchUploadComplete } from '@/utils/upload'
import type { FileInfo, BatchUploadCompleteItem, UploadProgress, UploadFileResult } from '@/utils/upload'
import type { Uploader } from '@/utils/upload'
import { useFormDataStore } from '../../stores-v2/formData'
import { Logger } from '../../utils/logger'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: {
      files: [],
      remark: '',
      metadata: {},
    },
    display: '0 个文件',
    meta: {},
  }),
})

const formDataStore = useFormDataStore()

// 常量定义
const MAX_DISPLAY_FILES = 3

// 配置
const filesConfig = computed(() => props.field.widget?.config || {})
const accept = computed(() => filesConfig.value.accept || '*')
const maxSize = computed(() => filesConfig.value.max_size)
const maxCount = computed(() => filesConfig.value.max_count || 5)

// 状态
const isDragging = ref(false)
const uploadingFiles = ref<UploadingFile[]>([])
const pendingCompleteQueue = ref<BatchUploadCompleteItem[]>([])
const batchCompleteTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const BATCH_COMPLETE_DELAY = 500
const BATCH_COMPLETE_MAX_SIZE = 10

// 图片预览相关状态
const previewVisible = ref(false)
const previewImageUrl = ref('')
const previewImageName = ref('')

// 文件详情弹窗相关状态
const fileDetailVisible = ref(false)
const currentDetailFile = ref<FileItem | null>(null)

// 上传中的文件状态
interface UploadingFile {
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

// 文件数据结构
interface FileItem {
  name: string
  source_name?: string
  storage?: string
  description: string
  hash: string
  size: number
  upload_ts: number
  local_path: string
  is_uploaded: boolean
  url: string
  server_url?: string
  downloaded?: boolean
}

interface FilesData {
  files: FileItem[]
  remark: string
  metadata: Record<string, any>
  upload_user?: string    // 文件上传的用户
  widget_type?: string    // Widget 类型，值为 "files"
  data_type?: string      // 数据类型，值为 "struct"
}

// 计算属性
const currentFiles = computed(() => {
  const raw = props.value?.raw
  if (raw && typeof raw === 'object' && 'files' in raw) {
    return (raw as FilesData).files || []
  }
  return []
})

const remark = computed({
  get: () => {
    const raw = props.value?.raw
    if (raw && typeof raw === 'object' && 'remark' in raw) {
      return (raw as FilesData).remark || ''
    }
    return ''
  },
  set: (val: string) => {
    updateRemark(val)
  },
})

const isDisabled = computed(() => {
  if (props.mode !== 'edit') return true
  if (filesConfig.value.disabled) return true
  if (!props.formRenderer) return true
  const router = props.formRenderer.getFunctionRouter()
  return !router || router === ''
})

const isMaxReached = computed(() => currentFiles.value.length >= maxCount.value)

const uploadTip = computed(() => {
  const parts: string[] = []
  parts.push(`支持 ${accept.value || '所有类型'}`)
  if (maxSize.value) {
    parts.push(`单个文件不超过 ${maxSize.value}`)
  }
  parts.push(`最多 ${maxCount.value} 个文件`)
  return parts.join('，')
})

const displayFiles = computed(() => {
  return currentFiles.value.slice(0, MAX_DISPLAY_FILES)
})

// 获取 router
const router = computed(() => {
  if (!props.formRenderer) return ''
  return props.formRenderer.getFunctionRouter()
})

// 格式化文件大小
function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

// 判断文件是否为图片
function isImageFile(file: FileItem): boolean {
  if (!file.name) return false
  const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg', '.ico']
  const fileName = file.name.toLowerCase()
  return imageExtensions.some(ext => fileName.endsWith(ext))
}

// 获取文件预览URL
async function getPreviewUrl(file: FileItem): Promise<string> {
  let previewURL = file.url

  // 如果是完整的 http/https URL，直接返回
  if (previewURL && (previewURL.startsWith('http://') || previewURL.startsWith('https://'))) {
    return previewURL
  }

  // 否则构建下载URL
  if (!previewURL || (!previewURL.startsWith('http://') && !previewURL.startsWith('https://'))) {
    previewURL = `/api/v1/storage/download/${encodeURIComponent(file.url)}`
  }

  // 如果是相对路径，需要添加token，使用blob URL
  if (previewURL.startsWith('/')) {
    const token = localStorage.getItem('token') || ''
    try {
      const res = await fetch(previewURL, {
        headers: {
          'X-Token': token,
        },
      })
      if (res.ok) {
        const blob = await res.blob()
        return window.URL.createObjectURL(blob)
      } else {
        throw new Error(`Failed to load image: ${res.statusText}`)
      }
    } catch (error) {
      Logger.error('[FilesWidget]', 'Failed to load preview image', error)
      throw error
    }
  }

  return previewURL
}

// 预览图片
async function handlePreviewImage(file: FileItem): Promise<void> {
  if (!isImageFile(file)) {
    ElMessage.warning('该文件不是图片格式，无法预览')
    return
  }

  try {
    previewImageName.value = file.name || '预览图片'
    previewImageUrl.value = await getPreviewUrl(file)
    previewVisible.value = true
  } catch (error: any) {
    Logger.error('[FilesWidget]', 'Preview failed', error)
    ElMessage.error(`预览失败: ${error.message}`)
  }
}

// 关闭预览
function handleClosePreview(): void {
  previewVisible.value = false
  // 如果是blob URL，需要释放
  if (previewImageUrl.value.startsWith('blob:')) {
    window.URL.revokeObjectURL(previewImageUrl.value)
  }
  previewImageUrl.value = ''
  previewImageName.value = ''
}

// 显示文件详情
function handleShowFileDetail(file: FileItem): void {
  currentDetailFile.value = file
  fileDetailVisible.value = true
  
  // 如果是图片文件，自动加载预览URL
  if (isImageFile(file) && file.is_uploaded) {
    getPreviewUrl(file).then(url => {
      previewImageUrl.value = url
    }).catch(error => {
      Logger.error('[FilesWidget]', 'Failed to load preview URL', error)
    })
  }
}

// 关闭文件详情
function handleCloseFileDetail(): void {
  fileDetailVisible.value = false
  currentDetailFile.value = null
  // 清理预览URL
  if (previewImageUrl.value.startsWith('blob:')) {
    window.URL.revokeObjectURL(previewImageUrl.value)
  }
  previewImageUrl.value = ''
}

// 解析文件大小限制
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

  const size = match[1]
  const unit = match[2].toUpperCase() as keyof typeof units
  const unitValue = units[unit]
  if (!unitValue) {
    Logger.error('[FilesWidget]', `Unknown unit: ${unit}`)
    return Infinity
  }
  return parseFloat(size) * unitValue
}

// 验证文件
function validateFile(file: File): boolean {
  const maxSizeBytes = parseMaxSize(maxSize.value)
  const currentFilesCount = currentFiles.value.length

  // 检查数量限制
  if (currentFilesCount >= maxCount.value) {
    ElMessage.error(`最多只能上传 ${maxCount.value} 个文件`)
    return false
  }

  // 检查大小限制
  if (file.size > maxSizeBytes) {
    ElMessage.error(`文件大小不能超过 ${maxSize.value}`)
    return false
  }

  // 检查文件类型
  if (accept.value && accept.value !== '*') {
    const acceptList = accept.value.split(',').map(a => a.trim())
    const fileName = file.name.toLowerCase()
    const fileType = file.type.toLowerCase()

    const isAccepted = acceptList.some((pattern: string) => {
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
      ElMessage.error(`不支持的文件类型，仅支持：${accept.value}`)
      return false
    }
  }

  return true
}

// 处理文件选择
async function handleFileSelect(rawFile: File): Promise<void> {
  if (props.mode !== 'edit') {
    ElMessage.error('当前模式不支持文件上传')
    return
  }

  if (!router.value) {
    ElMessage.error('缺少函数路径，无法上传文件')
    return
  }

  if (!validateFile(rawFile)) {
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
    rawFile,
  }

  // 定义取消方法
  uploadingFile.cancel = () => {
    if (uploadingFile.uploader) {
      uploadingFile.uploader.cancel()
      uploadingFile.status = 'error'
      uploadingFile.error = '上传已取消'
      ElMessage.warning('上传已取消')
      setTimeout(() => {
        const index = uploadingFiles.value.findIndex((f: UploadingFile) => f.uid === uid)
        if (index !== -1) {
          uploadingFiles.value.splice(index, 1)
        }
      }, 2000)
    }
  }

  // 定义重试方法
  uploadingFile.retry = () => {
    if (uploadingFile.rawFile) {
      uploadingFile.status = 'uploading'
      uploadingFile.percent = 0
      uploadingFile.error = undefined
      uploadingFile.speed = '0 KB/s'
      handleFileSelect(uploadingFile.rawFile)
    }
  }

  uploadingFiles.value.push(uploadingFile)

  try {
    const uploadResult: UploadFileResult = await uploadFile(
      router.value,
      rawFile,
      (progress: UploadProgress) => {
        const file = uploadingFiles.value.find((f: UploadingFile) => f.uid === uid)
        if (file) {
          file.percent = progress.percent
          file.speed = progress.speed || '0 KB/s'
        }
      }
    )

    uploadingFile.uploader = uploadResult.uploader
    uploadingFile.fileInfo = uploadResult.fileInfo
    uploadingFile.storage = uploadResult.storage

    const file = uploadingFiles.value.find((f: UploadingFile) => f.uid === uid)
    if (file) {
      file.status = 'success'
    }

    // 添加到批量complete队列
    if (uploadResult.fileInfo) {
      if (!uploadResult.fileInfo.hash) {
        Logger.warn('[FilesWidget]', `File ${uploadResult.fileInfo.file_name} has no hash`, {
          key: uploadResult.fileInfo.key,
          fileInfo: uploadResult.fileInfo,
        })
      }
      addToCompleteQueue({
        key: uploadResult.fileInfo.key,
        success: true,
        router: uploadResult.fileInfo.router,
        file_name: uploadResult.fileInfo.file_name,
        file_size: uploadResult.fileInfo.file_size,
        content_type: uploadResult.fileInfo.content_type,
        hash: uploadResult.fileInfo.hash || '',
      })
    }
  } catch (error: any) {
    Logger.error('[FilesWidget]', 'Upload failed', error)

    const file = uploadingFiles.value.find((f: UploadingFile) => f.uid === uid)
    if (file) {
      file.status = 'error'
      file.error = error.message || '上传失败'
    }

    if (error.fileInfo) {
      addToCompleteQueue({
        key: error.fileInfo.key,
        success: false,
        error: error.fileInfo.error || error.message || '上传失败',
        router: error.fileInfo.router,
        file_name: error.fileInfo.file_name,
        file_size: error.fileInfo.file_size,
        content_type: error.fileInfo.content_type,
        hash: error.fileInfo.hash,
      })
    }

    ElMessage.error(`上传失败: ${error.message || '未知错误'}`)
  }
}

// 添加到批量complete队列
function addToCompleteQueue(item: BatchUploadCompleteItem): void {
  pendingCompleteQueue.value.push(item)

  if (pendingCompleteQueue.value.length >= BATCH_COMPLETE_MAX_SIZE) {
    flushCompleteQueue()
    return
  }

  if (batchCompleteTimer.value) {
    clearTimeout(batchCompleteTimer.value)
  }
  batchCompleteTimer.value = setTimeout(() => {
    flushCompleteQueue()
  }, BATCH_COMPLETE_DELAY)
}

// 批量complete处理
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

    items.forEach(item => {
      const result = results.get(item.key)
      const uploadingFile = uploadingFiles.value.find((f: UploadingFile) => f.fileInfo?.key === item.key)

      if (result && item.success && result.status === 'completed') {
        if (uploadingFile && uploadingFile.fileInfo) {
          uploadingFile.downloadURL = result.download_url || ''

          const newFile: FileItem = {
            name: uploadingFile.name,
            source_name: uploadingFile.name,
            storage: uploadingFile.storage || '',
            description: '',
            hash: result.hash || uploadingFile.fileInfo?.hash || '',
            size: uploadingFile.size,
            upload_ts: Date.now(),
            local_path: '',
            is_uploaded: true,
            url: result.download_url || '',
            server_url: result.server_download_url || '',
            downloaded: false,
          }

          const currentFilesList = currentFiles.value
          updateFiles([...currentFilesList, newFile])

          setTimeout(() => {
            const index = uploadingFiles.value.findIndex((f: UploadingFile) => f.uid === uploadingFile.uid)
            if (index !== -1) {
              uploadingFiles.value.splice(index, 1)
            }
          }, 2000)
        }
      } else if (!item.success || (result && result.status === 'failed')) {
        if (uploadingFile) {
          uploadingFile.status = 'error'
          uploadingFile.error = result?.error || item.error || '上传失败'
        }
      }
    })

    const successCount = items.filter(item => item.success && results.get(item.key)?.status === 'completed').length
    if (successCount > 1) {
      ElMessage.success(`批量上传完成：${successCount} 个文件`)
    } else if (successCount === 1) {
      ElMessage.success('上传成功')
    }
  } catch (error: any) {
    Logger.error('[FilesWidget]', 'Batch complete failed', error)
    items.forEach(item => {
      const uploadingFile = uploadingFiles.value.find((f: UploadingFile) => f.fileInfo?.key === item.key)
      if (uploadingFile) {
        uploadingFile.status = 'error'
        uploadingFile.error = '批量通知失败'
      }
    })
  }
}

// 更新文件列表
async function updateFiles(files: FileItem[]): Promise<void> {
  const currentValue = props.value
  const data = (currentValue?.raw as FilesData) || {
    files: [],
    remark: '',
    metadata: {},
    upload_user: '',
    widget_type: 'files',  // 固定值
    data_type: 'struct',   // 固定值
  }

  // 获取当前用户信息（如果还没有设置）
  let uploadUser = data.upload_user || ''
  if (!uploadUser) {
    try {
      // 优先从 localStorage 读取用户信息（不需要调用 API）
      const savedUserStr = localStorage.getItem('user')
      if (savedUserStr) {
        const savedUser = JSON.parse(savedUserStr)
        uploadUser = savedUser.username || ''
      }
      
      // 如果 localStorage 中没有，尝试从 authStore 获取
      if (!uploadUser) {
        const { useAuthStore } = await import('@/stores/auth')
        const authStore = useAuthStore()
        uploadUser = authStore.userName || authStore.user?.username || ''
      }
      
      if (!uploadUser) {
        console.warn('[FilesWidget] 无法获取用户信息：用户未登录或用户信息为空')
      }
    } catch (error) {
      console.warn('[FilesWidget] 无法获取用户信息', error)
    }
  }

  const newData: FilesData = {
    ...data,
    files,
    upload_user: uploadUser,
    widget_type: 'files',  // 固定值
    data_type: 'struct',   // 固定值
  }

  formDataStore.setValue(props.fieldPath, {
    raw: newData,
    display: `${files.length} 个文件`,
    meta: {},
  })
}

// 删除文件
function handleDeleteFile(index: number): void {
  const currentFilesList = currentFiles.value
  const newFiles = [...currentFilesList]
  newFiles.splice(index, 1)
  updateFiles(newFiles)
  ElMessage.success('删除成功')
}

// 下载文件
async function handleDownloadFile(file: FileItem): Promise<void> {
  try {
    let downloadURL = file.url

    if (!downloadURL || (!downloadURL.startsWith('http://') && !downloadURL.startsWith('https://'))) {
      downloadURL = `/api/v1/storage/download/${encodeURIComponent(file.url)}`
    }

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

    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.name || 'download'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)

    ElMessage.success('下载成功')
  } catch (error: any) {
    Logger.error('[FilesWidget]', 'Download failed', error)
    ElMessage.error(`下载失败: ${error.message}`)
  }
}

// 更新文件描述
function handleUpdateDescription(index: number, description: string): void {
  const currentFilesList = currentFiles.value
  if (index < 0 || index >= currentFilesList.length) {
    return
  }
  const newFiles = [...currentFilesList]
  const fileToUpdate = newFiles[index]
  if (fileToUpdate) {
    newFiles[index] = { ...fileToUpdate, description }
    updateFiles(newFiles)
  }
}

// 更新备注
function updateRemark(remarkValue: string): void {
  const currentValue = props.value
  const data = (currentValue?.raw as FilesData) || {
    files: [],
    remark: '',
    metadata: {},
  }

  const newData: FilesData = {
    ...data,
    remark: remarkValue,
  }

  formDataStore.setValue(props.fieldPath, {
    raw: newData,
    display: `${data.files.length} 个文件`,
    meta: {},
  })
}

function handleUpdateRemark(): void {
  updateRemark(remark.value)
}

// 拖拽处理
function handleDragOver(e: DragEvent): void {
  isDragging.value = true
}

function handleDragLeave(e: DragEvent): void {
  isDragging.value = false
}

function handleDrop(e: DragEvent): void {
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    Array.from(files).forEach(file => {
      handleFileSelect(file)
    })
  }
}

// 文件选择处理
function handleFileChange(file: any): void {
  if (file.raw) {
    handleFileSelect(file.raw)
  }
}
</script>

<style scoped>
.files-widget {
  width: 100%;
}

/* 上传区域 */
.upload-area {
  margin-bottom: 20px;
  background-color: var(--el-bg-color);
  border: 2px dashed var(--el-border-color);
  border-radius: 8px;
  padding: 24px;
  transition: all 0.3s ease;
  cursor: pointer;
}

.upload-area.is-dragging {
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
}

.upload-area:hover {
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
}

.upload-dragger-content {
  text-align: center;
}

.upload-icon {
  color: var(--el-text-color-secondary);
}

.el-upload__text {
  margin-top: 12px;
  font-size: 16px;
  color: var(--el-text-color-primary);
  font-weight: 500;
}

.el-upload__text em {
  color: var(--el-color-primary);
  font-style: normal;
  font-weight: 500;
  margin-left: 4px;
}

.el-upload__tip {
  margin-top: 8px;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

/* 上传中的文件 */
.uploading-files {
  margin-bottom: 20px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.uploading-file {
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 10px;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.file-icon {
  color: var(--el-color-primary);
}

.file-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  font-weight: 500;
  flex: 1;
}

.file-name-clickable {
  cursor: pointer;
  color: var(--el-color-primary);
  text-decoration: underline;
  transition: color 0.2s;
}

.file-name-clickable:hover {
  color: var(--el-color-primary-dark-2);
}

.file-size {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.file-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}

.upload-speed {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.upload-error {
  font-size: 12px;
  color: var(--el-color-danger);
  flex: 1;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

/* 已上传的文件 */
.uploaded-files {
  margin-bottom: 0;
}

.uploaded-file {
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 10px;
  transition: all 0.2s ease;
}

.uploaded-file:hover {
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
}

.file-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.file-description {
  margin-bottom: 8px;
}

/* 🔥 详情模式下的紧凑文件列表样式 */
.detail-file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  margin-bottom: 6px;
  background-color: var(--el-bg-color);
  transition: all 0.2s ease;
}

.detail-file-item.file-clickable {
  cursor: pointer;
}

.detail-file-item:hover {
  background-color: var(--el-fill-color-light);
  border-color: var(--el-color-primary);
}

.detail-file-item .file-icon {
  color: var(--el-color-primary);
  flex-shrink: 0;
}

.detail-file-item .file-name {
  flex: 1;
  font-size: 13px;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.detail-file-item .file-size {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

.detail-file-item .file-actions-inline {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
  margin-left: auto;
}

/* 🔥 表格单元格模式下的简化样式 */
.files-table-cell {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}

.file-names {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.file-name-inline {
  font-size: 12px;
  color: var(--el-text-color-primary);
}

.more-files {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 备注 */
.files-remark {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--el-border-color-lighter);
}

/* 响应模式 */
.response-files {
  width: 100%;
}

.empty-files {
  padding: 20px;
  text-align: center;
  color: var(--el-text-color-secondary);
}

.remark-content {
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: pre-wrap;
}

/* 表格单元格模式 */
.files-table-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 0;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background-color: #f5f7fa;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.file-item:hover {
  background-color: #e4e7ed;
}

.file-item .file-name {
  font-size: 12px;
  color: #606266;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.more-files {
  margin-top: 4px;
  color: #909399;
  font-size: 12px;
  font-style: italic;
}

.empty-text {
  color: #909399;
}

/* 详情模式 */
.detail-files {
  width: 100%;
}

/* 图片预览容器 */
.image-preview-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 200px;
  padding: 20px;
}

/* 文件详情对话框 */
.file-detail-content {
  padding: 10px 0;
}

.file-preview-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.file-preview-section .section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.loading-preview {
  text-align: center;
  padding: 40px;
  color: var(--el-text-color-secondary);
}

.file-detail-actions {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--el-border-color-lighter);
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}
</style>

