<!--
  SelectWidget - 下拉选择组件
  🔥 统一架构组件
  
  功能：
  - 支持静态选项
  - 支持回调接口（OnSelectFuzzy）
  - 支持 displayInfo 显示
  - 支持聚合统计
-->

<template>
  <div class="select-widget">
    <!-- 编辑模式 -->
    <div v-if="mode === 'edit'" class="edit-select">
      <template v-if="shouldUseInlineSelect">
        <SelectWidgetInlineSelect
          v-model="internalValue"
          :options="options"
          :placeholder="selectPlaceholder"
          :disabled="!!widgetConfig.disabled"
          :clearable="allowInlineClear"
          :creatable="!!widgetConfig.creatable"
          :teleported="shouldTeleportPopper"
          :display-info-text="displayInfoText"
          :get-option-color="getOptionColor"
          :get-option-color-style="getOptionColorStyle"
          :get-option-display-info="getOptionDisplayInfo"
          @clear="handleClear"
        />
      </template>
      <template v-else>
        <SelectWidgetDialogTrigger
          :display-value="displayValue"
          :fallback-label="field.desc || `请选择${field.name}`"
          :display-info-text="displayInfoText"
          :has-value="hasCurrentValue"
          :show-clear="shouldShowDialogClear"
          @open="openDialog"
          @clear="handleClear"
        />
      </template>
      
      <!-- 🔥 显示 Statistics 统计信息（使用 FieldStatistics 组件） -->
      <!-- 🔥 在表格内部（depth > 0）时不显示，避免撑大表格单元格，统计信息会在表格下方统一显示 -->
      <FieldStatistics
        v-if="currentStatistics && Object.keys(currentStatistics).length > 0 && props.value?.raw && (props.depth || 0) === 0"
        :field="field"
        :value="props.value"
        :statistics="currentStatistics"
      />
      
    </div>
    
    <!-- 响应模式（只读） -->
    <SelectWidgetValueDisplay
      v-else-if="mode === 'response' || mode === 'table-cell' || mode === 'detail'"
      :mode="mode"
      :display-value="displayValue"
      :current-option-color="currentOptionColor"
    />
    
    <!-- 搜索模式 -->
    <div v-else-if="mode === 'search'" class="search-select">
      <template v-if="shouldUseInlineSelect">
        <SelectWidgetInlineSelect
          v-model="internalValue"
          :options="options"
          :placeholder="selectPlaceholder"
          :disabled="!!widgetConfig.disabled"
          :clearable="allowInlineClear"
          :creatable="!!widgetConfig.creatable"
          :search-mode="true"
          :teleported="shouldTeleportPopper"
          :get-option-color="getOptionColor"
          :get-option-color-style="getOptionColorStyle"
          :get-option-display-info="getOptionDisplayInfo"
          @clear="handleClear"
        />
      </template>
      <template v-else>
        <SearchSingleSelectDisplay
          :label="displayValue"
          :placeholder="`搜索${field.name}`"
          :has-value="hasCurrentValue"
          :show-clear="shouldShowDialogClear"
          :display-info-text="displayInfoText"
          @open="openDialog"
          @clear="handleClear"
        />
      </template>
    </div>

    <FuzzySearchDialog
      v-if="(mode === 'edit' || mode === 'search') && hasCallback"
      v-model="dialogVisible"
      :title="`选择${field.name}`"
      :placeholder="dialogPlaceholder"
      :suggestions="dialogSuggestions"
      :loading="loading"
      :is-multiselect="false"
      :get-item-color="getOptionColor"
      :append-to-body="shouldTeleportPopper"
      @search="handleDialogSearch"
      @select="handleDialogSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import FuzzySearchDialog, { type InputFuzzyItem } from './FuzzySearchDialog.vue'
