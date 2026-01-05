<!--
  TextAreaWidget - 文本域组件
  🔥 完全新增，不依赖旧代码
-->

<template>
  <div class="textarea-widget">
    <!-- 编辑模式 -->
    <el-input
      v-if="mode === 'edit'"
      v-model="internalValue"
      type="textarea"
      :disabled="false"
      :placeholder="editPlaceholder"
      :rows="rows"
      :maxlength="maxLength"
      :show-word-limit="showWordLimit"
      @blur="handleBlur"
    />
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-value">
      <pre>{{ displayValue }}</pre>
    </div>
    
    <!-- 表格单元格模式 -->
    <span v-else-if="mode === 'table-cell'" class="table-cell-value">
      {{ truncatedValue }}
    </span>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <div class="detail-content">
        <pre>{{ displayValue }}</pre>
      </div>
    </div>
    
    <!-- 搜索模式 -->
    <el-input
      v-else-if="mode === 'search'"
      v-model="internalValue"
      type="textarea"
      :placeholder="searchPlaceholder"
      :rows="3"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElInput } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { TextAreaWidgetConfig } from '@/core/types/widget-configs'

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
  return (props.field.widget?.config || {}) as TextAreaWidgetConfig
})

// 行数（从配置中获取，注意：TextAreaWidgetConfig 中没有 rows 字段，使用默认值）
const rows = computed(() => {
  return 4
})

// 最大长度（从验证规则或配置中获取，注意：TextAreaWidgetConfig 中没有 maxlength 字段）
const maxLength = computed(() => {
  const configMaxLength = undefined
  if (configMaxLength) {
    return configMaxLength
  }
  
  const validation = props.field.validation || ''
  const maxMatch = validation.match(/max=(\d+)/)
  return maxMatch ? Number(maxMatch[1]) : undefined
})

// 是否显示字数统计（注意：TextAreaWidgetConfig 中没有 showWordLimit 字段，使用默认值）
const showWordLimit = computed(() => {
  return false
})

// 编辑模式的 placeholder（优先级：widgetConfig.placeholder > field.desc > 默认值）
const editPlaceholder = computed(() => {
  if (widgetConfig.value.placeholder) {
    return widgetConfig.value.placeholder
  }
  if (props.field.desc) {
    return props.field.desc
  }
  return `请输入${props.field.name}`
})

// 搜索模式的 placeholder（优先级：widgetConfig.placeholder > 默认值）
const searchPlaceholder = computed(() => {
  if (widgetConfig.value.placeholder) {
    return widgetConfig.value.placeholder
  }
  return `搜索${props.field.name}`
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      // 优先使用 props.value，如果没有则使用 props.modelValue（兼容）
      const fieldValue = props.value || (props as any).modelValue
      const value = fieldValue?.raw
      return value !== null && value !== undefined ? String(value) : ''
    }
    return ''
  },
  set: (newValue: string) => {
    if (props.mode === 'edit') {
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        newValue,
        newValue
      )
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 显示值
const displayValue = computed(() => {
  // 优先使用 props.value，如果没有则使用 props.modelValue（兼容）
  const fieldValue = props.value || (props as any).modelValue
  if (!fieldValue) {
    return '-'
  }
  
  if (fieldValue.display) {
    return fieldValue.display
  }
  
  const raw = fieldValue.raw
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  return String(raw)
})

// 截断值（用于表格单元格）
const truncatedValue = computed(() => {
  const value = displayValue.value
  if (value.length > 50) {
    return value.substring(0, 50) + '...'
  }
  return value
})

function handleBlur(): void {
  // 可以在这里添加验证逻辑
}
</script>

<style scoped>
.textarea-widget {
  width: 100%;
}

.response-value {
  color: var(--el-text-color-regular);
}

.response-value pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
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

.detail-content pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style>

