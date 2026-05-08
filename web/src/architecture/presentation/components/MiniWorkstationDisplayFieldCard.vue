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
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 12px;
  background:
    linear-gradient(145deg, rgba(9, 28, 48, 0.62), rgba(4, 12, 24, 0.46)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent);
  overflow: hidden;
}

.mini-ws-display-field-card--compact {
  padding: 6px 8px;
  margin-bottom: 4px;
  border-color: rgba(96, 231, 255, 0.14);
  border-radius: 12px;
  background:
    linear-gradient(145deg, rgba(9, 28, 48, 0.62), rgba(4, 12, 24, 0.46)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent);
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
  background: rgba(34, 211, 238, 0.08);
  border-bottom: 1px solid rgba(96, 231, 255, 0.12);
}

.mini-ws-display-field-card__label,
.mini-ws-display-field-card__sidebar-label {
  font-weight: 800;
  color: var(--mini-cyber-text, #d8f8ff);
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
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 4.5em;
  overflow: hidden;
}

.mini-ws-display-field-card__sidebar-value {
  padding: 6px 10px;
  max-height: 200px;
  overflow-y: auto;
  background: rgba(2, 8, 18, 0.24);
}

.mini-ws-display-field-card__pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 11px;
  line-height: 1.5;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
}
.mini-ws-display-field-card__actions :deep(.el-button),
.mini-ws-display-field-card__sidebar-actions :deep(.el-button) {
  color: var(--mini-cyber-accent, #22d3ee);
}
.mini-ws-display-field-card__actions :deep(.el-button:hover),
.mini-ws-display-field-card__sidebar-actions :deep(.el-button:hover) {
  color: #ffffff;
}
</style>
