<template>
  <div class="select-container" :class="{ 'is-search-mode': searchMode }" @click="$emit('open')">
    <div class="select-content">
      <div class="select-main">
        <span class="select-label">{{ displayValue || fallbackLabel }}</span>
        <el-icon
          v-if="showClear"
          class="clear-icon"
          @click.stop="$emit('clear')"
        >
          <CircleClose />
        </el-icon>
        <el-icon class="input-icon"><ArrowDown /></el-icon>
      </div>
      <div v-if="displayInfoText" class="display-info-text">
        {{ displayInfoText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElIcon } from 'element-plus'
import { ArrowDown, CircleClose } from '@element-plus/icons-vue'

defineProps<{
  displayValue: string
  fallbackLabel: string
  displayInfoText?: string
  showClear: boolean
  searchMode?: boolean
}>()

defineEmits<{
  (e: 'open'): void
  (e: 'clear'): void
}>()
</script>

<style scoped lang="scss">
.select-container {
  width: 100%;
  min-height: 40px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background-color: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.select-container:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.select-container.is-search-mode {
  min-height: 32px;
  padding: 5px 10px;
  box-shadow: none;
}

.select-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.select-container.is-search-mode .select-content {
  gap: 0;
}

.select-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.select-label {
  flex: 1;
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-container.is-search-mode .select-label {
  font-size: 13px;
}

.display-info-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.clear-icon {
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: all 0.2s;
  flex-shrink: 0;
  cursor: pointer;
  padding: 2px;
  border-radius: 50%;
}

.clear-icon:hover {
  color: var(--el-color-danger);
  background-color: var(--el-color-danger-light-9);
}

.input-icon {
  color: var(--el-text-color-placeholder);
  transition: all 0.2s;
  font-size: 14px;
  flex-shrink: 0;
}

.select-container.is-search-mode .clear-icon,
.select-container.is-search-mode .input-icon {
  font-size: 14px;
}

.select-container:hover .input-icon {
  color: var(--el-color-primary);
  transform: translateY(1px);
}
</style>
