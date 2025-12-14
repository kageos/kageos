<template>
  <div v-if="displaySorts.length > 0" class="sort-info-bar">
    <div class="sort-info-content">
      <span class="sort-label">排序：</span>
      <div class="sort-items">
        <!-- 显示所有排序列 -->
        <template v-for="(sort, index) in displaySorts" :key="sort.field">
          <el-tag
            :type="index === 0 ? 'primary' : 'info'"
            size="small"
            closable
            @close="handleRemoveSort(sort.field)"
            class="sort-tag"
          >
            <span class="sort-field-name">{{ getFieldName(sort.field) }}</span>
            <el-icon class="sort-icon">
              <ArrowUp v-if="sort.order === 'asc'" />
              <ArrowDown v-else />
            </el-icon>
          </el-tag>
          <span v-if="index < displaySorts.length - 1" class="sort-separator">></span>
        </template>
      </div>
      <el-button
        v-if="sorts.length > 0"
        link
        type="primary"
        size="small"
        @click="handleClearAllSorts"
        class="clear-all-sorts-btn"
      >
        清除所有排序
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowUp, ArrowDown } from '@element-plus/icons-vue'
import { ElIcon, ElButton, ElTag } from 'element-plus'
import type { FieldConfig } from '@/core/types/field'

interface SortItem {
  field: string
  order: 'asc' | 'desc'
}

interface Props {
  /** 排序列表 */
  sorts: SortItem[]
  /** 显示用的排序列表（可能包含默认排序） */
  displaySorts: SortItem[]
  /** 可见字段列表（用于获取字段名称） */
  visibleFields: FieldConfig[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'remove-sort', fieldCode: string): void
  (e: 'clear-all-sorts'): void
}>()

/**
 * 获取字段名称
 * @param fieldCode 字段代码
 * @returns 字段名称
 */
const getFieldName = (fieldCode: string): string => {
  const field = props.visibleFields.find((f: FieldConfig) => f.code === fieldCode)
  return field?.name || fieldCode
}

/**
 * 处理移除单个排序
 */
const handleRemoveSort = (fieldCode: string): void => {
  emit('remove-sort', fieldCode)
}

/**
 * 处理清除所有排序
 */
const handleClearAllSorts = (): void => {
  emit('clear-all-sorts')
}
</script>

<style scoped>
/* 🔥 排序信息条样式 */
.sort-info-bar {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  display: flex;
  align-items: center;
}

.sort-info-content {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  flex-wrap: wrap;
}

.sort-label {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
  white-space: nowrap;
}

.sort-items {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  flex: 1;
}

.sort-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: default;
}

.sort-field-name {
  font-weight: 500;
}

.sort-icon {
  font-size: 12px;
  margin-left: 2px;
}

.sort-separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  font-weight: 500;
  margin: 0 4px;
}

.clear-all-sorts-btn {
  margin-left: auto;
  white-space: nowrap;
}
</style>

