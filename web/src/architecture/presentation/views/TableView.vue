<!--
  TableView - 表格视图
  统一架构的展示层组件
  
  ============================================
  📋 需求说明
  ============================================
  
  1. **表格数据展示**：
     - 从后端获取表格数据并渲染
     - 支持搜索、排序、分页
     - 支持批量操作（批量删除）
  
  2. **权限控制**：
     - 根据节点权限控制按钮显示/隐藏
     - 新增按钮：需要 `function:write` 权限
     - 编辑按钮：需要 `function:update` 权限
     - 删除按钮：需要 `function:delete` 权限
     - 提交时再次检查权限，防止绕过 UI 检查
  
  3. **URL 参数同步**：
     - 搜索条件同步到 URL（`field=value`）
     - 排序条件同步到 URL（`sorts=[{"field":"created_at","order":"desc"}]`）
     - 分页信息同步到 URL（`page=1&page_size=10`）
     - 新增弹窗状态同步到 URL（`_tab=OnTableAddRow`）
  
  ============================================
  🎯 设计思路
  ============================================
  
  1. **分层架构**：
     - Presentation Layer：纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
  
  2. **URL 同步**：
     - 使用 `useTableInitialization` 从 URL 初始化表格状态
     - 使用 `useTableParamURLSync` 同步表格状态到 URL
     - 使用事件驱动，避免直接操作路由
  
  ============================================
  📝 关键功能
  ============================================
  
  1. **表格操作**：
     - 新增：打开 FormDialog，提交时检查权限
     - 编辑：打开详情抽屉，在 useWorkspaceDetail 中检查权限
     - 删除：批量删除，提交前检查权限
  
  2. **数据加载**：
     - 从 TableApplicationService 加载数据
     - 支持搜索、排序、分页参数
     - 支持从 URL 恢复表格状态
  
  ============================================
  ⚠️ 注意事项
  ============================================
  
  1. **URL 同步**：
     - 新增弹窗状态使用 `_tab=OnTableAddRow` 标识
     - 详情抽屉状态使用 `_tab=detail` 标识（编辑模式不设置 `_tab`）
     - 表单字段参数只在新增模式下同步到 URL
  
  3. **数据流**：
     - TableView → TableApplicationService → TableDomainService → TableStateManager
     - 状态变化通过事件总线通知其他组件
-->