import FieldStatistics from './FieldStatistics.vue'
import SelectWidgetDialogTrigger from './SelectWidgetDialogTrigger.vue'
import SelectWidgetInlineSelect from './SelectWidgetInlineSelect.vue'
import SelectWidgetValueDisplay from './SelectWidgetValueDisplay.vue'
import SearchSingleSelectDisplay from '@/architecture/presentation/shared/components/SearchSingleSelectDisplay.vue'
import { prdPreviewContextKey } from '@/architecture/presentation/components/prdPreviewContext'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/presentation/context/formRuntimeContext'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { selectFuzzy } from '@/architecture/presentation/context/api/function'
import { isFieldRequired } from '@/architecture/domain/utils/validationUtils'
import { Logger } from '@/architecture/shared/logger'
import { SelectFuzzyQueryType, getOptionSolidColor, normalizeOptionColor } from '@/architecture/domain/constants/select'
import { convertValueToType } from '@/architecture/presentation/widgets/utils/valueConverter'
import { convertFormDataToRequestByType } from '@/architecture/presentation/widgets/utils/typeConverter'
import { widgetInitializerRegistry } from '@/architecture/presentation/widgets/initializers/WidgetInitializerRegistry'
import { SelectWidgetInitializer } from '@/architecture/presentation/widgets/initializers/SelectWidgetInitializer'
import { getWidgetOptionColors } from '@/architecture/domain/utils/widgetOptionColors'
import type { SelectOptionConfig, SelectWidgetConfig } from '@/architecture/domain/types/widget-configs'
import type { SelectOptionItem, SelectOptionValue } from './selectWidgetTypes'

type SelectValue = unknown
type FormRendererLike = NonNullable<WidgetComponentProps['formRenderer']>

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function hasValue(value: unknown): boolean {
  return value !== null && value !== undefined && value !== ''
}

function toOptionValue(value: unknown): SelectOptionValue {
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value
  }
  if (isRecord(value)) {
    return value
  }
  return String(value ?? '')
}

function toDisplayInfoRecord(value: unknown): Record<string, unknown> | undefined {
  return isRecord(value) ? value : undefined
}

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()
const prdPreviewContext = inject(prdPreviewContextKey, null)
const shouldTeleportPopper = computed(() => !prdPreviewContext?.interactive)

const formDataStore = useFormDataStore()
const callbackMethod = computed(() => props.formRenderer?.getFunctionMethod?.() || props.functionMethod || 'POST')
const callbackRouter = computed(() => props.formRenderer?.getFunctionRouter?.() || props.functionRouter || '')

// 获取配置（带类型）
const widgetConfig = computed(() => {
  return (props.field.widget?.config || {}) as SelectWidgetConfig
})

function normalizeOption(option: string | SelectOptionConfig): SelectOptionItem {
  if (typeof option === 'string') {
    return {
      label: option,
      value: option,
    }
  }

  return {
    label: option.label,
    value: toOptionValue(option.value),
    disabled: option.disabled,
    displayInfo: option.displayInfo ?? option.display_info,
    icon: option.icon,
  }
}

const options = ref<SelectOptionItem[]>([])

/**
 * 🔥 静态选项（从配置中获取，用于颜色索引对齐）
 * options_colors 数组与静态选项的索引对齐
 */
const staticOptions = computed<SelectOptionItem[]>(() => {
  const configOptions = widgetConfig.value.options || []
  if (Array.isArray(configOptions)) {
    return configOptions.map(normalizeOption)
  }
  return []
})

/**
 * 🔥 选项颜色配置
 * 
 * 只支持不带 # 的 6 位十六进制 RRGGBB，例如 FF5722、4CAF50。
 * 
 * options_colors 数组与 staticOptions 数组的索引对齐，通过索引获取对应选项的颜色
 */
const optionColors = computed(() => {
  return getWidgetOptionColors(widgetConfig.value)
})

/**
 * 获取当前选中值的颜色
 * 通过查找当前值在 options 中的索引，从 optionColors 数组中获取对应颜色
 * options_colors 数组与 options 数组的索引对齐
 */
const currentOptionColor = computed(() => {
  const rawValue = props.value?.raw
  if (rawValue === null || rawValue === undefined || rawValue === '') {
    return null
  }

  return getOptionColor(rawValue)
})

/**
 * 🔥 获取选项的颜色（用于下拉选项显示）
 * 注意：options_colors 数组与 staticOptions 数组的索引对齐
 * 即使 options 可能包含动态选项，颜色配置仍然基于 staticOptions 的索引
 */
function getOptionColor(value: SelectValue): string | null {
  const valueStr = String(value)
  // 🔥 在 staticOptions 中查找索引（因为 options_colors 与 staticOptions 对齐）
  const optionIndex = staticOptions.value.findIndex((opt) => String(opt.value) === valueStr)
  if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
    return normalizeOptionColor(optionColors.value[optionIndex]) ?? null
  }
  return null
}

