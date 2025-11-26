<!--
  TableWidget - 表格容器组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 支持 mode="edit" - 可编辑表格（新增、编辑、删除）
  - 支持 mode="response" - 只读表格
  - 支持 mode="table-cell" - 表格单元格
  - 聚合计算（使用 computed）
  - 详情抽屉
-->

<template>
  <div class="table-widget">
    <!-- 编辑模式 -->
    <template v-if="mode === 'edit'">
      <el-card
        shadow="hover"
        class="table-card"
      >
        <template #header>
          <div class="table-card-header">
            <span class="table-title">{{ field.name }}</span>
            <div class="table-header-actions">
              <el-button size="small" @click="handleImport">
                <el-icon><Upload /></el-icon>
                导入
              </el-button>
              <el-button size="small" @click="handleExport">
                <el-icon><Download /></el-icon>
                导出
              </el-button>
            </div>
          </div>
        </template>
        <div class="table-widget-content">
          <el-table :data="editMode.tableData.value" :stripe="false" class="table-widget-table">
        <el-table-column
          v-for="itemField in itemFields"
          :key="itemField.code"
          :prop="itemField.code"
          :label="itemField.name"
          :min-width="getColumnWidth(itemField)"
        >
          <template #default="{ row, $index }">
            <!-- 🔥 对于 form 和 table 类型字段，在编辑和显示状态下都使用简化显示 + 抽屉 -->
            <!-- 这样可以避免表格列过宽，保持布局整洁 -->
            <template v-if="itemField.widget?.type === 'form' || itemField.widget?.type === 'table'">
              <component
                :is="getWidgetComponent(itemField.widget?.type)"
                :field="itemField"
                :value="getRowFieldValue($index, itemField.code)"
                :model-value="getRowFieldValue($index, itemField.code)"
                @update:model-value="(v) => updateRowFieldValue($index, itemField.code, v)"
                :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                :form-manager="formManager"
                :form-renderer="formRenderer"
                mode="table-cell"
                :depth="(depth || 0) + 1"
              />
            </template>
            <!-- 其他类型字段：编辑状态直接编辑，显示状态简化显示 -->
            <template v-else>
              <!-- 编辑状态 -->
              <template v-if="editMode.editingIndex.value === $index">
                <component
                  :is="getWidgetComponent(itemField.widget?.type || 'input')"
                  :field="itemField"
                  :value="getRowFieldValue($index, itemField.code)"
                  :model-value="getRowFieldValue($index, itemField.code)"
                  @update:model-value="(v) => updateRowFieldValue($index, itemField.code, v)"
                  :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                  :form-manager="formManager"
                  :form-renderer="formRenderer"
                  mode="edit"
                  :depth="(depth || 0) + 1"
                />
              </template>
              <!-- 显示状态 -->
              <template v-else>
                <component
                  :is="getWidgetComponent(itemField.widget?.type || 'input')"
                  :field="itemField"
                  :value="getRowFieldValue($index, itemField.code)"
                  :model-value="getRowFieldValue($index, itemField.code)"
                  :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                  mode="table-cell"
                  :depth="(depth || 0) + 1"
                />
              </template>
            </template>
          </template>
        </el-table-column>
        
        <!-- 操作列 -->
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ $index }">
            <template v-if="editMode.editingIndex.value === $index">
              <el-button size="small" @click="handleSave($index)">保存</el-button>
              <el-button size="small" @click="editMode.cancelEditing()">取消</el-button>
            </template>
            <template v-else>
              <el-button size="small" @click="editMode.startEditing($index)">编辑</el-button>
              <el-button size="small" type="danger" @click="handleDelete($index)">删除</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
      
      <!-- 新增按钮 -->
      <div class="table-actions">
        <el-button type="primary" @click="editMode.startAdding()">新增</el-button>
      </div>
      
      <!-- 🔥 当前编辑行的字段统计信息（显示在表格下方） -->
      <!-- 🔥 使用所有行的数据来计算统计（表格场景） -->
      <FieldStatistics
        v-if="editingRowStatistics && Object.keys(editingRowStatistics).length > 0"
        :field="field"
        :value="getAllRowsData()"
        :statistics="editingRowStatistics"
      />
        </div>
      </el-card>
    </template>
    
    <!-- 响应模式（只读） -->
    <template v-else-if="mode === 'response'">
      <el-card
        shadow="never"
        class="table-card response-table-card"
      >
        <template #header>
          <div class="table-card-header">
            <span class="table-title">{{ field.name }}</span>
          </div>
        </template>
        <div class="table-widget-content">
          <el-table :data="responseTableData" :stripe="false" class="table-widget-table">
            <el-table-column
              v-for="itemField in itemFields"
              :key="itemField.code"
              :prop="itemField.code"
              :label="itemField.name"
              :min-width="getColumnWidth(itemField)"
            >
              <template #default="{ row, $index }">
                <!-- 🔥 对于 form 和 table 类型字段，使用 table-cell 模式显示（简化显示 + 详情抽屉） -->
                <!-- 这样可以避免表格列过宽，保持布局整洁 -->
                <template v-if="itemField.widget?.type === 'form' || itemField.widget?.type === 'table'">
                  <component
                    :is="getWidgetComponent(itemField.widget?.type)"
                    :field="itemField"
                    :value="getResponseRowFieldValue($index, itemField.code)"
                    :model-value="getResponseRowFieldValue($index, itemField.code)"
                    :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                    :form-manager="formManager"
                    :form-renderer="formRenderer"
                    mode="table-cell"
                    :depth="(depth || 0) + 1"
                  />
                </template>
                <!-- 🔥 其他类型字段：使用共享的渲染函数（与 TableRenderer 一致） -->
                <template v-else>
                  <template v-if="getCellContent(itemField, row[itemField.code]).isString">
                    {{ getCellContent(itemField, row[itemField.code]).content }}
                  </template>
                  <!-- 🔥 VNode 直接渲染：使用 render 函数 -->
                  <CellRenderer v-else :vnode="getCellContent(itemField, row[itemField.code]).content" />
                </template>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-card>
      
      <!-- 详情抽屉 -->
      <el-drawer
        v-model="responseMode.showDetailDrawer.value"
        title="详细信息"
        size="50%"
        destroy-on-close
      >
        <template #default>
          <div v-if="responseMode.currentDetailRow.value">
            <div
              v-for="itemField in itemFields"
              :key="itemField.code"
              class="detail-field"
            >
              <div class="field-label">{{ itemField.name }}</div>
              <div class="field-value">
                <component
                  :is="getWidgetComponent(itemField.widget?.type || 'input')"
                  :field="itemField"
                  :value="getResponseRowFieldValue(responseMode.currentDetailIndex.value, itemField.code)"
                  :model-value="getResponseRowFieldValue(responseMode.currentDetailIndex.value, itemField.code)"
                  :field-path="`${fieldPath}[${responseMode.currentDetailIndex.value}].${itemField.code}`"
                  mode="detail"
                  :depth="(depth || 0) + 1"
                />
              </div>
            </div>
          </div>
        </template>
      </el-drawer>
    </template>
    
    <!-- 表格单元格模式（简化显示 + 详情抽屉） -->
    <template v-else-if="mode === 'table-cell'">
      <el-button
        link
        type="primary"
        size="small"
        @click="tableCellMode.showDrawer.value = true"
        class="table-cell-button"
      >
        <span>{{ displayValue }}</span>
        <el-icon style="margin-left: 4px">
          <View />
        </el-icon>
      </el-button>
      
      <!-- 详情抽屉（支持编辑） -->
      <el-drawer
        v-model="tableCellMode.showDrawer.value"
        :title="field.name"
        size="70%"
        destroy-on-close
        :z-index="3000"
        append-to-body
      >
        <template #default>
          <div class="table-detail-content">
            <!-- 🔥 抽屉中根据上下文使用 edit 或 response 模式的渲染逻辑 -->
            <component
              :is="getWidgetComponent('table')"
              :field="field"
              :value="value"
              :model-value="value"
              @update:model-value="(v) => emit('update:modelValue', v)"
              :field-path="fieldPath"
              :form-manager="formManager"
              :form-renderer="formRenderer"
              :mode="drawerMode"
              :depth="(depth || 0) + 1"
            />
          </div>
        </template>
        <template #footer v-if="isInEditContext">
          <div class="drawer-footer">
            <el-button @click="tableCellMode.showDrawer.value = false">取消</el-button>
            <el-button type="primary" @click="handleTableCellConfirm">确认</el-button>
          </div>
        </template>
      </el-drawer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, ref } from 'vue'
