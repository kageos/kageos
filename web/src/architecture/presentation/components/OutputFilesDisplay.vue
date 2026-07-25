<!--
  OutputFilesDisplay - 输出文件展示组件（通用）
  用于工作台工具结果、或任意返回「含 files 数组」的 JSON 时的文件展示：预览、下载、打开。
  支持传入 result（原始 JSON 字符串/对象）或已解析的 fileGroups。
-->
<template>
  <div v-if="displayGroups.length > 0" class="output-files-display">
    <div class="output-files-head">
      <div class="output-files-head-title">
        <el-icon><FolderOpened /></el-icon>
        <span>{{ sectionTitle }}</span>
      </div>
      <el-button
        v-if="canArchiveDownload"
        class="output-files-archive-btn"
        size="small"
        text
        :icon="Download"
        :loading="archiveDownloading"
        @click="downloadArchive"
      >
        打包下载
      </el-button>
    </div>
    <div class="output-files-wrap">
      <div v-for="(group, gIdx) in displayGroups" :key="gIdx" class="output-files-group">
        <div v-if="displayGroups.length > 1" class="output-files-group-label">{{ group.label }}</div>
        <div class="output-files-list">
          <div
            v-for="(file, fIdx) in visibleGroupFiles(group, gIdx)"
            :key="fIdx"
            class="output-files-item"
            :class="{ 'output-files-item--media': isPreviewableMedia(file) }"
          >
            <div class="output-files-main">
              <div class="output-files-preview" v-if="isImageFile(file) && imagePreviewUrl(file)">
                <a :href="fileDisplayUrl(file)" target="_blank" rel="noopener noreferrer" class="output-files-preview-link">
                  <img
                    :src="imagePreviewUrl(file)"
                    :alt="fileDisplayName(file)"
                    loading="lazy"
                    class="output-files-img"
                    @error="onImageError"
                  />
                </a>
              </div>
              <div class="output-files-preview output-files-video-preview" v-else-if="isVideoFile(file)">
                <video
                  class="output-files-video"
                  controls
                  playsinline
                  preload="metadata"
                  :poster="videoPosterUrl(file) || undefined"
                >
                  <source :src="fileDisplayUrl(file)" :type="fileContentType(file) || undefined" />
                </video>
              </div>
              <div v-else class="output-files-icon">
                <el-icon><Document /></el-icon>
              </div>
              <div class="output-files-info">
                <a :href="fileDisplayUrl(file)" target="_blank" rel="noopener noreferrer" class="output-files-name">
                  {{ fileDisplayName(file) }}
                </a>
              </div>
            </div>
            <div class="output-files-footer">
              <span class="output-files-meta">
                <span v-if="fileFormat(file)" class="output-files-format">{{ fileFormat(file) }}</span>
                <span v-if="file.size != null" class="output-files-size">{{ formatFileSize(file.size) }}</span>
              </span>
              <div class="output-files-actions">
                <el-link type="primary" :href="fileDisplayUrl(file)" target="_blank" rel="noopener noreferrer">打开</el-link>
                <el-link
                  type="primary"
                  :href="fileDisplayUrl(file)"
                  target="_blank"
                  rel="noopener noreferrer"
                  :download="fileDisplayName(file)"
                >下载</el-link>
              </div>
            </div>
          </div>
        </div>
        <button
          v-if="groupCanCollapse(group)"
          type="button"
          class="output-files-collapse-btn"
          @click="toggleGroupExpanded(group, gIdx)"
        >
          {{ groupExpanded(group, gIdx) ? '收起' : `展开全部 ${group.files.length} 个文件` }}
          <span v-if="!groupExpanded(group, gIdx)">（还有 {{ hiddenFileCount(group) }} 个）</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Document, Download, FolderOpened } from '@element-plus/icons-vue'
import { resolveFileRefs, type ResolvedFile } from '@/architecture/presentation/context/api/storage'
import type { ToolResultMetadata } from '@/architecture/presentation/context/api/workspace'
import { extractFileGroupsFromResult, type OutputFileGroup, type OutputFileItem } from '@/architecture/presentation/composables/useOutputFileGroups'
import { normalizeStorageFileDisplayUrl } from '@/architecture/presentation/utils/storageFileUrl'
import { createZipBlob, type ZipEntryInput } from '@/architecture/presentation/utils/downloadZip'

