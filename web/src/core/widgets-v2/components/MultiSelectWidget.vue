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
    
    const fieldValue = {
      raw: finalValues,
      display: displayText || '未选择',
      meta: {
        displayInfo: displayInfos,
        statistics: currentStatistics.value
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
  await handleSearch(query, false)
}

// 选项点击时触发 - 提前设置标志
function handleOptionClick(): void {
  // 🔥 提前设置标志，确保在 handleVisibleChange 之前生效
  const currentLength = selectedValues.value.length
  const shouldClose = maxCount.value > 0 && currentLength >= maxCount.value - 1
  if (!shouldClose) {
    shouldKeepOpen.value = true
  }
}

// 移除标签时触发
function handleRemoveTag(): void {
  // 移除标签时也保持打开
  shouldKeepOpen.value = true
}

// 下拉框展开时触发
function handleVisibleChange(visible: boolean): void {
  // 🔥 关键：如果是因为选择而需要保持打开，但下拉框要关闭了，阻止关闭
  if (!visible && shouldKeepOpen.value) {
    // 阻止关闭：通过 DOM 操作重新打开下拉框
    nextTick(() => {
      if (selectRef.value) {
        const selectEl = selectRef.value as any
        const input = (selectEl.$el || selectEl.el || selectEl)?.querySelector?.('input')
        if (input) {
          input.focus()
          // 触发点击事件来打开下拉框
          const clickEvent = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
          input.dispatchEvent(clickEvent)
          setTimeout(() => {
            input.click()
          }, 10)
        }
      }
    })
    return
  }
  
  // 下拉框打开时，默认设置标志（为第一次选择做准备）
  if (visible) {
    const currentLength = selectedValues.value.length
    const shouldClose = maxCount.value > 0 && currentLength >= maxCount.value
    if (!shouldClose) {
      shouldKeepOpen.value = true
    }
    
    if (hasRemoteSearch.value) {
      if (dynamicOptions.value.length === 0) {
        handleSearch('', false)
      }
    }
  } else {
    // 用户主动关闭，清除标志
    shouldKeepOpen.value = false
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
