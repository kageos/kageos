<!--
  MultiSelectWidget - 多选组件
  重写版本，简化逻辑，修复标签显示问题
-->
<template>
  <div class="multiselect-widget">
    <!-- 编辑模式 -->
    <div v-if="mode === 'edit'" class="edit-multiselect">
      <!-- 参考单选的展示效果，使用条目式显示 -->
      <div class="select-container" @click="openDialog">
        <div class="select-content">
          <!-- 显示已选条目 -->
          <div v-if="selectedValues.length > 0" class="selected-items-list">
            <div
              v-for="(value, index) in selectedValues"
          :key="value"
              class="selected-item"
            >
              <div class="item-main">
                <span class="item-label">{{ getOptionLabel(value) }}</span>
                <el-icon class="item-close-icon" @click.stop="handleRemoveTag(value)">
                  <Close />
                </el-icon>
              </div>
              <div v-if="getItemDisplayInfo(value)" class="item-display-info">
                {{ getItemDisplayInfo(value) }}
              </div>
            </div>
          </div>
          <!-- 显示占位符 -->
          <div v-else class="select-main">
            <span class="select-label">{{ placeholder }}</span>
            <el-icon class="input-icon"><ArrowDown /></el-icon>
          </div>
          <!-- 显示总体 display_info -->
          <div v-if="selectedValues.length > 0 && displayInfoText" class="display-info-text">
            {{ displayInfoText }}
          </div>
        </div>
      </div>
      
      <!-- 模糊搜索对话框 -->
      <FuzzySearchDialog
        v-model="dialogVisible"
        :title="`选择${props.field.name}`"
        :placeholder="placeholder"
        :suggestions="dialogSuggestions"
        :loading="loading"
        :is-multiselect="true"
        :max-selections="maxCount"
        :selected-values="selectedValues"
        :get-item-color="getOptionColor"
        @search="handleDialogSearch"
        @select-multiple="handleDialogSelectMultiple"
        @select-all="handleDialogSelectAll"
      />
        </div>
    
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
import { computed, ref, watch, nextTick, onMounted, withDefaults } from 'vue'
import { ElInput, ElTag, ElIcon } from 'element-plus'
import { ArrowDown, Close } from '@element-plus/icons-vue'
import FuzzySearchDialog from './FuzzySearchDialog.vue'
import type { WidgetComponentProps } from '../types'
import { selectFuzzy } from '@/api/function'
import { Logger } from '../../utils/logger'
import { useFormDataStore } from '../../stores-v2/formData'
import { ExpressionParser } from '../../utils/ExpressionParser'
import { isStringDataType, getMultiSelectDefaultDataType, DataType } from '../../constants/widget'
import { SelectFuzzyQueryType, isStandardColor } from '../../constants/select'
import { convertValueToType } from '../utils/valueConverter'
import { createFieldValue } from '../utils/createFieldValue'

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
  const basePlaceholder = config.value.placeholder || `请选择${props.field.name}`
  // 🔥 如果有限制，在 placeholder 中显示最多可选数量
  if (maxCount.value > 0) {
    return `${basePlaceholder}（最多可选${maxCount.value}个）`
  }
  return basePlaceholder
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

