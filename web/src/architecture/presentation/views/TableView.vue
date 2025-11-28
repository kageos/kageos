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
          <!-- 如果是 id 列或配置了 is_link，渲染为链接 -->
          <span 
            v-if="field.code === 'id' || field.is_link"
            class="link-text"
            @click.stop="handleDetail(row)"
          >
            <WidgetComponent
              :field="field"
              :value="getRowFieldValue(row, field.code)"
              mode="table-cell"
              :row-data="row"
            />
          </span>
          <!-- 否则正常渲染 -->
          <WidgetComponent
            v-else
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

    <!-- 编辑对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑"
      width="600px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="editFormData" label-width="100px">
        <el-form-item
          v-for="field in editFields"
          :key="field.code"
          :label="field.name"
        >
          <WidgetComponent
            :field="field"
            :value="getEditFieldValue(field.code)"
            @update:model-value="(v) => updateEditField(field.code, v)"
            mode="edit"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitEdit" :loading="editFormLoading">
            确认
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
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

// 编辑表单状态
const editDialogVisible = ref(false)
const editFormLoading = ref(false)
const currentEditRowId = ref<string | number | null>(null)
const editFormData = ref<Record<string, any>>({})

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

// 编辑字段：默认使用响应字段作为编辑字段
// 实际项目中可能需要从 FunctionDetail 中获取专门的 update_fields
const editFields = computed(() => {
  return responseFields.value
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
  currentEditRowId.value = row.id
  // 深拷贝行数据到编辑表单
  // 注意：这里简化处理，直接使用行数据作为 raw value
  const formData: Record<string, any> = {}
  // 确保响应式丢失，使用 JSON 序列化/反序列化进行深拷贝
  const rowClone = JSON.parse(JSON.stringify(row))
  for (const key in rowClone) {
    formData[key] = rowClone[key]
  }
  editFormData.value = formData
  editDialogVisible.value = true
}

const getEditFieldValue = (fieldCode: string): FieldValue => {
  const value = editFormData.value[fieldCode]
  // 尝试从 editFields 中找到对应的字段配置来获取更多元数据（如果有）
  // 这里简化处理
  return { 
    raw: value, 
    display: typeof value === 'object' ? JSON.stringify(value) : String(value ?? ''), 
    meta: {} 
  }
}

const updateEditField = (fieldCode: string, value: FieldValue): void => {
  editFormData.value[fieldCode] = value.raw
}

const submitEdit = async (): Promise<void> => {
  if (!currentEditRowId.value) return
  
  try {
    editFormLoading.value = true
    await applicationService.updateRow(props.functionDetail, currentEditRowId.value, editFormData.value)
    ElMessage.success('更新成功')
    editDialogVisible.value = false
  } catch (error: any) {
    console.error('更新失败:', error)
    const msg = error?.response?.data?.message || '更新失败'
    ElMessage.error(msg)
  } finally {
    editFormLoading.value = false
  }
}

const handleDetail = (row: TableRow): void => {
  eventBus.emit('table:detail-row', { row })
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
      console.error('删除失败:', error)
      ElMessage.error('删除失败')
    }
  }
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeDataLoaded: (() => void) | null = null
let unsubscribeEditRow: (() => void) | null = null

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

  // 监听从详情页发起的编辑事件
  unsubscribeEditRow = eventBus.on('table:edit-row', ({ row }: { row: TableRow }) => {
    handleEdit(row)
  })
})

onUnmounted(() => {
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeDataLoaded) {
    unsubscribeDataLoaded()
  }
  if (unsubscribeEditRow) {
    unsubscribeEditRow()
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
  --el-table-border: none; /* 移除所有边框变量 */
}

/* 移除外层边框 */
:deep(.el-table__inner-wrapper::before) {
  display: none;
}

:deep(.el-table__border-left-patch) {
  display: none;
}

/* 移除所有边框 */
:deep(.el-table--border) {
  border: none;
}

:deep(.el-table--border .el-table__cell) {
  border-right: none;
}

/* 仅保留行底部分隔线 */
:deep(.el-table td.el-table__cell),
:deep(.el-table th.el-table__cell.is-leaf) {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

:deep(.el-table__header th.el-table__cell) {
  background-color: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-weight: 600;
  border-top: none;
}

.link-text {
  color: var(--el-color-primary);
  cursor: pointer;
  text-decoration: none;
  font-weight: 500;
  /* 增加点击区域 */
  display: inline-block;
  padding: 2px 4px;
  border-radius: 4px;
}

.link-text:hover {
  text-decoration: underline;
  background-color: var(--el-color-primary-light-9);
}

.el-pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
</style>

