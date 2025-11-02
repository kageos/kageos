<template>
  <div class="table-renderer">
    <!-- 工具栏 -->
    <div class="toolbar" v-if="hasAddCallback">
      <el-button type="primary" @click="handleAdd" :icon="Plus">
        新增
      </el-button>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <template v-for="field in searchableFields" :key="field.code">
          <!-- 🔥 通过 Widget 渲染搜索输入（组件自治） -->
          <el-form-item :label="field.name">
            <SearchInput
              :field="field"
              :search-type="field.search"
              :model-value="getSearchValue(field)"
              @update:model-value="(value) => updateSearchValue(field, value)"
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

    <!-- 表格 -->
    <el-table
      v-loading="loading"
      :data="tableData"
      border
      style="width: 100%"
      @sort-change="handleSortChange"
    >
      <el-table-column
        v-for="field in visibleFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name"
        :sortable="field.search ? 'custom' : false"
        :min-width="getColumnWidth(field)"
        :class-name="isIdColumn(field) ? 'id-column' : ''"
      >
        <template #default="{ row, $index }">
          <!-- 🔥 ID 列：可点击查看详情 -->
          <span 
            v-if="isIdColumn(field)" 
            class="id-cell clickable"
            @click="handleShowDetail(row, $index)"
            :title="'点击查看详情'"
          >
            {{ row[field.code] }}
          </span>
          <!-- 时间戳列 -->
          <span v-else-if="field.widget.type === 'timestamp'">
            {{ formatTimestamp(row[field.code], field.widget.config.format) }}
          </span>
          <!-- 普通列 -->
          <span v-else>{{ row[field.code] }}</span>
        </template>
      </el-table-column>

      <el-table-column 
        v-if="hasUpdateCallback || hasDeleteCallback" 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
      >
        <template #default="{ row }">
          <el-button 
            v-if="hasUpdateCallback"
            link 
            type="primary" 
            size="small"
            @click="handleEdit(row)"
          >
            <el-icon><Edit /></el-icon>
            编辑
          </el-button>
          <el-button 
            v-if="hasDeleteCallback"
            link 
            type="danger" 
            size="small"
            @click="handleDelete(row)"
          >
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
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

    <!-- 新增/编辑对话框 -->
    <FormDialog
      v-model="dialogVisible"
      :title="dialogTitle"
      :fields="props.functionData.response"
      :mode="dialogMode"
      :initial-data="currentRow"
      @submit="handleDialogSubmit"
    />

    <!-- 🔥 详情抽屉 -->
    <el-drawer
      v-model="showDetailDrawer"
      title="记录详情"
      direction="rtl"
      size="600px"
      class="detail-drawer"
    >
      <template #header>
        <div class="drawer-header">
          <span class="drawer-title">记录详情</span>
          <div class="drawer-navigation" v-if="tableData.length > 1">
            <el-button
              size="small"
              :disabled="currentDetailIndex <= 0"
              @click="handleNavigate('prev')"
            >
              <el-icon><ArrowLeft /></el-icon>
              上一个
            </el-button>
            <span class="nav-info">{{ currentDetailIndex + 1 }} / {{ tableData.length }}</span>
            <el-button
              size="small"
              :disabled="currentDetailIndex >= tableData.length - 1"
              @click="handleNavigate('next')"
            >
              下一个
              <el-icon><ArrowRight /></el-icon>
            </el-button>
          </div>
        </div>
      </template>

      <div class="detail-content" v-if="currentDetailRow">
        <el-descriptions :column="1" border>
          <el-descriptions-item
            v-for="field in visibleFields"
            :key="field.code"
            :label="field.name"
          >
            <template v-if="field.widget.type === 'timestamp'">
              {{ formatTimestamp(currentDetailRow[field.code], field.widget.config.format) }}
            </template>
            <template v-else>
              {{ currentDetailRow[field.code] || '-' }}
            </template>
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, Refresh, Edit, Delete, Plus, ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { executeFunction, tableAddRow, tableUpdateRow, tableDeleteRows } from '@/api/function'
import FormDialog from './FormDialog.vue'
import SearchInput from './SearchInput.vue'
import type { Function as FunctionType, FieldConfig, SearchParams } from '@/types'

interface Props {
  functionData: FunctionType
}

const props = defineProps<Props>()

