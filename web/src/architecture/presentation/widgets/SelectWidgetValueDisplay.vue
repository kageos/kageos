<template>
  <span v-if="mode === 'response'" class="response-value">
    <el-tag
      v-if="currentOptionColor"
      :style="getTagStyle(currentOptionColor)"
      size="small"
      effect="plain"
      class="select-tag"
    >
      {{ displayValue }}
    </el-tag>
    <el-tag v-else size="small" type="info" effect="plain" class="select-tag">{{ displayValue }}</el-tag>
  </span>

  <div v-else-if="mode === 'table-cell'" class="table-cell-value">
    <el-tag
      v-if="currentOptionColor"
      :style="getTagStyle(currentOptionColor)"
      size="small"
      effect="plain"
      class="select-tag"
    >
      {{ displayValue }}
    </el-tag>
    <el-tag v-else size="small" type="info" effect="plain" class="select-tag">{{ displayValue }}</el-tag>
  </div>

  <div v-else class="detail-value">
    <el-tag
      v-if="currentOptionColor"
      :style="getTagStyle(currentOptionColor)"
      effect="plain"
      class="select-tag"
    >
      {{ displayValue }}
    </el-tag>
    <el-tag v-else type="info" effect="plain" class="select-tag">{{ displayValue }}</el-tag>
  </div>
</template>

<script setup lang="ts">
import { ElTag } from 'element-plus'
import { getOptionLightPalette } from '@/architecture/domain/constants/select'

defineProps<{
  mode: 'response' | 'table-cell' | 'detail'
  displayValue: string
  currentOptionColor: string | null
}>()

function getTagStyle(color: string | null): Record<string, string> {
  const lightPalette = getOptionLightPalette(color)
  if (!lightPalette) {
    return {}
  }

  return {
    backgroundColor: lightPalette.backgroundColor,
    borderColor: lightPalette.borderColor,
    color: lightPalette.color
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
  border-radius: 4px;
  background-color: transparent !important;
  border: 1px solid var(--border-base);
  opacity: 0.95;
  transition: all 0.2s;
}

.select-tag.el-tag--info {
  border-color: var(--border-base) !important;
  color: var(--text-regular) !important;
}

.select-tag[style*="color:"] {
  border-color: currentColor !important;
}

.select-tag[style*="background-color"] {
  background-color: transparent !important;
  box-shadow: none !important;
  filter: none;
}

.select-tag:hover {
  opacity: 1;
}

.response-value :deep(.el-tag),
.table-cell-value :deep(.el-tag),
.detail-value :deep(.el-tag) {
  font-weight: 500;
  background-color: transparent !important;
  box-shadow: none;
}

.detail-content {
  color: var(--el-text-color-regular);
}
</style>
