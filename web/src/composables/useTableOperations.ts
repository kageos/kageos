/**
 * useTableOperations - 表格操作 Composable
 * 
 * 负责表格的所有业务逻辑：
 * - 数据加载（搜索、分页、排序）
 * - CRUD 操作（新增、编辑、删除）
 * - 状态管理
 * 
 * 设计原则：
 * - 单一职责：只负责业务逻辑，不涉及 UI
 * - 可复用：可在多个表格组件中复用
 * - 可测试：独立的函数，易于单元测试
 * - 类型安全：完整的 TypeScript 类型定义
 */

import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { executeFunction, tableAddRow, tableUpdateRow, tableDeleteRows } from '@/api/function'
import type { Function as FunctionType, SearchParams, TableResponse } from '@/types'
import type { FieldConfig } from '@/core/types/field'

export interface TableOperationsOptions {
  functionData: FunctionType
}

export interface TableOperationsReturn {
  // 状态
  loading: ReturnType<typeof ref<boolean>>
  tableData: ReturnType<typeof ref<any[]>>
  searchForm: ReturnType<typeof ref<Record<string, any>>>
  currentPage: ReturnType<typeof ref<number>>
  pageSize: ReturnType<typeof ref<number>>
  total: ReturnType<typeof ref<number>>
  sortField: ReturnType<typeof ref<string>>
  sortOrder: ReturnType<typeof ref<string>>
  
  // 计算属性
  searchableFields: ReturnType<typeof computed<FieldConfig[]>>
  visibleFields: ReturnType<typeof computed<FieldConfig[]>>
  hasAddCallback: ReturnType<typeof computed<boolean>>
  hasUpdateCallback: ReturnType<typeof computed<boolean>>
  hasDeleteCallback: ReturnType<typeof computed<boolean>>
  
  // 方法
  loadTableData: () => Promise<void>
  handleSearch: () => void
  handleReset: () => void
  handleSortChange: (sorts: Array<{ column?: any; prop?: string; order?: string | null }>) => void
  handleSizeChange: (size: number) => void
  handleCurrentChange: (page: number) => void
  handleAdd: (data: Record<string, any>) => Promise<boolean>
  handleUpdate: (id: number, data: Record<string, any>) => Promise<boolean>
  handleDelete: (id: number) => Promise<boolean>
  buildSearchParams: () => SearchParams
}

/**
 * 表格操作 Composable
 * 
 * @param options 配置选项
 * @returns 表格操作的状态和方法
 */
