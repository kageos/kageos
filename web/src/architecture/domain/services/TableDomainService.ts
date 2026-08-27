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

import type { IStateManager } from '../interfaces/IStateManager'
import type { IEventBus } from '../interfaces/IEventBus'
import { TableEvent } from '../interfaces/IEventBus'
import type { ITableGateway } from '../interfaces/ITableGateway'
import type {
  FieldConfig,
  FunctionDetail,
  TableSearchParams as SearchParams,
  SortItem,
  SortParams,
  TableDataHook,
  TableListResponse as TableResponse,
  TableRow,
  TableState
} from '../types'
export type {
  TableSearchParams as SearchParams,
  TableListResponse as TableResponse,
  SortItem,
  SortParams,
  TableDataHook,
  TableRow,
  TableState
} from '../types'
import {
  getTableListFields,
  getTableRequestFields,
  getTableRequestSearchFields,
  getTableSearchFields
} from '@/architecture/domain/utils/functionSchemaSelectors'
import { Logger } from '@/architecture/shared/logger'
import { getSearchFieldRawValue } from '@/architecture/domain/utils/searchFieldValue'

const normalizeSortItemOrder = (order: string | undefined): 'asc' | 'desc' | null => {
  if (order === 'asc' || order === 'desc') {
    return order
  }
  return null
}

const serializeSortsForRequest = (items: Array<SortItem | SortParams>): string => {
  return JSON.stringify(items.map(item => ({
    field: item.field,
    order: item.order
  })))
}

const parseSortsFromRequest = (sortsString: string): SortItem[] => {
  const trimmed = sortsString.trim()
  if (!trimmed) {
    return []
  }

  try {
    const items = JSON.parse(trimmed)
    if (!Array.isArray(items)) {
      return []
    }
    return items
      .map(item => {
        const field = typeof item?.field === 'string' ? item.field.trim() : ''
        const order = normalizeSortItemOrder(typeof item?.order === 'string' ? item.order.trim() : '')
        return field && order ? { field, order } : null
      })
      .filter((item): item is SortItem => Boolean(item))
  } catch {
    return []
  }
}

const isAllowedSortField = (
  sort: SortItem,
  requestFieldCodes: Set<string>,
  responseFieldCodes: Set<string>
): boolean => {
  return requestFieldCodes.has(sort.field) || responseFieldCodes.has(sort.field)
}

type TableRequestParams = SearchParams & {
  page: number
  page_size: number
  sorts?: string
}

export interface TableDataSnapshot {
  rows: TableRow[]
  total: number
  truncated: boolean
}

/**
 * 表格领域服务
 */
export class TableDomainService {
  /** 🔥 BeforeRender 钩子列表（按优先级排序） */
  private beforeRenderHooks: TableDataHook[] = []
  /** 只允许最后一次列表请求更新状态，避免过期请求回写新状态 */
  private latestLoadRequestId = 0

