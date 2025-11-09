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
      <el-table :data="editMode.tableData.value" border>
        <el-table-column
          v-for="itemField in itemFields"
          :key="itemField.code"
          :prop="itemField.code"
          :label="itemField.name"
          :min-width="getColumnWidth(itemField)"
        >
          <template #default="{ row, $index }">
            <!-- 编辑状态 -->
            <template v-if="editMode.editingIndex.value === $index || editMode.isAdding.value">
              <component
                :is="getWidgetComponent(itemField.widget?.type || 'input')"
                :field="itemField"
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
                :model-value="getRowFieldValue($index, itemField.code)"
                :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                mode="table-cell"
                :depth="(depth || 0) + 1"
              />
            </template>
          </template>
        </el-table-column>
        
        <!-- 操作列 -->
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ $index }">
            <template v-if="editMode.editingIndex.value === $index || editMode.isAdding.value">
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
          v-for="(value, label) in statistics.statisticsResult"
          :key="label"
          class="statistics-item"
        >
          <span class="statistics-label">{{ label }}:</span>
          <span class="statistics-value">{{ value }}</span>
        </div>
      </div>
    </template>
    
    <!-- 响应模式（只读） -->
    <template v-else-if="mode === 'response'">
      <el-table :data="tableData" border>
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
              :model-value="getRowFieldValue($index, itemField.code)"
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
                  :model-value="getRowFieldValue(responseMode.currentDetailIndex.value, itemField.code)"
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

const props = defineProps<WidgetComponentProps>()
const emit = defineEmits<WidgetComponentEmits>()

// 使用组合式函数
const { tableData, itemFields, getRowFieldValue, updateRowFieldValue, getAllRowsData } = useTableWidget(props)
const editMode = useTableEditMode(props)
const responseMode = useTableResponseMode()
const statistics = useTableStatistics(props, getAllRowsData)

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
  
  return String(raw)
})

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
  // 收集当前行的数据
  const rowData: Record<string, any> = {}
  
  itemFields.value.forEach(itemField => {
    const fieldPath = `${props.fieldPath}[${index}].${itemField.code}`
    const value = getRowFieldValue(index, itemField.code)
    rowData[itemField.code] = value?.raw
  })
  
  editMode.saveRow(rowData)
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

