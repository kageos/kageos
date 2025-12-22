<!--
  ProgressWidget - 进度条组件
  🔥 用于展示百分比、得票率等进度数据
-->

<template>
  <div class="progress-widget">
    <!-- 编辑模式（通常不支持编辑，但保留兼容性） -->
    <div v-if="mode === 'edit'" class="edit-progress">
      <el-input-number
        v-model="internalValue"
        :disabled="field.widget?.config?.disabled"
        :placeholder="field.desc || `请输入${field.name}`"
        :min="minValue"
        :max="maxValue"
        :step="0.01"
        :precision="2"
        :controls="true"
        @blur="handleBlur"
      />
      <span v-if="unit" class="unit-text">{{ unit }}</span>
    </div>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-progress">
      <el-progress
        :percentage="percentage"
        :format="formatText"
      />
    </div>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-progress">
      <el-progress
        :percentage="percentage"
        :format="formatText"
        :stroke-width="8"
      />
    </div>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-progress">
      <div class="detail-label">{{ field.name }}</div>
      <el-progress
        :percentage="percentage"
        :format="formatText"
        :stroke-width="12"
      />
    </div>
    
    <!-- 搜索模式（通常不支持搜索，但保留兼容性） -->
    <div v-else-if="mode === 'search'" class="search-progress">
      <el-input-number
        v-model="internalValue"
        :placeholder="`搜索${field.name}`"
        :min="minValue"
        :max="maxValue"
        :step="0.01"
        :precision="2"
        :controls="true"
      />
      <span v-if="unit" class="unit-text">{{ unit }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElProgress, ElInputNumber } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { createFieldValue } from '../utils/createFieldValue'
import type { ProgressWidgetConfig } from '@/core/types/widget-configs'

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
  return (props.field.widget?.config || {}) as ProgressWidgetConfig
})

// 最小值/最大值（从配置中获取，默认 0-100）
const minValue = computed(() => {
  const min = config.value.min
  if (min !== undefined && min !== null) {
    return Number(min)
  }
  return 0
})

const maxValue = computed(() => {
  const max = config.value.max
  if (max !== undefined && max !== null) {
    return Number(max)
  }
  return 100
})

// 单位（默认 %）
const unit = computed(() => config.value.unit || '%')

// 原始数值
const rawValue = computed(() => {
  const value = props.value?.raw
  if (value === null || value === undefined || value === '') {
    return 0
  }
  const num = Number(value)
  return isNaN(num) ? 0 : num
})

// 百分比（0-100）
const percentage = computed(() => {
  const value = rawValue.value
  const min = minValue.value
  const max = maxValue.value
  
  if (max === min) {
    return 0
  }
  
  // 将值映射到 0-100 范围
  const mapped = ((value - min) / (max - min)) * 100
  
  // 限制在 0-100 之间
  return Math.max(0, Math.min(100, mapped))
})

// 格式化后的显示值
const formattedValue = computed(() => {
  const value = rawValue.value
  const formatted = value.toFixed(2)
  return unit.value ? `${formatted} ${unit.value}` : formatted
})

// 格式化进度条文字
const formatText = computed(() => {
  return () => formattedValue.value
})

// 内部值（用于 v-model，仅在编辑模式）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      return rawValue.value
    }
    return undefined
  },
  set: (newValue: number | undefined) => {
    if (props.mode === 'edit') {
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        newValue ?? null,
        newValue !== undefined ? formattedValue.value : ''
      )
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

function handleBlur(): void {
  // 可以在这里添加验证逻辑
}
</script>

<style scoped>
.progress-widget {
  width: 100%;
}

.edit-progress,
.search-progress {
  display: flex;
  align-items: center;
  gap: 8px;
}

.unit-text {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  white-space: nowrap;
}

.response-progress {
  width: 100%;
}

.table-cell-progress {
  width: 100%;
}

.detail-progress {
  margin-bottom: 16px;
}

.detail-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
}

</style>

