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
import { buildSearchParamsString, buildURLSearchParams } from '@/utils/searchParams'
import { denormalizeSearchValue } from '@/utils/searchValueNormalizer'
import { parseCommaSeparatedString } from '@/utils/stringUtils'
import { SearchType } from '@/core/constants/search'
import { WidgetType } from '@/core/constants/widget'
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
  sorts: ReturnType<typeof ref<SortItem[]>>
  
  // 计算属性
  getFieldSortOrder: (fieldCode: string) => 'ascending' | 'descending' | null
  searchableFields: ReturnType<typeof computed<FieldConfig[]>>
  visibleFields: ReturnType<typeof computed<FieldConfig[]>>
  hasAddCallback: ReturnType<typeof computed<boolean>>
  hasUpdateCallback: ReturnType<typeof computed<boolean>>
  hasDeleteCallback: ReturnType<typeof computed<boolean>>
  hasManualSort: ReturnType<typeof ref<boolean>>
  
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
  restoreFromURL: () => void
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
    const idField = functionData.response.find(field => field.widget?.type === WidgetType.ID)
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
   * 获取字段的排序状态（用于 el-table-column 的 sort-orders）
   * 
   * ⚠️ 关键：Element Plus 的 el-table 在 custom 模式下，需要手动设置每个列的排序状态
   * 这样才能正确显示排序标识（箭头）
   * 
   * @param fieldCode 字段 code
   * @returns 排序方向：'ascending' | 'descending' | null（无排序）
   */
  const getFieldSortOrder = (fieldCode: string): 'ascending' | 'descending' | null => {
    const sortItem = sorts.value.find(item => item.field === fieldCode)
    if (!sortItem) {
      return null
    }
    return sortItem.order === 'asc' ? 'ascending' : 'descending'
  }
  
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
      page_size: pageSize.value,
      ...buildSearchParamsString(searchForm.value, searchableFields.value)
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
      const params = buildSearchParams()
      const response = await executeFunction(functionData.method, functionData.router, params) as TableResponse
      
      tableData.value = response.items || []
      if (response.paginated) {
        total.value = response.paginated.total_count
        currentPage.value = response.paginated.current_page
      }
    } catch (error: any) {
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
    
    // 搜索参数（使用工具函数）
    Object.assign(query, buildURLSearchParams(searchForm.value, searchableFields.value))
    
    // 🔥 清理空值参数（确保不会生成 field: 这样的空参数）
    Object.keys(query).forEach(key => {
      const value = query[key]
      if (!value || (typeof value === 'string' && (value.endsWith(':') || value.trim() === ''))) {
        delete query[key]
      }
    })
    
    // 🔥 清理 URL 中已存在的搜索参数（如果字段已清空，从 URL 中删除）
    const searchParamKeys = ['eq', 'like', 'in', 'contains', 'gte', 'lte']
    const newQuery: Record<string, string> = {}
    
    // 🔥 先复制所有非搜索参数（分页、排序等）
    Object.keys(route.query).forEach(key => {
      if (!searchParamKeys.includes(key)) {
        newQuery[key] = String(route.query[key])
      }
    })
    
    // 🔥 然后添加新的搜索参数（buildURLSearchParams 已经过滤了空值）
    Object.assign(newQuery, query)
    
    // 🔥 更新 URL（不触发导航）
    router.replace({ query: newQuery })
  }
  
  /**
   * 从 URL 恢复状态
   */
  /**
   * 从 URL 恢复状态（搜索条件、排序、分页）
   * 
   * ⚠️ 关键逻辑：
   * 1. 解析 URL 参数，恢复搜索表单的值
   * 2. 支持多个字段同时使用相同的搜索类型（如：多个 slider 字段使用 gte/lte）
   * 3. 对于范围搜索（gte/lte），需要区分时间戳类型和数字类型
   * 
   * URL 格式示例：
   * - 单个字段：gte=progress:50&lte=progress:80
   * - 多个字段：gte=progress:50,score:5&lte=progress:80,score:8
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
    
    // 恢复搜索（格式：eq=field:value 或 eq=field1:value1,field2:value2, like=field:value, in=field:value, gte=field:value, lte=field:value）
    // 🔥 支持多个字段使用相同搜索类型，格式：field1:value1,field2:value2
    searchableFields.value.forEach(field => {
      const searchType = field.search || ''
      
      if (searchType.includes(SearchType.EQ)) {
        const eqValue = query.eq
        if (eqValue) {
          // 🔥 支持多个字段：field1:value1,field2:value2
          const eqStr = String(eqValue)
          const parts = eqStr.split(',')
          for (const part of parts) {
            if (part.trim().startsWith(`${field.code}:`)) {
              const value = part.trim().substring(field.code.length + 1)
              if (value) {
                // 🔥 使用值规范化工具统一处理值转换
                const denormalizedValue = denormalizeSearchValue(value, {
                  widgetType: field.widget?.type,
                  searchType: field.search,
                  field
                })
                searchForm.value[field.code] = denormalizedValue
                break
              }
            }
          }
        }
      } else if (searchType.includes(SearchType.LIKE)) {
        const likeValue = query.like
        if (likeValue) {
          // 🔥 支持多个字段：field1:value1,field2:value2
          const likeStr = String(likeValue)
          const parts = likeStr.split(',')
          for (const part of parts) {
            if (part.trim().startsWith(`${field.code}:`)) {
              const value = part.trim().substring(field.code.length + 1)
              if (value) {
                searchForm.value[field.code] = value
                break
              }
            }
          }
        }
      } 
      // 🔥 必须先检查 contains，再检查 in，因为 "contains" 包含 "in" 子字符串
      else if (searchType.includes(SearchType.CONTAINS)) {
        // 🔥 contains 类型：用于多选场景，使用 FIND_IN_SET
        const containsValue = query.contains
        if (containsValue) {
          // 🔥 支持多个字段：使用逗号 , 分隔多个字段，与 in 操作符保持一致
          // 格式：contains=tags:高,中,otherField:value1,value2（与 in 操作符格式一致）
          const containsStr = String(containsValue)
          
          // 🔥 查找当前字段的部分（field:value1,value2,...）
          // 需要处理字段值中可能包含逗号的情况
          const fieldPrefix = `${field.code}:`
          const fieldIndex = containsStr.indexOf(fieldPrefix)
          
          if (fieldIndex >= 0) {
            // 找到字段开始位置
            const valueStart = fieldIndex + fieldPrefix.length
            let valueEnd = containsStr.length
            
            // 🔥 查找下一个字段的开始位置（下一个 field: 的位置）
            // 需要找到所有可能的字段名（从 searchableFields 中获取）
            const allFieldCodes = searchableFields.value.map(f => f.code)
            let nextFieldIndex = -1
            
            for (const otherFieldCode of allFieldCodes) {
              if (otherFieldCode === field.code) continue
              const otherFieldPrefix = `${otherFieldCode}:`
              const index = containsStr.indexOf(otherFieldPrefix, valueStart)
              if (index >= 0 && (nextFieldIndex < 0 || index < nextFieldIndex)) {
                nextFieldIndex = index
              }
            }
            
            if (nextFieldIndex >= 0) {
              valueEnd = nextFieldIndex
            }
            
            const valueStr = containsStr.substring(valueStart, valueEnd).trim()
            
            if (valueStr) {
              // 🔥 contains 类型：将逗号分隔的字符串转换为数组（用于多选组件显示）
              const values = parseCommaSeparatedString(valueStr)
              // 🔥 多选组件始终使用数组格式
              if (field.widget?.type === WidgetType.MULTI_SELECT) {
                searchForm.value[field.code] = values.length > 0 ? values : []
              } else {
                // 其他类型：如果只有一个值，保持字符串；多个值使用数组
                searchForm.value[field.code] = values.length > 1 ? values : (values.length === 1 ? values[0] : valueStr)
              }
            }
          }
        }
      } else if (searchType.includes(SearchType.IN)) {
        const inValue = query.in
        if (inValue) {
          // 🔥 支持多个字段：field1:value1,value2,field2:value3,value4
          // 格式：in=handler:liubeiluo,sina,otherField:value1,value2
          const inStr = String(inValue)
          
          // 🔥 找到当前字段的部分（field:value1,value2,...）
          // 需要处理字段值中可能包含逗号的情况
          const fieldPrefix = `${field.code}:`
          const fieldIndex = inStr.indexOf(fieldPrefix)
          
          if (fieldIndex >= 0) {
            // 找到字段开始位置
            const valueStart = fieldIndex + fieldPrefix.length
            let valueEnd = inStr.length
            
            // 🔥 查找下一个字段的开始位置（下一个 field: 的位置）
            // 需要找到所有可能的字段名（从 searchableFields 中获取）
            const allFieldCodes = searchableFields.value.map(f => f.code)
            let nextFieldIndex = -1
            
            for (const otherFieldCode of allFieldCodes) {
              if (otherFieldCode === field.code) continue
              const otherFieldPrefix = `${otherFieldCode}:`
              const index = inStr.indexOf(otherFieldPrefix, valueStart)
              if (index >= 0 && (nextFieldIndex < 0 || index < nextFieldIndex)) {
                nextFieldIndex = index
              }
            }
            
            if (nextFieldIndex >= 0) {
              valueEnd = nextFieldIndex
            }
            
            // 提取字段值部分
            const valueStr = inStr.substring(valueStart, valueEnd).trim()
            
              if (valueStr) {
                // 🔥 in 类型支持多选，需要将逗号分隔的字符串转换为数组
                // 注意：如果字段是 user 或 multiselect 类型且 search 包含 'in'，即使只有一个值也要转换为数组
                const values = parseCommaSeparatedString(valueStr)
                // 🔥 如果字段是 user 或 multiselect 类型，始终使用数组格式（因为 ElSelect 的 multiple 模式需要数组）
                if ((field.widget?.type === WidgetType.USER || field.widget?.type === WidgetType.MULTI_SELECT) && searchType.includes(SearchType.IN)) {
                  searchForm.value[field.code] = values.length > 0 ? values : []
                } else {
                  // 其他类型：如果只有一个值，保持字符串；多个值使用数组
                  searchForm.value[field.code] = values.length > 1 ? values : (values.length === 1 ? values[0] : valueStr)
                }
              }
          }
        }
      } else if (searchType.includes(SearchType.GTE) && searchType.includes(SearchType.LTE)) {
        const gteValue = query.gte
        const lteValue = query.lte
        
        // 解析 gte（支持多个字段）
        let gte: string | null = null
        if (gteValue) {
          const gteStr = String(gteValue)
          const parts = gteStr.split(',')
          for (const part of parts) {
            if (part.trim().startsWith(`${field.code}:`)) {
              gte = part.trim().substring(field.code.length + 1)
              break
            }
          }
        }
        
        // 解析 lte（支持多个字段）
        let lte: string | null = null
        if (lteValue) {
          const lteStr = String(lteValue)
          const parts = lteStr.split(',')
          for (const part of parts) {
            if (part.trim().startsWith(`${field.code}:`)) {
              lte = part.trim().substring(field.code.length + 1)
              break
            }
          }
        }
        
        if (gte || lte) {
          // 根据字段类型判断是数字范围还是日期范围
          // 🔥 检查 widget.type 或 data.type 是否为 timestamp
          /**
           * ⚠️ 关键：区分时间戳类型和数字类型
           * 时间戳类型：使用数组格式 [start, end]（用于 ElDatePicker）
           * 数字类型：使用对象格式 { min, max }（用于 slider 组件）
           */
          const fieldType = field.data?.type
          const widgetType = field.widget?.type
          const isTimestamp = fieldType === 'timestamp' || widgetType === 'timestamp'
          
          if (isTimestamp) {
            // 🔥 时间戳类型：将字符串转换为数字（ElDatePicker 的 valueFormat='x' 需要毫秒级时间戳）
            // 🔥 兼容旧格式：如果 URL 中的时间戳值 < 9999999999，认为是秒级，需要转换为毫秒级
            const SECONDS_THRESHOLD = 9999999999
            const convertTimestamp = (ts: string | null): number | null => {
              if (!ts) return null
              const num = Number(ts)
              // 如果值很小，可能是旧格式的秒级时间戳，转换为毫秒级
              if (num > 0 && num < SECONDS_THRESHOLD) {
                return num * 1000
              }
              return num
            }
            const timestampRange = [
              gte ? convertTimestamp(gte) : null,
              lte ? convertTimestamp(lte) : null
            ]
            searchForm.value[field.code] = timestampRange
          } else {
            // 数字类型（slider 组件）：使用对象格式 { min, max }
            searchForm.value[field.code] = {
              min: gte ? String(gte) : undefined,
              max: lte ? String(lte) : undefined
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
   * ⚠️ 关键规则：
   * 1. id 排序与其他排序互斥：id 是自增的，如果 id 排序在前面，其他排序就无意义了
   * 2. id 不能在最前面：规定 id 不能作为第一个排序条件
   * 3. 用户手动排序时，立即移除 id 排序（无论 id 是否在列表中）
   * 4. 支持多字段排序，新字段追加到列表末尾
   * 5. 同一字段重复排序会更新该字段的排序方向
   */
  const handleSortChange = (sortInfo: { prop?: string; order?: string }): void => {
    hasManualSort.value = true
    
    if (sortInfo && sortInfo.prop && sortInfo.order && sortInfo.order !== '') {
      const field = sortInfo.prop
      const order = sortInfo.order === 'ascending' ? 'asc' : 'desc'
      
      // ⚠️ 关键：id 排序与其他排序互斥
      // id 是自增的，如果 id 排序在前面，其他排序就无意义了
      // 规定：id 不能在最前面，用户手动排序时，立即移除 id 排序
      const idFieldCode = getIdFieldCode()
      if (idFieldCode) {
        // 移除 id 排序（无论 id 是否在列表中）
        removeSortByField(idFieldCode)
      }
      
      // 添加或更新排序项
      setSortItem(field, order)
    } else {
      // 取消该字段的排序
      if (sortInfo.prop) {
        removeSortByField(sortInfo.prop)
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
    // 🔥 初始化后加载数据
    loadTableData()
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
      await tableAddRow(functionData.method, functionData.router, data)
      ElMessage.success('新增成功')
      await loadTableData()
      return true
    } catch (error: any) {
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
      const updateData = {
        id,
        ...data
      }
      await tableUpdateRow(functionData.method, functionData.router, updateData)
      ElMessage.success('更新成功')
      await loadTableData()
      return true
    } catch (error: any) {
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
      
      await tableDeleteRows(functionData.method, functionData.router, [id])
      ElMessage.success('删除成功')
      await loadTableData()
      return true
    } catch (error: any) {
      if (error !== 'cancel') {
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
    sorts,
    
    // 计算属性
    getFieldSortOrder,
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
    buildSearchParams,
    syncToURL,
    restoreFromURL
  }
}

