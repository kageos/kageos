<!--
  TableWidget - 表格容器组件
  🔥 统一架构组件
  
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
      <div class="table-panel">
        <div class="table-widget-content">
          <el-table
            :data="editMode.tableData.value"
            :stripe="false"
            :row-class-name="getEditRowClassName"
            class="table-widget-table"
          >
            <el-table-column
              v-for="itemField in itemFields"
              :key="itemField.code"
              :prop="itemField.code"
              :label="itemField.name"
              class-name="table-widget-data-column"
              :min-width="getColumnWidth(itemField)"
              :align="getColumnAlign(itemField)"
              header-align="left"
              show-overflow-tooltip
            >
              <template #default="{ row, $index }">
                <template v-if="isNestedContainerField(itemField)">
                  <component
                    v-if="isEditRowFieldVisible($index, itemField)"
                    :is="getWidgetComponent(itemField.widget?.type, 'table-cell')"
                    :field="itemField"
                    :value="getRowFieldValue($index, itemField.code)"
                    :model-value="getRowFieldValue($index, itemField.code)"
                    @update:model-value="handleRowFieldModelUpdate($index, itemField.code, $event)"
                    :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                    :form-manager="formManager"
                    :form-renderer="formRenderer"
                    mode="table-cell"
                    :parent-mode="mode"
                    :depth="(depth || 0) + 1"
                  />
                  <span v-else class="table-cell-hidden-placeholder">-</span>
                </template>
                <template v-else>
                  <template v-if="editMode.editingIndex.value === $index">
                    <component
                      v-if="isEditRowFieldVisible($index, itemField)"
                      :is="getWidgetComponent(itemField.widget?.type || 'input', 'edit')"
                      :field="itemField"
                      :value="getRowFieldValue($index, itemField.code)"
                      :model-value="getRowFieldValue($index, itemField.code)"
                      @update:model-value="handleRowFieldModelUpdate($index, itemField.code, $event)"
                      :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                      :form-manager="formManager"
                      :form-renderer="formRenderer"
                      mode="edit"
                      :depth="(depth || 0) + 1"
                    />
                    <span v-else class="table-cell-hidden-placeholder">-</span>
                  </template>
                  <template v-else>
                    <component
                      v-if="isEditRowFieldVisible($index, itemField)"
                      :is="getWidgetComponent(itemField.widget?.type || 'input', 'table-cell')"
                      :field="itemField"
                      :value="getRowFieldValue($index, itemField.code)"
                      :model-value="getRowFieldValue($index, itemField.code)"
                      :field-path="`${fieldPath}[${$index}].${itemField.code}`"
                      mode="table-cell"
                      :depth="(depth || 0) + 1"
                    />
                    <span v-else class="table-cell-hidden-placeholder">-</span>
                  </template>
                </template>
              </template>
            </el-table-column>
            
            <!-- 操作列 -->
            <el-table-column
              label="操作"
              width="124"
              fixed="right"
              align="center"
              header-align="center"
              class-name="table-widget-actions-column"
            >
              <template #default="{ $index }">
                <div class="table-row-actions">
                  <template v-if="editMode.editingIndex.value === $index">
                    <el-tooltip content="保存" placement="top">
                      <el-button
                        size="small"
                        type="primary"
                        class="table-icon-button"
                        aria-label="保存"
                        @click="handleSave($index)"
                      >
                        <el-icon><Check /></el-icon>
                      </el-button>
                    </el-tooltip>
                    <el-tooltip content="取消" placement="top">
                      <el-button
                        size="small"
                        class="table-icon-button"
                        aria-label="取消"
                        @click="editMode.cancelEditing()"
                      >
                        <el-icon><Close /></el-icon>
                      </el-button>
                    </el-tooltip>
                  </template>
                  <template v-else>
                    <el-tooltip content="编辑" placement="top">
                      <el-button
                        size="small"
                        class="table-icon-button"
                        aria-label="编辑"
                        @click="editMode.startEditing($index)"
                      >
                        <el-icon><EditPen /></el-icon>
                      </el-button>
                    </el-tooltip>
                    <el-tooltip content="删除" placement="top">
                      <el-button
                        size="small"
                        type="danger"
                        plain
                        class="table-icon-button"
                        aria-label="删除"
                        @click="handleDelete($index)"
                      >
                        <el-icon><DeleteIcon /></el-icon>
                      </el-button>
                    </el-tooltip>
                  </template>
                </div>
              </template>
            </el-table-column>
          </el-table>
      
      <!-- 新增按钮 -->
      <div class="table-actions">
        <el-button type="primary" class="table-add-button" @click="editMode.startAdding()">
          <el-icon><Plus /></el-icon>
          <span>新增</span>
        </el-button>
      </div>

      <FieldStatistics
        v-if="editingRowStatistics && Object.keys(editingRowStatistics).length > 0"
        :field="field"
        :value="getAllRowsData()"
        :statistics="editingRowStatistics"
      />

        </div>
      </div>
    </template>
    
    <!-- 响应模式（只读） -->
    <template v-else-if="mode === 'response'">
      <div class="table-panel is-response">
        <div class="table-widget-content">
          <el-table :data="responseTableData" :stripe="false" class="table-widget-table">
            <el-table-column
              v-for="itemField in itemFields"
              :key="itemField.code"
              :prop="itemField.code"
              :label="itemField.name"
              class-name="table-widget-data-column"
              :min-width="getColumnWidth(itemField)"
              :align="getColumnAlign(itemField)"
              header-align="left"
              show-overflow-tooltip
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
                    v-if="isResponseRowFieldVisible($index, itemField)"
                    :is="getWidgetComponent(itemField.widget?.type || 'input', 'table-cell')"
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
                  <span v-else class="table-cell-hidden-placeholder">-</span>
                </template>
                <!-- 🔥 其他类型字段：响应表格中直接走组件渲染，确保 progress/files 等输出态完整显示 -->
                <template v-else>
                  <template v-if="!isResponseRowFieldVisible($index, itemField)">
                    <span class="table-cell-hidden-placeholder">-</span>
                  </template>
                  <component
                    v-else
                    :is="getWidgetComponent(itemField.widget?.type || 'input', 'table-cell')"
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
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      
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
              v-for="itemField in getVisibleResponseDetailFields(responseMode.currentDetailIndex.value)"
              :key="itemField.code"
              class="detail-field"
            >
              <div class="field-label">{{ itemField.name }}</div>
              <div class="field-value">
                <component
                  :is="getWidgetComponent(itemField.widget?.type || 'input', 'detail')"
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
        size="small"
        @click="tableCellMode.openDrawer()"
        class="table-cell-button"
      >
        <span>{{ displayValue }}</span>
        <el-icon class="table-cell-button-icon">
          <View />
        </el-icon>
      </el-button>
      
      <!-- 详情抽屉（根据上下文支持编辑或只读） -->
      <el-drawer
        v-model="tableCellMode.showDrawer.value"
        :title="field.name"
        :size="DRAWER_CONFIG.size"
        destroy-on-close
        :append-to-body="shouldAppendOverlayToBody"
        @close="tableCellMode.handleDrawerClose()"
      >
        <template #default>
          <div class="table-detail-content">
            <div class="table-detail-panel">
              <!-- 
                🔥 抽屉中根据上下文使用 edit 或 response 模式的渲染逻辑
                
                drawerMode 的值由 isInEditContext 决定：
                - 编辑上下文：drawerMode = 'edit' → 可编辑，支持数据修改
                - 响应上下文：drawerMode = 'response' → 只读，仅展示数据
              -->
              <component
                :is="getWidgetComponent('table', tableCellMode.drawerMode.value)"
                :field="field"
                :value="value"
                :model-value="value"
                @update:model-value="emit('update:modelValue', $event)"
                :field-path="fieldPath"
                :form-manager="formManager"
                :form-renderer="formRenderer"
                :mode="tableCellMode.drawerMode.value"
                :depth="(depth || 0) + 1"
              />
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
            <el-button type="primary" @click="handleTableCellConfirm">确认</el-button>
          </div>
        </template>
      </el-drawer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import { ElTable, ElTableColumn, ElButton, ElDrawer, ElIcon, ElTooltip } from 'element-plus'
