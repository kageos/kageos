<!--
  SwitchWidget - 开关组件
  🔥 统一架构组件
-->

<template>
  <div class="switch-widget">
    <!-- 编辑模式 -->
    <el-switch
      v-if="mode === 'edit'"
      v-model="internalValue"
      :disabled="false"
      :active-text="activeText"
      :inactive-text="inactiveText"
      @change="handleChange"
    />
    
    <!-- 响应模式（只读） -->
    <el-tag
      v-else-if="mode === 'response'"
      :type="displayValue ? 'success' : 'info'"
      size="small"
    >
      {{ displayValue ? activeText : inactiveText }}
    </el-tag>
    
    <!-- 表格单元格模式 - 使用带文字的开关 -->
    <el-switch
      v-else-if="mode === 'table-cell'"
      :model-value="displayValue"
      inline-prompt
      :active-text="activeText"
      :inactive-text="inactiveText"
      :disabled="false"
      @change="handleTableCellChange"
    />
    
    <!-- 详情模式 - 使用带文字的开关（只读） -->
    <el-switch
      v-else-if="mode === 'detail'"
      :model-value="displayValue"
      inline-prompt
      :active-text="activeText"
      :inactive-text="inactiveText"
      disabled
    />
    
    <!-- 搜索模式（不支持） -->
    <span v-else class="not-supported">搜索模式不支持开关组件</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElSwitch, ElTag } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { SwitchWidgetConfig } from '@/architecture/runtime/types/widget-configs'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 获取配置（带类型，注意：SwitchWidgetConfig 当前无配置项）
const widgetConfig = computed(() => {
  return (props.field.widget?.config || {}) as SwitchWidgetConfig
})

// 激活文本/非激活文本（注意：SwitchWidgetConfig 中没有这些字段，使用默认值）
const activeText = computed(() => {
  return '是'
})

const inactiveText = computed(() => {
  return '否'
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit') {
      const value = props.value?.raw
      return value === true || value === 'true' || value === 1 || value === '1'
    }
    return false
  },
  set: (newValue: boolean) => {
    if (props.mode === 'edit') {
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        newValue,
        newValue ? activeText.value : inactiveText.value
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
    return false
  }
  
  const raw = value.raw
  return raw === true || raw === 'true' || raw === 1 || raw === '1'
})

function handleChange(value: string | number | boolean): void {
  // 可以在这里添加验证逻辑
}

// 表格单元格模式下的变更处理
function handleTableCellChange(value: string | number | boolean): void {
  if (props.mode === 'table-cell') {
    const normalizedValue = value === true || value === 'true' || value === 1 || value === '1'
    // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
    const newFieldValue = createFieldValue(
      props.field,
      normalizedValue,
      normalizedValue ? activeText.value : inactiveText.value
    )
    
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
  }
}
</script>

<style scoped>
.switch-widget {
  width: 100%;
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

.not-supported {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
</style>
