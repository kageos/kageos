<!--
  WidgetComponent - Widget 组件包装器
  🔥 新架构的展示层组件
  
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
    :row-data="rowData"
  />
  <div v-else class="widget-error">
    组件未找到: {{ field.widget?.type || 'input' }}
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { widgetComponentFactory } from '@/core/factories-v2'
import type { FieldConfig, FieldValue } from '../../domain/types'
import type { WidgetMode } from '@/core/widgets-v2/types'

const props = withDefaults(defineProps<{
  field: FieldConfig
  value: FieldValue
  mode?: WidgetMode
  fieldPath?: string
  rowData?: any
}>(), {
  mode: 'edit',
  fieldPath: '',
  value: () => ({ raw: null, display: '', meta: {} })
})

const emit = defineEmits<{
  'update:modelValue': [value: FieldValue]
}>()

// 获取 Widget 组件
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