// 表格数据
const loading = ref(false)
const tableData = ref<any[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const sortField = ref('')
const sortOrder = ref('')

// 🔥 详情抽屉状态
const showDetailDrawer = ref(false)
const currentDetailRow = ref<any>(null)
const currentDetailIndex = ref(-1)

// 搜索表单
const searchForm = ref<Record<string, any>>({})

// 可搜索字段
const searchableFields = computed(() => {
  return props.functionData.response.filter(field => field.search)
})

// 可见字段（根据 table_permission 过滤）
const visibleFields = computed(() => {
  return props.functionData.response.filter(field => {
    const permission = field.table_permission
    // 🔥 列表中只显示：
    // - 空（全部权限）
    // - read（只读字段）
    // 不显示：
    // - create（只在新增表单显示）
    // - update（只在编辑表单显示）
    return !permission || permission === '' || permission === 'read'
  })
})

// 判断是否有新增回调
const hasAddCallback = computed(() => {
  const callbacks = props.functionData.callbacks || ''
  return callbacks.includes('OnTableAddRow')
})

// 判断是否有更新回调
const hasUpdateCallback = computed(() => {
  const callbacks = props.functionData.callbacks || ''
  return callbacks.includes('OnTableUpdateRow')
})

// 判断是否有删除回调
const hasDeleteCallback = computed(() => {
  const callbacks = props.functionData.callbacks || ''
  return callbacks.includes('OnTableDeleteRows')
})

// 对话框相关
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'update'>('create')
const dialogTitle = computed(() => dialogMode.value === 'create' ? '新增' : '编辑')
const currentRow = ref<Record<string, any>>({})

// 获取操作列宽度
const getActionColumnWidth = () => {
  let width = 80
  if (hasUpdateCallback.value) width += 60
  if (hasDeleteCallback.value) width += 60
  return width
}

// 格式化时间戳
const formatTimestamp = (timestamp: number, format = 'YYYY-MM-DD HH:mm:ss') => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  
  if (format.includes('HH:mm:ss')) {
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  }
  return `${year}-${month}-${day}`
}

// 获取列宽度
const getColumnWidth = (field: FieldConfig) => {
  if (field.widget.type === 'timestamp') return 180
  if (field.widget.type === 'text_area') return 300
  return 150
}

// 🔥 获取搜索值
const getSearchValue = (field: FieldConfig): any => {
  return searchForm.value[field.code] || null
}

// 🔥 更新搜索值
const updateSearchValue = (field: FieldConfig, value: any): void => {
  searchForm.value[field.code] = value
}

// 构建搜索参数
const buildSearchParams = (): SearchParams => {
  const params: SearchParams = {
    page: currentPage.value,
    page_size: pageSize.value
  }

  // 排序
  if (sortField.value && sortOrder.value) {
    params.sort = `${sortField.value}:${sortOrder.value}`
  }

  // 🔥 遍历搜索表单，构建查询参数（新逻辑）
  searchableFields.value.forEach(field => {
    const value = searchForm.value[field.code]
    if (!value) return

    const searchType = field.search || ''
    
    // 精确匹配
    if (searchType.includes('eq')) {
      params.eq = `${field.code}:${value}`
    }
    // 模糊查询
    else if (searchType.includes('like')) {
      params.like = `${field.code}:${value}`
    }
    // 包含查询
    else if (searchType.includes('in')) {
      params.in = `${field.code}:${value}`
    }
    // 范围查询
    else if (searchType.includes('gte') && searchType.includes('lte')) {
      // 可能是对象 {min, max} 或数组 [start, end]
      if (typeof value === 'object') {
        if (Array.isArray(value) && value.length === 2) {
          // 日期范围数组
          if (value[0]) params.gte = `${field.code}:${value[0]}`
          if (value[1]) params.lte = `${field.code}:${value[1]}`
        } else if (value.min !== undefined || value.max !== undefined) {
          // 数字范围对象
          if (value.min !== undefined && value.min !== null && value.min !== '') {
            params.gte = `${field.code}:${value.min}`
          }
          if (value.max !== undefined && value.max !== null && value.max !== '') {
            params.lte = `${field.code}:${value.max}`
          }
        }
      }
    }
  })

  return params
}

