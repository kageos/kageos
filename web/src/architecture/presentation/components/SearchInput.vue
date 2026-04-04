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
      class="search-control user-select-search"
      v-model="selectValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :filterable="inputConfig.props?.filterable"
      :remote="inputConfig.props?.remote"
      :remote-method="handleRemoteMethod"
      :loading="selectLoading || inputConfig.props?.loading"
      :popper-class="inputConfig.props?.popperClass"
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
        />
      </el-option>
    </el-select>
    <!-- 🔥 多选组件 -->
    <el-select
      v-else-if="isMultipleFallbackSelect"
      class="search-control user-select-search"
      v-model="selectValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :filterable="inputConfig.props?.filterable"
      :remote="inputConfig.props?.remote"
      :remote-method="handleRemoteMethod"
      :multiple="inputConfig.props?.multiple"
      :loading="selectLoading || inputConfig.props?.loading"
      :popper-class="inputConfig.props?.popperClass"
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
          <SearchUserTag
            v-for="value in fallbackTagSummary.visibleValues"
            :key="value"
            :label="getOptionLabel(value) || ''"
            :avatar="getUserInfoByValue(value)?.avatar || null"
            :initial="getUserTagInitial(value)"
            @remove="handleRemoveTag(value)"
          />
          <el-tag
            v-if="fallbackTagSummary.hiddenCount > 0"
            class="search-summary-tag"
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
            :closable="true"
            @close.stop="handleRemoveTag(value)"
            class="multiselect-tag"
          >
            {{ getOptionLabel(value) }}
          </el-tag>
          <el-tag
            v-if="fallbackTagSummary.hiddenCount > 0"
            class="search-summary-tag"
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
            class="search-summary-tag"
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
import { ref, computed, watch, nextTick, onMounted, onUnmounted, provide } from 'vue'
import { createPinia } from 'pinia'
import { ElTag } from 'element-plus'
import SearchSelectOptionContent from './SearchSelectOptionContent.vue'
import SearchUserTag from './SearchUserTag.vue'
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import { convertToFieldValue } from '@/utils/field'
import { normalizeSearchValue, denormalizeSearchValue } from '@/utils/searchValueNormalizer'
import { createSearchComponentConfig } from '@/utils/searchComponentConfig'
import { SearchConfig, SearchComponent, SearchType, hasSearchType } from '@/core/constants/search'
import { WidgetType } from '@/core/constants/widget'
import { parseCommaSeparatedString } from '@/utils/stringUtils'
import { isStandardColor, getStandardColorCSSVar, type StandardColorType } from '@/core/constants/select'
import { Logger } from '@/core/utils/logger'
import type { FieldConfig } from '@/core/types/field'
import { formDataStoreKey, useFormDataStore } from '@/core/stores-v2/formData'
import {
  buildSearchWidgetField,
  adaptSearchModelValueForWidget,
  resolveWidgetTypeForSearchRenderer,
  shouldUseWidgetSearchRenderer as resolveWidgetSearchMode
} from './utils/searchWidgetMode'
import { buildSearchControlStyle, buildSearchRangeFieldStyle } from './utils/searchControlStyle'
import { buildSearchTagSummary } from '@/architecture/presentation/widgets/utils/searchTagSummary'

type SearchOption = {
  label: string
  value: any
  userInfo?: any
  departmentInfo?: any
}

// 防抖函数
function debounce<T extends (...args: any[]) => any>(func: T, wait: number): T {
  let timeout: ReturnType<typeof setTimeout> | null = null
  return ((...args: any[]) => {
    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(() => {
      func(...args)
    }, wait)
  }) as T
}

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
  const denormalizedValue = denormalizeSearchValue(props.modelValue, {
    widgetType: searchWidgetType.value,
    searchType: props.searchType,
    field: widgetSearchField.value
  })

  return convertToFieldValue(
    adaptSearchModelValueForWidget(denormalizedValue, searchWidgetType.value),
    widgetSearchField.value
  )
})

const isFallbackSelect = computed(() => {
  return inputConfig.value.component === SearchComponent.EL_SELECT
})

const isSingleFallbackSelect = computed(() => {
  return isFallbackSelect.value && !inputConfig.value.props?.multiple
})

