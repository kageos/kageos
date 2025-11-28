/**
 * TableApplicationService - 表格应用服务
 * 
 * 职责：表格业务流程编排
 * - 监听事件，调用 Domain Services
 * - 协调表格数据加载和 CRUD 操作
 * - 不包含业务逻辑，只负责编排
 * 
 * 特点：
 * - 依赖 Domain Services
 * - 通过事件总线监听和触发事件
 * - 不包含业务逻辑，只负责流程编排
 */

import { TableDomainService } from '../../domain/services/TableDomainService'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import { WorkspaceEvent, TableEvent } from '../../domain/interfaces/IEventBus'
import type { FunctionDetail } from '../../domain/types'
import type { SearchParams, SortParams } from '../../domain/services/TableDomainService'

/**
 * 表格应用服务
 */
export class TableApplicationService {
  constructor(
    private domainService: TableDomainService,
    private eventBus: IEventBus
  ) {
    this.setupEventHandlers()
  }

  /**
   * 设置事件处理器
   */
  private setupEventHandlers(): void {
    // 监听搜索变化事件
    this.eventBus.on(TableEvent.searchChanged, async (payload: { searchParams: SearchParams }) => {
      // 可以在这里添加额外的业务逻辑
      // 例如：自动重新加载数据
    })

    // 监听排序变化事件
    this.eventBus.on(TableEvent.sortChanged, async (payload: { sortParams: SortParams }) => {
      // 可以在这里添加额外的业务逻辑
      // 例如：自动重新加载数据
    })

    // 监听分页变化事件
    this.eventBus.on(TableEvent.pageChanged, async (payload: { page: number, pageSize: number }) => {
      // 可以在这里添加额外的业务逻辑
      // 例如：自动重新加载数据
    })
  }

  /**
   * 处理函数加载完成
   */
  async handleFunctionLoaded(detail: FunctionDetail): Promise<void> {
    // 加载表格数据
    await this.domainService.loadData(detail)
  }

  /**
   * 加载表格数据（供外部调用）
   */
  async loadData(
    functionDetail: FunctionDetail,
    searchParams?: SearchParams,
    sortParams?: SortParams,
    pagination?: { page: number, pageSize: number }
  ): Promise<void> {
    await this.domainService.loadData(functionDetail, searchParams, sortParams, pagination)
  }

  /**
   * 更新搜索参数（供外部调用）
   */
  updateSearchParams(searchParams: SearchParams): void {
    this.domainService.updateSearchParams(searchParams)
  }

  /**
   * 更新排序参数（供外部调用）
   */
  updateSortParams(sortParams: SortParams): void {
    this.domainService.updateSortParams(sortParams)
  }

  /**
   * 更新分页参数（供外部调用）
   */
  updatePagination(page: number, pageSize: number): void {
    this.domainService.updatePagination(page, pageSize)
  }

  /**
   * 新增行（供外部调用）
   */
  async addRow(functionDetail: FunctionDetail, data: Record<string, any>): Promise<any> {
    const result = await this.domainService.addRow(functionDetail, data)
    // 重新加载数据
    await this.loadData(functionDetail)
    return result
  }

  /**
   * 更新行（供外部调用）
   */
  async updateRow(functionDetail: FunctionDetail, id: number | string, data: Record<string, any>): Promise<any> {
    const result = await this.domainService.updateRow(functionDetail, id, data)
    // 重新加载数据
    await this.loadData(functionDetail)
    return result
  }

  /**
   * 删除行（供外部调用）
   */
  async deleteRow(functionDetail: FunctionDetail, id: number | string): Promise<void> {
    await this.domainService.deleteRow(functionDetail, id)
    // 重新加载数据
    await this.loadData(functionDetail)
  }
}