import { ElTable, ElTableColumn, ElButton, ElDrawer, ElCard, ElIcon } from 'element-plus'
import { Upload, Download, View } from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useTableWidget } from '../composables/useTableWidget'
import { useTableEditMode } from '../composables/useTableEditMode'
import { useTableResponseMode } from '../composables/useTableResponseMode'
import { widgetComponentFactory } from '../../factories-v2'
import { FieldValue, type FieldConfig } from '../../types/field'
import { useFormDataStore } from '../../stores-v2/formData'
import type { ValidationEngine, ValidationResult } from '../../validation/types'
import { validateFieldValue, validateTableWidgetNestedFields, type WidgetValidationContext } from '../composables/useWidgetValidation'
import { Logger } from '../../utils/logger'
import { renderTableCell } from '../../utils/tableCellRenderer'
import FieldStatistics from './FieldStatistics.vue'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  } as FieldValue)
})
const emit = defineEmits<WidgetComponentEmits>()

// 使用组合式函数
const { tableData, itemFields, getRowFieldValue, updateRowFieldValue, getAllRowsData } = useTableWidget(props)
const editMode = useTableEditMode(props)
const responseMode = useTableResponseMode()

// table-cell 模式的状态管理
const tableCellMode = {
  showDrawer: ref(false)
}

