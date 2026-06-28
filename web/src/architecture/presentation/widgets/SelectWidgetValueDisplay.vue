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

  // 渲染空心标签时：去掉背景，并将文字和边框均设置为更清晰的颜色（例如使用原始颜色或其加深版本作为边框和文字色）
  return {
    borderColor: color || lightPalette.borderColor,
    color: color || lightPalette.color
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
  background-color: var(--el-fill-color-light) !important;
  border: 1px solid var(--border-base) !important;
  color: var(--text-secondary) !important;
  opacity: 0.95;
  transition: all 0.2s;
}

/* 默认标签：给一个极简浅灰底，避免“透明白”的鬼影 */
.select-tag.el-tag--info {
  background-color: var(--el-fill-color-light) !important;
  border-color: var(--border-base) !important;
  color: var(--text-secondary) !important;
}

/* 如果有具体颜色覆盖，隐藏背景并保留具体的边框和颜色 */
.select-tag[style*="border-color:"],
.select-tag[style*="color:"] {
  background-color: transparent !important;
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
