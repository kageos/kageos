<!--
  FormWidget - 表单容器组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 支持 mode="edit" - 可编辑表单
  - 支持 mode="response" - 只读表单
  - 递归渲染子组件
  - 支持条件渲染
-->

<template>
  <div class="form-widget">
    <!-- 编辑模式 -->
    <el-form
      v-if="mode === 'edit'"
      :model="formData"
      label-width="100px"
    >
      <el-form-item
        v-for="subField in visibleSubFields"
        :key="subField.code"
        :label="subField.name"
        :required="isFieldRequired(subField)"
      >
        <!-- 🔥 递归渲染子组件 -->
        <component
          :is="getWidgetComponent(subField.widget?.type || 'input')"
          :field="subField"
          :model-value="getSubFieldValue(subField.code)"
          @update:model-value="(v) => updateSubFieldValue(subField.code, v)"
          :field-path="`${fieldPath}.${subField.code}`"
          :form-manager="formManager"
          :form-renderer="formRenderer"
          :mode="mode"
          :depth="(depth || 0) + 1"
        />
      </el-form-item>
    </el-form>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-form">
      <div
        v-for="subField in visibleSubFields"
        :key="subField.code"
        class="response-field"
      >
        <div class="field-label">{{ subField.name }}</div>
        <div class="field-value">
          <!-- 🔥 递归渲染子组件 -->
          <component
            :is="getWidgetComponent(subField.widget?.type || 'input')"
            :field="subField"
            :model-value="getSubFieldValue(subField.code)"
            :field-path="`${fieldPath}.${subField.code}`"
            mode="response"
            :depth="(depth || 0) + 1"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElForm, ElFormItem } from 'element-plus'
import type { WidgetComponentProps } from '../types'
import { useFormWidget } from '../composables/useFormWidget'
import { widgetComponentFactory } from '../../factories-v2'
import type { FieldConfig } from '../../types/field'

const props = defineProps<WidgetComponentProps>()

// 使用组合式函数
const { visibleSubFields, getSubFieldValue, updateSubFieldValue } = useFormWidget(props)

// 表单数据（用于 el-form 绑定）
const formData = computed(() => {
  const data: Record<string, any> = {}
  visibleSubFields.value.forEach(subField => {
    const value = getSubFieldValue(subField.code)
    data[subField.code] = value?.raw
  })
  return data
})

// 获取组件
function getWidgetComponent(type: string) {
  return widgetComponentFactory.getRequestComponent(type)
}

// 检查字段是否必填
function isFieldRequired(field: FieldConfig): boolean {
  const validation = field.validation || ''
  return validation.includes('required') && !validation.includes('omitempty')
}
</script>

<style scoped>
.form-widget {
  width: 100%;
}

.response-form {
  width: 100%;
}

.response-field {
  margin-bottom: 16px;
}

.field-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.field-value {
  color: var(--el-text-color-regular);
}
</style>