// 🔥 判断 table-cell 模式是在编辑上下文还是响应上下文中使用
// 如果 formDataStore 中有这个字段的值，说明是在编辑模式中；否则是在响应模式中
const isInEditContext = computed(() => {
  const value = formDataStore.getValue(props.fieldPath)
  return value !== null && value !== undefined && value.raw !== null && value.raw !== undefined
})

// 🔥 table-cell 模式抽屉中使用的模式（根据上下文决定）
const drawerMode = computed(() => {
  return isInEditContext.value ? 'edit' : 'response'
})

// 获取 formDataStore
const formDataStore = useFormDataStore()

// 🔥 当前编辑行的字段统计信息（用于显示在表格下方）
// 收集当前编辑行所有字段的 statistics 配置，合并成一个对象
// 🔥 注意：保存后 editingIndex 会变成 null，但我们需要继续显示统计信息
// 所以需要检查是否有保存后的行数据
const editingRowStatistics = computed(() => {
  // 🔥 优先使用当前编辑行的数据
  let targetIndex = editMode.editingIndex.value
  
  // 如果不在编辑状态，尝试使用最后保存的行（通常是最后一行）
  if (targetIndex === null || targetIndex === undefined) {
    // 检查是否有数据行
    if (tableData.value.length > 0) {
      // 使用最后一行（通常是刚保存的）
      targetIndex = tableData.value.length - 1
    } else {
      return {}
    }
  }
  
  // 收集当前编辑行所有字段的 statistics 配置
  const rowStatistics: Record<string, string> = {}
  
  itemFields.value.forEach((itemField: any) => {
    const fieldPath = `${props.fieldPath}[${targetIndex}].${itemField.code}`
    const itemValue = formDataStore.getValue(fieldPath)
    
    // 如果该字段有 statistics 配置，收集它
    if (itemValue?.meta?.statistics && typeof itemValue.meta.statistics === 'object') {
      Object.entries(itemValue.meta.statistics).forEach(([label, expression]) => {
        if (typeof expression === 'string') {
          rowStatistics[label] = expression
        }
      })
    }
  })
  
  return rowStatistics
})

