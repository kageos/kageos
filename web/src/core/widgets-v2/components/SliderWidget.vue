<!--
  SliderWidget - 滑块/进度条组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 输入模式：显示为滑块（slider bar）
  - 输出模式：显示为进度条（progress bar），自动显示百分比和状态颜色
  - 搜索模式：范围搜索（两个输入框：最小值、最大值）
-->

<template>
  <div class="slider-widget">
    <!-- 编辑模式：滑块 -->
    <el-slider
      v-if="mode === 'edit'"
      v-model="internalValue"
      :min="min"
      :max="max"
      :step="step"
      :show-tooltip="true"
      :format-tooltip="formatTooltipFunc"
      :disabled="field.widget?.config?.disabled"
      @change="handleChange"
    />
    
    <!-- 响应模式（只读） -->
    <span v-else-if="mode === 'response'" class="response-value">
      {{ displayValue }}
    </span>
    
    <!-- 表格单元格模式：进度条 -->
    <el-progress
      v-else-if="mode === 'table-cell'"
      :percentage="percentage"
      :status="autoStatus"
      :stroke-width="6"
      :show-text="true"
      :format="formatProgress"
    />
    
    <!-- 详情模式：进度条 -->
    <el-progress
      v-else-if="mode === 'detail'"
      :percentage="percentage"
      :status="autoStatus"
      :stroke-width="6"
      :show-text="true"
      :format="formatProgress"
    />
    
    <!-- 搜索模式：范围输入 -->
    <div v-else-if="mode === 'search'" class="slider-search">
      <el-input-number
        v-model="minValue"
        :min="min"
        :max="max"
        :step="step"
        :placeholder="`最小${field.name}`"
        :precision="stepPrecision"
        @change="handleSearchChange"
      />
      <span class="separator">-</span>
      <el-input-number
        v-model="maxValue"
        :min="min"
        :max="max"
        :step="step"
        :placeholder="`最大${field.name}`"
        :precision="stepPrecision"
        @change="handleSearchChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElSlider, ElProgress, ElInputNumber } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 配置
const config = computed(() => props.field.widget?.config || {})

// 最小值、最大值、步长、单位
const min = computed(() => {
  const minValue = config.value.min
  if (minValue !== undefined && minValue !== null) {
    const num = Number(minValue)
    return isNaN(num) ? 0 : num
  }
  return 0 // 默认0
})

const max = computed(() => {
  const maxValue = config.value.max
  if (maxValue !== undefined && maxValue !== null) {
    const num = Number(maxValue)
    return isNaN(num) ? 100 : num
  }
  return 100 // 默认100
})

const step = computed(() => {
  const stepValue = config.value.step
  if (stepValue !== undefined && stepValue !== null) {
    const num = Number(stepValue)
    return isNaN(num) ? 1 : num
  }
  return 1 // 默认1
})

const unit = computed(() => config.value.unit || '')

// 计算步长的小数位数（用于 input-number 的 precision）
const stepPrecision = computed(() => {
  const stepStr = String(step.value)
  if (stepStr.includes('.')) {
    return stepStr.split('.')[1].length
  }
  return 0
})

// 默认值
const defaultValue = computed(() => {
  const def = config.value.default
  if (def !== undefined && def !== null) {
    const num = Number(def)
    return isNaN(num) ? undefined : num
  }
  return undefined
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit') {
      const value = props.value?.raw
      if (value !== null && value !== undefined) {
        return Number(value)
      }
      // 如果没有值且有默认值，返回默认值
      if (defaultValue.value !== undefined) {
        return defaultValue.value
      }
      return min.value // 默认返回最小值
    }
    return undefined
  },
  set: (newValue: number | undefined) => {
    if (props.mode === 'edit') {
      const value = newValue ?? null
      const display = value !== null ? (unit.value ? `${value}${unit.value}` : String(value)) : ''
      const newFieldValue = {
        raw: value,
        display,
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
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  const numValue = Number(raw)
  if (isNaN(numValue)) {
    return String(raw)
  }
  
  return unit.value ? `${numValue}${unit.value}` : String(numValue)
})

// 计算百分比（用于进度条显示）
const percentage = computed(() => {
  const value = props.value?.raw
  if (value === null || value === undefined) {
    return 0
  }
  
  const numValue = Number(value)
  if (isNaN(numValue)) {
    return 0
  }
  
  const range = max.value - min.value
  if (range === 0) return 0
  
  const pct = ((numValue - min.value) / range) * 100
  return Math.round(pct * 100) / 100 // 保留2位小数
})

// 自动判断状态颜色（根据百分比）
const autoStatus = computed(() => {
  const pct = percentage.value
  if (pct > 80) return 'success'
  if (pct >= 50) return 'warning'
  return 'danger'
})

// 格式化提示（自动带上单位）
const formatTooltipFunc = computed(() => {
  const unitValue = unit.value
  return (value: number) => {
    return unitValue ? `${value}${unitValue}` : String(value)
  }
})

// 格式化进度条文本（显示值和百分比）
const formatProgress = (percentage: number): string => {
  const value = props.value?.raw
  if (value === null || value === undefined) {
    return `${percentage.toFixed(0)}%`
  }
  
  const numValue = Number(value)
  if (isNaN(numValue)) {
    return `${percentage.toFixed(0)}%`
  }
  
  const valueStr = String(numValue)
  const unitValue = unit.value
  const valueDisplay = unitValue ? `${valueStr}${unitValue}` : valueStr
  return `${valueDisplay} (${percentage.toFixed(0)}%)`
}

// 搜索模式：最小值、最大值
const minValue = ref<number | undefined>(undefined)
const maxValue = ref<number | undefined>(undefined)

// 处理值变化
function handleChange(value: number): void {
  // 值变化已在 internalValue 的 setter 中处理
}

// 处理搜索变化
function handleSearchChange(): void {
  const searchValue: any = {}
  if (minValue.value !== undefined && minValue.value !== null) {
    searchValue.min = minValue.value
  }
  if (maxValue.value !== undefined && maxValue.value !== null) {
    searchValue.max = maxValue.value
  }
  
  const hasValue = Object.keys(searchValue).length > 0
  const newFieldValue = hasValue ? {
    raw: searchValue,
    display: '',
    meta: {}
  } : null
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
}

// 初始化：如果字段没有值，使用默认值
watch(
  () => props.value,
  (newValue: any) => {
    if (props.mode === 'edit' && (!newValue || newValue.raw === null || newValue.raw === undefined)) {
      if (defaultValue.value !== undefined) {
        internalValue.value = defaultValue.value
      }
    } else if (props.mode === 'search') {
      // 搜索模式：从 value.raw 中恢复 min/max
      if (newValue?.raw && typeof newValue.raw === 'object') {
        minValue.value = newValue.raw.min
        maxValue.value = newValue.raw.max
      } else {
        minValue.value = undefined
        maxValue.value = undefined
      }
    }
  },
  { immediate: true, deep: true }
)
</script>

<style scoped>
.slider-widget {
  width: 100%;
}

.response-value {
  color: var(--el-text-color-regular);
}

.slider-search {
  display: flex;
  align-items: center;
  gap: 8px;
}

.separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>

