/**
 * TableDomainService - 表格领域服务
 * 
 * 职责：表格相关的业务逻辑
 * - 加载表格数据
 * - 搜索、分页、排序
 * - CRUD 操作（新增、编辑、删除）
 * 
 * 特点：
 * - 依赖接口，不依赖具体实现
 * - 通过事件总线通信
 * - 通过状态管理器管理状态
 */

import type { IApiClient } from '../interfaces/IApiClient'
import type { IStateManager } from '../interfaces/IStateManager'
import type { IEventBus } from '../interfaces/IEventBus'
import { TableEvent } from '../interfaces/IEventBus'
import type { FunctionDetail, FieldConfig } from '../types'
import { getChangedFields } from '@/utils/objectDiff'
import { buildSearchParamsString } from '@/utils/searchParams'
import { denormalizeSearchValue } from '@/utils/searchValueNormalizer'
import { parseCommaSeparatedString } from '@/utils/stringUtils'
import { SearchType } from '@/core/constants/search'
import { WidgetType } from '@/core/constants/widget'
import { getScopedFieldQueryValue } from '@/utils/queryFieldNamespace'

/**
 * 表格数据项类型
 */
export interface TableRow {
  id: number | string
  [key: string]: any
}

/**
 * 表格响应类型
 */
export interface TableResponse {
  items: TableRow[]
  paginated?: {
    current_page: number
    page_size: number
    total_count: number
    total_pages: number
  }
}

/**
 * 搜索参数类型
 */
export interface SearchParams {
  [key: string]: any
}

/**
 * 排序参数类型
 */
export interface SortParams {
  field: string
  order: 'asc' | 'desc'
}

/**
 * 排序项类型
 */
export interface SortItem {
  field: string
  order: 'asc' | 'desc'
}

/**
 * 表格状态
 */
export interface TableState {
  data: TableRow[]
  loading: boolean
  searchParams: SearchParams
  searchForm: Record<string, any> // 🔥 新增：搜索表单数据（用于 UI 绑定）
  sortParams: SortParams | null
  sorts: SortItem[] // 🔥 新增：排序列表（支持多列排序）
  hasManualSort: boolean // 🔥 新增：是否手动排序
  pagination: {
    currentPage: number
    pageSize: number
    total: number
  }
}

/**
 * 表格数据加载生命周期钩子接口
 * 🔥 类似 GORM 的 Before/After 钩子，提供清晰的生命周期回调
 * 
 * 生命周期阶段：BeforeRender（渲染前）
 * - 执行时机：数据加载完成后、状态更新前、界面渲染前
 * - 目的：在渲染前预加载关联数据（用户信息、部门信息等），避免渲染时再发起请求
 * 
 * 执行流程：
 * 1. 调用 API 加载表格数据
 * 2. 🔥 BeforeRender 钩子执行（预加载关联数据）
 * 3. 更新状态（触发 Vue 响应式更新）
 * 4. 界面渲染（此时关联数据已在缓存中）
 * 
 * 使用场景：
 * - 用户信息预加载
 * - 部门信息预加载
 * - 其他关联数据预加载
 * 
 * 执行顺序：按照 priority 从小到大执行（priority 越小越早执行）
 */
export interface TableDataHook {
  /** 钩子名称（用于调试和日志） */
  name: string
  /** 优先级（越小越早执行，建议范围：0-1000） */
  priority: number
  /** 执行钩子的函数 */
  execute: (functionDetail: FunctionDetail, tableData: TableRow[]) => Promise<void>
}

/**
 * 表格领域服务
 */
export class TableDomainService {
  /** 🔥 BeforeRender 钩子列表（按优先级排序） */
  private beforeRenderHooks: TableDataHook[] = []
  /** 只允许最后一次列表请求更新状态，避免旧请求回写新状态 */
  private latestLoadRequestId = 0

  constructor(
    private apiClient: IApiClient,
    private stateManager: IStateManager<TableState>,
    private eventBus: IEventBus
  ) {}

  /**
   * 🔥 注册 BeforeRender 钩子（渲染前执行）
   * 
   * 执行时机：数据加载完成后、状态更新前、界面渲染前
   * 目的：在渲染前预加载关联数据，避免渲染时再发起请求
   * 
   * @param hook 生命周期钩子
   * 
   * 示例：
   * ```typescript
   * domainService.beforeRender({
   *   name: 'preload-user-info',
   *   priority: 100,
   *   execute: async (functionDetail, tableData) => {
   *     // 预加载用户信息
   *   }
   * })
   * ```
   */
  beforeRender(hook: TableDataHook): void {
    // 移除同名的旧钩子（允许更新）
    this.beforeRenderHooks = this.beforeRenderHooks.filter(h => h.name !== hook.name)
    // 添加新钩子
    this.beforeRenderHooks.push(hook)
    // 按优先级排序
    this.beforeRenderHooks.sort((a, b) => a.priority - b.priority)
  }

