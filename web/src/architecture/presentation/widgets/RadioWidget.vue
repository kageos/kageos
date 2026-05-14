<!--
  RadioWidget - 单选框组件
  用于单选场景（如性别、状态等）
-->

<template>
  <div class="radio-widget">
    <!-- 编辑模式 -->
    <div v-if="mode === 'edit'" class="radio-group">
      <el-radio-group
        v-model="selectedValue"
        :disabled="field.widget?.config?.disabled"
        @change="handleChange"
      >
        <el-radio
          v-for="option in options"
          :key="option.value"
          :label="option.value"
          class="radio-option"
          :class="{ 'is-selected': selectedValue === option.value }"
        >
          {{ option.label }}
        </el-radio>
      </el-radio-group>
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
    <div v-else-if="mode === 'search'" class="radio-group">
      <el-radio-group
        v-model="selectedValue"
        @change="handleChange"
      >
        <el-radio
          v-for="option in options"
          :key="option.value"
          :label="option.value"
          class="radio-option"
          :class="{ 'is-selected': selectedValue === option.value }"
        >
          {{ option.label }}
        </el-radio>
      </el-radio-group>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { ElRadio, ElRadioGroup } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { RadioWidgetConfig } from '@/architecture/runtime/types/widget-configs'

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
  return (props.field.widget?.config || {}) as RadioWidgetConfig
})

// 选项列表
const options = computed(() => {
  const opts = config.value.options || []
  return opts.map((opt: any) => {
    if (typeof opt === 'string') {
      return { label: opt, value: opt }
    }
    return opt
  })
})

// 默认值
const defaultValue = computed(() => {
  const def = config.value.render_default
  if (def !== undefined && def !== null) {
    return String(def)
  }
  return ''
})

// 选中的值
const selectedValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const raw = props.value?.raw
      if (raw !== null && raw !== undefined && raw !== '') {
        return raw
      }
      // 如果没有值且有默认值，返回默认值
      if (defaultValue.value) {
        return defaultValue.value
      }
      return null
    }
    return null
  },
  set: (newValue: any) => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const selectedOption = options.value.find((opt: any) => opt.value === newValue)
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const fieldValue = createFieldValue(
        props.field,
        newValue,
        selectedOption?.label || String(newValue)
      )
      
      formDataStore.setValue(props.fieldPath, fieldValue)
      emit('update:modelValue', fieldValue)
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
  
  // 尝试从选项中获取标签
  const option = options.value.find((opt: any) => opt.value === raw)
  return option ? option.label : String(raw)
})

// 处理值变化
function handleChange(value: any): void {
  selectedValue.value = value
}

// 初始化：如果字段没有值，使用默认值
watch(
  () => props.value,
  (newValue: any) => {
    if (!newValue || !newValue.raw || newValue.raw === '') {
      if (defaultValue.value) {
        selectedValue.value = defaultValue.value
      }
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.radio-widget {
  width: 100%;
}

.radio-group {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.radio-group :deep(.el-radio-group) {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.radio-group :deep(.radio-option) {
  margin-right: 0;
  min-height: 38px;
  padding: 0 14px 0 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 18px;
  background: var(--el-fill-color-blank);
  transition: border-color 0.2s, background-color 0.2s, box-shadow 0.2s, transform 0.2s;
}

.radio-group :deep(.radio-option:hover) {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-fill-color-light);
  transform: translateY(-1px);
}

.radio-group :deep(.radio-option .el-radio__input) {
  display: inline-flex;
  align-items: center;
}

.radio-group :deep(.radio-option .el-radio__inner) {
  width: 17px;
  height: 17px;
  border: 1.5px solid var(--el-border-color-darker);
  background: var(--el-bg-color);
  box-shadow: inset 0 1px 1px rgba(15, 23, 42, 0.05);
  transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease;
}

.radio-group :deep(.radio-option .el-radio__inner::after) {
  width: 7px;
  height: 7px;
  background: #ffffff;
}

.radio-group :deep(.radio-option .el-radio__label) {
  padding-left: 8px;
  color: var(--el-text-color-regular);
  font-weight: 500;
}

.radio-group :deep(.radio-option.is-selected) {
  border-color: var(--el-color-primary);
  background: linear-gradient(180deg, rgba(var(--el-color-primary-rgb), 0.1) 0%, rgba(var(--el-color-primary-rgb), 0.04) 100%);
  box-shadow: 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.18), 0 8px 18px rgba(var(--el-color-primary-rgb), 0.08);
}

.radio-group :deep(.radio-option.is-selected .el-radio__label) {
  color: var(--el-color-primary-dark-2);
}

.radio-group :deep(.radio-option.is-selected .el-radio__inner) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: inset 0 0 0 3px #ffffff, 0 3px 8px rgba(var(--el-color-primary-rgb), 0.24);
}

.radio-group :deep(.radio-option .el-radio__input.is-focus .el-radio__inner) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.12);
}

.radio-group :deep(.radio-option.is-disabled) {
  opacity: 0.65;
  cursor: not-allowed;
  transform: none;
}

.radio-group :deep(.radio-option.is-disabled.is-selected .el-radio__inner) {
  border-color: rgba(var(--el-color-primary-rgb), 0.5);
  background: rgba(var(--el-color-primary-rgb), 0.5);
  box-shadow: inset 0 0 0 3px var(--el-bg-color);
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
