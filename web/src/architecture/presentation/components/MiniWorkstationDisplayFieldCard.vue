<template>
  <div :class="['mini-ws-display-field-card', { 'mini-ws-display-field-card--compact': compact }]">
    <div :class="compact ? 'mini-ws-display-field-card__header' : 'mini-ws-display-field-card__sidebar-header'">
      <span :class="compact ? 'mini-ws-display-field-card__label' : 'mini-ws-display-field-card__sidebar-label'">{{ field.label }}</span>
      <div :class="compact ? 'mini-ws-display-field-card__actions' : 'mini-ws-display-field-card__sidebar-actions'">
        <el-button link size="small" type="primary" @click.stop="$emit('preview', field)">
          <el-icon :size="compact ? 11 : 12"><View /></el-icon> 预览
        </el-button>
        <el-button link size="small" type="primary" @click.stop="$emit('copy', field)">
          <el-icon :size="compact ? 11 : 12"><CopyDocument /></el-icon> 复制
        </el-button>
      </div>
    </div>
    <div v-if="compact" class="mini-ws-display-field-card__value">
      {{ previewValue }}
    </div>
    <div v-else class="mini-ws-display-field-card__sidebar-value">
      <pre class="mini-ws-display-field-card__pre">{{ field.value }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CopyDocument, View } from '@element-plus/icons-vue'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'

const props = defineProps<{
  field: OutputDisplayField
  compact?: boolean
}>()

defineEmits<{
  (e: 'preview', field: OutputDisplayField): void
  (e: 'copy', field: OutputDisplayField): void
}>()

const previewValue = computed(() => (
  props.field.value.length > 150 ? `${props.field.value.slice(0, 150)}…` : props.field.value
))
</script>

<style scoped>
.mini-ws-display-field-card {
  margin-bottom: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  overflow: hidden;
}

.mini-ws-display-field-card--compact {
  padding: 6px 8px;
  margin-bottom: 4px;
  border-color: var(--el-border-color-extra-light);
  border-radius: var(--el-border-radius-small);
  background: var(--el-bg-color);
}

.mini-ws-display-field-card__header,
.mini-ws-display-field-card__sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mini-ws-display-field-card__header {
  margin-bottom: 4px;
}

.mini-ws-display-field-card__sidebar-header {
  padding: 6px 10px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-extra-light);
}

.mini-ws-display-field-card__label,
.mini-ws-display-field-card__sidebar-label {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.mini-ws-display-field-card__label {
  font-size: 11px;
}

.mini-ws-display-field-card__sidebar-label {
  font-size: 12px;
}

.mini-ws-display-field-card__actions,
.mini-ws-display-field-card__sidebar-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.mini-ws-display-field-card__value {
  font-size: 11px;
  line-height: 1.5;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 4.5em;
  overflow: hidden;
}

.mini-ws-display-field-card__sidebar-value {
  padding: 6px 10px;
  max-height: 200px;
  overflow-y: auto;
}

.mini-ws-display-field-card__pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 11px;
  line-height: 1.5;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  color: var(--el-text-color-regular);
}
</style>
