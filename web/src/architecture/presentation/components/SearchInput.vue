<template>
  <div class="search-input">
    <WidgetComponent
      v-if="shouldUseWidgetSearchRenderer"
      class="search-control"
      :field="widgetSearchField"
      :value="widgetSearchFieldValue"
      :field-path="field.code"
      mode="search"
      :search-type="searchType"
      :function-method="functionMethod"
      :function-router="functionRouter"
      @update:model-value="handleWidgetFieldUpdate"
    />
    <!-- 🔥 精确搜索 / 模糊搜索 -->
    <el-input
      v-else-if="inputConfig.component === SearchComponent.EL_INPUT"
      class="search-control"
      v-model="localValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :disabled="inputConfig.props?.disabled"
      :style="controlStyle"
      @input="handleInput"
      @clear="handleClear"
    />

    <!-- 🔥 单选 fallback：统一走同一套下拉逻辑，避免样式/行为再继续分叉 -->
    <el-select
      v-else-if="isSingleFallbackSelect"
      class="search-control user-select-filter"
      v-model="selectValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :filterable="inputConfig.props?.filterable"
      :remote="inputConfig.props?.remote"
      :remote-method="handleRemoteMethod"
      :loading="selectLoading || inputConfig.props?.loading"
      :popper-class="inputConfig.props?.popperClass"
      :teleported="shouldTeleportPopper"
      :style="controlStyle"
      :reserve-keyword="inputConfig.props?.remote"
      @change="handleInput"
      @clear="handleClear"
      @visible-change="handleVisibleChange"
    >
      <el-option
        v-for="option in selectOptionsComputed"
        :key="getRenderedOptionValue(option)"
        :label="getRenderedOptionLabel(option)"
        :value="getRenderedOptionValue(option)"
      >
        <SearchSelectOptionContent
          :label="getRenderedOptionLabel(option)"
          :user-info="getRenderedOptionUserInfo(option)"
          :show-color-indicator="shouldShowColoredMultiFallbackOption"
          :color-style="getOptionColorStyle(getRenderedOptionValue(option))"
        />
      </el-option>
    </el-select>
    <!-- 🔥 多选组件 -->
    <el-select
      v-else-if="isMultipleFallbackSelect"
      class="search-control user-select-filter"
      v-model="selectValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :filterable="inputConfig.props?.filterable"
      :remote="inputConfig.props?.remote"
      :remote-method="handleRemoteMethod"
      :multiple="inputConfig.props?.multiple"
      :loading="selectLoading || inputConfig.props?.loading"
      :popper-class="inputConfig.props?.popperClass"
      :teleported="shouldTeleportPopper"
      :style="controlStyle"
      :collapse-tags="inputConfig.props?.multiple"
      :max-collapse-tags="SearchConfig.MAX_COLLAPSE_TAGS"
      :reserve-keyword="inputConfig.props?.remote && inputConfig.props?.multiple"
      @change="handleInput"
      @clear="handleClear"
      @visible-change="handleVisibleChange"
    >
      <!-- 🔥 自定义标签显示（multiple 模式） -->
      <template v-if="shouldUseCustomFallbackTags" #tag>
        <!-- 🔥 用户选择器：使用 user-cell 样式 -->
        <template v-if="shouldUseUserFallbackTags">
          <UserFilterChip
            v-for="value in fallbackTagSummary.visibleValues"
            :key="value"
            :label="getOptionLabel(value) || ''"
            :avatar="getUserInfoByValue(value)?.avatar || null"
            :initial="getUserTagInitial(value)"
            @remove="handleRemoveTag(value)"
          />
          <el-tag
            v-if="fallbackTagSummary.hiddenCount > 0"
            class="filter-summary-chip"
            size="small"
            disable-transitions
          >
            +{{ fallbackTagSummary.hiddenCount }}
          </el-tag>
        </template>
        <!-- 🔥 多选组件：使用带颜色的标签 -->
        <template v-else-if="shouldUseColoredFallbackTags">
          <el-tag
            v-for="value in fallbackTagSummary.visibleValues"
            :key="value"
            :type="getOptionColorType(value)"
            :color="getOptionColorValue(value)"
            :style="getSelectTagStyle(value)"
            :closable="true"
            @close.stop="handleRemoveTag(value)"
            class="multiselect-tag"
          >
            {{ getOptionLabel(value) }}
          </el-tag>
          <el-tag
            v-if="fallbackTagSummary.hiddenCount > 0"
            class="filter-summary-chip"
            size="small"
            disable-transitions
          >
            +{{ fallbackTagSummary.hiddenCount }}
          </el-tag>
        </template>
        <template v-else-if="shouldUseNeutralFallbackTags">
          <el-tag
            v-for="value in fallbackTagSummary.visibleValues"
            :key="value"
            :closable="true"
            @close.stop="handleRemoveTag(value)"
            class="multiselect-tag multiselect-tag-neutral"
          >
            {{ getOptionLabel(value) }}
          </el-tag>
          <el-tag
            v-if="fallbackTagSummary.hiddenCount > 0"
            class="filter-summary-chip"
            size="small"
            disable-transitions
          >
            +{{ fallbackTagSummary.hiddenCount }}
          </el-tag>
        </template>
      </template>
      
      <el-option
        v-for="option in selectOptionsComputed"
        :key="getRenderedOptionValue(option)"
        :label="getRenderedOptionLabel(option)"
        :value="getRenderedOptionValue(option)"
      >
        <SearchSelectOptionContent
          :label="getRenderedOptionLabel(option)"
          :user-info="getRenderedOptionUserInfo(option)"
          :show-color-indicator="shouldShowColoredMultiFallbackOption"
          :color-style="getOptionColorStyle(getRenderedOptionValue(option))"
        />
      </el-option>
    </el-select>

    <!-- 🔥 数字范围输入 -->
    <div v-else-if="inputConfig.component === SearchComponent.NUMBER_RANGE_INPUT" class="number-range">
      <el-input-number
        class="search-range-field"
        v-model="rangeValue.min"
        :placeholder="inputConfig.props?.minPlaceholder"
        :precision="inputConfig.props?.precision"
        :step="inputConfig.props?.step"
        :min="inputConfig.props?.min"
        :max="inputConfig.props?.max"
        :clearable="true"
        :controls-position="'right'"
        :style="rangeFieldStyle"
        @change="handleRangeChange"
      />
      <span class="range-separator">至</span>
      <el-input-number
        class="search-range-field"
        v-model="rangeValue.max"
        :placeholder="inputConfig.props?.maxPlaceholder"
        :precision="inputConfig.props?.precision"
        :step="inputConfig.props?.step"
        :min="inputConfig.props?.min"
        :max="inputConfig.props?.max"
        :clearable="true"
        :controls-position="'right'"
        :style="rangeFieldStyle"
        @change="handleRangeChange"
      />
    </div>

    <!-- 🔥 日期范围选择 -->
    <el-date-picker
      v-else-if="inputConfig.component === SearchComponent.EL_DATE_PICKER"
      class="search-control"
      v-model="dateRangeValue"
      :type="inputConfig.props?.type"
      :range-separator="inputConfig.props?.rangeSeparator"
      :start-placeholder="inputConfig.props?.startPlaceholder"
      :end-placeholder="inputConfig.props?.endPlaceholder"
      :format="inputConfig.props?.format"
      :value-format="inputConfig.props?.valueFormat"
      :shortcuts="inputConfig.props?.shortcuts"
      :clearable="inputConfig.props?.clearable"
      :teleported="shouldTeleportPopper"
      :style="controlStyle"
      @change="handleDateRangeChange"
      @clear="handleClear"
    />

    <!-- 🔥 文本范围输入（默认降级） -->
    <div v-else-if="inputConfig.component === SearchComponent.RANGE_INPUT" class="text-range">
      <el-input
        class="search-range-field"
        v-model="rangeValue.min"
        :placeholder="inputConfig.props?.minPlaceholder"
        clearable
        :style="rangeFieldStyle"
        @input="handleRangeChange"
      />
      <span class="range-separator">至</span>
      <el-input
        class="search-range-field"
        v-model="rangeValue.max"
        :placeholder="inputConfig.props?.maxPlaceholder"
        clearable
        :style="rangeFieldStyle"
        @input="handleRangeChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, watch, onUnmounted, provide, toRef, type ComputedRef } from 'vue'
