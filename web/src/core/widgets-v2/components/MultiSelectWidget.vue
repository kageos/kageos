<!--
  MultiSelectWidget - 多选组件
  重写版本，简化逻辑，修复标签显示问题
-->
<template>
  <div class="multiselect-widget">
    <!-- 编辑模式 -->
    <el-select
      v-if="mode === 'edit'"
      ref="selectRef"
      v-model="selectedValues"
      multiple
      filterable
      :remote="hasRemoteSearch"
      :remote-method="remoteMethod"
      :loading="loading"
      :placeholder="placeholder"
      :multiple-limit="maxCount"
      reserve-keyword
      :collapse-tags="false"
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
      clearable
      @change="handleChange"
      @visible-change="handleVisibleChange"
      @remove-tag="handleRemoveTag"
    >
      <!-- 自定义已选标签，应用颜色配置 -->
      <!-- Element Plus 的 #tag 插槽会替换整个标签区域，需要自己遍历所有选中的值 -->
      <template #tag="{ item, close }">
        <el-tag
          v-if="item"
          :type="getOptionColorType(item.value)"
          :color="getOptionColorValue(item.value)"
          :closable="true"
          @close.stop="close"
          class="multiselect-tag"
        >
          {{ getOptionLabel(item.value) }}
        </el-tag>
      </template>
      
      <el-option
        v-for="option in options"
        :key="`${option.value}-${option.label}`"
        :label="option.label"
        :value="option.value"
      >
        <!-- 🔥 在下拉选项中显示带颜色的标签（参考 Element Plus 官方示例） -->
        <div class="flex items-center">
          <span
            v-if="getOptionColor(option.value)"
            class="option-color-indicator"
            :style="getOptionColorStyle(option.value)"
          />
          <span>{{ option.label }}</span>
        </div>
      </el-option>
    </el-select>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-multiselect">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
        :type="getOptionColorType(value)"
        :color="getOptionColorValue(value)"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-multiselect">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
        size="small"
        :type="getOptionColorType(value)"
        :color="getOptionColorValue(value)"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
    
    <!-- 详情模式 -->
    <div v-else class="detail-multiselect">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
        :type="getOptionColorType(value)"
        :color="getOptionColorValue(value)"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, withDefaults } from 'vue'
import { ElSelect, ElOption, ElTag } from 'element-plus'
import type { WidgetComponentProps } from '../types'
import { selectFuzzy } from '@/api/function'
import { Logger } from '../../utils/logger'
import { useFormDataStore } from '../../stores-v2/formData'
import { ExpressionParser } from '../../utils/ExpressionParser'
import { isStringDataType, getMultiSelectDefaultDataType } from '../../constants/widget'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const formDataStore = useFormDataStore()

// 配置
const config = computed(() => props.field.widget?.config || {})

/**
 * 🔥 选项颜色配置
 * 
 * 支持两种颜色格式：
 * 1. Element Plus 标准颜色类型：success, warning, danger, info, primary
 *    使用 el-tag 的 type 属性
 * 2. 自定义颜色（hex 格式）：如 #FF5722, #4CAF50
 *    使用 el-tag 的 color 属性
 * 
 * options_colors 数组与 options 数组的索引对齐，通过索引获取对应选项的颜色
 */
const optionColors = computed(() => {
  return config.value.options_colors || []
})

const staticOptions = computed(() => {
  const opts = config.value.options || []
  return opts.map((opt: any) => {
    if (typeof opt === 'string') {
      return { label: opt, value: opt }
    }
    return opt
  })
})

// 动态选项（从回调接口获取）
const dynamicOptions = ref<Array<{ label: string; value: any; displayInfo?: any; icon?: string }>>([])

// 合并后的选项（静态 + 动态）
const options = computed(() => {
  if (hasRemoteSearch.value && dynamicOptions.value.length > 0) {
    return dynamicOptions.value
  }
  return staticOptions.value
})

const placeholder = computed(() => {
  return config.value.placeholder || `请选择${props.field.name}`
})

// 动态最大选择数量（从回调接口获取）
const dynamicMaxCount = ref<number>(0)
const maxCount = computed(() => {
  if (dynamicMaxCount.value > 0) {
    return dynamicMaxCount.value
  }
  return config.value.max_count || 0
})

