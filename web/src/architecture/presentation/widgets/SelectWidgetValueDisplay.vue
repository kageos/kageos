<template>
  <span v-if="mode === 'response'" class="response-value">
    {{ displayValue }}
  </span>

  <div v-else-if="mode === 'table-cell'" class="table-cell-value">
    <el-tag
      v-if="currentOptionColor"
      :type="getTagType(currentOptionColor)"
      :style="getTagStyle(currentOptionColor)"
      size="small"
      class="select-tag select-tag-outline"
    >
      {{ displayValue }}
    </el-tag>
    <span v-else>{{ displayValue }}</span>
  </div>

  <div v-else class="detail-value">
    <el-tag
      v-if="currentOptionColor"
      :type="getTagType(currentOptionColor)"
      :style="getTagStyle(currentOptionColor)"
      class="select-tag select-tag-outline"
    >
      {{ displayValue }}
    </el-tag>
    <span v-else class="detail-content">{{ displayValue }}</span>
  </div>
</template>

<script setup lang="ts">
import { ElTag } from 'element-plus'
import { getOptionSolidColor, normalizeOptionColor, type StandardColorType } from '@/core/constants/select'

defineProps<{
  mode: 'response' | 'table-cell' | 'detail'
  displayValue: string
  currentOptionColor: string | null
}>()

function getTagType(color: string | null): StandardColorType | undefined {
  void color
  return undefined
}

function getTagStyle(color: string | null): Record<string, string> {
  const normalizedColor = normalizeOptionColor(color)
  if (!normalizedColor) {
    return {}
  }

  const solidColor = getOptionSolidColor(normalizedColor)
  return {
    color: solidColor,
    borderColor: solidColor
  }
}
</script>

<style scoped lang="scss">
.response-value {
  color: var(--el-text-color-regular);
}

.table-cell-value {
  display: inline-flex;
  align-items: center;
}

.detail-value {
  margin-bottom: 16px;
  display: inline-flex;
  align-items: center;
}

.select-tag {
  font-weight: 500;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  opacity: 0.9;
  transition: opacity 0.2s;
}

.select-tag:hover {
  opacity: 1;
}

.select-tag-outline {
  background-color: transparent !important;
  border: 2px solid currentColor !important;
}

.select-tag-outline.el-tag--success {
  color: var(--el-color-success) !important;
  border-color: var(--el-color-success) !important;
}

.select-tag-outline.el-tag--warning {
  color: var(--el-color-warning) !important;
  border-color: var(--el-color-warning) !important;
}

.select-tag-outline.el-tag--danger {
  color: var(--el-color-danger) !important;
  border-color: var(--el-color-danger) !important;
}

.select-tag-outline.el-tag--info {
  color: var(--el-color-info) !important;
  border-color: var(--el-color-info) !important;
}

.select-tag-outline.el-tag--primary {
  color: var(--el-color-primary) !important;
  border-color: var(--el-color-primary) !important;
}

.select-tag-outline[style*="color"] {
  border-color: currentColor !important;
}

.table-cell-value :deep(.el-tag),
.detail-value :deep(.el-tag) {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.table-cell-value :deep(.el-tag[style*="background-color"]),
.detail-value :deep(.el-tag[style*="background-color"]) {
  color: #fff !important;
  font-weight: 500;
}

.detail-content {
  color: var(--el-text-color-regular);
}
</style>