// 🔥 当前编辑行的字段值（用于 FieldStatistics 组件）
// 构建一个包含所有字段 displayInfo 的对象，用于 FieldStatistics 计算
// 🔥 注意：保存后 editingIndex 会变成 null，但我们需要继续显示统计信息
// 所以需要检查是否有保存后的行数据
const editingRowFieldValue = computed(() => {
  // 🔥 优先使用当前编辑行的数据
  let targetIndex = editMode.editingIndex.value
  
  // 如果不在编辑状态，尝试使用最后保存的行（通常是最后一行）
  if (targetIndex === null || targetIndex === undefined) {
    // 检查是否有数据行
    if (tableData.value.length > 0) {
      // 使用最后一行（通常是刚保存的）
      targetIndex = tableData.value.length - 1
    } else {
      return null
    }
  }
  
  // 🔥 构建一个包含所有字段 displayInfo 的对象
  // FieldStatistics 期望 value 是一个对象，包含 meta.displayInfo 或直接是 displayInfo
  const rowData: Record<string, any> = {
    meta: {
      displayInfo: {}
    }
  }
  
  itemFields.value.forEach((itemField: any) => {
    const fieldPath = `${props.fieldPath}[${targetIndex}].${itemField.code}`
    const itemValue = formDataStore.getValue(fieldPath)
    
    // 🔥 合并 displayInfo（来自 Select 回调）
    // FieldStatistics 会从 value.meta.displayInfo 中查找
    if (itemValue?.meta?.displayInfo && typeof itemValue.meta.displayInfo === 'object') {
      Object.assign(rowData.meta.displayInfo, itemValue.meta.displayInfo)
    }
  })
  
  // 如果没有任何 displayInfo，返回 null
  if (Object.keys(rowData.meta.displayInfo).length === 0) {
    return null
  }
  
  return rowData
})

// 响应模式下的表格数据（从 props.value.raw 读取）
const responseTableData = computed(() => {
  if (props.mode === 'response') {
    return Array.isArray(props.value?.raw) ? props.value.raw : []
  }
  return []
})

// 响应模式下获取行的字段值（从 row 数据直接读取）
function getResponseRowFieldValue(rowIndex: number, fieldCode: string): FieldValue {
  if (props.mode !== 'response') {
    return { raw: null, display: '', meta: {} }
  }
  
  const tableData = responseTableData.value
  if (!tableData || rowIndex < 0 || rowIndex >= tableData.length) {
    return { raw: null, display: '', meta: {} }
  }
  
  const row = tableData[rowIndex]
  const rawValue = row?.[fieldCode]
  
  return {
    raw: rawValue ?? null,
    display: rawValue !== null && rawValue !== undefined 
      ? (typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue))
      : '',
    meta: {}
  }
}

/**
 * 🔥 获取表格单元格内容（用于模板，与 TableRenderer 一致）
 * 
 * 使用共享的渲染函数，确保渲染逻辑一致
 */
function getCellContent(field: FieldConfig, rawValue: any): { content: any, isString: boolean } {
  return renderTableCell(field, rawValue, {
    mode: 'table-cell',
    userInfoMap: props.userInfoMap || new Map(),
    fieldPath: field.code,
    formRenderer: props.formRenderer,
    formManager: props.formManager
  })
}

// 🔥 VNode 渲染组件（用于在模板中渲染 VNode，避免循环引用）
const CellRenderer = defineComponent({
  props: {
    vnode: {
      type: Object,
      required: true
    }
  },
  setup(props: { vnode: any }) {
    return () => props.vnode
  }
})

// 显示值（用于 table-cell 模式）
const displayValue = computed(() => {
  const value = props.value
  if (!value) {
    return '共 0 条记录'
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return '共 0 条记录'
  }
  
  if (Array.isArray(raw)) {
    return `共 ${raw.length} 条记录`
  }
  
  // 避免序列化循环引用的对象
  if (typeof raw === 'object') {
    try {
      return JSON.stringify(raw)
    } catch (e) {
      // 如果序列化失败（循环引用），返回简单描述
      return `共 0 条记录`
    }
  }
  
  return String(raw)
})

