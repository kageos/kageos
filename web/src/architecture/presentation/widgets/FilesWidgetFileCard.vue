<template>
  <div
    class="file-card-shell"
    :class="{
      'file-clickable': canOpenInBrowser,
      'is-editing-description': showInlineDescriptionEditor,
    }"
  >
    <div
      class="file-list-item"
      :class="{ 'file-clickable': canOpenInBrowser }"
      @click="handleCardClick"
    >
      <div v-if="showUploadUser && file.upload_user" class="file-upload-user" @click.stop>
        <UserDisplay
          :user-info="uploadUserInfo"
          :username="file.upload_user"
          mode="card"
          layout="vertical"
          :size="24"
        />
      </div>

      <div class="file-thumbnail">
        <el-image
          v-if="enableImagePreviewList && isImage && file.is_uploaded && imageSrc"
          :src="imageSrc"
          fit="contain"
          class="thumbnail-image"
          :preview-src-list="previewImageList"
          :initial-index="previewIndex"
          preview-teleported
          hide-on-click-modal
          @click.stop
        />
        <el-image
          v-else-if="isImage && file.is_uploaded && imageSrc"
          :src="imageSrc"
          fit="contain"
          class="thumbnail-image"
        />
        <el-icon
          v-else
          :size="32"
          :style="{ color: iconColor }"
          class="thumbnail-icon"
        >
          <component :is="iconComponent" />
        </el-icon>
      </div>

      <div class="file-info">
        <div
          class="file-name"
          :class="{ 'file-name-clickable': canOpenInBrowser }"
          :title="file.name"
        >
          {{ file.name }}
        </div>

        <div
          v-if="file.description && file.description.trim() && !showInlineDescriptionEditor"
          class="file-description-text"
        >
          <el-icon :size="12" class="description-icon">
            <Edit />
          </el-icon>
          <span class="description-content">{{ file.description }}</span>
        </div>

        <div
          v-else-if="showDescriptionPlaceholder && file.is_uploaded && !showInlineDescriptionEditor"
          class="file-description-placeholder"
        >
          <el-icon :size="12" class="description-icon">
            <Edit />
          </el-icon>
          <span class="description-hint">直接在当前文件卡片里补充备注</span>
        </div>

        <div class="file-meta">
          <span class="file-size">{{ sizeText }}</span>
          <el-tag
            v-if="canOpenInBrowser"
            size="small"
            type="success"
            effect="plain"
            class="preview-tag"
          >
            <el-icon :size="12" class="meta-tag-icon">
              <View />
            </el-icon>
            可预览
          </el-tag>
          <el-tag v-if="showUploadStatusTag" size="small" :type="file.is_uploaded ? 'success' : 'info'">
            {{ file.is_uploaded ? '已上传' : '本地' }}
          </el-tag>
          <span v-if="showUploadTime && uploadTimeText" class="file-upload-time">
            {{ uploadTimeText }}
          </span>
        </div>
      </div>

      <div v-if="hasActions" class="file-actions">
        <el-button
          v-if="showPreviewAction && file.is_uploaded && isImage"
          size="small"
          :icon="View"
          @click.stop="emit('preview-image')"
        >
          预览
        </el-button>
        <el-button
          v-if="showEditDescriptionAction && file.is_uploaded"
          size="small"
          :type="file.description && file.description.trim() ? 'default' : 'primary'"
          :plain="!(file.description && file.description.trim())"
          :icon="Edit"
          @click.stop="emit('edit-description')"
        >
          {{ showInlineDescriptionEditor ? '收起备注' : file.description?.trim() ? '编辑备注' : '添加备注' }}
        </el-button>
        <el-button
          v-if="showDownloadAction && file.is_uploaded"
          size="small"
          type="primary"
          :icon="Download"
          @click.stop="emit('download-file')"
        >
          下载
        </el-button>
        <el-popconfirm
          v-if="showDeleteAction"
          title="确定删除此文件？"
          @confirm="emit('delete-file')"
        >
          <template #reference>
            <el-button size="small" type="danger" :icon="Delete" @click.stop>
              删除
            </el-button>
          </template>
        </el-popconfirm>
      </div>
    </div>

    <div v-if="showInlineDescriptionEditor" class="file-description-editor" @click.stop>
      <div class="description-editor-header">
        <div class="description-editor-title">文件备注</div>
        <div class="description-editor-tip">和文件一起保存，后续查看也会直接展示在卡片里。</div>
      </div>
      <el-input
        :model-value="descriptionDraft"
        type="textarea"
        :rows="3"
        placeholder="补充这个文件的用途、版本或注意事项"
        maxlength="500"
        show-word-limit
        @update:model-value="handleDescriptionDraftUpdate"
      />
      <div class="description-editor-actions">
        <el-button size="small" @click="emit('cancel-description')">取消</el-button>
        <el-button size="small" type="primary" @click="emit('save-description')">保存备注</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElButton, ElIcon, ElImage, ElInput, ElPopconfirm, ElTag } from 'element-plus'
