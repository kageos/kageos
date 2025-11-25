<!--
  NumberWidget - 数字输入组件
  🔥 完全新增，不依赖旧代码
-->

<template>
  <div class="number-widget">
    <!-- 编辑模式 -->
    <el-input-number
      v-if="mode === 'edit'"
      v-model="internalValue"
      :disabled="field.widget?.config?.disabled"
      :placeholder="field.desc || `请输入${field.name}`"
      :min="minValue"
      :max="maxValue"
      :step="1"
      :precision="0"
      :controls="true"
      @blur="handleBlur"
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
    <el-input-number
      v-else-if="mode === 'search'"
      v-model="internalValue"
      :placeholder="`搜索${field.name}`"
      :min="minValue"
      :max="maxValue"
      :step="1"
      :precision="0"
      :controls="true"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElInputNumber } from 'element-plus'
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
      const value = props.value?.raw
      return value !== null && value !== undefined ? Number(value) : undefined
    }
    return undefined
  },
  set: (newValue: number | undefined) => {
    if (props.mode === 'edit') {
      const newFieldValue = {
        raw: newValue ?? null,
        display: newValue !== undefined ? String(newValue) : '',
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
  
  return String(raw)
})

function handleBlur(): void {
  // 可以在这里添加验证逻辑
}
</script>

<style scoped>
.number-widget {
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