<template>
  <div class="table-view" data-testid="table-view">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <!-- 新增按钮：不支持时展示禁用态，避免误以为缺按钮 -->
        <el-button 
          :type="hasAddCallback ? 'primary' : 'default'"
          :plain="!hasAddCallback"
          :disabled="!hasAddCallback"
          @click="handleAdd"
          :icon="hasAddCallback ? Plus : InfoFilled"
          class="action-btn"
          data-testid="table-add"
        >
          {{ hasAddCallback ? '新增' : '新增（当前表格不支持）' }}
        </el-button>
        <el-dropdown
          trigger="click"
          :disabled="spreadsheetBusy"
          @command="handleSpreadsheetCommand"
        >
          <el-button class="action-btn" :loading="spreadsheetBusy" data-testid="table-spreadsheet-actions">
            <el-icon><FolderOpened /></el-icon>
            导入 / 导出
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="template" :disabled="!hasAddCallback">
                <el-icon><DocumentAdd /></el-icon>
                下载导入模板
              </el-dropdown-item>
              <el-dropdown-item command="import" :disabled="!hasAddCallback">
                <el-icon><Upload /></el-icon>
                导入 Excel / CSV
              </el-dropdown-item>
              <el-dropdown-item command="export" divided>
                <el-icon><Download /></el-icon>
                导出当前筛选结果
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <TableSpreadsheetGuidePopover
          v-if="hasAddCallback"
          :fields="tableCreateFields"
        />
        <!-- 批量删除按钮：不支持时保留禁用提示 -->
        <el-button 
          v-if="!isBatchDeleteMode" 
          :type="hasDeleteCallback ? 'danger' : 'default'"
          :plain="!hasDeleteCallback"
          :disabled="!hasDeleteCallback"
          @click="hasDeleteCallback ? enterBatchDeleteMode() : undefined"
          :icon="hasDeleteCallback ? Delete : InfoFilled"
          class="action-btn"
        >
          {{ hasDeleteCallback ? '批量删除' : '批量删除（当前表格不支持）' }}
        </el-button>
        <template v-if="hasDeleteCallback && isBatchDeleteMode">
          <el-button 
            type="danger" 
            @click="handleBatchDelete"
            :icon="Delete"
            :disabled="selectedRows.length === 0"
            class="action-btn action-btn-danger"
          >
            删除选中 ({{ selectedRows.length }})
          </el-button>
          <el-button 
            @click="exitBatchDeleteMode"
            class="toolbar-secondary-btn"
          >
            取消
          </el-button>
        </template>
      </div>

      <div class="toolbar-right">
        <template v-if="searchableFields.length > 0">
          <el-button
            :type="searchBarExpanded ? 'primary' : 'default'"
            class="toolbar-search-btn"
            @click="searchBarExpanded ? handleSearch() : (searchBarExpanded = true)"
          >
            <el-icon><Search /></el-icon>
            <span v-if="searchBarExpanded">搜索</span>
            <span v-else>
              {{ activeSearchCount > 0 ? `筛选 (${activeSearchCount})` : '筛选' }}
            </span>
          </el-button>
          <el-button
            v-if="searchBarExpanded"
            class="toolbar-secondary-btn"
            @click="handleReset"
          >
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
          <el-button
            v-if="searchBarExpanded"
            text
            class="toolbar-collapse-btn"
            @click="searchBarExpanded = false"
          >
            <el-icon><ArrowUp /></el-icon>
            收起
          </el-button>
        </template>
      </div>
    </div>

    <!-- 搜索栏：字段区单独占位，动作并到工具栏 -->
    <div v-if="searchableFields.length > 0 && searchBarExpanded" class="search-bar-wrapper" data-testid="table-search">
      <div class="search-bar">
        <div class="search-bar-inner">
          <el-form
            :inline="false"
            label-position="top"
            :model="searchForm"
            class="search-form"
          >
            <template v-for="field in searchableFields" :key="field.code">
              <el-form-item :label="field.name" :class="getSearchFieldLayoutClass(field)">
                <SearchInput
                  :field="field"
                  search-type=""
                  :model-value="getSearchValue(field)"
                  :function-method="props.functionDetail.method || 'GET'"
                  :function-router="props.functionDetail.router"
                  @update:model-value="(value: unknown) => {
                    updateSearchValue(field, value, true)
                  }"
                />
              </el-form-item>
            </template>
          </el-form>
        </div>
      </div>
    </div>

    <!-- 🔥 排序信息条：显示当前排序状态 -->
    <div v-if="displaySorts.length > 0" class="sort-info-bar">
      <div class="sort-info-content">
        <span class="sort-label">排序：</span>
        <div class="sort-items">
          <template v-for="(sort, index) in displaySorts" :key="sort.field">
            <el-tag
              :type="index === 0 ? 'primary' : 'info'"
              size="small"
              closable
              @close="handleRemoveSort(sort.field)"
              class="sort-tag"
            >
              <span class="sort-field-name">{{ getFieldName(sort.field) }}</span>
              <el-icon class="sort-icon">
                <ArrowUp v-if="sort.order === 'asc'" />
                <ArrowDown v-else />
              </el-icon>
            </el-tag>
            <span v-if="index < displaySorts.length - 1" class="sort-separator">></span>
          </template>
        </div>
        <el-button
          v-if="sorts.length > 0"
          size="small"
          @click="handleClearAllSorts"
          class="table-clear-sorts-btn"
        >
          <el-icon><Refresh /></el-icon>
          清除排序
        </el-button>
      </div>
    </div>

    <!-- 加载中：表格骨架屏 -->
    <div v-if="loading" class="table-skeleton-wrap">
      <el-skeleton :rows="20" animated class="table-skeleton" />
    </div>
    <!-- 表格 -->
    <el-table
      v-else
      ref="tableRef"
      :data="tableData"
      :stripe="false"
      style="width: 100%"
      class="table-with-fixed-column table-row-clickable"
      data-testid="table-grid"
      @sort-change="handleSortChange"
      @selection-change="handleSelectionChange"
      @row-click="handleRowClick"
    >
      <!-- 复选框列（用于批量操作，仅在批量删除模式下显示） -->
      <el-table-column
        v-if="hasDeleteCallback && isBatchDeleteMode"
        type="selection"
        width="55"
        fixed="left"
        :selectable="checkSelectable"
      />

      <!-- 🔥 控制中心列（ID列） -->
      <el-table-column
        v-if="idField"
        :prop="idField.code"
        label=""
        fixed="left"
        width="132"
        class-name="control-column"
        :label-class-name="getSortHeaderClass(idField.code)"
        :sortable="getSortableConfig(idField)"
        :sort-order="sortOrderMap[idField.code] || null"
      >
        <template #default="{ row }">
          <button
            type="button"
            class="detail-icon-button"
            :title="getDetailIdTitle(row[idField.code])"
            @click.stop="handleDetail(row)"
          >
            <el-icon><View /></el-icon>
            <span class="detail-id-text">{{ getDetailIdText(row[idField.code]) }}</span>
          </button>
        </template>
      </el-table-column>

      <!-- 数据列（排除ID列） -->
      <el-table-column
        v-for="field in dataFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name"
        class-name="table-data-column"
        :sortable="getSortableConfig(field)"
        :sort-order="sortOrderMap[field.code] || null"
        :label-class-name="getSortHeaderClass(field.code)"
        :min-width="getColumnWidth(field)"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <WidgetComponent
            :field="field"
            :value="getRowFieldValue(row, field)"
            mode="table-cell"
            :row-data="row"
          />
        </template>
      </el-table-column>

      <!-- 操作列：统一为「更多」下拉，所有操作（链接 / 更新 / 删除）均放入下拉 -->
      <el-table-column 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
        class-name="action-column"
      >
        <template #default="{ row }">
          <el-dropdown
            trigger="click"
            placement="bottom-end"
            popper-class="table-action-dropdown"
            @command="(cmd: string) => handleActionCommand(cmd, row)"
          >
            <el-button size="small" class="action-more-btn">
              <el-icon><More /></el-icon>
              更多
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <!-- 链接项：1 个或多个都放下拉 -->
                <el-dropdown-item
                  v-for="linkField in linkFields"
                  :key="linkField.code"
                  :command="'link:' + linkField.code"
                >
                  <div class="dropdown-link-content">
                    <el-icon v-if="linkField.widget?.config?.icon" class="link-icon">
                      <component :is="linkField.widget.config.icon" />
                    </el-icon>
                    <el-icon v-else class="link-icon internal-icon"><Right /></el-icon>
                    <span>{{ getLinkText(linkField, row[linkField.code]) }}</span>
                  </div>
                </el-dropdown-item>
                <el-dropdown-item
                  :command="hasUpdateCallback ? 'update' : undefined"
                  :disabled="!hasUpdateCallback"
                  :divided="linkFields.length > 0"
                >
                  <span class="dropdown-action-item">
                    <el-icon><component :is="hasUpdateCallback ? Edit : InfoFilled" /></el-icon>
                    {{ hasUpdateCallback ? '更新' : '更新（当前表格不支持）' }}
                  </span>
                </el-dropdown-item>
                <el-dropdown-item
                  :command="hasDeleteCallback ? 'delete' : undefined"
                  :disabled="!hasDeleteCallback"
                  :divided="true"
                >
                  <span class="dropdown-action-item" :class="{ 'delete-action-text': hasDeleteCallback }">
                    <el-icon><component :is="hasDeleteCallback ? Delete : InfoFilled" /></el-icon>
                    {{ hasDeleteCallback ? '删除' : '删除（当前表格不支持）' }}
                  </span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
      
      <!-- 空数据占位符 -->
      <template #empty>
        <el-empty 
          description="暂无数据"
          :image-size="120"
        />
      </template>
    </el-table>

    <!-- 分页 -->
    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <FormDialog
      v-if="hasAddCallback"
      v-model="createDialogVisible"
      title="新增"
      :fields="getTableCreateFields(props.functionDetail)"
      mode="create"
      :router="props.functionDetail.router ?? ''"
      :full-code-path="props.functionDetail.full_code_path || props.functionDetail.router || ''"
      :method="props.functionDetail.method || 'POST'"
      @submit="handleCreateSubmit"
      @close="handleCreateDialogClose"
    />

    <TableSpreadsheetImportDialog
      v-if="hasAddCallback"
      v-model="spreadsheetImportVisible"
      :fields="tableCreateFields"
      :import-rows="importSpreadsheetRows"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElIcon, ElTable, ElForm, ElFormItem, ElButton, ElSkeleton, ElMessage } from 'element-plus'
