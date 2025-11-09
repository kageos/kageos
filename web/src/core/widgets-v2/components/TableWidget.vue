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
      <div class="table-widget-container">
        <div class="table-widget-header">
          <span class="table-title">{{ field.name }}</span>
        </div>
        <div class="table-widget-content">
          <el-table :data="editMode.tableData.value" border>
        <el-table-column
          v-for="itemField in itemFields"
          :key="itemField.code"
          :prop="itemField.code"
          :label="itemField.name"
          :min-width="getColumnWidth(itemField)"
        >
          <template #default="{ row, $index }">
            <!-- 🔥 对于 form 类型字段，在编辑和显示状态下都使用简化显示 + 抽屉 -->
            <!-- 这样可以避免表格列过宽，保持布局整洁 -->
            <template v-if="itemField.widget?.type === 'form'">
              <component
                :is="getWidgetComponent('form')"
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
      
      <!-- 聚合统计 -->
      <div v-if="statistics.statisticsConfig && Object.keys(statistics.statisticsConfig).length > 0" class="statistics">
        <div
          v-for="(value, label) in statistics.statisticsResult.value"
          :key="label"
          class="statistics-item"
        >
          <span class="statistics-label">{{ label }}:</span>
          <span class="statistics-value">{{ formatStatisticsValue(value) }}</span>
        </div>
      </div>
        </div>
      </div>
    </template>
    
    <!-- 响应模式（只读） -->
    <template v-else-if="mode === 'response'">
      <el-table :data="responseTableData" border>
        <el-table-column
          v-for="itemField in itemFields"
          :key="itemField.code"
          :prop="itemField.code"
          :label="itemField.name"
          :min-width="getColumnWidth(itemField)"
        >
          <template #default="{ row, $index }">
            <component
              :is="getWidgetComponent(itemField.widget?.type || 'input')"
              :field="itemField"
              :value="getResponseRowFieldValue($index, itemField.code)"
              :model-value="getResponseRowFieldValue($index, itemField.code)"
              :field-path="`${fieldPath}[${$index}].${itemField.code}`"
              mode="table-cell"
              :depth="(depth || 0) + 1"
            />
          </template>
        </el-table-column>
      </el-table>
      
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
    
    <!-- 表格单元格模式 -->
    <template v-else-if="mode === 'table-cell'">
      <span class="table-cell-value">
        {{ displayValue }}
      </span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElTable, ElTableColumn, ElButton, ElDrawer } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useTableWidget } from '../composables/useTableWidget'
import { useTableEditMode } from '../composables/useTableEditMode'
import { useTableResponseMode } from '../composables/useTableResponseMode'
import { useTableStatistics } from '../composables/useTableStatistics'
import { widgetComponentFactory } from '../../factories-v2'
import { FieldValue } from '../../types/field'
import { useFormDataStore } from '../../stores-v2/formData'

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
const statistics = useTableStatistics(props, getAllRowsData)

// 获取 formDataStore
const formDataStore = useFormDataStore()

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

// 显示值（用于 table-cell 模式）
const displayValue = computed(() => {
  const value = props.value
  if (!value) {
    return '-'
  }
  
  if (value.display) {
    return value.display
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  if (Array.isArray(raw)) {
    return `共 ${raw.length} 条`
  }
  
  // 避免序列化循环引用的对象
  if (typeof raw === 'object') {
    try {
      return JSON.stringify(raw)
    } catch (e) {
      // 如果序列化失败（循环引用），返回简单描述
      return `[对象]`
    }
  }
  
  return String(raw)
})

// 格式化统计值（避免循环引用和 computed ref）
function formatStatisticsValue(value: any): string {
  if (value === null || value === undefined) {
    return '-'
  }
  
  // 如果是 computed ref，获取其值
  if (value && typeof value === 'object' && '__v_isRef' in value && 'value' in value) {
    return formatStatisticsValue(value.value)
  }
  
  // 如果是基本类型，直接返回
  if (typeof value !== 'object') {
    return String(value)
  }
  
  // 如果是数组
  if (Array.isArray(value)) {
    return `[${value.length} 项]`
  }
  
  // 如果是对象，尝试序列化
  try {
    const str = JSON.stringify(value)
    // 如果序列化结果太长，截断
    if (str.length > 100) {
      return str.substring(0, 100) + '...'
    }
    return str
  } catch (e) {
    // 如果序列化失败（循环引用），返回简单描述
    return `[对象]`
  }
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
      
      // 确保 formDataStore 中有正确的值
      const fieldValue: FieldValue = {
        raw: rawValue,
        display: rawValue !== null && rawValue !== undefined ? String(rawValue) : '',
        meta: {}
      }
      formDataStore.setValue(fieldPath, fieldValue)
    })
  } catch (error) {
    console.error('[TableWidget] handleSave 错误:', error)
    throw error
  }
}

// 删除行
function handleDelete(index: number): void {
  editMode.deleteRow(index)
}
</script>

<style scoped>
.table-widget {
  width: 100%;
}

.table-actions {
  margin-top: 16px;
}

.statistics {
  margin-top: 16px;
  padding: 12px;
  background: var(--el-bg-color-page);
  border-radius: 4px;
}

.statistics-item {
  display: inline-block;
  margin-right: 24px;
  margin-bottom: 8px;
}

.statistics-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-right: 8px;
}

.statistics-value {
  color: var(--el-text-color-regular);
}

.table-cell-value {
  color: var(--el-text-color-regular);
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
</style>

