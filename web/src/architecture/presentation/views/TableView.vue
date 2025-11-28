<!--
  TableView - 表格视图
  🔥 新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
-->

<template>
  <div class="table-view">
    <!-- 工具栏 -->
    <div class="toolbar">
      <el-button
        v-if="hasAddCallback"
        type="primary"
        @click="handleAdd"
      >
        新增
      </el-button>
    </div>

    <!-- 搜索栏 -->
    <div v-if="searchableFields.length > 0" class="search-section">
      <el-form :model="searchForm" inline>
        <el-form-item
          v-for="field in searchableFields"
          :key="field.code"
          :label="field.name"
        >
          <WidgetComponent
            :field="field"
            :value="getSearchFieldValue(field.code)"
            @update:model-value="(v) => updateSearchField(field.code, v)"
            mode="search"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 表格 -->
    <el-table
      :data="tableData"
      v-loading="loading"
      style="width: 100%"
      border
      @sort-change="handleSortChange"
    >
      <el-table-column
        v-for="field in visibleFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name"
        min-width="150"
        :sortable="field.search ? 'custom' : false"
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
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="hasUpdateCallback"
            type="primary"
            size="small"
            @click="handleEdit(row)"
          >
            编辑
          </el-button>
          <el-button
            v-if="hasDeleteCallback"
            type="danger"
            size="small"
            @click="handleDelete(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <el-pagination
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="[10, 20, 50, 100]"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { eventBus, TableEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import type { TableRow, SearchParams, SortParams } from '../../domain/services/TableDomainService'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getTableStateManager()
const domainService = serviceFactory.getTableDomainService()
const applicationService = serviceFactory.getTableApplicationService()

// 从状态管理器获取状态
const tableData = computed(() => domainService.getData())
const loading = computed(() => domainService.isLoading())
const pagination = computed(() => domainService.getPagination())

const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 搜索表单
const searchForm = ref<Record<string, any>>({})

// 字段配置
const responseFields = computed(() => (props.functionDetail.response || []) as FieldConfig[])
const searchableFields = computed(() => {
  return responseFields.value.filter(field => field.search)
})
const visibleFields = computed(() => {
  return responseFields.value.filter(field => {
    const permission = field.table_permission || ''
    return permission === '' || permission === 'read'
  })
})

// 回调判断
const hasAddCallback = computed(() => {
  return props.functionDetail.callbacks?.includes('OnTableAddRow') || false
})
const hasUpdateCallback = computed(() => {
  return props.functionDetail.callbacks?.includes('OnTableUpdateRow') || false
})
const hasDeleteCallback = computed(() => {
  return props.functionDetail.callbacks?.includes('OnTableDeleteRows') || false
})

// 方法
const getSearchFieldValue = (fieldCode: string): FieldValue => {
  const value = searchForm.value[fieldCode]
  return value ? { raw: value, display: String(value), meta: {} } : { raw: null, display: '', meta: {} }
}

const updateSearchField = (fieldCode: string, value: FieldValue): void => {
  if (value) {
    searchForm.value[fieldCode] = value.raw
  } else {
    searchForm.value[fieldCode] = null
  }
}

const getRowFieldValue = (row: TableRow, fieldCode: string): FieldValue => {
  const value = row[fieldCode]
  return value ? { raw: value, display: String(value), meta: {} } : { raw: null, display: '', meta: {} }
}

const handleSearch = (): void => {
  const searchParams: SearchParams = { ...searchForm.value }
  applicationService.updateSearchParams(searchParams)
  applicationService.loadData(props.functionDetail, searchParams)
}

const handleReset = (): void => {
  searchForm.value = {}
  applicationService.updateSearchParams({})
  applicationService.loadData(props.functionDetail)
}

const handleSortChange = ({ prop, order }: { prop?: string, order?: string }): void => {
  if (prop && order) {
    const sortParams: SortParams = {
      field: prop,
      order: order === 'ascending' ? 'asc' : 'desc'
    }
    applicationService.updateSortParams(sortParams)
    applicationService.loadData(props.functionDetail, undefined, sortParams)
  }
}

const handleSizeChange = (size: number): void => {
  pageSize.value = size
  applicationService.updatePagination(currentPage.value, size)
  applicationService.loadData(props.functionDetail, undefined, undefined, { page: currentPage.value, pageSize: size })
}

const handleCurrentChange = (page: number): void => {
  currentPage.value = page
  applicationService.updatePagination(page, pageSize.value)
  applicationService.loadData(props.functionDetail, undefined, undefined, { page, pageSize: pageSize.value })
}

const handleAdd = async (): Promise<void> => {
  // TODO: 打开新增对话框
  console.log('新增')
}

const handleEdit = async (row: TableRow): Promise<void> => {
  // TODO: 打开编辑对话框
  console.log('编辑', row)
}

const handleDelete = async (row: TableRow): Promise<void> => {
  // TODO: 确认删除
  const id = row.id
  await applicationService.deleteRow(props.functionDetail, id)
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeDataLoaded: (() => void) | null = null

onMounted(() => {
  // 初始加载数据
  if (props.functionDetail) {
    applicationService.loadData(props.functionDetail)
  }

  // 监听函数加载完成事件
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, (payload: { detail: FunctionDetail }) => {
    if (payload.detail.template_type === 'table') {
      // Application Service 会自动处理
    }
  })

  // 监听数据加载完成事件
  unsubscribeDataLoaded = eventBus.on(TableEvent.dataLoaded, (payload: { data: TableRow[], pagination?: any }) => {
    total.value = payload.pagination?.total_count || 0
    currentPage.value = payload.pagination?.current_page || 1
    pageSize.value = payload.pagination?.page_size || 20
  })
})

onUnmounted(() => {
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeDataLoaded) {
    unsubscribeDataLoaded()
  }
})
</script>

<style scoped>
.table-view {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.toolbar {
  margin-bottom: 20px;
}

.search-section {
  margin-bottom: 20px;
  padding: 20px;
  background: var(--el-bg-color-page);
  border-radius: 4px;
}

/* 🔥 修复表格右边框 */
.el-table {
  flex: 1;
  overflow: auto;
  --el-table-border-color: var(--el-border-color-lighter);
}

:deep(.el-table__inner-wrapper::before) {
  display: none; /* 移除底部边框 */
}

:deep(.el-table--border) {
  border-right: none;
}

:deep(.el-table--border .el-table__cell) {
  border-right: none;
}

:deep(.el-table__header th.el-table__cell) {
  background-color: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.el-pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
</style>

