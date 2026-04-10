<template>
  <div class="select-widget-inline">
    <el-select
      :model-value="modelValue"
      :class="searchMode ? 'inline-select inline-select-search' : 'inline-select'"
      :placeholder="placeholder"
      :disabled="disabled"
      :clearable="clearable"
      :filterable="true"
      :allow-create="creatable"
      :default-first-option="creatable"
      @update:model-value="emit('update:modelValue', $event)"
      @clear="emit('clear')"
    >
      <el-option
        v-for="option in options"
        :key="String(option.value)"
        :label="option.label"
        :value="option.value"
        :disabled="option.disabled"
      >
        <div class="select-option">
          <span
            v-if="getOptionColor(option.value)"
            class="option-color-indicator"
            :style="getOptionColorStyle(option.value)"
          />
          <span class="option-label">{{ option.label }}</span>
          <span v-if="getOptionDisplayInfo(option)" class="display-info">
            {{ getOptionDisplayInfo(option) }}
          </span>
        </div>
      </el-option>
    </el-select>
    <div v-if="displayInfoText" class="display-info-text inline-select-display-info">
      {{ displayInfoText }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElOption, ElSelect } from 'element-plus'
import type { SelectOptionItem } from './selectWidgetTypes'

defineProps<{
  modelValue: any
  options: SelectOptionItem[]
  placeholder: string
  disabled: boolean
  clearable: boolean
  creatable: boolean
  searchMode?: boolean
  displayInfoText?: string
  getOptionColor: (value: any) => string | null
  getOptionColorStyle: (value: any) => Record<string, string>
  getOptionDisplayInfo: (option: SelectOptionItem) => string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
  clear: []
}>()
</script>

<style scoped lang="scss">
@use './styles/inlineSelectShared' as inlineSelectShared;

.select-widget-inline {
  width: 100%;
}

@include inlineSelectShared.inline-select-surface('.inline-select', 40px, 14px);
@include inlineSelectShared.inline-select-surface('.inline-select-search', 32px, 13px, $box-shadow: none);
@include inlineSelectShared.inline-select-display-info('.inline-select-display-info');
@include inlineSelectShared.inline-select-option-row('.select-option');

.display-info-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.option-color-indicator {
  display: inline-block !important;
  width: 12px !important;
  height: 12px !important;
  min-width: 12px !important;
  min-height: 12px !important;
  border-radius: 2px !important;
  flex-shrink: 0 !important;
  border: none !important;
  vertical-align: middle !important;
  filter: brightness(0.95) saturate(0.9);
  opacity: 0.9;
}
</style>