import { Search, Refresh, Delete, Plus, ArrowUp, ArrowDown, More, Right, Edit, View, InfoFilled, FolderOpened, DocumentAdd, Upload, Download } from '@element-plus/icons-vue'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../../presentation/widgets/WidgetComponent.vue'
import SearchInput from '@/architecture/presentation/components/SearchInput.vue'
import FormDialog from '@/architecture/presentation/components/FormDialog.vue'
import TableSpreadsheetImportDialog from '@/architecture/presentation/components/TableSpreadsheetImportDialog.vue'
import TableSpreadsheetGuidePopover from '@/architecture/presentation/components/TableSpreadsheetGuidePopover.vue'
import { getSortableConfig } from '@/architecture/domain/utils/fieldSort'
import { useTableInitialization } from '../composables/useTableInitialization'
import { useTableBatchDelete } from '../composables/useTableBatchDelete'
import { useTableAddDialogUrlSync } from '../composables/useTableAddDialogUrlSync'
import { useTableCreateActions } from '../composables/useTableCreateActions'
import { useTableLoadAndPagination } from '../composables/useTableLoadAndPagination'
import { useTableReferencePreload } from '../composables/useTableReferencePreload'
import { useTableRowActions } from '../composables/useTableRowActions'
import { useTableSearchAndSort } from '../composables/useTableSearchAndSort'
import { useTableUrlSync } from '../composables/useTableUrlSync'
import { useTableViewLifecycle } from '../composables/useTableViewLifecycle'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import type { TableRow } from '../../domain/types'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/architecture/domain/utils/createFieldValue'
import { getFunctionCallbacks, getTableCreateFields, getTableListFields } from '@/architecture/domain/utils/functionSchemaSelectors'
import { writeTablePageSizePreference } from './utils/tablePageSizePreference'
import { downloadTableData, downloadTableImportTemplate } from './utils/tableSpreadsheetFile'
import { TABLE_EXPORT_MAX_ROWS } from './utils/tableSpreadsheetRuntime'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

