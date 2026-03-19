<!--
  TimestampWidget - 时间戳组件
  🔥 完全新增，不依赖旧代码
-->

<template>
  <div class="timestamp-widget">
    <!-- 编辑模式：format 为 HH:mm 时用 el-time-picker，否则用 el-date-picker -->
    <el-time-picker
      v-if="mode === 'edit' && isTimeOnly"
      v-model="internalValue"
      :disabled="widgetConfig.disabled"
      :placeholder="field.desc || `请选择${field.name}`"
      :format="format"
      value-format="x"
      :clearable="true"
      @change="handleChange"
    />
    <el-date-picker
      v-else-if="mode === 'edit'"
      v-model="internalValue"
      :disabled="widgetConfig.disabled"
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
      <div class="detail-content">{{ displayValue }}</div>
    </div>
    
    <!-- 搜索模式 -->
    <el-time-picker
      v-else-if="mode === 'search' && isTimeOnly"
      v-model="internalValue"
      :format="format"
      value-format="x"
      :clearable="true"
    />
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
import { ElDatePicker, ElTimePicker } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { formatTimestamp } from '@/utils/date'
import type { TimestampWidgetConfig } from '@/core/types/widget-configs'

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
  return (props.field.widget?.config || {}) as TimestampWidgetConfig
})

// 是否为纯时间模式（format 为 HH:mm 或 HH:mm:ss 时用 el-time-picker）
const isTimeOnly = computed(() => {
  const fmt = widgetConfig.value.format || 'YYYY-MM-DD HH:mm:ss'
  return fmt === 'HH:mm' || fmt === 'HH:mm:ss'
})

// 选择器类型（仅 el-date-picker 使用，datetime 或 datetimerange）
const pickerType = computed(() => 'datetime')

// 格式
const format = computed(() => {
  return widgetConfig.value.format || 'YYYY-MM-DD HH:mm:ss'
})

// 值格式（默认返回时间戳毫秒）
// 注意：当 valueFormat 为 'x' 时，Element Plus 返回数字（毫秒时间戳）
// 当 valueFormat 为字符串格式时，Element Plus 返回字符串
// 但是 v-model 绑定的值始终是 Date 对象（Element Plus 内部处理）
const valueFormat = computed(() => {
  return 'x'  // TimestampWidgetConfig 中没有 valueFormat 字段，使用默认值
})

