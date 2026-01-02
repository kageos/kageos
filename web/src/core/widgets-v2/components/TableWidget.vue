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
          :align="getColumnAlign(itemField)"
          header-align="left"
        >
          <template #default="{ row, $index }">
            <!-- 
              🔥 嵌套字段渲染策略（edit 模式）
              
              问题：在表格单元格中直接渲染嵌套的 form/table 字段会导致：
              - 表格列过宽，布局混乱
              - 嵌套表格/表单占用大量空间，影响用户体验
              
              解决方案：
              - 对于 form 和 table 类型字段，统一使用 table-cell 模式显示
              - table-cell 模式会显示为简化形式（"共xx个字段"、"共xx条记录"）
              - 点击后打开抽屉，在抽屉中使用 edit 模式渲染完整内容，支持编辑
              
              关键点：
              - mode="table-cell"：使用表格单元格模式，显示简化信息
              - parent-mode="mode"：传递父级模式（这里是 'edit'），让嵌套组件知道上下文
              - 嵌套组件会根据 parentMode 判断：如果是 'edit'，抽屉中使用 edit 模式（可编辑）
            -->
            <template v-if="isNestedContainerField(itemField)">
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
                :parent-mode="mode"
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
        <el-table-column label="操作" width="150" fixed="right" header-align="left">
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
              :align="getColumnAlign(itemField)"
              header-align="left"
            >
              <template #default="{ row, $index }">
                <!-- 
                  🔥 嵌套字段渲染策略（response 模式）
                  
                  问题：在响应数据的表格中，嵌套的 form/table 字段如果直接渲染完整内容，会导致：
                  - 表格被撑爆，布局混乱
                  - 数据展示不清晰，难以阅读
                  
                  解决方案：
                  - 对于 form 和 table 类型字段，统一使用 table-cell 模式显示
                  - table-cell 模式会显示为简化形式（"共xx个字段"、"共xx条记录"）
                  - 点击后打开抽屉，在抽屉中使用 response 模式渲染完整内容，只读展示
                  
                  关键点：
                  - mode="table-cell"：使用表格单元格模式，显示简化信息
                  - parent-mode="mode"：传递父级模式（这里是 'response'），让嵌套组件知道上下文
                  - 嵌套组件会根据 parentMode 判断：如果是 'response'，抽屉中使用 response 模式（只读）
                -->
                <template v-if="isNestedContainerField(itemField)">
                  <component
                    :is="getWidgetComponent(itemField.widget?.type || 'input')"
                    :field="itemField"
                    :value="getResponseRowFieldValue($index, itemField.code)"
                    :model-value="getResponseRowFieldValue($index, itemField.code)"
                    :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                    :form-manager="formManager"
                    :form-renderer="formRenderer"
                    mode="table-cell"
                    :parent-mode="mode"
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
    
    <!-- 
      🔥 表格单元格模式（简化显示 + 详情抽屉）
      
      使用场景：
      - 在表格单元格中显示嵌套的 table 字段
      - 避免表格列过宽，保持布局整洁
      
      渲染逻辑：
      1. 显示简化信息：根据数据量显示 "共xx条记录"
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
        link
        type="primary"
        size="small"
        @click="tableCellMode.openDrawer()"
        class="table-cell-button"
      >
        <span>{{ displayValue }}</span>
        <el-icon style="margin-left: 4px">
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
      >
        <template #default>
          <div class="table-detail-content">
            <!-- 
              🔥 抽屉中根据上下文使用 edit 或 response 模式的渲染逻辑
              
              drawerMode 的值由 isInEditContext 决定：
              - 编辑上下文：drawerMode = 'edit' → 可编辑，支持数据修改
              - 响应上下文：drawerMode = 'response' → 只读，仅展示数据
            -->
            <component
              :is="getWidgetComponent('table')"
              :field="field"
              :value="value"
              :model-value="value"
              @update:model-value="(v) => emit('update:modelValue', v)"
              :field-path="fieldPath"
              :form-manager="formManager"
              :form-renderer="formRenderer"
              :mode="tableCellMode.drawerMode.value"
              :depth="(depth || 0) + 1"
            />
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
            <el-button @click="tableCellMode.closeDrawer()">取消</el-button>
            <el-button type="primary" @click="handleTableCellConfirm">确认</el-button>
          </div>
        </template>
      </el-drawer>
    </template>
    
    <!-- 导入对话框 -->
    <el-dialog
      v-model="importDialogVisible"
      title="批量导入"
      width="80%"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <template #default>
        <div class="import-dialog-content">
          <!-- 步骤1: 选择文件 -->
          <div v-if="!importFile" class="import-step">
            <h3>步骤 1: 选择 Excel 文件</h3>
            <el-upload
              :auto-upload="false"
              :on-change="handleFileSelect"
              :show-file-list="false"
              accept=".xlsx,.xls"
            >
              <el-button type="primary">
                <el-icon><Upload /></el-icon>
                选择文件
              </el-button>
            </el-upload>
            <div style="margin-top: 16px;">
              <el-button
                type="text"
                @click="handleDownloadTemplate"
                :loading="downloadingTemplate"
              >
                <el-icon><Download /></el-icon>
                下载导入模板
              </el-button>
            </div>
          </div>
          
          <!-- 步骤2: 预览数据 -->
          <div v-else class="import-step">
            <h3>步骤 2: 预览数据</h3>
            <div class="import-info">
              <p>文件: {{ importFile.name }}</p>
              <p>共解析 {{ importData.length }} 条数据</p>
              <p v-if="importErrors.length > 0" style="color: #f56c6c;">
                发现 {{ importErrors.length }} 个错误
              </p>
            </div>
            
            <!-- 错误列表 -->
            <el-alert
              v-if="importErrors.length > 0"
              type="error"
              :closable="false"
              style="margin-bottom: 16px;"
            >
              <template #title>
                <div>
                  <p>数据验证失败，请修正以下错误：</p>
                  <ul style="margin: 8px 0 0 20px;">
                    <li v-for="error in importErrors" :key="`${error.index}-${error.field}`">
                      第 {{ error.index + 1 }} 行，字段 "{{ error.field }}": {{ error.error }}
                    </li>
                  </ul>
                </div>
              </template>
            </el-alert>
            
            <!-- 数据预览表格 -->
            <el-table
              :data="importData"
              max-height="400"
              border
              stripe
            >
              <el-table-column type="index" label="行号" width="60" />
              <el-table-column
                v-for="field in itemFields"
                :key="field.code"
                :prop="field.code"
                :label="field.name"
                :min-width="120"
              >
                <template #default="{ row, $index }">
                  <span
                    :class="{
                      'error-cell': importErrors.some(e => e.index === $index && e.field === field.code)
                    }"
                  >
                    {{ row[field.code] ?? '' }}
                  </span>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </template>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="importDialogVisible = false">取消</el-button>
          <el-button
            v-if="importFile"
            @click="handleReSelectFile"
          >
            重新选择
          </el-button>
          <el-button
            v-if="importFile && importErrors.length === 0"
            type="primary"
            @click="handleSubmitImport"
            :loading="importing"
          >
            确认导入
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, ref } from 'vue'
import { ElTable, ElTableColumn, ElButton, ElDrawer, ElCard, ElIcon, ElDialog, ElUpload, ElAlert, ElMessage } from 'element-plus'
import { Upload, Download, View } from '@element-plus/icons-vue'
import * as XLSX from 'xlsx'
import { download, post } from '@/utils/request'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useTableWidget } from '../composables/useTableWidget'
import { useTableEditMode } from '../composables/useTableEditMode'
import { useTableResponseMode } from '../composables/useTableResponseMode'
import { useTableCellMode } from '../composables/useTableCellMode'
import { widgetComponentFactory } from '../../factories-v2'
import { FieldValue, type FieldConfig } from '../../types/field'
import { useFormDataStore } from '../../stores-v2/formData'
import { createEmptyFieldValue, createFieldValue } from '../utils/createFieldValue'
import type { ValidationEngine, ValidationResult } from '../../validation/types'
import { validateFieldValue, validateTableWidgetNestedFields, type WidgetValidationContext } from '../composables/useWidgetValidation'
import { Logger } from '../../utils/logger'
import { renderTableCell } from '../../utils/tableCellRenderer'
import FieldStatistics from './FieldStatistics.vue'

