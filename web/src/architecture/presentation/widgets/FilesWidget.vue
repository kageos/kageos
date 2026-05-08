<!--
  FilesWidget - 文件上传组件
  支持多文件上传、拖拽上传、文件管理
-->
<template>
  <div class="files-widget">
    <!-- 编辑模式 -->
    <template v-if="mode === 'edit'">
      <div v-if="!isDisabled" class="files-editor-shell">
        <!-- 上传区域 -->
        <div
          v-if="!isDisabled && !isMaxReached"
          class="upload-area"
          @drop.prevent="handleDrop"
          @dragover.prevent="handleDragOver"
          @dragleave.prevent="handleDragLeave"
          :class="{ 'is-dragging': isDragging }"
        >
          <div class="upload-area-summary">
            <div class="upload-area-title-row">
              <div class="upload-area-title">上传文件</div>
              <div class="upload-area-count">{{ currentFiles.length }}/{{ maxCount }}</div>
            </div>
            <div class="upload-area-meta">
              <span class="upload-meta-badge">
                {{ maxSize ? `单文件 ≤ ${maxSize}` : '大小不限' }}
              </span>
              <span class="upload-meta-badge">
                {{ accept === '*' ? '任意格式' : accept }}
              </span>
            </div>
          </div>
          <el-upload
            ref="uploadRef"
            :auto-upload="false"
            :show-file-list="false"
            :drag="true"
            :multiple="true"
            :accept="accept"
            :on-change="handleFileChange"
            @drop.native.stop.prevent="handleElUploadDrop"
            @dragover.native.stop.prevent
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
        <div v-else-if="!isDisabled && isMaxReached" class="upload-limit-tip">
          <el-icon class="limit-tip-icon"><Document /></el-icon>
          <span>已达到上传上限（{{ currentFiles.length }}/{{ maxCount }}），请先删除已有文件</span>
        </div>

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
        <div class="files-list">
          <FilesWidgetFileCard
            v-for="(file, index) in currentFiles"
            :key="file.ref || file.download_url || file.name || index"
            :file="file"
            :icon-component="getFileIcon(file.name)"
            :icon-color="getFileIconColor(file.name)"
            :is-image="isImageFile(file)"
            :image-src="getFileDisplayUrl(file)"
            :can-open-in-browser="canPreviewInBrowser(file)"
            :size-text="formatSize(file.size)"
            :upload-time-text="file.upload_ts ? formatTimestamp(file.upload_ts) : ''"
            :show-description-placeholder="true"
            :show-upload-status-tag="true"
            :enable-image-preview-list="true"
            :preview-image-list="previewImageList"
            :preview-index="getPreviewImageIndex(file)"
            :show-preview-action="isImageFile(file) && file.is_uploaded"
            :show-edit-description-action="file.is_uploaded"
            :show-delete-action="!isDisabled"
            @open-browser="handlePreviewInNewWindow(file)"
            @preview-image="handlePreviewImage(file)"
            @edit-description="handleEditDescription(index)"
            @update-description-draft="updateEditingDescription"
            @save-description="handleSaveDescription"
            @cancel-description="handleCancelDescription"
            @delete-file="handleDeleteFile(index)"
            :is-description-editing="editingDescriptionIndex === index"
            :description-draft="editingDescription"
          />
        </div>
      </div>
    </template>

    <!-- 响应/详情模式 -->
    <template v-else-if="isReadonlyMode">
      <div class="detail-files">
        <div v-if="currentFiles.length > 0" class="uploaded-files">
          <div class="detail-files-header">
            <div class="header-left">
              <div class="section-title">
                已上传文件 ({{ currentFiles.length }})
              </div>
            </div>
          </div>

          <div class="files-list">
            <FilesWidgetFileCard
              v-for="(file, index) in currentFiles"
              :key="file.ref || file.download_url || file.name || index"
              :file="file"
              :icon-component="getFileIcon(file.name)"
              :icon-color="getFileIconColor(file.name)"
              :is-image="isImageFile(file)"
              :image-src="getFileDisplayUrl(file)"
              :can-open-in-browser="canPreviewInBrowser(file)"
              :size-text="formatSize(file.size)"
              :upload-time-text="file.upload_ts ? formatTimestamp(file.upload_ts) : ''"
              :show-upload-user="true"
              :upload-user-info="getFileUploadUserInfo(file)"
              :show-download-action="file.is_uploaded"
              @open-browser="handlePreviewInNewWindow(file)"
              @download-file="handleDownloadFile(file)"
            />
          </div>
        </div>
        <div v-else class="empty-files">暂无文件</div>

      </div>
    </template>

    <!-- 表格单元格模式 -->
    <template v-else-if="mode === 'table-cell'">
      <div v-if="currentFiles.length > 0" class="files-table-cell">
        <!-- 🔥 完全照抄用户组件搜索框选中样式 -->
        <div class="files-select-display">
          <el-icon :size="20" class="files-icon-small">
            <Document />
          </el-icon>
          <span class="files-display-text">{{ currentFiles.length }} 个文件</span>
        </div>
      </div>
      <span v-else class="empty-text">-</span>
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

  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  ElUpload,
  ElButton,
  ElIcon,
  ElProgress,
  ElDialog,
  ElImage,
} from 'element-plus'
import {
  Upload,
  Document,
} from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import type { FilesWidgetConfig } from '@/core/types/widget-configs'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { useAuthStore } from '@/stores/auth'
import { useUserInfoStore } from '@/stores/userInfo'
import { resolveFileRefs, updateFileDescription } from '@/api/storage'
import { Logger } from '@/core/utils/logger'
import { formatTimestamp } from '@/utils/date'
import { useFilesDescriptionDialog } from './composables/useFilesDescriptionDialog'
import { useFilesPreviewAndActions } from './composables/useFilesPreviewAndActions'
import { useFilesUploadManager } from './composables/useFilesUploadManager'
import { useFilesUploadUsers } from './composables/useFilesUploadUsers'
import { useFilesValueSync } from './composables/useFilesValueSync'
import FilesWidgetFileCard from './FilesWidgetFileCard.vue'
import type { FileItem } from './filesWidgetTypes'
import { fileNameFromRef, parseFileRefs } from './filesWidgetTypes'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: '',
    display: '0 个文件',
    meta: {},
  }),
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()
const authStore = useAuthStore()
const userInfoStore = useUserInfoStore()

