<template>
  <div class="simple-upload">
    <div
      v-if="!uploading && !fileUrl"
      class="upload-area"
      :class="{ 'is-disabled': disabled }"
      @click="handleClick"
      @drop="handleDrop"
      @dragover.prevent
      @dragenter.prevent
    >
      <input
        ref="fileInputRef"
        type="file"
        :accept="accept"
        :disabled="disabled"
        style="display: none"
        @change="handleFileChange"
      />
      <div class="upload-content">
        <el-icon class="upload-icon">
          <UploadFilled />
        </el-icon>
        <div class="upload-text">
          <p class="upload-tip">点击或拖拽文件到此区域上传</p>
          <p v-if="accept" class="upload-hint">支持：{{ accept }}</p>
          <p v-if="maxSize" class="upload-hint">最大：{{ maxSize }}</p>
        </div>
      </div>
    </div>
    
    <div v-if="uploading" class="upload-progress">
      <el-progress
        type="circle"
        :percentage="uploadProgress"
        :width="120"
      />
      <p class="progress-text">上传中... {{ uploadProgress }}%</p>
    </div>
    
    <div v-if="fileUrl && !uploading" class="preview-container">
      <el-image
        v-if="isImage"
        :src="fileUrl"
        fit="cover"
        class="preview-image"
        :preview-src-list="[fileUrl]"
      />
      <div v-else class="file-preview">
        <el-icon class="file-icon">
          <Document />
        </el-icon>
        <p class="file-name">{{ fileName }}</p>
      </div>
      <div class="preview-overlay">
        <el-button
          type="danger"
          :icon="Delete"
          circle
          size="small"
          @click="handleRemove"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Delete, Document } from '@element-plus/icons-vue'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import type { UploadProgress } from '@/utils/upload/types'

interface Props {
  /** 
   * 上传路由（用于文件存储路径和分类）
   * 示例：
   * - 头像：router="username/avatar"
   * - 文档：router="username/documents"
   * - 图片：router="username/images"
   * - 如果不提供，默认使用 "username/upload"（从 localStorage 获取用户名）
   */
  router?: string
  /** 接受的文件类型（如：image/*, .jpg,.png） */
  accept?: string
  /** 最大文件大小（如：2MB, 5MB） */
  maxSize?: string
  /** 当前文件 URL（用于回显） */
  modelValue?: string
  /** 是否禁用 */
  disabled?: boolean
}

interface Emits {
  (e: 'update:modelValue', url: string): void
  (e: 'change', url: string | null): void
  (e: 'success', url: string): void
  (e: 'error', error: Error): void
}

const props = withDefaults(defineProps<Props>(), {
  router: () => {
    // 默认路由：从 localStorage 获取用户名，使用通用上传路径
    const savedUserStr = localStorage.getItem('user')
    if (savedUserStr) {
      try {
        const savedUser = JSON.parse(savedUserStr)
        return `${savedUser.username || 'default'}/upload`
      } catch {
        return 'default/upload'
      }
    }
    return 'default/upload'
  },
  accept: 'image/*',
  maxSize: '5MB',
  disabled: false,
})

const emit = defineEmits<Emits>()

// 状态
const uploading = ref(false)
const uploadProgress = ref(0)
const fileUrl = ref<string>(props.modelValue || '')
const fileName = ref<string>('')
const fileInputRef = ref<HTMLInputElement | null>(null)

// 计算属性
const isImage = computed(() => {
  if (!fileUrl.value) return false
  const ext = fileUrl.value.split('.').pop()?.toLowerCase() || ''
  return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'].includes(ext)
})

/**
 * 解析文件大小限制
 */
function parseMaxSize(maxSizeStr: string): number {
  if (!maxSizeStr) return Infinity
  const size = parseFloat(maxSizeStr)
  const unit = maxSizeStr.replace(/[\d.]/g, '').toUpperCase()
  const unitMap: Record<string, number> = {
    'B': 1,
    'KB': 1024,
    'MB': 1024 * 1024,
    'GB': 1024 * 1024 * 1024,
  }
  const unitValue = unitMap[unit] || 1
  return size * unitValue
}

/**
 * 验证文件
 */