  /**
   * 🔥 移除 BeforeRender 钩子
   * @param name 钩子名称
   */
  removeBeforeRenderHook(name: string): void {
    this.beforeRenderHooks = this.beforeRenderHooks.filter(h => h.name !== name)
  }

  /**
   * 🔥 获取所有 BeforeRender 钩子（用于调试）
   */
  getBeforeRenderHooks(): TableDataHook[] {
    return [...this.beforeRenderHooks]
  }

  /**
   * 加载表格数据
   */
  async loadData(functionDetail: FunctionDetail, searchParams?: SearchParams, sortParams?: SortParams, pagination?: { page: number, pageSize: number }): Promise<TableResponse> {
    const requestId = ++this.latestLoadRequestId
    const state = this.stateManager.getState()
    
    // 更新加载状态
    this.stateManager.setState({
      ...state,
      loading: true
    })

    try {
      // 构建请求参数
      const params: any = {
        ...(searchParams || state.searchParams),
        ...(pagination ? {
          page: pagination.page,
          page_size: pagination.pageSize
        } : {
          page: state.pagination.currentPage,
          page_size: state.pagination.pageSize
        })
      }

      // 添加排序参数
      // 🔥 优先使用 state.sorts（支持多列排序），如果没有则使用 sortParams（单个排序）
      if (state.sorts && state.sorts.length > 0) {
        // 支持多列排序：sorts=field1:order1,field2:order2
        params.sorts = state.sorts.map(item => `${item.field}:${item.order}`).join(',')
      } else if (sortParams || state.sortParams) {
        // 兼容单个排序参数
        const sort = sortParams || state.sortParams!
        params.sorts = `${sort.field}:${sort.order}`
      }

      // ⭐ 使用标准 API：/table/search/{full-code-path}
      const fullCodePath = functionDetail.router?.startsWith('/') 
        ? functionDetail.router 
        : `/${functionDetail.router || ''}`
      const url = `/workspace/api/v1/table/search${fullCodePath}`
      
      // Table 查询统一使用 GET 方法
      const response = await this.apiClient.get<TableResponse>(url, params)

      if (requestId !== this.latestLoadRequestId) {
        return response
      }
      
      // ⭐ 旧版本（已注释，保留用于参考）
      // const url = `/workspace/api/v1/run${functionDetail.router}`
      // const method = functionDetail.method?.toUpperCase() || 'GET'
      // let response: TableResponse
      // if (method === 'GET') {
      //   response = await this.apiClient.get<TableResponse>(url, params)
      // } else {
      //   response = await this.apiClient.post<TableResponse>(url, params)
      // }

      // 🔥 BeforeRender: 在数据加载完成后、状态更新前、界面渲染前执行所有钩子
      // 这样渲染时，所有关联数据（用户信息、部门信息等）都已经在缓存中
      // 按照优先级顺序执行
      for (const hook of this.beforeRenderHooks) {
        try {
          await hook.execute(functionDetail, response.items || [])
        } catch (error) {
          // 单个钩子失败不影响其他钩子执行
          console.error(`[TableDomainService] BeforeRender 钩子 ${hook.name} 执行失败`, error)
        }
      }

      if (requestId !== this.latestLoadRequestId) {
        return response
      }

      // 更新状态
      const latestState = this.stateManager.getState()
      this.stateManager.setState({
        ...latestState,
        data: response.items || [],
        loading: false,
        searchParams: searchParams || latestState.searchParams,
        sortParams: sortParams || latestState.sortParams,
        pagination: {
          ...latestState.pagination,
          currentPage: response.paginated?.current_page || latestState.pagination.currentPage,
          pageSize: response.paginated?.page_size || latestState.pagination.pageSize,
          total: response.paginated?.total_count || 0
        }
      })

      // 触发事件
      this.eventBus.emit(TableEvent.dataLoaded, { data: response.items, pagination: response.paginated })

      return response
    } catch (error) {
      // 更新加载状态
      if (requestId === this.latestLoadRequestId) {
        const latestState = this.stateManager.getState()
        this.stateManager.setState({
          ...latestState,
          loading: false
        })
      }
      throw error
    }
  }