const IMAGE_EXT = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg', '.ico', '.avif'])
const VIDEO_EXT = new Set(['.mp4', '.mov', '.m4v', '.webm', '.ogg', '.ogv', '.avi', '.mkv'])

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
    /** 是否展示多文件打包下载 */
    archiveDownload?: boolean
    /** 打包下载文件名 */
    archiveFileName?: string
    /** 是否在文件很多时折叠列表 */
    collapsible?: boolean
    /** 超过多少个文件时开启折叠 */
    collapseThreshold?: number
    /** 折叠时默认露出的文件数 */
    collapsedVisibleCount?: number
  }>(),
  {
    sectionTitle: '输出文件',
    archiveDownload: true,
    archiveFileName: '',
    collapsible: true,
    collapseThreshold: 4,
    collapsedVisibleCount: 4,
  }
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
      thumbnail_url: file.thumbnail_url ?? '',
      thumbnail_ref: file.thumbnail_ref ?? '',
      preview_kind: file.preview_kind ?? '',
      content_type: file.content_type ?? '',
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
const archiveDownloading = ref(false)
const archiveSourceFiles = computed(() => {
  return displayGroups.value.flatMap((group) =>
    group.files.map((file, fileIndex) => ({ file, group, fileIndex }))
  )
})
const canArchiveDownload = computed(() => props.archiveDownload && archiveSourceFiles.value.length > 1)
const expandedGroupKeys = ref<Set<string>>(new Set())

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
        thumbnail_ref: item.thumbnail_ref || file.thumbnail_ref,
        thumbnail_url: item.thumbnail_url || file.thumbnail_url,
        preview_kind: item.preview_kind || file.preview_kind,
        content_type: item.content_type || file.content_type,
      }
    })
  }))
}

function isImageFile(file: OutputFileItem): boolean {
  const contentType = fileContentType(file)
  if (contentType.startsWith('image/')) return true
  return IMAGE_EXT.has(fileExtension(file))
}

function isVideoFile(file: OutputFileItem): boolean {
  const previewKind = String(file.preview_kind || '').toLowerCase()
  if (previewKind === 'video') return true
  const contentType = fileContentType(file)
  if (contentType.startsWith('video/')) return true
  return VIDEO_EXT.has(fileExtension(file))
}

function isPreviewableMedia(file: OutputFileItem): boolean {
  return isImageFile(file) || isVideoFile(file)
}

function imagePreviewUrl(file: OutputFileItem): string {
  const thumbnailUrl = normalizeStorageFileDisplayUrl(String(file.thumbnail_url || ''))
  if (thumbnailUrl) return thumbnailUrl
  return isImageFile(file) ? fileDisplayUrl(file) : ''
}

function videoPosterUrl(file: OutputFileItem): string {
  return normalizeStorageFileDisplayUrl(String(file.thumbnail_url || ''))
}

function fileContentType(file: OutputFileItem): string {
  return String(file.content_type || '').toLowerCase()
}

function fileExtension(file: OutputFileItem): string {
  const name = (file.source_name ?? file.name ?? '') as string
  if (!name || !name.includes('.')) return ''
  return name.slice(name.lastIndexOf('.')).toLowerCase()
}

function fileDisplayName(file: OutputFileItem): string {
  return (file.source_name ?? file.name ?? '文件') as string
}

function fileDisplayUrl(file: OutputFileItem): string {
  return normalizeStorageFileDisplayUrl(file.download_url || file.ref || '')
}

function groupKey(group: OutputFileGroup, index: number): string {
  const refs = group.files.map(file => file.ref || file.download_url || file.name || '').join('|')
  return `${index}:${group.label}:${refs}`
}

function groupExpanded(group: OutputFileGroup, index: number): boolean {
  return expandedGroupKeys.value.has(groupKey(group, index))
}

function groupCanCollapse(group: OutputFileGroup): boolean {
  if (!props.collapsible) return false
  return group.files.length > normalizedCollapseThreshold()
}

function visibleGroupFiles(group: OutputFileGroup, index: number): OutputFileItem[] {
  if (!groupCanCollapse(group) || groupExpanded(group, index)) return group.files
  return group.files.slice(0, normalizedCollapsedVisibleCount())
}

function hiddenFileCount(group: OutputFileGroup): number {
  return Math.max(0, group.files.length - normalizedCollapsedVisibleCount())
}

