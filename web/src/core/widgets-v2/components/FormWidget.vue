<!--
  FormWidget - 表单容器组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 支持 mode="edit" - 可编辑表单
  - 支持 mode="response" - 只读表单
  - 支持 mode="table-cell" - 表格单元格（简化显示 + 详情抽屉）
  - 递归渲染子组件
  - 支持条件渲染
-->

<template>
  <div class="form-widget">
    <!-- 编辑模式 -->
    <el-card
      v-if="mode === 'edit'"
      shadow="hover"
      class="form-card"
    >
      <template #header>
        <div class="form-card-header">
          <span class="form-title">{{ field.name }}</span>
        </div>
      </template>
      <el-form
        :model="formData"
        label-width="100px"
        class="form-widget-form"
      >
        <el-form-item
          v-for="subField in visibleSubFields"
          :key="subField.code"
          :label="subField.name"
          :required="isFieldRequired(subField)"
          :error="getSubFieldError(subField.code)"
          class="form-widget-item"
        >
          <!-- 🔥 递归渲染子组件 -->
          <component
            :is="getWidgetComponent(subField.widget?.type || 'input')"
            :field="subField"
            :value="getSubFieldValue(subField.code)"
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
    </el-card>
    
    <!-- 响应模式（只读） -->
    <el-card
      v-else-if="mode === 'response'"
      shadow="never"
      class="form-card response-form-card"
    >
      <template #header>
        <div class="form-card-header">
          <span class="form-title">{{ field.name }}</span>
        </div>
      </template>
      <div class="response-form">
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
              :value="getSubFieldValue(subField.code)"
              :model-value="getSubFieldValue(subField.code)"
              :field-path="`${fieldPath}.${subField.code}`"
              mode="response"
              :depth="(depth || 0) + 1"
            />
          </div>
        </div>
      </div>
    </el-card>
    
    <!-- 表格单元格模式（简化显示 + 详情抽屉） -->
    <template v-else-if="mode === 'table-cell'">
      <el-button
        link
        type="primary"
        size="small"
        @click="showDetailDrawer = true"
        class="form-field-button"
      >
        <span>共 {{ fieldCount }} 个字段</span>
        <el-icon style="margin-left: 4px">
          <View />
        </el-icon>
      </el-button>
      
      <!-- 详情抽屉（支持编辑和查看） -->
      <el-drawer
        v-model="showDetailDrawer"
        :title="field.name"
        size="60%"
        destroy-on-close
        :z-index="3000"
        append-to-body
      >
        <template #default>
          <div class="form-detail-content">
            <!-- 🔥 抽屉中使用与正常编辑模式完全一致的渲染逻辑 -->
            <!-- 直接使用 edit 模式的渲染方式，确保逻辑一致 -->
            <el-form
              :model="formData"
              label-width="120px"
            >
              <el-form-item
                v-for="subField in visibleSubFields"
                :key="subField.code"
                :label="subField.name"
                :required="isFieldRequired(subField)"
              >
                <!-- 🔥 递归渲染子组件，使用与正常编辑模式完全相同的逻辑 -->
                <component
                  :is="getWidgetComponent(subField.widget?.type || 'input')"
                  :field="subField"
                  :value="getSubFieldValue(subField.code)"
                  :model-value="getSubFieldValue(subField.code)"
                  @update:model-value="(v) => updateSubFieldValue(subField.code, v)"
                  :field-path="`${fieldPath}.${subField.code}`"
                  :form-manager="formManager"
                  :form-renderer="formRenderer"
                  mode="edit"
                  :depth="(depth || 0) + 1"
                />
              </el-form-item>
            </el-form>
          </div>
        </template>
      </el-drawer>
    </template>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-form">
      <div
        v-for="subField in visibleSubFields"
        :key="subField.code"
        class="detail-field"
      >
        <div class="field-label">{{ subField.name }}</div>
        <div class="field-value">
          <component
            :is="getWidgetComponent(subField.widget?.type || 'input')"
            :field="subField"
            :value="getSubFieldValue(subField.code)"
            :model-value="getSubFieldValue(subField.code)"
            :field-path="`${fieldPath}.${subField.code}`"
            mode="detail"
            :depth="(depth || 0) + 1"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElForm, ElFormItem, ElButton, ElDrawer, ElIcon, ElCard } from 'element-plus'
import { View } from '@element-plus/icons-vue'
import type { WidgetComponentProps } from '../types'
import { useFormWidget } from '../composables/useFormWidget'
import { widgetComponentFactory } from '../../factories-v2'
import type { FieldConfig } from '../../types/field'
import type { ValidationEngine, ValidationResult } from '../../validation/types'
import { validateFieldValue, validateFormWidgetNestedFields, type WidgetValidationContext } from '../composables/useWidgetValidation'

