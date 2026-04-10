<!--
  FormWidget - 表单容器组件
  用于渲染嵌套的表单结构（form/struct 类型字段）
  
  ============================================
  📋 需求说明
  ============================================
  
  1. **嵌套表单渲染**：
     - 支持渲染 `form` 或 `struct` 类型的字段
     - 递归渲染子字段，支持任意嵌套深度
     - 支持 form 嵌套 table，table 嵌套 form 等复杂结构
  
  2. **多种渲染模式**：
     - `edit` 模式：可编辑表单，显示输入控件
     - `response` 模式：只读展示，显示值但不允许编辑
     - `table-cell` 模式：表格单元格中显示，简化版本
     - `detail` 模式：详情展示，可能有更丰富的格式化
  
  3. **字段验证**：
     - 支持字段验证错误显示
     - 验证错误使用字段的中文名称（`field.name`）
     - 使用 `:prop` 和 `:error` 属性关联验证错误
  
  ============================================
  🎯 设计思路
  ============================================
  
  1. **递归渲染**：
     - 遍历字段的 `data.fields` 或 `widget.fields` 获取子字段
     - 使用 `WidgetComponent` 递归渲染每个子字段
     - 字段路径使用 `${fieldPath}.${subField.code}` 格式
  
  2. **模式适配**：
     - 根据 `mode` 属性渲染不同的 UI
     - `edit` 模式：使用 `el-form-item` 和输入控件
     - `response` 模式：使用 `<span>` 或 `<div>` 展示值
  
  3. **验证集成**：
     - 使用 `:prop` 属性关联字段路径
     - 使用 `:error` 属性显示验证错误
     - 从 FormDomainService 获取字段验证错误
  
  ============================================
  📝 关键功能
  ============================================
  
  1. **子字段获取**：
     - 优先从 `field.data.fields` 获取子字段
     - 其次从 `field.widget.fields` 获取子字段
     - 支持条件渲染（`depend_on`）
  
  2. **字段值管理**：
     - 使用 `getSubFieldValue` 获取子字段值
     - 使用 `handleSubFieldUpdate` 更新子字段值
     - 字段值格式：`FieldValue`（包含 raw、display、meta）
  
  3. **验证错误显示**：
     - 使用 `getSubFieldError` 获取子字段验证错误
     - 验证错误显示在字段下方
  
  ============================================
  ⚠️ 注意事项
  ============================================
  
  1. **字段路径**：
     - 必须使用 `${fieldPath}.${subField.code}` 格式
     - 确保嵌套字段的路径唯一性
  
  2. **字段 ID 生成**：
     - 使用 `:prop` 确保 Element Plus 生成正确的 `id` 属性
     - 嵌套字段的 `id` 格式：`form_field_path_sub_field_code`
  
  3. **验证错误关联**：
     - 使用 `:prop` 和 `:error` 属性关联验证错误
     - 验证错误路径必须与字段路径一致
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
        label-position="left"
        :label-width="FORM_LABEL_WIDTH"
        class="form-widget-form"
      >
        <template v-for="subField in visibleSubFields" :key="subField.code">
          <div v-if="labelsOnTop" class="form-field-label-top">
            <label class="field-label">
              {{ subField.name }}
              <span v-if="isFieldRequired(subField)" class="required">*</span>
            </label>
            <el-form-item class="form-item-no-label">
              <component
                :is="getWidgetComponent(subField.widget?.type || 'input', mode)"
                :field="subField"
                :value="getSubFieldValue(subField.code)"
                :model-value="getSubFieldValue(subField.code)"
                @update:model-value="handleSubFieldModelUpdate(subField.code, $event)"
                :field-path="`${fieldPath}.${subField.code}`"
                :form-manager="formManager"
                :form-renderer="formRenderer"
                :mode="mode"
                :depth="(depth || 0) + 1"
              />
            </el-form-item>
          </div>
          <el-form-item
            v-else
            :label="subField.name"
            :required="isFieldRequired(subField)"
            class="form-widget-item"
          >
            <component
              :is="getWidgetComponent(subField.widget?.type || 'input', mode)"
              :field="subField"
              :value="getSubFieldValue(subField.code)"
              :model-value="getSubFieldValue(subField.code)"
              @update:model-value="handleSubFieldModelUpdate(subField.code, $event)"
              :field-path="`${fieldPath}.${subField.code}`"
              :form-manager="formManager"
              :form-renderer="formRenderer"
              :mode="mode"
              :depth="(depth || 0) + 1"
            />
          </el-form-item>
        </template>
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
            <component
              :is="getWidgetComponent(subField.widget?.type || 'input', 'response')"
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
    
    <!-- 
      🔥 表格单元格模式（简化显示 + 详情抽屉）
      
      使用场景：
      - 在表格单元格中显示嵌套的 form 字段
      - 避免表格列过宽，保持布局整洁
      
      渲染逻辑：
      1. 显示简化信息：根据字段数量显示 "共xx个字段"
      2. 点击按钮：打开抽屉查看完整内容
      3. 抽屉模式：根据 parentMode 决定使用 edit 还是 response 模式
         - parentMode='edit' → 抽屉使用 edit 模式（可编辑，有确认按钮）
         - parentMode='response' → 抽屉使用 response 模式（只读，无确认按钮）
      
      预期行为：
      - 表格单元格中只显示简化信息，不占用过多空间
      - 点击后可以在抽屉中查看和编辑完整内容
      - 编辑模式下可以修改数据，响应模式下只能查看
    -->
    <template v-else-if="mode === 'table-cell'">
      <el-button
        size="small"
        @click="tableCellMode.openDrawer()"
        class="form-field-button"
      >
        <span>共 {{ fieldCount }} 个字段</span>
        <el-icon class="form-field-button-icon">
          <View />
        </el-icon>
      </el-button>
      
      <!-- 详情抽屉（根据上下文支持编辑或只读） -->
      <el-drawer
        v-model="tableCellMode.showDrawer.value"
        :title="field.name"
        :size="DRAWER_CONFIG.size"
        destroy-on-close
        append-to-body
        @close="tableCellMode.handleDrawerClose()"
      >
        <template #default>
          <div class="form-detail-content">
            <div class="form-detail-panel">
              <el-form
                :model="formData"
                label-position="left"
                :label-width="FORM_LABEL_WIDTH"
              >
                <template v-for="subField in visibleSubFields" :key="subField.code">
                  <div v-if="labelsOnTop" class="form-field-label-top">
                    <label class="field-label">
                      {{ subField.name }}
                      <span v-if="isFieldRequired(subField)" class="required">*</span>
                    </label>
                    <el-form-item class="form-item-no-label">
                      <component
                        :is="getWidgetComponent(subField.widget?.type || 'input', tableCellMode.drawerMode.value)"
                        :field="subField"
                        :value="getSubFieldValue(subField.code)"
                        :model-value="getSubFieldValue(subField.code)"
                        @update:model-value="handleSubFieldModelUpdate(subField.code, $event)"
                        :field-path="`${fieldPath}.${subField.code}`"
                        :form-manager="formManager"
                        :form-renderer="formRenderer"
                        :mode="tableCellMode.drawerMode.value"
                        :depth="(depth || 0) + 1"
                      />
                    </el-form-item>
                  </div>
                  <el-form-item
                    v-else
                    :label="subField.name"
                    :required="isFieldRequired(subField)"
                  >
                    <component
                      :is="getWidgetComponent(subField.widget?.type || 'input', tableCellMode.drawerMode.value)"
                      :field="subField"
                      :value="getSubFieldValue(subField.code)"
                      :model-value="getSubFieldValue(subField.code)"
                      @update:model-value="handleSubFieldModelUpdate(subField.code, $event)"
                      :field-path="`${fieldPath}.${subField.code}`"
                      :form-manager="formManager"
                      :form-renderer="formRenderer"
                      :mode="tableCellMode.drawerMode.value"
                      :depth="(depth || 0) + 1"
                    />
                  </el-form-item>
                </template>
              </el-form>
            </div>
          </div>
        </template>
        <!-- 
          🔥 确认按钮只在编辑上下文中显示
          
          预期行为：
          - 编辑上下文：显示确认按钮，用户可以保存修改
          - 响应上下文：不显示确认按钮，因为数据是只读的
        -->
        <template #footer v-if="tableCellMode.isInEditContext.value">
          <div class="drawer-footer">
            <el-button @click="tableCellMode.cancelDrawer()">取消</el-button>
            <el-button type="primary" @click="handleFormCellConfirm">确认</el-button>
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
            :is="getWidgetComponent(subField.widget?.type || 'input', mode)"
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
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import { useFormWidget } from '@/architecture/presentation/widgets/composables/useFormWidget'
import { useTableCellMode } from '@/architecture/presentation/widgets/composables/useTableCellMode'
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import type { ValidationEngine, ValidationResult } from '@/core/validation'
import { validateFieldValue, validateFormWidgetNestedFields, type WidgetValidationContext } from '@/architecture/presentation/widgets/composables/useWidgetValidation'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { FORM_LABEL_WIDTH, FORM_QUESTIONNAIRE_TRIGGER_CHARS } from '@/architecture/presentation/utils/formLayout'