// 快捷选择（纯时间选择器不显示日期类快捷方式）
const shortcuts = computed(() => {
  if (isTimeOnly.value) {
    return undefined
  }
  const showShortcuts = true
  if (!showShortcuts) {
    return undefined
  }
  
  const now = new Date()
  
  // 丰富的快捷选择选项
  return [
    // ========== 基础时间 ==========
    {
      text: '现在',
      value: () => new Date(now)
    },
    {
      text: '今天',
      value: () => {
        const date = new Date(now)
        date.setHours(0, 0, 0, 0)
        return date
      }
    },
    {
      text: '明天',
      value: () => {
        const date = new Date(now)
        date.setDate(date.getDate() + 1)
        date.setHours(0, 0, 0, 0)
        return date
      }
    },
    {
      text: '昨天',
      value: () => {
        const date = new Date(now)
        date.setDate(date.getDate() - 1)
        date.setHours(0, 0, 0, 0)
        return date
      }
    },
    // ========== 相对时间（此刻） ==========
    {
      text: '昨天此刻',
      value: () => new Date(now.getTime() - 24 * 60 * 60 * 1000)
    },
    {
      text: '明天此刻',
      value: () => new Date(now.getTime() + 24 * 60 * 60 * 1000)
    },
    // ========== 相对时间（小时） ==========
    {
      text: '一小时后',
      value: () => new Date(now.getTime() + 1 * 60 * 60 * 1000)
    },
    {
      text: '两小时后',
      value: () => new Date(now.getTime() + 2 * 60 * 60 * 1000)
    },
    {
      text: '三小时后',
      value: () => new Date(now.getTime() + 3 * 60 * 60 * 1000)
    },
    {
      text: '六小时后',
      value: () => new Date(now.getTime() + 6 * 60 * 60 * 1000)
    },
    {
      text: '十二小时后',
      value: () => new Date(now.getTime() + 12 * 60 * 60 * 1000)
    },
    {
      text: '一小时前',
      value: () => new Date(now.getTime() - 1 * 60 * 60 * 1000)
    },
    {
      text: '两小时前',
      value: () => new Date(now.getTime() - 2 * 60 * 60 * 1000)
    },
    {
      text: '三小时前',
      value: () => new Date(now.getTime() - 3 * 60 * 60 * 1000)
    },
    // ========== 相对时间（天） ==========
    {
      text: '一天后',
      value: () => new Date(now.getTime() + 24 * 60 * 60 * 1000)
    },
    {
      text: '两天后',
      value: () => new Date(now.getTime() + 2 * 24 * 60 * 60 * 1000)
    },
    {
      text: '三天后',
      value: () => new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000)
    },
    {
      text: '一周后',
      value: () => new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)
    },
    {
      text: '一个月后',
      value: () => new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000)
    },
    {
      text: '一天前',
      value: () => new Date(now.getTime() - 24 * 60 * 60 * 1000)
    },
    {
      text: '两天前',
      value: () => new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000)
    },
    {
      text: '一周前',
      value: () => new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
    },
    {
      text: '一个月前',
      value: () => new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
    },
    // ========== 相对时间（周） ==========
    {
      text: '下周一',
      value: () => {
        const date = new Date(now)
        const daysUntilNextMonday = (8 - now.getDay()) % 7 || 7
        date.setDate(now.getDate() + daysUntilNextMonday)
        date.setHours(0, 0, 0, 0)
        return date
      }
    },
    {
      text: '上周一',
      value: () => {
        const date = new Date(now)
        const daysSinceLastMonday = (now.getDay() + 6) % 7
        date.setDate(now.getDate() - daysSinceLastMonday - 7)
        date.setHours(0, 0, 0, 0)
        return date
      }
    },
    // ========== 相对时间（月） ==========
    {
      text: '下个月',
      value: () => {
        const date = new Date(now.getFullYear(), now.getMonth() + 1, 1)
        return date
      }
    },
    {
      text: '上个月',
      value: () => {
        const date = new Date(now.getFullYear(), now.getMonth() - 1, 1)
        return date
      }
    },
    // ========== 相对时间（年） ==========
    {
      text: '明年',
      value: () => {
        const date = new Date(now.getFullYear() + 1, 0, 1)
        return date
      }
    },
    {
      text: '去年',
      value: () => {
        const date = new Date(now.getFullYear() - 1, 0, 1)
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
      // 注意：系统统一使用毫秒级时间戳，直接使用
      if (typeof value === 'number') {
        return new Date(value)
      }
      
      // 如果是数组（范围选择）
      if (Array.isArray(value)) {
        return value.map(v => new Date(v))
      }
      
      return value
    }
    return null
  },
  set: (newValue: Date | [Date, Date] | number | [number, number] | string | [string, string] | null) => {
    if (props.mode === 'edit') {
      let rawValue: number | [number, number] | null = null
      
      if (newValue === null || newValue === undefined) {
        rawValue = null
      } else if (Array.isArray(newValue)) {
        // 范围选择：处理数组
        rawValue = newValue.map(v => {
          if (v instanceof Date) {
            return v.getTime()
          } else if (typeof v === 'number') {
            return v
          } else if (typeof v === 'string') {
            // 字符串可能是时间戳字符串或日期字符串
            const num = Number(v)
            if (!isNaN(num)) {
              return num
            }
            // 尝试解析日期字符串
            return new Date(v).getTime()
          }
          throw new Error(`[TimestampWidget] 无法转换值: ${v}`)
        }) as [number, number]
      } else {
        // 单个值
        if (newValue instanceof Date) {
        rawValue = newValue.getTime()
        } else if (typeof newValue === 'number') {
          rawValue = newValue
        } else if (typeof newValue === 'string') {
          // 字符串可能是时间戳字符串或日期字符串
          const num = Number(newValue)
          if (!isNaN(num)) {
            rawValue = num
          } else {
            // 尝试解析日期字符串
            rawValue = new Date(newValue).getTime()
          }
        } else {
          throw new Error(`[TimestampWidget] 无法转换值类型: ${typeof newValue}, 值: ${newValue}`)
        }
      }
      
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        rawValue,
        formatTimestamp(rawValue as number)
      )
      
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
  
  const raw = value.raw
  if (raw === null || raw === undefined) {
    return '-'
  }
  
  // 🔥 优先使用 raw 值进行格式化，确保时间戳字段始终被正确格式化
  // 即使 value.display 已经有值，也要重新格式化（因为可能是之前转换错误的值）
  if (typeof raw === 'number') {
    // 🔥 formatTimestamp 会自动判断秒级/毫秒级，直接调用即可
    return formatTimestamp(raw, props.field.widget?.config?.format)
  }
  
  if (Array.isArray(raw)) {
    return raw.map(v => formatTimestamp(v, props.field.widget?.config?.format)).join(' 至 ')
  }
  
  // 如果 raw 不是数字，尝试使用 display 值
  if (value.display) {
    return value.display
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


