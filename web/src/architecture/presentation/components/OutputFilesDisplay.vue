<!--
  OutputFilesDisplay - 输出文件展示组件（通用）
  用于工作台工具结果、或任意返回「含 files 数组」的 JSON 时的文件展示：预览、下载、打开。
  支持传入 result（原始 JSON 字符串/对象）或已解析的 fileGroups。
-->
<template>
  <div v-if="displayGroups.length > 0" class="output-files-display">
    <div class="output-files-head">
      <el-icon><FolderOpened /></el-icon>
      <span>{{ sectionTitle }}</span>
    </div>
    <div class="output-files-wrap">
      <div v-for="(group, gIdx) in displayGroups" :key="gIdx" class="output-files-group">
        <div v-if="displayGroups.length > 1" class="output-files-group-label">{{ group.label }}</div>
        <div class="output-files-list">
          <div
            v-for="(file, fIdx) in group.files"
            :key="fIdx"
            class="output-files-item"
          >
            <div class="output-files-preview" v-if="isImageFile(file)">
              <a :href="file.download_url" target="_blank" rel="noopener noreferrer" class="output-files-preview-link">
                <img
                  :src="file.download_url"
                  :alt="fileDisplayName(file)"
                  loading="lazy"
                  class="output-files-img"
                  @error="onImageError"
                />
              </a>
            </div>
            <div v-else class="output-files-icon">
              <el-icon><Document /></el-icon>
            </div>
            <div class="output-files-info">
              <a :href="file.download_url" target="_blank" rel="noopener noreferrer" class="output-files-name">
                {{ fileDisplayName(file) }}
              </a>
              <span class="output-files-meta">
                <span v-if="fileFormat(file)" class="output-files-format">{{ fileFormat(file) }}</span>
                <span v-if="file.size != null" class="output-files-size">{{ formatFileSize(file.size) }}</span>
              </span>
              <div class="output-files-actions">
                <el-link type="primary" :href="file.download_url" target="_blank" rel="noopener noreferrer">打开</el-link>
                <el-link type="primary" :href="file.download_url" target="_blank" rel="noopener noreferrer" download>下载</el-link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Document, FolderOpened } from '@element-plus/icons-vue'
import { resolveFileRefs, type ResolvedFile } from '@/api/storage'
import type { ToolResultMetadata } from '@/api/workspace'
import { extractFileGroupsFromResult, type OutputFileGroup, type OutputFileItem } from '@/architecture/presentation/composables/useOutputFileGroups'

const IMAGE_EXT = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg'])

const props = withDefaults(
  defineProps<{
    /** 原始结构化返回（JSON 字符串或对象），需配合 metadata.display_file_fields 解析文件字段 */
    result?: string | object
    /** 工具结果元数据，声明哪些 result 字段按文件展示 */
    metadata?: ToolResultMetadata
    /** 已解析的文件组，若传入则优先使用，不解析 result */
    fileGroups?: OutputFileGroup[]
    /** 区块标题，如「输出文件」「上传的文件」 */
    sectionTitle?: string
  }>(),
  { sectionTitle: '输出文件' }
)

const sourceGroups = computed((): OutputFileGroup[] => {
  if (props.fileGroups != null && props.fileGroups.length > 0) return props.fileGroups
  return extractFileGroupsFromResult(props.result, props.metadata)
})

const resolvedGroups = ref<OutputFileGroup[]>([])
const resolvedByRef = new Map<string, ResolvedFile>()
let resolveSeq = 0

const sourceGroupsSignature = computed(() => {
  return JSON.stringify(sourceGroups.value.map(group => ({
    label: group.label,
    files: group.files.map(file => ({
      ref: file.ref ?? '',
      name: file.name ?? '',
      source_name: file.source_name ?? '',
      download_url: file.download_url ?? '',
      size: file.size ?? null,
    })),
  })))
})

