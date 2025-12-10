<template>
  <el-card shadow="hover" class="stat-card" :class="{ 'stat-card--clickable': clickable }" @click="handleClick">
    <div class="stat-card__content">
      <div class="stat-card__icon" :style="{ color: iconColor }">
        <el-icon :size="32">
          <component :is="icon" />
        </el-icon>
      </div>
      <div class="stat-card__info">
        <div class="stat-card__value">{{ value }}</div>
        <div class="stat-card__label">{{ label }}</div>
        <div v-if="description" class="stat-card__description">{{ description }}</div>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  icon: any // 图标组件
  label: string // 标签
  value: number | string // 数值
  description?: string // 描述
  iconColor?: string // 图标颜色
  clickable?: boolean // 是否可点击
}

const props = withDefaults(defineProps<Props>(), {
  iconColor: 'var(--el-color-primary)',
  clickable: false
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
.stat-card {
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
  
  &__content {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  
  &__icon {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    border-radius: 8px;
    background: var(--el-color-primary-light-9);
  }
  
  &__info {
    flex: 1;
    min-width: 0;
  }
  
  &__value {
    font-size: 28px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    line-height: 1.2;
    margin-bottom: 4px;
  }
  
  &__label {
    font-size: 14px;
    color: var(--el-text-color-regular);
    line-height: 1.4;
    margin-bottom: 2px;
  }
  
  &__description {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
  }
}
</style>

