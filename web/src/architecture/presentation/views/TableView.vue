<!--
  TableView - 表格视图
  新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
  - URL 参数同步（搜索、排序、分页）
  - 排序信息条显示
-->

<template>
  <div class="table-view">
    <!-- 工具栏 -->
    <div class="toolbar" v-if="hasAddCallback">
      <el-button type="primary" @click="handleAdd" :icon="Plus">
        新增
      </el-button>
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
                const isClearing = (value === null || value === '') && 
                                   searchForm && 
                                   searchForm[field.code] !== undefined
                updateSearchValue(field, value, isClearing)
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
      v-loading="loading"
      :data="tableData"
      :stripe="false"
      style="width: 100%"
      class="table-with-fixed-column"
      @sort-change="handleSortChange"
    >
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
            
            <!-- 删除按钮 -->
            <el-button 
              v-if="hasDeleteCallback"
              link 
              type="danger" 
              size="small"
              class="delete-btn"
              @click.stop="handleDelete(row)"
            >
              <el-icon><Delete /></el-icon>
              删除
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
      @submit="handleCreateSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElIcon } from 'element-plus'
import { Search, Refresh, Delete, Plus, ArrowUp, ArrowDown, More, Right } from '@element-plus/icons-vue'
import { eventBus, TableEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
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
import LinkWidget from '@/core/widgets-v2/components/LinkWidget.vue'
import { TABLE_PARAM_KEYS, SEARCH_PARAM_KEYS } from '@/utils/urlParams'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import type { TableRow, SearchParams, SortParams, SortItem } from '../../domain/services/TableDomainService'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

const route = useRoute()
const router = useRouter()

// 依赖注入
const stateManager = serviceFactory.getTableStateManager()
const domainService = serviceFactory.getTableDomainService()
const applicationService = serviceFactory.getTableApplicationService()

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

// 创建对话框
const createDialogVisible = ref(false)

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
  
  // 先保留所有非 table 相关的参数（包括 link 跳转携带的参数）
  Object.keys(route.query).forEach(key => {
    const value = route.query[key]
    
    // 早期返回：跳过空值
    if (value === null || value === undefined) {
      return
    }

    // 保留以 _ 开头的参数（前端状态参数）
    if (key.startsWith('_')) {
      newQuery[key] = String(value)
      return
    }
    
    // 🔥 跳过搜索参数：搜索参数的作用域是函数级别的
    // 旧参数的作用域只能在那个函数，一旦切换函数，必须换成那个函数的搜索参数
    // 切换函数时，必须清除上一个函数的搜索参数，只使用当前函数的 searchForm 中的参数
    // 这样可以避免函数切换时保留上一个函数的搜索参数，防止状态污染
    if (searchParamKeys.includes(key)) {
      // 搜索参数完全由当前函数的 searchForm 决定，不从 URL 中保留旧参数
      // 搜索参数会在 buildTableQueryParams 中根据当前函数的 searchForm 重新构建
      return
    }
    
    // 保留不在 tableParamKeys 和 searchParamKeys 中的参数（这些可能是 link 跳转携带的参数，如 topic_id=4）
    if (!tableParamKeys.includes(key) && !requestFieldCodes.has(key)) {
      newQuery[key] = String(value)
    }
  })
  
  return newQuery
}

/**
 * 同步状态到 URL
 * 🔥 重要：URL 参数必须和接口请求参数完全对齐
 * URL 中的参数 = 接口请求的参数（包括分页、排序、搜索等）
 */
const syncToURL = (): void => {
  // 🔥 检查当前函数类型，如果是 form 函数，不应该调用 syncToURL
  // 这可以防止路由切换时，form 函数的 URL 被添加 table 参数
  if (props.functionDetail.template_type !== TEMPLATE_TYPE.TABLE) {
    return
  }
  
  // 构建表格查询参数
  const query = buildTableQueryParams()
  
  // 获取 request 字段代码集合（用于过滤）
  const requestFields = Array.isArray(props.functionDetail.request) ? props.functionDetail.request : []
  const requestFieldCodes = new Set<string>()
  requestFields.forEach((field: FieldConfig) => {
    requestFieldCodes.add(field.code)
  })
  
  // 保留现有参数并合并新的 table 参数
  const newQuery = preserveExistingParams(requestFieldCodes)
  Object.assign(newQuery, query)
  
  // 🔥 确保路由更新：如果路径相同，使用 replace 更新 query；如果路径不同，使用 replace 更新 path 和 query
  // 这样可以确保 URL 刷新，即使路径相同也能触发路由变化
  const currentPath = route.path
  router.replace({ 
    path: currentPath, 
    query: newQuery 
  }).catch(() => {})
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
  
  await applicationService.loadData(props.functionDetail, searchParams, sortParams, pagination)
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
  const url = value?.raw || ''
  if (!url) return
  
  // 解析 "[text]url" 格式
  const match = url.match(/^\[([^\]]+)\](.+)$/)
  const actualUrl = match ? match[2] : url
  
  // 获取链接配置
  const linkConfig = linkField.widget?.config || {}
  const target = linkConfig.target || '_self'
  
  // 处理 URL，添加 /workspace 前缀
  const resolvedUrl = resolveWorkspaceUrl(actualUrl, router.currentRoute.value)
  
  // 判断是否是外链
  const isExternal = resolvedUrl.startsWith('http://') || resolvedUrl.startsWith('https://')
  
  // 处理跳转
  if (isExternal) {
    window.open(resolvedUrl, '_blank')
  } else {
    if (target === '_blank') {
      window.open(resolvedUrl, '_blank')
    } else {
      router.push(resolvedUrl)
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
}

const handleCreateSubmit = async (data: Record<string, any>): Promise<void> => {
  try {
    await applicationService.addRow(props.functionDetail, data)
    ElMessage.success('新增成功')
    createDialogVisible.value = false
  } catch (error: any) {
    const msg = error?.response?.data?.message || '新增失败'
    ElMessage.error(msg)
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

// ==================== 生命周期 ====================

let unsubscribeDataLoaded: (() => void) | null = null

// 🔥 使用 composable 统一管理初始化逻辑
const { initializeTable } = useTableInitialization({
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
  isMounted // 🔥 传递挂载状态，用于防止卸载后继续加载数据
})

onMounted(async () => {
  const functionId = props.functionDetail.id
  const router = props.functionDetail.router
  
  // 🔥 设置挂载状态
  isMounted.value = true
  
  // 初始化表格
  await initializeTable()
  
  // 监听数据加载完成事件
  unsubscribeDataLoaded = eventBus.on(TableEvent.dataLoaded, (payload: { data: TableRow[], pagination?: any }) => {
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
})

onUnmounted(() => {
  const functionId = props.functionDetail.id
  const router = props.functionDetail.router
  
  // 🔥 设置卸载状态，防止继续加载数据
  isMounted.value = false
  
  if (unsubscribeDataLoaded) {
    unsubscribeDataLoaded()
  }
})
</script>

<style scoped>
.table-view {
  padding: 20px;
  background: var(--el-bg-color);
  position: relative;
  z-index: 1;
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
</style>
