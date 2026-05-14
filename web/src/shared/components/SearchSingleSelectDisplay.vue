<template>
  <div
    class="search-single-select-display"
    :class="{ 'has-value': hasValue, 'is-disabled': disabled }"
    :title="hasValue ? label : placeholder"
    @click="!disabled && emit('open')"
  >
    <div class="search-single-select-main">
      <template v-if="hasValue">
        <div class="search-selected-value">
          <span v-if="$slots.leading" class="search-selected-leading">
            <slot name="leading" />
          </span>
          <span class="search-selected-label">{{ label }}</span>
        </div>
      </template>
      <span v-else class="search-placeholder">{{ placeholder }}</span>

      <div class="search-select-actions">
        <el-icon
          v-if="showClear && !disabled"
          class="selected-value-remove"
          @click.stop="emit('clear')"
        >
          <Close />
        </el-icon>
        <el-icon class="search-open-icon"><ArrowDown /></el-icon>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ElIcon } from 'element-plus'
import { ArrowDown, Close } from '@element-plus/icons-vue'

withDefaults(defineProps<{
  label: string
  placeholder: string
  hasValue: boolean
  showClear?: boolean
  disabled?: boolean
  displayInfoText?: string
}>(), {
  showClear: false,
  disabled: false,
  displayInfoText: ''
})

const emit = defineEmits<{
  (e: 'open'): void
  (e: 'clear'): void
}>()
</script>

<style scoped>
.search-single-select-display {
  width: 100%;
  height: 32px;
  padding: 0 11px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--el-bg-color) 96%, var(--el-color-primary) 4%), var(--el-bg-color)),
    var(--el-fill-color-blank);
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease, box-shadow 0.2s ease;
  box-sizing: border-box;
  display: flex;
  align-items: center;
}

.search-single-select-display:hover:not(.is-disabled) {
  border-color: rgba(var(--el-color-primary-rgb), 0.46);
  background:
    linear-gradient(180deg, rgba(var(--el-color-primary-rgb), 0.1), rgba(var(--el-color-primary-rgb), 0.045)),
    var(--el-bg-color);
  box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.08);
}

.search-single-select-display.has-value {
  border-color: rgba(var(--el-color-primary-rgb), 0.2);
}

.search-single-select-display:focus-within {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.12);
}

.search-single-select-display.is-disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.search-single-select-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
  width: 100%;
}

.search-selected-value {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  flex: 1;
}

.search-selected-leading {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.search-selected-label,
.search-placeholder {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  line-height: 1.2;
}

.search-selected-label {
  color: var(--el-text-color-primary);
  font-weight: 600;
  flex: 1;
}

.search-single-select-display:hover:not(.is-disabled) .search-selected-label {
  color: var(--el-color-primary);
}

.search-placeholder {
  color: var(--el-text-color-secondary);
  flex: 1;
}

.search-select-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.selected-value-remove {
  width: 18px;
  height: 18px;
  border-radius: 4px;
  color: var(--el-text-color-secondary);
  background-color: transparent;
  transition: all 0.2s;
  cursor: pointer;
}

.selected-value-remove:hover {
  color: var(--el-color-danger);
  background-color: var(--el-fill-color-light);
}

.search-open-icon {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  flex-shrink: 0;
  transition: all 0.2s;
}

.search-single-select-display:hover:not(.is-disabled) .search-open-icon,
.search-single-select-display.has-value .search-open-icon {
  color: var(--el-color-primary);
}

:slotted(.search-selected-indicator) {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--el-color-primary);
  flex-shrink: 0;
}

:slotted(.search-selected-avatar) {
  width: 16px !important;
  height: 16px !important;
  flex-shrink: 0;
}

:slotted(.search-selected-icon) {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

:slotted(.search-selected-indicator) {
  display: none;
}
</style>
