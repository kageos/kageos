<!--
  SliderWidget - 滑块/进度条组件
  🔥 统一架构组件
  
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
      :show-stops="showStops"
      :show-tooltip="true"
      :marks="marks"
      :format-tooltip="formatTooltipFunc"
      :disabled="field.widget?.config?.disabled"
      @change="handleChange"
    />
    
    <!-- 响应模式（只读）：显示为进度条 -->
    <div v-else-if="mode === 'response'" class="response-progress">
      <el-progress
        :percentage="percentage"
        :status="autoStatus"
        :stroke-width="20"
        :text-inside="true"
        :format="formatProgressText"
      />
    </div>
    
    <!-- 表格单元格模式：进度条 -->
    <el-progress
      v-else-if="mode === 'table-cell'"
      :percentage="percentage"
      :status="autoStatus"
      :stroke-width="20"
      :text-inside="true"
      :format="formatProgressText"
    />
    
    <!-- 详情模式：进度条 -->
    <el-progress
      v-else-if="mode === 'detail'"
      :percentage="percentage"
      :status="autoStatus"
      :stroke-width="20"
      :text-inside="true"
      :format="formatProgressText"
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
import 'element-plus/es/components/slider/style/css'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { Logger } from '@/architecture/shared/logger'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { SliderWidgetConfig } from '@/architecture/domain/types/widget-configs'
import { buildSliderMarks } from '@/architecture/presentation/widgets/utils/sliderMarks'

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
const config = computed(() => {
  return (props.field.widget?.config || {}) as SliderWidgetConfig
})

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
    return stepStr.split('.')[1]?.length ?? 0
  }
  return 0
})

const showStops = computed(() => {
  const range = max.value - min.value
  if (range <= 0 || step.value <= 0) {
    return false
  }

  // 步长刻度过多时只保留关键标签，避免滑道被过密 stops 压垮。
  return range / step.value <= 40
})

