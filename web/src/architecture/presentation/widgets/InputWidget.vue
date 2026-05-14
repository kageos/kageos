<!--
  InputWidget - 输入框组件
  🔥 统一架构组件
  
  功能：
  - 支持 mode="edit" - 可编辑输入
  - 支持 mode="response" - 只读展示
  - 支持 mode="table-cell" - 表格单元格
  - 支持 mode="detail" - 详情展示
  - 支持 mode="search" - 搜索输入
-->

<template>
  <div class="input-widget">
    <!-- 编辑模式 -->
    <el-input
      v-if="mode === 'edit'"
      v-model="internalValue"
      :disabled="widgetConfig.disabled"
      :placeholder="widgetConfig.placeholder || field.desc || `请输入${field.name}`"
      :clearable="true"
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
    <el-input
      v-else-if="mode === 'search'"
      v-model="internalValue"
      :placeholder="`搜索${field.name}`"
      :clearable="true"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElInput } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/infrastructure/stores/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { InputWidgetConfig } from '@/architecture/domain/types/widget-configs'

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
  return (props.field.widget?.config || {}) as InputWidgetConfig
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const value = props.value?.raw
      return value !== null && value !== undefined ? String(value) : ''
    }
    return ''
  },
  set: (newValue: string) => {
    if (props.mode === 'edit' || props.mode === 'search') {
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        newValue,
        newValue
      )
      
      if (props.mode === 'edit') {
        // 同步到 Store (仅编辑模式)
        formDataStore.setValue(props.fieldPath, newFieldValue)
      }
      
      // 触发 v-model 更新
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
  
  // 优先使用 display，如果没有则使用 raw
  if (value.display) {
    return value.display
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  return String(raw)
})

// 失焦处理
function handleBlur(): void {
  // 可以在这里添加验证逻辑
}
</script>

<style scoped>
.input-widget {
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