export function useTableOperations(options: TableOperationsOptions): TableOperationsReturn {
  const { functionData } = options
  
  // ==================== 状态 ====================
  
  /** 加载状态 */
  const loading = ref(false)
  
  /** 表格数据 */
  const tableData = ref<any[]>([])
  
  /** 搜索表单数据 */
  const searchForm = ref<Record<string, any>>({})
  
  /** 当前页码 */
  const currentPage = ref(1)
  
  /** 每页数量 */
  const pageSize = ref(20)
  
  /** 总记录数 */
  const total = ref(0)
  
  /** 排序字段 */
  const sortField = ref('')
  
  /** 排序方向 */
  const sortOrder = ref('')
  
  // ==================== 计算属性 ====================
  
  /**
   * 可搜索字段（配置了 search 的字段）
   */
  const searchableFields = computed(() => {
    return functionData.response.filter(field => field.search)
  })
  
  /**
   * 可见字段（根据 table_permission 过滤）
   * 
   * 列表中只显示：
   * - 空（全部权限）
   * - read（只读字段）
   * 
   * 不显示：
   * - create（只在新增表单显示）
   * - update（只在编辑表单显示）
   */
  const visibleFields = computed(() => {
    return functionData.response.filter(field => {
      const permission = field.table_permission
      return !permission || permission === '' || permission === 'read'
    })
  })
  
  /**
   * 是否有新增回调
   */
  const hasAddCallback = computed(() => {
    const callbacks = functionData.callbacks || ''
    return callbacks.includes('OnTableAddRow')
  })
  
  /**
   * 是否有更新回调
   */
  const hasUpdateCallback = computed(() => {
    const callbacks = functionData.callbacks || ''
    return callbacks.includes('OnTableUpdateRow')
  })
  
  /**
   * 是否有删除回调
   */
  const hasDeleteCallback = computed(() => {
    const callbacks = functionData.callbacks || ''
    return callbacks.includes('OnTableDeleteRows')
  })
  
  // ==================== 业务逻辑 ====================
  
  /**
   * 构建搜索参数
   * 
   * 将搜索表单数据转换为后端需要的 SearchParams 格式
   * 支持：精确匹配(eq)、模糊查询(like)、包含查询(in)、范围查询(gte/lte)
   */
  const buildSearchParams = (): SearchParams => {
    const params: SearchParams = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    
    // 排序（格式：sorts=field:order）
    // 后端支持多列排序（sorts=field1:order1,field2:order2），但我们目前只支持单列排序
    if (sortField.value && sortOrder.value) {
      params.sorts = `${sortField.value}:${sortOrder.value}`
    }
    
    // 遍历搜索表单，构建查询参数
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
  
  /**
   * 加载表格数据
   * 
   * 调用后端 API 获取表格数据，支持搜索、分页、排序
   */
  const loadTableData = async (): Promise<void> => {
    try {
      loading.value = true
      console.log('[useTableOperations] 加载数据')
      console.log('[useTableOperations]   Method:', functionData.method)
      console.log('[useTableOperations]   Router:', functionData.router)
      
      const params = buildSearchParams()
      console.log('[useTableOperations] 查询参数:', params)
      
      const response = await executeFunction(functionData.method, functionData.router, params) as TableResponse
      console.log('[useTableOperations] 数据加载成功:', response)
      
      tableData.value = response.items || []
      if (response.paginated) {
        total.value = response.paginated.total_count
        currentPage.value = response.paginated.current_page
      }
    } catch (error: any) {
      console.error('[useTableOperations] 加载数据失败:', error)
      ElMessage.error(error.message || '加载数据失败')
      tableData.value = []
    } finally {
      loading.value = false
    }
  }
  
  /**
   * 搜索
   * 重置到第一页并重新加载数据
   */
  const handleSearch = (): void => {
    currentPage.value = 1
    loadTableData()
  }
  
  /**
   * 重置搜索
   * 清空搜索表单、排序，重置到第一页并重新加载数据
   */
  const handleReset = (): void => {
    searchForm.value = {}
    currentPage.value = 1
    sortField.value = ''
    sortOrder.value = ''
    loadTableData()
  }
  
  /**
   * 排序变化
   * @param sorts 排序信息数组 [{ column: 列对象, prop: 字段名, order: 'ascending' | 'descending' | null }]
   * 
   * 注意：Element Plus 的 sort-change 事件传递的是 sorts 数组（支持多列排序）
   * 但我们的后端只支持单列排序，所以只取第一个
   */
  const handleSortChange = (sorts: Array<{ column?: any; prop?: string; order?: string | null }>): void => {
    // 只处理第一个排序（后端只支持单列排序）
    const sort = sorts && sorts.length > 0 ? sorts[0] : null
    
    if (sort && sort.prop && sort.order) {
      sortField.value = sort.prop
      sortOrder.value = sort.order === 'ascending' ? 'asc' : sort.order === 'descending' ? 'desc' : ''
    } else {
      // 取消排序
      sortField.value = ''
      sortOrder.value = ''
    }
    
    loadTableData()
  }
  
  /**
   * 分页大小变化
   * @param newSize 新的每页数量
   */
  const handleSizeChange = (newSize: number): void => {
    pageSize.value = newSize
    currentPage.value = 1
    loadTableData()
  }
  
  /**
   * 当前页变化
   * @param newPage 新的页码
   */
  const handleCurrentChange = (newPage: number): void => {
    currentPage.value = newPage
    loadTableData()
  }
  
  /**
   * 新增记录
   * @param data 新增的数据
   * @returns 是否成功
   */
  const handleAdd = async (data: Record<string, any>): Promise<boolean> => {
    try {
      console.log('[useTableOperations] 新增记录:', data)
      await tableAddRow(functionData.method, functionData.router, data)
      ElMessage.success('新增成功')
      await loadTableData()
      return true
    } catch (error: any) {
      console.error('[useTableOperations] 新增失败:', error)
      ElMessage.error(error.message || '新增失败')
      return false
    }
  }
  
  /**
   * 更新记录
   * @param id 记录 ID
   * @param data 更新的数据
   * @returns 是否成功
   */
  const handleUpdate = async (id: number, data: Record<string, any>): Promise<boolean> => {
    try {
      console.log('[useTableOperations] 更新记录:', { id, data })
      const updateData = {
        id,
        ...data
      }
      await tableUpdateRow(functionData.method, functionData.router, updateData)
      ElMessage.success('更新成功')
      await loadTableData()
      return true
    } catch (error: any) {
      console.error('[useTableOperations] 更新失败:', error)
      ElMessage.error(error.message || '更新失败')
      return false
    }
  }
  
  /**
   * 删除记录
   * @param id 记录 ID
   * @returns 是否成功
   */
  const handleDelete = async (id: number): Promise<boolean> => {
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
      
      console.log('[useTableOperations] 删除记录, ID:', id)
      await tableDeleteRows(functionData.method, functionData.router, [id])
      ElMessage.success('删除成功')
      await loadTableData()
      return true
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('[useTableOperations] 删除失败:', error)
        ElMessage.error(error.message || '删除失败')
      }
      return false
    }
  }
  
  // ==================== 返回 ====================
  
  return {
    // 状态
    loading,
    tableData,
    searchForm,
    currentPage,
    pageSize,
    total,
    sortField,
    sortOrder,
    
    // 计算属性
    searchableFields,
    visibleFields,
    hasAddCallback,
    hasUpdateCallback,
    hasDeleteCallback,
    
    // 方法
    loadTableData,
    handleSearch,
    handleReset,
    handleSortChange,
    handleSizeChange,
    handleCurrentChange,
    handleAdd,
    handleUpdate,
    handleDelete,
    buildSearchParams
  }
}

