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
import type { SearchParams, SortParams, TableRow } from '../../domain/services/TableDomainService'
import { useUserInfoStore } from '@/stores/userInfo'
import { useDepartmentInfoStore } from '@/stores/departmentInfo'
import type { UserInfo } from '@/types'

/**
 * 表格应用服务
 */
export class TableApplicationService {
  constructor(
    private domainService: TableDomainService,
    private eventBus: IEventBus
  ) {
    this.setupEventHandlers()
    this.setupPreloadHooks() // 🔥 设置预加载钩子
  }

  /**
   * 🔥 设置所有 BeforeRender 钩子
   * 可以在这里注册多个不同类型的预加载逻辑
   * 
   * 生命周期：BeforeRender（渲染前执行）
   * - 执行时机：数据加载完成后、状态更新前、界面渲染前
   * - 目的：在渲染前预加载关联数据，避免渲染时再发起请求
   */
  private setupPreloadHooks(): void {
    // 注册用户信息预加载钩子
    this.domainService.beforeRender({
      name: 'preload-user-info',
      priority: 100, // 优先级，越小越早执行
      execute: this.preloadUserInfoFromTableData.bind(this)
    })
    
    // 注册部门信息预加载钩子
    this.domainService.beforeRender({
      name: 'preload-department-info',
      priority: 110, // 优先级，在用户信息之后执行
      execute: this.preloadDepartmentInfoFromTableData.bind(this)
    })
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
    // 🔥 调用 domainService.loadData，BeforeRender 钩子会在更新状态之前自动执行
    await this.domainService.loadData(functionDetail, searchParams, sortParams, pagination)
  }

  /**
   * 🔥 预加载表格数据中的用户信息到 store 缓存
   * 在数据更新后、渲染前调用，确保渲染时所有用户信息都在缓存中
   * 这个方法被设置为 TableDomainService 的预加载回调
   */
  private async preloadUserInfoFromTableData(functionDetail: FunctionDetail, tableData: TableRow[]): Promise<void> {
    try {
      // 1. 识别所有用户字段（response 字段）
      const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
      const userFields = responseFields.filter(f => f.widget?.type === 'user')
      
      if (userFields.length === 0 || !tableData || tableData.length === 0) {
        return
      }
      
      // 2. 从表格数据中收集所有用户名
      const usernames = new Set<string>()
      tableData.forEach(row => {
        userFields.forEach(field => {
          const value = row[field.code]
          if (value !== null && value !== undefined && value !== '') {
            usernames.add(String(value))
          }
        })
      })
      
      if (usernames.size === 0) {
        return
      }
      
      // 3. 🔥 批量查询用户信息到 store 缓存（这是关键！）
      // 调用 batchGetUserInfo 会把用户信息加载到 userInfoStore 的缓存中
      // 渲染时，UserDisplay 组件调用 getUserInfo 或 batchGetUserInfo 都能命中缓存
      const userInfoStore = useUserInfoStore()
      const usernamesArray = [...usernames]
      await userInfoStore.batchGetUserInfo(usernamesArray)
    } catch (error) {
      // 静默失败，不影响表格数据加载
      console.error('[TableApplicationService] 预加载用户信息失败', error)
    }
  }

  /**
   * 🔥 预加载表格数据中的部门信息到 store 缓存
   * 在数据更新后、渲染前调用，确保渲染时所有部门信息都在缓存中
   * 这个方法被设置为 TableDomainService 的预加载回调
   */
  private async preloadDepartmentInfoFromTableData(functionDetail: FunctionDetail, tableData: TableRow[]): Promise<void> {
    try {
      // 1. 识别所有部门字段（response 字段）
      const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
      const departmentFields = responseFields.filter(f => f.widget?.type === 'department' || f.widget?.type === 'departments')
      
      if (departmentFields.length === 0 || !tableData || tableData.length === 0) {
        return
      }
      
      // 2. 从表格数据中收集所有部门路径
      const paths = new Set<string>()
      tableData.forEach(row => {
        departmentFields.forEach(field => {
          const value = row[field.code]
          if (value !== null && value !== undefined && value !== '') {
            if (typeof value === 'string' && value.includes(',')) {
              // 处理逗号分隔的字符串（如 departments:/dept1,/dept2）
              value.split(',').forEach(v => {
                const trimmed = v.trim()
                if (trimmed) paths.add(trimmed)
              })
            } else {
              // 处理单个字符串（如 department=/dept1）
              paths.add(String(value))
            }
          }
        })
      })
      
      if (paths.size === 0) {
        return
      }
      
      // 3. 🔥 批量查询部门信息到 store 缓存（这是关键！）
      // 调用 batchGetDepartmentInfo 会把部门信息加载到 departmentInfoStore 的缓存中
      // 渲染时，DepartmentDisplay 组件调用 getDepartmentInfo 或 batchGetDepartmentInfo 都能命中缓存
      const departmentInfoStore = useDepartmentInfoStore()
      const pathsArray = [...paths]
      await departmentInfoStore.batchGetDepartmentInfo(pathsArray)
    } catch (error) {
      // 静默失败，不影响表格数据加载
      console.error('[TableApplicationService] 预加载部门信息失败', error)
    }
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
  async updateRow(
    functionDetail: FunctionDetail,
    id: number | string,
    data: Record<string, any>,
    oldData?: Record<string, any>
  ): Promise<any> {
    const result = await this.domainService.updateRow(functionDetail, id, data, oldData)
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