const route = useRoute()
const router = useRouter()
// 依赖注入（使用 IServiceProvider 接口，遵循依赖倒置原则）
const serviceProvider: IServiceProvider = serviceFactory
const stateManager = serviceProvider.getTableStateManager()
const domainService = serviceProvider.getTableDomainService()
const applicationService = serviceProvider.getTableApplicationService()
const workspaceStateManager = serviceProvider.getWorkspaceStateManager()  // 用于获取当前函数节点上下文

// 🔥 从状态管理器获取状态（统一状态管理）
const tableData = computed(() => stateManager.getState().data)
const loading = computed(() => stateManager.getState().loading)
const pagination = computed(() => stateManager.getState().pagination)
const searchForm = computed({
  get: () => stateManager.getState().searchForm,
  set: (value) => {
    const state = stateManager.getState()
    stateManager.setState({ ...state, searchForm: value })
  }
})
const sorts = computed({
  get: () => stateManager.getState().sorts,
  set: (value) => {
    const state = stateManager.getState()
    stateManager.setState({ ...state, sorts: value })
  }
})
const hasManualSort = computed({
  get: () => stateManager.getState().hasManualSort,
  set: (value) => {
    const state = stateManager.getState()
    stateManager.setState({ ...state, hasManualSort: value })
  }
})

// 分页相关（从 StateManager 获取）
const currentPage = computed({
  get: () => pagination.value.currentPage,
  set: (val) => {
    const state = stateManager.getState()
    stateManager.setState({
      ...state,
      pagination: { ...state.pagination, currentPage: val }
    })
  }
})
const pageSize = computed({
  get: () => pagination.value.pageSize,
  set: (val) => {
    const state = stateManager.getState()
    stateManager.setState({
      ...state,
      pagination: { ...state.pagination, pageSize: val }
    })
  }
})
const total = computed(() => pagination.value.total)

const getSortHeaderClass = (fieldCode: string): string => {
  const order = sortOrderMap.value[fieldCode]
  if (!order) return ''
  return `sort-header-active sort-header-${order === 'ascending' ? 'asc' : 'desc'}`
}

// ==================== 对话框相关 ====================

// 创建对话框
const createDialogVisible = ref(false)
const spreadsheetImportVisible = ref(false)
const spreadsheetBusy = ref(false)
const tableCreateFields = computed(() => getTableCreateFields(props.functionDetail))
const tableName = computed(() => props.functionDetail.name || props.functionDetail.code || '表格')

const {
  preloadUserInfoFromSearchForm,
  preloadDepartmentInfoFromSearchForm
} = useTableReferencePreload()

let loadTableDataRef: () => Promise<void> = async () => {}
let buildDefaultSortsRef: () => { field: string; order: 'asc' | 'desc' }[] = () => []

const { syncToURL } = useTableUrlSync({
  functionDetail: () => props.functionDetail,
  routeQuery: () => route.query,
  stateManager,
  buildDefaultSorts: () => buildDefaultSortsRef()
})

const {
  searchBarExpanded,
  idField,
  searchableFields,
  activeSearchCount,
  getSearchFieldLayoutClass,
  linkFields,
  dataFields,
  buildDefaultSorts,
  sortOrderMap,
  displaySorts,
  getFieldName,
  handleRemoveSort,
  handleClearAllSorts,
  handleSortChange,
  getSearchValue,
  updateSearchValue,
  handleSearch,
  handleReset
} = useTableSearchAndSort({
  functionDetail: () => props.functionDetail,
  domainService,
  stateManager,
  searchForm,
  sorts,
  hasManualSort,
  syncToURL,
  loadTableData: () => loadTableDataRef()
})

buildDefaultSortsRef = buildDefaultSorts

// 🔥 restoreFromURL 已移至 useTableInitialization composable

const {
  isMounted,
  skipNextTableLoad,
  loadTableData: loadTableData,
  handleSizeChange,
  handleCurrentChange
} = useTableLoadAndPagination({
  functionDetail: () => props.functionDetail,
  stateManager,
  domainService,
  applicationService,
  buildDefaultSorts: () => buildDefaultSortsRef(),
  syncToURL,
  onPageSizeChange: (size) => writeTablePageSizePreference(props.functionDetail, size)
})

