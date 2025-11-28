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

import { ref, computed, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { executeFunction, tableAddRow, tableUpdateRow, tableDeleteRows } from '@/api/function'
import { buildSearchParamsString, buildURLSearchParams } from '@/utils/searchParams'
import { denormalizeSearchValue } from '@/utils/searchValueNormalizer'
import { parseCommaSeparatedString } from '@/utils/stringUtils'
import { getChangedFields } from '@/utils/objectDiff'
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
  handleUpdate: (id: number, data: Record<string, any>, oldData?: Record<string, any>) => Promise<boolean>
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
   * 可搜索字段（用于搜索表单显示）
   * 
   * ==================== 搜索设计说明 ====================
   * 
   * 【核心概念】
   * - response 字段：针对这个接口的主表的字段
   *   - 这些字段直接存储在主表中（如 `crm_meeting_room_booking` 表的 `subject`、`booker` 等）
   *   - 必须明确指定 `search` 标签值（如 `"like"`、`"in"`、`"eq"` 等）才会显示在搜索表单中
   *   - 搜索时转换为 URL 查询参数格式：`like=remark:测试`、`in=status:待处理,处理中` 等
   *   - 如果 `search` 是空字符串 `""` 或 `"-"`，则不会显示在搜索表单中
   * 
   * - request 字段：非这张表的参数（扩展字段）
   *   - 这些字段不在主表中，可能是：
   *     * 计算字段（如 `status`，根据时间实时计算，不存储在数据库）
   *     * 外表字段（如 `room_name`，来自关联表 `crm_meeting_room`，需要通过 JOIN 或子查询获取）
   *     * 其他扩展字段（用于搜索但不在主表中的字段）
   *   - 本身就是用于搜索的表单参数，不需要 `search` 标签（但可以设置 `search: "-"` 明确表示不支持搜索）
   *   - 搜索时直接作为 `k=v` 形式：`status=进行中`、`room_name=会议室A` 等
   *   - 如果 `search` 是 `"-"`，则不会显示在搜索表单中
   * 
   * 【合并策略】
   * 1. 从 response 中获取所有可搜索字段（主表字段，必须有明确的 search 标签值）
   * 2. 从 request 中获取所有字段（扩展字段，用于搜索，不需要 search 标签）
   * 3. 智能合并：如果同一个字段在两个地方都存在，保留 response 的 search 信息，使用 request 的其他配置
   * 
   * 【示例】
   * - response 中的 `subject` 字段：`search: "like"` → 显示在搜索表单，搜索时转换为 `like=subject:测试`
   * - response 中的 `status` 字段：`search: "-"` → 不显示在搜索表单（明确表示不支持搜索）
   * - request 中的 `room_name` 字段：`search: ""` → 显示在搜索表单，搜索时转换为 `room_name=会议室A`
   * - request 中的 `status` 字段：`search: "-"` → 不显示在搜索表单（明确表示不支持搜索）
   */
  const searchableFields = computed(() => {
    // 从 response 中获取可搜索字段（主表字段，必须有明确的 search 标签值）
    // ⚠️ 关键：response 字段必须明确指定 search 值（不能是空字符串、"-"、null、undefined）
    // 只有 search 有明确值（如 "in"、"like"、"eq" 等）的字段才会显示在搜索表单中
    const responseSearchableFields = Array.isArray(functionData.response) 
      ? functionData.response.filter(field => {
          const search = field.search
          // 必须有值，且不是 "-"（明确表示不支持搜索），且不是空字符串
          return search && search !== '-' && search !== '' && search.trim() !== ''
        })
      : []
    
    // 从 request 中获取所有字段（表单参数，用于搜索，不需要 search 标签）
    // ⚠️ 关键：request 字段本身就是用于搜索的表单参数，不需要 search 标签
    const requestFields = functionData.request
    const requestAllFields = Array.isArray(requestFields)
      ? requestFields  // 获取所有 request 字段（都是用于搜索的表单参数）
      : []
    
    // 合并：使用 Map 去重，如果同一个字段在两个地方都存在，智能合并
    // ⚠️ 关键：保留 response 字段的 search 信息（如果 request 字段没有 search）
    // 但使用 request 字段的其他配置（如 widget.config，因为可能更完整）
    const fieldMap = new Map<string, FieldConfig>()
    
    // 先添加 response 字段
    responseSearchableFields.forEach(field => {
      fieldMap.set(field.code, field)
    })
    
    // 再添加 request 字段，智能合并
    // ⚠️ 关键：排除 search 为 "-" 的字段（明确表示不支持搜索）
    requestAllFields.forEach(field => {
      // 如果 request 字段的 search 是 "-"，跳过（不显示在搜索表单中）
      if (field.search === '-') {
        return
      }
      
      const existingField = fieldMap.get(field.code)
      if (existingField) {
        // 如果字段已存在（在 response 中），智能合并：
        // 1. 保留 response 的 search 信息（如果 request 没有 search 或 search 是 ""）
        // 2. 使用 request 的其他配置（widget.config 等，因为可能更完整）
        const mergedField: FieldConfig = {
          ...field,  // 使用 request 字段作为基础
          // 优先使用 request 的 search，但如果 request 的 search 是 "" 或 "-"，则使用 response 的
          search: (field.search && field.search !== '-' && field.search !== '') 
            ? field.search 
            : (existingField.search || null),
        }
        fieldMap.set(field.code, mergedField)
      } else {
        // 如果字段不存在，直接添加
        fieldMap.set(field.code, field)
      }
    })
    
    // 返回合并后的字段列表
    return Array.from(fieldMap.values())
  })

  /**
   * 可搜索字段（来自 response，用于 URL 查询参数）
   * 
   * 【说明】
   * - response 字段：针对这个接口的主表的字段
   * - 这些字段会转换为 URL 查询参数格式：`like=remark:测试`、`in=status:待处理,处理中` 等
   * - 必须明确指定 search 值（不能是空字符串、"-"、null、undefined）
   * 
   * 【示例】
   * - `subject` 字段：`search: "like"` → 转换为 `like=subject:测试`
   * - `booker` 字段：`search: "in"` → 转换为 `in=booker:user1,user2`
   */
  const responseSearchableFields = computed(() => {
    // ⚠️ 关键：确保 response 是数组，且 search 有明确值
    return Array.isArray(functionData.response)
      ? functionData.response.filter(field => {
          const search = field.search
          // 必须有值，且不是 "-"（明确表示不支持搜索），且不是空字符串
          return search && search !== '-' && search !== '' && search.trim() !== ''
        })
      : []
  })

  /**
   * 可搜索字段（来自 request，用于请求体）
   * 
   * 【说明】
   * - request 字段：非这张表的参数（扩展字段，如计算字段、外表字段等）
   * - 这些字段会直接作为 `k=v` 形式：`{"room_name": "测试", "status": "进行中"}`
   * - 本身就是用于搜索的表单参数，不需要 search 标签
   * 
   * 【注意】
   * - 此 computed 主要用于区分 request 和 response 字段的处理方式
   * - 实际使用中，request 字段的处理在 `buildSearchParams` 和 `syncToURL` 中直接遍历所有 request 字段
   */
  const requestSearchableFields = computed(() => {
    // ⚠️ 关键：functionData.request 的类型是 any，需要确保它是数组
    // ⚠️ 注意：这里只过滤有 search 标签的字段，但实际上 request 字段不需要 search 标签
    // 此 computed 主要用于向后兼容，实际逻辑在 buildSearchParams 中直接遍历所有 request 字段
    const requestFields = functionData.request
    return Array.isArray(requestFields)
      ? requestFields.filter(field => field.search && field.search !== '-')
      : []
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
   * ==================== 搜索参数构建说明 ====================
   * 
   * 【response 字段处理】
   * - 针对这个接口的主表的字段
   * - 转换为 URL 查询参数格式：`like=remark:测试`、`in=status:待处理,处理中` 等
   * - 使用 search 标签定义的格式（如 `"like"`、`"in"`、`"eq"` 等）
   * 
   * 【request 字段处理】
   * - 非这张表的参数（扩展字段，如计算字段、外表字段等）
   * - 直接作为 `k=v` 形式：`room_name=测试`、`status=进行中`
   * - 不管有没有 search 标签，都作为查询参数或请求体字段
   * 
   * 【请求格式示例】
   * - GET 请求：`?like=remark:测试&room_name=测试&status=进行中&sorts=id:desc`
   * - POST 请求：请求体包含 `{"like": "remark:测试", "room_name": "测试", "status": "进行中", "sorts": "id:desc"}`
   * 
   * 【支持的搜索类型】
   * - 精确匹配(eq)：`eq=id:123`
   * - 模糊查询(like)：`like=subject:测试`
   * - 包含查询(in)：`in=status:待处理,处理中`
   * - 范围查询(gte/lte)：`gte=start_time:1234567890&lte=end_time:1234567890`
   */
  const buildSearchParams = (): SearchParams & Record<string, any> => {
    // ⚠️ 关键：如果同一个字段同时在 request 和 response 中，优先使用 request 的处理方式（k=v 形式）
    // 1. 获取所有 request 字段的 code，用于排除
    const requestFields = functionData.request
    const requestFieldCodes = new Set<string>()
    if (Array.isArray(requestFields)) {
      requestFields.forEach(field => {
        requestFieldCodes.add(field.code)
      })
    }
    
    // 2. 构建 response 字段的搜索参数（URL 查询参数格式，如 `like=remark:测试`）
    // ⚠️ 关键：排除所有 request 字段，避免重复处理
    const responseFieldsForParams = responseSearchableFields.value.filter(
      field => !requestFieldCodes.has(field.code)
    )
    const responseParams = buildSearchParamsString(searchForm.value, responseFieldsForParams)
    
    // 3. 构建 request 字段的搜索参数（直接作为 `k=v` 形式，如 `room_name=测试`）
    // ⚠️ 关键：request 字段不管有没有 search 标签，都直接作为 k=v 形式
    const requestParams: Record<string, any> = {}
    if (Array.isArray(requestFields)) {
      requestFields.forEach(field => {
        const value = searchForm.value[field.code]
        // 检查值是否为空（包括空数组、空字符串、null、undefined）
        if (value !== null && value !== undefined && 
            !(Array.isArray(value) && value.length === 0) && 
            !(typeof value === 'string' && value.trim() === '')) {
          requestParams[field.code] = value
        }
      })
    }
    
    // 4. 合并所有参数
    // 注意：使用 `SearchParams & Record<string, any>` 类型，允许添加任意字段（request 字段）
    // ⚠️ 关键：request 字段会覆盖 response 字段的处理结果（如果同一个字段在两个地方都存在）
    const params: SearchParams & Record<string, any> = {
      page: currentPage.value,
      page_size: pageSize.value,
      ...responseParams,  // response 字段的搜索参数（URL 查询参数格式，如 `like=remark:测试`）
      ...requestParams    // request 字段的搜索参数（直接作为 `k=v` 形式，如 `room_name=测试`）
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
   * 
   * ==================== URL 同步说明 ====================
   * 
   * 【response 字段】
   * - 针对这个接口的主表的字段
   * - 转换为 URL 查询参数格式：`like=remark:测试`、`in=status:待处理,处理中` 等
   * 
   * 【request 字段】
   * - 非这张表的参数（扩展字段，如计算字段、外表字段等）
   * - 直接作为 `k=v` 形式：`room_name=测试`、`status=进行中`
   * 
   * 【URL 格式示例】
   * `?page=1&page_size=20&like=subject:测试&room_name=会议室A&status=进行中&sorts=id:desc`
   */
  const syncToURL = (): void => {
    const query: Record<string, string> = {}
    
    // 🔥 分页参数：始终添加到 URL，即使是默认值也要添加，方便分享和恢复状态
    query.page = String(currentPage.value)
    query.page_size = String(pageSize.value)
    
    // 🔥 排序参数：始终添加到 URL（如果有排序的话）
    const finalSorts = sorts.value.length > 0 
      ? sorts.value 
      : (hasManualSort.value ? [] : buildDefaultSorts())
    
    if (finalSorts.length > 0) {
      query.sorts = finalSorts.map(item => `${item.field}:${item.order}`).join(',')
    }
    // 🔥 关键：如果排序为空，不设置 query.sorts，这样在后续处理中会从 URL 中删除
    
    // ==================== 搜索参数同步到 URL ====================
    // 
    // 【response 字段处理】
    // - 针对这个接口的主表的字段
    // - 使用 buildURLSearchParams 处理，产生 `like=remark:测试` 格式
    // 
    // 【request 字段处理】
    // - 非这张表的参数（扩展字段，如计算字段、外表字段等）
    // - 直接作为 `k=v` 形式，产生 `status=进行中` 格式
    // 
    // ⚠️ 重要：如果同一个字段同时在 request 和 response 中，优先使用 request 的处理方式（k=v 形式）
    
    // 1. 获取所有 request 字段的 code，用于排除
    const requestFields = functionData.request
    const requestFieldCodes = new Set<string>()
    if (Array.isArray(requestFields)) {
      requestFields.forEach(field => {
        requestFieldCodes.add(field.code)
      })
    }
    
    // 2. response 字段的搜索参数（URL 查询参数格式，如 `like=remark:测试`）
    // ⚠️ 关键：排除所有 request 字段，避免重复处理
    const responseFieldsForURL = responseSearchableFields.value.filter(
      field => !requestFieldCodes.has(field.code)
    )
    Object.assign(query, buildURLSearchParams(searchForm.value, responseFieldsForURL))
    
    // 3. request 字段的搜索参数（直接作为 `k=v` 形式，如 `status=进行中`）
    // ⚠️ 关键：request 字段会覆盖 response 字段的处理结果（如果同一个字段在两个地方都存在）
    if (Array.isArray(requestFields)) {
      requestFields.forEach(field => {
        const value = searchForm.value[field.code]
        // 检查值是否为空（包括空数组、空字符串、null、undefined）
        if (value !== null && value !== undefined && 
            !(Array.isArray(value) && value.length === 0) && 
            !(typeof value === 'string' && value.trim() === '')) {
          // 直接作为 k=v 形式添加到 URL 查询参数
          // ⚠️ 这会覆盖 response 字段的处理结果（如果同一个字段在两个地方都存在）
          query[field.code] = Array.isArray(value) ? value.join(',') : String(value)
        }
      })
    }
    
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
    
    // ⚠️ 注意：requestFieldCodes 已经在上面声明过了，这里直接使用
    
    // 🔥 先复制所有非搜索参数（分页、排序等），但排除：
    // 1. searchParamKeys（response 字段的搜索参数，如 eq, like, in 等）
    // 2. sorts（因为我们要根据当前状态决定是否保留）
    // 3. request 字段（因为我们要根据当前状态决定是否保留，如果已清空则删除）
    // 4. 🔥 保留以 _ 开头的参数（前端状态参数，如 _detail_id, _detail_function_id），这些参数不会被删除
    Object.keys(route.query).forEach(key => {
      // 🔥 保留以 _ 开头的参数（前端状态参数）
      if (key.startsWith('_')) {
        newQuery[key] = String(route.query[key])
      } else if (!searchParamKeys.includes(key) && key !== 'sorts' && !requestFieldCodes.has(key)) {
        newQuery[key] = String(route.query[key])
      }
    })
    
    // 🔥 然后添加新的参数（包括排序和搜索）
    // 如果 query 中有 sorts，会添加；如果没有，则不会添加（从而从 URL 中删除）
    // 如果 query 中有 request 字段，会添加；如果没有，则不会添加（从而从 URL 中删除）
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
   * 4. 🔥 只恢复属于当前函数的字段，避免数据污染
   * 
   * URL 格式示例：
   * - 单个字段：gte=progress:50&lte=progress:80
   * - 多个字段：gte=progress:50,score:5&lte=progress:80,score:8
   */
  const restoreFromURL = (): void => {
    const query = route.query
    
    // 🔥 获取当前函数的所有字段 code，用于验证 URL 参数是否属于当前函数
    const currentRequestFieldCodes = new Set<string>()
    const currentResponseFieldCodes = new Set<string>()
    
    if (Array.isArray(functionData.request)) {
      functionData.request.forEach(field => {
        currentRequestFieldCodes.add(field.code)
      })
    }
    
    if (Array.isArray(functionData.response)) {
      functionData.response.forEach(field => {
        currentResponseFieldCodes.add(field.code)
      })
    }
    
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
    
    // 恢复排序（只恢复属于当前函数的字段）
    if (query.sorts) {
      const sortsString = String(query.sorts)
      const sortItems: SortItem[] = []
      sortsString.split(',').forEach(sortStr => {
        const parts = sortStr.trim().split(':')
        if (parts.length === 2) {
          const field = parts[0] || ''
          const order = parts[1] as 'asc' | 'desc'
          // 🔥 只恢复属于当前函数的字段
          if (field && (order === 'asc' || order === 'desc') && 
              (currentRequestFieldCodes.has(field) || currentResponseFieldCodes.has(field))) {
            sortItems.push({ field, order })
          }
        }
      })
      if (sortItems.length > 0) {
        sorts.value = sortItems
        hasManualSort.value = true
      }
    }
    
    // ==================== 从 URL 恢复搜索条件 ====================
    // 
    // 【response 字段恢复】
    // - 针对这个接口的主表的字段
    // - 从 URL 查询参数中解析：`like=remark:测试` → 恢复为 `searchForm.remark = "测试"`
    // - 支持多个字段同时使用相同的搜索类型：`like=subject:测试,remark:备注`
    // 
    // 【request 字段恢复】
    // - 非这张表的参数（扩展字段，如计算字段、外表字段等）
    // - 直接从 URL 查询参数中读取：`room_name=测试` → 恢复为 `searchForm.room_name = "测试"`
    
    // 1. 恢复 request 字段（直接从 URL 查询参数中读取，k=v 形式）
    // 🔥 只恢复属于当前函数的字段
    const requestFields = functionData.request
    if (Array.isArray(requestFields)) {
      requestFields.forEach(field => {
        // 🔥 验证字段是否属于当前函数（双重检查，确保安全）
        if (!currentRequestFieldCodes.has(field.code)) {
          return
        }
        const value = query[field.code]
        if (value !== undefined && value !== null && value !== '') {
          // 直接使用 URL 中的值
          searchForm.value[field.code] = String(value)
        }
      })
    }
    
    // 2. 恢复 response 字段（从 URL 查询参数中解析，格式：eq=field:value, like=field:value 等）
    // 格式：eq=field:value 或 eq=field1:value1,field2:value2, like=field:value, in=field:value, gte=field:value, lte=field:value
    // 🔥 支持多个字段使用相同搜索类型，格式：field1:value1,field2:value2
    // 🔥 只恢复属于当前函数的字段，避免数据污染
    responseSearchableFields.value.forEach(field => {
      // 🔥 验证字段是否属于当前函数（双重检查，确保安全）
      if (!currentResponseFieldCodes.has(field.code)) {
        return
      }
      
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
    // 🔥 初始化后同步状态到 URL（确保即使 URL 是干净的，也会将当前状态同步到 URL）
    syncToURL()
    // 🔥 初始化后加载数据
    loadTableData()
  }
  
  // 初始化（在首次创建时）
  initialize()
  
  // 监听 URL 变化，恢复状态（避免循环更新）
  let isRestoringFromURL = false
  let isSyncingToURL = false
  watch(() => route.query, () => {
    // 🔥 如果正在同步到 URL，跳过（避免循环）
    if (isSyncingToURL) return
    // 🔥 如果正在从 URL 恢复，跳过（避免循环）
    if (isRestoringFromURL) return
    
    isRestoringFromURL = true
    restoreFromURL()
    // 🔥 如果 URL 是干净的（没有查询参数），恢复默认状态后同步到 URL
    const hasQueryParams = Object.keys(route.query).length > 0
    if (!hasQueryParams) {
      isSyncingToURL = true
      nextTick(() => {
        syncToURL()
        isSyncingToURL = false
      })
    }
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
      // 🔥 使用 ElNotification 显示更漂亮的提示
      ElNotification({
        title: '新增成功',
        message: '记录已成功添加',
        type: 'success',
        duration: 3000,
        position: 'top-right'
      })
      await loadTableData()
      return true
    } catch (error: any) {
      // 🔥 优先使用后端返回的错误信息
      const errorMessage = error?.response?.data?.msg 
        || error?.response?.data?.message 
        || error?.message 
        || '新增失败'
      ElMessage.error(errorMessage)
      return false
    }
  }
  
  /**
   * 更新记录
   * @param id 记录 ID
   * @param data 更新的数据（新值）
   * @param oldData 旧数据（用于对比，找出变更的字段）
   * @returns 是否成功
   */
  const handleUpdate = async (id: number, data: Record<string, any>, oldData?: Record<string, any>): Promise<boolean> => {
    try {
      // ⚠️ 关键：如果提供了 oldData，只传递变更的字段
      // 格式：{"id": 2, "updates": {"name": "802"}, "old_values": {"name": "801"}}
      let updateData: Record<string, any>
      
      if (oldData) {
        // 对比旧值和新值，找出变更的字段
        const { updates, oldValues } = getChangedFields(oldData, data)
        
        updateData = {
          id,              // ID 单独传递（用于明确标识要更新的记录）
          updates,         // 只包含变更的字段（可以包含 id，但 GORM 会自动忽略 id）
          old_values: oldValues  // 变更字段的旧值（用于审计）
        }
      } else {
        // 向后兼容：如果没有提供 oldData，传递全量数据（旧版本行为）
        // 注意：这种情况下，Updates 可能包含 id，后端会处理
        updateData = {
          id,
          ...data
        }
      }
      
      await tableUpdateRow(functionData.method, functionData.router, updateData)
      // 🔥 使用 ElNotification 显示更漂亮的提示
      ElNotification({
        title: '更新成功',
        message: '记录已成功更新',
        type: 'success',
        duration: 3000,
        position: 'top-right'
      })
      await loadTableData()
      return true
    } catch (error: any) {
      // 🔥 优先使用后端返回的错误信息
      const errorMessage = error?.response?.data?.msg 
        || error?.response?.data?.message 
        || error?.message 
        || '更新失败'
      // 🔥 使用 ElNotification 显示更漂亮的错误提示
      ElNotification({
        title: '更新失败',
        message: errorMessage,
        type: 'error',
        duration: 5000,
        position: 'top-right'
      })
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
      // 🔥 使用 ElNotification 显示更漂亮的提示
      ElNotification({
        title: '删除成功',
        message: '记录已成功删除',
        type: 'success',
        duration: 3000,
        position: 'top-right'
      })
      await loadTableData()
      return true
    } catch (error: any) {
      if (error !== 'cancel') {
        // 🔥 优先使用后端返回的错误信息
        const errorMessage = error?.response?.data?.msg 
          || error?.response?.data?.message 
          || error?.message 
          || '删除失败'
        // 🔥 使用 ElNotification 显示更漂亮的错误提示
        ElNotification({
          title: '删除失败',
          message: errorMessage,
          type: 'error',
          duration: 5000,
          position: 'top-right'
        })
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
    hasManualSort,
    
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

