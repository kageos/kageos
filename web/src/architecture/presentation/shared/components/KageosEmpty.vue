<template>
  <div class="kageos-empty" :class="[size, { 'is-bordered': border }]">
    <div class="empty-icon-wrapper">
      <slot name="image">
        <el-icon class="empty-icon"><component :is="iconComponent" /></el-icon>
      </slot>
    </div>
    <div class="empty-description">
      <slot name="description">
        <p>{{ description || defaultDescription }}</p>
      </slot>
    </div>
    <div v-if="$slots.default" class="empty-bottom">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Document, Search, Files, Box, Warning } from '@element-plus/icons-vue'

const props = withDefaults(defineProps<{
  description?: string
  image?: string
  icon?: 'document' | 'search' | 'files' | 'box' | 'warning'
  size?: 'small' | 'default' | 'large'
  border?: boolean
}>(), {
  icon: 'box',
  size: 'default',
  border: false
})

const defaultDescription = computed(() => {
  switch (props.icon) {
    case 'search': return '暂无搜索结果'
    case 'document': return '暂无文档'
    case 'files': return '暂无文件'
    case 'warning': return '暂无权限或状态异常'
    default: return '暂无数据'
  }
})

const iconComponent = computed(() => {
  switch (props.icon) {
    case 'search': return Search
    case 'document': return Document
    case 'files': return Files
    case 'warning': return Warning
    case 'box':
    default: return Box
  }
})
</script>

<style scoped>
.kageos-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  box-sizing: border-box;
  width: 100%;
}

.kageos-empty.is-bordered {
  border: 1px dashed var(--el-border-color-light);
  border-radius: var(--el-border-radius-base);
  background-color: var(--el-fill-color-lighter);
}

.kageos-empty.small {
  padding: 16px;
}

.kageos-empty.large {
  padding: 64px 24px;
}

.empty-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  color: var(--el-text-color-placeholder);
}

.kageos-empty.small .empty-icon-wrapper {
  margin-bottom: 8px;
}

.empty-icon {
  font-size: 48px;
  opacity: 0.8;
}

.kageos-empty.small .empty-icon {
  font-size: 24px;
}

.kageos-empty.large .empty-icon {
  font-size: 72px;
}

.empty-description {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  text-align: center;
}

.kageos-empty.small .empty-description {
  font-size: 13px;
}

.empty-bottom {
  margin-top: 16px;
}
</style>