  /**
   * 更新搜索参数
   */
  updateSearchParams(searchParams: SearchParams): void {
    const state = this.stateManager.getState()
    
    this.stateManager.setState({
      ...state,
      searchParams: { ...state.searchParams, ...searchParams }
    })

    // 触发事件
    this.eventBus.emit(TableEvent.searchChanged, { searchParams })
  }

  /**
   * 更新排序参数
   */
  updateSortParams(sortParams: SortParams): void {
    const state = this.stateManager.getState()
    
    this.stateManager.setState({
      ...state,
      sortParams
    })

    // 触发事件
    this.eventBus.emit(TableEvent.sortChanged, { sortParams })
  }

  /**
   * 更新分页参数
   */
  updatePagination(page: number, pageSize: number): void {
    const state = this.stateManager.getState()
    
    this.stateManager.setState({
      ...state,
      pagination: {
        ...state.pagination,
        currentPage: page,
        pageSize
      }
    })

    // 触发事件
    this.eventBus.emit(TableEvent.pageChanged, { page, pageSize })
  }

  /**
   * 新增行
   */
  async addRow(functionDetail: FunctionDetail, data: Record<string, any>): Promise<TableRow> {
    const router = this.requireFunctionRouter(functionDetail)
    // ⭐ 使用标准 API：POST /workspace/api/v1/table/create/{full-code-path}
    const fullCodePath = router.startsWith('/')
      ? router
      : `/${router}`
    const url = `/workspace/api/v1/table/create${fullCodePath}`
    const response = await this.apiClient.post<TableRow>(url, data)

    // 触发事件
    this.eventBus.emit(TableEvent.rowAdded, { row: response })

    return response
  }

  /**
   * 更新行
   */
  async updateRow(
    functionDetail: FunctionDetail,
    id: number | string,
    data: Record<string, any>,
    oldData?: Record<string, any>
  ): Promise<TableRow> {
    const router = this.requireFunctionRouter(functionDetail)
    // ⭐ 使用标准 API：PUT /workspace/api/v1/table/update/{full-code-path}
    const fullCodePath = router.startsWith('/') 
      ? router
      : `/${router}`
    const url = `/workspace/api/v1/table/update${fullCodePath}`
    
    // 构建更新负载
    const payload = this.buildUpdatePayload(id, data, oldData)
    
    // 使用 PUT 方法调用新接口
    const response = await this.apiClient.put<TableRow>(url, payload)

    // 触发事件
    this.eventBus.emit(TableEvent.rowUpdated, { id, row: response })

    return response
  }

  /**
   * 删除行
   */
  async deleteRow(functionDetail: FunctionDetail, id: number | string): Promise<void> {
    const router = this.requireFunctionRouter(functionDetail)
    // ⭐ 使用标准 API：DELETE /workspace/api/v1/table/delete/{full-code-path}
    const fullCodePath = router.startsWith('/')
      ? router
      : `/${router}`
    const url = `/workspace/api/v1/table/delete${fullCodePath}`
    const ids = [typeof id === 'string' ? parseInt(id, 10) : id]
    await this.apiClient.delete<void>(url, { ids })

    // 触发事件
    this.eventBus.emit(TableEvent.rowDeleted, { ids: [id] })
  }

  /**
   * 获取表格数据
   */
  getData(): TableRow[] {
    return this.stateManager.getState().data
  }

  /**
   * 获取加载状态
   */
  isLoading(): boolean {
    return this.stateManager.getState().loading
  }

  /**
   * 获取分页信息
   */
  getPagination() {
    return this.stateManager.getState().pagination
  }

  /**
   * 获取可搜索字段（遵循依赖倒置原则，业务逻辑在 Domain Layer）
   */
  getSearchableFields(functionDetail: FunctionDetail): FieldConfig[] {
    const response = Array.isArray(functionDetail.response) ? functionDetail.response : []
    const request = Array.isArray(functionDetail.request) ? functionDetail.request : []
    
    // 从 response 中获取可搜索字段（主表字段，必须有明确的 search 标签值）
    const responseSearchableFields = response.filter((field: FieldConfig) => {
      const search = field.search
      return search && search !== '-' && search !== '' && search.trim() !== ''
    })
    
    // 从 request 中获取所有字段（扩展字段，用于搜索，不需要 search 标签）
    const requestAllFields = request.filter((field: FieldConfig) => {
      return field.search !== '-' // 排除明确表示不支持搜索的字段
    })
    
    // 合并：使用 Map 去重
    const fieldMap = new Map<string, FieldConfig>()
    responseSearchableFields.forEach((field: FieldConfig) => {
      fieldMap.set(field.code, field)
    })
    requestAllFields.forEach((field: FieldConfig) => {
      const existingField = fieldMap.get(field.code)
      if (existingField) {
        // 智能合并
        const mergedField: FieldConfig = {
          ...field,
          search: (field.search && field.search !== '-' && field.search !== '') 
            ? field.search 
            : (existingField.search || undefined),
        }
        fieldMap.set(field.code, mergedField)
      } else {
        fieldMap.set(field.code, field)
      }
    })
    
    return Array.from(fieldMap.values())
  }

