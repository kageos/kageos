/**
 * ServiceFactory - 服务工厂
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **依赖注入管理**：
 *    - 统一创建和配置所有服务实例
 *    - 简化依赖注入流程
 *    - 提供默认配置，支持自定义配置
 * 
 * 2. **服务提供**：
 *    - 实现 `IServiceProvider` 接口
 *    - 提供 Domain Services、Application Services、State Managers 等
 *    - 支持懒加载（按需创建）
 * 
 * 3. **可扩展性**：
 *    - 可以轻松替换实现（通过构造函数配置）
 *    - 支持测试时注入 Mock 对象
 *    - 遵循依赖倒置原则
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **依赖倒置原则**：
 *    - 实现 `IServiceProvider` 接口
 *    - Presentation Layer 只依赖接口，不依赖具体实现
 *    - 可以轻松替换实现，提高可测试性
 * 
 * 2. **服务分层**：
 *    - Domain Services：业务逻辑层
 *    - Application Services：业务流程编排层
 *    - State Managers：状态管理层
 *    - Infrastructure Services：基础设施层（EventBus、ApiClient 等）
 * 
 * 3. **懒加载**：
 *    - 服务实例按需创建（首次调用时创建）
 *    - 避免不必要的初始化开销
 *    - 支持循环依赖的解决
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **服务创建**：
 *    - 创建 Domain Services（FormDomainService、TableDomainService 等）
 *    - 创建 Application Services（FormApplicationService、TableApplicationService 等）
 *    - 创建 State Managers（FormStateManager、TableStateManager 等）
 * 
 * 2. **依赖注入**：
 *    - 自动注入依赖（EventBus、ApiClient、StateManager 等）
 *    - 支持自定义依赖（通过构造函数配置）
 * 
 * 3. **服务获取**：
 *    - 通过 `getXXXService()` 方法获取服务实例
 *    - 首次调用时创建，后续调用返回已创建的实例
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **服务生命周期**：
 *    - 服务实例是单例的（同一 ServiceFactory 实例）
 *    - 服务实例在首次调用时创建
 * 
 * 2. **依赖顺序**：
 *    - State Managers 不依赖其他服务
 *    - Domain Services 依赖 State Managers 和 Infrastructure Services
 *    - Application Services 依赖 Domain Services
 * 
 * 3. **测试支持**：
 *    - 可以通过构造函数注入 Mock 对象
 *    - 支持单元测试和集成测试
 * 
 * ============================================
 * 📚 相关文档
 * ============================================
 * 
 * - 服务提供者接口：`web/src/architecture/domain/interfaces/IServiceProvider.ts`
 * - 新架构扩展性分析：`web/docs/新架构扩展性分析报告.md`
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
import { serviceTreeLoader } from '../serviceTreeLoader'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import type { IApiClient } from '../../domain/interfaces/IApiClient'
import type { ICacheManager } from '../../domain/interfaces/ICacheManager'
import type { IFunctionLoader } from '../../domain/interfaces/IFunctionLoader'
import type { IServiceTreeLoader } from '../../domain/interfaces/IServiceTreeLoader'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'

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
 * 
 * 🔥 实现 IServiceProvider 接口，遵循依赖倒置原则
 */
export class ServiceFactory implements IServiceProvider {
  private eventBus: IEventBus
  private apiClient: IApiClient
  private cacheManager: ICacheManager
  private functionLoader: IFunctionLoader
  private serviceTreeLoader: IServiceTreeLoader

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
    this.serviceTreeLoader = serviceTreeLoader
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
        this.eventBus,
        this.serviceTreeLoader
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

  // ========== Infrastructure Services ==========
  /**
   * 获取事件总线
   */
  getEventBus(): IEventBus {
    return this.eventBus
  }

  /**
   * 获取 API 客户端
   */
  getApiClient(): IApiClient {
    return this.apiClient
  }

  /**
   * 获取缓存管理器
   */
  getCacheManager(): ICacheManager {
    return this.cacheManager
  }

  /**
   * 获取函数加载器
   */
  getFunctionLoader(): IFunctionLoader {
    return this.functionLoader
  }

  /**
   * 获取服务树加载器
   */
  getServiceTreeLoader(): IServiceTreeLoader {
    return this.serviceTreeLoader
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