// 对话框相关状态
const dialogVisible = ref(false)
const dialogSuggestions = ref<Array<{ label: string; value: any; displayInfo?: any; icon?: string }>>([])

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
    // 先转换为字符串数组用于内部处理（查找 options、显示等）
    let stringValues = newValues.map(v => String(v))
    
    // 🔥 去重：移除重复的值
    stringValues = Array.from(new Set(stringValues))
    
    if (maxCount.value > 0 && stringValues.length > maxCount.value) {
      Logger.warn('MultiSelectWidget', `${props.field.code} 超出数量限制! 限制: ${maxCount.value}, 实际: ${stringValues.length}`)
      stringValues = stringValues.slice(0, maxCount.value)
    }
    
    const displayInfos = stringValues.map((val: any) => {
      const option = options.value.find((opt: any) => String(opt.value) === val)
      return option?.displayInfo || null
    })
    
    const displayText = stringValues.map((val: any) => {
      const option = options.value.find((opt: any) => String(opt.value) === val)
      return option?.label || String(val)
    }).join(', ')
    
    // 🔥 计算行内聚合统计（如果有 statistics 配置）
    const rowStatistics = calculateRowStatistics(displayInfos, currentStatistics.value)
    
    /**
     * 🔥 根据 field.data.type 决定 raw 的格式和类型
     * - 如果 type 是 string：提交逗号分隔的字符串（如 "紧急,低优先级"）
     * - 如果 type 是 []string：提交字符串数组（如 ["紧急", "低优先级"]）
     * - 如果 type 是 []int：提交整数数组（如 [1, 2]）
     * - 如果 type 是 []float：提交浮点数数组（如 [1.5, 2.3]）
     */
    let rawValue: any
    const dataType = fieldDataType.value
    
    if (isStringDataType(dataType)) {
      // 提交逗号分隔的字符串
      rawValue = stringValues.length > 0 ? stringValues.join(',') : ''
    } else {
      // 提交数组，需要根据数组元素类型进行转换
      if (dataType.startsWith('[]')) {
        const elementType = dataType.slice(2) // 去掉 '[]' 前缀，得到元素类型
        rawValue = stringValues.map((val: string) => {
          // 根据元素类型转换
          if (elementType === 'int' || elementType === 'integer') {
            const intVal = parseInt(val, 10)
            return isNaN(intVal) ? val : intVal
          } else if (elementType === 'float' || elementType === 'number') {
            const floatVal = parseFloat(val)
            return isNaN(floatVal) ? val : floatVal
          } else if (elementType === 'bool' || elementType === 'boolean') {
            const strVal = String(val)
            return strVal === 'true' || strVal === '1' || (typeof val === 'boolean' && val === true)
          } else {
            // 默认保持字符串
            return val
          }
        })
      } else {
        // 未知的数组类型，保持字符串数组
        rawValue = stringValues
      }
    }
    
    // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
    const fieldValue = createFieldValue(
      props.field,
      rawValue,
      displayText || '未选择',
      {
        displayInfo: displayInfos,
        statistics: currentStatistics.value,
        rowStatistics: rowStatistics
      }
    )
    
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