// 是否支持远程搜索
const hasRemoteSearch = computed(() => {
  return props.field.callbacks?.includes('OnSelectFuzzy') || false
})

// 加载状态
const loading = ref(false)

// 下拉框引用
const selectRef = ref<InstanceType<typeof ElSelect> | null>(null)

// 是否因为选择而需要保持打开
const shouldKeepOpen = ref(false)

/**
 * 🔥 多选组件支持两种数据类型：
 * 1. string 类型：提交时使用逗号分隔的字符串格式（如 "紧急,低优先级"）
 *    适用于后端字段类型为 string，需要存储到数据库的字符串字段
 * 2. []string 或 array 类型：提交时使用数组格式（如 ["紧急", "低优先级"]）
 *    适用于后端字段类型为 []string，可以存储数组
 * 
 * 根据 field.data.type 自动决定提交格式，确保与后端字段类型严格对齐
 */
const fieldDataType = computed(() => {
  return props.field.data?.type || getMultiSelectDefaultDataType()
})

/**
 * 解析原始值为数组
 */
function parseRawValue(raw: any): string[] {
  if (Array.isArray(raw)) {
    return raw.map(v => String(v))
  }
  if (typeof raw === 'string' && raw) {
    if (raw.includes(',')) {
      return raw.split(',').map(v => v.trim()).filter(v => v)
    }
    return [raw]
  }
  return []
}

// 选中的值（数组）
const selectedValues = computed({
  get: () => {
    return parseRawValue(props.value?.raw)
  },
  set: (newValues: any[]) => {
    let finalValues = newValues.map(v => String(v))
    
    if (maxCount.value > 0 && finalValues.length > maxCount.value) {
      Logger.warn('MultiSelectWidget', `${props.field.code} 超出数量限制! 限制: ${maxCount.value}, 实际: ${finalValues.length}`)
      finalValues = finalValues.slice(0, maxCount.value)
    }
    
    const displayInfos = finalValues.map((val: any) => {
      const option = options.value.find((opt: any) => String(opt.value) === val)
      return option?.displayInfo || null
    })
    
    const displayText = finalValues.map((val: any) => {
      const option = options.value.find((opt: any) => String(opt.value) === val)
      return option?.label || String(val)
    }).join(', ')
    
    // 🔥 计算行内聚合统计（如果有 statistics 配置）
    const rowStatistics = calculateRowStatistics(displayInfos, currentStatistics.value)
    
    /**
     * 🔥 根据 field.data.type 决定 raw 的格式
     * - 如果 type 是 string：提交逗号分隔的字符串（如 "紧急,低优先级"）
     * - 如果 type 是 []string 或其他数组类型：提交数组（如 ["紧急", "低优先级"]）
     */
    let rawValue: any
    const dataType = fieldDataType.value
    if (isStringDataType(dataType)) {
      // 提交逗号分隔的字符串
      rawValue = finalValues.length > 0 ? finalValues.join(',') : ''
    } else {
      // 提交数组（[]string 或其他数组类型）
      rawValue = finalValues
    }
    
    const fieldValue = {
      raw: rawValue,
      display: displayText || '未选择',
      meta: {
        displayInfo: displayInfos,
        statistics: currentStatistics.value,
        rowStatistics: rowStatistics
      }
    }
    
    formDataStore.setValue(props.fieldPath, fieldValue)
    emit('update:modelValue', fieldValue)
  }
})

// 当前统计信息（从回调接口获取）
const currentStatistics = ref<Record<string, any>>({})

// 显示值（用于只读模式）
const displayValues = computed(() => {
  return parseRawValue(props.value?.raw)
})

// 获取选项标签
function getOptionLabel(value: any): string {
  if (value === null || value === undefined) return ''
  
  const valueStr = String(value)
  const option = options.value.find((opt: any) => String(opt.value) === valueStr)
  return option ? option.label : valueStr
}

/**
 * 移除指定值
 */
function handleRemoveValue(value: any): void {
  const newValues = selectedValues.value.filter(v => String(v) !== String(value))
  selectedValues.value = newValues
}

/**
 * 判断是否是 Element Plus 标准颜色类型
 */
function isStandardColor(color: string): boolean {
  return ['success', 'warning', 'danger', 'info', 'primary'].includes(color)
}