import { Check, Close, Delete as DeleteIcon, EditPen, Plus, View } from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useTableWidget } from '@/architecture/presentation/widgets/composables/useTableWidget'
import { useTableEditMode } from '@/architecture/presentation/widgets/composables/useTableEditMode'
import type { FieldValue } from '@/architecture/domain/types'
import FieldStatistics from './FieldStatistics.vue'
import { useTableWidgetDisplay } from '@/architecture/presentation/widgets/composables/useTableWidgetDisplay'
import { useTableWidgetEditActions } from '@/architecture/presentation/widgets/composables/useTableWidgetEditActions'
import { prdPreviewContextKey } from '@/architecture/presentation/components/prdPreviewContext'

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
const prdPreviewContext = inject(prdPreviewContextKey, null)
const shouldAppendOverlayToBody = computed(() => !prdPreviewContext?.interactive)

// 使用组合式函数
const { tableData, itemFields, getRowFieldValue, updateRowFieldValue, getAllRowsData } = useTableWidget(props)
const editMode = useTableEditMode(props)

const {
  responseMode,
  tableCellMode,
  responseTableData,
  getResponseRowFieldValue,
  getEditRowFieldPresenceState,
  isEditRowFieldVisible,
  isResponseRowFieldVisible,
  getVisibleResponseDetailFields,
  displayValue,
  handleTableCellConfirm,
  getColumnWidth,
  getColumnAlign,
  getWidgetComponent,
  isNestedContainerField
} = useTableWidgetDisplay(props, {
  tableData,
  itemFields
})