watch(sourceGroupsSignature, async () => {
  const seq = ++resolveSeq
  const groups = sourceGroups.value
  const refs = Array.from(new Set(groups.flatMap(group => group.files.map(file => file.ref).filter(Boolean)))) as string[]
  resolvedGroups.value = mergeResolvedGroups(groups)
  if (refs.length === 0) {
    return
  }
  const unresolvedRefs = refs.filter(ref => !resolvedByRef.has(ref))
  if (unresolvedRefs.length === 0) {
    return
  }
  try {
    const resolved = await resolveFileRefs(unresolvedRefs, 'browser')
    for (const file of resolved) {
      resolvedByRef.set(file.ref, file)
    }
    if (seq === resolveSeq) {
      resolvedGroups.value = mergeResolvedGroups(sourceGroups.value)
    }
  } catch {
    if (seq === resolveSeq) {
      resolvedGroups.value = groups
    }
  }
}, { immediate: true })

/** 展示用的文件组：优先 fileGroups，否则从 result 解析 */
const displayGroups = computed((): OutputFileGroup[] => resolvedGroups.value)

function mergeResolvedGroups(groups: OutputFileGroup[]): OutputFileGroup[] {
  return groups.map(group => ({
    ...group,
    files: group.files.map((file) => {
      if (!file.ref) return file
      const item = resolvedByRef.get(file.ref)
      if (!item) return file
      return {
        ...file,
        name: item.name || file.name,
        source_name: item.source_name || file.source_name || item.name,
        size: item.size ?? file.size,
        download_url: item.download_url || file.download_url,
      }
    })
  }))
}

function isImageFile(file: OutputFileItem): boolean {
  const name = (file.source_name ?? file.name ?? '') as string
  const ext = name.includes('.') ? name.slice(name.lastIndexOf('.')).toLowerCase() : ''
  return IMAGE_EXT.has(ext)
}

function fileDisplayName(file: OutputFileItem): string {
  return (file.source_name ?? file.name ?? '文件') as string
}

/** 从文件名解析扩展名，用于展示格式（如 PDF、PNG、XLSX） */
function fileFormat(file: OutputFileItem): string {
  const name = (file.source_name ?? file.name ?? '') as string
  if (!name || !name.includes('.')) return ''
  const ext = name.slice(name.lastIndexOf('.') + 1).toUpperCase()
  return ext || ''
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function onImageError(e: Event) {
  const el = e.target as HTMLImageElement
  if (el) el.style.display = 'none'
}
</script>

<style scoped lang="scss">
.output-files-display {
  margin-bottom: 12px;

  &:last-child {
    margin-bottom: 0;
  }
}

.output-files-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;

  .el-icon {
    font-size: 14px;
  }
}

.output-files-wrap {
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-small);
  padding: 12px;
}

.output-files-group {
  margin-bottom: 12px;

  &:last-child {
    margin-bottom: 0;
  }
}

.output-files-group-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
  text-transform: capitalize;
}

.output-files-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.output-files-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-small);
  min-width: 160px;
  max-width: 280px;
}

.output-files-preview {
  flex-shrink: 0;
  width: 64px;
  height: 64px;
  border-radius: var(--el-border-radius-small);
  overflow: hidden;
  background: var(--el-fill-color);
}

.output-files-preview-link {
  display: block;
  width: 100%;
  height: 100%;
}

.output-files-img {
  width: 100%;
  height: 100%;
  object-fit: cover;

  &[style*='display: none'] {
    visibility: hidden;
  }
}

.output-files-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color);
  border-radius: var(--el-border-radius-small);
  color: var(--el-text-color-secondary);
  font-size: 24px;
}

.output-files-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.output-files-name {
  font-size: 13px;
  color: var(--el-color-primary);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &:hover {
    text-decoration: underline;
  }
}

.output-files-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.output-files-format {
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.output-files-size {
  font-size: 12px;
}

.output-files-actions {
  display: flex;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
}
</style>
