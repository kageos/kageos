<!--
  TableView - 表格视图
  新架构的展示层组件
  
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
     - 搜索条件同步到 URL（`like=field:value`）
     - 排序条件同步到 URL（`sorts=field:order`）
     - 分页信息同步到 URL（`page=1&page_size=20`）
     - 新增弹窗状态同步到 URL（`_tab=OnTableAddRow`）
  
  ============================================
  🎯 设计思路
  ============================================
  
  1. **分层架构**：
     - Presentation Layer：纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
  
  2. **权限检查**：
     - UI 层面：使用 `canCreate`、`canUpdate`、`canDelete` 控制按钮显示
     - 提交时：在 `handleCreateSubmit`、`handleBatchDelete` 中再次检查权限
     - 权限来源：从 `currentFunctionNode` 获取节点权限信息
  
  3. **URL 同步**：
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
  
  3. **权限提示**：
     - 无权限时显示锁定图标和"需权限"提示
     - 点击无权限按钮时跳转到权限申请页面
  
  ============================================
  ⚠️ 注意事项
  ============================================
  
  1. **权限检查**：
     - 必须在 UI 层面和提交时都检查权限
     - 权限检查失败时，显示提示并跳转到申请页面
  
  2. **URL 同步**：
     - 新增弹窗状态使用 `_tab=OnTableAddRow` 标识
     - 详情抽屉状态使用 `_tab=detail` 标识（编辑模式不设置 `_tab`）
     - 表单字段参数只在新增模式下同步到 URL
  
  3. **数据流**：
     - TableView → TableApplicationService → TableDomainService → TableStateManager
     - 状态变化通过事件总线通知其他组件
-->