import { createPinia } from 'pinia'
import { ElTag } from 'element-plus'
import SearchSelectOptionContent from './SearchSelectOptionContent.vue'
import UserFilterChip from './UserFilterChip.vue'
import { prdPreviewContextKey } from './prdPreviewContext'
import { widgetComponentFactory } from '@/architecture/presentation/widgets/registry'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import { ErrorHandler } from '@/architecture/presentation/utils/ErrorHandler'
import { convertToFieldValue } from '@/architecture/domain/utils/field'
import { normalizeSearchValue, denormalizeSearchValue } from '@/architecture/domain/utils/searchValueNormalizer'
import { getSearchFieldRawValue, isStoredSearchFieldValue } from '@/architecture/domain/utils/searchFieldValue'
import { createSearchComponentConfig } from '@/architecture/presentation/components/utils/searchComponentConfig'
import { SearchConfig, SearchComponent, hasSearchType } from '@/architecture/domain/constants/search'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { Logger } from '@/architecture/shared/logger'
import type { FieldConfig } from '@/architecture/domain/types'
import { formDataStoreKey, useFormDataStore } from '@/architecture/infrastructure/stores/formData'
import {
  buildSearchWidgetField,
  adaptSearchModelValueForWidget,
  resolveWidgetTypeForSearchRenderer,
  shouldUseWidgetSearchRenderer as resolveWidgetSearchMode
} from './utils/searchWidgetMode'
import { buildSearchControlStyle, buildSearchRangeFieldStyle } from './utils/searchControlStyle'
import { useSearchInputModelState } from './composables/useSearchInputModelState'
import { useSearchInputFallbackSelect } from './composables/useSearchInputFallbackSelect'
import type { SearchInputConfig } from './utils/searchInputTypes'