// 处理 table-cell 模式的确认按钮
function handleTableCellConfirm(): void {
  // 关闭抽屉即可，数据已经通过 update:modelValue 事件更新
  tableCellMode.showDrawer.value = false
}


// 获取列宽
function getColumnWidth(field: any): number {
  // 简单的列宽计算（可以根据需要扩展）
  const type = field.widget?.type || 'input'
  
  if (type === 'timestamp') {
    return 180
  }
  if (type === 'switch') {
    return 100
  }
  if (type === 'number' || type === 'float') {
    return 120
  }
  
  return 150
}

// 获取组件
function getWidgetComponent(type: string) {
  return widgetComponentFactory.getRequestComponent(type)
}

// 保存行
function handleSave(index: number): void {
  try {
    // 收集当前行的数据，并确保 formDataStore 中的值都被正确设置
    const rowData: Record<string, any> = {}
    
    itemFields.value.forEach(itemField => {
      const fieldPath = `${props.fieldPath}[${index}].${itemField.code}`
      const value = getRowFieldValue(index, itemField.code)
      
      // 确保值存在，如果不存在则使用默认值
      const fieldValue: FieldValue = value || {
        raw: null,
        display: '',
        meta: {}
      }
      
      // 确保 formDataStore 中有这个值
      formDataStore.setValue(fieldPath, fieldValue)
      
      // 收集到 rowData 中
      rowData[itemField.code] = fieldValue.raw ?? null
    })
    
    // 保存行（这会更新 tableData，从而更新 formDataStore 中的整个数组）
    // 在 saveRow 之前保存状态，因为 saveRow 会调用 cancelEditing() 重置状态
    const wasAdding = editMode.isAdding.value
    const currentLength = tableData.value.length
    
    editMode.saveRow(rowData)
    
    // 保存后，再次确保 formDataStore 中每个字段路径的值都是最新的
    // 如果是新增，索引会变成数组的最后一个索引
    const finalIndex = wasAdding ? currentLength : index
    
    itemFields.value.forEach(itemField => {
      const fieldPath = `${props.fieldPath}[${finalIndex}].${itemField.code}`
      const rawValue = rowData[itemField.code]
      
      // 🔥 获取保存前的值，保留 meta 信息（displayInfo、statistics 等）
      const previousValue = getRowFieldValue(index, itemField.code)
      const previousMeta = previousValue?.meta || {}
      
      // 确保 formDataStore 中有正确的值，并保留 meta 信息
      const fieldValue: FieldValue = {
        raw: rawValue,
        display: rawValue !== null && rawValue !== undefined ? String(rawValue) : '',
        meta: {
          ...previousMeta, // 🔥 保留原有的 meta 信息（displayInfo、statistics 等）
        }
      }
      formDataStore.setValue(fieldPath, fieldValue)
    })
  } catch (error) {
    Logger.error('TableWidget', 'handleSave 错误', error)
    throw error
  }
}

// 删除行
function handleDelete(index: number): void {
  editMode.deleteRow(index)
}

/**
 * 验证当前 Widget 及其嵌套字段
 * 
 * 符合依赖倒置原则：TableWidget 自己负责验证嵌套字段
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
  
  // 2. 验证嵌套字段（TableWidget 自己负责）
  const nestedErrors = validateTableWidgetNestedFields(props.field, props.fieldPath, context)
  
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

// 处理导入
function handleImport(): void {
  // TODO: 实现导入功能
  console.log('导入功能待实现')
}

// 处理导出
function handleExport(): void {
  // TODO: 实现导出功能
  console.log('导出功能待实现')
}

// 🔥 暴露验证方法给父组件
defineExpose({
  validate
})
</script>

<style scoped>
.table-widget {
  width: 100%;
}

/* 🔥 表格卡片样式（参考 FormWidget，保持样式一致） */
.table-card {
  width: 100%;
  margin-bottom: 24px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
}

.table-card:last-child {
  margin-bottom: 0;
}

.table-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.table-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.table-header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.table-widget-content {
  width: 100%;
  padding: 0;
}

