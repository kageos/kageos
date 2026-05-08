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
          :min-width="getColumnWidth(itemField)"
          :align="getColumnAlign(itemField)"
          header-align="left"
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
          <el-table-column label="操作" width="150" fixed="right" header-align="left">
            <template #default="{ $index }">
              <div class="table-row-actions">
                <template v-if="editMode.editingIndex.value === $index">
                  <el-button size="small" type="primary" @click="handleSave($index)">保存</el-button>
                  <el-button size="small" @click="editMode.cancelEditing()">取消</el-button>
                </template>
                <template v-else>
                  <el-button size="small" @click="editMode.startEditing($index)">编辑</el-button>
                  <el-button size="small" type="danger" @click="handleDelete($index)">删除</el-button>
                </template>
              </div>
            </template>
          </el-table-column>
      </el-table>
      
      <!-- 新增按钮 -->
      <div class="table-actions">
        <el-button type="primary" class="table-add-button" @click="editMode.startAdding()">新增</el-button>
      </div>

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
                <!-- 🔥 其他类型字段：使用共享的渲染函数（与 TableRenderer 一致） -->
                <template v-else>
                  <template v-if="!isResponseRowFieldVisible($index, itemField)">
                    <span class="table-cell-hidden-placeholder">-</span>
                  </template>
                  <template v-else-if="getCellContent(itemField, row[itemField.code]).isString">
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
import { ElTable, ElTableColumn, ElButton, ElDrawer, ElCard, ElIcon } from 'element-plus'
import { View } from '@element-plus/icons-vue'
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
  getCellContent,
  CellRenderer,
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

.table-widget-content {
  width: 100%;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 响应模式表格卡片样式 */
.response-table-card {
  background-color: var(--el-bg-color-page);
}

:deep(.table-card .el-card__body) {
  padding: 18px 20px 16px;
}

.table-actions {
  margin-top: 0;
  padding: 16px 0 0;
  border-top: 1px solid var(--el-border-color-extra-light);
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.table-add-button {
  min-width: 88px;
}

.table-row-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
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
:deep(.table-detail-content .table-card) {
  margin-bottom: 0;
}

:deep(.table-detail-content .el-card__body) {
  padding: 20px 22px;
}

:deep(.table-detail-content .form-widget-form .el-form-item:last-child) {
  margin-bottom: 0;
}

:deep(.table-detail-content .table-actions) {
  padding-left: 0;
  padding-right: 0;
}

:deep(.table-detail-panel > .table-widget > .table-card),
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

:deep(.table-widget-table .el-table__body td) {
  vertical-align: top;
}

:deep(.table-widget-table .el-table__body td .cell) {
  display: flex !important;
  justify-content: flex-start !important;
  align-items: center !important;
  width: 100%;
  min-height: 52px;
  padding-top: 12px;
  padding-bottom: 12px;
}

:deep(.table-widget-table .is-editing-row > td) {
  background-color: var(--el-fill-color-extra-light) !important;
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

</style>