<template>
  <div class="table-view">
    <!-- ⭐ 权限不足提示：使用 PermissionDeniedView 组件 -->
    <PermissionDeniedView v-if="permissionError" />

    <!-- 工具栏 -->
    <div class="toolbar" v-if="hasAddCallback || hasDeleteCallback">
      <div class="toolbar-left">
        <!-- 导入按钮：需要 table:write 权限 -->
        <el-button 
          v-if="hasAddCallback"
          :type="canCreate ? 'default' : 'default'"
          :plain="!canCreate"
          @click="canCreate ? handleImport() : handleApplyPermissionForAction(FunctionPermission.write)"
          :icon="Upload"
          class="action-btn"
          :class="{ 'action-btn-no-permission': !canCreate }"
        >
          <template v-if="!canCreate">
            <el-icon><Lock /></el-icon>
            导入（需{{ getPermissionShortName(FunctionPermission.write) }}）
          </template>
          <template v-else>导入</template>
        </el-button>
        <!-- 新增按钮：需要 table:write 权限，无权限时可点击跳转申请 -->
        <el-button 
          v-if="hasAddCallback" 
          :type="canCreate ? 'primary' : 'default'"
          :plain="!canCreate"
          @click="canCreate ? handleAdd() : handleApplyPermissionForAction(FunctionPermission.write)" 
          :icon="Plus"
          class="action-btn"
          :class="{ 'action-btn-no-permission': !canCreate }"
        >
          <template v-if="!canCreate">
            <el-icon><Lock /></el-icon>
            新增（需{{ getPermissionShortName(FunctionPermission.write) }}）
          </template>
          <template v-else>新增</template>
        </el-button>
        <!-- 批量删除按钮：需要 table:delete 权限，无权限时可点击跳转申请 -->
        <el-button 
          v-if="hasDeleteCallback && !isBatchDeleteMode" 
          :type="canDelete ? 'danger' : 'default'"
          :plain="!canDelete"
          @click="canDelete ? enterBatchDeleteMode() : handleApplyPermissionForAction(FunctionPermission.delete)"
          :icon="canDelete ? Delete : Lock"
          class="action-btn"
          :class="{ 'action-btn-no-permission': !canDelete }"
        >
          {{ canDelete ? '批量删除' : `批量删除（需${getPermissionShortName(FunctionPermission.delete)}）` }}
        </el-button>
        <template v-if="hasDeleteCallback && isBatchDeleteMode">
          <el-button 
            type="danger" 
            @click="handleBatchDelete"
            :icon="Delete"
            :disabled="selectedRows.length === 0"
          >
            删除选中 ({{ selectedRows.length }})
          </el-button>
          <el-button 
            @click="exitBatchDeleteMode"
          >
            取消
          </el-button>
        </template>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div v-if="searchableFields.length > 0" class="search-bar">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <template v-for="field in searchableFields" :key="field.code">
          <el-form-item :label="field.name">
            <SearchInput
              :field="field"
              :search-type="field.search || ''"
              :model-value="getSearchValue(field)"
              :function-method="props.functionDetail.method || 'GET'"
              :function-router="props.functionDetail.router"
              @update:model-value="(value: any) => {
                // 🔥 修复：用户选择选项时应该立即触发搜索（无论选择还是清空）
                // 之前的逻辑是 isClearing 时才搜索，这是完全错误的
                // 正确的逻辑是：任何值变化都应该立即触发搜索
                updateSearchValue(field, value, true)
              }"
            />
          </el-form-item>
        </template>

        <el-form-item>
          <el-button type="primary" @click="handleSearch">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="handleReset">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
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
          link
          type="primary"
          size="small"
          @click="handleClearAllSorts"
          class="clear-all-sorts-btn"
        >
          清除所有排序
        </el-button>
      </div>
    </div>

    <!-- 表格 -->
    <el-table
      ref="tableRef"
      v-loading="loading"
      :data="tableData"
      :stripe="false"
      style="width: 100%"
      class="table-with-fixed-column"
      @sort-change="handleSortChange"
      @selection-change="handleSelectionChange"
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
        width="80"
        class-name="control-column"
        :sortable="getSortableConfig(idField)"
        :sort-order="sortOrderMap[idField.code] || null"
      >
        <template #default="{ row }">
          <span 
            class="link-text"
            @click.stop="handleDetail(row)"
          >
            #{{ row[idField.code] }}
          </span>
        </template>
      </el-table-column>

      <!-- 数据列（排除ID列） -->
      <el-table-column
        v-for="field in dataFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name"
        :sortable="getSortableConfig(field)"
        :sort-order="sortOrderMap[field.code] || null"
        :min-width="getColumnWidth(field)"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <WidgetComponent
            :field="field"
            :value="getRowFieldValue(row, field.code)"
            mode="table-cell"
            :row-data="row"
          />
        </template>
      </el-table-column>

      <!-- 操作列 -->
      <el-table-column 
        v-if="hasDeleteCallback || linkFields.length > 0" 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
        class-name="action-column"
      >
        <template #default="{ row }">
          <div class="action-buttons">
            <!-- 链接区域：只有 1 个链接时直接显示，超过 1 个时使用下拉菜单 -->
            <template v-if="linkFields.length === 1">
              <LinkWidget
                :field="linkFields[0]"
                :value="convertToFieldValue(row[linkFields[0].code], linkFields[0])"
                :field-path="linkFields[0].code"
                mode="table-cell"
                class="action-link"
              />
            </template>
            
            <!-- 多个链接下拉菜单（超过 1 个时显示） -->
            <el-dropdown
              v-else-if="linkFields.length > 1"
              trigger="click"
              placement="bottom-end"
              @command="(fieldCode: string) => handleLinkClick(fieldCode, row)"
            >
              <el-button link type="primary" size="small" class="more-links-btn">
                <el-icon><More /></el-icon>
                更多
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item
                    v-for="linkField in linkFields"
                    :key="linkField.code"
                    :command="linkField.code"
                  >
                    <div class="dropdown-link-content">
                      <el-icon v-if="linkField.widget?.config?.icon" class="link-icon">
                        <component :is="linkField.widget.config.icon" />
                      </el-icon>
                      <el-icon v-else class="link-icon internal-icon"><Right /></el-icon>
                      <span>{{ getLinkText(linkField, row[linkField.code]) }}</span>
                    </div>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            
            <!-- 更新按钮：需要 table:update 权限，无权限时可点击跳转申请 -->
            <!-- 注意：更新操作通过详情抽屉实现，这里点击打开详情抽屉 -->
            <el-button 
              v-if="hasUpdateCallback"
              link 
              :type="canUpdate ? 'primary' : 'default'"
              size="small"
              class="update-btn"
              :class="{ 'action-btn-no-permission': !canUpdate }"
              @click.stop="canUpdate ? handleDetail(row) : handleApplyPermissionForAction(FunctionPermission.update)"
            >
              <el-icon><component :is="canUpdate ? Edit : Lock" /></el-icon>
              {{ canUpdate ? '更新' : `更新（需${getPermissionShortName(FunctionPermission.update)}）` }}
            </el-button>
            <!-- 删除按钮：需要 table:delete 权限，无权限时可点击跳转申请 -->
            <el-button 
              v-if="hasDeleteCallback"
              link 
              :type="canDelete ? 'danger' : 'default'"
              size="small"
              class="delete-btn"
              :class="{ 'action-btn-no-permission': !canDelete }"
              @click.stop="canDelete ? handleDelete(row) : handleApplyPermissionForAction(FunctionPermission.delete)"
            >
              <el-icon><component :is="canDelete ? Delete : Lock" /></el-icon>
              {{ canDelete ? '删除' : `删除（需${getPermissionShortName(FunctionPermission.delete)}）` }}
            </el-button>
          </div>
        </template>
      </el-table-column>
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
      :fields="props.functionDetail.response || []"
      mode="create"
      :router="props.functionDetail.router"
      :method="props.functionDetail.method || 'POST'"
      :initial-data="createFormInitialData"
      @submit="handleCreateSubmit"
      @close="handleCreateDialogClose"
    />

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
                v-for="(field, fieldIndex) in editableFields"
                :key="field?.code || `field-${fieldIndex}`"
                :prop="field?.code"
                :label="field?.name || '未知字段'"
                :min-width="120"
              >
                <template #default="{ row, $index }">
                  <span
                    :class="{
                      'error-cell': importErrors.some(e => e.index === $index && e.field === field?.name)
                    }"
                  >
                    <template v-if="field?.code">
                      <!-- 时间戳字段：转换为可读格式 -->
                      <template v-if="(field.code === 'created_at' || field.code === 'updated_at') && field.widget?.type === 'timestamp'">
                        {{ row[field.code] ? new Date(row[field.code]).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '' }}
                      </template>
                      <!-- 其他字段：直接显示 -->
                      <template v-else>
                        {{ row[field.code] ?? '' }}
                      </template>
                    </template>
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
import { computed, ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElIcon, ElTable, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElText, ElCard, ElUpload, ElAlert } from 'element-plus'
import { Search, Refresh, Delete, Plus, ArrowUp, ArrowDown, More, Right, Lock, Document, Key, Edit, Upload, Download } from '@element-plus/icons-vue'
import { eventBus, TableEvent, WorkspaceEvent, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/utils/routeSource'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../../presentation/widgets/WidgetComponent.vue'
import SearchInput from '@/components/SearchInput.vue'
import FormDialog from '@/components/FormDialog.vue'
import { getSortableConfig } from '@/utils/fieldSort'
import { buildURLSearchParams } from '@/utils/searchParams'
import { WidgetType } from '@/core/constants/widget'
import { useTableInitialization } from '../composables/useTableInitialization'
import { convertToFieldValue } from '@/utils/field'
import { resolveWorkspaceUrl } from '@/utils/route'
import { parseLinkValue, addLinkTypeToUrl, isLinkNavigation, LINK_TYPE_QUERY_KEY } from '@/utils/linkNavigation'
import LinkWidget from '@/architecture/presentation/widgets/LinkWidget.vue'
import { TABLE_PARAM_KEYS, SEARCH_PARAM_KEYS } from '@/utils/urlParams'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { useUserInfoStore } from '@/stores/userInfo'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import type { TableRow, SearchParams, SortParams, SortItem } from '../../domain/services/TableDomainService'
import type { UserInfo } from '@/types'
import { hasPermission, TablePermissions, FunctionPermission, buildPermissionApplyURL, getPermissionShortName } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import type { PermissionInfo } from '@/utils/permission'
import { parseExcelFile } from '@/utils/excelImport'
import PermissionDeniedView from '../components/PermissionDeniedView.vue'

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
const workspaceStateManager = serviceProvider.getWorkspaceStateManager()  // ⭐ 用于获取当前函数节点的权限信息

// 🔥 从状态管理器获取状态（统一状态管理）
const tableData = computed(() => stateManager.getData())
const loading = computed(() => stateManager.isLoading())
const pagination = computed(() => stateManager.getPagination())
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

// ==================== 批量选择相关 ====================

/** 是否处于批量删除模式 */
const isBatchDeleteMode = ref(false)

/** 选中的行数据 */
const selectedRows = ref<TableRow[]>([])

/** 表格引用（用于控制复选框状态） */
const tableRef = ref<InstanceType<typeof ElTable> | null>(null)

/**
 * 进入批量删除模式
 */
const enterBatchDeleteMode = (): void => {
  isBatchDeleteMode.value = true
  selectedRows.value = []
  // 清空之前的选择
  if (tableRef.value) {
    tableRef.value.clearSelection()
  }
}

/**
 * 退出批量删除模式
 */
const exitBatchDeleteMode = (): void => {
  isBatchDeleteMode.value = false
  selectedRows.value = []
  // 清空选择
  if (tableRef.value) {
    tableRef.value.clearSelection()
  }
}

/**
 * 处理选择变化
 * @param selection 选中的行数组
 */
const handleSelectionChange = (selection: TableRow[]): void => {
  selectedRows.value = selection
}

/**
 * 判断行是否可选
 * @param row 行数据
 * @param index 行索引
 * @returns 是否可选
 */
const checkSelectable = (row: TableRow, index: number): boolean => {
  // 所有行都可以选择
  return true
}

/**
 * 批量删除
 */
const handleBatchDelete = async (): Promise<void> => {
  if (selectedRows.value.length === 0) {
    ElMessage.warning('请先选择要删除的记录')
    return
  }

  // 🔥 安全修复：检查删除权限
  const node = currentFunctionNode.value
  if (!node) {
    ElMessage.error('无法获取函数节点信息，无法验证权限')
    return
  }
  
  if (!hasPermission(node, TablePermissions.delete)) {
    ElNotification.warning({
      title: '权限不足',
      message: '您没有删除该表格记录的权限',
      duration: 3000
    })
    // 跳转到权限申请页面
    const applyUrl = buildPermissionApplyURL(
      node.full_code_path || '',
      TablePermissions.delete,
      props.functionDetail?.template_type
    )
    router.push(applyUrl)
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedRows.value.length} 条记录吗？`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    // 获取所有选中行的 ID
    const ids = selectedRows.value
      .map((row: TableRow) => {
        // 尝试从 id 字段获取，如果没有则尝试从 idField 获取
        if (row.id) return row.id
        if (idField.value && row[idField.value.code]) {
          return row[idField.value.code]
        }
        return null
      })
      .filter((id: any): id is number => id !== null && typeof id === 'number')

    if (ids.length === 0) {
      ElMessage.error('无法获取记录 ID，删除失败')
      return
    }

    // 调用批量删除 API
    const { tableDeleteRows } = await import('@/api/function')
    await tableDeleteRows(props.functionDetail.method || 'GET', props.functionDetail.router, ids)

    // 显示成功提示
    ElNotification.success({
      title: '删除成功',
      message: `已成功删除 ${ids.length} 条记录`,
      duration: 3000,
      position: 'top-right'
    })

    // 清空选择
    selectedRows.value = []
    if (tableRef.value) {
      tableRef.value.clearSelection()
    }

    // 退出批量删除模式
    isBatchDeleteMode.value = false

    // 重新加载数据
    await loadTableData()
  } catch (error: any) {
    if (error !== 'cancel') {
      const errorMessage = error?.response?.data?.msg || error?.message || '批量删除失败'
      ElNotification.error({
        title: '删除失败',
        message: errorMessage,
        duration: 5000,
        position: 'top-right'
      })
    }
  }
}

// ==================== 对话框相关 ====================

// 创建对话框
const createDialogVisible = ref(false)

// 从 URL 查询参数中提取新增表单的初始数据
const createFormInitialData = computed(() => {
  const initialData: Record<string, any> = {}
  const query = route.query
  
  // 只有存在 _tab=OnTableAddRow 参数时才提取初始数据
  if (query._tab !== 'OnTableAddRow') {
    return initialData
  }
  
  // 遍历所有查询参数，如果字段在 response 中，添加到 initialData
  if (props.functionDetail?.response) {
    props.functionDetail.response.forEach((field: FieldConfig) => {
      const fieldCode = field.code
      const queryValue = query[fieldCode]
      
      // 🔥 处理数组类型的查询参数（取第一个值）
      const value = Array.isArray(queryValue) ? queryValue[0] : queryValue
      
      if (value !== undefined && value !== null && value !== '') {
        // 类型转换：根据字段类型转换值
        if (field.data?.type === 'int' || field.data?.type === 'integer') {
          const intValue = parseInt(String(value), 10)
          if (!isNaN(intValue)) {
            initialData[fieldCode] = intValue
          }
        } else if (field.data?.type === 'float' || field.data?.type === 'number') {
          const floatValue = parseFloat(String(value))
          if (!isNaN(floatValue)) {
            initialData[fieldCode] = floatValue
          }
        } else if (field.data?.type === 'bool' || field.data?.type === 'boolean') {
          const strValue = String(value)
          initialData[fieldCode] = strValue === 'true' || strValue === '1'
        } else {
          initialData[fieldCode] = value
        }
      }
    })
  }
  
  return initialData
})

// ==================== 用户信息预加载 ====================

const userInfoStore = useUserInfoStore()

// 🔥 移除 userInfoMap，UserDisplay 组件直接从 userInfoStore 读取（预加载已完成，store 中肯定有缓存）

/**
 * 🔥 预加载用户信息（时机 1：搜索表单中的用户信息）
 * 在数据加载前预加载，确保搜索表单渲染时已有用户信息
 */
async function preloadUserInfoFromSearchForm(functionDetail: FunctionDetail, searchFormData: Record<string, any>): Promise<void> {
  try {
    // 1. 识别所有用户字段（request + response）
    // 🔥 确保 request 和 response 是数组
    const requestFields = Array.isArray(functionDetail.request) ? functionDetail.request : []
    const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
    
    const userFields = [
      ...requestFields.filter(f => f.widget?.type === 'user'),
      ...responseFields.filter(f => f.widget?.type === 'user')
    ]
    
    if (userFields.length === 0) {
      return
    }
    
    // 2. 从搜索表单中收集所有用户名
    const usernames = new Set<string>()
    userFields.forEach(field => {
      const value = searchFormData[field.code]
      if (value) {
        // 处理数组（如 in=create_by:luobei,zhangsan）
        if (Array.isArray(value)) {
          value.forEach(v => {
            if (v) usernames.add(String(v))
          })
        } else {
          // 处理字符串（如 creator=liubeiluo）
          usernames.add(String(value))
        }
      }
    })
    
    if (usernames.size === 0) {
      return
    }
    
    // 3. 批量查询用户信息（使用 batchGetUserInfo，自动处理过期数据）
    // 🔥 预加载到 store 缓存即可，UserDisplay 组件会直接从 store 读取
    await userInfoStore.batchGetUserInfo([...usernames])
  } catch (error) {
    console.error('[TableView] 预加载搜索表单中的用户信息失败', error)
  }
}

/**
 * 🔥 预加载用户信息（时机 2：表格数据中的用户信息）
 * 在数据加载后预加载，确保表格渲染时已有用户信息
 */
async function preloadUserInfoFromTableData(functionDetail: FunctionDetail, tableDataArray: TableRow[]): Promise<void> {
  try {
    // 1. 识别所有用户字段（response 字段）
    // 🔥 确保 response 是数组
    const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
    const userFields = responseFields.filter(f => f.widget?.type === 'user')
    
    if (userFields.length === 0 || !tableDataArray || tableDataArray.length === 0) {
      return
    }
    
    // 2. 从表格数据中收集所有用户名
    const usernames = new Set<string>()
    tableDataArray.forEach(row => {
      userFields.forEach(field => {
        const value = row[field.code]
        if (value !== null && value !== undefined && value !== '') {
          usernames.add(String(value))
        }
      })
    })
    
    if (usernames.size === 0) {
      return
    }
    
    // 3. 批量查询用户信息（使用 batchGetUserInfo，自动处理过期数据）
    // 🔥 预加载到 store 缓存即可，UserDisplay 组件会直接从 store 读取
    await userInfoStore.batchGetUserInfo([...usernames])
  } catch (error) {
    console.error('[TableView] 预加载表格数据中的用户信息失败', error)
  }
}

// ==================== 字段计算属性 ====================

/**
 * ID 字段（用于控制中心列）
 */
const idField = computed(() => {
  return (props.functionDetail.response || []).find((field: FieldConfig) => field.widget?.type === WidgetType.ID)
})

/**
 * 可搜索字段（从 Domain Service 获取，遵循依赖倒置原则）
 */
const searchableFields = computed(() => {
  return domainService.getSearchableFields(props.functionDetail)
})

/**
 * 可见字段（根据 table_permission 过滤）
 */
const visibleFields = computed(() => {
  return (props.functionDetail.response || []).filter((field: FieldConfig) => {
    const permission = field.table_permission || ''
    return permission === '' || permission === 'read'
  })
})

/**
 * Link 字段（用于操作列显示）
 */
const linkFields = computed(() => {
  return visibleFields.value.filter((field: FieldConfig) => field.widget?.type === WidgetType.LINK)
})

/**
 * 数据字段（排除ID列和Link列，Link列在操作区域显示）
 */
const dataFields = computed(() => {
  return visibleFields.value.filter((field: FieldConfig) => 
    field.widget?.type !== WidgetType.ID && field.widget?.type !== WidgetType.LINK
  )
})

// ==================== 排序相关 ====================

/**
 * 获取 ID 字段的 code
 */
const getIdFieldCode = (): string | null => {
  return idField.value?.code || null
}

/**
 * 构建默认排序（id 降序）
 */
const buildDefaultSorts = (): SortItem[] => {
  const idFieldCode = getIdFieldCode()
  if (idFieldCode) {
    return [{ field: idFieldCode, order: 'desc' }]
  }
  return []
}

/**
 * 排序状态映射（用于 el-table-column 的 sort-order）
 */
const sortOrderMap = computed<Record<string, 'ascending' | 'descending' | null>>(() => {
  const map: Record<string, 'ascending' | 'descending' | null> = {}
    sorts.value.forEach((sort: SortItem) => {
      map[sort.field] = sort.order === 'asc' ? 'ascending' : 'descending'
    })
  // 如果没有手动排序且存在 ID 字段，显示默认的 ID 排序
  if (sorts.value.length === 0 && !hasManualSort.value && idField.value) {
    map[idField.value.code] = 'descending'
  }
  return map
})

/**
 * 显示排序列表（用于排序信息条）
 */
const displaySorts = computed(() => {
  if (sorts.value.length > 0) {
    return sorts.value
  }
  // 如果没有手动排序且存在 ID 字段，显示默认的 ID 排序
  if (idField.value && !hasManualSort.value) {
    return [{ field: idField.value.code, order: 'desc' as const }]
  }
  return []
})

/**
 * 获取字段名称
 */
const getFieldName = (fieldCode: string): string => {
  const field = visibleFields.value.find((f: FieldConfig) => f.code === fieldCode)
  return field?.name || fieldCode
}

/**
 * 移除单个排序
 */
const handleRemoveSort = (fieldCode: string): void => {
    sorts.value = sorts.value.filter((item: SortItem) => item.field !== fieldCode)
  if (sorts.value.length === 0) {
    hasManualSort.value = false
  }
  syncToURL()
  loadTableData()
}

/**
 * 清除所有排序
 */
const handleClearAllSorts = (): void => {
  sorts.value = []
  hasManualSort.value = false
  syncToURL()
  loadTableData()
}

/**
 * 排序变化
 */
const handleSortChange = (sortInfo: { prop?: string; order?: string }): void => {
  const currentState = stateManager.getState()
  let newSorts = [...currentState.sorts]
  
  if (sortInfo && sortInfo.prop && sortInfo.order && sortInfo.order !== '') {
    const field = sortInfo.prop
    const order = sortInfo.order === 'ascending' ? 'asc' : 'desc'
    
    // id 排序与其他排序互斥
    const idFieldCode = getIdFieldCode()
    if (idFieldCode) {
      newSorts = newSorts.filter((item: SortItem) => item.field !== idFieldCode)
    }
    
    // 添加或更新排序项
    const existingIndex = newSorts.findIndex((item: SortItem) => item.field === field)
    if (existingIndex >= 0) {
      newSorts[existingIndex].order = order
    } else {
      newSorts.push({ field, order })
    }
  } else {
    // 取消该字段的排序
    if (sortInfo.prop) {
      newSorts = newSorts.filter((item: SortItem) => item.field !== sortInfo.prop)
    }
  }
  
  stateManager.setState({
    ...currentState,
    sorts: newSorts,
    hasManualSort: true
  })
  
  syncToURL()
  loadTableData()
}

// ==================== 搜索相关 ====================

/**
 * 获取搜索值
 */
const getSearchValue = (field: FieldConfig): any => {
  const value = searchForm.value[field.code]
  return value === undefined ? null : value
}

/**
 * 更新搜索值
 */
const updateSearchValue = (field: FieldConfig, value: any, shouldSearch: boolean = false): void => {
  const currentState = stateManager.getState()
  const newSearchForm = { ...currentState.searchForm }
  
  if (value === null || value === undefined || 
      (Array.isArray(value) && value.length === 0) || 
      (typeof value === 'string' && value.trim() === '')) {
    delete newSearchForm[field.code]
  } else {
    newSearchForm[field.code] = value
  }
  
  stateManager.setState({ ...currentState, searchForm: newSearchForm })
  syncToURL()
  if (shouldSearch) {
    loadTableData()
  }
}

/**
 * 搜索
 */
const handleSearch = (): void => {
  const currentState = stateManager.getState()
  stateManager.setState({
    ...currentState,
    pagination: {
      ...currentState.pagination,
      currentPage: 1
    }
  })
  syncToURL()
  loadTableData()
}

/**
 * 重置搜索
 */
const handleReset = (): void => {
  const currentState = stateManager.getState()
  stateManager.setState({
    ...currentState,
    searchForm: {},
    sorts: [],
    hasManualSort: false,
    pagination: {
      ...currentState.pagination,
      currentPage: 1
    }
  })
  syncToURL()
  loadTableData()
}

// ==================== URL 同步 ====================

/**
 * 构建表格查询参数（分页、排序、搜索）
 */
const buildTableQueryParams = (): Record<string, string> => {
  const query: Record<string, string> = {}
  const currentState = stateManager.getState()
  
  // 分页参数
  query.page = String(currentState.pagination.currentPage)
  query.page_size = String(currentState.pagination.pageSize)
  
  // 排序参数
  const finalSorts = currentState.sorts.length > 0 
    ? currentState.sorts 
    : (currentState.hasManualSort ? [] : buildDefaultSorts())
  
  if (finalSorts.length > 0) {
    query.sorts = finalSorts.map((item: SortItem) => `${item.field}:${item.order}`).join(',')
  }
  
  // 搜索参数（response 字段）
  const responseFields = (props.functionDetail.response || []).filter((field: FieldConfig) => {
    const search = field.search
    return search && search !== '-' && search !== '' && search.trim() !== ''
  })
  const requestFields = Array.isArray(props.functionDetail.request) ? props.functionDetail.request : []
  const requestFieldCodes = new Set<string>()
  requestFields.forEach((field: FieldConfig) => {
    requestFieldCodes.add(field.code)
  })
  
  const responseFieldsForURL = responseFields.filter(
    (field: FieldConfig) => !requestFieldCodes.has(field.code)
  )
  Object.assign(query, buildURLSearchParams(searchForm.value, responseFieldsForURL))
  
  // 搜索参数（request 字段）
  requestFields.forEach((field: FieldConfig) => {
    const value = searchForm.value[field.code]
    
    // 早期返回：跳过空值
    if (value === null || value === undefined) {
      return
    }
    
    // 早期返回：跳过空数组
    if (Array.isArray(value) && value.length === 0) {
      return
    }
    
    // 早期返回：跳过空字符串
    if (typeof value === 'string' && value.trim() === '') {
      return
    }
    
    query[field.code] = Array.isArray(value) ? value.join(',') : String(value)
  })
  
  // 清理空值参数
  Object.keys(query).forEach(key => {
    const value = query[key]
    
    // 早期返回：保留有效值
    if (value && typeof value === 'string' && !value.endsWith(':') && value.trim() !== '') {
      return
    }
    
    // 删除空值或无效值
    delete query[key]
  })
  
  return query
}

/**
 * 保留 URL 中的现有参数（除了 table 相关的参数）
 * 这样可以保留 link 组件跳转时携带的参数（如 eq=topic_id:1, topic_id=4 等）
 * 使用早期返回优化条件判断
 */
const preserveExistingParams = (requestFieldCodes: Set<string>): Record<string, string> => {
  const newQuery: Record<string, string> = {}
  const tableParamKeys = TABLE_PARAM_KEYS
  const searchParamKeys = SEARCH_PARAM_KEYS
  
  // 🔥 检查是否是 link 跳转（通过 _link_type 参数）
  // link 跳转时，URL 中的参数是用户明确指定的（来自 link 值），应该全部保留
  const isLinkNav = isLinkNavigation(route.query as Record<string, any>)
  
  // 先保留所有非 table 相关的参数（包括 link 跳转携带的参数）
  Object.keys(route.query).forEach(key => {
    const value = route.query[key]
    
    // 早期返回：跳过空值
    if (value === null || value === undefined) {
      return
    }

    // 保留以 _ 开头的参数（前端状态参数），但清除 _link_type（临时参数）
    if (key.startsWith('_')) {
      if (key !== LINK_TYPE_QUERY_KEY) {
        newQuery[key] = String(value)
      }
      return
    }
    
    // 🔥 搜索参数处理：
    // - link 跳转时：保留所有搜索参数（因为这是用户明确指定的）
    // - 非 link 跳转时：不保留搜索参数，搜索参数完全由当前函数的 searchForm 决定
    //   这样当用户删除搜索选项时，URL 中的搜索参数会被清除
    if (searchParamKeys.includes(key as any)) {
      if (isLinkNav) {
        // link 跳转：保留搜索参数
        newQuery[key] = String(value)
      }
      // 非 link 跳转：不保留搜索参数，让 buildTableQueryParams 根据 searchForm 重新构建
      return
    }
    
    // 保留不在 tableParamKeys 和 searchParamKeys 中的参数（这些可能是 link 跳转携带的参数，如 topic_id=4）
    if (!tableParamKeys.includes(key as any) && !requestFieldCodes.has(key)) {
      newQuery[key] = String(value)
    }
  })
  
  return newQuery
}

/**
 * 同步状态到 URL
 * 🔥 重要：URL 参数必须和接口请求参数完全对齐
 * URL 中的参数 = 接口请求的参数（包括分页、排序、搜索等）
 * 
 * 🔥 阶段2：改为事件驱动，通过 RouteManager 统一处理路由更新
 */
const syncToURL = (): void => {
  // 🔥 检查当前函数类型，如果是 form 函数，不应该调用 syncToURL
  // 这可以防止路由切换时，form 函数的 URL 被添加 table 参数
  if (props.functionDetail.template_type !== TEMPLATE_TYPE.TABLE) {
    return
  }
  
  // 构建表格查询参数
  const query = buildTableQueryParams()
  
  // 🔥 检查当前 URL 是否有查询参数
  // 如果 URL 没有查询参数（刚切换函数），不应该保留任何旧参数
  const hasQueryParams = Object.keys(route.query).length > 0
    const isLinkNav = isLinkNavigation(route.query as Record<string, any>)
  
  // 获取 request 字段代码集合（用于过滤）
  const requestFields = Array.isArray(props.functionDetail.request) ? props.functionDetail.request : []
  const requestFieldCodes = new Set<string>()
  requestFields.forEach((field: FieldConfig) => {
    requestFieldCodes.add(field.code)
  })
  
  // 🔥 如果 URL 没有查询参数（刚切换函数），直接使用新的查询参数，不保留任何旧参数
  let newQuery: Record<string, string | string[]>
    if (!hasQueryParams && !isLinkNav) {
    // 刚切换函数，URL 是空的，直接使用新的查询参数
    newQuery = { ...query }
  } else {
    // URL 有查询参数，保留现有参数并合并新的 table 参数
    newQuery = preserveExistingParams(requestFieldCodes)
    Object.assign(newQuery, query)
  }
  
  // 🔥 阶段2：改为发出事件，通过 RouteManager 统一处理路由更新
  
  eventBus.emit(RouteEvent.updateRequested, {
    query: newQuery,
    preserveParams: {
      table: true,        // 保留 table 参数（page, page_size, sorts）
      search: true,       // 保留搜索参数（eq, like, in 等）
      state: true,        // 保留状态参数（_ 开头）
      linkNavigation: isLinkNav  // 如果是 link 跳转，保留所有参数
    },
    source: RouteSource.TABLE_SYNC
  })
}

// 🔥 restoreFromURL 已移至 useTableInitialization composable

// ==================== 数据加载 ====================

// 🔥 组件挂载状态（用于防止卸载后继续加载数据）
const isMounted = ref(false)

/**
 * 加载表格数据
 */
const loadTableData = async (): Promise<void> => {
  const functionId = props.functionDetail.id
  const router = props.functionDetail.router
  
  // 🔥 检查组件是否还在挂载状态，如果已卸载，不加载数据
  if (!isMounted.value) {
    return
  }
  
  // 构建搜索参数
  const searchParams: SearchParams = {}
  
  // 🔥 从 StateManager 获取状态
  const currentState = stateManager.getState()
  
  // 使用 Domain Service 构建搜索参数（遵循依赖倒置原则）
  const builtSearchParams = domainService.buildSearchParams(props.functionDetail, currentState.searchForm)
  Object.assign(searchParams, builtSearchParams)
  
  // 排序参数
  const finalSorts = currentState.sorts.length > 0 
    ? currentState.sorts 
    : (currentState.hasManualSort ? [] : buildDefaultSorts())
  
  const sortParams: SortParams | undefined = finalSorts.length > 0 ? {
    field: finalSorts[0].field,
    order: finalSorts[0].order
  } : undefined
  
  // 分页参数
  const pagination = {
    page: currentState.pagination.currentPage,
    pageSize: currentState.pagination.pageSize
  }
  
  // 🔥 再次检查组件是否还在挂载状态（可能在异步操作期间卸载了）
  if (!isMounted.value) {
    return
  }
  
  try {
  await applicationService.loadData(props.functionDetail, searchParams, sortParams, pagination)
  } catch (error: any) {
    // 🔥 处理错误：当 API 返回 code !== 0 时，显示错误消息
    // request.ts 的响应拦截器在 code !== 0 时会 reject，并创建错误对象
    // 错误对象包含 response 属性，其中包含完整的响应数据
    let errorMessage = '加载数据失败，请稍后重试'
    
    // 🔥 统一使用 msg 字段
    // 尝试从 error.response.data 中获取错误消息（request.ts 第 99-101 行）
    if (error?.response?.data) {
      const responseData = error.response.data
      errorMessage = responseData.msg || errorMessage
    } else if (error?.message) {
      // 如果错误对象本身有 message（request.ts 第 99 行创建的）
      errorMessage = error.message
    }
    
    ElMessage.error(errorMessage)
  }
}

// ==================== 其他方法 ====================

const getRowFieldValue = (row: TableRow, fieldCode: string): FieldValue => {
  const value = row[fieldCode]
  return value ? { raw: value, display: String(value), meta: {} } : { raw: null, display: '', meta: {} }
}

/**
 * 获取操作列宽度
 * 根据是否有删除回调和链接字段动态计算宽度
 */
const getActionColumnWidth = (): number => {
  let width = 60  // 基础宽度
  if (hasDeleteCallback.value) width += 60  // 删除按钮宽度
  
  // 只有 1 个链接时直接显示，超过 1 个时使用下拉菜单
  if (linkFields.value.length === 1) {
    width += 80  // 单个链接约 80px
  } else if (linkFields.value.length > 1) {
    width += 50  // 下拉菜单按钮宽度
  }
  
  return Math.min(Math.max(width, 140), 200)  // 最小 140px，最大 200px
}

/**
 * 获取链接文本（从链接值中提取）
 */
const getLinkText = (linkField: FieldConfig, rawValue: any): string => {
  const value = convertToFieldValue(rawValue, linkField)
  const url = value?.raw || ''
  if (!url) return linkField.name || '链接'
  
  // 解析 "[text]url" 格式
  const match = url.match(/^\[([^\]]+)\](.+)$/)
  if (match) {
    return match[1]  // 返回文本部分
  }
  
  // 如果没有文本，使用字段名称或配置的 text
  return linkField.widget?.config?.text || linkField.name || '链接'
}

/**
 * 处理链接点击（用于下拉菜单）
 */
const handleLinkClick = (fieldCode: string, row: any) => {
  const linkField = linkFields.value.find(f => f.code === fieldCode)
  if (!linkField) return
  
  // 获取链接值
  const value = convertToFieldValue(row[fieldCode], linkField)
  const raw = value?.raw || ''
  if (!raw) return
  
  // 解析 JSON 格式的链接值
  const parsedLink = parseLinkValue(raw)
  
  // 获取链接配置
  const linkConfig = linkField.widget?.config || {}
  const target = linkConfig.target || '_self'
  
  // 处理 URL，添加 /workspace 前缀
  const resolvedUrl = resolveWorkspaceUrl(parsedLink.url, router.currentRoute.value)
  
  // 判断是否是外链
  const isExternal = resolvedUrl.startsWith('http://') || resolvedUrl.startsWith('https://')
  
  // 处理跳转
  if (isExternal) {
    window.open(resolvedUrl, '_blank')
  } else {
    // 🔥 阶段3：改为事件驱动，通过 RouteManager 统一处理路由更新
    // 如果 link 值中有 type 信息，通过 query 参数传递
    const finalUrl = addLinkTypeToUrl(resolvedUrl, parsedLink.type)
    
    if (target === '_blank') {
      window.open(finalUrl, '_blank')
    } else {
      // 🔥 阶段3：改为事件驱动，通过 RouteManager 统一处理路由更新
      // 解析 URL，提取 path 和 query
      // 注意：finalUrl 可能是相对路径（如 /workspace/xxx?param=value）
      let path = finalUrl
      const query: Record<string, string> = {}
      
      // 检查是否有查询参数
      const queryIndex = finalUrl.indexOf('?')
      if (queryIndex >= 0) {
        path = finalUrl.substring(0, queryIndex)
        const queryString = finalUrl.substring(queryIndex + 1)
        const params = new URLSearchParams(queryString)
        params.forEach((value, key) => {
          query[key] = value
        })
      }
      
      // 🔥 发出路由更新请求事件
      eventBus.emit(RouteEvent.updateRequested, {
        path,
        query,
        replace: false,  // link 跳转使用 push，保留历史记录
        preserveParams: {
          linkNavigation: true  // link 跳转：保留所有参数
        },
        source: RouteSource.TABLE_LINK_CLICK
      })
    }
  }
}

const getColumnWidth = (field: FieldConfig): number => {
  if (field.widget?.type === WidgetType.TIMESTAMP) return 180
  if (field.widget?.type === WidgetType.TEXT_AREA) return 300
  return 150
}

const handleAdd = (): void => {
  createDialogVisible.value = true
  
  // 更新 URL 为 ?_tab=OnTableAddRow（用于分享和直接跳转）
  const query: Record<string, string | string[]> = {}
  // 保留现有参数
  Object.keys(route.query).forEach(key => {
    const value = route.query[key]
    if (value !== null && value !== undefined) {
      query[key] = Array.isArray(value) 
        ? value.filter(v => v !== null).map(v => String(v))
        : String(value)
    }
  })
  // 添加新增弹窗参数
  query._tab = 'OnTableAddRow'
  
  // 🔥 发出路由更新请求事件
  eventBus.emit(RouteEvent.updateRequested, {
    query,
    replace: true,
    preserveParams: {
      state: true  // 保留状态参数
    },
    source: RouteSource.TABLE_ADD_DIALOG_OPEN
  })
}

const handleCreateSubmit = async (data: Record<string, any>): Promise<void> => {
  // 🔥 安全修复：提交前再次检查权限，避免绕过 UI 权限检查
  const node = currentFunctionNode.value
  if (!node) {
    ElMessage.error('无法获取函数节点信息，无法验证权限')
    return
  }
  
  if (!hasPermission(node, TablePermissions.write)) {
    ElNotification.warning({
      title: '权限不足',
      message: '您没有新增该表格记录的权限',
      duration: 3000
    })
    // 跳转到权限申请页面
    const applyUrl = buildPermissionApplyURL(
      node.full_code_path || '',
      TablePermissions.write,
      props.functionDetail?.template_type
    )
    router.push(applyUrl)
    return
  }
  
  try {
    await applicationService.addRow(props.functionDetail, data)
    ElMessage.success('新增成功')
    createDialogVisible.value = false
    // 清理 URL 中的 _tab 参数
    handleCreateDialogClose()
  } catch (error: any) {
    // 🔥 统一使用 msg 字段
    // 根据 request.ts 的响应拦截器，当 code !== 0 时，会创建 Error 对象并附加 response
    const errorMessage = error?.response?.data?.msg || error?.message || '新增失败'
    ElMessage.error(errorMessage)
  }
}

// 导入相关状态
const importDialogVisible = ref(false)
const importFile = ref<File | null>(null)
const importData = ref<any[]>([])
const importErrors = ref<Array<{ index: number; field: string; error: string }>>([])
const importing = ref(false)
const downloadingTemplate = ref(false)

// 获取可编辑字段（用于导入）
const editableFields = computed(() => {
  return (props.functionDetail.response || []).filter((field: FieldConfig) => {
    // 过滤掉 null 或 undefined 的字段
    if (!field) {
      return false
    }
    // 排除 ID 字段
    if (field.widget?.type === 'id') {
      return false
    }
    
    // 检查是否是系统字段
    const isSystemField = field.code === 'created_at' || field.code === 'create_by' ||
                          field.code === 'updated_at' || field.code === 'updated_by'
    
    // 系统字段：即使 table_permission 是 'read'，也要包含（用于导入）
    if (isSystemField) {
      return true
    }
    
    // 非系统字段：只包含可编辑字段（table_permission 为空或 update）
    const tablePermission = (field as any).table_permission
    return !tablePermission || tablePermission === 'update'
  })
})

// 处理导入
function handleImport(): void {
  importDialogVisible.value = true
  importFile.value = null
  importData.value = []
  importErrors.value = []
}

// 下载模板
async function handleDownloadTemplate(): Promise<void> {
  if (!props.functionDetail.router) {
    ElMessage.error('无法获取函数路由，无法下载模板')
    return
  }
  
  downloadingTemplate.value = true
  try {
    const { download } = await import('@/utils/request')
    const fullCodePath = props.functionDetail.router.startsWith('/') ? props.functionDetail.router : `/${props.functionDetail.router}`
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
  handleParseExcelFile(rawFile)
}

// 解析 Excel 文件
async function handleParseExcelFile(file: File): Promise<void> {
  try {
    const result = await parseExcelFile(file, editableFields.value)
    
    // 获取当前用户名，用于设置系统字段默认值
    const { useAuthStore } = await import('@/stores/auth')
    const authStore = useAuthStore()
    const currentUsername = authStore.userName || ''
    const currentTimestamp = Date.now()
    
    // 为系统字段设置默认值（如果表格中没有值）
    const processedData = result.data.map((row: any) => {
      const processedRow = { ...row }
      
      // 处理创建时间：如果为空或无效，使用当前时间（毫秒时间戳）
      if (!processedRow.created_at || processedRow.created_at === null || processedRow.created_at === '') {
        processedRow.created_at = currentTimestamp
      }
      
      // 处理创建用户：如果为空或 me()/$me，使用当前用户名（兼容旧格式）
      if (!processedRow.create_by || processedRow.create_by === 'me()' || processedRow.create_by === '$me' || processedRow.create_by === null || processedRow.create_by === '') {
        processedRow.create_by = currentUsername
      }
      
      // 处理更新时间：如果为空或无效，使用当前时间（毫秒时间戳）
      if (!processedRow.updated_at || processedRow.updated_at === null || processedRow.updated_at === '') {
        processedRow.updated_at = currentTimestamp
      }
      
      // 处理更新用户：如果为空或 me()/$me，使用当前用户名（兼容旧格式）
      if (!processedRow.updated_by || processedRow.updated_by === 'me()' || processedRow.updated_by === '$me' || processedRow.updated_by === null || processedRow.updated_by === '') {
        processedRow.updated_by = currentUsername
      }
      
      return processedRow
    })
    
    importData.value = processedData
    importErrors.value = result.errors
    
    if (result.errors.length > 0) {
      ElMessage.warning(`解析完成，发现 ${result.errors.length} 个错误，请修正后重新导入`)
    } else {
      ElMessage.success(`解析完成，共 ${processedData.length} 条有效数据`)
    }
  } catch (error: any) {
    ElMessage.error(`解析 Excel 文件失败: ${error.message || '未知错误'}`)
  }
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
  
  if (!props.functionDetail.router) {
    ElMessage.error('无法获取函数路由，无法导入数据')
    return
  }
  
  importing.value = true
  try {
    const { post } = await import('@/utils/request')
    const { useAuthStore } = await import('@/stores/auth')
    const authStore = useAuthStore()
    const currentUsername = authStore.userName || ''
    
    // 在提交前处理系统字段，确保所有数据都是真实值
    const processedData = importData.value.map((row: any) => {
      const processedRow = { ...row }
      
      // 处理创建时间：如果为空或无效，使用当前时间（毫秒时间戳）
      if (!processedRow.created_at || processedRow.created_at === null || processedRow.created_at === '') {
        processedRow.created_at = Date.now()
      }
      
      // 处理创建用户：如果为空或 me()/$me，使用当前用户名（兼容旧格式）
      if (!processedRow.create_by || processedRow.create_by === 'me()' || processedRow.create_by === '$me' || processedRow.create_by === null || processedRow.create_by === '') {
        processedRow.create_by = currentUsername
      }
      
      // 处理更新时间：如果为空或无效，使用当前时间（毫秒时间戳）
      if (!processedRow.updated_at || processedRow.updated_at === null || processedRow.updated_at === '') {
        processedRow.updated_at = Date.now()
      }
      
      // 处理更新用户：如果为空或 me()/$me，使用当前用户名（兼容旧格式）
      if (!processedRow.updated_by || processedRow.updated_by === 'me()' || processedRow.updated_by === '$me' || processedRow.updated_by === null || processedRow.updated_by === '') {
        processedRow.updated_by = currentUsername
      }
      
      return processedRow
    })
    
    const fullCodePath = props.functionDetail.router.startsWith('/') ? props.functionDetail.router : `/${props.functionDetail.router}`
    const result = await post(`/workspace/api/v1/table/batch-create${fullCodePath}`, {
      data: processedData
    })
    
    // 响应拦截器已经处理了 code，成功时直接返回 data
    // result 就是 OnTableCreateInBatchesResp 的数据
    const successCount = result?.success_count || 0
    const failCount = result?.fail_count || 0
    
    if (failCount > 0) {
      ElMessage.warning(`导入完成：成功 ${successCount} 条，失败 ${failCount} 条`)
      // 显示失败详情
      if (result?.errors && result.errors.length > 0) {
        const errorMsg = result.errors.map((e: any) => `第 ${e.index + 1} 行: ${e.error || e}`).join('\n')
        ElMessage.error(`失败详情:\n${errorMsg}`)
      }
    } else {
      ElMessage.success(`成功导入 ${successCount} 条数据`)
    }
    
    // 关闭对话框
    importDialogVisible.value = false
    
    // 刷新表格数据
    await applicationService.loadData(props.functionDetail)
  } catch (error: any) {
    ElMessage.error(`导入失败: ${error.message || '未知错误'}`)
  } finally {
    importing.value = false
  }
}

// 关闭新增对话框时清理 URL 中的 _tab 参数
const handleCreateDialogClose = (): void => {
  const query = { ...route.query }
  if (query._tab === 'OnTableAddRow') {
    delete query._tab
    // 清理所有表单字段参数（保留其他参数如搜索、分页等）
    if (props.functionDetail?.response) {
      props.functionDetail.response.forEach((field: FieldConfig) => {
        if (query[field.code]) {
          delete query[field.code]
        }
      })
    }
    // 🔥 通过事件总线更新路由，统一管理
    eventBus.emit(RouteEvent.updateRequested, {
      query,
      replace: true,
      preserveParams: {
        table: true,  // 保留 table 参数（分页、排序等）
        search: true, // 保留搜索参数
        state: true   // 保留状态参数
      },
      source: RouteSource.TABLE_CREATE_DIALOG_CLOSE
    })
  }
}

const handleDetail = (row: TableRow): void => {
  // 🔥 获取当前表格数据和索引
  // 注意：TableStateManager 使用 data 字段存储表格数据，不是 tableData
  const tableData = stateManager.getData() || []
  const index = tableData.findIndex((r: any) => {
    // 尝试通过 id 字段匹配
    if (r.id && row.id && r.id === row.id) return true
    // 如果没有 id，尝试通过所有字段匹配
    return JSON.stringify(r) === JSON.stringify(row)
  })
  
  eventBus.emit('table:detail-row', { 
    row, 
    index: index >= 0 ? index : undefined,
    tableData: tableData.length > 0 ? tableData : undefined
  })
}

const handleDelete = async (row: TableRow): Promise<void> => {
  try {
    await ElMessageBox.confirm('确定要删除该行数据吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const id = row.id
    await applicationService.deleteRow(props.functionDetail, id)
    ElMessage.success('删除成功')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleSizeChange = (size: number): void => {
  const currentState = stateManager.getState()
  stateManager.setState({
    ...currentState,
    pagination: {
      ...currentState.pagination,
      pageSize: size,
      currentPage: 1
    }
  })
  syncToURL()
  loadTableData()
}

const handleCurrentChange = (page: number): void => {
  const currentState = stateManager.getState()
  stateManager.setState({
    ...currentState,
    pagination: {
      ...currentState.pagination,
      currentPage: page
    }
  })
  syncToURL()
  loadTableData()
}

// ==================== 回调判断 ====================

const hasAddCallback = computed(() => {
  return props.functionDetail.callbacks?.includes('OnTableAddRow') || false
})

const hasDeleteCallback = computed(() => {
  return props.functionDetail.callbacks?.includes('OnTableDeleteRows') || false
})

const hasUpdateCallback = computed(() => {
  return props.functionDetail.callbacks?.includes('OnTableUpdateRow') || false
})

// ⭐ 权限检查：获取当前函数节点的权限信息
const currentFunctionNode = computed(() => {
  return workspaceStateManager.getCurrentFunction()
})

// ⭐ 是否有新增权限
const canCreate = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return false
  return hasPermission(node, TablePermissions.write)
})

// ⭐ 是否有更新权限
const canUpdate = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return false
  return hasPermission(node, TablePermissions.update)
})

// ⭐ 是否有删除权限
const canDelete = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return false
  return hasPermission(node, TablePermissions.delete)
})

// ⭐ 权限错误状态
const permissionErrorStore = usePermissionErrorStore()
const permissionError = computed<PermissionInfo | null>(() => permissionErrorStore.currentError)

// ⭐ 为特定操作申请权限（PermissionDeniedView 组件已处理权限错误显示）
const handleApplyPermissionForAction = (action: string) => {
  const node = currentFunctionNode.value
  if (!node || !node.full_code_path) {
    ElMessage.warning('无法获取资源路径，无法申请权限')
    return
  }
  
  // 使用 buildPermissionApplyURL 构建 URL（传递 template_type 以便正确显示权限选项）
  const applyUrl = buildPermissionApplyURL(node.full_code_path, action, node.template_type)
  router.push(applyUrl)
}

// ==================== 生命周期 ====================

let unsubscribeDataLoaded: (() => void) | null = null
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeQueryChanged: (() => void) | null = null

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
  preloadUserInfoFromSearchForm // 🔥 时机 1：预加载搜索表单中的用户信息
})

onMounted(async () => {
  // ⭐ 清除之前的权限错误（切换函数时清除）
  permissionErrorStore.clearError()
  
  // 🔥 设置挂载状态
  isMounted.value = true
  
  // 🔥 阶段4：设置 URL 变化监听（监听 RouteEvent.queryChanged）
  setupQueryWatch()
  
  // 初始化表格（状态清空逻辑已在 initializeTable 中处理）
  await initializeTable()
  
  // 监听数据加载完成事件
  unsubscribeDataLoaded = eventBus.on(TableEvent.dataLoaded, async (payload: { data: TableRow[], pagination?: any }) => {
    // 🔥 检查组件是否还在挂载状态
    if (!isMounted.value) {
      return
    }
    
    // 🔥 通过 StateManager 更新分页信息，而不是直接写入 computed
    const currentState = stateManager.getState()
    stateManager.setState({
      ...currentState,
      pagination: {
        currentPage: payload.pagination?.current_page || currentState.pagination.currentPage,
        pageSize: payload.pagination?.page_size || currentState.pagination.pageSize,
        total: payload.pagination?.total_count || 0
      }
    })
  })
  
  // 🔥 移除用户信息预加载完成事件的监听
  // 预加载已经在 TableDomainService.loadData 中通过 preloadUserInfoCallback 完成了
  // 用户信息已在 store 缓存中，UserDisplay 组件会直接从 store 读取
  
  // 🔥 监听函数加载完成事件（Tab 切换时触发）
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, async (payload: { detail: FunctionDetail }) => {
    if (payload.detail.template_type === TEMPLATE_TYPE.TABLE && payload.detail.id === props.functionDetail.id) {
      // 🔥 Tab 切换时，重新初始化表格，确保界面刷新
      if (isMounted.value) {
        await initializeTable()
      }
    }
  })
  
  // 🔥 设置新增弹窗 URL 监听（监听 RouteEvent.queryChanged）
  setupAddDialogUrlWatch()
})

// 从 URL 恢复新增弹窗
const restoreAddDialogFromURL = (query: any): void => {
  const tabParam = query._tab as string
  
  // 检查是否存在 _tab=OnTableAddRow 参数
  if (tabParam === 'OnTableAddRow' && hasAddCallback.value && isMounted.value) {
    // 打开新增弹窗
    createDialogVisible.value = true
  } else if (tabParam !== 'OnTableAddRow' && createDialogVisible.value) {
    // 如果 _tab 参数被移除或改变，关闭弹窗
    createDialogVisible.value = false
  }
}

// 设置 URL 参数监听（用于分享链接和直接跳转）
// 🔥 阶段4：改为监听 RouteEvent.queryChanged 事件，而不是直接 watch route.query
// 这样可以避免程序触发的路由更新导致循环
const setupAddDialogUrlWatch = () => {
  // 🔥 初始化时检查 URL 参数（页面刷新场景）
  // 如果 URL 中已经有 _tab=OnTableAddRow，打开新增弹窗
  if (route.query._tab === 'OnTableAddRow') {
    nextTick(() => {
      restoreAddDialogFromURL(route.query)
    })
  }
  
  // 监听 URL 参数变化（浏览器前进/后退场景）
  unsubscribeQueryChanged = eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, oldQuery: any, source: string }) => {
    // 🔥 只处理用户操作（浏览器前进/后退）或外部变化，不处理程序触发的更新
    if (payload.source === 'router-change') {
      restoreAddDialogFromURL(payload.query)
    }
  })
}

onUnmounted(() => {
  const functionId = props.functionDetail.id
  const router = props.functionDetail.router
  
  // 🔥 设置卸载状态，防止继续加载数据
  isMounted.value = false
  
  if (unsubscribeDataLoaded) {
    unsubscribeDataLoaded()
  }
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeQueryChanged) {
    unsubscribeQueryChanged()
  }
})
</script>

<style scoped>
.table-view {
  padding: 20px;
  background: var(--el-bg-color);
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
  padding: 12px 0;
}

.toolbar-left {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-bar {
  margin-bottom: 20px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

/* 🔥 排序信息条样式 */
.sort-info-bar {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
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
  font-size: 14px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
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

.clear-all-sorts-btn {
  margin-left: auto;
  white-space: nowrap;
}

.search-form {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* 🔥 表格基础样式 */
:deep(.el-table) {
  background-color: var(--el-bg-color) !important;
  border: none !important;
  flex: 1;
  overflow: auto;
}

:deep(.el-table__inner-wrapper) {
  border: none !important;
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
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr.el-table__row--striped) {
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr.el-table__row--striped td) {
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr:hover > td) {
  background-color: var(--el-fill-color-light) !important;
}

:deep(.el-table__header th.el-table__cell) {
  background-color: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-weight: 600;
  border-top: none;
}

:deep(.el-table td.el-table__cell),
:deep(.el-table th.el-table__cell.is-leaf) {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.link-text {
  color: var(--el-color-primary);
  cursor: pointer;
  text-decoration: none;
  font-weight: 500;
  display: inline-block;
  padding: 2px 4px;
  border-radius: 4px;
}

.link-text:hover {
  text-decoration: underline;
  background-color: var(--el-color-primary-light-9);
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  justify-content: flex-end;
}

.action-link {
  margin: 0;
}

.more-links-btn {
  margin: 0;
}

.dropdown-link-content {
  display: flex;
  align-items: center;
  gap: 6px;
}

.link-icon {
  font-size: 14px;
}

.internal-icon {
  color: var(--el-color-primary);
}

.delete-btn {
  flex-shrink: 0;
  white-space: nowrap;
  min-width: fit-content;
}

/* 🔥 权限错误显示样式已移至 PermissionDeniedView 组件 */

/* 无权限按钮样式优化 */
.action-btn-no-permission {
  color: var(--el-text-color-secondary) !important;
  border-color: var(--el-border-color-light) !important;
  
  &:hover {
    color: var(--el-color-primary) !important;
    border-color: var(--el-color-primary-light-7) !important;
    background-color: var(--el-color-primary-light-9) !important;
  }
  
  .el-icon {
    margin-right: 4px;
  }
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
</style>

