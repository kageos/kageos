<!--
  SelectWidget - 下拉选择组件
  🔥 完全新增，不依赖旧代码
  
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
      <!-- 显示当前选中值和 display_info -->
      <div class="select-container" @click="openDialog">
        <div class="select-content">
          <div class="select-main">
            <span class="select-label">{{ displayValue || (field.desc || `请选择${field.name}`) }}</span>
            <!-- 🔥 清除按钮（当有值且不是必填字段时显示） -->
            <el-icon
              v-if="props.value?.raw != null && props.value?.raw !== '' && !isFieldRequired(field)"
              class="clear-icon"
              @click.stop="handleClear"
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
      
      <!-- 🔥 显示 Statistics 统计信息（使用 FieldStatistics 组件） -->
      <!-- 🔥 在表格内部（depth > 0）时不显示，避免撑大表格单元格，统计信息会在表格下方统一显示 -->
      <FieldStatistics
        v-if="currentStatistics && Object.keys(currentStatistics).length > 0 && props.value?.raw && (props.depth || 0) === 0"
        :field="field"
        :value="props.value"
        :statistics="currentStatistics"
      />
      
      <!-- 模糊搜索对话框（单选模式） -->
      <FuzzySearchDialog
        v-model="dialogVisible"
        :title="`选择${field.name}`"
        :placeholder="field.desc || `请输入搜索关键词`"
        :suggestions="dialogSuggestions"
        :loading="loading"
        :is-multiselect="false"
        :get-item-color="getOptionColor"
        @search="handleDialogSearch"
        @select="handleDialogSelect"
      />
    </div>
    
    <!-- 响应模式（只读） -->
    <span v-else-if="mode === 'response'" class="response-value">
      {{ displayValue }}
    </span>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-value">
      <el-tag
        v-if="currentOptionColor"
        :type="isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        :color="!isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        size="small"
        class="select-tag select-tag-outline"
      >
        {{ displayValue }}
      </el-tag>
      <span v-else>{{ displayValue }}</span>
    </div>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <el-tag
        v-if="currentOptionColor"
        :type="isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        :color="!isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        class="select-tag select-tag-outline"
      >
        {{ displayValue }}
      </el-tag>
      <span v-else class="detail-content">{{ displayValue }}</span>
    </div>
    
    <!-- 搜索模式 -->
    <div v-else-if="mode === 'search'" class="search-select">
      <div class="select-container" @click="openDialog">
        <div class="select-content">
          <div class="select-main">
            <span class="select-label">{{ displayValue || `搜索${field.name}` }}</span>
            <!-- 🔥 清除按钮（当有值且不是必填字段时显示） -->
            <el-icon
              v-if="props.value?.raw != null && props.value?.raw !== '' && !isFieldRequired(field)"
              class="clear-icon"
              @click.stop="handleClear"
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
      
      <!-- 🔥 显示 Statistics 统计信息（使用 FieldStatistics 组件） -->
      <!-- 🔥 在表格内部（depth > 0）时不显示，避免撑大表格单元格，统计信息会在表格下方统一显示 -->
      <FieldStatistics
        v-if="currentStatistics && Object.keys(currentStatistics).length > 0 && props.value?.raw && (props.depth || 0) === 0"
        :field="field"
        :value="props.value"
        :statistics="currentStatistics"
      />
      
      <!-- 模糊搜索对话框（单选模式） -->
      <FuzzySearchDialog
        v-model="dialogVisible"
        :title="`选择${field.name}`"
        :placeholder="`请输入搜索关键词`"
        :suggestions="dialogSuggestions"
        :loading="loading"
        :is-multiselect="false"
        :get-item-color="getOptionColor"
        @search="handleDialogSearch"
        @select="handleDialogSelect"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick, withDefaults } from 'vue'
