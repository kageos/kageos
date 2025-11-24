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
    <el-select
      v-if="mode === 'edit'"
      v-model="internalValue"
      :disabled="field.widget?.config?.disabled"
      :placeholder="field.desc || `请选择${field.name}`"
      :clearable="true"
      :filterable="isFilterable"
      :loading="loading"
      :remote="hasCallback"
      :remote-method="handleRemoteSearch"
      popper-class="select-dropdown-popper"
      :popper-options="{
        strategy: 'fixed',
        modifiers: [
          {
            name: 'computeStyles',
            options: {
              adaptive: false,
              roundOffsets: false
            }
          },
          {
            name: 'offset',
            options: {
              offset: [0, 4]
            }
          }
        ]
      }"
      @change="handleChange"
    >
      <el-option
        v-for="option in options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
        :disabled="option.disabled"
      >
        <div class="select-option">
          <!-- 🔥 显示颜色指示器（如果有颜色配置，放在左侧） -->
          <span
            v-if="getOptionColor(option.value)"
            class="option-color-indicator"
            :style="getOptionColorStyle(option.value)"
          />
          <span class="option-label">{{ option.label }}</span>
          <span v-if="option.displayInfo" class="display-info">{{ option.displayInfo }}</span>
        </div>
      </el-option>
    </el-select>
    
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
    <el-select
      v-else-if="mode === 'search'"
      v-model="internalValue"
      :placeholder="`搜索${field.name}`"
      :clearable="true"
      :filterable="isFilterable"
      :loading="loading"
      :remote="hasCallback"
      :remote-method="handleRemoteSearch"
    >
      <el-option
        v-for="option in options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
      >
        <!-- 🔥 显示颜色指示器（如果有颜色配置，放在左侧） -->
        <div class="select-option">
          <span
            v-if="getOptionColor(option.value)"
            class="option-color-indicator"
            :style="getOptionColorStyle(option.value)"
          />
          <span class="option-label">{{ option.label }}</span>
        </div>
      </el-option>
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { ElSelect, ElOption, ElMessage, ElTag } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { selectFuzzy } from '@/api/function'
import { Logger } from '../../utils/logger'
import { SelectFuzzyQueryType, isStandardColor, getStandardColorCSSVar, type StandardColorType } from '../../constants/select'
import { convertValueToType } from '../utils/valueConverter'

const COMPONENT_NAME = 'SelectWidget'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 选项列表
const options = ref<Array<{ label: string; value: any; disabled?: boolean; displayInfo?: string }>>([])

/**
 * 🔥 静态选项（从配置中获取，用于颜色索引对齐）
 * options_colors 数组与静态选项的索引对齐
 */
