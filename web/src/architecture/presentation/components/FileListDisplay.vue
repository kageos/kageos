<!--
  FileListDisplay - 通用文件列表展示组件
  用于工作台消息附件、表单响应文件等场景，支持 tag（紧凑标签）与 list（列表带图标/大小/链接）两种展示模式。
  文件数据来自 storage resolve 后的展示元信息。
-->
<template>
  <div :class="['file-list-display', `file-list-display--${mode}`]">
    <!-- tag 模式：紧凑标签，可点击打开 -->
    <template v-if="mode === 'tag'">
      <el-tag
        v-for="(f, idx) in displayFiles"
        :key="keyFor(f, idx)"
        size="small"
        class="file-tag"
        :type="tagType"
        @click="openFile(f)"
      >
        <el-icon class="file-tag-icon"><component :is="fileIcon(f)" /></el-icon>
        <span class="file-tag-name">{{ displayName(f) }}</span>
      </el-tag>
      <span v-if="overflowCount > 0" class="file-overflow">等 {{ overflowCount }} 个文件</span>
    </template>

    <!-- list 模式：列表行，图标 + 名称 + 大小 + 链接 -->
    <ul v-else class="file-list">
      <li v-for="(f, idx) in displayFiles" :key="keyFor(f, idx)" class="file-list-item">
        <div class="file-list-main">
          <el-icon class="file-list-icon"><component :is="fileIcon(f)" /></el-icon>
          <span class="file-list-name" :title="displayName(f)">{{ displayName(f) }}</span>
        </div>
        <div class="file-list-footer">
          <span v-if="showSize && f.size != null" class="file-list-size">{{ formatSize(f.size) }}</span>
          <el-link
            v-if="f.url && showLink"
            type="primary"
            :href="f.url"
            target="_blank"
            rel="noopener"
            class="file-list-link"
          >
            打开
          </el-link>
        </div>
      </li>
      <li v-if="overflowCount > 0" class="file-list-overflow">等 {{ overflowCount }} 个文件</li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Document, Picture, VideoPlay } from '@element-plus/icons-vue'

/** storage resolve 后的单条文件展示信息 */
export interface DisplayFileItem {
  name: string
  source_name?: string
  url?: string
  size?: number
  [key: string]: unknown
}

const props = withDefaults(
  defineProps<{
    /** 文件列表 */
    files: DisplayFileItem[]
    /** 展示模式：tag 紧凑标签，list 列表行 */
    mode?: 'tag' | 'list'
    /** 最多展示条数，超出显示「等 N 个文件」 */
    maxDisplay?: number
    /** 是否显示文件大小（list 模式有效） */
    showSize?: boolean
    /** 是否显示打开链接（list 模式有效） */
    showLink?: boolean
    /** tag 模式下的 el-tag type */
    tagType?: '' | 'success' | 'info' | 'warning' | 'danger'
  }>(),
  {
    mode: 'tag',
    maxDisplay: 20,
    showSize: true,
    showLink: true,
    tagType: 'info',
  }
)

const displayFiles = computed(() => {
  const list = props.files || []
  const max = props.maxDisplay ?? 20
  return list.slice(0, max)
})

const overflowCount = computed(() => {
  const list = props.files || []
  const max = props.maxDisplay ?? 20
  return list.length > max ? list.length - max : 0
})

function keyFor(f: DisplayFileItem, idx: number): string {
  return (f.hash as string) || (f.name + idx) || String(idx)
}

function displayName(f: DisplayFileItem): string {
  return f.name || (f.source_name as string) || '文件'
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function fileIcon(f: DisplayFileItem): typeof Document {
  const name = (f.name || f.source_name || '').toLowerCase()
  if (/\.(png|jpg|jpeg|gif|webp|bmp|svg)$/i.test(name)) return Picture
  if (/\.(mp4|webm|mov|avi|mkv)$/i.test(name)) return VideoPlay
  if (/\.(pdf)$/i.test(name)) return Document
  return Document
}

function openFile(f: DisplayFileItem): void {
  if (f.url) window.open(f.url, '_blank', 'noopener')
}
</script>

<style scoped>
.file-list-display {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.file-list-display--list {
  flex-direction: column;
  align-items: stretch;
  gap: 0;
}

/* tag 模式 */
.file-tag {
  cursor: pointer;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.file-tag-icon {
  flex-shrink: 0;
  font-size: 14px;
}
.file-tag-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-overflow {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* list 模式 */
.file-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.file-list-item {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 5px;
  padding: 6px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 13px;
  min-width: 0;
}
.file-list-item:last-child {
  border-bottom: none;
}
.file-list-main,
.file-list-footer {
  display: flex;
  align-items: center;
  min-width: 0;
}
.file-list-main {
  gap: 8px;
}
.file-list-footer {
  justify-content: space-between;
  gap: 10px;
  padding-left: 26px;
}
.file-list-icon {
  flex-shrink: 0;
  font-size: 18px;
  color: var(--el-text-color-secondary);
}
.file-list-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.file-list-size {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.file-list-link {
  flex-shrink: 0;
}
.file-list-overflow {
  padding: 4px 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