import { ElInput, ElMessage, ElTag, ElIcon } from 'element-plus'
import { ArrowDown, CircleClose } from '@element-plus/icons-vue'
import FuzzySearchDialog from './FuzzySearchDialog.vue'
import FieldStatistics from './FieldStatistics.vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { selectFuzzy } from '@/api/function'
import { isFieldRequired } from '@/core/utils/validationUtils'
import { Logger } from '@/core/utils/logger'
import { SelectFuzzyQueryType, isStandardColor, getStandardColorCSSVar, type StandardColorType } from '@/core/constants/select'
import { convertValueToType } from '@/architecture/presentation/widgets/utils/valueConverter'
import { convertFormDataToRequestByType } from '@/architecture/presentation/widgets/utils/typeConverter'
// 🔥 使用事件驱动：监听表单初始化完成事件，统一处理 OnSelectFuzzy 字段
import { eventBus, FormEvent } from '@/architecture/infrastructure/eventBus'
import { widgetInitializerRegistry } from '@/architecture/presentation/widgets/initializers/WidgetInitializerRegistry'
import { SelectWidgetInitializer } from '@/architecture/presentation/widgets/initializers/SelectWidgetInitializer'
import type { SelectWidgetConfig } from '@/core/types/widget-configs'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 获取配置（带类型）
const widgetConfig = computed(() => {
  return (props.field.widget?.config || {}) as SelectWidgetConfig
})

// 选项列表
const options = ref<Array<{ label: string; value: any; disabled?: boolean; displayInfo?: string }>>([])

/**
 * 🔥 静态选项（从配置中获取，用于颜色索引对齐）
 * options_colors 数组与静态选项的索引对齐
 */
const staticOptions = computed(() => {
  const configOptions = widgetConfig.value.options || []
  if (Array.isArray(configOptions)) {
    if (typeof configOptions[0] === 'string') {
      // 字符串数组
      return configOptions.map(opt => ({
        label: opt,
        value: opt
      }))
    } else {
      // 对象数组
      return configOptions
    }
  }
  return []
})

/**
 * 🔥 选项颜色配置
 * 
 * 支持两种颜色格式：
 * 1. Element Plus 标准颜色类型：success, warning, danger, info, primary
 *    使用 el-tag 的 type 属性
 * 2. 自定义颜色（hex 格式）：如 #FF5722, #4CAF50
 *    使用 el-tag 的 color 属性
 * 
 * options_colors 数组与 staticOptions 数组的索引对齐，通过索引获取对应选项的颜色
 */
const optionColors = computed(() => {
  return widgetConfig.value.options_colors || []
})

// isStandardColor 已从 constants/select 导入

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
  
  // 查找当前值在 options 中的索引
  const optionIndex = options.value.findIndex(opt => opt.value === rawValue)
  if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
    return optionColors.value[optionIndex]
  }
  
  return null
})

/**
 * 🔥 获取选项的颜色（用于下拉选项显示）
 * 注意：options_colors 数组与 staticOptions 数组的索引对齐
 * 即使 options 可能包含动态选项，颜色配置仍然基于 staticOptions 的索引
 */
function getOptionColor(value: any): string | null {
  const valueStr = String(value)
  // 🔥 在 staticOptions 中查找索引（因为 options_colors 与 staticOptions 对齐）
  const optionIndex = staticOptions.value.findIndex((opt: any) => String(opt.value) === valueStr)
  if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
    return optionColors.value[optionIndex]
  }
  return null
}

/**
 * 🔥 获取选项的颜色样式对象（用于 span 的 style 绑定）
 */