function toggleGroupExpanded(group: OutputFileGroup, index: number): void {
  const key = groupKey(group, index)
  const next = new Set(expandedGroupKeys.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  expandedGroupKeys.value = next
}

function normalizedCollapseThreshold(): number {
  return Math.max(1, props.collapseThreshold)
}

function normalizedCollapsedVisibleCount(): number {
  return Math.max(1, Math.min(props.collapsedVisibleCount, normalizedCollapseThreshold()))
}

async function downloadArchive(): Promise<void> {
  if (archiveDownloading.value) return

  const sources = archiveSourceFiles.value
  if (sources.length <= 1) return

  archiveDownloading.value = true
  try {
    const usedNames = new Map<string, number>()
    const entries: ZipEntryInput[] = []
    for (const source of sources) {
      const url = fileDisplayUrl(source.file)
      const displayName = fileDisplayName(source.file) || `file-${source.fileIndex + 1}`
      if (!url) {
        throw new Error(`文件「${displayName}」缺少下载地址`)
      }

      const response = await fetch(url)
      if (!response.ok) {
        throw new Error(`文件「${displayName}」下载失败：HTTP ${response.status}`)
      }

      const folder = displayGroups.value.length > 1 ? sanitizeArchiveSegment(source.group.label) : ''
      const entryName = uniqueArchiveEntryName(
        folder ? `${folder}/${sanitizeArchiveSegment(displayName)}` : sanitizeArchiveSegment(displayName),
        usedNames
      )
      entries.push({
        name: entryName,
        data: await response.blob(),
      })
    }

    const zipBlob = await createZipBlob(entries)
    triggerBlobDownload(zipBlob, archiveDownloadName())
    ElMessage.success(`已打包 ${entries.length} 个文件`)
  } catch (error) {
    const message = error instanceof Error ? error.message : '打包下载失败'
    ElMessage.error(message)
  } finally {
    archiveDownloading.value = false
  }
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

function archiveDownloadName(): string {
  const rawName = props.archiveFileName.trim() || `output-files-${formatTimestamp(new Date())}.zip`
  const safeName = rawName.replace(/[\\/:*?"<>|\x00-\x1f]/g, '_').trim() || 'output-files.zip'
  return safeName.toLowerCase().endsWith('.zip') ? safeName : `${safeName}.zip`
}

function formatTimestamp(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0')
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
    '-',
    pad(date.getHours()),
    pad(date.getMinutes()),
    pad(date.getSeconds()),
  ].join('')
}

function sanitizeArchiveSegment(value: string): string {
  return String(value || 'file')
    .replace(/[\\/:*?"<>|\x00-\x1f]/g, '_')
    .replace(/^\.+$/, '_')
    .trim() || 'file'
}

function uniqueArchiveEntryName(name: string, usedNames: Map<string, number>): string {
  const normalized = name || 'file'
  const used = usedNames.get(normalized) || 0
  usedNames.set(normalized, used + 1)
  if (used === 0) {
    return normalized
  }

  const slashIndex = normalized.lastIndexOf('/')
  const dir = slashIndex >= 0 ? normalized.slice(0, slashIndex + 1) : ''
  const base = slashIndex >= 0 ? normalized.slice(slashIndex + 1) : normalized
  const dotIndex = base.lastIndexOf('.')
  if (dotIndex > 0) {
    return `${dir}${base.slice(0, dotIndex)} (${used + 1})${base.slice(dotIndex)}`
  }
  return `${dir}${base} (${used + 1})`
}

function triggerBlobDownload(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
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
  justify-content: space-between;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;

  .el-icon {
    font-size: 14px;
  }
}

.output-files-head-title {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.output-files-archive-btn {
  flex-shrink: 0;
  height: 24px;
  padding: 0 6px;
  font-size: 12px;
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
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  gap: 12px;
}

.output-files-item {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  min-height: 112px;
  padding: 10px 12px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-small);
}

.output-files-item--media {
  grid-column: 1 / -1;
  justify-self: center;
  width: 100%;
  min-height: 0;
  padding: 8px;

  .output-files-main {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .output-files-info {
    width: min(100%, 1520px);
    margin-top: 8px;
  }

  .output-files-preview {
    width: min(100%, 1520px);
    height: auto;
    aspect-ratio: 16 / 9;
    background: #0b0f16;
  }
}

.output-files-main {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  align-items: start;
  gap: 10px;
  min-width: 0;
}

.output-files-preview {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
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
  object-fit: contain;

  &[style*='display: none'] {
    visibility: hidden;
  }
}

.output-files-video-preview {
  border: 1px solid var(--el-border-color-lighter);
}

.output-files-video {
  width: 100%;
  height: 100%;
  display: block;
  background: #0b0f16;
}

.output-files-icon {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
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
  line-height: 1.45;
  color: var(--el-color-primary);
  text-decoration: none;
  overflow: hidden;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;

  &:hover {
    text-decoration: underline;
  }
}

.output-files-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-extra-light);
}

.output-files-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
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
  flex: 0 0 auto;
  justify-content: flex-end;
  gap: 10px;
  font-size: 12px;
}

.output-files-collapse-btn {
  width: 100%;
  min-height: 32px;
  margin-top: 10px;
  border: 1px dashed var(--el-border-color);
  border-radius: var(--el-border-radius-small);
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
  font-size: 12px;
  cursor: pointer;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
  }
}

@media (max-width: 560px) {
  .output-files-wrap {
    padding: 8px;
  }

  .output-files-list {
    gap: 8px;
  }

  .output-files-item {
    min-height: 0;
    padding: 9px 10px;
  }

  .output-files-main {
    grid-template-columns: 44px minmax(0, 1fr);
  }

  .output-files-preview,
  .output-files-icon {
    width: 44px;
    height: 44px;
  }

  .output-files-item--media .output-files-preview {
    width: 100%;
    height: auto;
  }

  .output-files-item--media .output-files-info {
    width: 100%;
  }

  .output-files-footer {
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
  }
}
</style>