/* 响应模式表格卡片样式 */
.response-table-card {
  background-color: var(--el-bg-color-page);
}

.table-actions {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-extra-light);
}


.table-cell-value {
  color: var(--el-text-color-regular);
}

.table-cell-button {
  padding: 0;
  height: auto;
  font-size: 14px;
}

/* 详情抽屉内容 */
.table-detail-content {
  padding: 16px 0;
  /* 确保下拉菜单可以正常显示 */
  overflow: visible;
  position: relative;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.detail-field {
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

/* 🔥 表格样式（与 TableRenderer 一致，移除边框和斑马纹） */
:deep(.table-widget-table) {
  background-color: var(--el-bg-color) !important;
}

/* 🔥 移除表格边框（左右竖线） */
:deep(.table-widget-table) {
  border: none !important;
}

:deep(.table-widget-table .el-table__inner-wrapper) {
  border: none !important;
}

:deep(.table-widget-table .el-table__header-wrapper) {
  border: none !important;
}

:deep(.table-widget-table .el-table__body-wrapper) {
  border: none !important;
}

:deep(.table-widget-table th),
:deep(.table-widget-table td) {
  border-right: none !important;
  border-left: none !important;
}

:deep(.table-widget-table th:first-child),
:deep(.table-widget-table td:first-child) {
  border-left: none !important;
}

:deep(.table-widget-table th:last-child),
:deep(.table-widget-table td:last-child) {
  border-right: none !important;
}

:deep(.table-widget-table .el-table__body tr) {
  background-color: var(--el-bg-color) !important;
}

/* 🔥 移除斑马纹：确保所有行背景色一致 */
:deep(.table-widget-table .el-table__body tr.el-table__row--striped) {
  background-color: var(--el-bg-color) !important;
}

:deep(.table-widget-table .el-table__body tr.el-table__row--striped td) {
  background-color: var(--el-bg-color) !important;
}

:deep(.table-widget-table .el-table__body tr:hover > td) {
  background-color: var(--el-fill-color-light) !important;
}

/* 🔥 确保表格的所有列（包括 fixed 列）不会遮挡对话框 */
/* 🔥 使用极低的 z-index，确保对话框（z-index: 10000）始终在上方 */
:deep(.el-table__fixed-right),
:deep(.el-table__fixed-left) {
  z-index: 0 !important;
}

:deep(.el-table__fixed-right .el-table__fixed-body-wrapper),
:deep(.el-table__fixed-left .el-table__fixed-body-wrapper) {
  z-index: 0 !important;
}

/* 🔥 确保表格单元格内容不会遮挡对话框 */
:deep(.el-table__body-wrapper),
:deep(.el-table__header-wrapper) {
  z-index: 0 !important;
}

:deep(.el-table__body tr),
:deep(.el-table__body td),
:deep(.el-table__header tr),
:deep(.el-table__header th) {
  z-index: 0 !important;
  position: relative;
}

/* 🔥 确保表格单元格内的组件不会遮挡对话框 */
:deep(.el-table__body td > *),
:deep(.el-table__body td .el-input),
:deep(.el-table__body td .el-select),
:deep(.el-table__body td .el-button) {
  z-index: 0 !important;
  position: relative;
}

/* 🔥 确保编辑状态下的组件不会遮挡对话框 */
:deep(.el-table__body td .select-widget),
:deep(.el-table__body td .edit-select),
:deep(.el-table__body td .select-container),
:deep(.el-table__body td .multi-select-widget),
:deep(.el-table__body td .number-widget),
:deep(.el-table__body td .input-widget),
:deep(.el-table__body td .float-widget) {
  z-index: 0 !important;
  position: relative;
}

/* 🔥 确保编辑状态下的组件内的所有子元素也不会遮挡对话框 */
:deep(.el-table__body td .select-widget *),
:deep(.el-table__body td .edit-select *),
:deep(.el-table__body td .multi-select-widget *),
:deep(.el-table__body td .number-widget *),
:deep(.el-table__body td .input-widget *),
:deep(.el-table__body td .float-widget *) {
  z-index: 0 !important;
}
</style>