function getOptionColorStyle(value: any): Record<string, string> {
  const color = getOptionColor(value)
  if (!color) return {}
  
  const isStandard = isStandardColor(color)
  // 🔥 对于标准颜色，也需要设置背景色（使用 Element Plus 的颜色变量）
  const backgroundColor = isStandard 
    ? getStandardColorCSSVar(color as StandardColorType) 
    : color
  
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

// 加载状态
const loading = ref(false)

// 是否有回调接口
const hasCallback = computed(() => {
  return props.field.callbacks?.includes('OnSelectFuzzy') || false
})

// 对话框相关状态
const dialogVisible = ref(false)
const dialogSuggestions = ref<Array<{ label: string; value: any; displayInfo?: any; icon?: string }>>([])

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
  set: (newValue: any) => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const selectedOption = options.value.find(opt => opt.value === newValue)
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

// 获取 display_info 的显示文本
const displayInfoText = computed(() => {
  const value = props.value
  if (!value || !value.raw) {
    return ''
  }
  
  // 🔥 优先从 meta.displayInfo 获取（这是保存的值）
  if (value.meta?.displayInfo) {
    const info = value.meta.displayInfo
    // 如果是数组（多选的情况），取第一个
    if (Array.isArray(info) && info.length > 0) {
      const firstInfo = info[0]
      if (firstInfo && typeof firstInfo === 'object') {
        const infoItems: string[] = []
        Object.entries(firstInfo).forEach(([key, val]) => {
          if (val !== null && val !== undefined && val !== '') {
            infoItems.push(`${key}: ${val}`)
          }
        })
        if (infoItems.length > 0) {
          // 限制显示数量，避免过长
          if (infoItems.length > 5) {
            return infoItems.slice(0, 5).join(' | ') + ' ...'
          }
          return infoItems.join(' | ')
        }
      }
    } else if (typeof info === 'object' && info !== null) {
      // 如果是对象（单选的情况）
      const infoItems: string[] = []
      Object.entries(info).forEach(([key, val]) => {
        if (val !== null && val !== undefined && val !== '') {
          infoItems.push(`${key}: ${val}`)
        }
      })
      if (infoItems.length > 0) {
        // 限制显示数量，避免过长
        if (infoItems.length > 5) {
          return infoItems.slice(0, 5).join(' | ') + ' ...'
        }
        return infoItems.join(' | ')
      }
    }
  }
  
  // 🔥 如果 meta 中没有，从 options 中查找
  const selectedOption = options.value.find((opt: any) => {
    return opt.value === value.raw || String(opt.value) === String(value.raw)
  })
  
  if (selectedOption?.displayInfo) {
    const info = selectedOption.displayInfo
    if (typeof info === 'object' && info !== null) {
      const infoItems: string[] = []
      Object.entries(info).forEach(([key, val]) => {
        if (val !== null && val !== undefined && val !== '') {
          infoItems.push(`${key}: ${val}`)
        }
      })
      if (infoItems.length > 0) {
        if (infoItems.length > 5) {
          return infoItems.slice(0, 5).join(' | ') + ' ...'
        }
        return infoItems.join(' | ')
      }
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
  
  // 🔥 在详情模式下，优先使用 detailDisplayValue（通过回调获取的）
  // 如果 value.display 为空或等于 raw（说明没有有意义的显示值），则使用 detailDisplayValue
  if (props.mode === 'detail') {
    // 如果 detailDisplayValue 有值（通过回调获取的），优先使用
    if (detailDisplayValue.value) {
      return detailDisplayValue.value
    }
    // 如果 value.display 为空或等于 raw，说明没有有意义的显示值，尝试从 options 中查找
    if ((!value.display || value.display === '' || String(value.display) === String(value.raw)) && value.raw !== null && value.raw !== undefined && value.raw !== '') {
      const matchedOption = options.value.find((opt: any) => {
        // 支持多种类型比较
        return opt.value === value.raw || String(opt.value) === String(value.raw)
      })
      if (matchedOption) {
        return matchedOption.label
      }
      // 如果找不到匹配的选项，返回 raw 值（作为后备）
      return String(value.raw)
    }
    // 如果 value.display 有值且不等于 raw，使用 value.display
    if (value.display && String(value.display) !== String(value.raw)) {
      return value.display
    }
    // 如果 value.display 为空，返回 raw 值
    return value.raw !== null && value.raw !== undefined ? String(value.raw) : '-'
  }
  
  // 🔥 非详情模式下，优先使用 value.display
  if (value.display) {
    return value.display
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  return String(raw)
})

// 初始化选项
function initOptions(): void {
  const configOptions = widgetConfig.value.options
  if (configOptions && Array.isArray(configOptions)) {
    if (typeof configOptions[0] === 'string') {
      // 字符串数组
      options.value = configOptions.map(opt => ({
        label: opt,
        value: opt
      }))
    } else {
      // 对象数组
      options.value = configOptions
    }
  }
  
  // 🔥 如果有回调接口且有初始值，触发一次搜索（包括详情模式）
  // 详情模式下也需要触发回调，通过 by_value 查询来获取选项标签
  // ⚠️ 注意：详情模式下由 watch 处理，这里只处理非详情模式
  if (hasCallback.value && props.value?.raw && props.mode !== 'detail') {
    if (props.formRenderer) {
      handleSearch(props.value.raw, true) // by_value
    }
  }
  
  // 🔥 详情模式下，如果已经有 formRenderer，由 watch 处理
  // 如果没有 formRenderer，等待 watch 检测到 formRenderer 后再触发
}

// 处理远程搜索（保留用于兼容）
async function handleRemoteSearch(query: string): Promise<void> {
  if (!hasCallback.value) {
    return
  }
  
  await handleSearch(query, false) // by_keyword
}

// 打开对话框
async function openDialog(): Promise<void> {
  dialogVisible.value = true
  
  // 如果有回调接口
  if (hasCallback.value) {
    // 🔥 检查 formRenderer 是否存在
    if (!props.formRenderer) {
      console.warn('[SelectWidget] openDialog: formRenderer 不存在，无法触发回调', {
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
    dialogSuggestions.value = options.value.map((opt: any) => ({
      label: opt.label,
      value: opt.value,
      displayInfo: opt.displayInfo,
      display_info: opt.displayInfo, // 同时提供两种格式，确保兼容
      icon: opt.icon
    }))
  }
}

// 处理对话框搜索
async function handleDialogSearch(keyword: string): Promise<void> {
  if (hasCallback.value) {
    // 🔥 检查 formRenderer 是否存在
    if (!props.formRenderer) {
      console.warn('[SelectWidget] handleDialogSearch: formRenderer 不存在，无法触发回调', {
        fieldCode: props.field.code,
        keyword
      })
      return
    }
    
    await handleSearch(keyword, false)
    
    // 更新对话框建议列表
    dialogSuggestions.value = options.value.map((opt: any) => ({
      label: opt.label,
      value: opt.value,
      displayInfo: opt.displayInfo,
      display_info: opt.displayInfo, // 同时提供两种格式，确保兼容
      icon: opt.icon
    }))
  } else {
    // 静态选项，本地过滤
    const filtered = staticOptions.value.filter((opt: any) => {
      const label = typeof opt === 'string' ? opt : opt.label
      return label.toLowerCase().includes(keyword.toLowerCase())
    })
    dialogSuggestions.value = filtered.map((opt: any) => {
      if (typeof opt === 'string') {
        return { label: opt, value: opt }
      }
      return {
        label: opt.label,
        value: opt.value,
        displayInfo: opt.displayInfo,
        display_info: opt.displayInfo, // 同时提供两种格式，确保兼容
        icon: opt.icon
      }
    })
  }
}

// 处理对话框选择（单选模式）
function handleDialogSelect(item: { value: any; label?: string; displayInfo?: any }): void {
  // 🔥 更新 options，确保选择的项的 displayInfo 被保存
  const existingOption = options.value.find((opt: any) => String(opt.value) === String(item.value))
  if (!existingOption) {
    // 如果 options 中没有，添加进去
    options.value.push({
      label: item.label || String(item.value),
      value: item.value,
      displayInfo: item.displayInfo
    })
  } else if (item.displayInfo && !existingOption.displayInfo) {
    // 如果 options 中有但没有 displayInfo，更新它
    existingOption.displayInfo = item.displayInfo
  }
  
  const selectedOption = options.value.find((opt: any) => String(opt.value) === String(item.value))
  // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
  const newFieldValue = createFieldValue(
    props.field,
    item.value,
    item.label || selectedOption?.label || String(item.value),
    {
      displayInfo: item.displayInfo || selectedOption?.displayInfo,
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
async function handleSearch(query: string | number, isByValue: boolean): Promise<void> {
  if (!hasCallback.value || !props.formRenderer) {
    return
  }
  
  const method = props.formRenderer.getFunctionMethod()
  const router = props.formRenderer.getFunctionRouter()
  
  if (!router) {
    console.warn('[SelectWidget] 无法获取函数路由，取消回调', { fieldCode: props.field.code, router })
    return
  }
  
  
  loading.value = true
  
  try {
    // 🔥 类型转换：根据 value_type 将字符串转换为正确的类型
    const valueType = props.field.data?.type || 'string'
    let convertedValue: any = query
    
    // 🔥 如果 query 已经是数字类型，不需要转换
    if (isByValue && typeof query === 'string' && valueType !== 'string') {
      // 使用统一的类型转换工具函数
      convertedValue = convertValueToType(query, valueType, 'SelectWidget')
    }
    
    // 🔥 获取提交数据并根据字段类型进行转换
    // 使用统一的类型转换函数，确保所有字段都根据 field.data.type 正确转换
    const submitData = props.formRenderer.getSubmitData()
    const functionDetail = props.formRenderer.getFunctionDetail?.()
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
      currentStatistics.value = response.statistics
      // 如果当前已有选中值，立即更新 meta.statistics
      if (props.value?.raw) {
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
      options.value = response.items.map((item: any) => ({
        label: item.label || String(item.value),
        value: item.value,
        disabled: false,
        displayInfo: item.display_info || item.displayInfo
      }))
      
      // 🔥 如果是通过 by_value 查询，找到匹配的选项并更新显示值
      if (isByValue && props.value?.raw) {
        const matchedOption = options.value.find((opt: any) => {
          // 支持多种类型比较
          return opt.value === props.value.raw || String(opt.value) === String(props.value.raw)
        })
        if (matchedOption) {
          // 🔥 在详情模式下，更新 detailDisplayValue
          if (props.mode === 'detail') {
            detailDisplayValue.value = matchedOption.label
          }
          // 🔥 在编辑模式下，如果 value.display 为空或等于 raw，更新 display 值
          if (props.mode === 'edit' && (!props.value.display || String(props.value.display) === String(props.value.raw))) {
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
  } catch (error: any) {
    Logger.error('SelectWidget', '回调失败', error)
    ElMessage.error(error?.message || '查询失败')
    options.value = []
  } finally {
    loading.value = false
  }
}

// 当前统计信息（从回调接口获取）
const currentStatistics = ref<Record<string, string>>({})

// 处理值变化
function handleChange(value: any): void {
  // 值变化时，保存 displayInfo 和 statistics
  const selectedOption = options.value.find(opt => opt.value === value)
  if (selectedOption) {
    // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
    const newFieldValue = createFieldValue(
      props.field,
      value,
      selectedOption.label,
      {
        displayInfo: selectedOption.displayInfo,
        statistics: currentStatistics.value  // 🔥 保存 statistics 配置
      }
    )
    
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
  }
}

// 处理聚焦（已移除，因为 Element Plus 的 remote-method 会在聚焦时自动触发）
// 如果同时使用 handleFocus 和 remote-method，会导致重复回调

// 🔥 事件监听器（用于在表单初始化完成后统一处理）
let unsubscribeFormInitialized: (() => void) | null = null

// 🔥 注册监听器的函数（避免重复代码）
const registerFormInitializedListener = () => {
  // 🔥 先取消注册旧的监听器（避免重复注册）
  if (unsubscribeFormInitialized) {
    unsubscribeFormInitialized()
    unsubscribeFormInitialized = null
  }
  
  // 注册新的监听器
  unsubscribeFormInitialized = eventBus.on(FormEvent.initialized, () => {
    Logger.debug('[SelectWidget]', 'FormEvent.initialized 收到', { 
      fieldCode: props.field.code,
      hasCallback: hasCallback.value,
      rawValue: props.value?.raw,
      formRenderer: !!props.formRenderer,
      getFunctionDetail: !!props.formRenderer?.getFunctionDetail,
      functionDetail: props.formRenderer?.getFunctionDetail?.(),
      functionId: props.formRenderer?.getFunctionDetail?.()?.id,
      lastSearchedValue: lastSearchedValue.value,
      lastSearchedRouter: lastSearchedRouter.value,
      lastSearchedFunctionId: lastSearchedFunctionId.value
    })
    // 如果当前字段有 OnSelectFuzzy 回调，且有值，触发搜索获取 label
    if (hasCallback.value && props.value?.raw !== null && props.value?.raw !== undefined && props.formRenderer) {
      nextTick(() => {
        if (props.formRenderer && !isSearching.value && props.value?.raw !== lastSearchedValue.value) {
          Logger.debug('[SelectWidget]', '触发 triggerSearchIfNeeded (FormEvent.initialized)', { 
            fieldCode: props.field.code,
            rawValue: props.value?.raw,
            functionId: props.formRenderer?.getFunctionDetail?.()?.id,
            lastSearchedValue: lastSearchedValue.value,
            lastSearchedRouter: lastSearchedRouter.value,
            lastSearchedFunctionId: lastSearchedFunctionId.value
          })
          triggerSearchIfNeeded(props.value.raw, props.formRenderer, props.mode)
        }
      })
    }
  })
}

// 🔥 取消注册监听器的函数
const unregisterFormInitializedListener = () => {
  if (unsubscribeFormInitialized) {
    unsubscribeFormInitialized()
    unsubscribeFormInitialized = null
  }
}

// 初始化
onMounted(() => {
  initOptions()
  
  // 🔥 注册 SelectWidget 初始化器（组件自治）
  // 只在有 OnSelectFuzzy 回调时才注册
  if (hasCallback.value) {
    widgetInitializerRegistry.register('select', new SelectWidgetInitializer())
    Logger.debug('[SelectWidget]', '注册初始化器', {
      fieldCode: props.field.code,
      widgetType: 'select'
    })
  }
  
  // 🔥 移除 FormEvent.initialized 监听器，统一使用 SelectWidgetInitializer 处理初始化
  // 🔥 SelectWidgetInitializer 在 useFunctionParamInitialization 的 triggerWidgetInitialization 中调用
  // 🔥 这样可以避免重复调用回显接口，并且保证初始化逻辑的统一性
  // 🔥 如果未来需要保留这个监听器，需要添加防重复调用的机制
})

// 🔥 组件卸载时取消注册监听器
onUnmounted(() => {
  // 🔥 只在已注册的情况下才取消注册
  if (unsubscribeFormInitialized) {
    Logger.debug('[SelectWidget]', 'onUnmounted - 取消注册监听器', { 
      fieldCode: props.field.code
    })
    unregisterFormInitializedListener()
  }
})

onUnmounted(() => {
  unregisterFormInitializedListener()
})

// 🔥 监听 value 和 formRenderer 变化，如果值变化了，重新触发回调获取标签
// 使用一个标志来防止重复调用
const isSearching = ref(false)
const lastSearchedValue = ref<any>(null)
const lastSearchedRouter = ref<string | null>(null) // 🔥 记录上次搜索使用的 router
const lastSearchedFunctionId = ref<number | null>(null) // 🔥 记录上次搜索使用的函数 ID

// 🔥 优化：控制日志输出（默认关闭，调试时可以改为 true）
const ENABLE_DETAILED_LOGS = false

// 🔥 触发搜索的辅助函数（避免重复代码）
const triggerSearchIfNeeded = (rawValue: any, formRenderer: any, mode: string) => {
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
  
  if (!hasCallback.value || !formRenderer) {
    if (shouldLog) {
      Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 跳过：无回调或无 formRenderer', {
        fieldCode: props.field.code,
        hasCallback: hasCallback.value,
        formRenderer: !!formRenderer
      })
    }
    return false
  }
  
  const currentRouter = formRenderer.getFunctionRouter?.()
  if (!currentRouter) {
    if (shouldLog) {
      Logger.debug('[SelectWidget]', 'triggerSearchIfNeeded 跳过：无 currentRouter', {
        fieldCode: props.field.code
      })
    }
    return false
  }
  
  // 🔥 获取当前函数 ID（用于防重复调用）
  const currentFunctionId = formRenderer.getFunctionDetail?.()?.id || null
  
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
        props.formRenderer && 
        newRaw !== null && 
        newRaw !== undefined && 
        newRaw !== oldRaw) {
      triggerSearchIfNeeded(newRaw, props.formRenderer, props.mode)
    }
  }
)
</script>

<style scoped>
.select-widget {
  width: 100%;
}

.edit-select,
.search-select {
  width: 100%;
  position: relative;
}

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

.select-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
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

.select-label:empty::before {
  content: attr(data-placeholder);
  color: var(--el-text-color-placeholder);
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

.select-container:hover .input-icon {
  color: var(--el-color-primary);
  transform: translateY(1px);
}

.select-option {
  display: flex;
  align-items: center;
}

.select-option > *:not(:last-child) {
  margin-right: 8px;
}

.option-label {
  flex: 1;
}

.display-info {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: auto;
}

.response-value {
  color: var(--el-text-color-regular);
}

.table-cell-value {
  display: inline-flex;
  align-items: center;
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

/* 🔥 空心样式：使用边框和透明背景 */
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

/* 🔥 多选模式样式（从 MultiSelectWidget 复制） */
.edit-multiselect {
  width: 100%;
}

.selected-tags-container {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 4px 8px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background-color: var(--el-fill-color-blank);
  cursor: pointer;
  transition: border-color 0.2s;
}

.selected-tags-container:hover {
  border-color: var(--el-color-primary);
}

.tags-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.input-wrapper {
  flex: 1;
  min-width: 120px;
  position: relative;
}

.multiselect-input {
  width: 100%;
}

.multiselect-tag {
  margin: 0;
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 4px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 自定义颜色的空心标签：使用边框颜色 */
.select-tag-outline[style*="color"] {
  border-color: currentColor !important;
}

.table-cell-value .el-tag {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* 自定义颜色的 tag，确保文字清晰 */
.table-cell-value .el-tag[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
}

.detail-value {
  margin-bottom: 16px;
  display: inline-flex;
  align-items: center;
}

.detail-value .el-tag {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* 自定义颜色的 tag，确保文字清晰 */
.detail-value .el-tag[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
}

/* 🔥 下拉选项中的颜色指示器样式 */
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
  /* 🔥 降低亮度：使用 filter 降低饱和度和亮度 */
  filter: brightness(0.95) saturate(0.9);
  opacity: 0.9;
}

.detail-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.detail-content {
  color: var(--el-text-color-regular);
}
</style>