const staticOptions = computed(() => {
  const configOptions = props.field.widget?.config?.options || []
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
  return props.field.widget?.config?.options_colors || []
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
    const color = optionColors.value[optionIndex]
    Logger.debug(COMPONENT_NAME, `getOptionColor - value: ${valueStr}, index: ${optionIndex}, color: ${color}`)
    return color
  }
  // 🔥 调试日志：未找到颜色
  Logger.debug(COMPONENT_NAME, `getOptionColor - value: ${valueStr}, not found in staticOptions`, {
    staticOptions: staticOptions.value,
    optionColors: optionColors.value
  })
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
  
  Logger.debug(COMPONENT_NAME, `getOptionColorStyle - value: ${value}, color: ${color}, isStandard: ${isStandard}, backgroundColor: ${backgroundColor}`)
  
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

// 是否可搜索
const isFilterable = computed(() => {
  return props.field.widget?.config?.filterable !== false
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      return props.value?.raw ?? null
    }
    return null
  },
  set: (newValue: any) => {
    if (props.mode === 'edit') {
      const selectedOption = options.value.find(opt => opt.value === newValue)
      const newFieldValue = {
        raw: newValue,
        display: selectedOption?.label || String(newValue),
        meta: {
          displayInfo: selectedOption?.displayInfo
        }
      }
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 🔥 详情模式下通过回调获取的显示值（用于存储）
const detailDisplayValue = ref<string | null>(null)

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
  const configOptions = props.field.widget?.config?.options
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

// 处理远程搜索
async function handleRemoteSearch(query: string): Promise<void> {
  if (!hasCallback.value) {
    return
  }
  
  await handleSearch(query, false) // by_keyword
}

// 处理搜索
async function handleSearch(query: string | number, isByValue: boolean): Promise<void> {
  if (!hasCallback.value || !props.formRenderer) {
    return
  }
  
  const method = props.formRenderer.getFunctionMethod()
  const router = props.formRenderer.getFunctionRouter()
  
  if (!router) {
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
      convertedValue = convertValueToType(query, valueType, COMPONENT_NAME)
    }
    
    const requestBody = {
      code: props.field.code,
      type: isByValue ? SelectFuzzyQueryType.BY_VALUE : SelectFuzzyQueryType.BY_KEYWORD,
      value: convertedValue, // 🔥 使用转换后的值
      request: props.formRenderer.getSubmitData(),
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
        const newFieldValue = {
          ...props.value,
          meta: {
            ...props.value.meta,
            statistics: currentStatistics.value
          }
        }
        formDataStore.setValue(props.fieldPath, newFieldValue)
      }
    }
    
    if (response.items && Array.isArray(response.items)) {
      options.value = response.items.map((item: any) => ({
        label: item.label || String(item.value),
        value: item.value,
        disabled: false,
        displayInfo: item.display_info
      }))
      
      // 🔥 如果是在详情模式下通过 by_value 查询，找到匹配的选项并更新显示值
      if (isByValue && props.mode === 'detail' && props.value?.raw) {
        const matchedOption = options.value.find((opt: any) => {
          // 支持多种类型比较
          return opt.value === props.value.raw || String(opt.value) === String(props.value.raw)
        })
        if (matchedOption) {
          // 🔥 更新 detailDisplayValue，这样 displayValue 计算属性就能显示正确的标签
          detailDisplayValue.value = matchedOption.label
          Logger.debug(COMPONENT_NAME, '详情模式回调成功，更新 detailDisplayValue', {
            raw: props.value.raw,
            label: matchedOption.label,
            detailDisplayValue: detailDisplayValue.value
          })
        } else {
          Logger.warn(COMPONENT_NAME, '详情模式回调成功，但未找到匹配的选项', {
            raw: props.value.raw,
            options: options.value
          })
        }
      }
    } else {
      options.value = []
    }
  } catch (error: any) {
    Logger.error(COMPONENT_NAME, '回调失败', error)
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
    const newFieldValue = {
      raw: value,
      display: selectedOption.label,
      meta: {
        displayInfo: selectedOption.displayInfo,
        statistics: currentStatistics.value  // 🔥 保存 statistics 配置
      }
    }
    
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
  }
}

// 处理聚焦（已移除，因为 Element Plus 的 remote-method 会在聚焦时自动触发）
// 如果同时使用 handleFocus 和 remote-method，会导致重复回调

// 初始化
onMounted(() => {
  initOptions()
  
  // 🔥 详情模式下，如果已经有 formRenderer 和值，立即触发一次回调
  // 因为 watch 可能在组件挂载时 formRenderer 还没传递过来
  if (props.mode === 'detail' && hasCallback.value && props.value?.raw && props.formRenderer) {
    nextTick(() => {
      if (!isSearching.value && props.value?.raw !== lastSearchedValue.value) {
        isSearching.value = true
        lastSearchedValue.value = props.value.raw
        detailDisplayValue.value = null
        handleSearch(props.value.raw, true).finally(() => {
          isSearching.value = false
        })
      }
    })
  }
})

// 🔥 监听 value 和 formRenderer 变化，在详情模式下如果值变化了，重新触发回调获取标签
// 使用一个标志来防止重复调用
const isSearching = ref(false)
const lastSearchedValue = ref<any>(null)

watch(
  () => [props.value?.raw, props.formRenderer, props.mode],
  ([newRaw, formRenderer, mode], oldValues) => {
    // 🔥 处理首次执行时 oldValues 为 undefined 的情况
    const [oldRaw, oldFormRenderer, oldMode] = oldValues || [undefined, undefined, undefined]
    
    // 只在详情模式下，且有回调接口，且有值，且有 formRenderer 时触发
    if (
      mode === 'detail' && 
      hasCallback.value && 
      newRaw !== null && 
      newRaw !== undefined && 
      formRenderer &&
      !isSearching.value &&
      // 🔥 关键：如果值或 formRenderer 发生了变化，或者还没有搜索过这个值，就触发
      (newRaw !== lastSearchedValue.value || formRenderer !== oldFormRenderer || mode !== oldMode)
    ) {
      isSearching.value = true
      lastSearchedValue.value = newRaw
      // 重置 detailDisplayValue，等待回调返回后更新
      detailDisplayValue.value = null
      handleSearch(newRaw, true).finally(() => {
        isSearching.value = false
      })
    }
  },
  { immediate: true } // 🔥 立即执行一次，确保在组件挂载时就能触发
)
</script>

<style scoped>
.select-widget {
  width: 100%;
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

<style>
/* 全局样式：确保下拉菜单在抽屉中正常显示 */
.select-dropdown-popper {
  z-index: 3001 !important;
}

.select-dropdown-popper .el-select-dropdown {
  z-index: 3001 !important;
}
</style>