function validateFile(file: File): boolean {
  const maxSize = parseMaxSize(props.maxSize)
  
  // 检查文件大小
  if (file.size > maxSize) {
    ElMessage.error(`文件大小不能超过 ${props.maxSize}`)
    return false
  }
  
  // 检查文件类型
  if (props.accept && props.accept !== '*') {
    const accept = props.accept.split(',').map(a => a.trim())
    const fileName = file.name.toLowerCase()
    const fileType = file.type.toLowerCase()
    
    const isAccepted = accept.some((pattern: string) => {
      // 扩展名匹配：.jpg
      if (pattern.startsWith('.')) {
        return fileName.endsWith(pattern)
      }
      // MIME 通配符：image/*
      if (pattern.includes('/*')) {
        const prefix = pattern.split('/')[0]
        return prefix && fileType && fileType.startsWith(prefix)
      }
      // MIME 类型：image/jpeg
      return fileType === pattern
    })
    
    if (!isAccepted) {
      ElMessage.error(`不支持的文件类型，仅支持：${props.accept}`)
      return false
    }
  }
  
  return true
}

/**
 * 点击上传区域
 */
function handleClick(): void {
  if (props.disabled || uploading.value) {
    return
  }
  fileInputRef.value?.click()
}

/**
 * 处理文件选择（通过 input）
 */
function handleFileChange(event: Event): void {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    handleFileSelect(file)
    // 清空 input，允许重复选择同一文件
    target.value = ''
  }
}

/**
 * 处理拖拽
 */
function handleDrop(event: DragEvent): void {
  event.preventDefault()
  if (props.disabled || uploading.value) {
    return
  }
  const file = event.dataTransfer?.files?.[0]
  if (file) {
    handleFileSelect(file)
  }
}

/**
 * 处理文件选择（通过拖拽或点击）
 */
async function handleFileSelect(file: File): Promise<void> {
  if (props.disabled) {
    ElMessage.warning('上传已禁用')
    return
  }
  
  if (!validateFile(file)) {
    return
  }
  
  uploading.value = true
  uploadProgress.value = 0
  fileName.value = file.name
  
  try {
    // 使用统一上传工具
    const uploadResult = await uploadFile(
      props.router,
      file,
      (progress: UploadProgress) => {
        uploadProgress.value = progress.percent
      }
    )
    
    // 通知后端上传完成
    if (uploadResult.fileInfo) {
      const completeResult = await notifyUploadComplete({
        key: uploadResult.fileInfo.key,
        success: true,
        router: uploadResult.fileInfo.router,
        file_name: uploadResult.fileInfo.file_name,
        file_size: uploadResult.fileInfo.file_size,
        content_type: uploadResult.fileInfo.content_type,
        hash: uploadResult.fileInfo.hash,
      })
      
      if (completeResult?.download_url) {
        const downloadUrl = completeResult.download_url
        fileUrl.value = downloadUrl
        emit('update:modelValue', downloadUrl)
        emit('change', downloadUrl)
        emit('success', downloadUrl)
        ElMessage.success('上传成功')
      } else {
        throw new Error('获取下载地址失败')
      }
    }
  } catch (error: any) {
    console.error('[SimpleUpload] 上传失败:', error)
    const errorMessage = error?.message || '上传失败'
    ElMessage.error(errorMessage)
    emit('error', error instanceof Error ? error : new Error(errorMessage))
    emit('change', null)
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

/**
 * 移除文件
 */
function handleRemove(): void {
  fileUrl.value = ''
  emit('update:modelValue', '')
  emit('change', null)
}

// 监听 modelValue 变化
import { watch } from 'vue'
watch(() => props.modelValue, (newValue) => {
  fileUrl.value = newValue || ''
})

// 监听文件选择（通过 el-upload 的 before-upload 触发）
// 由于我们阻止了默认上传，需要手动处理
// 这里我们需要修改 el-upload 的使用方式
</script>

<style scoped>
.simple-upload {
  width: 100%;
}

.upload-area {
  width: 100%;
  min-height: 180px;
  border: 2px dashed var(--el-border-color);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.3s;
  background: var(--el-fill-color-lighter);
}

.upload-area:hover:not(.is-disabled) {
  border-color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}

.upload-area.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.upload-icon {
  font-size: 48px;
  color: var(--el-text-color-secondary);
  margin-bottom: 16px;
}

.upload-text {
  text-align: center;
}

.upload-tip {
  font-size: 14px;
  color: var(--el-text-color-primary);
  margin: 8px 0;
}

.upload-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 4px 0;
}

.upload-progress {
  width: 100%;
  min-height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.progress-text {
  margin-top: 16px;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.preview-container {
  position: relative;
  width: 100%;
  min-height: 180px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: hidden;
}

.preview-image {
  width: 100%;
  height: 180px;
  object-fit: cover;
}

.file-preview {
  width: 100%;
  height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-lighter);
}

.file-icon {
  font-size: 64px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.file-name {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.3s;
}

.preview-container:hover .preview-overlay {
  opacity: 1;
}
</style>

