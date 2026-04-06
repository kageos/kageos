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
        <!-- 新增按钮：需要 table:write 权限，无权限时可点击跳转申请 -->
        <el-button 
          v-if="hasAddCallback" 
          :type="canCreate ? 'primary' : 'default'"
          :plain="!canCreate"
          @click="canCreate ? handleAdd() : handleApplyPermissionForAction(TablePermission.write)"
          :icon="Plus"
          class="action-btn"
          :class="{ 'action-btn-no-permission': !canCreate }"
        >
          <template v-if="!canCreate">
            <el-icon><Lock /></el-icon>
            新增（需{{ getPermissionShortName(TablePermission.write) }}）
          </template>
          <template v-else>新增</template>
        </el-button>
        <!-- 批量删除按钮：需要 table:delete 权限，无权限时可点击跳转申请 -->
        <el-button 
          v-if="hasDeleteCallback && !isBatchDeleteMode" 
          :type="canDelete ? 'danger' : 'default'"
          :plain="!canDelete"
          @click="canDelete ? enterBatchDeleteMode() : handleApplyPermissionForAction(TablePermission.delete)"
          :icon="canDelete ? Delete : Lock"
          class="action-btn"
          :class="{ 'action-btn-no-permission': !canDelete }"
        >
          {{ canDelete ? '批量删除' : `批量删除（需${getPermissionShortName(TablePermission.delete)}）` }}
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

    <!-- 搜索栏：科幻风折叠，默认收起 -->
    <div v-if="searchableFields.length > 0" class="search-bar-wrapper">
      <!-- 收起时：终端条样式 -->
      <div
        v-if="!searchBarExpanded"
        class="search-bar-collapsed sci-fi-panel"
        @click="searchBarExpanded = true"
      >
        <span class="sci-fi-accent-bar" />
        <span class="sci-fi-dot" />
        <span class="search-bar-toggle">
          <el-icon class="sci-fi-icon"><ArrowDown /></el-icon>
          <span class="sci-fi-label">展开搜索</span>
          <span v-if="activeSearchCount > 0" class="sci-fi-badge">
            {{ activeSearchCount }} 个筛选
          </span>
        </span>
      </div>
      <!-- 展开时：完整表单 + 收起控制 -->
      <div v-else class="search-bar sci-fi-panel sci-fi-panel-expanded">
        <span class="sci-fi-accent-bar" />
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
                  :search-type="field.search || ''"
                  :model-value="getSearchValue(field)"
                  :function-method="props.functionDetail.method || 'GET'"
                  :function-router="props.functionDetail.router"
                  @update:model-value="(value: any) => {
                    updateSearchValue(field, value, true)
                  }"
                />
              </el-form-item>
            </template>

            <div class="search-actions">
              <el-button type="primary" @click="handleSearch" class="sci-fi-btn-primary">
                <el-icon><Search /></el-icon>
                搜索
              </el-button>
              <el-button @click="handleReset" class="sci-fi-btn-secondary">
                <el-icon><Refresh /></el-icon>
                重置
              </el-button>
              <button
                type="button"
                class="sci-fi-fold-btn"
                @click.stop="searchBarExpanded = false"
              >
                <el-icon><ArrowUp /></el-icon>
                <span>收起</span>
              </button>
            </div>
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
            :value="getRowFieldValue(row, field)"
            mode="table-cell"
            :row-data="row"
          />
        </template>
      </el-table-column>

      <!-- 操作列：统一为「更多」下拉，所有操作（链接 / 更新 / 删除）均放入下拉 -->
      <el-table-column 
        v-if="hasDeleteCallback || hasUpdateCallback || linkFields.length > 0" 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
        class-name="action-column"
      >
        <template #default="{ row }">
          <el-dropdown
            trigger="click"
            placement="bottom-end"
            @command="(cmd: string) => handleActionCommand(cmd, row)"
          >
            <el-button link type="primary" size="small" class="action-more-btn">
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
                <!-- 更新：需要 table:update 权限 -->
                <el-dropdown-item
                  v-if="hasUpdateCallback"
                  :command="'update'"
                  :divided="linkFields.length > 0"
                >
                  <span class="dropdown-action-item">
                    <el-icon><component :is="canUpdate ? Edit : Lock" /></el-icon>
                    {{ canUpdate ? '更新' : `更新（需${getPermissionShortName(TablePermission.update)}）` }}
                  </span>
                </el-dropdown-item>
                <!-- 删除：需要 table:delete 权限 -->
                <el-dropdown-item
                  v-if="hasDeleteCallback"
                  :command="'delete'"
                  :divided="linkFields.length > 0 || hasUpdateCallback"
                >
                  <span class="dropdown-action-item delete-action-text">
                    <el-icon><component :is="canDelete ? Delete : Lock" /></el-icon>
                    {{ canDelete ? '删除' : `删除（需${getPermissionShortName(TablePermission.delete)}）` }}
                  </span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
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
      :router="props.functionDetail.router ?? ''"
      :method="props.functionDetail.method || 'POST'"
      @submit="handleCreateSubmit"
      @close="handleCreateDialogClose"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElIcon, ElTable, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElText, ElCard, ElSkeleton } from 'element-plus'
