<!--
  NumberWidget - 数字输入组件
  🔥 完全新增，不依赖旧代码
-->

<template>
  <div class="number-widget">
    <!-- 编辑模式 -->
    <div v-if="mode === 'edit'" class="number-input-wrapper">
      <el-input-number
        v-model="internalValue"
        :disabled="field.widget?.config?.disabled"
        :placeholder="field.desc || `请输入${field.name}`"
        :min="minValue"
        :max="maxValue"
        :step="stepValue"
        :precision="0"
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
    <div v-else-if="mode === 'search'" class="number-input-wrapper">
      <el-input-number
        v-model="internalValue"
        :placeholder="`搜索${field.name}`"
        :min="minValue"
        :max="maxValue"
        :step="stepValue"
        :precision="0"
        :controls="true"
      />
      <span v-if="unit" class="unit-text">{{ unit }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { ElInputNumber } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { createFieldValue } from '../utils/createFieldValue'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 获取配置
const config = computed(() => props.field.widget?.config || {})

// 步长（从配置中读取，默认为 1）
const stepValue = computed(() => {
  const step = config.value.step
  if (step !== undefined && step !== null) {
    const num = Number(step)
    return isNaN(num) ? 1 : num
  }
  return 1
})

// 单位（从配置中读取）
const unit = computed(() => config.value.unit || '')

// 默认值（从配置中读取）
const defaultValue = computed(() => {
  const def = config.value.default
  return def !== undefined ? Number(def) : undefined
})

// 最小值/最大值（从验证规则中提取）
const minValue = computed(() => {
  const validation = props.field.validation || ''
  const minMatch = validation.match(/min=(\d+)/)
  return minMatch ? Number(minMatch[1]) : undefined
})

const maxValue = computed(() => {
  const validation = props.field.validation || ''
  const maxMatch = validation.match(/max=(\d+)/)
  return maxMatch ? Number(maxMatch[1]) : undefined
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      // 优先使用当前值
      const value = props.value?.raw
      if (value !== null && value !== undefined) {
        return Number(value)
      }
      // 如果没有值，使用默认值
      if (defaultValue.value !== undefined) {
        return defaultValue.value
      }
      return undefined
    }
    return undefined
  },
  set: (newValue: number | undefined) => {
    if (props.mode === 'edit') {
      const formatted = newValue !== undefined ? String(newValue) : ''
      const display = unit.value ? `${formatted} ${unit.value}` : formatted
      
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        newValue ?? null,
        display
      )
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 显示值（包含单位）
const displayValue = computed(() => {
  const value = props.value
  if (!value) {
    return '-'
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  const formatted = String(raw)
  return unit.value ? `${formatted} ${unit.value}` : formatted
})

// 初始化默认值
onMounted(() => {
  if (props.mode === 'edit' && defaultValue.value !== undefined) {
    const currentValue = props.value?.raw
    if (currentValue === null || currentValue === undefined) {
      internalValue.value = defaultValue.value
    }
  }
})

function handleBlur(): void {
  // 可以在这里添加验证逻辑
}
</script>

<style scoped>
.number-widget {
  width: 100%;
}

.number-input-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
}

.unit-text {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  white-space: nowrap;
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

