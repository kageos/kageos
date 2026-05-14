<!--
  CheckboxWidget - 复选框组件
  支持多选场景（如兴趣爱好、标签选择等）
-->

<template>
  <div class="checkbox-widget">
    <!-- 编辑模式 -->
    <div v-if="mode === 'edit'" class="checkbox-group">
      <el-checkbox-group
        v-model="selectedValues"
        :disabled="field.widget?.config?.disabled"
        @change="handleChange"
      >
        <el-checkbox
          v-for="option in options"
          :key="option.value"
          :label="option.value"
          class="checkbox-option"
          :class="{ 'is-selected': selectedValues.includes(option.value) }"
        >
          {{ option.label }}
        </el-checkbox>
      </el-checkbox-group>
    </div>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-checkbox">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-checkbox">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
        size="small"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-checkbox">
      <div class="detail-content">
        <el-tag
          v-for="(value, index) in displayValues"
          :key="index"
          class="tag-item"
        >
          {{ getOptionLabel(value) }}
        </el-tag>
        <span v-if="displayValues.length === 0" class="empty-text">-</span>
      </div>
    </div>
    
    <!-- 搜索模式 -->
    <div v-else-if="mode === 'search'" class="checkbox-group">
      <el-checkbox-group
        v-model="selectedValues"
        @change="handleChange"
      >
        <el-checkbox
          v-for="option in options"
          :key="option.value"
          :label="option.value"
          class="checkbox-option"
          :class="{ 'is-selected': selectedValues.includes(option.value) }"
        >
          {{ option.label }}
        </el-checkbox>
      </el-checkbox-group>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElCheckbox, ElCheckboxGroup, ElTag } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { CheckboxWidgetConfig } from '@/architecture/runtime/types/widget-configs'

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
  return (props.field.widget?.config || {}) as CheckboxWidgetConfig
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
const defaultValues = computed(() => {
  const def = config.value.render_default
  if (Array.isArray(def)) {
    return def
  }
  if (typeof def === 'string' && def) {
    return def.split(',').map(s => s.trim()).filter(Boolean)
  }
  return []
})

// 选中的值（数组）
const selectedValues = computed({
  get: () => {
    const raw = props.value?.raw
    if (Array.isArray(raw)) {
      return raw
    }
    if (typeof raw === 'string' && raw) {
      return [raw]
    }
    // 如果没有值且有默认值，返回默认值
    if (defaultValues.value.length > 0) {
      return defaultValues.value
    }
    return []
  },
  set: (newValues: any[]) => {
    if (props.mode === 'edit' || props.mode === 'search') {
      const displayText = newValues.map((val: any) => {
        const option = options.value.find((opt: any) => opt.value === val)
        return option?.label || String(val)
      }).join(', ')
      
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const fieldValue = createFieldValue(
        props.field,
        newValues,
        displayText || '未选择'
      )
      
      formDataStore.setValue(props.fieldPath, fieldValue)
      emit('update:modelValue', fieldValue)
    }
  }
})

// 显示值（用于只读模式）
const displayValues = computed(() => {
  const raw = props.value?.raw
  if (Array.isArray(raw)) {
    return raw
  }
  if (typeof raw === 'string' && raw) {
    return [raw]
  }
  return []
})

// 获取选项标签
function getOptionLabel(value: any): string {
  const option = options.value.find((opt: any) => opt.value === value)
  return option ? option.label : String(value)
}

// 处理值变化
function handleChange(values: any[]): void {
  selectedValues.value = values
}

// 初始化：如果字段没有值，使用默认值
watch(
  () => props.value,
  (newValue: any) => {
    if (!newValue || !newValue.raw || (Array.isArray(newValue.raw) && newValue.raw.length === 0)) {
      if (defaultValues.value.length > 0) {
        selectedValues.value = defaultValues.value
      }
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.checkbox-widget {
  width: 100%;
}

.checkbox-group {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.checkbox-group :deep(.el-checkbox-group) {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.checkbox-group :deep(.checkbox-option) {
  min-height: 38px;
  margin-right: 0;
  padding: 0 13px 0 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  background: var(--el-bg-color);
  box-sizing: border-box;
  transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.checkbox-group :deep(.checkbox-option:hover) {
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  background: rgba(var(--el-color-primary-rgb), 0.04);
  transform: translateY(-1px);
}

.checkbox-group :deep(.checkbox-option .el-checkbox__input) {
  display: inline-flex;
  align-items: center;
}

.checkbox-group :deep(.checkbox-option .el-checkbox__inner) {
  width: 17px;
  height: 17px;
  border: 1.5px solid var(--el-border-color-darker);
  border-radius: 4px;
  background: var(--el-bg-color);
  background-image: none;
  box-shadow: inset 0 1px 1px rgba(15, 23, 42, 0.05);
  transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease;
}

.checkbox-group :deep(.checkbox-option .el-checkbox__inner::after) {
  top: 1px;
  left: 5px;
  width: 4px;
  height: 9px;
  border-color: #ffffff;
  border-width: 0 2px 2px 0;
}

.checkbox-group :deep(.checkbox-option .el-checkbox__label) {
  padding-left: 8px;
  color: var(--el-text-color-regular);
  font-weight: 500;
}

.checkbox-group :deep(.checkbox-option.is-selected) {
  border-color: var(--el-color-primary);
  background: linear-gradient(180deg, rgba(var(--el-color-primary-rgb), 0.1) 0%, rgba(var(--el-color-primary-rgb), 0.04) 100%);
  box-shadow: 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.18), 0 8px 18px rgba(var(--el-color-primary-rgb), 0.08);
}

.checkbox-group :deep(.checkbox-option.is-selected:hover) {
  background: rgba(var(--el-color-primary-rgb), 0.11);
}

.checkbox-group :deep(.checkbox-option.is-selected .el-checkbox__label) {
  color: var(--el-color-primary-dark-2);
}

.checkbox-group :deep(.checkbox-option .el-checkbox__input.is-checked .el-checkbox__inner),
.checkbox-group :deep(.checkbox-option.is-selected .el-checkbox__inner) {
  border-color: var(--el-color-primary);
  background: linear-gradient(135deg, var(--el-color-primary) 0%, color-mix(in srgb, var(--el-color-primary) 76%, #0f172a) 100%);
  background-image: none;
  box-shadow: 0 3px 8px rgba(var(--el-color-primary-rgb), 0.28);
}

.checkbox-group :deep(.checkbox-option .el-checkbox__input.is-focus .el-checkbox__inner) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.12);
}

.checkbox-group :deep(.checkbox-option.is-disabled) {
  cursor: not-allowed;
  opacity: 0.68;
  transform: none;
  box-shadow: none;
}

.checkbox-group :deep(.checkbox-option.is-disabled:hover) {
  border-color: var(--el-border-color);
  background: var(--el-bg-color);
  transform: none;
}

.checkbox-group :deep(.checkbox-option.is-disabled.is-selected) {
  border-color: rgba(var(--el-color-primary-rgb), 0.32);
  background: rgba(var(--el-color-primary-rgb), 0.06);
}

.checkbox-group :deep(.checkbox-option .el-checkbox__input.is-disabled.is-checked .el-checkbox__inner) {
  border-color: rgba(var(--el-color-primary-rgb), 0.5);
  background: rgba(var(--el-color-primary-rgb), 0.5);
  background-image: none;
  box-shadow: none;
}

.response-checkbox,
.table-cell-checkbox,
.detail-content {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tag-item {
  margin-right: 4px;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}

.detail-checkbox {
  margin-bottom: 16px;
}

.detail-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}
</style>