/**
 * 获取选项的颜色
 * 🔥 注意：options_colors 数组与 staticOptions 数组的索引对齐
 * 即使 options 可能包含 dynamicOptions，颜色配置仍然基于 staticOptions 的索引
 */
function getOptionColor(value: any): string | null {
  const valueStr = String(value)
  // 🔥 在 staticOptions 中查找索引（因为 options_colors 与 staticOptions 对齐）
  const optionIndex = staticOptions.value.findIndex((opt: any) => String(opt.value) === valueStr)
  if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
    const color = optionColors.value[optionIndex]
    // 🔥 调试日志：检查颜色配置
    if (process.env.NODE_ENV === 'development') {
      console.log(`[MultiSelectWidget] getOptionColor - value: ${valueStr}, index: ${optionIndex}, color: ${color}`)
    }
    return color
  }
  // 🔥 调试日志：未找到颜色
  if (process.env.NODE_ENV === 'development') {
    console.log(`[MultiSelectWidget] getOptionColor - value: ${valueStr}, not found in staticOptions`)
    console.log(`[MultiSelectWidget] staticOptions:`, staticOptions.value)
    console.log(`[MultiSelectWidget] optionColors:`, optionColors.value)
  }
  return null
}

/**
 * 获取选项的颜色类型（用于 el-tag 的 type 属性）
 */
function getOptionColorType(value: any): string | undefined {
  const color = getOptionColor(value)
  if (!color) return undefined
  const isStandard = isStandardColor(color)
  // 🔥 调试日志
  if (process.env.NODE_ENV === 'development') {
    console.log(`[MultiSelectWidget] getOptionColorType - value: ${value}, color: ${color}, isStandard: ${isStandard}, result: ${isStandard ? color : undefined}`)
  }
  return isStandard ? color : undefined
}

/**
 * 获取选项的颜色值（用于 el-tag 的 color 属性）
 * 🔥 注意：el-tag 的 color 属性只接受自定义颜色（hex），标准颜色使用 type 属性
 */
function getOptionColorValue(value: any): string | undefined {
  const color = getOptionColor(value)
  if (!color) {
    // 🔥 调试日志：未找到颜色
    if (process.env.NODE_ENV === 'development') {
      console.log(`[MultiSelectWidget] getOptionColorValue - value: ${value}, no color found`)
    }
    return undefined
  }
  const isStandard = isStandardColor(color)
  const result = !isStandard ? color : undefined
  // 🔥 调试日志
  if (process.env.NODE_ENV === 'development') {
    console.log(`[MultiSelectWidget] getOptionColorValue - value: ${value}, color: ${color}, isStandard: ${isStandard}, result: ${result}`)
  }
  return result
}

/**
 * 🔥 获取选项的颜色样式对象（用于 span 的 style 绑定）
 */
function getOptionColorStyle(value: any): Record<string, string> {
  const colorValue = getOptionColorValue(value)
  const color = getOptionColor(value)
  const backgroundColor = colorValue || color || ''
  
  // 🔥 调试日志
  if (process.env.NODE_ENV === 'development') {
    console.log(`[MultiSelectWidget] getOptionColorStyle - value: ${value}, colorValue: ${colorValue}, color: ${color}, backgroundColor: ${backgroundColor}`)
  }
  
  // 🔥 确保 backgroundColor 有值，并且使用 !important 确保样式生效
  const style: Record<string, string> = {
    marginRight: '8px'
  }
  
  if (backgroundColor) {
    // 🔥 使用内联样式设置 backgroundColor，确保优先级最高
    style.backgroundColor = backgroundColor
    style.display = 'inline-block'
    style.width = '12px'
    style.height = '12px'
    style.minWidth = '12px'
    style.minHeight = '12px'
    style.borderRadius = '2px'
    style.flexShrink = '0'
    style.border = 'none'
    style.verticalAlign = 'middle'
  }
  
  return style
}

/**
 * 🔥 计算行内聚合统计
 */