const isMultipleFallbackSelect = computed(() => {
  return isFallbackSelect.value && !!inputConfig.value.props?.multiple
})

const shouldUseUserFallbackTags = computed(() => {
  return inputConfig.value.props?.popperClass === 'user-select-dropdown-popper'
})

const shouldUseColoredFallbackTags = computed(() => {
  return isMultiselectWidget.value
})

const shouldUseCustomFallbackTags = computed(() => {
  return isMultipleFallbackSelect.value
})

const shouldUseNeutralFallbackTags = computed(() => {
  return shouldUseCustomFallbackTags.value &&
    !shouldUseUserFallbackTags.value &&
    !shouldUseColoredFallbackTags.value
})

const shouldShowColoredMultiFallbackOption = computed(() => {
  return isMultiselectWidget.value || isSelectWidget.value
})

const fallbackTagSummary = computed(() => {
  const values = Array.isArray(localValue.value) ? localValue.value : []
  return buildSearchTagSummary(values, 1)
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

// 本地值（单值）
const localValue = ref(props.modelValue)

// 🔥 用于控制是否显示值（在 remote 模式下，等选项加载完成后再显示）
const shouldShowValue = ref(true)

// 🔥 计算属性：用于 el-select 的 v-model（避免使用三元表达式）
const selectValue = computed({
  get: () => {
    if (!shouldShowValue.value) {
      // 单选返回 null，多选返回 []
      return inputConfig.value.props?.multiple ? [] : null
    }
    return localValue.value
  },
  set: (val: any) => {
    localValue.value = val
  }
})

// 🔥 防止循环更新的标志
const isInternalUpdate = ref(false)

// 日期范围值（用于 ElDatePicker，数组格式 [start, end]）
const dateRangeValue = ref<[number | string | null, number | string | null] | null>(null)

// 范围值（最小值、最大值，用于 NumberRangeInput 和 RangeInput）
// 🔥 对于时间戳类型，可能是数组 [start, end]，对于数字类型，可能是对象 { min, max }
const rangeValue = ref<any>({
  min: undefined,
  max: undefined
})

// 初始化范围值（在 watch 中处理，这里不需要初始化）

// 下拉选项列表（用于 remote 模式）
const selectOptions = ref<SearchOption[]>([])

// 下拉加载状态
const selectLoading = ref(false)

// 🔥 判断是否是多选组件
const isMultiselectWidget = computed(() => {
  return props.field.widget?.type === WidgetType.MULTI_SELECT
})

// 🔥 判断是否是单选组件
const isSelectWidget = computed(() => {
  return props.field.widget?.type === WidgetType.SELECT
})

// 🔥 获取选项颜色配置
// ⚠️ 关键：直接从 field.widget.config.options_colors 获取，确保能正确获取到 request 字段的颜色配置
const optionColors = computed(() => {
  // 直接从 field.widget.config 获取 options_colors（无论是 response 还是 request 字段）
  const colors = props.field.widget?.config?.options_colors || []
  return colors
})

// 🔥 获取静态选项（用于颜色匹配）
// ⚠️ 关键：优先使用 inputConfig 中的 options（来自 createSearchComponentConfig），
// 如果没有则使用 field.widget.config.options（原始配置）
const staticOptions = computed(() => {
  // 优先使用 inputConfig 中的 options（搜索组件配置中的选项）
  const inputConfigOptions = inputConfig.value.props?.options
  if (inputConfigOptions && Array.isArray(inputConfigOptions) && inputConfigOptions.length > 0) {
    const mapped = inputConfigOptions.map((opt: any) => {
      if (typeof opt === 'string') {
        return { label: opt, value: opt }
      }
      return opt
    })
    return mapped
  }
  
  // 回退到使用 field.widget.config.options（原始配置）
  const opts = props.field.widget?.config?.options || []
  const mapped = opts.map((opt: any) => {
    if (typeof opt === 'string') {
      return { label: opt, value: opt }
    }
    return opt
  })
  return mapped
})

/**
 * 判断是否是 Element Plus 标准颜色类型
 * 
 * 标准颜色：success, warning, danger, info, primary
 * 自定义颜色：以 # 开头的 hex 颜色（如：#FF9800）
 */
// isStandardColor 已从 constants/select 导入

/**
 * 获取选项的颜色
 * 
 * ⚠️ 关键：通过选项索引匹配颜色
 * options_colors 数组的索引对应 options 数组的索引
 * 
 * ⚠️ 重要：使用 staticOptions（与 selectOptionsComputed 使用相同的选项源）进行匹配
 * 确保颜色配置能正确应用到搜索表单中的选项
 * 
 * @param value - 选项值
 * @returns 颜色值（标准颜色名或自定义 hex 颜色），如果未找到返回 null
 */
function getOptionColor(value: any): string | null {
  if (!value) return null
  if (!optionColors.value || optionColors.value.length === 0) return null
  
  const valueStr = String(value)
  // ⚠️ 关键：使用 staticOptions（与 selectOptionsComputed 使用相同的选项源）进行匹配
  const optionIndex = staticOptions.value.findIndex((opt: any) => {
    const optValue = typeof opt === 'object' ? opt.value : opt
    return String(optValue) === valueStr
  })
  
  if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
    return optionColors.value[optionIndex] || null
  }
  
  return null
}

/**
 * 获取选项的颜色类型（用于 el-tag 的 type 属性）
 * 
 * ⚠️ 注意：只有标准颜色才使用 type 属性
 * 自定义颜色使用 color 属性
 * 
 * @param value - 选项值
 * @returns 标准颜色类型（success/warning/danger/info/primary），如果不是标准颜色返回 undefined
 */
function getOptionColorType(value: any): StandardColorType | undefined {
  const color = getOptionColor(value)
  if (!color) return undefined
  return isStandardColor(color) ? (color as StandardColorType) : undefined
}

/**
 * 获取选项的颜色值（用于 el-tag 的 color 属性）
 * 
 * ⚠️ 注意：只有自定义颜色才使用 color 属性
 * 标准颜色使用 type 属性
 * 
 * @param value - 选项值
 * @returns 自定义颜色值（hex 格式，如：#FF9800），如果是标准颜色返回 undefined
 */
function getOptionColorValue(value: any): string | undefined {
  const color = getOptionColor(value)
  if (!color) return undefined
  return !isStandardColor(color) ? color : undefined
}

// 🔥 获取单选标签的样式对象（用于设置边框颜色）
// ⚠️ 注意：对于标准颜色，使用 el-tag 的 type 属性，不需要设置 style
// 对于自定义颜色，才需要设置 style
function getSelectTagStyle(value: any): Record<string, string> {
  const color = getOptionColor(value)
  if (!color) return {}
  
  const isStandard = isStandardColor(color)
  const style: Record<string, string> = {}
  
  // ⚠️ 关键：对于标准颜色，不需要设置 style，使用 el-tag 的 type 属性即可
  // 对于自定义颜色，才需要设置 style
  if (!isStandard) {
    // 自定义颜色：直接使用颜色值设置边框颜色
    style.borderColor = color
    style.color = color
  }
  
  return style
}

// 🔥 获取选项的颜色样式对象（用于 span 的 style 绑定）
function getOptionColorStyle(value: any): Record<string, string> {
  const color = getOptionColor(value)
  if (!color) return {}
  
  const isStandard = isStandardColor(color)
  // 🔥 对于标准颜色，也需要设置背景色（使用 CSS 变量）
  const backgroundColor = isStandard 
    ? getStandardColorCSSVar(color as StandardColorType) 
    : color
  
  const style: Record<string, string> = {
    marginRight: '8px',
    display: 'inline-block',
    width: '12px',
    height: '12px',
    minWidth: '12px',
    minHeight: '12px',
    borderRadius: '2px',
    flexShrink: '0',
    border: 'none',
    verticalAlign: 'middle',
    /* 🔥 降低亮度：使用 filter 降低饱和度和亮度 */
    filter: 'brightness(0.95) saturate(0.9)',
    opacity: '0.9'
  }
  
  if (backgroundColor) {
    style.backgroundColor = backgroundColor
  }
  
  return style
}

// 🔥 获取选项标签
function getOptionLabel(value: any): string {
  if (value === null || value === undefined) return ''
  const valueStr = String(value)
  const option = selectOptionsComputed.value.find((opt: any) => {
    const optValue = typeof opt === 'object' ? opt.value : opt
    return String(optValue) === valueStr
  })
  if (option) {
    return typeof option === 'object' ? option.label : option
  }
  return valueStr
}

function getRenderedOptionValue(option: any): any {
  return typeof option === 'object' ? option.value : option
}

function getRenderedOptionLabel(option: any): string {
  return typeof option === 'object' ? option.label : String(option)
}

function getRenderedOptionUserInfo(option: any): SearchOption['userInfo'] | null {
  return typeof option === 'object' && option?.userInfo ? option.userInfo : null
}

function getUserTagInitial(value: any): string {
  const userInfo = getUserInfoByValue(value)
  if (userInfo?.username) {
    return userInfo.username[0]?.toUpperCase() || 'U'
  }

  const label = getOptionLabel(value)
  return label?.[0]?.toUpperCase() || 'U'
}

// 🔥 移除标签
function handleRemoveTag(valueToRemove: any): void {
  if (Array.isArray(localValue.value)) {
    const newValues = localValue.value.filter(v => String(v) !== String(valueToRemove))
    localValue.value = newValues
    handleInput(newValues)
  }
}

// 🔥 根据值获取用户信息（用于标签显示）
const getUserInfoByValue = (value: any): any => {
  if (!value) return null
  if (!selectOptions.value || !Array.isArray(selectOptions.value)) return null
  const option = selectOptions.value.find((opt: any) => {
    if (!opt) return false
    const optValue = typeof opt === 'object' ? opt.value : opt
    return String(optValue) === String(value)
  })
  return option?.userInfo || null
}

/**
 * 提取下拉选项（兼容静态 options 和 remote 模式）
 * 
 * ⚠️ 优先级：静态 options > remote 动态选项
 * 静态 options 来自 widget.config.options（后端配置）
 * 动态选项来自 remote-method（用户搜索）
 */
const selectOptionsComputed = computed(() => {
  if (inputConfig.value.component !== SearchComponent.EL_SELECT) {
    return []
  }
  // 如果有静态 options，使用静态 options
  const staticOptions = inputConfig.value.props?.options
  if (staticOptions && staticOptions.length > 0) {
    return staticOptions
  }
  // 否则使用 remote 模式下的动态选项
  return selectOptions.value
})

// 🔥 处理 remote-method（如果有）
const handleRemoteMethod = async (query: string) => {
  if (inputConfig.value.component !== SearchComponent.EL_SELECT || !inputConfig.value.onRemoteMethod) {
    return
  }
  
  selectLoading.value = true
  try {
    const options = await inputConfig.value.onRemoteMethod(query)
    // 🔥 保留已选中的选项，避免丢失用户信息
    const currentValue = localValue.value
    const existingOptions = selectOptions.value || []
    
    // 合并新选项和已选中的选项
    const mergedOptions = [...(options || [])]
    
    // 如果有已选中的值，确保它们在选项中
    if (currentValue) {
      const valuesToCheck = Array.isArray(currentValue) ? currentValue : [currentValue]
      valuesToCheck.forEach((val: any) => {
        if (val && !mergedOptions.find((opt: any) => {
          const optValue = typeof opt === 'object' ? opt.value : opt
          return String(optValue) === String(val)
        })) {
          // 如果已选中的值不在新选项中，尝试从现有选项中查找
          const existingOption = existingOptions.find((opt: any) => {
            const optValue = typeof opt === 'object' ? opt.value : opt
            return String(optValue) === String(val)
          })
          if (existingOption) {
            mergedOptions.push(existingOption)
          }
        }
      })
    }
    
    selectOptions.value = mergedOptions
  } catch (error) {
    Logger.error('SearchInput', 'Remote method error', error)
    selectOptions.value = []
  } finally {
    selectLoading.value = false
  }
}

// 🔥 处理下拉框显示/隐藏事件
const handleVisibleChange = (visible: boolean) => {
  if (visible) {
    // 下拉框打开时，如果有已选中的值但选项为空，初始化选项
    const currentValue = localValue.value
    if (currentValue && (Array.isArray(currentValue) ? currentValue.length > 0 : true)) {
      const hasOptions = selectOptions.value && selectOptions.value.length > 0
      if (!hasOptions && inputConfig.value.onInitOptions) {
        // 如果选项为空，重新初始化
        nextTick(() => {
          initSelectedOptions()
        })
      }
    }
  }
}

// 🔥 初始化已选中的值对应的选项（用于 remote 模式回显）
const initSelectedOptions = async () => {
  if (inputConfig.value.component !== SearchComponent.EL_SELECT) {
    return
  }
  
  // 获取当前已选中的值
  const currentValue = localValue.value
  if (!currentValue) {
    return
  }
  
  // 🔥 优先使用 onInitOptions（如果存在），用于批量查询已选中值
  if (inputConfig.value.onInitOptions) {
    // 🔥 立即设置 loading 状态，并隐藏值，避免显示原始值
    selectLoading.value = true
    shouldShowValue.value = false
    // 🔥 在下一个 tick 执行，确保 el-select 已经渲染并显示 loading 状态
    await nextTick()
    try {
      // 🔥 处理值：提取所有值（数组或逗号分隔的字符串）
      let queryValues: any[] = []
      if (Array.isArray(currentValue) && currentValue.length > 0) {
        // 数组：使用所有值
        queryValues = currentValue
      } else if (typeof currentValue === 'string' && currentValue.includes(',')) {
        // 逗号分隔的字符串：分割为数组
        queryValues = currentValue.split(',').map(s => s.trim()).filter(s => s)
      } else if (currentValue !== null && currentValue !== undefined && currentValue !== '') {
        // 单个值：转换为数组
        queryValues = [currentValue]
      }
      
      if (queryValues.length === 0) {
        shouldShowValue.value = true
        return
      }
      
      // 🔥 如果只有一个值，使用单个值查询；如果有多个值，使用数组查询（by_values）
      const options = await inputConfig.value.onInitOptions(queryValues.length === 1 ? queryValues[0] : queryValues)
      selectOptions.value = options || []
      
      // 🔥 确保 localValue 的类型与选项中的 value 类型一致（用于 el-select 匹配）
      if (options && options.length > 0) {
        // 🔥 如果是多选模式，需要匹配所有值；如果是单选模式，只匹配第一个值
        const isMultiple = inputConfig.value.props?.multiple || false
        
        if (isMultiple) {
          // 多选模式：更新为匹配的值数组
          const matchedValues: any[] = []
          queryValues.forEach(queryVal => {
            const matchedOption = options.find((opt: any) => {
              const optValue = typeof opt === 'object' ? opt.value : opt
              return optValue === queryVal || String(optValue) === String(queryVal)
            })
            if (matchedOption) {
              const matchedValue = typeof matchedOption === 'object' ? matchedOption.value : matchedOption
              matchedValues.push(matchedValue)
            }
          })
          if (matchedValues.length > 0) {
            localValue.value = matchedValues
          }
        } else {
          // 单选模式：只匹配第一个值
          const queryValue = queryValues[0]
          const matchedOption = options.find((opt: any) => {
            const optValue = typeof opt === 'object' ? opt.value : opt
            return optValue === queryValue || String(optValue) === String(queryValue)
          })
          if (matchedOption) {
            const matchedValue = typeof matchedOption === 'object' ? matchedOption.value : matchedOption
            localValue.value = matchedValue
          }
        }
      }
      
      // 🔥 选项加载完成，允许显示值
      shouldShowValue.value = true
    } catch (error) {
      Logger.error('SearchInput', 'Init selected options error', error)
      selectOptions.value = []
      shouldShowValue.value = true
    } finally {
      selectLoading.value = false
    }
    return
  }
  
  // 🔥 如果没有 onInitOptions，回退到使用 onRemoteMethod（逐个查询）
  if (!inputConfig.value.onRemoteMethod) {
    return
  }
  
  // 如果是数组（multiple 模式），需要为每个值查询选项
  if (Array.isArray(currentValue) && currentValue.length > 0) {
    selectLoading.value = true
    try {
      // 为每个已选中的值查询对应的选项
      const optionPromises = currentValue.map(async (val: any) => {
        if (!val) return null
        const options = await inputConfig.value.onRemoteMethod(String(val))
        return options?.find((opt: any) => {
          const optValue = typeof opt === 'object' ? opt.value : opt
          return String(optValue) === String(val)
        })
      })
      
      const options = (await Promise.all(optionPromises)).filter(Boolean)
      selectOptions.value = options
    } catch (error) {
      Logger.error('SearchInput', 'Init selected options error', error)
    } finally {
      selectLoading.value = false
    }
  } else if (typeof currentValue === 'string' && currentValue.trim()) {
    // 单个值，直接查询
    selectLoading.value = true
    try {
      const options = await inputConfig.value.onRemoteMethod(String(currentValue))
      // 确保当前值在选项中
      const currentOption = options?.find((opt: any) => {
        const optValue = typeof opt === 'object' ? opt.value : opt
        return String(optValue) === String(currentValue)
      })
      if (currentOption) {
        selectOptions.value = [currentOption]
      } else if (options && options.length > 0) {
        selectOptions.value = options
      }
    } catch (error) {
      Logger.error('SearchInput', 'Init selected options error', error)
    } finally {
      selectLoading.value = false
    }
  }
}

/**
 * 🔥 通过 widgets-v2 获取搜索输入配置（重构版本）
 * 
 * 重构说明：
 * - 按照 v2 的设计思路重新实现
 * - 根据 field.widget.type 和 searchType 生成配置
 * - 兼容现有的 SearchInput 逻辑（配置对象方式）
 * 
 * 注意：v2 组件支持 mode="search"，但 SearchInput 需要配置对象
 * 所以这里创建一个适配层，根据 v2 的思路生成配置
 */
/**
 * 生成搜索组件配置
 * 🔥 使用工具函数统一生成配置，遵循单一职责原则
 */
const inputConfig = computed(() => {
  try {
    return createSearchComponentConfig(
      props.field,
      props.searchType,
      props.functionMethod,
      props.functionRouter
    )
  } catch (error) {
    // ✅ 使用 ErrorHandler 统一处理错误
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

/**
 * 处理单值输入（带防抖，实时同步URL）
 * 🔥 使用值规范化工具统一处理值转换
 */
const handleInputDebounced = debounce((value: any) => {
  const normalizedValue = normalizeSearchValue(value, {
    widgetType: props.field.widget?.type,
    searchType: props.searchType,
    field: props.field
  })
  
  emit('update:modelValue', normalizedValue)
}, SearchConfig.DEBOUNCE_DELAY)

const handleInput = (value: any) => {
  // 🔥 标记为内部更新，防止触发 watch
  isInternalUpdate.value = true
  localValue.value = value
  // 🔥 使用防抖，避免频繁更新URL
  handleInputDebounced(value)
  // 🔥 延迟重置标志，确保 watch 能正确判断（防抖时间 + 一个 tick）
  setTimeout(() => {
    isInternalUpdate.value = false
  }, SearchConfig.INTERNAL_UPDATE_DELAY)
}

const handleWidgetFieldUpdate = (value: any) => {
  const normalizedValue = normalizeSearchValue(value?.raw, {
    widgetType: searchWidgetType.value,
    searchType: props.searchType,
    field: widgetSearchField.value
  })

  emit('update:modelValue', normalizedValue)
}

// 处理清空事件（ElInput、ElSelect、ElDatePicker 等组件的 clearable）
const handleClear = () => {
  localValue.value = null
  dateRangeValue.value = null
  rangeValue.value = { min: undefined, max: undefined }
  // 🔥 清空时立即触发更新，不使用防抖
  emit('update:modelValue', null)
}

/**
 * 处理范围输入变化（NumberRangeInput 和 RangeInput）
 * 
 * ⚠️ 关键逻辑：
 * 1. 如果 min 和 max 都为空，传递 null（表示清空搜索条件）
 * 2. 否则传递 { min, max } 对象（用于构建 URL 参数 gte/lte）
 * 
 * 注意：空字符串会被转换为 undefined，避免传递无效值
 */
const handleRangeChange = () => {
  const min = rangeValue.value.min
  const max = rangeValue.value.max
  
  // 如果 min 和 max 都为空，传递 null 而不是空对象
  if ((min === undefined || min === null || min === '') && 
      (max === undefined || max === null || max === '')) {
    emit('update:modelValue', null)
  } else {
    emit('update:modelValue', {
      min: min === '' ? undefined : min,
      max: max === '' ? undefined : max
    })
  }
}

// 处理日期范围输入（ElDatePicker）
const handleDateRangeChange = (value: [number | string | null, number | string | null] | null) => {
  dateRangeValue.value = value
  // 🔥 ElDatePicker 返回数组格式 [start, end]，直接传递给父组件
  emit('update:modelValue', value)
}

// 监听外部值变化
watch(() => props.modelValue, (newValue: any, oldValue: any) => {
  if (shouldUseWidgetSearchRenderer.value) {
    return
  }

  // 🔥 如果是内部更新触发的，跳过处理
  if (isInternalUpdate.value) {
    return
  }
  
  // 🔥 如果值没有实际变化，跳过处理（避免循环更新）
  const newValueStr = JSON.stringify(newValue)
  const oldValueStr = JSON.stringify(oldValue)
  if (newValueStr === oldValueStr) {
    return
  }
  
  /**
   * 处理范围搜索（gte/lte）的值更新
   * 
   * ⚠️ 关键逻辑：每个 SearchInput 实例都有独立的 rangeValue
   * 只有当 newValue 是当前字段的范围值时，才更新 rangeValue
   * 这样可以避免多个 slider 字段之间的值互相影响
   * 
   * 判断条件：
   * 1. 字段支持范围搜索（searchType 包含 gte 和 lte）
   * 2. 字段是 slider 或使用范围输入组件
   * 3. newValue 是范围类型（数组或包含 min/max 的对象）
   */
  const isRangeSearch = props.searchType?.includes('gte') && props.searchType?.includes('lte')
  const isSliderWidget = props.field.widget?.type === WidgetType.SLIDER
  const isRangeInput = inputConfig.value.component === SearchComponent.NUMBER_RANGE_INPUT || 
                       inputConfig.value.component === SearchComponent.RANGE_INPUT
  
  if ((isSliderWidget || isRangeInput) && isRangeSearch) {
    // 数组格式（时间戳范围），用于 ElDatePicker
    if (Array.isArray(newValue)) {
      dateRangeValue.value = [
        newValue[0] || null,
        newValue[1] || null
      ]
      // 同时设置 rangeValue 用于其他范围输入组件
      rangeValue.value = {
        min: newValue[0] || undefined,
        max: newValue[1] || undefined
      }
    } 
    // 对象格式（数字范围），用于 slider 组件
    // ⚠️ 关键：必须检查 newValue 是否包含 min 或 max 属性
    // 这样可以避免其他字段的值（如字符串、数字等）影响当前字段
    else if (newValue && typeof newValue === 'object' && !Array.isArray(newValue) && ('min' in newValue || 'max' in newValue)) {
      rangeValue.value = {
        min: newValue.min !== undefined && newValue.min !== null ? newValue.min : undefined,
        max: newValue.max !== undefined && newValue.max !== null ? newValue.max : undefined
      }
      dateRangeValue.value = null
    } 
    // null 或 undefined：清空当前字段的值
    else if (newValue === null || newValue === undefined) {
      rangeValue.value = { min: undefined, max: undefined }
      dateRangeValue.value = null
    }
    // ⚠️ 如果 newValue 不是范围类型，不更新 rangeValue
    // 这样可以避免其他字段的值影响当前字段（例如：字符串、数字等）
  } else if (isRangeSearch && inputConfig.value.component === SearchComponent.EL_DATE_PICKER) {
    // 🔥 日期范围选择器
    if (Array.isArray(newValue)) {
      dateRangeValue.value = [
        newValue[0] || null,
        newValue[1] || null
      ]
    } else {
      dateRangeValue.value = null
    }
  } else {
    // 🔥 对于多选模式（multiple），确保值是数组格式
    // 注意：需要根据 searchType 判断，而不是依赖 inputConfig（因为 inputConfig 可能还没准备好）
    // 注意：多选组件只支持 contains 搜索类型
    const isMultiselectContains = props.field.widget?.type === WidgetType.MULTI_SELECT && props.searchType?.includes(SearchType.CONTAINS)
    
    if (isMultiselectContains) {
      // 多选组件搜索场景（只支持 contains）
      let newLocalValue: any[] = []
      if (newValue === null || newValue === undefined || newValue === '') {
        newLocalValue = []
      } else if (Array.isArray(newValue)) {
        newLocalValue = newValue
      } else if (typeof newValue === 'string') {
        // 🔥 如果是字符串，可能是逗号分隔的值（用于 contains 搜索），需要转换为数组供 el-select 显示
        // 多选组件在搜索时使用 contains 条件（FIND_IN_SET），后端存储是逗号分隔的字符串
        newLocalValue = newValue ? newValue.split(',').map(v => v.trim()).filter(v => v) : []
      } else {
        newLocalValue = [newValue]
      }
      
      // 🔥 只有当值真正变化时才更新，避免循环更新
      const currentValueStr = JSON.stringify(localValue.value)
      const newValueStr = JSON.stringify(newLocalValue)
      if (currentValueStr !== newValueStr) {
        localValue.value = newLocalValue
      }
    } else if (inputConfig.value.component === SearchComponent.EL_SELECT && inputConfig.value.props?.multiple) {
      // 其他多选场景（如 user 组件）
      if (newValue === null || newValue === undefined || newValue === '') {
        localValue.value = []
      } else if (Array.isArray(newValue)) {
        localValue.value = newValue
      } else if (typeof newValue === 'string') {
        // 字符串转换为数组
        localValue.value = parseCommaSeparatedString(newValue)
      } else {
        localValue.value = [newValue]
      }
    } else {
      localValue.value = newValue
    }
    
    // 🔥 当值变化时，如果是 remote 模式的 ElSelect，初始化已选中值的选项
    if (inputConfig.value.component === SearchComponent.EL_SELECT && 
        inputConfig.value.props?.remote && 
        localValue.value && 
        (Array.isArray(localValue.value) ? localValue.value.length > 0 : true)) {
      // 🔥 如果有 onInitOptions，先隐藏值，等选项加载完成后再显示
      if (inputConfig.value.onInitOptions) {
        shouldShowValue.value = false
      }
      // 延迟执行，确保 inputConfig 已更新
      nextTick(() => {
        initSelectedOptions()
      })
    } else {
      // 🔥 如果不是 remote 模式或没有值，允许显示值
      shouldShowValue.value = true
    }
  }
}, { immediate: true })

// 🔥 监听 inputConfig 变化，初始化已选中值的选项
watch(() => inputConfig.value, (newConfig, oldConfig) => {
  if (shouldUseWidgetSearchRenderer.value) {
    return
  }

  // 🔥 只有当 inputConfig 真正变化时才触发（避免初始化时重复调用）
  if (newConfig === oldConfig) {
    return
  }
  if (newConfig.component === SearchComponent.EL_SELECT && 
      newConfig.props?.remote && 
      localValue.value && 
      (Array.isArray(localValue.value) ? localValue.value.length > 0 : true)) {
    nextTick(() => {
      initSelectedOptions()
    })
  }
}, { immediate: false })

// 🔥 组件挂载时，如果已有值且是 remote 模式，立即触发初始化（避免先显示原始值）
onMounted(() => {
  if (shouldUseWidgetSearchRenderer.value) {
    return
  }

  if (inputConfig.value.component === SearchComponent.EL_SELECT && 
      inputConfig.value.props?.remote && 
      localValue.value && 
      (Array.isArray(localValue.value) ? localValue.value.length > 0 : true)) {
    // 🔥 如果有 onInitOptions，先隐藏值，等选项加载完成后再显示
    if (inputConfig.value.onInitOptions) {
      shouldShowValue.value = false
    }
    // 立即触发初始化，不等待 watch
    nextTick(() => {
      initSelectedOptions()
    })
  } else {
    // 🔥 如果不是 remote 模式或没有值，允许显示值
    shouldShowValue.value = true
  }
})
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
.user-select-search :deep(.el-select__tags) {
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

.search-summary-tag {
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
  color: #fff !important;
  font-weight: 500;
  /* 🔥 降低亮度：使用 filter 降低饱和度和亮度 */
  filter: brightness(0.95) saturate(0.9);
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
