<!--
  MultiSelectWidget - 多选组件
  简洁版本，专注于核心功能
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
      collapse-tags
      :max-collapse-tags="3"
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
      <el-option
        v-for="option in options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
        @click="handleOptionClick"
      />
    </el-select>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-multiselect">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
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

// 选中的值（数组）
const selectedValues = computed({
  get: () => {
    const raw = props.value?.raw
    if (Array.isArray(raw)) {
      return raw
    }
    if (typeof raw === 'string' && raw) {
      return [raw]
    }
    return []
  },
  set: (newValues: any[]) => {
    let finalValues = newValues
    if (maxCount.value > 0 && finalValues.length > maxCount.value) {
      Logger.warn('MultiSelectWidget', `${props.field.code} 超出数量限制! 限制: ${maxCount.value}, 实际: ${finalValues.length}`)
      finalValues = finalValues.slice(0, maxCount.value)
    }
    
    const displayInfos = finalValues.map((val: any) => {
      const option = options.value.find((opt: any) => opt.value === val)
      return option?.displayInfo || null
    })
    
    const displayText = finalValues.map((val: any) => {
      const option = options.value.find((opt: any) => opt.value === val)
      return option?.label || String(val)
    }).join(', ')
    
    // 🔥 计算行内聚合统计（如果有 statistics 配置）
    const rowStatistics = calculateRowStatistics(displayInfos, currentStatistics.value)
    
    const fieldValue = {
      raw: finalValues,
      display: displayText || '未选择',
      meta: {
        displayInfo: displayInfos,
        statistics: currentStatistics.value,
        rowStatistics: rowStatistics  // 🔥 保存行内聚合结果
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
  const raw = props.value?.raw
  if (Array.isArray(raw)) {
    return raw
  }
  if (typeof raw === 'string' && raw) {
    return [raw]
  }
  return []
})

// 获取选项标签
function getOptionLabel(value: any): string {
  const option = options.value.find((opt: any) => opt.value === value)
  return option ? option.label : String(value)
}

/**
 * 🔥 计算行内聚合统计（MultiSelect 自己的职责）
 * 使用选中的选项的 displayInfo 和 statistics 配置来计算
 */
function calculateRowStatistics(
  displayInfos: any[],
  statisticsConfig: Record<string, string> | null
): Record<string, any> {
  if (!statisticsConfig || Object.keys(statisticsConfig).length === 0) {
    return {}
  }
  
  // 过滤掉 null 的 displayInfo
  const validDisplayInfos = displayInfos.filter(info => info && typeof info === 'object')
  
  if (validDisplayInfos.length === 0) {
    return {}
  }
  
  const result: Record<string, any> = {}
  
  try {
    // 遍历统计配置，计算每个统计项
    for (const [label, expression] of Object.entries(statisticsConfig)) {
      try {
        // 使用表达式解析器计算（使用 displayInfo 数组作为数据源）
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
    const queryType: 'by_keyword' | 'by_value' = isByValue ? 'by_value' : 'by_keyword'
    const requestBody = {
      code: props.field.code,
      type: queryType,
      value: query,
      request: props.formRenderer?.getSubmitData?.() || {},
      value_type: props.field.data?.type || '[]string'
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
  // 🔥 搜索时保持下拉框打开状态（不清除 shouldKeepOpen）
  // 但搜索完成后，如果用户没有继续操作，应该允许关闭
  await handleSearch(query, false)
  // 搜索完成后，如果下拉框仍然打开，保持 shouldKeepOpen 状态
}

// 选项点击时触发 - 提前设置标志
function handleOptionClick(): void {
  // 🔥 提前设置标志，确保在 handleVisibleChange 之前生效
  const currentLength = selectedValues.value.length
  const shouldClose = maxCount.value > 0 && currentLength >= maxCount.value - 1
  if (!shouldClose) {
    shouldKeepOpen.value = true
  } else {
    // 如果已达到最大数量，清除标志，允许关闭
    shouldKeepOpen.value = false
  }
}

// 移除标签时触发
function handleRemoveTag(): void {
  // 移除标签时也保持打开（因为用户可能想继续选择）
  shouldKeepOpen.value = true
}

// 下拉框展开时触发
function handleVisibleChange(visible: boolean): void {
  if (visible) {
    // 下拉框打开时，根据当前选择数量决定是否需要保持打开
    const currentLength = selectedValues.value.length
    const shouldClose = maxCount.value > 0 && currentLength >= maxCount.value
    if (!shouldClose) {
      shouldKeepOpen.value = true
    } else {
      shouldKeepOpen.value = false
    }
    
    // 如果有远程搜索，且选项为空，触发初始搜索
    if (hasRemoteSearch.value) {
      if (dynamicOptions.value.length === 0) {
        handleSearch('', false)
      }
    }
  } else {
    // 下拉框关闭时
    // 🔥 关键：只有在选择选项时才保持打开，用户点击外部或按 ESC 时应该关闭
    // 延迟检查，给用户操作时间（点击选项后可能会触发关闭事件）
    setTimeout(() => {
      // 如果不需要保持打开，直接清除标志并允许关闭
      if (!shouldKeepOpen.value) {
        return
      }
      
      // 检查焦点是否还在输入框
      const input = selectRef.value?.$el?.querySelector('input')
      const isInputFocused = document.activeElement === input
      
      // 如果焦点不在输入框，说明用户想关闭（点击外部或按 ESC），清除标志并允许关闭
      if (!isInputFocused) {
        shouldKeepOpen.value = false
        return
      }
      
      // 如果是选择后需要保持打开，且焦点还在输入框，阻止关闭
      if (shouldKeepOpen.value && isInputFocused) {
        // 阻止关闭：通过 DOM 操作重新打开下拉框
        nextTick(() => {
          if (selectRef.value) {
            const selectEl = selectRef.value as any
            const currentInput = (selectEl.$el || selectEl.el || selectEl)?.querySelector?.('input')
            if (currentInput && document.activeElement === currentInput) {
              // 重新打开下拉框：尝试多种方式
              currentInput.focus()
              // 方法1：使用 Element Plus Select 的内部方法
              if (selectEl.handleMenuEnter) {
                selectEl.handleMenuEnter()
              } else if (selectEl.toggleMenu) {
                selectEl.toggleMenu()
              } else if (selectEl.setSoftFocus) {
                selectEl.setSoftFocus()
              } else {
                // 方法2：直接设置 visible 属性（如果存在）
                if (selectEl.visible !== undefined) {
                  selectEl.visible = true
                } else {
                  // 方法3：触发点击事件
                  currentInput.click()
                }
              }
            } else {
              // 如果焦点不在输入框，清除标志
              shouldKeepOpen.value = false
            }
          } else {
            // 如果组件引用不存在，清除标志
    shouldKeepOpen.value = false
          }
        })
      }
    }, 100)
  }
}

// 处理值变化 - 阻止下拉框关闭
function handleChange(values: any[]): void {
  // 先更新值
  selectedValues.value = values
  
  // 设置标志
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
// 🔥 注意：只在组件初始化时触发，用户选择后不会触发
const hasInitialized = ref(false)
watch(
  () => [hasRemoteSearch.value, props.value?.raw],
  ([hasCallback, rawValue]: [boolean, any]) => {
    // 只在首次初始化时触发，避免用户选择后触发
    if (!hasInitialized.value && hasCallback && rawValue && Array.isArray(rawValue) && rawValue.length > 0) {
      hasInitialized.value = true
      handleSearch(rawValue, true)
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
  gap: 4px;
}

.tag-item {
  margin-right: 4px;
}

.empty-text {
  color: #999;
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