// 加载表格数据
const loadTableData = async () => {
  try {
    loading.value = true
    console.log('[TableRenderer] 加载数据')
    console.log('[TableRenderer]   Method:', props.functionData.method)
    console.log('[TableRenderer]   Router:', props.functionData.router)
    
    const params = buildSearchParams()
    console.log('[TableRenderer] 查询参数:', params)
    
    const response = await executeFunction(props.functionData.method, props.functionData.router, params)
    console.log('[TableRenderer] 数据加载成功:', response)
    
    tableData.value = response.items || []
    if (response.paginated) {
      total.value = response.paginated.total_count
      currentPage.value = response.paginated.current_page
    }
  } catch (error) {
    console.error('[TableRenderer] 加载数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  currentPage.value = 1
  loadTableData()
}

// 重置
const handleReset = () => {
  searchForm.value = {}
  currentPage.value = 1
  sortField.value = ''
  sortOrder.value = ''
  loadTableData()
}

// 排序变化
const handleSortChange = ({ prop, order }: any) => {
  sortField.value = prop
  sortOrder.value = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
  loadTableData()
}

// 分页变化
const handleSizeChange = (newSize: number) => {
  pageSize.value = newSize
  currentPage.value = 1
  loadTableData()
}

const handleCurrentChange = (newPage: number) => {
  currentPage.value = newPage
  loadTableData()
}

// 新增
const handleAdd = () => {
  dialogMode.value = 'create'
  currentRow.value = {}
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: any) => {
  dialogMode.value = 'update'
  currentRow.value = { ...row }
  dialogVisible.value = true
}

// 删除
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除这条记录吗？',
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    console.log('[TableRenderer] 删除记录, ID:', row.id)
    
    // 调用删除回调
    await tableDeleteRows(props.functionData.method, props.functionData.router, [row.id])
    
    ElMessage.success('删除成功')
    
    // 重新加载数据
    loadTableData()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('[TableRenderer] 删除失败:', error)
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 对话框提交
const handleDialogSubmit = async (data: Record<string, any>) => {
  try {
    console.log('[TableRenderer] 提交表单')
    console.log('[TableRenderer]   Mode:', dialogMode.value)
    console.log('[TableRenderer]   Data:', data)
    
    if (dialogMode.value === 'create') {
      // 调用新增回调
      await tableAddRow(props.functionData.method, props.functionData.router, data)
      ElMessage.success('新增成功')
    } else {
      // 调用更新回调（需要包含 id）
      const updateData = {
        id: currentRow.value.id,
        ...data
      }
      await tableUpdateRow(props.functionData.method, props.functionData.router, updateData)
      ElMessage.success('更新成功')
    }
    
    // 关闭对话框
    dialogVisible.value = false
    
    // 重新加载数据
    loadTableData()
  } catch (error: any) {
    console.error('[TableRenderer] 提交失败:', error)
    ElMessage.error(error.message || '操作失败')
  }
}

// 🔥 判断是否是 ID 列
const isIdColumn = (field: FieldConfig): boolean => {
  const code = field.code.toLowerCase()
  return code === 'id' || code === 'ID' || code.endsWith('_id') || code.endsWith('Id')
}

// 🔥 显示详情
const handleShowDetail = (row: any, index: number) => {
  currentDetailRow.value = row
  currentDetailIndex.value = index
  showDetailDrawer.value = true
}

// 🔥 导航（上一个/下一个）
const handleNavigate = (direction: 'prev' | 'next') => {
  if (!tableData.value || tableData.value.length === 0) return

  if (direction === 'prev' && currentDetailIndex.value > 0) {
    currentDetailIndex.value--
    currentDetailRow.value = tableData.value[currentDetailIndex.value]
  } else if (direction === 'next' && currentDetailIndex.value < tableData.value.length - 1) {
    currentDetailIndex.value++
    currentDetailRow.value = tableData.value[currentDetailIndex.value]
  }
}

// 监听函数变化，重新加载数据
watch(() => props.functionData, () => {
  searchForm.value = {}
  currentPage.value = 1
  loadTableData()
}, { immediate: true })
</script>

<style scoped>
.table-renderer {
  padding: 20px;
  background: var(--el-bg-color);
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

/* 确保表格单元格背景色正确 */
:deep(.el-table) {
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr) {
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr:hover > td) {
  background-color: var(--el-fill-color-light) !important;
}

/* 确保table内的link按钮清晰可见 */
:deep(.el-button.is-link) {
  font-weight: 500 !important;
}

:deep(.el-button.is-link.el-button--primary) {
  color: var(--el-text-color-primary) !important;
}

:deep(.el-button.is-link.el-button--primary:hover) {
  color: var(--el-color-primary) !important;
}

:deep(.el-button.is-link.el-button--danger) {
  color: var(--el-text-color-primary) !important;
}

:deep(.el-button.is-link.el-button--danger:hover) {
  color: var(--el-color-danger) !important;
}

/* 🔥 ID 列样式 */
:deep(.id-column) {
  .id-cell {
    color: var(--el-color-primary);
    cursor: pointer;
    font-weight: 500;
    transition: all 0.2s;
    
    &:hover {
      color: var(--el-color-primary-light-3);
      text-decoration: underline;
    }
  }
}

/* 🔥 详情抽屉样式 */
.detail-drawer {
  :deep(.el-drawer__header) {
    margin-bottom: 0;
    padding: 20px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .drawer-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
  }

  .drawer-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .drawer-navigation {
    display: flex;
    align-items: center;
    gap: 12px;

    .nav-info {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      min-width: 60px;
      text-align: center;
    }
  }

  .detail-content {
    padding: 20px;

    :deep(.el-descriptions) {
      .el-descriptions__label {
        width: 150px;
        background-color: var(--el-fill-color-light);
        font-weight: 500;
      }

      .el-descriptions__content {
        color: var(--el-text-color-primary);
      }
    }
  }
}
</style>