const props = defineProps<WidgetComponentProps>()

// 使用组合式函数
const { visibleSubFields, getSubFieldValue, updateSubFieldValue } = useFormWidget(props)

// 详情抽屉状态（用于 table-cell 模式）
const showDetailDrawer = ref(false)

// 字段数量（用于 table-cell 模式显示）
const fieldCount = computed(() => {
  const raw = props.value?.raw
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    return Object.keys(raw).length
  }
  return visibleSubFields.value.length
})

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

/**
 * 获取嵌套字段的错误信息（用于显示在表单项下方）
 */
function getSubFieldError(subFieldCode: string): string {
  const subFieldPath = `${props.fieldPath}.${subFieldCode}`
  
  // 从 formRenderer 获取错误（如果可用）
  if (props.formRenderer && typeof (props.formRenderer as any).getFieldError === 'function') {
    return (props.formRenderer as any).getFieldError(subFieldPath)
  }
  
  return ''
}

/**
 * 验证当前 Widget 及其嵌套字段
 * 
 * 符合依赖倒置原则：FormWidget 自己负责验证嵌套字段
 * 
 * @param validationEngine 验证引擎
 * @param allFields 所有字段配置
 * @param fieldErrors 错误存储 Map（用于存储嵌套字段的错误）
 * @returns 当前字段的错误列表
 */
function validate(
  validationEngine: ValidationEngine | null,
  allFields: FieldConfig[],
  fieldErrors: Map<string, ValidationResult[]>
): ValidationResult[] {
  const context: WidgetValidationContext = {
    validationEngine,
    allFields,
    fieldErrors
  }
  
  // 1. 验证当前字段（如果有验证规则）
  const currentFieldErrors = validateFieldValue(props.field, props.fieldPath, context)
  updateFieldErrors(props.fieldPath, currentFieldErrors, fieldErrors)
  
  // 2. 验证嵌套字段（FormWidget 自己负责）
  const nestedErrors = validateFormWidgetNestedFields(props.field, props.fieldPath, context)
  
  // 3. 将嵌套字段的错误存储到 fieldErrors 中
  nestedErrors.forEach((errors, path) => {
    updateFieldErrors(path, errors, fieldErrors)
  })
  
  return currentFieldErrors
}

/**
 * 更新字段错误状态
 */
function updateFieldErrors(
  fieldPath: string,
  errors: ValidationResult[],
  fieldErrors: Map<string, ValidationResult[]>
): void {
  if (errors.length > 0) {
    fieldErrors.set(fieldPath, errors)
  } else {
    fieldErrors.delete(fieldPath)
  }
}

// 🔥 暴露验证方法给父组件
defineExpose({
  validate
})
</script>

<style scoped>
.form-widget {
  width: 100%;
}

/* Form 卡片样式 */
.form-card {
  width: 100%;
  margin-bottom: 24px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
}

.form-card:last-child {
  margin-bottom: 0;
}

.form-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.form-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.response-form-card {
  background-color: var(--el-bg-color-page);
}

.response-form {
  width: 100%;
}

.response-field {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
}

.response-field:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.field-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
  font-size: 14px;
}

.field-value {
  color: var(--el-text-color-regular);
}

/* Form 表单项间距 */
:deep(.form-widget-form .el-form-item) {
  margin-bottom: 20px;
}

:deep(.form-widget-form .el-form-item:last-child) {
  margin-bottom: 0;
}

/* 表格单元格模式 */
.form-field-button {
  padding: 0;
  height: auto;
  font-size: 14px;
}

/* 详情抽屉内容 */
.form-detail-content {
  padding: 16px 0;
  /* 确保下拉菜单可以正常显示 */
  overflow: visible;
  position: relative;
}

.detail-field {
  margin-bottom: 24px;
}

.detail-form {
  width: 100%;
}

/* 确保抽屉内的下拉菜单可以正常显示 */
:deep(.el-select-dropdown) {
  z-index: 3001 !important;
}

:deep(.el-popper) {
  z-index: 3001 !important;
}

/* 确保抽屉本身不会遮挡下拉菜单 */
:deep(.el-drawer__body) {
  overflow: visible !important;
}

:deep(.el-drawer) {
  overflow: visible !important;
}

/* 全局样式：确保下拉菜单在抽屉中正常显示 */
:deep(.select-dropdown-popper) {
  z-index: 3001 !important;
}

:deep(.select-dropdown-popper .el-select-dropdown) {
  z-index: 3001 !important;
}
</style>
