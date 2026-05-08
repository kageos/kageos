<template>
  <div :class="['mini-ws-file-card', { 'mini-ws-file-card--compact': compact }]" @click="$emit('preview', file)">
    <div v-if="isImageFile" class="mini-ws-file-card__thumb">
      <img :src="file.href" :alt="file.name" loading="lazy" />
    </div>
    <div v-else class="mini-ws-file-card__icon">
      <el-icon :size="compact ? 18 : 20"><Document /></el-icon>
      <span v-if="extension" class="mini-ws-file-card__ext">{{ extension }}</span>
    </div>
    <div class="mini-ws-file-card__info">
      <span class="mini-ws-file-card__name" :title="file.name">{{ file.name }}</span>
      <div class="mini-ws-file-card__actions">
        <el-button link size="small" type="primary" @click.stop="$emit('preview', file)">
          <el-icon :size="compact ? 11 : 12"><View /></el-icon> 预览
        </el-button>
        <el-button link size="small" type="primary" @click.stop="$emit('download', file)">
          <el-icon :size="compact ? 11 : 12"><Download /></el-icon> 下载
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Document, Download, View } from '@element-plus/icons-vue'
import type { FilePanelItem } from '../composables/useMiniWorkstationPanel'

const props = defineProps<{
  file: FilePanelItem
  compact?: boolean
}>()

defineEmits<{
  (e: 'preview', file: FilePanelItem): void
  (e: 'download', file: FilePanelItem): void
}>()

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg'])

const extension = computed(() => ((props.file.name || '').match(/\.(\w+)$/)?.[1] || '').toUpperCase())
const isImageFile = computed(() => {
  const ext = (props.file.name || '').toLowerCase().match(/\.\w+$/)?.[0] || ''
  return IMAGE_EXTS.has(ext)
})
</script>

<style scoped>
.mini-ws-file-card {
  display: flex;
  gap: 8px;
  padding: 8px;
  margin-bottom: 6px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 12px;
  background:
    linear-gradient(145deg, rgba(9, 28, 48, 0.62), rgba(4, 12, 24, 0.46)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent);
  transition: transform 0.16s ease, border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease;
  cursor: pointer;
}

.mini-ws-file-card:hover {
  transform: translateY(-1px);
  border-color: rgba(34, 211, 238, 0.46);
  background:
    linear-gradient(145deg, rgba(16, 46, 72, 0.72), rgba(5, 16, 30, 0.52)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.05), transparent);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.18), 0 0 16px rgba(34, 211, 238, 0.08);
}

.mini-ws-file-card--compact {
  padding: 6px 8px;
  margin-bottom: 4px;
}

.mini-ws-file-card__thumb {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid rgba(96, 231, 255, 0.18);
  box-shadow: 0 0 18px rgba(34, 211, 238, 0.08);
}

.mini-ws-file-card__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.mini-ws-file-card--compact .mini-ws-file-card__thumb {
  width: 40px;
  height: 40px;
}

.mini-ws-file-card__icon {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(96, 231, 255, 0.16);
  border-radius: 10px;
  background:
    radial-gradient(circle at 50% 24%, rgba(34, 211, 238, 0.22), transparent 50%),
    rgba(3, 10, 22, 0.56);
  color: var(--mini-cyber-accent, #22d3ee);
}

.mini-ws-file-card--compact .mini-ws-file-card__icon {
  width: 40px;
  height: 40px;
}

.mini-ws-file-card__ext {
  font-size: 9px;
  font-weight: 600;
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
  margin-top: 2px;
  text-transform: uppercase;
}

.mini-ws-file-card__info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.mini-ws-file-card__name {
  font-size: 12px;
  color: var(--mini-cyber-text, #d8f8ff);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.4;
}

.mini-ws-file-card__actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.mini-ws-file-card--compact .mini-ws-file-card__actions {
  margin-top: 2px;
}

.mini-ws-file-card--compact .mini-ws-file-card__actions :deep(.el-button) {
  font-size: 11px;
}
.mini-ws-file-card__actions :deep(.el-button) {
  color: var(--mini-cyber-accent, #22d3ee);
}
.mini-ws-file-card__actions :deep(.el-button:hover) {
  color: #ffffff;
}
</style>