function calculateRowStatistics(
  displayInfos: any[],
  statisticsConfig: Record<string, string> | null
): Record<string, any> {
  if (!statisticsConfig || Object.keys(statisticsConfig).length === 0) {
    return {}
  }
  
  const validDisplayInfos = displayInfos.filter(info => info && typeof info === 'object')
  
  if (validDisplayInfos.length === 0) {
    return {}
  }
  
  const result: Record<string, any> = {}
  
  try {
    for (const [label, expression] of Object.entries(statisticsConfig)) {
      try {
        const value = ExpressionParser.evaluate(expression, validDisplayInfos)
        result[label] = value
      } catch (error) {
        Logger.error(`[MultiSelectWidget] 行内聚合计算失败: ${label} = ${expression}`, error)
        result[label] = 0
      }
    }
  } catch (error) {
    Logger.error('[MultiSelectWidget] 行内聚合计算失败', error)
  }
  
  return result
}

/**
 * 处理搜索（OnSelectFuzzy 回调）
 */
async function handleSearch(query: string | any[], isByValue = false): Promise<void> {
  if (!hasRemoteSearch.value) {
    return
  }
  
  const method = props.formRenderer?.getFunctionMethod?.()
  const router = props.formRenderer?.getFunctionRouter?.()
  
  if (!router) {
    Logger.error('MultiSelectWidget', `${props.field.code} 无法获取函数路由，取消回调`)
    return
  }

  loading.value = true

  try {
    let queryType: 'by_keyword' | 'by_value' | 'by_values'
    if (isByValue) {
      queryType = Array.isArray(query) ? 'by_values' : 'by_value'
    } else {
      queryType = 'by_keyword'
    }
    
    const requestBody = {
      code: props.field.code,
      type: queryType,
      value: query,
      request: props.formRenderer?.getSubmitData?.() || {},
      value_type: props.field.data?.type || getMultiSelectDefaultDataType()
    }

    const response = await selectFuzzy(method || 'POST', router, requestBody)

    if (response.error_msg) {
      Logger.error('MultiSelectWidget', `${props.field.code} 回调错误: ${String(response.error_msg)}`)
      dynamicOptions.value = []
      return
    }

    if (response.max_selections !== undefined) {
      dynamicMaxCount.value = response.max_selections
    }

    if (response.statistics) {
      currentStatistics.value = response.statistics
    }

    dynamicOptions.value = (response.items || []).map((item: any) => ({
      label: item.label || item.value,
      value: item.value,
      displayInfo: item.display_info || item.displayInfo,
      icon: item.icon
    }))

  } catch (error: any) {
    Logger.error('MultiSelectWidget', `${props.field.code} 回调失败:`, error)
    dynamicOptions.value = []
  } finally {
    loading.value = false
  }
}

// 远程搜索方法
async function remoteMethod(query: string): Promise<void> {
  await handleSearch(query, false)
}

// 选项点击时触发
function handleOptionClick(): void {
  const currentLength = selectedValues.value.length
  const shouldClose = maxCount.value > 0 && currentLength >= maxCount.value - 1
  if (!shouldClose) {
    shouldKeepOpen.value = true
  } else {
    shouldKeepOpen.value = false
  }
}

// 移除标签时触发
function handleRemoveTag(): void {
  shouldKeepOpen.value = true
}

// 下拉框展开时触发
function handleVisibleChange(visible: boolean): void {
  if (visible) {
    const currentLength = selectedValues.value.length
    const shouldClose = maxCount.value > 0 && currentLength >= maxCount.value
    if (!shouldClose) {
      shouldKeepOpen.value = true
    } else {
      shouldKeepOpen.value = false
    }
    
    if (hasRemoteSearch.value) {
      if (dynamicOptions.value.length === 0) {
        handleSearch('', false)
      }
    }
  } else {
    setTimeout(() => {
      if (!shouldKeepOpen.value) {
        return
      }
      
      const input = selectRef.value?.$el?.querySelector('input')
      const isInputFocused = document.activeElement === input
      
      if (!isInputFocused) {
        shouldKeepOpen.value = false
        return
      }
      
      if (shouldKeepOpen.value && isInputFocused) {
        nextTick(() => {
          if (selectRef.value) {
            const selectEl = selectRef.value as any
            const currentInput = (selectEl.$el || selectEl.el || selectEl)?.querySelector?.('input')
            if (currentInput && document.activeElement === currentInput) {
              currentInput.focus()
              if (selectEl.handleMenuEnter) {
                selectEl.handleMenuEnter()
              } else if (selectEl.toggleMenu) {
                selectEl.toggleMenu()
              } else if (selectEl.setSoftFocus) {
                selectEl.setSoftFocus()
              } else {
                if (selectEl.visible !== undefined) {
                  selectEl.visible = true
                } else {
                  currentInput.click()
                }
              }
            } else {
              shouldKeepOpen.value = false
            }
          } else {
            shouldKeepOpen.value = false
          }
        })
      }
    }, 100)
  }
}