// 默认值
const defaultValue = computed(() => {
  const def = config.value.render_default
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
      if (value !== null && value !== undefined && value !== '') {
        const numValue = Number(value)
        // 🔥 关键：如果转换失败，使用默认值或最小值
        if (!isNaN(numValue) && isFinite(numValue)) {
          // 确保值在 min 和 max 范围内
          const clampedValue = Math.max(min.value, Math.min(max.value, numValue))
          return clampedValue
        }
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
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        value,
        display
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

/**
 * 自动判断状态颜色（根据百分比）
 * 
 * ⚠️ 重要：Element Plus 的 el-progress 只支持以下 status 值：
 * - ""（空字符串）
 * - "success"（成功/绿色）
 * - "exception"（异常/红色，对应 danger）
 * - "warning"（警告/黄色）
 * 
 * 注意：不支持 "danger"，必须使用 "exception"
 * 
 * 判断规则：
 * - > 80%：success（绿色）
 * - 50-80%：warning（黄色）
 * - < 50%：exception（红色）
 */
const autoStatus = computed(() => {
  const pct = percentage.value
  if (pct > 80) return 'success'
  if (pct >= 50) return 'warning'
  return 'exception' // Element Plus 中 exception 对应错误/危险状态
})

// 格式化提示（自动带上单位）
const formatTooltipFunc = computed(() => {
  const unitValue = unit.value
  return (value: number) => {
    return unitValue ? `${value}${unitValue}` : String(value)
  }
})

/**
 * 🔥 计算 marks（标记点）
 * 
 * 根据 min、max、step 生成标记点，显示值和标记
 * 标记点包括：最小值、最大值，以及中间的关键点（如果范围不太大）
 */
const marks = computed(() => {
  return buildSliderMarks({
    min: min.value,
    max: max.value,
    step: step.value,
    unit: unit.value
  })
})

/**
 * 格式化进度条文本（显示值和百分比）
 * 
 * ⚠️ 重要：此函数必须返回字符串，不能是异步的
 * 参考 Element Plus 官方示例：const format = (percentage) => (percentage === 100 ? 'Full' : `${percentage}%`)
 * 
 * 显示逻辑：
 * - 如果单位是 %：只显示值（如：50%），避免重复显示百分比
 * - 如果单位不是 %：显示值和单位，以及百分比（如：8.5分 (85%)）
 * 
 * @param percentage - 百分比值（0-100）
 * @returns 格式化后的文本
 */
function formatProgressText(percentage: number): string {
  // 验证 percentage 值（防止无效值导致显示异常）
  if (isNaN(percentage) || !isFinite(percentage)) {
    Logger.warn('SliderWidget', 'formatProgressText: invalid percentage', percentage)
    return '0%'
  }
  
  const value = props.value?.raw
  if (value === null || value === undefined) {
    return `${percentage.toFixed(0)}%`
  }
  
  const numValue = Number(value)
  if (isNaN(numValue)) {
    return `${percentage.toFixed(0)}%`
  }
  
  // 根据步长决定小数位数（例如：step=0.1 时，显示 1 位小数）
  const stepStr = String(step.value)
  const decimals = stepStr.includes('.') ? (stepStr.split('.')[1]?.length ?? 0) : 0
  const valueStr = numValue.toFixed(decimals)
  
  const unitValue = unit.value
  const isPercentageUnit = unitValue === '%' || unitValue === '％'
  
  // ⚠️ 关键：如果单位本身就是百分比，只显示值，不重复显示百分比
  // 例如：单位是 %，值是 50，显示 "50%"，而不是 "50% (50%)"
  if (isPercentageUnit) {
    return `${valueStr}%`
  }
  
  // 如果单位不是百分比，显示值和单位，以及百分比
  // 例如：单位是 "分"，值是 8.5，显示 "8.5分 (85%)"
  const valueDisplay = unitValue ? `${valueStr}${unitValue}` : valueStr
  return `${valueDisplay} (${percentage.toFixed(0)}%)`
}

/**
 * 搜索模式：最小值、最大值
 * 
 * ⚠️ 注意：每个 SliderWidget 实例都有独立的 minValue 和 maxValue
 * 多个 slider 字段可以同时存在搜索值，互不影响
 */
const minValue = ref<number | undefined>(undefined)
const maxValue = ref<number | undefined>(undefined)

/**
 * 处理编辑模式的值变化
 * 注意：值变化已在 internalValue 的 setter 中处理，这里不需要额外逻辑
 */
function handleChange(value: number | number[]): void {
  // 值变化已在 internalValue 的 setter 中处理
}

/**
 * 处理搜索模式的值变化
 * 
 * ⚠️ 关键：将 min/max 值转换为 { min, max } 对象格式
 * 这个对象会被传递给父组件，最终转换为 URL 参数：gte=field:min&lte=field:max
 * 
 * 如果 min 和 max 都为空，传递 null（表示清空搜索条件）
 */
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
  } : {
    raw: null,
    display: '',
    meta: {}
  }
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
}

/**
 * 监听值变化，处理初始化和值恢复
 * 
 * ⚠️ 关键逻辑：
 * 1. 编辑模式：
 *    - 如果字段没有值，使用默认值
 *    - 如果值存在但转换失败或超出范围，自动修正
 * 2. 搜索模式：从 value.raw 中恢复 min/max（用于 URL 恢复）
 * 
 * 注意：使用 deep: true 确保能监听到对象内部的变化
 */
watch(
  () => props.value,
  (newValue: any) => {
    if (props.mode === 'edit') {
      if (!newValue || newValue.raw === null || newValue.raw === undefined || newValue.raw === '') {
      // 编辑模式：如果字段没有值，使用默认值
      if (defaultValue.value !== undefined) {
        internalValue.value = defaultValue.value
        }
      } else {
        // 🔥 关键：如果值存在，确保它能正确显示
        // 通过设置 internalValue 来触发值的验证和修正
        const numValue = Number(newValue.raw)
        if (!isNaN(numValue) && isFinite(numValue)) {
          // 确保值在范围内
          const clampedValue = Math.max(min.value, Math.min(max.value, numValue))
          // 只有当值发生变化时才更新，避免无限循环
          if (internalValue.value !== clampedValue) {
            internalValue.value = clampedValue
          }
        } else {
          // 如果值转换失败，使用默认值或最小值
          if (defaultValue.value !== undefined) {
            internalValue.value = defaultValue.value
          } else {
            internalValue.value = min.value
          }
        }
      }
    } else if (props.mode === 'search') {
      // 搜索模式：从 value.raw 中恢复 min/max
      // ⚠️ 重要：只有当 newValue.raw 是对象时才恢复，避免其他类型的数据影响
      if (newValue?.raw && typeof newValue.raw === 'object' && !Array.isArray(newValue.raw)) {
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

.response-progress {
  width: 100%;
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