// 抽屉配置常量
const DRAWER_CONFIG = {
  size: '60%'
} as const

const props = defineProps<WidgetComponentProps>()
const formDataStore = useFormDataStore()

// 使用组合式函数
const { visibleSubFields, getSubFieldValue, updateSubFieldValue } = useFormWidget(props)

// table-cell 模式的公共逻辑
const tableCellMode = useTableCellMode(props)

// 字段数量（用于 table-cell 模式显示）
const fieldCount = computed(() => {
  const currentFieldValue = formDataStore.data.has(props.fieldPath)
    ? formDataStore.getValue(props.fieldPath)
    : props.value
  const raw = currentFieldValue?.raw
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    return Object.keys(raw).length
  }
  return visibleSubFields.value.length
})

// 处理 table-cell 模式的确认按钮
function handleFormCellConfirm(): void {
  tableCellMode.confirmDrawer()
}

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
function getWidgetComponent(type: string, widgetMode: string = props.mode) {
  if (widgetMode === 'response') {
    return widgetComponentFactory.getResponseComponent(type)
  }
  return widgetComponentFactory.getRequestComponent(type)
}

function handleSubFieldModelUpdate(subFieldCode: string, value: FieldValue): void {
  updateSubFieldValue(subFieldCode, value)
}

// 检查字段是否必填
function isFieldRequired(field: FieldConfig): boolean {
  const validation = field.validation || ''
  return validation.includes('required') && !validation.includes('omitempty')
}