// 抽屉配置常量
const DRAWER_CONFIG = {
  size: '70%'
} as const

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

// table-cell 模式的公共逻辑
const tableCellMode = useTableCellMode(props)

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
  // 🔥 查找对应的 itemField（优先使用 itemField，如果没有则使用 props.field）
  const itemField = itemFields.value.find(f => f.code === fieldCode) || props.field
  
  if (props.mode !== 'response') {
    // 🔥 使用 createEmptyFieldValue 确保结构一致
    return createEmptyFieldValue(itemField)
  }
  
  const tableData = responseTableData.value
  if (!tableData || rowIndex < 0 || rowIndex >= tableData.length) {
    // 🔥 使用 createEmptyFieldValue 确保结构一致
    return createEmptyFieldValue(itemField)
  }
  
  const row = tableData[rowIndex]
  const rawValue = row?.[fieldCode]
  
  const display = rawValue !== null && rawValue !== undefined 
    ? (typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue))
    : ''
  
  // 🔥 使用 createFieldValue 确保结构一致
  return createFieldValue(
    itemField,
    rawValue ?? null,
    display
  )
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
  tableCellMode.closeDrawer()
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

// 获取列对齐方式
function getColumnAlign(field: any): 'left' | 'center' | 'right' {
  // 🔥 优先使用字段配置中的对齐方式
  const configAlign = field.widget?.config?.align
  if (configAlign === 'left' || configAlign === 'center' || configAlign === 'right') {
    return configAlign
  }
  
  // 🔥 所有列统一左对齐
  return 'left'
}