// 获取配置（带类型）
const filesConfig = computed(() => {
  return (props.field.widget?.config || {}) as FilesWidgetConfig
})
const accept = computed(() => filesConfig.value.accept || '*')
const maxSize = computed(() => filesConfig.value.max_size)
const maxCount = computed(() => filesConfig.value.max_count || 5)
const isReadonlyMode = computed(() => props.mode === 'response' || props.mode === 'detail')

const currentRefs = computed(() => parseFileRefs(props.value?.raw))
const resolvedFiles = ref<FileItem[]>([])

const currentFiles = computed<FileItem[]>(() => resolvedFiles.value)

function buildPlaceholderFile(refValue: string): FileItem {
  const parts = refValue.split('/')
  return {
    ref: refValue,
    bucket: parts[0] || '',
    key: parts.slice(1).join('/'),
    name: fileNameFromRef(refValue),
    source_name: fileNameFromRef(refValue),
    size: 0,
    is_uploaded: true,
  }
}

watch(currentRefs, async (refs) => {
  if (refs.length === 0) {
    resolvedFiles.value = []
    return
  }

  resolvedFiles.value = refs.map(buildPlaceholderFile)

  try {
    const resolved = await resolveFileRefs(refs, 'browser')
    const byRef = new Map(resolved.map(item => [item.ref, item]))
    resolvedFiles.value = refs.map((refValue) => {
      const item = byRef.get(refValue)
      if (!item) return buildPlaceholderFile(refValue)
      return {
        ref: item.ref,
        bucket: item.bucket,
        key: item.key,
        name: item.name || fileNameFromRef(refValue),
        source_name: item.source_name || item.name || fileNameFromRef(refValue),
        storage: item.storage,
        description: item.description || '',
        hash: item.hash || '',
        size: item.size || 0,
        upload_ts: item.upload_ts,
        content_type: item.content_type,
        is_uploaded: true,
        download_url: item.download_url || '',
        server_download_url: item.server_download_url || '',
        upload_user: item.upload_user,
        error: item.error,
      }
    })
  } catch (error) {
    Logger.error('FilesWidget', '解析文件引用失败', error)
  }
}, { immediate: true })

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

const { getFileUploadUserInfo } = useFilesUploadUsers({
  mode: () => props.mode,
  currentFiles,
  userInfoStore,
})

const router = computed(() => {
  if (!props.formRenderer) return ''
  return props.formRenderer.getFunctionRouter()
})