/**
 * 🔥 获取选项的颜色样式对象（用于 span 的 style 绑定）
 */
function getOptionColorStyle(value: SelectValue): Record<string, string> {
  const color = getOptionColor(value)
  if (!color) return {}
  const backgroundColor = getOptionSolidColor(color)
  
  // 🔥 确保 backgroundColor 有值，并且使用 !important 确保样式生效
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

function formatDisplayInfo(info: unknown): string {
  if (!info) {
    return ''
  }

  const normalizedInfo = Array.isArray(info) ? info[0] : info
  if (!normalizedInfo) {
    return ''
  }

  if (!isRecord(normalizedInfo)) {
    return String(normalizedInfo)
  }

  const infoItems: string[] = []
  Object.entries(normalizedInfo).forEach(([key, val]) => {
    if (val !== null && val !== undefined && val !== '') {
      infoItems.push(`${key}: ${val}`)
    }
  })

  if (infoItems.length === 0) {
    return ''
  }

  if (infoItems.length > 5) {
    return infoItems.slice(0, 5).join(' | ') + ' ...'
  }

  return infoItems.join(' | ')
}

function getOptionDisplayInfo(option: SelectOptionItem): string {
  return formatDisplayInfo(option.displayInfo)
}

// 加载状态
const loading = ref(false)

// 是否有回调接口
const hasCallback = computed(() => {
  return props.field.callbacks?.includes('OnSelectFuzzy') || false
})

const shouldUseInlineSelect = computed(() => !hasCallback.value)

const allowInlineClear = computed(() => {
  return props.mode === 'search' || !isFieldRequired(props.field)
})

const selectPlaceholder = computed(() => {
  if (props.mode === 'search') {
    return widgetConfig.value.placeholder || `搜索${props.field.name}`
  }

  return widgetConfig.value.placeholder || props.field.desc || `请选择${props.field.name}`
})

const dialogPlaceholder = computed(() => {
  if (props.mode === 'search') {
    return '请输入搜索关键词'
  }
  return props.field.desc || '请输入搜索关键词'
})

// 对话框相关状态
const dialogVisible = ref(false)
const dialogSuggestions = ref<InputFuzzyItem[]>([])

// 🔥 SelectWidget 是纯单选组件，不需要多选相关逻辑
// 注意：SelectWidget 始终支持搜索（通过 FuzzySearchDialog），不需要 filterable 配置

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      return props.value?.raw ?? null
    }
    return null
  },
  set: (newValue: SelectValue) => {
    if (props.mode === 'edit' || props.mode === 'search') {
      if (newValue === null || newValue === undefined || newValue === '') {
        handleClear()
        return
      }

      const selectedOption = options.value.find(opt => opt.value === newValue || String(opt.value) === String(newValue))
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        newValue,
        selectedOption?.label || String(newValue),
        {
          displayInfo: selectedOption?.displayInfo
        }
      )
      
      if (props.mode === 'edit') {
        formDataStore.setValue(props.fieldPath, newFieldValue)
      }
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 🔥 详情模式下通过回调获取的显示值（用于存储）
const detailDisplayValue = ref<string | null>(null)

function hasSelectedValue(raw: unknown): boolean {
  return hasValue(raw)
}

function findSelectedOption(raw: unknown): SelectOptionItem | undefined {
  if (!hasSelectedValue(raw)) {
    return undefined
  }

  return options.value.find((opt) => {
    return opt.value === raw || String(opt.value) === String(raw)
  })
}

const hasCurrentValue = computed(() => {
  return hasSelectedValue(props.value?.raw)
})

const shouldShowDialogClear = computed(() => {
  return hasCurrentValue.value && (props.mode === 'search' || !isFieldRequired(props.field))
})

// 获取 display_info 的显示文本
const displayInfoText = computed(() => {
  const value = props.value
  if (!value || value.raw === null || value.raw === undefined || value.raw === '') {
    return ''
  }
  
  // 🔥 优先从 meta.displayInfo 获取（这是保存的值）
  if (value.meta?.displayInfo) {
    const metaDisplayInfo = formatDisplayInfo(value.meta.displayInfo)
    if (metaDisplayInfo) {
      return metaDisplayInfo
    }
  }
  
  // 🔥 如果 meta 中没有，从 options 中查找
  const selectedOption = options.value.find((opt) => {
    return opt.value === value.raw || String(opt.value) === String(value.raw)
  })
  
  if (selectedOption?.displayInfo) {
    const optionDisplayInfo = formatDisplayInfo(selectedOption.displayInfo)
    if (optionDisplayInfo) {
      return optionDisplayInfo
    }
  }
  
  return ''
})

// 显示值
const displayValue = computed(() => {
  const value = props.value
  if (!value) {
    return '-'
  }

  const raw = value.raw
  if (!hasSelectedValue(raw)) {
    return '-'
  }

  const matchedOption = findSelectedOption(raw)
  
  // 🔥 在详情模式下，优先使用 detailDisplayValue（通过回调获取的）
  // 如果 value.display 为空或等于 raw（说明没有有意义的显示值），则使用 detailDisplayValue
  if (props.mode === 'detail') {
    // 如果 detailDisplayValue 有值（通过回调获取的），优先使用
    if (detailDisplayValue.value) {
      return detailDisplayValue.value
    }
    // 如果 value.display 为空或等于 raw，说明没有有意义的显示值，尝试从 options 中查找
    if (!value.display || value.display === '' || String(value.display) === String(raw)) {
      if (matchedOption) {
        return matchedOption.label
      }
      // 如果找不到匹配的选项，返回 raw 值（作为后备）
      return String(raw)
    }
    // 如果 value.display 有值且不等于 raw，使用 value.display
    if (value.display && String(value.display) !== String(raw)) {
      return value.display
    }
    // 如果 value.display 为空，返回 raw 值
    return String(raw)
  }
  
  // 🔥 非详情模式下，优先使用有意义的 display；如果 display 只是 raw，则回退到 option.label
  if (value.display && String(value.display) !== String(raw)) {
    return value.display
  }
  if (matchedOption?.label) {
    return matchedOption.label
  }
  
  return String(raw)
})

// 初始化选项
function initOptions(): void {
  if (Array.isArray(widgetConfig.value.options)) {
    options.value = [...staticOptions.value]
  }
  
  // 🔥 如果有回调接口且有初始值，触发一次搜索（包括详情模式）
  // 详情模式下也需要触发回调，通过 by_value 查询来获取选项标签
  // ⚠️ 注意：详情模式下由 watch 处理，这里只处理非详情模式
  if (hasCallback.value && hasSelectedValue(props.value?.raw) && props.mode !== 'detail' && callbackRouter.value) {
    handleSearch(props.value.raw, true) // by_value
  }
  
  // 🔥 详情模式下，如果已经有 formRenderer，由 watch 处理
  // 如果没有 formRenderer，等待 watch 检测到 formRenderer 后再触发
}

// 打开对话框
async function openDialog(): Promise<void> {
  dialogVisible.value = true
  
  // 如果有回调接口
  if (hasCallback.value) {
    if (!callbackRouter.value) {
      Logger.warn('[SelectWidget]', 'openDialog: functionRouter 不存在，无法触发回调', {
        fieldCode: props.field.code
      })
      return
    }
    
    // 🔥 如果已有值，通过 by_value 搜索获取对应的选项和 label
    if (props.value?.raw !== null && props.value?.raw !== undefined && props.value?.raw !== '') {
      await handleSearch(props.value.raw, true) // by_value 搜索
    } else {
      // 没有值，触发空搜索加载初始选项
      await handleDialogSearch('')
    }
  } else {
    // 静态选项，直接使用
    dialogSuggestions.value = options.value.map((opt) => ({
      label: opt.label,
      value: opt.value,
      displayInfo: toDisplayInfoRecord(opt.displayInfo),
      display_info: toDisplayInfoRecord(opt.displayInfo), // 同时提供两种格式，确保兼容
      icon: opt.icon
    }))
  }
}

// 处理对话框搜索
async function handleDialogSearch(keyword: string): Promise<void> {
  if (hasCallback.value) {
    if (!callbackRouter.value) {
      Logger.warn('[SelectWidget]', 'handleDialogSearch: functionRouter 不存在，无法触发回调', {
        fieldCode: props.field.code,
        keyword
      })
      return
    }
    
    await handleSearch(keyword, false)
    
    // 更新对话框建议列表
    dialogSuggestions.value = options.value.map((opt) => ({
      label: opt.label,
      value: opt.value,
      displayInfo: toDisplayInfoRecord(opt.displayInfo),
      display_info: toDisplayInfoRecord(opt.displayInfo), // 同时提供两种格式，确保兼容
      icon: opt.icon
    }))
  } else {
    // 静态选项，本地过滤
    const filtered = staticOptions.value.filter((opt) => {
      return opt.label.toLowerCase().includes(keyword.toLowerCase())
    })
    dialogSuggestions.value = filtered.map((opt) => {
      return {
        label: opt.label,
        value: opt.value,
        displayInfo: toDisplayInfoRecord(opt.displayInfo),
        display_info: toDisplayInfoRecord(opt.displayInfo), // 同时提供两种格式，确保兼容
        icon: opt.icon
      }
    })
  }
}

// 处理对话框选择（单选模式）
function handleDialogSelect(item: InputFuzzyItem): void {
  // 🔥 更新 options，确保选择的项的 displayInfo 被保存
  const existingOption = options.value.find((opt) => String(opt.value) === String(item.value))
  if (!existingOption) {
    // 如果 options 中没有，添加进去
    options.value.push({
      label: item.label || String(item.value),
      value: toOptionValue(item.value),
      displayInfo: item.displayInfo
    })
  } else if (item.displayInfo && !existingOption.displayInfo) {
    // 如果 options 中有但没有 displayInfo，更新它
    existingOption.displayInfo = item.displayInfo
  }
  
  const selectedOption = options.value.find((opt) => String(opt.value) === String(item.value))
  // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
  const newFieldValue = createFieldValue(
    props.field,
    item.value,
    item.label || selectedOption?.label || String(item.value),
    {
    displayInfo: item.displayInfo || item.display_info || selectedOption?.displayInfo,
      statistics: currentStatistics.value  // 🔥 保存 statistics 配置
    }
  )
  
  // 🔥 确保值被正确保存到 formDataStore
  formDataStore.setValue(props.fieldPath, newFieldValue)
  
  // 🔥 同时触发 emit，确保 FormView 也能收到更新
  emit('update:modelValue', newFieldValue)
  
  // 🔥 调试日志：检查值是否正确保存
  const savedValue = formDataStore.getValue(props.fieldPath)
  if (savedValue?.raw !== item.value) {
    Logger.warn('SelectWidget', `值保存失败: fieldPath=${props.fieldPath}, expected=${item.value}, actual=${savedValue?.raw}`)
  }
  
  // 🔥 关闭对话框
  dialogVisible.value = false
}

// 🔥 处理清除值
function handleClear(): void {
  // 创建空值
  const emptyFieldValue = createFieldValue(
    props.field,
    null,
    '',
    {}
  )
  
  // 更新 formDataStore
  if (props.mode === 'edit') {
    formDataStore.setValue(props.fieldPath, emptyFieldValue)
  }
  
  // 触发 emit
  emit('update:modelValue', emptyFieldValue)
}


// 处理搜索
async function handleSearch(query: string | number | SelectValue, isByValue: boolean): Promise<void> {
  if (!hasCallback.value || !callbackRouter.value) {
    return
  }
  
  const method = callbackMethod.value
  const router = callbackRouter.value
  
  if (!router) {
    Logger.warn('[SelectWidget]', '无法获取函数路由，取消回调', {
      fieldCode: props.field.code,
      router
    })
    return
  }
  
  
  loading.value = true
  
  try {
    // 🔥 类型转换：根据 value_type 将字符串转换为正确的类型
    const valueType = props.field.data?.type || 'string'
    let convertedValue: unknown = query
    
    // 🔥 如果 query 已经是数字类型，不需要转换
    if (isByValue && typeof query === 'string' && valueType !== 'string') {
      // 使用统一的类型转换工具函数
      convertedValue = convertValueToType(query, valueType, 'SelectWidget')
    }
    
    // 🔥 获取提交数据并根据字段类型进行转换
    // 使用统一的类型转换函数，确保所有字段都根据 field.data.type 正确转换
    const submitData = props.formRenderer?.getSubmitData?.() || {}
    const functionDetail = props.formRenderer?.getFunctionDetail?.()
    const requestData = convertFormDataToRequestByType(submitData, functionDetail || {})
    
    const requestBody = {
      code: props.field.code,
      type: isByValue ? SelectFuzzyQueryType.BY_VALUE : SelectFuzzyQueryType.BY_KEYWORD,
      value: convertedValue, // 🔥 使用转换后的值
      request: requestData,  // 🔥 使用转换后的请求数据
      value_type: valueType
    }
    
    const response = await selectFuzzy(method, router, requestBody)
    
    if (response.error_msg) {
      ElMessage.error(response.error_msg)
      options.value = []
      return
    }
    
    // 🔥 保存 statistics 配置（用于聚合计算）
    if (response.statistics && typeof response.statistics === 'object') {
      currentStatistics.value = Object.fromEntries(
        Object.entries(response.statistics).map(([key, value]) => [key, String(value)])
      )
      // 如果当前已有选中值，立即更新 meta.statistics
      if (hasSelectedValue(props.value?.raw)) {
        // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
        const newFieldValue = createFieldValue(
          props.field,
          props.value.raw,
          props.value.display || String(props.value.raw),
          {
            ...props.value.meta,
            statistics: currentStatistics.value
          }
        )
        formDataStore.setValue(props.fieldPath, newFieldValue)
      }
    }
    
    // 🔥 SelectWidget 是单选组件，不需要处理 max_selections
    // max_selections 只在 MultiSelectWidget（多选组件）里有意义
    
    if (response.items && Array.isArray(response.items)) {
      // OnSelectFuzzy 回调约定：
      // - label: 候选项主文案，搜索态和选中态优先展示它
      // - value: 实际提交值
      // - display_info/displayInfo: 次级结构化信息，用于弹窗候选项补充说明和选中后的 displayInfo 展示
      options.value = response.items.map((item) => ({
        label: item.label || String(item.value),
        value: toOptionValue(item.value),
        disabled: false,
        displayInfo: item.display_info || item.displayInfo
      }))
      
      // 🔥 如果是通过 by_value 查询，找到匹配的选项并更新显示值
      if (isByValue && hasSelectedValue(props.value?.raw)) {
        const matchedOption = options.value.find((opt) => {
          // 支持多种类型比较
          return opt.value === props.value.raw || String(opt.value) === String(props.value.raw)
        })
        if (matchedOption) {
          // 🔥 在详情模式下，更新 detailDisplayValue
          if (props.mode === 'detail') {
            detailDisplayValue.value = matchedOption.label
          }
          // 🔥 在编辑/搜索模式下，如果 value.display 为空或等于 raw，更新 display 值。
          // 搜索栏从 URL 恢复时只有 raw，必须把回显结果 emit 给父层，否则下一次状态同步会退回 raw value。
          if (
            (props.mode === 'edit' || props.mode === 'search') &&
            (!props.value.display || String(props.value.display) === String(props.value.raw))
          ) {
            // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
            const newFieldValue = createFieldValue(
              props.field,
              props.value.raw,
              matchedOption.label,
              {
                ...props.value.meta,
                displayInfo: matchedOption.displayInfo
              }
            )
            formDataStore.setValue(props.fieldPath, newFieldValue)
            emit('update:modelValue', newFieldValue)
          }
        }
      }
    } else {
      options.value = []
    }
  } catch (error) {
    Logger.error('SelectWidget', '回调失败', error)
    ElMessage.error(error instanceof Error ? error.message : '查询失败')
    options.value = []
  } finally {
    loading.value = false
  }
}

// 当前统计信息（从回调接口获取）
const currentStatistics = ref<Record<string, string>>({})

// 初始化
onMounted(() => {
  initOptions()
  
  // 🔥 注册 SelectWidget 初始化器（组件自治）
  // 只在有 OnSelectFuzzy 回调时才注册
  if (hasCallback.value && props.mode === 'edit') {
    widgetInitializerRegistry.register('select', new SelectWidgetInitializer())
    Logger.debug('[SelectWidget]', '注册初始化器', {
      fieldCode: props.field.code,
      widgetType: 'select'
    })
  }
  
  // 🔥 统一使用 SelectWidgetInitializer 处理初始化
  // 🔥 SelectWidgetInitializer 在 useFunctionParamInitialization 的 triggerWidgetInitialization 中调用
  // 🔥 这样可以避免重复调用回显接口，并且保证初始化逻辑的统一性
  // 🔥 如果未来需要保留这个监听器，需要添加防重复调用的机制
})

// 🔥 组件卸载时取消注册初始化器
onUnmounted(() => {
  // 🔥 取消注册初始化器（防止内存泄漏）
  if (hasCallback.value && props.mode === 'edit') {
    widgetInitializerRegistry.unregister('select')
    Logger.debug('[SelectWidget]', 'onUnmounted - 取消注册初始化器', {
      fieldCode: props.field.code,
      widgetType: 'select'
    })
  }
})

// 🔥 监听 value 和 formRenderer 变化，如果值变化了，重新触发回调获取标签
// 使用一个标志来防止重复调用
const isSearching = ref(false)
const lastSearchedValue = ref<SelectValue>(null)
const lastSearchedRouter = ref<string | null>(null) // 🔥 记录上次搜索使用的 router
const lastSearchedFunctionId = ref<number | null>(null) // 🔥 记录上次搜索使用的函数 ID

// 🔥 优化：控制日志输出（默认关闭，调试时可以改为 true）
const ENABLE_DETAILED_LOGS = false

// 🔥 触发搜索的辅助函数（避免重复代码）
const triggerSearchIfNeeded = (rawValue: SelectValue, formRenderer: FormRendererLike | undefined, mode: string) => {
  // 🔥 优化：减少日志输出，只在关键节点输出
  const shouldLog = ENABLE_DETAILED_LOGS
  
  if (shouldLog) {
    Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 开始', {
      fieldCode: props.field.code,
      rawValue,
      hasCallback: hasCallback.value,
      formRenderer: !!formRenderer,
    })
  }
  
  // 🔥 移除 keep-alive 后，组件每次都会重新挂载，不需要检查激活状态
  
  if (!hasCallback.value || !callbackRouter.value) {
    if (shouldLog) {
      Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 跳过：无回调或无 router', {
        fieldCode: props.field.code,
        hasCallback: hasCallback.value,
        formRenderer: !!formRenderer
      })
    }
    return false
  }
  
  const currentRouter = formRenderer?.getFunctionRouter?.() || callbackRouter.value
  if (!currentRouter) {
    if (shouldLog) {
      Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 跳过：无 currentRouter', {
        fieldCode: props.field.code
      })
    }
    return false
  }
  
  // 🔥 获取当前函数 ID（用于防重复调用）
  const currentFunctionId = formRenderer?.getFunctionDetail?.()?.id || null
  
  if (shouldLog) {
    Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 当前状态', {
      fieldCode: props.field.code,
      rawValue,
      currentRouter,
      currentFunctionId,
      lastSearchedValue: lastSearchedValue.value,
      lastSearchedRouter: lastSearchedRouter.value,
      lastSearchedFunctionId: lastSearchedFunctionId.value,
      isSearching: isSearching.value
    })
  }
  
  // 🔥 如果 router 或 functionId 变化了，重置搜索状态
  if (currentRouter !== lastSearchedRouter.value || currentFunctionId !== lastSearchedFunctionId.value) {
    if (shouldLog) {
      Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 重置搜索状态（router 或 functionId 变化）', {
        fieldCode: props.field.code,
        currentRouter,
        lastSearchedRouter: lastSearchedRouter.value,
        currentFunctionId,
        lastSearchedFunctionId: lastSearchedFunctionId.value
      })
    }
    lastSearchedValue.value = null
    lastSearchedRouter.value = currentRouter
    lastSearchedFunctionId.value = currentFunctionId
  }
  
  // 🔥 检查是否需要触发搜索
  // 如果函数 ID 相同、值相同、router 相同，说明已经搜索过，不需要重复调用
  const shouldTrigger = 
    rawValue !== null && 
    rawValue !== undefined && 
    !isSearching.value &&
    // 🔥 关键：如果值变化了，或者 router 变化了，或者 functionId 变化了，或者还没有搜索过这个值，就触发
    (rawValue !== lastSearchedValue.value || 
     currentRouter !== lastSearchedRouter.value || 
     currentFunctionId !== lastSearchedFunctionId.value)
  
  if (shouldLog) {
    Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 判断结果', {
      fieldCode: props.field.code,
      shouldTrigger,
      reasons: {
        hasValue: rawValue !== null && rawValue !== undefined,
        notSearching: !isSearching.value,
        valueChanged: rawValue !== lastSearchedValue.value,
        routerChanged: currentRouter !== lastSearchedRouter.value,
        functionIdChanged: currentFunctionId !== lastSearchedFunctionId.value
      }
    })
  }
  
  if (shouldTrigger) {
    if (shouldLog) {
      Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded ✅ 触发搜索', {
        fieldCode: props.field.code,
        rawValue,
        currentRouter,
        currentFunctionId
      })
    }
    isSearching.value = true
    lastSearchedValue.value = rawValue
    lastSearchedRouter.value = currentRouter
    lastSearchedFunctionId.value = currentFunctionId
    // 重置 detailDisplayValue（仅详情模式需要）
    if (mode === 'detail') {
      detailDisplayValue.value = null
    }
    // 🔥 通过 by_value 搜索获取对应的 label 和 displayInfo
    handleSearch(rawValue, true).finally(() => {
      if (shouldLog) {
        Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 搜索完成', {
          fieldCode: props.field.code,
          rawValue,
          currentFunctionId
        })
      }
      isSearching.value = false
    })
    return true
  }
  
  if (shouldLog) {
    Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded ❌ 跳过搜索（防重复）', {
      fieldCode: props.field.code,
      rawValue,
      lastSearchedValue: lastSearchedValue.value,
      currentRouter,
      lastSearchedRouter: lastSearchedRouter.value,
      currentFunctionId,
      lastSearchedFunctionId: lastSearchedFunctionId.value
    })
  }
  return false
}