loadTableDataRef = loadTableData

// ==================== 批量选择相关 ====================

const {
  isBatchDeleteMode,
  selectedRows,
  tableRef,
  enterBatchDeleteMode,
  exitBatchDeleteMode,
  handleSelectionChange,
  checkSelectable,
  handleBatchDelete
} = useTableBatchDelete({
  functionDetail: () => props.functionDetail,
  idField: () => idField.value || undefined,
  loadTableData
})

// ==================== 其他方法 ====================

const getRowFieldValue = (row: TableRow, field: FieldConfig): FieldValue => {
  const value = row[field.code]
  if (value === null || value === undefined || value === '') {
    return createEmptyRawFieldValue()
  }
  return createAutoFieldValue(value, field)
}

const getDetailIdText = (value: unknown): string => {
  if (value === null || value === undefined || value === '') {
    return '-'
  }

  if (
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean' ||
    typeof value === 'bigint' ||
    typeof value === 'symbol' ||
    typeof value === 'function'
  ) {
    return String(value)
  }

  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

const getDetailIdTitle = (value: unknown): string => {
  const text = getDetailIdText(value)
  return text === '-' ? '查看详情' : `查看详情 ${text}`
}

/**
 * 获取操作列宽度（统一「更多」下拉，固定宽度）
 */
const getActionColumnWidth = (): number => {
  return 90
}

// ==================== 回调判断 ====================

const hasAddCallback = computed(() => {
  return getFunctionCallbacks(props.functionDetail).includes('OnTableAddRow')
})

const hasDeleteCallback = computed(() => {
  return getFunctionCallbacks(props.functionDetail).includes('OnTableDeleteRows')
})

const hasUpdateCallback = computed(() => {
  return getFunctionCallbacks(props.functionDetail).includes('OnTableUpdateRow')
})

const importSpreadsheetRows = async (
  rows: Array<{ rowNumber: number, data: Record<string, unknown> }>
) => applicationService.addRows(props.functionDetail, rows)

const handleSpreadsheetCommand = async (command: string | number | object) => {
  if (command === 'import') {
    if (hasAddCallback.value) spreadsheetImportVisible.value = true
    return
  }
  if (command !== 'template' && command !== 'export') return

  spreadsheetBusy.value = true
  try {
    if (command === 'template') {
      await downloadTableImportTemplate(tableCreateFields.value, tableName.value)
      return
    }

    const snapshot = await applicationService.loadDataSnapshot(props.functionDetail, {
      maxRows: TABLE_EXPORT_MAX_ROWS,
      pageSize: 500
    })
    await downloadTableData(getTableListFields(props.functionDetail), snapshot.rows, tableName.value)
    if (snapshot.truncated) {
      ElMessage.warning(`结果共 ${snapshot.total} 行，本次已导出前 ${snapshot.rows.length} 行`)
    } else {
      ElMessage.success(`已导出 ${snapshot.rows.length} 行`)
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '导入 / 导出操作失败')
  } finally {
    spreadsheetBusy.value = false
  }
}

const {
  handleAdd,
  handleCreateSubmit,
  handleCreateDialogClose
} = useTableCreateActions({
  routeQuery: () => route.query,
  functionDetail: () => props.functionDetail,
  workspaceStateManager,
  applicationService,
  createDialogVisible
})

const {
  handleActionCommand,
  getLinkText,
  getColumnWidth,
  handleDetail,
  handleRowClick
} = useTableRowActions({
  functionDetail: () => props.functionDetail,
  router,
  stateManager,
  applicationService,
  linkFields: () => linkFields.value,
  skipNextTableLoad
})

// 🔥 使用 composable 统一管理初始化逻辑
const { initializeTable, setupQueryWatch } = useTableInitialization({
  functionDetail: computed(() => props.functionDetail),
  domainService,
  applicationService,
  stateManager,
  searchForm,
  sorts,
  hasManualSort,
  buildDefaultSorts,
  syncToURL,
  loadTableData,
  isMounted, // 🔥 传递挂载状态，用于防止卸载后继续加载数据
  preloadUserInfoFromSearchForm, // 🔥 时机 1：预加载搜索表单中的用户信息
  preloadDepartmentInfoFromSearchForm // 🔥 时机 1：预加载搜索表单中的部门信息
})

useTableAddDialogUrlSync({
  createDialogVisible,
  hasAddCallback: () => hasAddCallback.value,
  isMounted: () => isMounted.value
})

useTableViewLifecycle({
  functionDetailId: props.functionDetail.id,
  isMounted,
  initializeTable,
  setupQueryWatch,
  stateManager
})
</script>

<style scoped>
.table-view {
  --table-control-bg: var(--app-shell-panel-bg-strong);
  --table-control-bg-hover: rgba(var(--el-color-primary-rgb), 0.06);
  --table-control-border: var(--app-shell-panel-border);
  --table-control-border-hover: rgba(var(--el-color-primary-rgb), 0.28);
  --table-control-shadow-hover: 0 8px 18px rgba(15, 23, 42, 0.08);
  padding: 10px 0 0;
  background: transparent;
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
}

.toolbar {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 0;
  border: none !important;
  border-radius: 0;
  background: transparent !important;
  box-shadow: none !important;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-left: auto;
}

.toolbar :deep(.el-button) {
  height: 36px;
  border-radius: 8px;
  font-weight: 600;
  transition: border-color 0.18s ease, background-color 0.18s ease, color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.toolbar .action-btn {
  padding: 0 14px;
  box-shadow: none;
}

.toolbar .action-btn:not(.el-button--primary) {
  border: 1px solid var(--table-control-border);
  background: var(--table-control-bg);
}

.toolbar-search-btn:not(.el-button--primary),
.toolbar-secondary-btn {
  padding: 0 14px;
  border: 1px solid var(--table-control-border);
  background: var(--table-control-bg);
  box-shadow: none;
}

.toolbar .action-btn:not(.el-button--primary):hover,
.toolbar-search-btn:not(.el-button--primary):hover,
.toolbar-secondary-btn:hover {
  border-color: var(--table-control-border-hover);
  color: var(--el-color-primary);
  background: var(--table-control-bg-hover);
  box-shadow: var(--table-control-shadow-hover);
  transform: translateY(-1px);
}

.toolbar .action-btn.el-button--primary {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.toolbar .action-btn.el-button--primary:hover {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
  transform: translateY(-1px);
}

.toolbar .action-btn.el-button--danger,
.toolbar .action-btn-danger {
  border-color: rgba(239, 68, 68, 0.26);
  background: #fff1f2;
  color: #dc2626;
  box-shadow: none;
}

.toolbar .action-btn.el-button--danger:hover,
.toolbar .action-btn-danger:hover {
  border-color: rgba(239, 68, 68, 0.38);
  background: #ffe4e6;
  color: #b91c1c;
  box-shadow: 0 10px 24px rgba(239, 68, 68, 0.14);
  transform: translateY(-1px);
}

.toolbar-search-btn.el-button--primary {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.toolbar-search-btn.el-button--primary:hover {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
  transform: translateY(-1px);
}

.toolbar-collapse-btn {
  padding: 0 2px;
  color: var(--el-text-color-secondary);
}

.toolbar-collapse-btn:hover {
  color: var(--el-color-primary);
}

.search-bar-wrapper {
  margin-bottom: 16px;
}

.search-bar {
  margin-bottom: 0;
  border: none;
  background: transparent;
  box-shadow: none;
}

.search-bar-inner {
  padding: 16px;
  border-radius: 8px;
  border: 1px solid var(--table-control-border);
  background: var(--app-shell-panel-bg-strong);
  box-shadow: var(--app-auth-card-shadow-soft);
}

.search-bar .search-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px 14px;
  align-items: start;
}

.search-bar :deep(.search-form > .el-form-item) {
  display: flex;
  flex-direction: column;
  justify-self: stretch;
  margin-right: 0;
  margin-bottom: 0;
  width: 100%;
  min-width: 0;
  align-items: flex-start;
}

.search-bar :deep(.search-form > .el-form-item.search-field-layout--wide) {
  grid-column: span 2;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__label-wrap) {
  width: 100%;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__label) {
  width: 100%;
  display: flex;
  justify-content: flex-start;
  padding: 0 0 6px;
  line-height: 1.25;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__content) {
  display: flex;
  align-items: stretch;
  flex: 1 1 auto;
  width: 100%;
  min-width: 0;
  margin-left: 0 !important;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__content > *) {
  width: 100%;
  min-width: 0;
}

.search-bar :deep(.el-input__wrapper),
.search-bar :deep(.el-select__wrapper),
.search-bar :deep(.el-textarea__inner),
.search-bar :deep(.el-date-editor .el-input__wrapper),
.search-bar :deep(.department-select-display),
.search-bar :deep(.user-search-display) {
  background: var(--app-auth-input-bg);
  border-color: var(--app-auth-input-border);
  border-radius: 8px;
  box-shadow: none;
  transition: all 0.3s ease;
}

.search-bar :deep(.el-input__wrapper:hover),
.search-bar :deep(.el-select__wrapper:hover),
.search-bar :deep(.el-textarea__inner:hover),
.search-bar :deep(.el-date-editor .el-input__wrapper:hover),
.search-bar :deep(.department-select-display:hover),
.search-bar :deep(.user-search-display:hover) {
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.search-bar :deep(.el-input__wrapper.is-focus),
.search-bar :deep(.el-select__wrapper.is-focused),
.search-bar :deep(.el-textarea__inner:focus),
.search-bar :deep(.el-date-editor .el-input__wrapper.is-focus) {
  border-color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-focus);
}

.search-bar :deep(.el-input__inner::placeholder),
.search-bar :deep(.el-textarea__inner::placeholder) {
  color: #94a3b8;
}

/* 🔥 排序信息条样式 */
.sort-info-bar {
  margin-bottom: 16px;
  padding: 10px 12px;
  background: var(--app-shell-panel-bg-strong);
  border: 1px solid var(--table-control-border);
  border-radius: 8px;
  box-shadow: var(--app-auth-card-shadow-soft);
  display: flex;
  align-items: center;
}

.sort-info-content {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  flex-wrap: wrap;
}

.sort-label {
  height: 26px;
  display: inline-flex;
  align-items: center;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--app-shell-panel-muted-bg);
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 700;
  white-space: nowrap;
}

.sort-items {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  flex: 1;
}

.sort-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: default;
}

.sort-field-name {
  font-weight: 500;
}

.sort-icon {
  font-size: 12px;
  margin-left: 2px;
}

.sort-separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  font-weight: 500;
  margin: 0 4px;
}

