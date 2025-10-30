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
          <!-- 精确匹配 (eq) -->
          <el-form-item v-if="field.search?.includes('eq')" :label="field.name">
            <el-input
              v-model="searchForm[`eq_${field.code}`]"
              :placeholder="`请输入${field.name}`"
              clearable
              style="width: 200px"
            />
          </el-form-item>

          <!-- 模糊查询 (like) -->
          <el-form-item v-if="field.search?.includes('like')" :label="field.name">
            <el-input
              v-model="searchForm[`like_${field.code}`]"
              :placeholder="`请输入${field.name}`"
              clearable
              style="width: 200px"
            />
          </el-form-item>

          <!-- 包含查询 (in) - 下拉选择 -->
          <el-form-item v-if="field.search?.includes('in') && field.widget.config.options" :label="field.name">
            <el-select
              v-model="searchForm[`in_${field.code}`]"
              :placeholder="`请选择${field.name}`"
              clearable
              style="width: 200px"
            >
              <el-option
                v-for="option in field.widget.config.options"
                :key="option"
                :label="option"
                :value="option"
              />
            </el-select>
          </el-form-item>

          <!-- 时间范围查询 (gte, lte) -->
          <el-form-item v-if="field.search?.includes('gte') || field.search?.includes('lte')" :label="field.name">
            <el-date-picker
              v-model="searchForm[`daterange_${field.code}`]"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              style="width: 360px"
              value-format="x"
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
      stripe
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
      >
        <template #default="{ row }">
          <span v-if="field.widget.type === 'timestamp'">
            {{ formatTimestamp(row[field.code], field.widget.config.format) }}
          </span>
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, Refresh, Edit, Delete, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { executeFunction, tableAddRow, tableUpdateRow, tableDeleteRows } from '@/api/function'
import FormDialog from './FormDialog.vue'
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

// 搜索表单
const searchForm = ref<Record<string, any>>({})

// 可搜索字段
const searchableFields = computed(() => {
  return props.functionData.response.filter(field => field.search)
})

// 可见字段（根据 table_permission 控制）
const visibleFields = computed(() => {
  return props.functionData.response.filter(field => {
    const permission = field.table_permission
    // 列表中应该显示：
    // - 空（全部权限）
    // - read（只读）
    // - update 不显示
    // - create 不显示
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

  // 遍历搜索表单，构建查询参数
  Object.keys(searchForm.value).forEach(key => {
    const value = searchForm.value[key]
    if (!value) return

    // 解析字段类型和字段名
    const [type, fieldCode] = key.split('_')

    if (type === 'eq' || type === 'like' || type === 'in') {
      params[type] = `${fieldCode}:${value}`
    } else if (type === 'daterange' && Array.isArray(value) && value.length === 2) {
      // 时间范围
      params.gte = `${fieldCode}:${value[0]}`
      params.lte = `${fieldCode}:${value[1]}`
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
}

.search-bar {
  margin-bottom: 20px;
  padding: 20px;
  background: var(--el-bg-color-page);
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
</style>