import { Delete, Download, Edit, View } from '@element-plus/icons-vue'
import type { UserInfo } from '@/types'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import type { FileItem } from './filesWidgetTypes'

interface Props {
  file: FileItem
  iconComponent: any
  iconColor: string
  isImage: boolean
  imageSrc?: string
  canOpenInBrowser: boolean
  sizeText: string
  uploadTimeText?: string
  showUploadUser?: boolean
  uploadUserInfo?: UserInfo | null
  showDescriptionPlaceholder?: boolean
  showUploadStatusTag?: boolean
  showUploadTime?: boolean
  enableImagePreviewList?: boolean
  previewImageList?: string[]
  previewIndex?: number
  showPreviewAction?: boolean
  showEditDescriptionAction?: boolean
  showDownloadAction?: boolean
  showDeleteAction?: boolean
  isDescriptionEditing?: boolean
  descriptionDraft?: string
}

const props = withDefaults(defineProps<Props>(), {
  uploadTimeText: '',
  showUploadUser: false,
  uploadUserInfo: null,
  imageSrc: '',
  showDescriptionPlaceholder: false,
  showUploadStatusTag: false,
  showUploadTime: true,
  enableImagePreviewList: false,
  previewImageList: () => [],
  previewIndex: 0,
  showPreviewAction: false,
  showEditDescriptionAction: false,
  showDownloadAction: false,
  showDeleteAction: false,
  isDescriptionEditing: false,
  descriptionDraft: '',
})

const emit = defineEmits<{
  (e: 'open-browser'): void
  (e: 'preview-image'): void
  (e: 'edit-description'): void
  (e: 'update-description-draft', value: string): void
  (e: 'save-description'): void
  (e: 'cancel-description'): void
  (e: 'download-file'): void
  (e: 'delete-file'): void
}>()

const hasActions = computed(() => {
  return props.showPreviewAction || props.showEditDescriptionAction || props.showDownloadAction || props.showDeleteAction
})

const showInlineDescriptionEditor = computed(() => {
  return props.showEditDescriptionAction && props.file.is_uploaded && props.isDescriptionEditing
})

function handleDescriptionDraftUpdate(value: string | number): void {
  emit('update-description-draft', String(value ?? ''))
}

function handleCardClick(): void {
  if (!props.canOpenInBrowser) {
    return
  }
  emit('open-browser')
}
</script>

<style scoped>
.file-card-shell {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background-color: var(--el-bg-color);
  transition: all 0.2s ease;
  overflow: hidden;
}

.file-card-shell.file-clickable {
  cursor: pointer;
}

.file-card-shell:hover,
.file-card-shell.is-editing-description {
  border-color: var(--el-color-primary);
  background-color: var(--el-fill-color-light);
}

.file-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background-color: transparent;
}

.file-upload-user {
  flex-shrink: 0;
  margin-right: 12px;
  min-width: 80px;
}

.file-thumbnail {
  width: 60px;
  height: 60px;
  flex-shrink: 0;
  box-sizing: border-box;
  padding: 4px;
  border-radius: 6px;
  overflow: hidden;
  background-color: var(--el-fill-color-light);
  display: flex;
  align-items: center;
  justify-content: center;
}

.thumbnail-image {
  width: 100%;
  height: 100%;
  display: block;
  min-width: 0;
  min-height: 0;
  box-sizing: border-box;
  background-color: var(--el-fill-color-light);
}

.thumbnail-image :deep(.el-image__wrapper) {
  width: 100%;
  height: 100%;
}

.thumbnail-image :deep(.el-image__inner) {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: contain;
  object-position: center;
}

.thumbnail-icon {
  flex-shrink: 0;
}

.file-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-name-clickable {
  color: var(--el-color-primary);
}

.file-description-text {
  display: flex;
  align-items: flex-start;
  gap: 4px;
  margin-top: 4px;
  margin-bottom: 2px;
  padding: 4px 8px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
}

.file-description-text .description-icon {
  flex-shrink: 0;
  margin-top: 2px;
  color: var(--el-text-color-placeholder);
}

.file-description-text .description-content {
  flex: 1;
  word-break: break-word;
}

.file-description-placeholder {
  display: flex;
  align-items: flex-start;
  gap: 4px;
  margin-top: 4px;
  margin-bottom: 2px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  line-height: 1.5;
}

.file-description-placeholder .description-icon {
  flex-shrink: 0;
  margin-top: 2px;
}

.file-description-placeholder .description-hint {
  flex: 1;
  font-style: italic;
}

.file-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex-wrap: wrap;
}

.file-size {
  flex-shrink: 0;
}

.preview-tag {
  flex-shrink: 0;
}

.meta-tag-icon {
  margin-right: 4px;
}

.file-upload-time {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}

.file-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.file-description-editor {
  padding: 0 14px 14px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color));
}

.description-editor-header {
  margin-bottom: 10px;
  padding-top: 12px;
}

.description-editor-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.description-editor-tip {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.description-editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}
</style>