.table-clear-sorts-btn {
  margin-left: auto;
  white-space: nowrap;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--table-control-border);
  border-radius: 8px;
  background: var(--table-control-bg);
  color: var(--el-text-color-regular);
  font-weight: 600;
}

.table-clear-sorts-btn:hover {
  border-color: var(--table-control-border-hover);
  background: var(--table-control-bg-hover);
  color: var(--el-color-primary);
}

/* 表格骨架屏（加载中） */
.table-skeleton-wrap {
  min-height: 320px;
  padding: 4px 0 16px;
}

.table-skeleton-wrap .table-skeleton {
  width: 100%;
}

@media (max-width: 900px) {
  .search-bar .search-form {
    grid-template-columns: 1fr;
  }

  .search-bar :deep(.search-form > .el-form-item.search-field-layout--wide) {
    grid-column: 1 / -1;
  }
}

.table-skeleton-wrap .el-skeleton__item {
  margin-bottom: 12px;
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* 🔥 表格基础样式 */
:deep(.el-table) {
  background-color: var(--app-shell-panel-bg-strong) !important;
  border: 1px solid var(--app-shell-panel-border) !important;
  border-radius: 8px !important;
  box-shadow: var(--app-shell-panel-shadow-soft);
  flex: 1;
  overflow: auto;
}

:deep(.el-table__inner-wrapper) {
  border: none !important;
  border-radius: 8px !important;
}

:deep(.el-table__inner-wrapper::before) {
  display: none !important;
}

:deep(.el-table__header-wrapper) {
  border: none !important;
}

:deep(.el-table__body-wrapper) {
  border: none !important;
}

:deep(.el-table th),
:deep(.el-table td) {
  border-right: none !important;
  border-left: none !important;
}

:deep(.el-table th:first-child),
:deep(.el-table td:first-child) {
  border-left: none !important;
}

:deep(.el-table th:last-child),
:deep(.el-table td:last-child) {
  border-right: none !important;
}

:deep(.el-table__body tr) {
  background-color: transparent !important;
}

:deep(.el-table__body tr.el-table__row--striped) {
  background-color: transparent !important;
}

:deep(.el-table__body tr.el-table__row--striped td) {
  background-color: var(--app-shell-panel-bg-strong) !important;
}

:deep(.el-table__body tr:hover > td) {
  background-color: rgba(var(--el-color-primary-rgb), 0.04) !important;
}

/* 整行可点击进入详情 */
:deep(.table-row-clickable .el-table__body tr) {
  cursor: pointer;
}

:deep(.el-table__header th.el-table__cell) {
  background-color: var(--app-shell-panel-muted-bg);
  color: var(--el-text-color-secondary);
  font-weight: 600;
  border-top: none;
}

:deep(.el-table__header th.el-table__cell.sort-header-active) {
  color: var(--el-color-primary);
}

:deep(.el-table__header th.el-table__cell.sort-header-active .cell) {
  font-weight: 700;
}

:deep(.el-table__header th.el-table__cell.sort-header-asc .sort-caret.ascending) {
  border-bottom-color: var(--el-color-primary) !important;
}

:deep(.el-table__header th.el-table__cell.sort-header-desc .sort-caret.descending) {
  border-top-color: var(--el-color-primary) !important;
}

:deep(.el-table__header th.el-table__cell.sort-header-active .caret-wrapper) {
  opacity: 1;
}

:deep(.el-table td.el-table__cell),
:deep(.el-table th.el-table__cell.is-leaf) {
  border-bottom: 1px solid var(--app-shell-panel-border);
}

:deep(.el-table td.el-table__cell) {
  background: var(--app-shell-panel-bg-strong) !important;
}

:deep(.table-data-column .cell) {
  display: block;
  min-width: 0;
  line-height: 1.45;
  white-space: normal;
  overflow: hidden;
}

:deep(.table-data-column .cell > *) {
  min-width: 0;
  max-width: 100%;
}

:deep(.table-data-column .table-cell-value),
:deep(.table-data-column .files-display-text) {
  display: block;
  min-width: 0;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:deep(.table-data-column .input-widget .table-cell-value),
:deep(.table-data-column .textarea-widget .table-cell-value),
:deep(.table-data-column .rich-text-widget .table-cell-value),
:deep(.table-data-column .text-widget .table-cell-value),
:deep(.table-data-column .table-cell-text),
:deep(.table-data-column .formatted-content),
:deep(.table-data-column .text-content),
:deep(.table-data-column .code-content),
:deep(.table-data-column .html-table-cell),
:deep(.table-data-column .markdown-table-cell),
:deep(.table-data-column .csv-preview),
:deep(.table-data-column .csv-preview-text),
:deep(.table-data-column .html-content-preview) {
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

:deep(.table-data-column .table-cell-multiselect),
:deep(.table-data-column .files-table-cell),
:deep(.table-data-column .files-select-display) {
  min-width: 0;
  max-width: 100%;
  flex-wrap: nowrap;
  overflow: hidden;
}

:deep(.table-data-column .files-table-cell),
:deep(.table-data-column .files-table-preview-list),
:deep(.table-data-column .files-select-display) {
  width: 100%;
  justify-content: flex-start;
  text-align: left;
}

.table-view :deep(.table-data-column .formatted-content),
.table-view :deep(.table-data-column .text-content),
.table-view :deep(.table-data-column .code-content),
.table-view :deep(.table-data-column .html-table-cell),
.table-view :deep(.table-data-column .markdown-table-cell) {
  padding: 0;
  border: none;
  background: transparent;
  box-shadow: none;
}

:deep(.control-column .cell) {
  min-width: 0;
  overflow: hidden;
}

/* 固定列必须拥有实体底色，否则横向滚动时会与经过其下方的单元格叠字。 */
:deep(.table-with-fixed-column td.control-column.el-table-fixed-column--left),
:deep(.table-with-fixed-column td.action-column.el-table-fixed-column--right) {
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color)) !important;
}

:deep(.table-with-fixed-column th.control-column.el-table-fixed-column--left),
:deep(.table-with-fixed-column th.action-column.el-table-fixed-column--right) {
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light)) !important;
}

