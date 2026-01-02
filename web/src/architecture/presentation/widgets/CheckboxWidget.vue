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
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { CheckboxWidgetConfig } from '@/core/types/widget-configs'

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
  const def = config.value.default
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
  gap: 12px;
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

