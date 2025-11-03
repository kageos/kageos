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

import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { executeFunction, tableAddRow, tableUpdateRow, tableDeleteRows } from '@/api/function'
import type { Function as FunctionType, SearchParams, TableResponse } from '@/types'
import type { FieldConfig } from '@/core/types/field'

/**
 * 排序项接口
 */
interface SortItem {
  field: string  // 字段名
  order: 'asc' | 'desc'  // 排序方向
}

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
  isDefaultSort: ReturnType<typeof computed<boolean>>
  defaultSortConfig: ReturnType<typeof computed<{ prop: string; order: 'descending' } | null>>
  
  // 方法
  loadTableData: () => Promise<void>
  handleSearch: () => void
  handleReset: () => void
  handleSortChange: (sortInfo: { prop?: string; order?: string }) => void
  syncToURL: () => void
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
  
  // ==================== Vue Router ====================
  
  const route = useRoute()
  const router = useRouter()
  
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
  
  /** 排序列表（支持多字段排序） */
  const sorts = ref<SortItem[]>([])
  
  /** 用户是否手动操作过排序 */
  const hasManualSort = ref(false)
  
  // ==================== 辅助函数 ====================
  
  /**
   * 获取 ID 字段的 code
   */
  const getIdFieldCode = (): string | null => {
    const idField = functionData.response.find(field => field.widget?.type === 'ID')
    return idField?.code || null
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
   * 从排序列表移除指定字段
   */
  const removeSortByField = (field: string): void => {
    sorts.value = sorts.value.filter(item => item.field !== field)
  }
  
  /**
   * 添加或更新排序项
   */
  const setSortItem = (field: string, order: 'asc' | 'desc'): void => {
    // 移除已有的该字段排序
    removeSortByField(field)
    // 添加到列表末尾
    sorts.value.push({ field, order })
  }
  
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
  
  /**
   * 当前是否使用默认排序（id 降序）
   * 用于 UI 显示默认排序状态
   */
  const isDefaultSort = computed(() => {
    return !hasManualSort.value && sorts.value.length === 0
  })
  
  /**
   * 获取默认排序配置（用于 el-table 的 default-sort）
   * 返回 { prop: string, order: 'descending' } 或 null
   */
  const defaultSortConfig = computed(() => {
    if (!isDefaultSort.value) return null
    
    const idFieldCode = getIdFieldCode()
    if (idFieldCode) {
      return {
        prop: idFieldCode,
        order: 'descending' as const
      }
    }
    return null
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
    
    // 排序（格式：sorts=field1:order1,field2:order2）
    // 支持多字段排序
    if (sorts.value.length > 0) {
      params.sorts = sorts.value.map(item => `${item.field}:${item.order}`).join(',')
    } else {
      // 如果没有手动排序且存在 ID 字段，使用默认排序（id 降序）
      const defaultSorts = buildDefaultSorts()
      if (defaultSorts.length > 0) {
        params.sorts = defaultSorts.map(item => `${item.field}:${item.order}`).join(',')
      }
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
      console.log('[useTableOperations] 排序参数:', {
        sorts: sorts.value,
        hasManualSort: hasManualSort.value,
        sortsString: params.sorts
      })
      
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
    syncToURL()
    loadTableData()
  }
  
  /**
   * 同步状态到 URL
   */
  const syncToURL = (): void => {
    const query: Record<string, string> = {}
    
    // 分页参数
    if (currentPage.value > 1) {
      query.page = String(currentPage.value)
    }
    if (pageSize.value !== 20) {
      query.page_size = String(pageSize.value)
    }
    
    // 排序参数
    const finalSorts = sorts.value.length > 0 
      ? sorts.value 
      : (hasManualSort.value ? [] : buildDefaultSorts())
    
    if (finalSorts.length > 0) {
      query.sorts = finalSorts.map(item => `${item.field}:${item.order}`).join(',')
    }
    
    // 搜索参数
    searchableFields.value.forEach(field => {
      const value = searchForm.value[field.code]
      if (!value) return
      
      const searchType = field.search || ''
      if (searchType.includes('eq')) {
        query[`eq_${field.code}`] = String(value)
      } else if (searchType.includes('like')) {
        query[`like_${field.code}`] = String(value)
      } else if (searchType.includes('in')) {
        query[`in_${field.code}`] = String(value)
      } else if (searchType.includes('gte') && searchType.includes('lte')) {
        if (typeof value === 'object') {
          if (Array.isArray(value) && value.length === 2) {
            if (value[0]) query[`gte_${field.code}`] = String(value[0])
            if (value[1]) query[`lte_${field.code}`] = String(value[1])
          } else if (value.min !== undefined || value.max !== undefined) {
            if (value.min !== undefined && value.min !== null && value.min !== '') {
              query[`gte_${field.code}`] = String(value.min)
            }
            if (value.max !== undefined && value.max !== null && value.max !== '') {
              query[`lte_${field.code}`] = String(value.max)
            }
          }
        }
      }
    })
    
    // 更新 URL（不触发导航）
    router.replace({ query: { ...route.query, ...query } })
  }
  
  /**
   * 从 URL 恢复状态
   */
  const restoreFromURL = (): void => {
    const query = route.query
    
    // 恢复分页
    if (query.page) {
      const page = parseInt(String(query.page), 10)
      if (!isNaN(page) && page > 0) {
        currentPage.value = page
      }
    }
    if (query.page_size) {
      const size = parseInt(String(query.page_size), 10)
      if (!isNaN(size) && size > 0) {
        pageSize.value = size
      }
    }
    
    // 恢复排序
    if (query.sorts) {
      const sortsString = String(query.sorts)
      const sortItems: SortItem[] = []
      sortsString.split(',').forEach(sortStr => {
        const parts = sortStr.trim().split(':')
        if (parts.length === 2) {
          const field = parts[0] || ''
          const order = parts[1] as 'asc' | 'desc'
          if (field && (order === 'asc' || order === 'desc')) {
            sortItems.push({ field, order })
          }
        }
      })
      if (sortItems.length > 0) {
        sorts.value = sortItems
        hasManualSort.value = true
      }
    }
    
    // 恢复搜索
    searchableFields.value.forEach(field => {
      const searchType = field.search || ''
      if (searchType.includes('eq')) {
        const value = query[`eq_${field.code}`]
        if (value) searchForm.value[field.code] = String(value)
      } else if (searchType.includes('like')) {
        const value = query[`like_${field.code}`]
        if (value) searchForm.value[field.code] = String(value)
      } else if (searchType.includes('in')) {
        const value = query[`in_${field.code}`]
        if (value) searchForm.value[field.code] = String(value)
      } else if (searchType.includes('gte') && searchType.includes('lte')) {
        const gteValue = query[`gte_${field.code}`]
        const lteValue = query[`lte_${field.code}`]
        if (gteValue || lteValue) {
          // 根据字段类型判断是数字范围还是日期范围
          const fieldType = field.data?.type
          if (fieldType === 'timestamp' || fieldType === 'datetime') {
            searchForm.value[field.code] = [gteValue ? String(gteValue) : null, lteValue ? String(lteValue) : null]
          } else {
            searchForm.value[field.code] = {
              min: gteValue ? String(gteValue) : undefined,
              max: lteValue ? String(lteValue) : undefined
            }
          }
        }
      }
    })
  }
  
  /**
   * 重置搜索
   * 清空搜索表单、排序，重置到第一页并重新加载数据
   */
  const handleReset = (): void => {
    searchForm.value = {}
    currentPage.value = 1
    sorts.value = []
    hasManualSort.value = false
    syncToURL()
    loadTableData()
  }
  
  /**
   * 排序变化
   * @param sortInfo 排序信息对象 { prop: 字段名, order: 'ascending' | 'descending' | '' }
   * 
   * 注意：Element Plus 的 sort-change 事件传递的是单个对象
   * - order 为 'ascending' 表示升序
   * - order 为 'descending' 表示降序
   * - order 为 ''（空字符串）或不存在时表示取消排序
   * 
   * 规则：
   * 1. 用户手动排序时，移除默认的 id 排序
   * 2. 支持多字段排序，新字段追加到列表末尾
   * 3. 同一字段重复排序会更新该字段的排序方向
   */
  const handleSortChange = (sortInfo: { prop?: string; order?: string }): void => {
    console.log('[useTableOperations] 排序变化:', sortInfo)
    
    hasManualSort.value = true
    
    if (sortInfo && sortInfo.prop && sortInfo.order && sortInfo.order !== '') {
      const field = sortInfo.prop
      const order = sortInfo.order === 'ascending' ? 'asc' : 'desc'
      
      // 如果是第一次手动排序，移除默认的 id 排序
      const idFieldCode = getIdFieldCode()
      if (idFieldCode) {
        // 移除 id 排序（用户手动排序时，id 排序会被移除）
        removeSortByField(idFieldCode)
      }
      
      // 添加或更新排序项
      setSortItem(field, order)
      
      console.log('[useTableOperations] 设置排序:', { field, order, allSorts: sorts.value })
    } else {
      // 取消该字段的排序
      if (sortInfo.prop) {
        removeSortByField(sortInfo.prop)
        console.log('[useTableOperations] 取消字段排序:', sortInfo.prop)
      }
    }
    
    syncToURL()
    loadTableData()
  }
  
  /**
   * 分页大小变化
   * @param newSize 新的每页数量
   */
  const handleSizeChange = (newSize: number): void => {
    pageSize.value = newSize
    currentPage.value = 1
    syncToURL()
    loadTableData()
  }
  
  /**
   * 当前页变化
   * @param newPage 新的页码
   */
  const handleCurrentChange = (newPage: number): void => {
    currentPage.value = newPage
    syncToURL()
    loadTableData()
  }
  
  // ==================== 初始化 ====================
  
  /**
   * 初始化：从 URL 恢复状态或使用默认排序
   */
  const initialize = (): void => {
    restoreFromURL()
    // 如果 URL 中没有排序且没有手动排序，使用默认排序
    if (sorts.value.length === 0 && !hasManualSort.value) {
      const defaultSorts = buildDefaultSorts()
      if (defaultSorts.length > 0) {
        sorts.value = defaultSorts
      }
    }
  }
  
  // 初始化（在首次创建时）
  initialize()
  
  // 监听 URL 变化，恢复状态（避免循环更新）
  let isRestoringFromURL = false
  watch(() => route.query, () => {
    if (isRestoringFromURL) return
    isRestoringFromURL = true
    restoreFromURL()
    loadTableData().finally(() => {
      isRestoringFromURL = false
    })
  }, { deep: true })
  
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
    sortField: computed(() => sorts.value[0]?.field || ''),
    sortOrder: computed(() => sorts.value[0]?.order || ''),
    
    // 计算属性
    searchableFields,
    visibleFields,
    hasAddCallback,
    hasUpdateCallback,
    hasDeleteCallback,
    isDefaultSort,
    defaultSortConfig,
    
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
    buildSearchParams,
    syncToURL
  }
}