const {
  editingRowStatistics,
  handleRowFieldModelUpdate,
  getEditRowClassName,
  handleSave,
  handleDelete,
  validate
} = useTableWidgetEditActions(props, {
  tableData,
  itemFields,
  editMode,
  getRowFieldValue,
  updateRowFieldValue,
  getEditRowFieldPresenceState
})

// 🔥 暴露验证方法给父组件
defineExpose({
  validate
})
</script>

<style scoped>
.table-widget {
  width: 100%;
}

.table-panel {
  width: 100%;
  margin-bottom: 24px;
  overflow: hidden;
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.table-panel:last-child {
  margin-bottom: 0;
}

.table-widget-content {
  width: 100%;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0;
}

.table-panel.is-response {
  background-color: var(--el-bg-color);
}

.table-actions {
  margin-top: 0;
  padding: 10px 14px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: linear-gradient(180deg, var(--el-fill-color-lighter) 0%, var(--el-bg-color) 100%);
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.table-add-button {
  min-width: 82px;
  height: 32px;
  border-radius: 6px;
  gap: 4px;
}

.table-row-actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-wrap: nowrap;
}

.table-icon-button {
  width: 30px;
  height: 30px;
  padding: 0;
  border-radius: 6px;
}

.table-icon-button + .table-icon-button {
  margin-left: 0;
}

.table-cell-value {
  color: var(--el-text-color-regular);
}

.table-cell-button {
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

.table-cell-button:hover,
.table-cell-button:focus-visible {
  border-color: transparent;
  background: transparent;
  color: var(--el-text-color-primary);
}

.table-cell-button-icon {
  margin-left: 4px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

.table-cell-hidden-placeholder {
  color: var(--el-text-color-placeholder);
  display: inline-flex;
  align-items: center;
  min-height: 32px;
}

/* 详情抽屉内容 */
.table-detail-content {
  padding: 24px;
  box-sizing: border-box;
  /* 确保下拉菜单可以正常显示 */
  overflow: visible;
  position: relative;
}

.table-detail-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

:deep(.table-detail-content .form-card),
:deep(.table-detail-content .table-panel) {
  margin-bottom: 0;
}

:deep(.table-detail-content .el-card__body) {
  padding: 20px 22px;
}

:deep(.table-detail-content .form-widget-form .el-form-item:last-child) {
  margin-bottom: 0;
}

:deep(.table-detail-content .table-actions) {
  padding-left: 16px;
  padding-right: 16px;
}

:deep(.table-detail-panel > .table-widget > .table-panel),
:deep(.table-detail-panel > .form-widget > .form-card) {
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

/* 🔥 表格样式：外层 panel 承载边框，表格内部保持轻量分隔 */
:deep(.table-widget-table) {
  --el-table-border-color: var(--el-border-color-lighter);
  background-color: var(--el-bg-color) !important;
}

/* 🔥 移除内部竖向边框，避免和外层边框叠在一起 */
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

:deep(.table-widget-table .el-table__inner-wrapper::before) {
  display: none;
}

:deep(.table-widget-table thead),
:deep(.table-widget-table th.el-table__cell) {
  background: linear-gradient(180deg, var(--el-fill-color-lighter) 0%, var(--el-bg-color) 100%) !important;
}

:deep(.table-widget-table th.el-table__cell) {
  color: var(--el-text-color-primary);
  font-weight: 600;
  border-bottom: 1px solid var(--el-border-color-light) !important;
}

:deep(.table-widget-table th.el-table__cell .cell) {
  font-size: 13px;
  line-height: 20px;
  padding-top: 11px;
  padding-bottom: 11px;
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
  transition: background-color 0.16s ease;
}

/* 🔥 移除斑马纹：确保所有行背景色一致 */
:deep(.table-widget-table .el-table__body tr.el-table__row--striped) {
  background-color: var(--el-bg-color) !important;
}

:deep(.table-widget-table .el-table__body tr.el-table__row--striped td) {
  background-color: var(--el-bg-color) !important;
}

:deep(.table-widget-table .el-table__body tr:hover > td) {
  background-color: var(--el-fill-color-lighter) !important;
}


/* 🔥 强制所有单元格内容左对齐 */
:deep(.table-widget-table .el-table__body td),
:deep(.table-widget-table .el-table__body td .cell) {
  text-align: left !important;
}

:deep(.table-widget-table .el-table__body td) {
  vertical-align: top;
}

:deep(.table-widget-table .el-table__body td .cell) {
  display: flex !important;
  justify-content: flex-start !important;
  align-items: center !important;
  width: 100%;
  min-height: 48px;
  padding-top: 10px;
  padding-bottom: 10px;
}

:deep(.table-widget-table .el-table__body td.table-widget-actions-column .cell) {
  justify-content: center !important;
}

:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .cell) {
  min-width: 0;
  line-height: 1.45;
  white-space: normal;
  overflow: hidden;
}

:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .cell > *) {
  min-width: 0;
  max-width: 100%;
}

:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .table-cell-value),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .files-display-text) {
  display: block;
  min-width: 0;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .input-widget .table-cell-value),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .textarea-widget .table-cell-value),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .rich-text-widget .table-cell-value),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .text-widget .table-cell-value),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .table-cell-text),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .formatted-content),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .text-content),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .code-content),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .html-table-cell),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .markdown-table-cell),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .csv-preview),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .csv-preview-text),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .html-content-preview) {
  display: -webkit-box;
  min-width: 0;
  max-width: 100%;
  line-height: 1.45;
  white-space: normal;
  overflow-wrap: anywhere;
  overflow: hidden;
  text-overflow: ellipsis;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .table-cell-multiselect),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .files-table-cell),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .files-select-display) {
  min-width: 0;
  max-width: 100%;
  flex-wrap: nowrap;
  overflow: hidden;
}

:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .files-table-cell),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .files-table-preview-list),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .files-select-display) {
  width: 100%;
  justify-content: flex-start;
  text-align: left;
}

:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .formatted-content),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .text-content),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .code-content),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .html-table-cell),
:deep(.table-widget-table .el-table__body tr:not(.is-editing-row) td.table-widget-data-column .markdown-table-cell) {
  padding: 0;
  border: none;
  background: transparent;
  box-shadow: none;
}

:deep(.table-widget-table .is-editing-row > td) {
  background-color: var(--el-fill-color-extra-light) !important;
}

:deep(.table-widget-table .is-editing-row > td:first-child) {
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}

:deep(.table-widget-table .is-editing-row:hover > td) {
  background-color: var(--el-fill-color-light) !important;
}

:deep(.table-widget-table .is-editing-row > td .cell) {
  align-items: flex-start !important;
}

:deep(.table-widget-table .is-editing-row .el-input),
:deep(.table-widget-table .is-editing-row .el-input-number),
:deep(.table-widget-table .is-editing-row .el-select),
:deep(.table-widget-table .is-editing-row .el-date-editor),
:deep(.table-widget-table .is-editing-row .el-cascader) {
  width: 100%;
  max-width: 100%;
}

.table-widget-content > :deep(.field-statistics) {
  width: auto;
  margin: 14px;
}

:deep(.table-widget-table .el-table__empty-block) {
  min-height: 88px;
}

:deep(.table-widget-table .el-table__empty-text) {
  color: var(--el-text-color-secondary);
}

</style>