import { Search, Refresh, Delete, Plus, ArrowUp, ArrowDown, More, Right, Lock, Document, Key, Edit } from '@element-plus/icons-vue'
import { eventBus, TableEvent, WorkspaceEvent, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/utils/routeSource'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../../presentation/widgets/WidgetComponent.vue'
import SearchInput from '@/architecture/presentation/components/SearchInput.vue'
import FormDialog from '@/architecture/presentation/components/FormDialog.vue'
import { getSortableConfig } from '@/utils/fieldSort'
import { WidgetType } from '@/core/constants/widget'
import { useTableInitialization } from '../composables/useTableInitialization'
import { convertToFieldValue } from '@/utils/field'
import { resolveWorkspaceUrl } from '@/utils/route'
import { parseLinkValue, addLinkTypeToUrl, isLinkNavigation } from '@/utils/linkNavigation'
import LinkWidget from '@/architecture/presentation/widgets/LinkWidget.vue'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { useUserInfoStore } from '@/stores/userInfo'
import { useDepartmentInfoStore } from '@/stores/departmentInfo'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import type { TableRow, SortItem } from '../../domain/services/TableDomainService'
import type { UserInfo } from '@/types'
import { hasPermission, TablePermission, getPermissionShortName } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import type { PermissionInfo } from '@/utils/permission'
import PermissionDeniedView from '../components/PermissionDeniedView.vue'
import { buildNextTableSyncQuery } from './utils/tableViewURLRuntime'
import {
  buildTableLoadRequest,
  buildTableLoadingState,
  decideTableLoadGuard,
  getTableLoadErrorMessage
} from './utils/tableViewLoadRuntime'
import {
  buildBatchDeleteIds,
  buildTablePermissionApplyURL,
  resolveTableActionCommand
} from './utils/tableViewActionRuntime'
import {
  buildTableAddDialogOpenRequest,
  buildTableCreateDialogCloseRequest,
  buildTableDetailRowPayload,
  buildTableLinkRouteRequest,
  resolveTableAddDialogVisibility
} from './utils/tableViewRouteRuntime'
import { resolveSearchFieldLayoutClass } from './utils/searchFieldLayout'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/core/utils/createFieldValue'

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

// ==================== 搜索区域折叠（持久化到 localStorage） ====================

const STORAGE_KEY_PREFIX = 'table-search-bar-expanded:'

/** 当前表格的存储 key（按 router 区分，不同表格独立记忆） */
const searchBarStorageKey = computed(() => {
  const router = props.functionDetail?.router
  return router ? `${STORAGE_KEY_PREFIX}${router}` : ''
})

/** 从本地恢复搜索区域展开状态（未存过则默认展开） */
const loadSearchBarExpanded = (): boolean => {
  const key = searchBarStorageKey.value
  if (!key) return true
  try {
    const raw = localStorage.getItem(key)
    if (raw === 'true') return true
    if (raw === 'false') return false
  } catch (_) {}
  return true
}

/** 搜索区域是否展开（默认展开；有可搜索字段时从 localStorage 恢复上次选择） */
const searchBarExpanded = ref(true)

/** 持久化搜索区域展开状态 */
const saveSearchBarExpanded = (value: boolean): void => {
  const key = searchBarStorageKey.value
  if (!key) return
  try {
    localStorage.setItem(key, String(value))
  } catch (_) {}
}

// 表格（router）变化时从本地恢复搜索区展开状态
watch(
  searchBarStorageKey,
  (key) => {
    if (key) searchBarExpanded.value = loadSearchBarExpanded()
  },
  { immediate: true }
)

// 用户展开/收起时写入本地
watch(searchBarExpanded, (v) => saveSearchBarExpanded(v))

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
  
  if (!hasPermission(node, TablePermission.delete)) {
    ElNotification.warning({
      title: '权限不足',
      message: '您没有删除该表格记录的权限',
      duration: 3000
    })
    const applyUrl = buildTablePermissionApplyURL(node, TablePermission.delete)
    if (applyUrl) {
      router.push(applyUrl)
    }
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
    const ids = buildBatchDeleteIds(selectedRows.value, idField.value?.code)

    if (ids.length === 0) {
      ElMessage.error('无法获取记录 ID，删除失败')
      return
    }

    // 调用批量删除 API
    const { tableDeleteRows } = await import('@/api/function')
    const functionRouter = props.functionDetail.router ?? ''
    if (!functionRouter) {
      ElMessage.error('函数路由缺失，无法执行批量删除')
      return
    }
    await tableDeleteRows(props.functionDetail.method || 'GET', functionRouter, ids)

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

// ==================== 用户信息预加载 ====================

const userInfoStore = useUserInfoStore()

// 🔥 移除 userInfoMap，UserDisplay 组件直接从 userInfoStore 读取（预加载已完成，store 中肯定有缓存）

/**
 * 🔥 预加载用户信息（时机 1：搜索表单中的用户信息）
 * 在数据加载前预加载，确保搜索表单渲染时已有用户信息
 */
const preloadUserInfoFromSearchForm = async (functionDetail: FunctionDetail, searchFormData: Record<string, any>): Promise<void> => {
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
const preloadUserInfoFromTableData = async (functionDetail: FunctionDetail, tableDataArray: TableRow[]): Promise<void> => {
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

// ==================== 部门信息预加载 ====================

const departmentInfoStore = useDepartmentInfoStore()

/**
 * 🔥 预加载部门信息（时机 1：搜索表单中的部门信息）
 * 在数据加载前预加载，确保搜索表单渲染时已有部门信息
 */
const preloadDepartmentInfoFromSearchForm = async (functionDetail: FunctionDetail, searchFormData: Record<string, any>): Promise<void> => {
  try {
    // 1. 识别所有部门字段（request + response）
    // 🔥 确保 request 和 response 是数组
    const requestFields = Array.isArray(functionDetail.request) ? functionDetail.request : []
    const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
    
    const departmentFields = [
      ...requestFields.filter(f => f.widget?.type === 'department' || f.widget?.type === 'departments'),
      ...responseFields.filter(f => f.widget?.type === 'department' || f.widget?.type === 'departments')
    ]
    
    if (departmentFields.length === 0) {
      return
    }
    
    // 2. 从搜索表单中收集所有部门路径
    const paths = new Set<string>()
    departmentFields.forEach(field => {
      const value = searchFormData[field.code]
      if (value) {
        // 处理数组（如 in=department:/dept1,/dept2）
        if (Array.isArray(value)) {
          value.forEach(v => {
            if (v) paths.add(String(v))
          })
        } else if (typeof value === 'string' && value.includes(',')) {
          // 处理逗号分隔的字符串（如 departments:/dept1,/dept2）
          value.split(',').forEach(v => {
            const trimmed = v.trim()
            if (trimmed) paths.add(trimmed)
          })
        } else {
          // 处理单个字符串（如 department=/dept1）
          paths.add(String(value))
        }
      }
    })
    
    if (paths.size === 0) {
      return
    }
    
    // 3. 批量查询部门信息（使用 batchGetDepartmentInfo，自动处理过期数据）
    // 🔥 预加载到 store 缓存即可，DepartmentDisplay 组件会直接从 store 读取
    await departmentInfoStore.batchGetDepartmentInfo([...paths])
  } catch (error) {
    console.error('[TableView] 预加载搜索表单中的部门信息失败', error)
  }
}

// 导出函数供 useTableInitialization 使用

/**
 * 🔥 预加载部门信息（时机 2：表格数据中的部门信息）
 * 在数据加载后预加载，确保表格渲染时已有部门信息
 */
const preloadDepartmentInfoFromTableData = async (functionDetail: FunctionDetail, tableDataArray: TableRow[]): Promise<void> => {
  try {
    // 1. 识别所有部门字段（response 字段）
    // 🔥 确保 response 是数组
    const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
    const departmentFields = responseFields.filter(f => f.widget?.type === 'department' || f.widget?.type === 'departments')
    
    if (departmentFields.length === 0 || !tableDataArray || tableDataArray.length === 0) {
      return
    }
    
    // 2. 从表格数据中收集所有部门路径
    const paths = new Set<string>()
    tableDataArray.forEach(row => {
      departmentFields.forEach(field => {
        const value = row[field.code]
        if (value !== null && value !== undefined && value !== '') {
          if (typeof value === 'string' && value.includes(',')) {
            // 处理逗号分隔的字符串（如 departments:/dept1,/dept2）
            value.split(',').forEach(v => {
              const trimmed = v.trim()
              if (trimmed) paths.add(trimmed)
            })
          } else {
            // 处理单个字符串（如 department=/dept1）
            paths.add(String(value))
          }
        }
      })
    })
    
    if (paths.size === 0) {
      return
    }
    
    // 3. 批量查询部门信息（使用 batchGetDepartmentInfo，自动处理过期数据）
    // 🔥 预加载到 store 缓存即可，DepartmentDisplay 组件会直接从 store 读取
    await departmentInfoStore.batchGetDepartmentInfo([...paths])
  } catch (error) {
    console.error('[TableView] 预加载表格数据中的部门信息失败', error)
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

/** 当前生效的筛选条件数量（收起时在「展开搜索」旁显示） */
const activeSearchCount = computed(() => {
  const form = stateManager.getState().searchForm
  if (!form || typeof form !== 'object') return 0
  return Object.keys(form).filter((k) => {
    const v = form[k]
    if (v === undefined || v === null) return false
    if (typeof v === 'string' && v.trim() === '') return false
    if (Array.isArray(v) && v.length === 0) return false
    return true
  }).length
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

const getSearchFieldLayoutClass = (field: FieldConfig): string => {
  return resolveSearchFieldLayoutClass(field)
}

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
      const existingSort = newSorts[existingIndex]
      if (existingSort) {
        existingSort.order = order
      }
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
  
  const isLinkNav = isLinkNavigation(route.query as Record<string, any>)
  const newQuery = buildNextTableSyncQuery({
    routeQuery: route.query as Record<string, any>,
    functionDetail: props.functionDetail,
    state: stateManager.getState(),
    buildDefaultSorts,
    isLinkNavigation: isLinkNav
  })
  
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

/** 🔥 打开详情时置为 true，下一次 loadTableData 被调用时直接 return，不请求接口、不出现骨架屏 */
const skipNextTableLoad = ref(false)

/**
 * 加载表格数据
 */
const loadTableData = async (): Promise<void> => {
  const guardResult = decideTableLoadGuard({
    isMounted: isMounted.value,
    skipNextTableLoad: skipNextTableLoad.value
  })

  if (guardResult === 'skip-unmounted') {
    return
  }

  if (guardResult === 'skip-next-load') {
    skipNextTableLoad.value = false
    const state = stateManager.getState()
    stateManager.setState(buildTableLoadingState(state, false))
    return
  }

  // 🔥 立即设置 loading，避免刷新时先出现「暂无数据」再出现 loading
  const stateBeforeLoad = stateManager.getState()
  stateManager.setState(buildTableLoadingState(stateBeforeLoad, true))

  // 🔥 从 StateManager 获取状态
  const currentState = stateManager.getState()
  const { searchParams, sortParams, pagination } = buildTableLoadRequest({
    functionDetail: props.functionDetail,
    state: currentState,
    buildDefaultSorts,
    buildSearchParams: (functionDetail, searchForm) =>
      domainService.buildSearchParams(functionDetail, searchForm)
  })
  
  // 🔥 再次检查组件是否还在挂载状态（可能在异步操作期间卸载了）
  if (!isMounted.value) {
    return
  }
  
  try {
    await applicationService.loadData(props.functionDetail, searchParams, sortParams, pagination)
  } catch (error: any) {
    ElMessage.error(getTableLoadErrorMessage(error))
  }
}

// ==================== 其他方法 ====================

const getRowFieldValue = (row: TableRow, field: FieldConfig): FieldValue => {
  const value = row[field.code]
  if (value === null || value === undefined || value === '') {
    return createEmptyRawFieldValue()
  }
  return createAutoFieldValue(value, field)
}

/**
 * 获取操作列宽度（统一「更多」下拉，固定宽度）
 */
const getActionColumnWidth = (): number => {
  return 90
}

/**
 * 操作列统一下拉命令分发：link:字段码 | update | delete
 */
const handleActionCommand = (command: string, row: TableRow): void => {
  const action = resolveTableActionCommand({
    command,
    canUpdate: canUpdate.value,
    canDelete: canDelete.value
  })

  if (action.type === 'link') {
    handleLinkClick(action.fieldCode, row)
    return
  }

  if (action.type === 'detail') {
    handleDetail(row, action.initialMode)
    return
  }

  if (action.type === 'delete') {
    handleDelete(row)
    return
  }

  if (action.type === 'apply-permission') {
    handleApplyPermissionForAction(action.action)
  }
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
      // 🔥 发出路由更新请求事件
      eventBus.emit(RouteEvent.updateRequested, {
        ...buildTableLinkRouteRequest(finalUrl),
        source: RouteSource.TABLE_LINK_CLICK
      })
    }
  }
}

const getColumnWidth = (field: FieldConfig): number => {
  if (field.widget?.type === WidgetType.TIMESTAMP) return 180
  if (field.widget?.type === WidgetType.TEXT_AREA) return 300
  // 部门字段需要更宽的列宽，以便横向显示多个部门标签
  if (field.widget?.type === 'department' || field.widget?.type === 'departments') return 300
  // 用户字段也需要更宽的列宽
  if (field.widget?.type === 'user' || field.widget?.type === 'users') return 250
  return 150
}

const handleAdd = (): void => {
  createDialogVisible.value = true
  
  // 🔥 发出路由更新请求事件
  eventBus.emit(RouteEvent.updateRequested, {
    ...buildTableAddDialogOpenRequest(route.query as Record<string, any>),
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
  
  if (!hasPermission(node, TablePermission.write)) {
    ElNotification.warning({
      title: '权限不足',
      message: '您没有新增该表格记录的权限',
      duration: 3000
    })
    const applyUrl = buildTablePermissionApplyURL(node, TablePermission.write)
    if (applyUrl) {
      router.push(applyUrl)
    }
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

// 关闭新增对话框时清理 URL 中的 _tab 参数
const handleCreateDialogClose = (): void => {
  const request = buildTableCreateDialogCloseRequest({
    routeQuery: route.query as Record<string, any>,
    responseFieldCodes: Array.isArray(props.functionDetail?.response)
      ? props.functionDetail.response.map((field: FieldConfig) => field.code)
      : []
  })

  if (request) {
    // 🔥 通过事件总线更新路由，统一管理
    eventBus.emit(RouteEvent.updateRequested, {
      ...request,
      source: RouteSource.TABLE_CREATE_DIALOG_CLOSE
    })
  }
}

/**
 * 打开详情抽屉
 * @param row 行数据
 * @param initialMode 初始模式：'read' 仅查看（如点击 #id），'edit' 直接编辑（如点击「更新」）
 */
const handleDetail = (row: TableRow, initialMode: 'read' | 'edit' = 'read'): void => {
  // 🔥 标记「接下来一次」表格加载跳过，避免打开详情时 URL 变化触发的 loadTableData 再次请求和骨架屏
  skipNextTableLoad.value = true

  const tableData = stateManager.getState().data || []

  eventBus.emit('table:detail-row', buildTableDetailRowPayload({
    row,
    tableData,
    initialMode
  }))
}

/** 行点击：整行可点击进入详情，排除操作列、复选框等 */
const handleRowClick = (row: TableRow, _column: any, event: Event): void => {
  const target = event.target as HTMLElement
  if (target?.closest?.('.action-column, .el-dropdown, .el-checkbox')) return
  handleDetail(row)
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
  return workspaceStateManager.getState().currentFunction
})

// ⭐ 是否有新增权限
const canCreate = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return false
  return hasPermission(node, TablePermission.write)
})

// ⭐ 是否有更新权限
const canUpdate = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return false
  return hasPermission(node, TablePermission.update)
})

// ⭐ 是否有删除权限
const canDelete = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return false
  return hasPermission(node, TablePermission.delete)
})

// ⭐ 权限错误状态
const permissionErrorStore = usePermissionErrorStore()
const permissionError = computed<PermissionInfo | null>(() => permissionErrorStore.currentError)

// ⭐ 为特定操作申请权限（PermissionDeniedView 组件已处理权限错误显示）
const handleApplyPermissionForAction = (action: string) => {
  const node = currentFunctionNode.value
  const applyUrl = buildTablePermissionApplyURL(node, action)
  if (!applyUrl) {
    ElMessage.warning('无法获取资源路径，无法申请权限')
    return
  }

  router.push(applyUrl)
}

// ==================== 生命周期 ====================

let unsubscribeDataLoaded: (() => void) | null = null
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeTableQueryChanged: (() => void) | null = null
let unsubscribeAddDialogQueryChanged: (() => void) | null = null

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

onMounted(async () => {
  // ⭐ 清除之前的权限错误（切换函数时清除）
  permissionErrorStore.clearError()
  
  // 🔥 设置挂载状态
  isMounted.value = true
  
  // 🔥 阶段4：设置 URL 变化监听（监听 RouteEvent.queryChanged）
  unsubscribeTableQueryChanged = setupQueryWatch()
  
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
  createDialogVisible.value = resolveTableAddDialogVisibility({
    query,
    hasAddCallback: hasAddCallback.value,
    isMounted: isMounted.value,
    currentVisible: createDialogVisible.value
  })
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
  unsubscribeAddDialogQueryChanged = eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, oldQuery: any, source: string }) => {
    // 🔥 只处理用户操作（浏览器前进/后退）或外部变化，不处理程序触发的更新
    if (payload.source === 'router-change') {
      restoreAddDialogFromURL(payload.query)
    }
  })
}

onUnmounted(() => {
  // 🔥 设置卸载状态，防止继续加载数据
  isMounted.value = false
  
  if (unsubscribeDataLoaded) {
    unsubscribeDataLoaded()
  }
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeTableQueryChanged) {
    unsubscribeTableQueryChanged()
  }
  if (unsubscribeAddDialogQueryChanged) {
    unsubscribeAddDialogQueryChanged()
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

/* ========== 科幻风搜索栏折叠 ========== */
.search-bar-wrapper {
  margin-bottom: 16px;
}

/* 共用：带左边高亮条与微光的面板 */
.sci-fi-panel {
  position: relative;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  transition: box-shadow 0.25s ease, border-color 0.25s ease;
}


/* 左侧高亮条（电源/状态条） */
.sci-fi-accent-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: linear-gradient(180deg, rgba(0, 212, 255, 0.9), rgba(0, 212, 255, 0.4));
  box-shadow: 0 0 10px rgba(0, 212, 255, 0.5);
}

/* 收起态：终端条，可点击整行 */
.search-bar-collapsed {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px 10px 20px;
  cursor: pointer;
  min-height: 40px;
  transition: background 0.2s ease;
}


/* 状态小点（可选呼吸感） */
.sci-fi-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(0, 212, 255, 0.8);
  box-shadow: 0 0 8px rgba(0, 212, 255, 0.6);
  flex-shrink: 0;
  animation: sci-fi-pulse 2s ease-in-out infinite;
}

@keyframes sci-fi-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 8px rgba(0, 212, 255, 0.6); }
  50% { opacity: 0.6; box-shadow: 0 0 4px rgba(0, 212, 255, 0.4); }
}

