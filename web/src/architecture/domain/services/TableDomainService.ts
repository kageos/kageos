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
import type { FunctionDetail } from '../types'

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
 * 表格状态
 */
export interface TableState {
  data: TableRow[]
  loading: boolean
  searchParams: SearchParams
  sortParams: SortParams | null
  pagination: {
    currentPage: number
    pageSize: number
    total: number
  }
}

/**
 * 表格领域服务
 */
export class TableDomainService {
  constructor(
    private apiClient: IApiClient,
    private stateManager: IStateManager<TableState>,
    private eventBus: IEventBus
  ) {}

  /**
   * 加载表格数据
   */
  async loadData(functionDetail: FunctionDetail, searchParams?: SearchParams, sortParams?: SortParams, pagination?: { page: number, pageSize: number }): Promise<TableResponse> {
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
      if (sortParams || state.sortParams) {
        const sort = sortParams || state.sortParams!
        params.sorts = `${sort.field}:${sort.order}`
      }

      // 调用 API
      const url = `/api/v1/run${functionDetail.router}`
      const method = functionDetail.method?.toUpperCase() || 'GET'
      
      let response: TableResponse
      if (method === 'GET') {
        response = await this.apiClient.get<TableResponse>(url, params)
      } else {
        response = await this.apiClient.post<TableResponse>(url, params)
      }

      // 更新状态
      this.stateManager.setState({
        ...state,
        data: response.items || [],
        loading: false,
        searchParams: searchParams || state.searchParams,
        sortParams: sortParams || state.sortParams,
        pagination: {
          currentPage: response.paginated?.current_page || state.pagination.currentPage,
          pageSize: response.paginated?.page_size || state.pagination.pageSize,
          total: response.paginated?.total_count || 0
        }
      })

      // 触发事件
      this.eventBus.emit(TableEvent.dataLoaded, { data: response.items, pagination: response.paginated })

      return response
    } catch (error) {
      // 更新加载状态
      this.stateManager.setState({
        ...state,
        loading: false
      })
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
    const url = `/api/v1/callback${functionDetail.router}`
    const method = functionDetail.method?.toUpperCase() || 'POST'
    
    let response: TableRow
    if (method === 'GET') {
      response = await this.apiClient.get<TableRow>(url, {
        _type: 'OnTableAddRow',
        ...data
      })
    } else {
      response = await this.apiClient.post<TableRow>(url, {
        _type: 'OnTableAddRow',
        ...data
      })
    }

    // 触发事件
    this.eventBus.emit(TableEvent.rowAdded, { row: response })

    return response
  }

  /**
   * 更新行
   */
  async updateRow(functionDetail: FunctionDetail, id: number | string, data: Record<string, any>): Promise<TableRow> {
    const url = `/api/v1/callback${functionDetail.router}`
    const method = functionDetail.method?.toUpperCase() || 'PUT'
    
    let response: TableRow
    if (method === 'GET') {
      response = await this.apiClient.get<TableRow>(url, {
        _type: 'OnTableUpdateRow',
        id,
        ...data
      })
    } else {
      response = await this.apiClient.post<TableRow>(url, {
        _type: 'OnTableUpdateRow',
        id,
        ...data
      })
    }

    // 触发事件
    this.eventBus.emit(TableEvent.rowUpdated, { id, row: response })

    return response
  }

  /**
   * 删除行
   */
  async deleteRow(functionDetail: FunctionDetail, id: number | string): Promise<void> {
    const url = `/api/v1/callback${functionDetail.router}`
    const method = functionDetail.method?.toUpperCase() || 'DELETE'
    
    if (method === 'GET') {
      await this.apiClient.get(url, {
        _type: 'OnTableDeleteRows',
        ids: [id]
      })
    } else {
      await this.apiClient.post(url, {
        _type: 'OnTableDeleteRows',
        ids: [id]
      })
    }

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
}