// 获取组件
function getWidgetComponent(type: string) {
  return widgetComponentFactory.getRequestComponent(type)
}

/**
 * 判断字段是否为嵌套容器类型（form 或 table）
 */
function isNestedContainerField(field: FieldConfig): boolean {
  return field.widget?.type === 'form' || field.widget?.type === 'table'
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
      
      // 收集到 rowData 中（只保存 raw 值）
      rowData[itemField.code] = fieldValue.raw ?? null
    })
    
    // 保存行（这会更新 tableData，从而更新 formDataStore 中的整个数组）
    editMode.saveRow(rowData)
    
    // 保存后，再次确保 formDataStore 中每个字段路径的值都是最新的
    // 🔥 无论新增还是编辑，都使用 index（因为 saveRow 已经把数据保存到正确位置了）
    const finalIndex = index
    
    itemFields.value.forEach(itemField => {
      const fieldPath = `${props.fieldPath}[${finalIndex}].${itemField.code}`
      const rawValue = rowData[itemField.code]
      
      // 🔥 获取当前的值，保留 meta 和 display 信息
      const currentValue = formDataStore.getValue(fieldPath)
      
      // 确保 formDataStore 中有正确的值，并保留 display 和 meta 信息
      const fieldValue: FieldValue = {
        raw: rawValue,
        display: currentValue?.display || (rawValue !== null && rawValue !== undefined ? String(rawValue) : ''),
        meta: {
          ...(currentValue?.meta || {}), // 🔥 保留原有的 meta 信息（displayInfo、statistics 等）
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

// 导入相关状态
const importDialogVisible = ref(false)
const importFile = ref<File | null>(null)
const importData = ref<any[]>([])
const importErrors = ref<Array<{ index: number; field: string; error: string }>>([])
const importing = ref(false)
const downloadingTemplate = ref(false)

// 处理导入
function handleImport(): void {
  importDialogVisible.value = true
  importFile.value = null
  importData.value = []
  importErrors.value = []
}

// 下载模板
async function handleDownloadTemplate(): Promise<void> {
  const functionDetail = props.formRenderer?.getFunctionDetail?.()
  if (!functionDetail?.router) {
    ElMessage.error('无法获取函数路由，无法下载模板')
    return
  }
  
  downloadingTemplate.value = true
  try {
    const fullCodePath = functionDetail.router.startsWith('/') ? functionDetail.router : `/${functionDetail.router}`
    await download(`/workspace/api/v1/table/template${fullCodePath}`)
    ElMessage.success('模板下载成功')
  } catch (error: any) {
    ElMessage.error(`下载模板失败: ${error.message || '未知错误'}`)
  } finally {
    downloadingTemplate.value = false
  }
}

// 选择文件
function handleFileSelect(file: any): void {
  const rawFile = file.raw
  if (!rawFile) return
  
  importFile.value = rawFile
  
  // 解析 Excel
  parseExcelFile(rawFile)
}

// 解析 Excel 文件
function parseExcelFile(file: File): void {
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const data = new Uint8Array(e.target?.result as ArrayBuffer)
      const workbook = XLSX.read(data, { type: 'array' })
      
      // 获取第一个工作表
      const firstSheetName = workbook.SheetNames[0]
      const worksheet = workbook.Sheets[firstSheetName]
      
      // 转换为 JSON（第一行作为键名）
      const jsonData = XLSX.utils.sheet_to_json(worksheet, { header: 1, defval: '' })
      
      if (jsonData.length < 3) {
        ElMessage.error('Excel 文件格式错误：至少需要 3 行（字段名称、字段代码、示例数据）')
        return
      }
      
      // 第一行：字段名称（中文）
      // 第二行：字段代码（英文，用于映射）
      // 第三行：示例数据
      // 第四行开始：数据行
      const fieldNames = jsonData[0] as string[]
      const fieldCodes = jsonData[1] as string[]
      const dataRows = jsonData.slice(3) as any[][]
      
      // 构建字段映射（字段代码 -> 列索引）
      const fieldCodeMap = new Map<string, number>()
      fieldCodes.forEach((code, index) => {
        if (code) {
          fieldCodeMap.set(code, index)
        }
      })
      
      // 转换数据
      const convertedData: any[] = []
      const errors: Array<{ index: number; field: string; error: string }> = []
      
      dataRows.forEach((row, rowIndex) => {
        // 跳过空行
        if (row.every(cell => !cell || cell.toString().trim() === '')) {
          return
        }
        
        const rowData: any = {}
        let hasError = false
        
        // 根据字段代码映射数据
        itemFields.value.forEach((field) => {
          const colIndex = fieldCodeMap.get(field.code)
          if (colIndex !== undefined && colIndex < row.length) {
            const cellValue = row[colIndex]
            // 转换数据类型
            const convertedValue = convertFieldValue(field, cellValue)
            rowData[field.code] = convertedValue
            
            // 验证数据
            const validationError = validateFieldValue(field, convertedValue)
            if (validationError) {
              errors.push({
                index: convertedData.length, // 使用 convertedData 的长度作为索引（实际数据行号）
                field: field.name,
                error: validationError
              })
              hasError = true
            }
          } else if (isFieldRequired(field)) {
            // 必填字段缺失
            errors.push({
              index: convertedData.length, // 使用 convertedData 的长度作为索引（实际数据行号）
              field: field.name,
              error: '必填字段不能为空'
            })
            hasError = true
          }
        })
        
        if (!hasError || Object.keys(rowData).length > 0) {
          convertedData.push(rowData)
        }
      })
      
      importData.value = convertedData
      importErrors.value = errors
      
      if (errors.length > 0) {
        ElMessage.warning(`解析完成，发现 ${errors.length} 个错误，请修正后重新导入`)
      } else {
        ElMessage.success(`解析完成，共 ${convertedData.length} 条有效数据`)
      }
    } catch (error: any) {
      ElMessage.error(`解析 Excel 文件失败: ${error.message || '未知错误'}`)
      Logger.error('TableWidget', '解析 Excel 失败', error)
    }
  }
  reader.readAsArrayBuffer(file)
}

// 转换字段值（根据 data.type 转换，只使用 widget.go 中定义的数据类型）
// 注意：这个函数应该使用 excelImport.ts 中的 convertFieldValue，保持一致性
// 这里保留是为了兼容，但建议统一使用 excelImport.ts
function convertFieldValue(field: FieldConfig, value: any): any {
  if (value === null || value === undefined || value === '') {
    return null
  }
  
  const dataType = (field.data as any)?.type || 'string'
  const widgetType = field.widget?.type || 'input'
  
  // 根据 data.type 转换数据类型（只使用 widget.go 中定义的类型）
  switch (dataType) {
    case 'int':
      const num = Number(value)
      return isNaN(num) ? null : num
      
    case 'float':
      const floatNum = Number(value)
      return isNaN(floatNum) ? null : floatNum
      
    case 'bool':
      if (typeof value === 'boolean') return value
      if (typeof value === 'string') {
        const lower = value.toLowerCase()
        return lower === 'true' || lower === '1' || lower === '是' || lower === 'yes'
      }
      return Boolean(value)
      
    case '[]string':
      // 字符串数组：支持逗号分隔的字符串
      if (Array.isArray(value)) return value
      if (typeof value === 'string') {
        return value.split(',').map(v => v.trim()).filter(Boolean)
      }
      return [value]
      
    case '[]int':
      // 整数数组：支持逗号分隔的字符串，转换为数字数组
      if (Array.isArray(value)) {
        return value.map(v => {
          const num = Number(v)
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      if (typeof value === 'string') {
        return value.split(',').map(v => {
          const num = Number(v.trim())
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      return [Number(value)]
      
    case '[]float':
      // 浮点数数组：支持逗号分隔的字符串，转换为浮点数数组
      if (Array.isArray(value)) {
        return value.map(v => {
          const num = Number(v)
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      if (typeof value === 'string') {
        return value.split(',').map(v => {
          const num = Number(v.trim())
          return isNaN(num) ? null : num
        }).filter(v => v !== null)
      }
      return [Number(value)]
      
    case 'string':
    default:
      // 字符串类型：保持原样，但如果是 multiselect widget，需要特殊处理
      if (widgetType === 'multiselect') {
        // multiselect 但 data.type 是 string，需要转换为逗号分隔的字符串
        if (Array.isArray(value)) {
          return value.join(',')
        }
        if (typeof value === 'string') {
          // 已经是字符串，直接返回
          return value
        }
      }
      // timestamp widget 也保持字符串格式（日期时间字符串）
      return value.toString()
  }
}

// 验证字段值
function validateFieldValue(field: FieldConfig, value: any): string | null {
  // 必填验证
  if (isFieldRequired(field)) {
    if (value === null || value === undefined || value === '' || 
        (Array.isArray(value) && value.length === 0)) {
      return '必填字段不能为空'
    }
  }
  
  // 类型验证
  const dataType = (field.data as any)?.type || 'string'
  const widgetType = field.widget?.type || 'input'
  
  if (value !== null && value !== undefined && value !== '') {
    switch (widgetType) {
      case 'number':
      case 'float':
        if (isNaN(Number(value))) {
          return '必须是数字'
        }
        break
        
      case 'switch':
        if (typeof value !== 'boolean') {
          return '必须是布尔值'
        }
        break
    }
  }
  
  // 长度验证
  const validation = field.validation
  if (validation && typeof value === 'string') {
    if (validation.includes('min=')) {
      const minMatch = validation.match(/min=(\d+)/)
      if (minMatch && value.length < Number(minMatch[1])) {
        return `长度不能少于 ${minMatch[1]} 个字符`
      }
    }
    if (validation.includes('max=')) {
      const maxMatch = validation.match(/max=(\d+)/)
      if (maxMatch && value.length > Number(maxMatch[1])) {
        return `长度不能超过 ${maxMatch[1]} 个字符`
      }
    }
  }
  
  return null
}

// 检查字段是否必填
function isFieldRequired(field: FieldConfig): boolean {
  return field.validation?.includes('required') || false
}

// 重新选择文件
function handleReSelectFile(): void {
  importFile.value = null
  importData.value = []
  importErrors.value = []
}

// 提交导入
async function handleSubmitImport(): Promise<void> {
  if (importData.value.length === 0) {
    ElMessage.warning('没有可导入的数据')
    return
  }
  
  const functionDetail = props.formRenderer?.getFunctionDetail?.()
  if (!functionDetail?.router) {
    ElMessage.error('无法获取函数路由，无法导入数据')
    return
  }
  
  importing.value = true
  try {
    const fullCodePath = functionDetail.router.startsWith('/') ? functionDetail.router : `/${functionDetail.router}`
    const response = await post(`/workspace/api/v1/table/batch-create${fullCodePath}`, {
      data: importData.value
    })
    
    if (response.code === 0) {
      const result = response.data || {}
      const successCount = result.success_count || 0
      const failCount = result.fail_count || 0
      
      if (failCount > 0) {
        ElMessage.warning(`导入完成：成功 ${successCount} 条，失败 ${failCount} 条`)
        // 显示失败详情
        if (result.errors && result.errors.length > 0) {
          const errorMsg = result.errors.map((e: any) => `第 ${e.index + 1} 行: ${e.error}`).join('\n')
          ElMessage.error(`失败详情:\n${errorMsg}`)
        }
      } else {
        ElMessage.success(`成功导入 ${successCount} 条数据`)
      }
      
      // 关闭对话框
      importDialogVisible.value = false
      
      // 刷新表格数据（触发父组件刷新）
      emit('update:modelValue', {
        ...props.value,
        raw: null, // 触发重新加载
        display: props.value.display,
        meta: props.value.meta
      })
    } else {
      ElMessage.error(response.msg || '导入失败')
    }
  } catch (error: any) {
    ElMessage.error(`导入失败: ${error.message || '未知错误'}`)
    Logger.error('TableWidget', '导入失败', error)
  } finally {
    importing.value = false
  }
}

// 处理导出（待实现）
function handleExport(): void {
  Logger.warn('TableWidget', '导出功能待实现')
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

.import-dialog-content {
  padding: 16px 0;
}

.import-step {
  margin-bottom: 24px;
}

.import-step h3 {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 600;
}

.import-info {
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.import-info p {
  margin: 4px 0;
  font-size: 14px;
}

.error-cell {
  color: #f56c6c;
  font-weight: 500;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
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


/* 🔥 强制所有单元格内容左对齐 */
:deep(.table-widget-table .el-table__body td),
:deep(.table-widget-table .el-table__body td .cell) {
  text-align: left !important;
}

:deep(.table-widget-table .el-table__body td .cell) {
  display: flex !important;
  justify-content: flex-start !important;
  align-items: center !important;
}

</style>