  constructor(
    private tableGateway: ITableGateway,
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
    // 移除同名钩子（允许更新）
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
      const params: TableRequestParams = {
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
        params.sorts = serializeSortsForRequest(state.sorts)
      } else if (sortParams || state.sortParams) {
        const sort = sortParams || state.sortParams!
        params.sorts = serializeSortsForRequest([sort])
      }

      const response = await this.tableGateway.loadRows({ functionDetail, params })

      if (requestId !== this.latestLoadRequestId) {
        return response
      }

      // 🔥 BeforeRender: 在数据加载完成后、状态更新前、界面渲染前执行所有钩子
      // 这样渲染时，所有关联数据（用户信息、部门信息等）都已经在缓存中
      // 按照优先级顺序执行
      for (const hook of this.beforeRenderHooks) {
        try {
          await hook.execute(functionDetail, response.items || [])
        } catch (error) {
          // 单个钩子失败不影响其他钩子执行
          Logger.error('TableDomainService', `BeforeRender 钩子 ${hook.name} 执行失败`, error)
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
   * 读取当前筛选条件下的数据快照，不改变列表的加载状态、分页或当前数据。
   * 用于导出等旁路操作，避免用户导出后被强制跳页。
   */
  async loadDataSnapshot(
    functionDetail: FunctionDetail,
    options: { startRow?: number, maxRows?: number, pageSize?: number, sorts?: SortItem[] } = {}
  ): Promise<TableDataSnapshot> {
    const state = this.stateManager.getState()
    const startRow = Math.max(0, Math.floor(options.startRow ?? 0))
    const maxRows = Math.max(1, options.maxRows ?? 10000)
    const pageSize = Math.min(500, Math.max(1, options.pageSize ?? 500))
    const rows: TableRow[] = []
    let page = Math.floor(startRow / pageSize) + 1
    let pageOffset = startRow % pageSize
    let total = 0

    while (rows.length < maxRows) {
      const params: TableRequestParams = {
        ...state.searchParams,
        page,
        page_size: pageSize
      }

      if (options.sorts && options.sorts.length > 0) {
        params.sorts = serializeSortsForRequest(options.sorts)
      } else if (state.sorts && state.sorts.length > 0) {
        params.sorts = serializeSortsForRequest(state.sorts)
      } else if (state.sortParams) {
        params.sorts = serializeSortsForRequest([state.sortParams])
      }

      const response = await this.tableGateway.loadRows({ functionDetail, params })
      const pageRows = response.items || []
      total = response.paginated?.total_count ?? pageRows.length
      const availableRows = pageRows.slice(pageOffset)
      rows.push(...availableRows.slice(0, maxRows - rows.length))

      if (
        pageRows.length === 0
        || startRow + rows.length >= total
        || pageRows.length < params.page_size
      ) {
        break
      }
      page += 1
      pageOffset = 0
    }

    return {
      rows,
      total,
      truncated: rows.length < total
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
  async addRow(
    functionDetail: FunctionDetail,
    data: Record<string, unknown>,
    options?: { operation?: 'create' | 'import' }
  ): Promise<TableRow> {
    const response = await this.tableGateway.addRow(functionDetail, data, options)

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
    data: Record<string, unknown>,
    oldData?: Record<string, unknown>
  ): Promise<TableRow> {
    const response = await this.tableGateway.updateRow({
      functionDetail,
      id,
      data,
      oldData
    })

    // 触发事件
    this.eventBus.emit(TableEvent.rowUpdated, { id, row: response })

    return response
  }

  /**
   * 删除行
   */
  async deleteRow(functionDetail: FunctionDetail, id: number | string): Promise<void> {
    await this.tableGateway.deleteRow(functionDetail, id)

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
    return getTableSearchFields(functionDetail)
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
    searchForm: Record<string, unknown>
    sorts: Array<{ field: string; order: 'asc' | 'desc' }>
    pagination: { page: number; pageSize: number }
  } {
    
    const searchForm: Record<string, unknown> = {}
    const sorts: Array<{ field: string; order: 'asc' | 'desc' }> = []
    
    // 获取当前函数的所有字段 code
    const allRequestFields = getTableRequestFields(functionDetail)
    const requestFields = getTableRequestSearchFields(functionDetail)
    const responseFields = getTableListFields(functionDetail)
    
    const currentRequestFieldCodes = new Set<string>()
    const currentResponseFieldCodes = new Set<string>()
    
    allRequestFields.forEach((field: FieldConfig) => {
      currentRequestFieldCodes.add(field.code)
    })
    responseFields.forEach((field: FieldConfig) => {
      currentResponseFieldCodes.add(field.code)
    })
    
    // 恢复分页
    let page = 1
    let pageSize = 10
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
      parseSortsFromRequest(sortsString).forEach(sort => {
        if (isAllowedSortField(sort, currentRequestFieldCodes, currentResponseFieldCodes)) {
          sorts.push(sort)
        }
      })
    }
    
    // 恢复搜索条件（request 字段）
    // request 字段是 sdk-app 入参，只读取原始 field.code，例如 `genre=诗`。
    // 不支持 `s_`/`f_` 命名空间，也不读取 `__display` 伴随参数。
    requestFields.forEach((field: FieldConfig) => {
      if (!currentRequestFieldCodes.has(field.code)) return
      const value = query[field.code]
      if (value !== undefined && value !== null && value !== '') {
        searchForm[field.code] = String(value)
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
  buildSearchParams(functionDetail: FunctionDetail, searchForm: Record<string, unknown>): SearchParams {
    const searchParams: SearchParams = {}
    
    // request 字段直传给 sdk-app。
    const request = getTableRequestSearchFields(functionDetail)
    
    request.forEach((field: FieldConfig) => {
      const rawValue = getSearchFieldRawValue(searchForm[field.code])
      if (rawValue !== null && rawValue !== undefined && 
          !(Array.isArray(rawValue) && rawValue.length === 0) && 
          !(typeof rawValue === 'string' && rawValue.trim() === '')) {
        searchParams[field.code] = rawValue
      }
    })
    
    return searchParams
  }

}
