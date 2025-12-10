<!--
  WidgetComponent - Widget 组件包装器
  新架构的展示层组件
  
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
    :form-renderer="formRenderer"
    :function-method="functionMethod"
    :function-router="functionRouter"
    :user-info-map="userInfoMap"
    :function-name="functionName"
    :record-id="recordId"
  />
  <div v-else class="widget-error">
    组件未找到: {{ field.widget?.type || 'input' }}
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { Logger } from '@/core/utils/logger'
import { widgetComponentFactory } from '@/core/factories-v2'
import type { FieldConfig, FieldValue } from '../../domain/types'
import type { WidgetMode } from '@/core/widgets-v2/types'

const props = withDefaults(defineProps<{
  field: FieldConfig
  value: FieldValue
  mode?: WidgetMode
  fieldPath?: string
  rowData?: any
  formRenderer?: any // FormRenderer 上下文（用于 OnSelectFuzzy 回调）
  functionMethod?: string // 函数 HTTP 方法（用于 OnSelectFuzzy 回调）
  functionRouter?: string // 函数路由（用于 OnSelectFuzzy 回调）
  userInfoMap?: Map<string, any> // 用户信息映射（用于 UserWidget 批量查询优化）
  functionName?: string // 函数名称（用于 FilesWidget 打包下载命名）
  recordId?: string | number // 记录ID（用于 FilesWidget 打包下载命名）
}>(), {
  mode: 'edit',
  fieldPath: '',
  value: () => ({ raw: null, display: '', meta: {} }),
  userInfoMap: () => new Map(),
  functionName: undefined,
  recordId: undefined
})

const emit = defineEmits<{
  'update:modelValue': [value: FieldValue]
}>()

// 调试日志：只在 formRenderer 缺失且需要时警告（response 模式不需要 formRenderer）
// 已移除调试日志

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

