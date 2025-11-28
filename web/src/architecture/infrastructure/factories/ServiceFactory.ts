/**
 * ServiceFactory - 服务工厂
 * 
 * 职责：创建和配置所有服务实例，简化依赖注入
 * 
 * 特点：
 * - 统一管理依赖注入
 * - 提供默认配置
 * - 可以轻松替换实现
 */

import { eventBus } from '../eventBus'
import { apiClient } from '../apiClient'
import { cacheManager } from '../cacheManager'
import { functionLoader } from '../functionLoader'
import { WorkspaceStateManager } from '../stateManager/WorkspaceStateManager'
import { FormStateManager } from '../stateManager/FormStateManager'
import { TableStateManager } from '../stateManager/TableStateManager'
import { WorkspaceDomainService } from '../../domain/services/WorkspaceDomainService'
import { FormDomainService } from '../../domain/services/FormDomainService'
import { TableDomainService } from '../../domain/services/TableDomainService'
import { WorkspaceApplicationService } from '../../application/services/WorkspaceApplicationService'
import { FormApplicationService } from '../../application/services/FormApplicationService'
import { TableApplicationService } from '../../application/services/TableApplicationService'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import type { IApiClient } from '../../domain/interfaces/IApiClient'
import type { ICacheManager } from '../../domain/interfaces/ICacheManager'
import type { IFunctionLoader } from '../../domain/interfaces/IFunctionLoader'

/**
 * 服务工厂配置
 */
export interface ServiceFactoryConfig {
  eventBus?: IEventBus
  apiClient?: IApiClient
  cacheManager?: ICacheManager
  functionLoader?: IFunctionLoader
}

/**
 * 服务工厂
 */
export class ServiceFactory {
  private eventBus: IEventBus
  private apiClient: IApiClient
  private cacheManager: ICacheManager
  private functionLoader: IFunctionLoader

  // Domain Services
  private workspaceDomainService?: WorkspaceDomainService
  private formDomainService?: FormDomainService
  private tableDomainService?: TableDomainService

  // Application Services
  private workspaceApplicationService?: WorkspaceApplicationService
  private formApplicationService?: FormApplicationService
  private tableApplicationService?: TableApplicationService

  // State Managers
  private workspaceStateManager?: WorkspaceStateManager
  private formStateManager?: FormStateManager
  private tableStateManager?: TableStateManager

  constructor(config?: ServiceFactoryConfig) {
    this.eventBus = config?.eventBus || eventBus
    this.apiClient = config?.apiClient || apiClient
    this.cacheManager = config?.cacheManager || cacheManager
    this.functionLoader = config?.functionLoader || functionLoader
  }

  /**
   * 获取工作空间状态管理器
   */
  getWorkspaceStateManager(): WorkspaceStateManager {
    if (!this.workspaceStateManager) {
      this.workspaceStateManager = new WorkspaceStateManager()
    }
    return this.workspaceStateManager
  }

  /**
   * 获取表单状态管理器
   */
  getFormStateManager(): FormStateManager {
    if (!this.formStateManager) {
      this.formStateManager = new FormStateManager()
    }
    return this.formStateManager
  }

  /**
   * 获取表格状态管理器
   */
  getTableStateManager(): TableStateManager {
    if (!this.tableStateManager) {
      this.tableStateManager = new TableStateManager()
    }
    return this.tableStateManager
  }

  /**
   * 获取工作空间领域服务
   */
  getWorkspaceDomainService(): WorkspaceDomainService {
    if (!this.workspaceDomainService) {
      const stateManager = this.getWorkspaceStateManager()
      this.workspaceDomainService = new WorkspaceDomainService(
        this.functionLoader,
        stateManager,
        this.eventBus
      )
    }
    return this.workspaceDomainService
  }

  /**
   * 获取表单领域服务
   */
  getFormDomainService(): FormDomainService {
    if (!this.formDomainService) {
      const stateManager = this.getFormStateManager()
      this.formDomainService = new FormDomainService(
        stateManager,
        this.eventBus
      )
    }
    return this.formDomainService
  }

  /**
   * 获取表格领域服务
   */
  getTableDomainService(): TableDomainService {
    if (!this.tableDomainService) {
      const stateManager = this.getTableStateManager()
      this.tableDomainService = new TableDomainService(
        this.apiClient,
        stateManager,
        this.eventBus
      )
    }
    return this.tableDomainService
  }

  /**
   * 获取工作空间应用服务
   */
  getWorkspaceApplicationService(): WorkspaceApplicationService {
    if (!this.workspaceApplicationService) {
      const domainService = this.getWorkspaceDomainService()
      this.workspaceApplicationService = new WorkspaceApplicationService(
        domainService,
        this.eventBus
      )
    }
    return this.workspaceApplicationService
  }

  /**
   * 获取表单应用服务
   */
  getFormApplicationService(): FormApplicationService {
    if (!this.formApplicationService) {
      const domainService = this.getFormDomainService()
      this.formApplicationService = new FormApplicationService(
        domainService,
        this.eventBus,
        this.apiClient
      )
    }
    return this.formApplicationService
  }

  /**
   * 获取表格应用服务
   */
  getTableApplicationService(): TableApplicationService {
    if (!this.tableApplicationService) {
      const domainService = this.getTableDomainService()
      this.tableApplicationService = new TableApplicationService(
        domainService,
        this.eventBus
      )
    }
    return this.tableApplicationService
  }

  /**
   * 重置所有服务（用于测试或清理）
   */
  reset(): void {
    this.workspaceDomainService = undefined
    this.formDomainService = undefined
    this.tableDomainService = undefined
    this.workspaceApplicationService = undefined
    this.formApplicationService = undefined
    this.tableApplicationService = undefined
    this.workspaceStateManager = undefined
    this.formStateManager = undefined
    this.tableStateManager = undefined
  }
}

// 导出单例实例
export const serviceFactory = new ServiceFactory()

