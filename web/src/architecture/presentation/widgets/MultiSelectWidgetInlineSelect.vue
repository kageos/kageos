<template>
  <div class="multiselect-inline">
    <el-select
      :model-value="modelValue"
      :class="searchMode ? 'inline-multiselect inline-multiselect-search' : 'inline-multiselect'"
      multiple
      filterable
      :clearable="true"
      :collapse-tags="true"
      :max-collapse-tags="1"
      :placeholder="placeholder"
      :disabled="disabled"
      :allow-create="creatable"
      :default-first-option="creatable"
      :teleported="teleported"
      @update:model-value="emit('update:modelValue', $event)"
      @clear="emit('clear')"
    >
      <template #tag>
        <el-tag
          v-for="value in visibleValues"
          :key="String(value)"
          :type="getOptionColorType(value)"
          :color="getOptionColorValue(value)"
          effect="light"
          :style="getOptionTagStyle(value)"
          :closable="true"
          :class="['filter-selected-chip', 'inline-selected-tag', { 'inline-selected-tag-neutral': !getOptionColor(value) }]"
          @close.stop="emit('remove-tag', value)"
        >
          {{ getOptionLabel(value) }}
        </el-tag>
        <el-tag
          v-if="hiddenCount > 0"
          class="filter-selected-chip filter-summary-chip inline-summary-tag"
          size="small"
          disable-transitions
        >
          +{{ hiddenCount }}
        </el-tag>
      </template>

      <el-option
        v-for="option in options"
        :key="String(option.value)"
        :label="option.label"
        :value="option.value"
        :disabled="option.disabled"
      >
        <div class="multiselect-option">
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
    <div v-if="displayInfoText" class="display-info-text inline-display-info-text">
      {{ displayInfoText }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElOption, ElSelect, ElTag } from 'element-plus'
import type { StandardColorType } from '@/architecture/domain/constants/select'
import type { MultiSelectOptionItem } from './multiSelectWidgetTypes'

withDefaults(defineProps<{
  modelValue: any[]
  options: MultiSelectOptionItem[]
  placeholder: string
  disabled: boolean
  creatable: boolean
  teleported?: boolean
  visibleValues: any[]
  hiddenCount: number
  searchMode?: boolean
  displayInfoText?: string
  getOptionLabel: (value: any) => string
  getOptionColor: (value: any) => string | null
  getOptionColorType: (value: any) => StandardColorType | undefined
  getOptionColorValue: (value: any) => string | undefined
  getOptionTagStyle: (value: any) => Record<string, string>
  getOptionColorStyle: (value: any) => Record<string, string>
  getOptionDisplayInfo: (option: MultiSelectOptionItem) => string
}>(), {
  teleported: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: any[]]
  clear: []
  'remove-tag': [value: any]
}>()
</script>

<style scoped lang="scss">
@use './styles/inlineSelectShared' as inlineSelectShared;

.multiselect-inline {
  width: 100%;
}

@include inlineSelectShared.inline-select-surface('.inline-multiselect', 40px, 13px, $padding-top: 4px, $padding-bottom: 4px, $align-start: true);
@include inlineSelectShared.inline-select-surface('.inline-multiselect-search', 32px, 13px, $box-shadow: none, $padding-left: 9px, $padding-right: 9px, $padding-top: 1px, $padding-bottom: 1px, $align-start: true);
@include inlineSelectShared.inline-select-display-info('.inline-display-info-text', 2px);
@include inlineSelectShared.inline-select-option-row('.multiselect-option');

.inline-multiselect :deep(.el-select__selection) {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 28px;
}

.inline-multiselect :deep(.el-select__input-wrapper) {
  min-width: 96px;
}

.inline-multiselect :deep(.el-select__input) {
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.inline-multiselect :deep(.el-select__placeholder) {
  font-size: 13px;
}

.inline-multiselect :deep(.el-select__tags-text) {
  max-width: none;
}

.filter-selected-chip {
  margin: 0;
  max-width: min(100%, 160px);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
}

.inline-selected-tag {
  max-width: min(100%, 180px);
}

.inline-selected-tag,
.inline-summary-tag {
  height: 24px;
  line-height: 22px;
  border-radius: 999px;
  padding: 0 10px;
  border-width: 1px;
  border-style: solid;
  box-shadow: none;
  font-weight: 500;
}

.inline-selected-tag-neutral,
.inline-summary-tag {
  background-color: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  border-color: var(--el-border-color-lighter);
}

.inline-selected-tag :deep(.el-tag__close) {
  margin-left: 6px;
}

.filter-summary-chip {
  flex-shrink: 0;
}

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
  border-radius: 2px;
  flex-shrink: 0;
  border: none;
  vertical-align: middle;
  filter: brightness(0.95) saturate(0.9);
  opacity: 0.9;
}
</style>
