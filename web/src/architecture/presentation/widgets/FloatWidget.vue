<!--
  FloatWidget - 浮点数输入组件
  🔥 统一架构组件
-->

<template>
  <div class="float-widget">
    <!-- 编辑模式 -->
    <div v-if="mode === 'edit'" class="float-input-wrapper">
    <el-input-number
      v-model="internalValue"
      :disabled="field.widget?.config?.disabled"
      :placeholder="placeholder"
      :min="minValue"
      :max="maxValue"
        :step="stepValue"
        :precision="precisionValue"
      :controls="true"
      @blur="handleBlur"
    />
      <span v-if="unit" class="unit-text">{{ unit }}</span>
    </div>
    
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
    <div v-else-if="mode === 'search'" class="float-input-wrapper">
    <el-input-number
      v-model="internalValue"
      :placeholder="`搜索${field.name}`"
      :min="minValue"
      :max="maxValue"
        :step="stepValue"
        :precision="precisionValue"
      :controls="true"
    />
      <span v-if="unit" class="unit-text">{{ unit }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { ElInputNumber } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { FloatWidgetConfig } from '@/architecture/domain/types/widget-configs'

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
  return (props.field.widget?.config || {}) as FloatWidgetConfig
})

// 精度（小数位数）
const precisionValue = computed(() => {
  const precision = config.value.precision
  if (precision !== undefined && precision !== null) {
    const num = Number(precision)
    return isNaN(num) ? 2 : num
  }
  return 2 // 默认2位小数
})

// 步长
const stepValue = computed(() => {
  const step = config.value.step
  if (step !== undefined && step !== null) {
    const num = Number(step)
    return isNaN(num) ? 0.01 : num
  }
  return 0.01 // 默认0.01
})

// 单位
const unit = computed(() => config.value.unit || '')

// 占位符（优先级：config.placeholder > field.desc > 默认值）
const placeholder = computed(() => {
  if (config.value.placeholder) {
    return config.value.placeholder
  }
  if (props.field.desc) {
    return props.field.desc
  }
  return `请输入${props.field.name}`
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

// 最小值/最大值（优先使用 SDK widget config，兼容 validate 中的 min/max/gte/lte）
const minValue = computed(() => {
  return parseConfigNumber(config.value.min) ?? parseValidationNumber(props.field.validation, ['min', 'gte'])
})

const maxValue = computed(() => {
  return parseConfigNumber(config.value.max) ?? parseValidationNumber(props.field.validation, ['max', 'lte'])
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const value = props.value?.raw
      if (value !== null && value !== undefined) {
        return Number(value)
      }
      // 如果没有值且有默认值，返回默认值
      if (defaultValue.value !== undefined) {
        return defaultValue.value
      }
      return undefined
    }
    return undefined
  },
  set: (newValue: number | undefined) => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const formatted = newValue !== undefined ? newValue.toFixed(precisionValue.value) : ''
      const display = unit.value ? `${formatted} ${unit.value}` : formatted
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        newValue ?? null,
        display
      )
      
      if (props.mode === 'edit') {
        formDataStore.setValue(props.fieldPath, newFieldValue)
      }
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
    const display = String(value.display)
    return unit.value ? `${display} ${unit.value}` : display
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  const numValue = Number(raw)
  if (isNaN(numValue)) {
    return String(raw)
  }
  
  const formatted = numValue.toFixed(precisionValue.value)
  return unit.value ? `${formatted} ${unit.value}` : formatted
})

function handleBlur(): void {
  // 可以在这里添加验证逻辑
}

// 初始化：如果字段没有值，使用默认值
watch(
  () => props.value,
  (newValue: any) => {
    if (props.mode === 'edit' && (!newValue || newValue.raw === null || newValue.raw === undefined)) {
      if (defaultValue.value !== undefined) {
        internalValue.value = defaultValue.value
      }
    }
  },
  { immediate: true }
)

function parseConfigNumber(value: unknown): number | undefined {
  if (value === undefined || value === null || value === '') {
    return undefined
  }
  const num = Number(value)
  return Number.isFinite(num) ? num : undefined
}

function parseValidationNumber(validation: string | undefined, keys: string[]): number | undefined {
  if (!validation) {
    return undefined
  }
  for (const key of keys) {
    const match = validation.match(new RegExp(`(?:^|,)${key}=(-?\\d+(?:\\.\\d+)?)`))
    if (match?.[1] !== undefined) {
      const value = Number(match[1])
      if (Number.isFinite(value)) {
        return value
      }
    }
  }
  return undefined
}
</script>

<style scoped>
.float-widget {
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

.float-input-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
}

.unit-text {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  white-space: nowrap;
}
</style>