// 获取 display_info 的显示文本（用于输入框下方显示）
const displayInfoText = computed(() => {
  if (selectedValues.value.length === 0) {
    return ''
  }
  
  // 🔥 优先从 props.value.meta.displayInfo 获取（这是保存的值）
  const metaDisplayInfos = props.value?.meta?.displayInfo
  if (metaDisplayInfos && Array.isArray(metaDisplayInfos) && metaDisplayInfos.length > 0) {
    const infoItems: string[] = []
    metaDisplayInfos.forEach((info: any) => {
      if (info && typeof info === 'object') {
        Object.entries(info).forEach(([key, value]) => {
          if (value !== null && value !== undefined && value !== '') {
            const item = `${key}: ${value}`
            if (!infoItems.includes(item)) {
              infoItems.push(item)
            }
          }
        })
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
  
  // 🔥 如果 meta 中没有，尝试从 options 中查找
  const displayInfos = selectedValues.value.map((val: any) => {
    const option = options.value.find((opt: any) => String(opt.value) === String(val))
    return option?.displayInfo || null
  }).filter((info: any) => info && typeof info === 'object')
  
  if (displayInfos.length === 0) {
    return ''
  }
  
  // 提取所有 display_info 的键值对，格式化为文本
  const infoItems: string[] = []
  displayInfos.forEach((info: any) => {
    if (info && typeof info === 'object') {
      Object.entries(info).forEach(([key, value]) => {
        if (value !== null && value !== undefined && value !== '') {
          const item = `${key}: ${value}`
          if (!infoItems.includes(item)) {
            infoItems.push(item)
          }
        }
      })
    }
  })
  
  // 限制显示数量，避免过长
  if (infoItems.length > 5) {
    return infoItems.slice(0, 5).join(' | ') + ' ...'
  }
  
  return infoItems.join(' | ')
})

// 获取单个条目的 display_info 文本
function getItemDisplayInfo(value: any): string {
  const valueStr = String(value)
  // 从 options 中查找
  const option = options.value.find((opt: any) => String(opt.value) === valueStr)
  if (option?.displayInfo && typeof option.displayInfo === 'object') {
    const infoItems: string[] = []
    Object.entries(option.displayInfo).forEach(([key, val]) => {
      if (val !== null && val !== undefined && val !== '') {
        infoItems.push(`${key}: ${val}`)
      }
    })
    if (infoItems.length > 0) {
      // 限制显示数量，避免过长
      if (infoItems.length > 3) {
        return infoItems.slice(0, 3).join(' | ') + ' ...'
      }
      return infoItems.join(' | ')
    }
  }
  return ''
}

// 获取选项标签
function getOptionLabel(value: any): string {
  if (value === null || value === undefined) return ''
  
  const valueStr = String(value)
  
  // 1. 优先从 options 中查找
  const option = options.value.find((opt: any) => String(opt.value) === valueStr)
  if (option) {
    return option.label
  }
  
  // 2. 如果 options 中没有，尝试从 props.value.display 中解析
  // 注意：display 是逗号分隔的字符串，我们需要知道每个值对应的 label
  // 由于 display 是聚合的，我们无法直接解析单个值的 label
  // 所以这里还是返回 valueStr，但会在初始化时通过 by_values 查询来加载 labels
  
  // 3. 如果还没有，返回 valueStr（作为后备）
  return valueStr
}

/**
 * 移除指定值
 */
function handleRemoveValue(value: any): void {
  const newValues = selectedValues.value.filter((v: any) => String(v) !== String(value))
  selectedValues.value = newValues
}

/**
 * 判断是否是 Element Plus 标准颜色类型
 */
// isStandardColor 已从 constants/select 导入

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
    return optionColors.value[optionIndex]
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
  return isStandard ? color : undefined
}

/**
 * 获取选项的颜色值（用于 el-tag 的 color 属性）
 * 🔥 注意：el-tag 的 color 属性只接受自定义颜色（hex），标准颜色使用 type 属性
 */
function getOptionColorValue(value: any): string | undefined {
  const color = getOptionColor(value)
  if (!color) {
    return undefined
  }
  const isStandard = isStandardColor(color)
  return !isStandard ? color : undefined
}

/**
 * 🔥 获取选项的颜色样式对象（用于 span 的 style 绑定）
 */
function getOptionColorStyle(value: any): Record<string, string> {
  const colorValue = getOptionColorValue(value)
  const color = getOptionColor(value)
  const backgroundColor = colorValue || color || ''
  
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
  
  // 🔥 对于 selected() 函数，使用第一个选中项的 DisplayInfo
  const firstSelectedItem = validDisplayInfos.length > 0 ? validDisplayInfos[0] : null
  
  const result: Record<string, any> = {}
  
  try {
    for (const [label, expression] of Object.entries(statisticsConfig)) {
      try {
        // 🔥 传递 selectedItem 参数，用于 selected() 函数
        const value = ExpressionParser.evaluate(expression, validDisplayInfos, firstSelectedItem)
        result[label] = value
      } catch (error: any) {
        Logger.error(`[MultiSelectWidget] 行内聚合计算失败: ${label} = ${expression}`, error)
        result[label] = 0
      }
    }
  } catch (error: any) {
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
      queryType = Array.isArray(query) ? SelectFuzzyQueryType.BY_VALUES : SelectFuzzyQueryType.BY_VALUE
    } else {
      queryType = SelectFuzzyQueryType.BY_KEYWORD
    }
    
    // 🔥 对于 by_values 查询，需要确保传递的值类型正确
    // 如果 field.data.type 是 []int，query 应该是整数数组
    let queryValue: any = query
    if (isByValue && Array.isArray(query)) {
      const dataType = props.field.data?.type || getMultiSelectDefaultDataType()
      if (dataType.startsWith('[]')) {
        const elementType = dataType.slice(2)
        if (elementType === 'int' || elementType === 'integer') {
          queryValue = query.map((v: any) => {
            const intVal = parseInt(String(v), 10)
            return isNaN(intVal) ? v : intVal
          })
        } else if (elementType === 'float' || elementType === 'number') {
          queryValue = query.map((v: any) => {
            const floatVal = parseFloat(String(v))
            return isNaN(floatVal) ? v : floatVal
          })
        }
      }
    }
    
    const requestBody = {
      code: props.field.code,
      type: queryType,
      value: queryValue,
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

// 远程搜索方法（保留用于兼容）
async function remoteMethod(query: string): Promise<void> {
  await handleSearch(query, false)
}

// 打开对话框
async function openDialog(): Promise<void> {
  dialogVisible.value = true
  // 如果有远程搜索，触发一次空搜索加载初始选项
  if (hasRemoteSearch.value) {
    await handleDialogSearch('')
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
  if (hasRemoteSearch.value) {
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

// 处理对话框多选确认
function handleDialogSelectMultiple(items: Array<{ value: any; label?: string; displayInfo?: any }>): void {
  const newValues = items.map(item => item.value)
  // 合并已选值和新增值，去重
  const allValues = Array.from(new Set([...selectedValues.value, ...newValues]))
  
  // 🔥 更新 options，确保新选择的项的 displayInfo 被保存
  items.forEach(item => {
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
  })
  
  // 应用数量限制
  if (maxCount.value > 0 && allValues.length > maxCount.value) {
    const limitedValues = allValues.slice(0, maxCount.value)
    selectedValues.value = limitedValues
              } else {
    selectedValues.value = allValues
  }
  
  // 🔥 关闭对话框
  dialogVisible.value = false
}

// 处理对话框全选
function handleDialogSelectAll(items: Array<{ value: any; label?: string; displayInfo?: any }>): void {
  const newValues = items.map(item => item.value)
  // 合并已选值和全选值，去重
  const allValues = Array.from(new Set([...selectedValues.value, ...newValues]))
  
  // 🔥 更新 options，确保全选的项的 displayInfo 被保存
  items.forEach(item => {
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
  })
  
  // 应用数量限制
  if (maxCount.value > 0 && allValues.length > maxCount.value) {
    const limitedValues = allValues.slice(0, maxCount.value)
    selectedValues.value = limitedValues
          } else {
    selectedValues.value = allValues
          }
  
  // 🔥 关闭对话框
  dialogVisible.value = false
}

// 移除标签时触发
function handleRemoveTag(valueToRemove?: any): void {
  if (valueToRemove !== undefined) {
    // 🔥 从 selectedValues 中移除指定值
    const newValues = selectedValues.value.filter((v: any) => String(v) !== String(valueToRemove))
    selectedValues.value = newValues
  }
}

// 处理值变化（保留用于兼容）
function handleChange(values: any[]): void {
  selectedValues.value = values
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

// 初始化：如果有回调接口且有初始值，触发一次 by_values 查询来加载选项
const hasInitialized = ref(false)
const lastSearchedValues = ref<string[]>([])

// 在 onMounted 中处理，确保 formRenderer 已经传递过来
onMounted(() => {
  // 🔥 如果有回调接口且有初始值，立即触发一次回调
  // 因为 watch 可能在组件挂载时 formRenderer 还没传递过来
  if (hasRemoteSearch.value && props.value?.raw && props.formRenderer) {
    nextTick(() => {
      const values = parseRawValue(props.value?.raw)
      if (values.length > 0 && !hasInitialized.value) {
        hasInitialized.value = true
        lastSearchedValues.value = values
        handleSearch(values, true)
      }
    })
  }
})

// 监听 formRenderer 和 value 变化，确保在 formRenderer 准备好后触发回调
watch(
  () => [hasRemoteSearch.value, props.value?.raw, props.formRenderer],
  ([hasCallback, rawValue, formRenderer]: [boolean, any, any]) => {
    if (!hasInitialized.value && hasCallback && rawValue && formRenderer) {
      const values = parseRawValue(rawValue)
      if (values.length > 0) {
        // 检查是否已经搜索过这些值
        const valuesStr = values.sort().join(',')
        const lastSearchedStr = lastSearchedValues.value.sort().join(',')
        if (valuesStr !== lastSearchedStr) {
          hasInitialized.value = true
          lastSearchedValues.value = values
          handleSearch(values, true)
        }
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

.edit-multiselect {
  position: relative;
  width: 100%;
}

/* 🔥 参考单选的样式，使用相同的容器样式 */
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
  min-height: 24px;
}

.select-label {
  flex: 1;
  color: var(--el-text-color-placeholder);
  font-size: 14px;
  line-height: 1.5;
}

/* 条目列表样式 */
.selected-items-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.selected-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 8px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 4px;
  border: 1px solid var(--el-border-color-lighter);
  transition: all 0.2s;
}

.selected-item:hover {
  background-color: var(--el-fill-color-light);
  border-color: var(--el-border-color);
}

.item-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.item-label {
  flex: 1;
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.5;
  font-weight: 500;
}

.item-close-icon {
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
  flex-shrink: 0;
}

.item-close-icon:hover {
  color: var(--el-color-danger);
  transform: scale(1.1);
}

.item-display-info {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  padding-left: 4px;
}

.display-info-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  margin: 0;
  opacity: 0.9;
}

/* 自定义颜色的 tag，确保文字清晰 */
.table-cell-multiselect .tag-item[style*="background-color"],
.detail-multiselect .tag-item[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
  /* 🔥 降低亮度：使用 filter 降低饱和度和亮度 */
  filter: brightness(0.95) saturate(0.9);
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
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  opacity: 0.9;
}

.response-multiselect .tag-item {
  margin-right: 4px;
}

.empty-text {
  color: #999;
}

/* 编辑模式下的自定义标签样式 - 参考单选的标签样式 */
.multiselect-tag {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  margin: 0;
  opacity: 0.9;
  transition: all 0.2s;
  font-size: 12px;
  padding: 2px 8px;
  height: 22px;
  line-height: 1.5;
}

.multiselect-tag:hover {
  opacity: 1;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12);
}

/* 自定义颜色的 tag，确保文字清晰 */
.multiselect-tag[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
  /* 🔥 降低亮度：使用 filter 降低饱和度 */
  filter: brightness(0.95) saturate(0.9);
}

/* 🔥 确保 el-tag 的 color 属性正确应用（通过内联样式） */
/* Element Plus 的 el-tag 组件会自动将 color 属性转换为内联样式 */

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
  /* 🔥 降低亮度：使用 filter 降低饱和度和亮度 */
  filter: brightness(0.95) saturate(0.9);
  opacity: 0.9;
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