  /**
   * 从 URL 恢复状态（遵循依赖倒置原则，业务逻辑在 Domain Layer）
   * 
   * @param functionDetail 函数详情
   * @param query URL 查询参数
   * @returns 恢复后的状态 { searchForm, sorts, pagination }
   */
  restoreFromURL(
    functionDetail: FunctionDetail,
    query: Record<string, string | string[]>
  ): {
    searchForm: Record<string, any>
    sorts: Array<{ field: string; order: 'asc' | 'desc' }>
    pagination: { page: number; pageSize: number }
  } {
    
    const searchForm: Record<string, any> = {}
    const sorts: Array<{ field: string; order: 'asc' | 'desc' }> = []
    
    // 获取当前函数的所有字段 code
    const requestFields = Array.isArray(functionDetail.request) ? functionDetail.request : []
    const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
    
    const currentRequestFieldCodes = new Set<string>()
    const currentResponseFieldCodes = new Set<string>()
    
    requestFields.forEach((field: FieldConfig) => {
      currentRequestFieldCodes.add(field.code)
    })
    responseFields.forEach((field: FieldConfig) => {
      currentResponseFieldCodes.add(field.code)
    })
    
    // 恢复分页
    let page = 1
    let pageSize = 20
    if (query.page) {
      const pageNum = parseInt(String(query.page), 10)
      if (!isNaN(pageNum) && pageNum > 0) {
        page = pageNum
      }
    }
    if (query.page_size) {
      const size = parseInt(String(query.page_size), 10)
      if (!isNaN(size) && size > 0) {
        pageSize = size
      }
    }
    
    // 恢复排序
    if (query.sorts) {
      const sortsString = String(query.sorts)
      sortsString.split(',').forEach((sortStr: string) => {
        const parts = sortStr.trim().split(':')
        if (parts.length === 2) {
          const field = parts[0] || ''
          const order = parts[1] as 'asc' | 'desc'
          if (field && (order === 'asc' || order === 'desc') && 
              (currentRequestFieldCodes.has(field) || currentResponseFieldCodes.has(field))) {
            sorts.push({ field, order })
          }
        }
      })
    }
    
    // 恢复搜索条件（request 字段）
    requestFields.forEach((field: FieldConfig) => {
      if (!currentRequestFieldCodes.has(field.code)) return
      const value = getScopedFieldQueryValue(query, field.code, 'search')
      if (value !== undefined && value !== null && value !== '') {
        searchForm[field.code] = String(value)
      }
    })
    
    // 恢复搜索条件（response 字段）
    const responseSearchableFields = responseFields.filter((field: FieldConfig) => {
      const search = field.search
      return search && search !== '-' && search !== '' && search.trim() !== ''
    })
    
    const searchableFields = this.getSearchableFields(functionDetail)
    
    responseSearchableFields.forEach((field: FieldConfig) => {
      if (!currentResponseFieldCodes.has(field.code)) return
      
      const searchType = field.search || ''
      
      if (searchType.includes(SearchType.EQ)) {
        const eqValue = query.eq
        if (eqValue) {
          const eqStr = String(eqValue)
          const parts = eqStr.split(',')
          for (const part of parts) {
            if (part.trim().startsWith(`${field.code}:`)) {
              const value = part.trim().substring(field.code.length + 1)
              if (value) {
                const denormalizedValue = denormalizeSearchValue(value, {
                  widgetType: field.widget?.type,
                  searchType: field.search,
                  field
                })
                searchForm[field.code] = denormalizedValue
                break
              }
            }
          }
        }
      } else if (searchType.includes(SearchType.LIKE)) {
        const likeValue = query.like
        if (likeValue) {
          const likeStr = String(likeValue)
          const parts = likeStr.split(',')
          for (const part of parts) {
            if (part.trim().startsWith(`${field.code}:`)) {
              const value = part.trim().substring(field.code.length + 1)
              if (value) {
                searchForm[field.code] = value
                break
              }
            }
          }
        }
      } else if (searchType.includes(SearchType.CONTAINS)) {
        const containsValue = query.contains
        if (containsValue) {
          const containsStr = String(containsValue)
          const fieldPrefix = `${field.code}:`
          const fieldIndex = containsStr.indexOf(fieldPrefix)
          if (fieldIndex >= 0) {
            const valueStart = fieldIndex + fieldPrefix.length
            let valueEnd = containsStr.length
            const allFieldCodes = searchableFields.map((f: FieldConfig) => f.code)
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
              const values = parseCommaSeparatedString(valueStr)
              if (field.widget?.type === WidgetType.MULTI_SELECT) {
                searchForm[field.code] = values.length > 0 ? values : []
              } else {
                searchForm[field.code] = values.length > 1 ? values : (values.length === 1 ? values[0] : valueStr)
              }
            }
          }
        }
      } else if (searchType.includes(SearchType.IN)) {
        const inValue = query.in
        if (inValue) {
          const inStr = String(inValue)
          const fieldPrefix = `${field.code}:`
          const fieldIndex = inStr.indexOf(fieldPrefix)
          if (fieldIndex >= 0) {
            const valueStart = fieldIndex + fieldPrefix.length
            let valueEnd = inStr.length
            const allFieldCodes = searchableFields.map((f: FieldConfig) => f.code)
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
            const valueStr = inStr.substring(valueStart, valueEnd).trim()
            if (valueStr) {
              const values = parseCommaSeparatedString(valueStr)
              if ((field.widget?.type === WidgetType.USER || field.widget?.type === WidgetType.MULTI_SELECT) && searchType.includes(SearchType.IN)) {
                searchForm[field.code] = values.length > 0 ? values : []
              } else {
                searchForm[field.code] = values.length > 1 ? values : (values.length === 1 ? values[0] : valueStr)
              }
            }
          }
        }
      } else if (searchType.includes(SearchType.GTE) && searchType.includes(SearchType.LTE)) {
        const gteValue = query.gte
        const lteValue = query.lte
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
          const fieldType = field.data?.type
          const widgetType = field.widget?.type
          const isTimestamp = fieldType === 'timestamp' || widgetType === 'timestamp'
          if (isTimestamp) {
            const SECONDS_THRESHOLD = 9999999999
            const convertTimestamp = (ts: string | null): number | null => {
              if (!ts) return null
              const num = Number(ts)
              if (num > 0 && num < SECONDS_THRESHOLD) {
                return num * 1000
              }
              return num
            }
            const timestampRange = [
              gte ? convertTimestamp(gte) : null,
              lte ? convertTimestamp(lte) : null
            ]
            searchForm[field.code] = timestampRange
          } else {
            searchForm[field.code] = {
              min: gte ? String(gte) : undefined,
              max: lte ? String(lte) : undefined
            }
          }
        }
      }
    })
    
    return {
      searchForm,
      sorts,
      pagination: { page, pageSize }
    }
  }

  /**
   * 构建搜索参数（遵循依赖倒置原则，业务逻辑在 Domain Layer）
   */
  buildSearchParams(functionDetail: FunctionDetail, searchForm: Record<string, any>): SearchParams {
    const searchParams: SearchParams = {}
    
    // response 字段的搜索参数
    const response = Array.isArray(functionDetail.response) ? functionDetail.response : []
    const request = Array.isArray(functionDetail.request) ? functionDetail.request : []
    
    const responseFields = response.filter((field: FieldConfig) => {
      const search = field.search
      return search && search !== '-' && search !== '' && search.trim() !== ''
    })
    
    const requestFieldCodes = new Set<string>()
    request.forEach((field: FieldConfig) => {
      requestFieldCodes.add(field.code)
    })
    
    const responseFieldsForParams = responseFields.filter(
      (field: FieldConfig) => !requestFieldCodes.has(field.code)
    )
    
    // 使用工具函数构建 response 字段的搜索参数
    Object.assign(searchParams, buildSearchParamsString(searchForm, responseFieldsForParams))
    
    // request 字段的搜索参数
    request.forEach((field: FieldConfig) => {
      const value = searchForm[field.code]
      if (value !== null && value !== undefined && 
          !(Array.isArray(value) && value.length === 0) && 
          !(typeof value === 'string' && value.trim() === '')) {
        searchParams[field.code] = value
      }
    })
    
    return searchParams
  }

  private buildUpdatePayload(
    id: number | string,
    newData: Record<string, any>,
    oldData?: Record<string, any>
  ): Record<string, any> {
    if (oldData) {
      const { updates, oldValues } = getChangedFields(oldData, newData)
      return {
        id,
        updates,
        old_values: oldValues
      }
    }

    return {
      id,
      ...newData
    }
  }

  private requireFunctionRouter(functionDetail: FunctionDetail): string {
    const router = functionDetail.router
    if (!router) {
      throw new Error('函数路由不存在')
    }
    return router
  }
}