interface Props {
  field: FieldConfig
  searchType: string
  modelValue: any
  // 🔥 用于 selectFuzzy 回调（可选）
  functionMethod?: string
  functionRouter?: string
}

interface Emits {
  (e: 'update:modelValue', value: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const prdPreviewContext = inject(prdPreviewContextKey, null)
const shouldTeleportPopper = computed(() => !prdPreviewContext?.interactive)

const scopedSearchPinia = createPinia()
const scopedSearchFormDataStore = useFormDataStore(scopedSearchPinia)
provide(formDataStoreKey, scopedSearchFormDataStore)

const searchWidgetType = computed(() => {
  return resolveWidgetTypeForSearchRenderer({
    widgetType: props.field.widget?.type,
    searchType: props.searchType
  })
})

const widgetSearchField = computed(() => {
  return buildSearchWidgetField(props.field, props.searchType)
})

const hasSelectFuzzyCallback = computed(() => {
  return props.field.callbacks?.includes('OnSelectFuzzy') || false
})

const shouldPreferInlineSelectSearch = computed(() => {
  return (
    (searchWidgetType.value === WidgetType.SELECT || searchWidgetType.value === WidgetType.MULTI_SELECT) &&
    !hasSelectFuzzyCallback.value
  )
})

const shouldUseWidgetSearchRenderer = computed(() => {
  if (shouldPreferInlineSelectSearch.value) {
    return false
  }

  return resolveWidgetSearchMode({
    widgetType: searchWidgetType.value,
    searchType: props.searchType,
    hasRegisteredWidget: widgetComponentFactory.hasRequestComponent(searchWidgetType.value)
  })
})

const widgetSearchFieldValue = computed(() => {
  const rawValue = getSearchFieldRawValue(props.modelValue)

  const denormalizedValue = denormalizeSearchValue(rawValue, {
    widgetType: searchWidgetType.value,
    searchType: props.searchType,
    field: widgetSearchField.value
  })

  const adaptedValue = adaptSearchModelValueForWidget(denormalizedValue, searchWidgetType.value)

  if (isStoredSearchFieldValue(props.modelValue)) {
    return {
      ...props.modelValue,
      raw: adaptedValue,
      display: props.modelValue.display || String(adaptedValue ?? ''),
      meta: props.modelValue.meta || {}
    }
  }

  return convertToFieldValue(adaptedValue, widgetSearchField.value)
})

const inputConfig = computed(() => {
  try {
    return createSearchComponentConfig(
      props.field,
      props.searchType,
      props.functionMethod,
      props.functionRouter
    )
  } catch (error) {
    return ErrorHandler.handleWidgetError('SearchInput.inputConfig', error, {
      showMessage: false,
      fallbackValue: {
        component: SearchComponent.EL_INPUT,
        props: {
          placeholder: `请输入${props.field.name}`,
          clearable: true,
          style: { width: SearchConfig.DEFAULT_INPUT_WIDTH }
        }
      }
    })
  }
})

const controlStyle = computed(() => {
  return buildSearchControlStyle(inputConfig.value.props?.style)
})

const rangeFieldStyle = computed(() => {
  return buildSearchRangeFieldStyle()
})

watch(
  () => (shouldUseWidgetSearchRenderer.value ? widgetSearchFieldValue.value : null),
  (newValue) => {
    if (shouldUseWidgetSearchRenderer.value && newValue) {
      scopedSearchFormDataStore.setValue(props.field.code, newValue)
    }
  },
  { immediate: true, deep: true }
)

onUnmounted(() => {
  scopedSearchFormDataStore.clear()
})

let triggerInitSelectedOptions: () => Promise<void> = async () => {}

const {
  localValue,
  shouldShowValue,
  selectValue,
  dateRangeValue,
  rangeValue,
  handleInput,
  handleClear,
  handleRangeChange,
  handleDateRangeChange
} = useSearchInputModelState({
  field: props.field,
  searchType: props.searchType,
  modelValue: toRef(props, 'modelValue'),
  inputConfig: inputConfig as ComputedRef<SearchInputConfig>,
  shouldUseWidgetSearchRenderer,
  emitUpdate: (value) => emit('update:modelValue', value),
  initSelectedOptions: () => triggerInitSelectedOptions()
})

const {
  selectLoading,
  isSingleFallbackSelect,
  isMultipleFallbackSelect,
  isMultiselectWidget,
  isSelectWidget,
  shouldUseUserFallbackTags,
  shouldUseColoredFallbackTags,
  shouldUseCustomFallbackTags,
  shouldUseNeutralFallbackTags,
  shouldShowColoredMultiFallbackOption,
  fallbackTagSummary,
  selectOptionsComputed,
  getOptionColorType,
  getOptionColorValue,
  getSelectTagStyle,
  getOptionColorStyle,
  getOptionLabel,
  getRenderedOptionValue,
  getRenderedOptionLabel,
  getRenderedOptionUserInfo,
  getUserInfoByValue,
  getUserTagInitial,
  handleRemoveTag,
  handleRemoteMethod,
  handleVisibleChange,
  initSelectedOptions
} = useSearchInputFallbackSelect({
  field: props.field,
  inputConfig: inputConfig as ComputedRef<SearchInputConfig>,
  localValue,
  shouldShowValue,
  handleInput
})
triggerInitSelectedOptions = initSelectedOptions

const handleWidgetFieldUpdate = (value: any) => {
  const normalizedValue = normalizeSearchValue(value?.raw, {
    widgetType: searchWidgetType.value,
    searchType: props.searchType,
    field: widgetSearchField.value
  })

  if (value && typeof value === 'object' && 'raw' in value && 'display' in value) {
    emit('update:modelValue', {
      ...value,
      raw: normalizedValue,
      meta: value.meta || {}
    })
    return
  }

  emit('update:modelValue', normalizedValue)
}
</script>

<style scoped>
.search-input {
  display: flex;
  align-items: stretch;
  flex: 1 1 auto;
  width: 100%;
  min-width: 0;
}

.search-control {
  display: block;
  flex: 1 1 auto;
  width: 100%;
  min-width: 0;
}

.number-range,
.text-range {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.range-separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  flex-shrink: 0;
}

.search-input :deep(.el-input),
.search-input :deep(.el-select),
.search-input :deep(.el-date-editor),
.search-input :deep(.el-date-editor--daterange),
.search-input :deep(.el-date-editor--timerange),
.search-input :deep(.el-time-editor),
.search-input :deep(.el-input-number),
.search-input :deep(.widget-component) {
  width: 100%;
  min-width: 0;
}

.search-input :deep(.el-select__wrapper),
.search-input :deep(.el-input__wrapper),
.search-input :deep(.el-date-editor .el-input__wrapper),
.search-input :deep(.el-textarea__inner) {
  width: 100%;
  min-width: 0;
}

.search-input :deep(.select-widget),
.search-input :deep(.multiselect-widget),
.search-input :deep(.department-widget),
.search-input :deep(.user-search-widget),
.search-input :deep(.textarea-widget),
.search-input :deep(.rich-text-widget) {
  width: 100%;
  min-width: 0;
}

.search-input :deep(.select-container),
.search-input :deep(.department-select-display),
.search-input :deep(.departments-select-display),
.search-input :deep(.user-search-display) {
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.number-range :deep(.el-input-number),
.text-range :deep(.el-input) {
  flex: 1 1 0;
  min-width: 0;
}

.search-range-field {
  flex: 1 1 0;
  min-width: 0;
}

/* 🔥 用户选择器选中后的标签样式（multiple 模式，使用 user-cell 样式） */
.user-select-filter :deep(.el-select__tags) {
  display: flex;
  flex-wrap: nowrap;
  gap: 6px;
  align-items: center;
  overflow: hidden;
  min-width: 0;
  max-width: 100%;
}

/* 🔥 多选组件标签样式 */
.multiselect-tag {
  font-weight: 500;
  border: 1px solid var(--el-border-color-lighter);
  background-color: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  box-shadow: none;
  margin-right: 6px;
  margin-bottom: 2px;
  opacity: 0.9;
  transition: opacity 0.2s;
}

.filter-summary-chip {
  flex-shrink: 0;
  margin-right: 0;
  border: 1px solid var(--el-border-color-lighter);
  background-color: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}

.multiselect-tag:hover {
  opacity: 1;
}

/* 自定义颜色的 tag，确保文字清晰 */
.multiselect-tag[style*="background-color"] {
  font-weight: 500;
}

/* 🔥 单选组件的标签样式：使用空心样式（outline） */
.select-tag {
  font-weight: 500;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  opacity: 0.9;
  transition: opacity 0.2s;
}

.select-tag:hover {
  opacity: 1;
}

/* 🔥 单选组件标签样式：使用空心样式（outline） */
.select-tag-outline {
  background-color: transparent !important;
  border: 2px solid currentColor !important;
}

/* 标准颜色的空心标签 */
.select-tag-outline.el-tag--success {
  color: var(--el-color-success) !important;
  border-color: var(--el-color-success) !important;
}

.select-tag-outline.el-tag--warning {
  color: var(--el-color-warning) !important;
  border-color: var(--el-color-warning) !important;
}

.select-tag-outline.el-tag--danger {
  color: var(--el-color-danger) !important;
  border-color: var(--el-color-danger) !important;
}

.select-tag-outline.el-tag--info {
  color: var(--el-color-info) !important;
  border-color: var(--el-color-info) !important;
}

.select-tag-outline.el-tag--primary {
  color: var(--el-color-primary) !important;
  border-color: var(--el-color-primary) !important;
}

/* 自定义颜色的空心标签：使用边框颜色 */
.select-tag-outline[style*="color"] {
  border-color: currentColor !important;
}

</style>

<style>
/* 🔥 用户选择器下拉框样式（全局样式，与 UserWidget 保持一致） */
.user-select-dropdown-popper .el-select-dropdown__item {
  padding: 8px 12px;
}

.user-select-dropdown-popper .el-select-dropdown__item:hover {
  background-color: var(--el-fill-color-light);
}

</style>
