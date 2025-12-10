<template>
  <el-card shadow="hover" class="relation-card" :class="{ 'relation-card--clickable': clickable }" @click="handleClick">
    <div class="relation-card__header">
      <div class="relation-card__title">
        <el-icon class="relation-card__icon" :style="{ color: iconColor }">
          <component :is="icon" />
        </el-icon>
        <span>{{ title }}</span>
      </div>
      <el-tag v-if="tag" :type="tagType" size="small">{{ tag }}</el-tag>
    </div>
    <div v-if="description" class="relation-card__description">{{ description }}</div>
    <div v-if="items && items.length > 0" class="relation-card__items">
      <div v-for="(item, index) in items" :key="index" class="relation-card__item">
        <el-icon class="relation-card__item-icon">
          <component :is="itemIcon" />
        </el-icon>
        <span class="relation-card__item-text">{{ item }}</span>
      </div>
    </div>
    <div v-else-if="emptyText" class="relation-card__empty">{{ emptyText }}</div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  icon: any // 图标组件
  title: string // 标题
  description?: string // 描述
  items?: string[] // 关联项列表
  itemIcon?: any // 关联项图标
  tag?: string // 标签
  tagType?: 'success' | 'warning' | 'danger' | 'info' // 标签类型
  iconColor?: string // 图标颜色
  emptyText?: string // 空状态文本
  clickable?: boolean // 是否可点击
}

const props = withDefaults(defineProps<Props>(), {
  iconColor: 'var(--el-color-primary)',
  tagType: 'info',
  clickable: false,
  itemIcon: 'CircleCheck'
})

const emit = defineEmits<{
  click: []
}>()

function handleClick() {
  if (props.clickable) {
    emit('click')
  }
}
</script>

<style scoped lang="scss">
.relation-card {
  transition: all 0.3s;
  border: 1px solid var(--el-border-color-lighter);
  
  &--clickable {
    cursor: pointer;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      border-color: var(--el-color-primary);
    }
  }
  
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }
  
  &__title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  
  &__icon {
    font-size: 18px;
  }
  
  &__description {
    font-size: 14px;
    color: var(--el-text-color-regular);
    line-height: 1.6;
    margin-bottom: 12px;
  }
  
  &__items {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  &__item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: var(--el-text-color-regular);
  }
  
  &__item-icon {
    font-size: 14px;
    color: var(--el-color-success);
  }
  
  &__item-text {
    flex: 1;
    min-width: 0;
  }
  
  &__empty {
    font-size: 14px;
    color: var(--el-text-color-placeholder);
    text-align: center;
    padding: 16px 0;
  }
}
</style>

