<template>
  <div :class="containerClass">
    <el-tag
      v-for="(value, index) in displayValues"
      :key="index"
      class="tag-item"
      :size="mode === 'table-cell' ? 'small' : undefined"
      :type="getOptionColorType(value)"
      :color="getOptionColorValue(value)"
      :style="getOptionTagStyle(value)"
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
.detail-multiselect .tag-item {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  margin: 0;
  opacity: 0.9;
}

.table-cell-multiselect .tag-item[style*="background-color"],
.detail-multiselect .tag-item[style*="background-color"] {
  font-weight: 500;
  filter: brightness(0.95) saturate(0.9);
}

.table-cell-multiselect .tag-item.el-tag--success,
.table-cell-multiselect .tag-item.el-tag--warning,
.table-cell-multiselect .tag-item.el-tag--danger,
.table-cell-multiselect .tag-item.el-tag--info,
.table-cell-multiselect .tag-item.el-tag--primary,
.detail-multiselect .tag-item.el-tag--success,
.detail-multiselect .tag-item.el-tag--warning,
.detail-multiselect .tag-item.el-tag--danger,
.detail-multiselect .tag-item.el-tag--info,
.detail-multiselect .tag-item.el-tag--primary {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  opacity: 0.9;
}

.response-multiselect .tag-item {
  margin-right: 4px;
}

.empty-text {
  color: #999;
}
</style>
