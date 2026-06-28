<template>
  <div :class="containerClass">
    <el-tag
      v-for="(value, index) in displayValues"
      :key="index"
      class="tag-item"
      :size="mode === 'table-cell' ? 'small' : undefined"
      :type="getOptionColorType(value) ? getOptionColorType(value) : 'primary'"
      :color="getOptionColorValue(value)"
      :style="getOptionTagStyle(value)"
      effect="light"
    >
      {{ getOptionLabel(value) }}
    </el-tag>
    <span v-if="displayValues.length === 0" class="empty-text">-</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElTag } from 'element-plus'
import type { StandardColorType } from '@/architecture/domain/constants/select'

const props = defineProps<{
  mode: 'response' | 'table-cell' | 'detail'
  displayValues: string[]
  getOptionLabel: (value: any) => string
  getOptionColorType: (value: any) => StandardColorType | undefined
  getOptionColorValue: (value: any) => string | undefined
  getOptionTagStyle: (value: any) => Record<string, string>
}>()

const containerClass = computed(() => {
  if (props.mode === 'response') return 'response-multiselect'
  if (props.mode === 'table-cell') return 'table-cell-multiselect'
  return 'detail-multiselect'
})
</script>

<style scoped lang="scss">
.response-multiselect,
.table-cell-multiselect,
.detail-multiselect {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.table-cell-multiselect .tag-item,
.detail-multiselect .tag-item,
.response-multiselect .tag-item {
  font-weight: 500;
  border-radius: 4px;
  margin: 0;
  border: 1px solid var(--border-base);
}

.table-cell-multiselect .tag-item.el-tag--primary,
.detail-multiselect .tag-item.el-tag--primary,
.response-multiselect .tag-item.el-tag--primary {
  border-color: rgba(var(--color-primary-rgb), 0.3) !important;
}

/* 恢复带背景色的浅色色板渲染，避免空心透明感 */
.table-cell-multiselect .tag-item[style*="border-color:"],
.detail-multiselect .tag-item[style*="border-color:"],
.response-multiselect .tag-item[style*="border-color:"],
.table-cell-multiselect .tag-item[style*="color:"],
.detail-multiselect .tag-item[style*="color:"],
.response-multiselect .tag-item[style*="color:"] {
  box-shadow: none !important;
  filter: none;
}


.empty-text {
  color: var(--text-disabled);
  font-style: italic;
}
</style>
