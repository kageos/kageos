<template>
  <div :class="['mini-key-info-body', { 'mini-key-info-body--compact': compact }]">
    <template v-if="uploadedFiles.length > 0">
      <div class="mini-file-section-title">
        <el-icon :size="compact ? 12 : 13"><UploadFilled /></el-icon>
        上传文件 ({{ uploadedFiles.length }})
      </div>
      <MiniWorkstationFileCard
        v-for="(file, index) in uploadedFiles"
        :key="'u' + index"
        :file="file"
        :compact="compact"
        @preview="$emit('preview-file', file)"
        @download="$emit('download-file', file)"
      />
    </template>

    <template v-if="outputFiles.length > 0">
      <div class="mini-file-section-title">
        <el-icon :size="compact ? 12 : 13"><FolderOpened /></el-icon>
        输出文件 ({{ outputFiles.length }})
      </div>
      <MiniWorkstationFileCard
        v-for="(file, index) in outputFiles"
        :key="'o' + index"
        :file="file"
        :compact="compact"
        @preview="$emit('preview-file', file)"
        @download="$emit('download-file', file)"
      />
    </template>

    <template v-if="displayFields.length > 0">
      <div class="mini-file-section-title">
        <el-icon :size="compact ? 12 : 13"><Memo /></el-icon>
        输出数据 ({{ displayFields.length }})
      </div>
      <MiniWorkstationDisplayFieldCard
        v-for="(field, index) in displayFields"
        :key="'df' + index"
        :field="field"
        :compact="compact"
        @preview="$emit('preview-field', field)"
        @copy="$emit('copy-field', field)"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { FolderOpened, Memo, UploadFilled } from '@element-plus/icons-vue'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import MiniWorkstationDisplayFieldCard from './MiniWorkstationDisplayFieldCard.vue'
import MiniWorkstationFileCard from './MiniWorkstationFileCard.vue'
import type { FilePanelItem } from '@/architecture/presentation/composables/useMiniWorkstationPanel'

withDefaults(defineProps<{
  compact?: boolean
  uploadedFiles: FilePanelItem[]
  outputFiles: FilePanelItem[]
  displayFields: OutputDisplayField[]
}>(), {
  compact: false,
})

defineEmits<{
  (e: 'preview-file', file: FilePanelItem): void
  (e: 'download-file', file: FilePanelItem): void
  (e: 'preview-field', field: OutputDisplayField): void
  (e: 'copy-field', field: OutputDisplayField): void
}>()
</script>

<style scoped>
.mini-key-info-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.mini-key-info-body--compact {
  max-height: 360px;
}

.mini-file-section-title {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  padding: 6px 4px 4px;
  margin-top: 4px;
}

.mini-file-section-title:first-child {
  margin-top: 0;
}
</style>