const labelsOnTop = computed(() =>
  visibleSubFields.value.some((f) => (f.name?.length ?? 0) > FORM_QUESTIONNAIRE_TRIGGER_CHARS)
)

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
    fieldErrors,
    formDataStore
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

/* 短 label：右对齐，靠近右侧输入框，减少 label 与输入框间距 */
:deep(.form-widget-form .el-form-item:not(.form-item-no-label) .el-form-item__label) {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  text-align: right;
  padding-right: 8px;
}

/* Form 表单项间距 */
:deep(.form-widget-form .el-form-item) {
  margin-bottom: 20px;
}

:deep(.form-widget-form .el-form-item:last-child) {
  margin-bottom: 0;
}

/* 长 label：label 在上方 */
.form-field-label-top {
  margin-bottom: 20px;

  .field-label {
    display: block;
    font-size: 14px;
    color: var(--el-text-color-regular);
    margin-bottom: 8px;
    line-height: 1.4;
    text-align: left;

    .required {
      color: var(--el-color-danger);
      margin-left: 2px;
    }
  }

  .form-item-no-label {
    margin-bottom: 0;

    :deep(.el-form-item__label) {
      display: none;
    }
    :deep(.el-form-item__content) {
      margin-left: 0 !important;
    }
  }
}

/* 表格单元格模式 */
.form-field-button {
  padding: 0;
  height: auto;
  font-size: 14px;
  line-height: 1.4;
  border-color: transparent;
  background: transparent;
  color: var(--el-text-color-regular);
  justify-content: flex-start;
  max-width: 100%;
  text-align: left;
}

.form-field-button:hover,
.form-field-button:focus-visible {
  border-color: transparent;
  background: transparent;
  color: var(--el-text-color-primary);
}

.form-field-button-icon {
  margin-left: 4px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

/* 详情抽屉内容 */
.form-detail-content {
  padding: 24px;
  box-sizing: border-box;
  /* 确保下拉菜单可以正常显示 */
  overflow: visible;
  position: relative;
}

.form-detail-panel {
  box-sizing: border-box;
}

:deep(.form-detail-content .form-card),
:deep(.form-detail-content .table-card) {
  margin-bottom: 0;
}

:deep(.form-detail-content .el-card__body) {
  padding: 20px 22px;
}

:deep(.form-detail-content .form-widget-form .el-form-item:last-child) {
  margin-bottom: 0;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.detail-field {
  margin-bottom: 24px;
}

.detail-form {
  width: 100%;
}

/* 确保抽屉本身不会遮挡下拉菜单 */
:deep(.el-drawer__body) {
  overflow: visible !important;
}

:deep(.el-drawer) {
  overflow: visible !important;
}
</style>