// 🔥 保留一个简单的 watch 来处理值变化（仅在 formRenderer 已准备好且组件激活时）
// 🔥 优化：只在有回调且不是 table-cell 模式时才监听值变化
// 🔥 注意：如果 value.display 已经存在且不等于 raw，说明已经通过 SelectWidgetInitializer 初始化过了，不需要再触发
watch(
  () => props.value?.raw,
  (newRaw, oldRaw) => {
    // 🔥 减少日志输出：只在有回调且值真正变化时输出日志
    if (hasCallback.value && props.mode !== 'table-cell' && newRaw !== oldRaw) {
      Logger.debug('[SelectWidget]', 'watch props.value?.raw 触发', {
        fieldCode: props.field.code,
        newRaw,
        oldRaw,
        formRenderer: !!props.formRenderer,
        hasDisplay: !!props.value?.display,
        display: props.value?.display,
        displayEqualsRaw: props.value?.display && String(props.value.display) === String(newRaw)
      })
    }
    
    // 🔥 如果 value.display 已经存在且不等于 raw，说明已经通过 SelectWidgetInitializer 初始化过了，不需要再触发
    // 这样可以避免在初始化时重复调用回显接口
    if (props.value?.display && 
        String(props.value.display) !== String(newRaw) && 
        props.value.display !== '') {
      Logger.debug('[SelectWidget]', 'watch props.value?.raw 跳过：已有 display 值（已初始化）', {
        fieldCode: props.field.code,
        newRaw,
        display: props.value.display
      })
      return  // 已经初始化过了，不需要再触发
    }
    
    // 🔥 如果 oldRaw 是 null 且 newRaw 有值，说明是初始化阶段（从 URL 或默认值恢复）
    // 此时应该等待 SelectWidgetInitializer 处理，而不是立即触发搜索
    if (oldRaw === null && newRaw !== null && newRaw !== undefined) {
      // 如果已经搜索过这个值，不需要再触发
      if (lastSearchedValue.value === newRaw) {
        Logger.debug('[SelectWidget]', 'watch props.value?.raw 跳过：初始化阶段且已搜索过', {
          fieldCode: props.field.code,
          newRaw,
          lastSearchedValue: lastSearchedValue.value
        })
        return
      }
      // 初始化阶段，等待 SelectWidgetInitializer 统一处理
      Logger.debug('[SelectWidget]', 'watch props.value?.raw 跳过：初始化阶段，等待 SelectWidgetInitializer 处理', {
        fieldCode: props.field.code,
        newRaw,
        oldRaw
      })
      return
    }
    
    // 只在 formRenderer 已准备好且值真正变化且有回调时触发（用户手动修改值的场景）
    // 🔥 移除 keep-alive 后，组件每次都会重新挂载，不需要检查激活状态
    if (hasCallback.value && 
        props.mode !== 'table-cell' && 
        callbackRouter.value && 
        newRaw !== null && 
        newRaw !== undefined && 
        newRaw !== oldRaw) {
      triggerSearchIfNeeded(newRaw, props.formRenderer || undefined, props.mode)
    }
  }
)
</script>

<style scoped lang="scss">
.select-widget {
  width: 100%;
}

.edit-select,
.search-select {
  width: 100%;
  position: relative;
}
</style>