.search-bar-collapsed .search-bar-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.sci-fi-icon {
  color: rgba(0, 212, 255, 0.9);
  font-size: 16px;
  transition: transform 0.2s ease;
}

.search-bar-collapsed:hover .sci-fi-icon {
  transform: translateY(1px);
  color: rgb(0, 212, 255);
}

.sci-fi-label {
  letter-spacing: 0.5px;
  font-weight: 500;
}

.sci-fi-badge {
  margin-left: 8px;
  padding: 2px 8px;
  font-size: 12px;
  border-radius: 10px;
  border: 1px solid rgba(0, 212, 255, 0.5);
  background: rgba(0, 212, 255, 0.08);
  color: rgba(0, 212, 255, 0.95);
  letter-spacing: 0.3px;
}

/* 展开态面板 */
.sci-fi-panel-expanded {
  background: var(--el-bg-color);
}

.sci-fi-panel-expanded .sci-fi-accent-bar {
  display: none;
}

.search-bar-inner {
  padding: 20px 20px 20px 24px;
}

.search-bar {
  margin-bottom: 0;
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
  color: var(--el-text-color-regular);
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

.search-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  justify-content: flex-end;
  grid-column: 1 / -1;
  padding-top: 4px;
  width: 100%;
  min-width: 0;
}

/* 收起按钮：终端风格 */
.sci-fi-fold-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  margin-left: 4px;
  font-size: 13px;
  color: rgba(0, 212, 255, 0.9);
  background: transparent;
  border: 1px solid rgba(0, 212, 255, 0.4);
  border-radius: 6px;
  cursor: pointer;
  letter-spacing: 0.3px;
  transition: background 0.2s ease, border-color 0.2s ease, color 0.2s ease;
}

.sci-fi-fold-btn:hover {
  background: rgba(0, 212, 255, 0.1);
  border-color: rgba(0, 212, 255, 0.7);
  color: rgb(0, 212, 255);
}

.sci-fi-btn-primary {
  border-color: var(--el-color-primary);
}

.sci-fi-btn-secondary {
  border-color: var(--el-border-color);
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

/* 表格骨架屏（加载中） */
.table-skeleton-wrap {
  min-height: 320px;
  padding: 16px 0;
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

  .search-actions {
    flex-wrap: wrap;
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

/* 整行可点击进入详情 */
:deep(.table-row-clickable .el-table__body tr) {
  cursor: pointer;
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

.action-more-btn {
  margin: 0;
  padding: 0 4px;
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

</style>