// 处理值变化
function handleChange(values: any[]): void {
  selectedValues.value = values
  
  const shouldClose = maxCount.value > 0 && values.length >= maxCount.value
  if (!shouldClose) {
    shouldKeepOpen.value = true
  } else {
    shouldKeepOpen.value = false
  }
}

// 初始化：如果字段没有值，使用默认值
watch(
  () => props.value,
  (newValue: any) => {
    if (!newValue || !newValue.raw) {
      const defaultValue = config.value.default
      if (Array.isArray(defaultValue) && defaultValue.length > 0) {
        selectedValues.value = defaultValue
      }
    }
  },
  { immediate: true }
)

// 初始化：如果有回调接口且有初始值，触发一次 by_value 查询来加载选项
const hasInitialized = ref(false)
watch(
  () => [hasRemoteSearch.value, props.value?.raw],
  ([hasCallback, rawValue]: [boolean, any]) => {
    if (!hasInitialized.value && hasCallback && rawValue) {
      const values = parseRawValue(rawValue)
      if (values.length > 0) {
        hasInitialized.value = true
        handleSearch(values, true)
      }
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.multiselect-widget {
  width: 100%;
}

.response-multiselect,
.table-cell-multiselect,
.detail-multiselect {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.table-cell-multiselect .tag-item,
.detail-multiselect .tag-item {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  margin: 0;
}

/* 自定义颜色的 tag，确保文字清晰 */
.table-cell-multiselect .tag-item[style*="background-color"],
.detail-multiselect .tag-item[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
}

/* 标准颜色的 tag，增强对比度 */
.table-cell-multiselect .tag-item.el-tag--success,
.table-cell-multiselect .tag-item.el-tag--warning,
.table-cell-multiselect .tag-item.el-tag--danger,
.table-cell-multiselect .tag-item.el-tag--info,
.table-cell-multiselect .tag-item.el-tag--primary,
.detail-multiselect .tag-item.el-tag--success,
.detail-multiselect .tag-item.el-tag--warning,
.detail-multiselect .tag-item.el-tag--danger,
.detail-multiselect .tag-item.el-tag--info,
.detail-multiselect .tag-item.el-tag--primary {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.response-multiselect .tag-item {
  margin-right: 4px;
}

.empty-text {
  color: #999;
}

/* 编辑模式下的自定义标签样式 */
.multiselect-tag {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  margin-right: 6px;
  margin-bottom: 2px;
}

/* 自定义颜色的 tag，确保文字清晰 */
.multiselect-tag[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
}

/* 🔥 确保 el-tag 的 color 属性正确应用（通过内联样式） */
.multiselect-tag.el-tag {
  /* 确保自定义颜色能够正确显示 */
  /* Element Plus 的 el-tag 组件会自动将 color 属性转换为内联样式 */
}

/* 标准颜色的 tag，增强对比度 */
.multiselect-tag.el-tag--success,
.multiselect-tag.el-tag--warning,
.multiselect-tag.el-tag--danger,
.multiselect-tag.el-tag--info,
.multiselect-tag.el-tag--primary {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* 🔥 下拉选项中的颜色指示器样式（参考 Element Plus 官方示例） */
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
}

/* 选项容器样式 */
.flex {
  display: flex;
}

.items-center {
  align-items: center;
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

/* 🔥 全局样式：确保下拉选项中的颜色指示器正确显示 */
.select-dropdown-popper .option-color-indicator {
  display: inline-block !important;
  width: 12px !important;
  height: 12px !important;
  min-width: 12px !important;
  min-height: 12px !important;
  border-radius: 2px !important;
  flex-shrink: 0 !important;
  border: none !important;
  vertical-align: middle !important;
  /* 🔥 注意：background-color 通过内联样式设置，这里不设置，避免覆盖 */
}
</style>
