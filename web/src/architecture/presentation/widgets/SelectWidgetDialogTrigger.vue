<template>
  <div
    class="select-container"
    :class="{ 'is-search-mode': searchMode, 'has-value': hasValue }"
    @click="$emit('open')"
  >
    <div class="select-content">
      <div class="select-main">
        <template v-if="searchMode && hasValue">
          <div class="search-selected-tag">
            <span class="search-selected-indicator" />
            <span class="select-label">{{ displayValue }}</span>
          </div>
          <div class="select-actions">
            <el-icon
              v-if="showClear"
              class="search-tag-remove"
              @click.stop="$emit('clear')"
            >
              <Close />
            </el-icon>
            <el-icon class="input-icon input-icon-active"><ArrowDown /></el-icon>
          </div>
        </template>
        <template v-else>
          <span :class="hasValue ? 'select-label' : 'select-placeholder'">
            {{ hasValue ? displayValue : fallbackLabel }}
          </span>
          <div class="select-actions">
            <el-icon
              v-if="showClear"
              class="clear-icon"
              @click.stop="$emit('clear')"
            >
              <CircleClose />
            </el-icon>
            <el-icon class="input-icon"><ArrowDown /></el-icon>
          </div>
        </template>
      </div>
      <div v-if="displayInfoText" class="display-info-text">
        {{ displayInfoText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElIcon } from 'element-plus'
import { ArrowDown, CircleClose, Close } from '@element-plus/icons-vue'

defineProps<{
  displayValue: string
  fallbackLabel: string
  displayInfoText?: string
  showClear: boolean
  hasValue: boolean
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

.select-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.select-label,
.select-placeholder {
  flex: 1;
  font-size: 14px;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-label {
  color: var(--el-text-color-primary);
}

.select-placeholder {
  color: var(--el-text-color-placeholder);
}

.select-container.is-search-mode .select-label,
.select-container.is-search-mode .select-placeholder {
  font-size: 13px;
}

.select-container.is-search-mode.has-value {
  border-color: var(--el-color-primary-light-5);
  background: linear-gradient(180deg, var(--el-color-primary-light-9) 0%, var(--el-fill-color-blank) 100%);
}

.select-container.is-search-mode.has-value:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 16%, transparent);
}

.search-selected-tag {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  flex: 1;
  padding: 3px 8px;
  border-radius: 999px;
  background-color: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
}

.search-selected-tag .select-label {
  font-weight: 500;
  color: var(--el-color-primary-dark-2);
}

.search-selected-indicator {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background-color: var(--el-color-primary);
  flex-shrink: 0;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 14%, transparent);
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

.search-tag-remove {
  width: 18px;
  height: 18px;
  border-radius: 999px;
  color: var(--el-color-primary);
  background-color: color-mix(in srgb, var(--el-color-primary) 10%, white);
  transition: all 0.2s;
  cursor: pointer;
}

.search-tag-remove:hover {
  color: white;
  background-color: var(--el-color-danger);
}

.input-icon-active {
  color: var(--el-color-primary);
}

.select-container:hover .input-icon {
  color: var(--el-color-primary);
  transform: translateY(1px);
}
</style>
