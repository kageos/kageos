<!--
  TimestampWidget - 时间戳组件
  🔥 完全新增，不依赖旧代码
-->

<template>
  <div class="timestamp-widget">
    <!-- 编辑模式 -->
    <el-date-picker
      v-if="mode === 'edit'"
      v-model="internalValue"
      :disabled="field.widget?.config?.disabled"
      :placeholder="field.desc || `请选择${field.name}`"
      :type="pickerType"
      :format="format"
      :value-format="valueFormat"
      :clearable="true"
      :shortcuts="shortcuts"
      @change="handleChange"
    />
    
    <!-- 响应模式（只读） -->
    <span v-else-if="mode === 'response'" class="response-value">
      {{ displayValue }}
    </span>
    
    <!-- 表格单元格模式 -->
    <span v-else-if="mode === 'table-cell'" class="table-cell-value">
      {{ displayValue }}
    </span>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <div class="detail-label">{{ field.name }}</div>
      <div class="detail-content">{{ displayValue }}</div>
    </div>
    
    <!-- 搜索模式 -->
    <el-date-picker
      v-else-if="mode === 'search'"
      v-model="internalValue"
      :type="searchType"
      :format="format"
      :value-format="valueFormat"
      :clearable="true"
      :shortcuts="shortcuts"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElDatePicker } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { formatTimestamp } from '@/utils/date'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 选择器类型
const pickerType = computed(() => {
  return props.field.widget?.config?.type || 'datetime'
})

// 格式
const format = computed(() => {
  return props.field.widget?.config?.format || 'YYYY-MM-DD HH:mm:ss'
})

// 值格式（默认返回时间戳毫秒）
const valueFormat = computed(() => {
  return props.field.widget?.config?.valueFormat || 'x'
})

// 快捷选择
const shortcuts = computed(() => {
  const showShortcuts = props.field.widget?.config?.shortcuts !== false
  if (!showShortcuts) {
    return undefined
  }
  
  // 简单的快捷选择（可以根据需要扩展）
  return [
    {
      text: '今天',
      value: () => new Date()
    },
    {
      text: '昨天',
      value: () => {
        const date = new Date()
        date.setTime(date.getTime() - 3600 * 1000 * 24)
        return date
      }
    },
    {
      text: '一周前',
      value: () => {
        const date = new Date()
        date.setTime(date.getTime() - 3600 * 1000 * 24 * 7)
        return date
      }
    }
  ]
})

// 搜索类型
const searchType = computed(() => {
  if (props.searchType?.includes('gte') && props.searchType?.includes('lte')) {
    return 'datetimerange'
  }
  return 'datetime'
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const value = props.value?.raw
      if (value === null || value === undefined) {
        return null
      }
      
      // 🔥 如果是时间戳，转换为 Date 对象
      // 注意：系统统一使用毫秒级时间戳，但为了兼容性，需要判断
      if (typeof value === 'number') {
        // 🔥 自动判断时间戳是秒级还是毫秒级
        // 规则：如果时间戳 < 9999999999，认为是秒级，需要乘以 1000
        const SECONDS_THRESHOLD = 9999999999
        const timestamp = value > 0 && value < SECONDS_THRESHOLD ? value * 1000 : value
        return new Date(timestamp)
      }
      
      // 如果是数组（范围选择）
      if (Array.isArray(value)) {
        return value.map(v => new Date(v))
      }
      
      return value
    }
    return null
  },
  set: (newValue: Date | [Date, Date] | null) => {
    if (props.mode === 'edit') {
      let rawValue: number | [number, number] | null = null
      
      if (newValue === null) {
        rawValue = null
      } else if (Array.isArray(newValue)) {
        rawValue = newValue.map(v => v.getTime())
      } else {
        rawValue = newValue.getTime()
      }
      
      const newFieldValue = {
        raw: rawValue,
        display: formatTimestamp(rawValue as number),
        meta: {}
      }
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 显示值
const displayValue = computed(() => {
  const value = props.value
  if (!value) {
    return '-'
  }
  
  if (value.display) {
    return value.display
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined) {
    return '-'
  }
  
  // 格式化时间戳
  if (typeof raw === 'number') {
    // 🔥 formatTimestamp 会自动判断秒级/毫秒级，直接调用即可
    return formatTimestamp(raw, props.field.widget?.config?.format)
  }
  
  if (Array.isArray(raw)) {
    return raw.map(v => formatTimestamp(v)).join(' 至 ')
  }
  
  return String(raw)
})

// 处理值变化
function handleChange(value: Date | [Date, Date] | null): void {
  // 已经在 computed setter 中处理
}
</script>

<style scoped>
.timestamp-widget {
  width: 100%;
}

.response-value {
  color: var(--el-text-color-regular);
}

.table-cell-value {
  color: var(--el-text-color-regular);
}

.detail-value {
  margin-bottom: 16px;
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