function resolveCurrentUploadUser(): string {
  try {
    const savedUserStr = localStorage.getItem('user')
    if (savedUserStr) {
      const savedUser = JSON.parse(savedUserStr) as { username?: string }
      if (savedUser.username) {
        return savedUser.username
      }
    }

    const storeUser = authStore.userName || authStore.user?.username || ''
    if (storeUser) {
      return storeUser
    }
  } catch (error) {
    Logger.warn('FilesWidget', '无法获取用户信息', error)
  }

  Logger.warn('FilesWidget', '无法获取用户信息：用户未登录或用户信息为空')
  return ''
}

const {
  updateFiles,
  handleDeleteFile,
  handleUpdateDescription,
} = useFilesValueSync({
  value: () => props.value,
  fieldPath: () => props.fieldPath,
  currentFiles: () => currentFiles.value,
  setCurrentFiles: (files) => {
    resolvedFiles.value = files
  },
  persistDescription: async (file, description) => {
    try {
      await updateFileDescription(file.ref, description)
    } catch (error) {
      Logger.error('FilesWidget', '保存文件描述失败', error)
      throw error
    }
  },
  formDataStore,
  emitUpdateModelValue: (value) => emit('update:modelValue', value),
})

const {
  previewVisible,
  previewImageUrl,
  previewImageName,
  previewImageList,
  formatSize,
  isImageFile,
  canPreviewInBrowser,
  getFileDisplayUrl,
  getFileIcon,
  getFileIconColor,
  handlePreviewInNewWindow,
  getPreviewImageIndex,
  handlePreviewImage,
  handleClosePreview,
  handleDownloadFile
} = useFilesPreviewAndActions({
  currentFiles
})

const {
  isDragging,
  uploadingFiles,
  handleDragOver,
  handleDragLeave,
  handleDrop,
  handleElUploadDrop,
  handleFileChange,
} = useFilesUploadManager({
  mode: () => props.mode,
  router: () => router.value,
  accept,
  maxSize,
  maxCount,
  currentFiles,
  updateFiles,
  resolveUploadUser: resolveCurrentUploadUser,
})

const {
  editingDescriptionIndex,
  editingDescription,
  handleEditDescription,
  updateEditingDescription,
  handleSaveDescription,
  handleCancelDescription,
} = useFilesDescriptionDialog({
  currentFiles,
  handleUpdateDescription,
})

</script>

<style scoped>
.files-widget {
  width: 100%;
}

.files-editor-shell {
  margin-bottom: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-bg-color);
  overflow: hidden;
}

/* 上传区域 */
.upload-area {
  margin-bottom: 0;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--el-color-primary) 3%, var(--el-bg-color)) 0%, var(--el-bg-color) 100%);
  border: none;
  border-bottom: 1px solid var(--el-border-color-lighter);
  border-radius: 0;
  padding: 12px 14px;
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

.upload-area-summary {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 9px;
}

.upload-area-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.upload-area-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.upload-area-count {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}

.upload-area-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.upload-meta-badge {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 100%;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 1.2;
}

.upload-limit-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
  padding: 14px 16px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 13px;
}

.limit-tip-icon {
  color: var(--el-color-warning);
  flex-shrink: 0;
}

.upload-dragger-content {
  min-height: 68px;
  padding: 8px 10px;
  box-sizing: border-box;
  text-align: center;
}

.upload-icon {
  font-size: 32px !important;
  color: var(--el-text-color-secondary);
}

.el-upload__text {
  margin-top: 6px;
  font-size: 13px;
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
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 上传中的文件 */
.uploading-files {
  margin-bottom: 12px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.uploading-file {
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 10px;
  margin-bottom: 8px;
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

/* 🔥 详情模式：参考旧版本的卡片式布局 */
.detail-files-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.files-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
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

/* 响应模式 */
.response-files {
  width: 100%;
}

.empty-files {
  padding: 20px;
  text-align: center;
  color: var(--el-text-color-secondary);
}

/* 表格单元格模式 */
.files-table-cell {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 🔥 完全照抄用户组件搜索框选中样式 */
.files-select-display {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--el-bg-color);
  border-radius: 4px;
  padding: 2px 8px;
}

.files-select-display .files-icon-small {
  flex-shrink: 0;
  color: var(--el-color-primary);
}

.files-select-display .files-display-text {
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
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

</style>
