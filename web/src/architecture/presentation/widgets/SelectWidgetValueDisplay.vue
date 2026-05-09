<template>
  <span v-if="mode === 'response'" class="response-value">
    <el-tag
      v-if="currentOptionColor"
      :style="getTagStyle(currentOptionColor)"
      size="small"
      effect="light"
      class="select-tag"
    >
      {{ displayValue }}
    </el-tag>
    <span v-else>{{ displayValue }}</span>
  </span>

  <div v-else-if="mode === 'table-cell'" class="table-cell-value">
    <el-tag
      v-if="currentOptionColor"
      :style="getTagStyle(currentOptionColor)"
      size="small"
      effect="light"
      class="select-tag"
    >
      {{ displayValue }}
    </el-tag>
    <span v-else>{{ displayValue }}</span>
  </div>

  <div v-else class="detail-value">
    <el-tag
      v-if="currentOptionColor"
      :style="getTagStyle(currentOptionColor)"
      effect="light"
      class="select-tag"
    >
      {{ displayValue }}
    </el-tag>
    <span v-else class="detail-content">{{ displayValue }}</span>
  </div>
</template>

<script setup lang="ts">
import { ElTag } from 'element-plus'
import { getOptionLightPalette } from '@/core/constants/select'

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
  border: 1px solid currentColor;
  box-shadow: none;
  opacity: 0.9;
  transition: opacity 0.2s;
}

.select-tag:hover {
  opacity: 1;
}

.response-value :deep(.el-tag),
.table-cell-value :deep(.el-tag),
.detail-value :deep(.el-tag) {
  font-weight: 500;
  box-shadow: none;
}

.detail-content {
  color: var(--el-text-color-regular);
}
</style>
