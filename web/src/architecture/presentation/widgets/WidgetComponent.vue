<!--
  WidgetComponent - Widget 组件包装器
  统一架构的展示层组件
  
  职责：
  - 根据字段类型动态加载 Widget 组件
  - 传递统一的 Props
  - 处理事件
-->

<template>
  <component
    :is="widgetComponent"
    v-if="widgetComponent"
    :field="field"
    :value="value"
    :model-value="value"
    @update:model-value="handleUpdate"
    :field-path="fieldPath"
    :mode="mode"
    :search-type="searchType"
    :row-data="rowData"
    :form-renderer="formRenderer"
    :function-method="functionMethod"
    :function-router="functionRouter"
  />
  <div v-else class="widget-error">
    组件未找到: {{ field.widget?.type || 'input' }}
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { widgetComponentFactory } from '@/architecture/presentation/widgets/registry'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import type { WidgetMode } from '@/architecture/presentation/widgets/types'
import { createEmptyRawFieldValue } from '@/architecture/runtime/utils/createFieldValue'

const props = withDefaults(defineProps<{
  field: FieldConfig
  value: FieldValue
  mode?: WidgetMode
  fieldPath?: string
  searchType?: string
  rowData?: any
  formRenderer?: any // FormRenderer 上下文（用于 OnSelectFuzzy 回调）
  functionMethod?: string // 函数 HTTP 方法（用于 OnSelectFuzzy 回调）
  functionRouter?: string // 函数路由（用于 OnSelectFuzzy 回调）
}>(), {
  mode: 'edit',
  fieldPath: '',
  value: () => createEmptyRawFieldValue(),
})

const emit = defineEmits<{
  'update:modelValue': [value: FieldValue]
}>()

// 获取 Widget 组件
// 🔥 优化：基础组件已经在模块加载时同步注册，无需等待
// 只有 FormWidget 和 TableWidget 需要异步注册，但应用启动时会等待它们注册完成
const widgetComponent = computed(() => {
  const type = props.field.widget?.type || 'input'
  
  if (props.mode === 'response') {
    return widgetComponentFactory.getResponseComponent(type)
  } else {
    return widgetComponentFactory.getRequestComponent(type)
  }
})

// 处理更新事件
const handleUpdate = (value: FieldValue): void => {
  emit('update:modelValue', value)
}
</script>

<style scoped>
.widget-error {
  color: var(--el-color-danger);
  font-size: 12px;
}
</style>