:deep(.table-with-fixed-column .el-table__body tr:hover > td.control-column.el-table-fixed-column--left),
:deep(.table-with-fixed-column .el-table__body tr:hover > td.action-column.el-table-fixed-column--right) {
  background: var(--el-fill-color-light) !important;
}

:deep(.table-with-fixed-column td.action-column .cell) {
  opacity: 1 !important;
  visibility: visible !important;
}

.detail-icon-button {
  min-width: 44px;
  width: 100%;
  max-width: 100%;
  height: 32px;
  padding: 0 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--el-color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.detail-icon-button .el-icon {
  flex: 0 0 auto;
}

.detail-icon-button:hover {
  background-color: var(--el-color-primary-light-9);
}

.detail-icon-button:focus-visible {
  outline: 2px solid var(--el-color-primary-light-5);
  outline-offset: 2px;
}

.detail-id-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
}

.action-more-btn {
  margin: 0;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--table-control-border);
  border-radius: 8px;
  background: var(--table-control-bg);
  color: var(--el-text-color-regular);
  font-weight: 600;
  box-shadow: none;
}

.action-more-btn:hover,
.action-more-btn:focus {
  border-color: var(--table-control-border-hover);
  background: var(--table-control-bg-hover);
  color: var(--el-color-primary);
  box-shadow: var(--table-control-shadow-hover);
}

.dropdown-link-content,
.dropdown-action-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.link-icon {
  font-size: 14px;
}

.internal-icon {
  color: var(--el-color-primary);
}

.delete-action-text {
  color: var(--el-color-danger);
}

</style>

<style lang="scss">
.table-action-dropdown.el-popper {
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 8px;
  box-shadow: 0 16px 34px rgba(15, 23, 42, 0.14);
}

.table-action-dropdown .el-dropdown-menu {
  padding: 6px;
  border-radius: 8px;
}

.table-action-dropdown .el-dropdown-menu__item {
  min-height: 34px;
  border-radius: 6px;
  font-weight: 500;
}

.table-action-dropdown .el-dropdown-menu__item:not(.is-disabled):hover {
  background: rgba(var(--el-color-primary-rgb), 0.07);
  color: var(--el-color-primary);
}
</style>
